package imdb

import (
	"bufio"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

// ratingRecord deliberately uses a compact fixed-width representation. The
// IMDb ratings dataset contains millions of rows, so a map of strings would
// consume substantially more memory than this sorted slice.
type ratingRecord struct {
	id     uint64
	rating float32
	votes  uint32
}

type fileSignature struct {
	path    string
	size    int64
	modTime time.Time
}

type RatingsProvider struct {
	path      func() string
	mu        sync.Mutex
	signature fileSignature
	data      []ratingRecord
	err       error
}

func NewRatingsProvider(path string) *RatingsProvider {
	return NewDynamicRatingsProvider(func() string { return path })
}

func NewDynamicRatingsProvider(path func() string) *RatingsProvider {
	return &RatingsProvider{path: path}
}

func (p *RatingsProvider) Name() string { return "imdb-non-commercial-dataset" }

func (p *RatingsProvider) currentPath() string {
	if p.path == nil {
		return ""
	}
	return strings.TrimSpace(p.path())
}

func (p *RatingsProvider) Available() bool {
	path := p.currentPath()
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// Invalidate drops only the in-memory index. The on-disk dataset remains in
// place and is lazily reloaded by the next lookup. This keeps refresh jobs
// responsive and avoids blocking server startup on parsing a large file.
func (p *RatingsProvider) Invalidate() {
	p.mu.Lock()
	p.signature = fileSignature{}
	p.data = nil
	p.err = nil
	p.mu.Unlock()
}

func (p *RatingsProvider) Lookup(ctx context.Context, externalID string) (float64, int64, bool, error) {
	id, ok := parseTConst(externalID)
	if !ok {
		return 0, 0, false, nil
	}
	path := p.currentPath()
	if path == "" {
		return 0, 0, false, domain.ErrUnavailable
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return 0, 0, false, domain.ErrUnavailable
	}
	signature := fileSignature{path: filepath.Clean(path), size: info.Size(), modTime: info.ModTime()}

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.signature != signature {
		p.data, p.err = loadRatings(ctx, path)
		p.signature = signature
	}
	if p.err != nil {
		return 0, 0, false, p.err
	}
	index := sort.Search(len(p.data), func(index int) bool { return p.data[index].id >= id })
	if index >= len(p.data) || p.data[index].id != id {
		return 0, 0, false, nil
	}
	record := p.data[index]
	return float64(record.rating), int64(record.votes), true, nil
}

func loadRatings(ctx context.Context, path string) ([]ratingRecord, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	buffered := bufio.NewReader(file)
	var reader io.Reader = buffered
	magic, _ := buffered.Peek(2)
	if len(magic) == 2 && magic[0] == 0x1f && magic[1] == 0x8b {
		compressed, gzipErr := gzip.NewReader(buffered)
		if gzipErr != nil {
			return nil, fmt.Errorf("open imdb gzip: %w", gzipErr)
		}
		defer compressed.Close()
		reader = compressed
	}

	data := make([]ratingRecord, 0, 1_600_000)
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	first := true
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		line := scanner.Text()
		if first {
			first = false
			if line != "tconst\taverageRating\tnumVotes" {
				return nil, fmt.Errorf("invalid IMDb ratings header")
			}
			continue
		}
		firstTab := strings.IndexByte(line, '\t')
		if firstTab < 0 {
			continue
		}
		secondRelative := strings.IndexByte(line[firstTab+1:], '\t')
		if secondRelative < 0 {
			continue
		}
		secondTab := firstTab + 1 + secondRelative
		id, validID := parseTConst(line[:firstTab])
		rating, ratingErr := strconv.ParseFloat(line[firstTab+1:secondTab], 32)
		votes, votesErr := strconv.ParseUint(line[secondTab+1:], 10, 32)
		if !validID || ratingErr != nil || votesErr != nil {
			continue
		}
		data = append(data, ratingRecord{id: id, rating: float32(rating), votes: uint32(votes)})
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("read imdb ratings: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("IMDb ratings dataset contains no records")
	}
	if !sort.SliceIsSorted(data, func(left, right int) bool { return data[left].id < data[right].id }) {
		sort.Slice(data, func(left, right int) bool { return data[left].id < data[right].id })
	}
	return data, nil
}

func parseTConst(value string) (uint64, bool) {
	value = strings.TrimSpace(value)
	if len(value) < 3 || !strings.EqualFold(value[:2], "tt") {
		return 0, false
	}
	id, err := strconv.ParseUint(value[2:], 10, 64)
	return id, err == nil
}

var _ ports.RatingsProvider = (*RatingsProvider)(nil)
