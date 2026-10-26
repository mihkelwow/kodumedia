package local

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/platform/childprocess"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

var subtitleExtensions = map[string]domain.SubtitleKind{
	".srt": domain.SubtitleKindText, ".ass": domain.SubtitleKindText, ".ssa": domain.SubtitleKindText,
	".vtt": domain.SubtitleKindText, ".sub": domain.SubtitleKindText, ".smi": domain.SubtitleKindText,
	".sami": domain.SubtitleKindText, ".ttml": domain.SubtitleKindText, ".dfxp": domain.SubtitleKindText,
	".sup": domain.SubtitleKindImage, ".idx": domain.SubtitleKindImage,
}

var imageSubtitleCodecs = map[string]bool{
	"hdmv_pgs_subtitle": true, "dvd_subtitle": true, "dvb_subtitle": true, "xsub": true,
}

type Store struct {
	ffmpeg       func() string
	cacheRoot    string
	downloadRoot string
}

func New(ffmpeg func() string, dataDir string) (*Store, error) {
	cacheRoot := filepath.Join(dataDir, "subtitles", "cache")
	downloadRoot := filepath.Join(dataDir, "subtitles", "downloaded")
	for _, directory := range []string{cacheRoot, downloadRoot} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			return nil, fmt.Errorf("create subtitle directory: %w", err)
		}
	}
	return &Store{ffmpeg: ffmpeg, cacheRoot: cacheRoot, downloadRoot: downloadRoot}, nil
}

func (s *Store) executable() string {
	if s.ffmpeg != nil {
		if value := strings.TrimSpace(s.ffmpeg()); value != "" {
			return value
		}
	}
	return "ffmpeg"
}

func (s *Store) Discover(ctx context.Context, media domain.MediaItem) ([]domain.SubtitleTrack, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	tracks := make([]domain.SubtitleTrack, 0, 8)
	for _, track := range media.Technical.Tracks {
		if track.Kind != "subtitle" {
			continue
		}
		kind := domain.SubtitleKindText
		if imageSubtitleCodecs[strings.ToLower(track.Codec)] {
			kind = domain.SubtitleKindImage
		}
		id := fmt.Sprintf("embedded-%d", track.Index)
		tracks = append(tracks, domain.SubtitleTrack{
			ID: id, Source: domain.SubtitleSourceEmbedded, Kind: kind, Format: track.Codec,
			Language: normalizeLanguage(track.Language), Title: firstNonEmpty(track.Title, subtitleTitle(track.Language, track.Forced, track.HearingImpaired)),
			StreamIndex: track.Index, StreamOrdinal: track.StreamOrdinal, Default: track.Default,
			Forced: track.Forced, HearingImpaired: track.HearingImpaired, RequiresBurnIn: kind == domain.SubtitleKindImage,
		})
	}

	if strings.TrimSpace(media.File.Path) != "" {
		paths := candidateDirectories(media.File.Path)
		for _, directory := range paths {
			entries, err := os.ReadDir(directory)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) {
					continue
				}
				return nil, err
			}
			for _, entry := range entries {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
				if entry.IsDir() {
					continue
				}
				path := filepath.Join(directory, entry.Name())
				track, ok := s.externalTrack(media, path, domain.SubtitleSourceExternal)
				if ok {
					tracks = append(tracks, track)
				}
			}
		}
	}

	downloadDir := filepath.Join(s.downloadRoot, safeMediaID(media.ID))
	if entries, err := os.ReadDir(downloadDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			track, ok := s.externalTrack(media, filepath.Join(downloadDir, entry.Name()), domain.SubtitleSourceDownloaded)
			if ok {
				tracks = append(tracks, track)
			}
		}
	}

	tracks = deduplicate(tracks)
	sort.SliceStable(tracks, func(i, j int) bool {
		if tracks[i].Default != tracks[j].Default {
			return tracks[i].Default
		}
		if tracks[i].Forced != tracks[j].Forced {
			return tracks[i].Forced
		}
		if tracks[i].Language != tracks[j].Language {
			return tracks[i].Language < tracks[j].Language
		}
		return tracks[i].Title < tracks[j].Title
	})
	for index := range tracks {
		if tracks[index].Kind == domain.SubtitleKindText {
			tracks[index].URL = "/api/v1/media/" + media.ID + "/subtitles/" + tracks[index].ID
		}
	}
	return tracks, nil
}

func (s *Store) Resolve(ctx context.Context, media domain.MediaItem, id string) (domain.SubtitleTrack, error) {
	tracks, err := s.Discover(ctx, media)
	if err != nil {
		return domain.SubtitleTrack{}, err
	}
	for _, track := range tracks {
		if track.ID == id {
			return track, nil
		}
	}
	return domain.SubtitleTrack{}, domain.ErrNotFound
}

func (s *Store) WebVTT(ctx context.Context, media domain.MediaItem, track domain.SubtitleTrack) (string, error) {
	if track.Kind != domain.SubtitleKindText {
		return "", fmt.Errorf("%w: pildisubtiiter tuleb videosse põletada", domain.ErrUnsupported)
	}
	cacheDir := filepath.Join(s.cacheRoot, safeMediaID(media.ID))
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", err
	}
	fingerprint := subtitleFingerprint(media, track)
	output := filepath.Join(cacheDir, fingerprint+".vtt")
	if info, err := os.Stat(output); err == nil && info.Size() > 6 {
		return output, nil
	}
	temporary := output + ".tmp"
	_ = os.Remove(temporary)
	if err := s.convert(ctx, media, track, temporary); err != nil {
		_ = os.Remove(temporary)
		return "", err
	}
	if err := validateWebVTT(temporary); err != nil {
		_ = os.Remove(temporary)
		return "", err
	}
	if err := os.Rename(temporary, output); err != nil {
		_ = os.Remove(temporary)
		return "", err
	}
	return output, nil
}

func (s *Store) SaveDownloaded(ctx context.Context, media domain.MediaItem, fileName string, content io.Reader) (domain.SubtitleTrack, error) {
	if content == nil {
		return domain.SubtitleTrack{}, domain.ErrInvalidInput
	}
	fileName = sanitizeFileName(fileName)
	if fileName == "" {
		fileName = "opensubtitles.srt"
	}
	extension := strings.ToLower(filepath.Ext(fileName))
	if _, ok := subtitleExtensions[extension]; !ok || extension == ".sup" || extension == ".idx" {
		fileName = strings.TrimSuffix(fileName, extension) + ".srt"
	}
	directory := filepath.Join(s.downloadRoot, safeMediaID(media.ID))
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return domain.SubtitleTrack{}, err
	}
	destination := uniquePath(directory, fileName)
	temporary, err := os.CreateTemp(directory, ".subtitle-*.tmp")
	if err != nil {
		return domain.SubtitleTrack{}, err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	written, copyErr := io.Copy(temporary, io.LimitReader(content, 20<<20+1))
	closeErr := temporary.Close()
	if copyErr != nil {
		return domain.SubtitleTrack{}, copyErr
	}
	if closeErr != nil {
		return domain.SubtitleTrack{}, closeErr
	}
	if written > 20<<20 {
		return domain.SubtitleTrack{}, fmt.Errorf("%w: subtiitrifail on liiga suur", domain.ErrInvalidInput)
	}
	if err := os.Rename(temporaryPath, destination); err != nil {
		return domain.SubtitleTrack{}, err
	}
	track, ok := s.externalTrack(media, destination, domain.SubtitleSourceDownloaded)
	if !ok {
		return domain.SubtitleTrack{}, fmt.Errorf("%w: tundmatu subtiitrivorming", domain.ErrUnsupported)
	}
	track.URL = "/api/v1/media/" + media.ID + "/subtitles/" + track.ID
	return track, nil
}

// MaintainCache prunes only generated WebVTT data. Downloaded subtitles are
// user-selected files and are never removed automatically.
func (s *Store) MaintainCache(ctx context.Context, catalog ports.Catalog, maximumBytes int64) int64 {
	if s == nil || catalog == nil {
		return 0
	}
	before := subtitleDirectorySize(s.cacheRoot)
	valid := map[string]struct{}{}
	for offset := 0; ; offset += 500 {
		page, err := catalog.ListMedia(ctx, domain.MediaQuery{Limit: 500, Offset: offset})
		if err != nil {
			return 0
		}
		for _, item := range page.Items {
			if item.File.Path != "" {
				valid[safeMediaID(item.ID)] = struct{}{}
			}
		}
		if offset+len(page.Items) >= page.Total {
			break
		}
	}
	type candidate struct {
		path string
		size int64
		last time.Time
	}
	var candidates []candidate
	var total int64
	entries, _ := os.ReadDir(s.cacheRoot)
	for _, entry := range entries {
		path := filepath.Join(s.cacheRoot, entry.Name())
		if !entry.IsDir() {
			_ = os.Remove(path)
			continue
		}
		if _, keep := valid[entry.Name()]; !keep {
			_ = os.RemoveAll(path)
			continue
		}
		size := subtitleDirectorySize(path)
		info, _ := entry.Info()
		last := time.Time{}
		if info != nil {
			last = info.ModTime()
		}
		total += size
		candidates = append(candidates, candidate{path: path, size: size, last: last})
	}
	if maximumBytes > 0 && total > maximumBytes {
		sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].last.Before(candidates[j].last) })
		for _, item := range candidates {
			if total <= maximumBytes {
				break
			}
			if os.RemoveAll(item.path) == nil {
				total -= item.size
			}
		}
	}
	after := subtitleDirectorySize(s.cacheRoot)
	if after >= before {
		return 0
	}
	return before - after
}

func subtitleDirectorySize(root string) int64 {
	var total int64
	_ = filepath.WalkDir(root, func(_ string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		if info, infoErr := entry.Info(); infoErr == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

func (s *Store) convert(ctx context.Context, media domain.MediaItem, track domain.SubtitleTrack, output string) error {
	base := []string{"-hide_banner", "-loglevel", "error", "-nostdin", "-y"}
	if track.Source == domain.SubtitleSourceEmbedded {
		args := append(base, "-i", media.File.Path, "-map", fmt.Sprintf("0:%d", track.StreamIndex), "-c:s", "webvtt", "-f", "webvtt", output)
		return runFFmpeg(ctx, s.executable(), args)
	}
	if strings.TrimSpace(track.Path) == "" {
		return domain.ErrNotFound
	}
	encodings := []string{"", "UTF-8", "UTF-16", "UTF-16LE", "UTF-16BE", "WINDOWS-1257", "WINDOWS-1252", "WINDOWS-1251", "WINDOWS-1256", "ISO-8859-13", "ISO-8859-1", "GB18030", "SHIFT-JIS", "EUC-KR"}
	var lastErr error
	for _, encoding := range encodings {
		_ = os.Remove(output)
		args := append([]string(nil), base...)
		if encoding != "" {
			args = append(args, "-sub_charenc", encoding)
		}
		args = append(args, "-i", track.Path, "-map", "0:s:0?", "-c:s", "webvtt", "-f", "webvtt", output)
		if err := runFFmpeg(ctx, s.executable(), args); err == nil {
			if validateWebVTT(output) == nil {
				return nil
			}
		} else {
			lastErr = err
		}
	}
	if lastErr == nil {
		lastErr = errors.New("subtiitrit ei saanud WebVTT-vormingusse teisendada")
	}
	return fmt.Errorf("subtitle conversion failed: %w", lastErr)
}

func runFFmpeg(ctx context.Context, binary string, args []string) error {
	command := exec.CommandContext(ctx, binary, args...)
	childprocess.Hide(command)
	output, err := command.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if len(message) > 1200 {
			message = message[len(message)-1200:]
		}
		if message == "" {
			message = err.Error()
		}
		return errors.New(message)
	}
	return nil
}

func validateWebVTT(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(data) < 6 || !strings.HasPrefix(strings.TrimPrefix(string(data), "\ufeff"), "WEBVTT") {
		return errors.New("FFmpeg ei loonud korrektset WebVTT faili")
	}
	return nil
}

func (s *Store) externalTrack(media domain.MediaItem, path string, source domain.SubtitleSource) (domain.SubtitleTrack, bool) {
	extension := strings.ToLower(filepath.Ext(path))
	kind, ok := subtitleExtensions[extension]
	if !ok {
		return domain.SubtitleTrack{}, false
	}
	// A VobSub subtitle is a two-file .idx/.sub pair. The .idx file is the
	// descriptor that FFmpeg opens; exposing the binary .sub payload as a
	// second text track would create a broken duplicate in the player.
	if extension == ".sub" {
		idxPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".idx"
		if _, err := os.Stat(idxPath); err == nil {
			return domain.SubtitleTrack{}, false
		}
	}
	name := strings.TrimSuffix(filepath.Base(path), extension)
	if source == domain.SubtitleSourceExternal && !matchesMediaBase(media.File.Path, path) {
		return domain.SubtitleTrack{}, false
	}
	labels := subtitleLabelSuffix(media.File.Path, name)
	language, forced, hearing := parseLabels(labels)
	id := string(source) + "-" + shortHash(strings.ToLower(filepath.Clean(path)))
	return domain.SubtitleTrack{
		ID: id, Source: source, Kind: kind, Format: strings.TrimPrefix(extension, "."), Language: language,
		Title: subtitleTitle(language, forced, hearing), Path: path, Forced: forced, HearingImpaired: hearing,
		RequiresBurnIn: kind == domain.SubtitleKindImage,
	}, true
}

func subtitleLabelSuffix(mediaPath, subtitleBase string) string {
	mediaBase := strings.TrimSuffix(filepath.Base(mediaPath), filepath.Ext(mediaPath))
	if len(subtitleBase) >= len(mediaBase) && strings.EqualFold(subtitleBase[:len(mediaBase)], mediaBase) {
		return strings.TrimLeft(subtitleBase[len(mediaBase):], " ._-[]()")
	}
	return subtitleBase
}

func candidateDirectories(mediaPath string) []string {
	base := filepath.Dir(mediaPath)
	result := []string{base}
	for _, name := range []string{"Subs", "Subtitles", "subs", "subtitles"} {
		candidate := filepath.Join(base, name)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			result = append(result, candidate)
		}
	}
	return result
}

func matchesMediaBase(mediaPath, subtitlePath string) bool {
	mediaBase := normalizeBase(strings.TrimSuffix(filepath.Base(mediaPath), filepath.Ext(mediaPath)))
	subBase := normalizeBase(strings.TrimSuffix(filepath.Base(subtitlePath), filepath.Ext(subtitlePath)))
	if mediaBase == "" || subBase == "" {
		return false
	}
	return subBase == mediaBase || strings.HasPrefix(subBase, mediaBase+" ")
}

func normalizeBase(value string) string {
	value = strings.ToLower(value)
	var builder strings.Builder
	space := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			space = false
		} else if builder.Len() > 0 && !space {
			builder.WriteByte(' ')
			space = true
		}
	}
	return strings.TrimSpace(builder.String())
}

func parseLabels(name string) (language string, forced, hearing bool) {
	lower := strings.ToLower(name)
	hearing = strings.Contains(lower, "hearing-impaired") || strings.Contains(lower, "hearing_impaired")
	for _, token := range strings.FieldsFunc(lower, func(r rune) bool {
		return r == '.' || r == '_' || r == '-' || r == ' ' || r == '[' || r == ']' || r == '(' || r == ')'
	}) {
		switch token {
		case "forced", "force":
			forced = true
		case "sdh", "hi", "cc", "hearingimpaired", "hearing-impaired":
			hearing = true
		default:
			if language == "" {
				language = normalizeLanguage(token)
			}
		}
	}
	return language, forced, hearing
}

func normalizeLanguage(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	aliases := map[string]string{"est": "et", "estonian": "et", "eng": "en", "english": "en", "fin": "fi", "finnish": "fi", "rus": "ru", "russian": "ru", "deu": "de", "ger": "de", "fra": "fr", "fre": "fr", "jpn": "ja", "kor": "ko", "chi": "zh", "zho": "zh"}
	if mapped := aliases[value]; mapped != "" {
		return mapped
	}
	if len(value) == 2 || len(value) == 3 {
		return value
	}
	return ""
}

func subtitleTitle(language string, forced, hearing bool) string {
	parts := make([]string, 0, 3)
	if language != "" {
		parts = append(parts, strings.ToUpper(language))
	} else {
		parts = append(parts, "Subtiitrid")
	}
	if forced {
		parts = append(parts, "forced")
	}
	if hearing {
		parts = append(parts, "SDH")
	}
	return strings.Join(parts, " · ")
}

func subtitleFingerprint(media domain.MediaItem, track domain.SubtitleTrack) string {
	value := media.ID + "|" + track.ID + "|" + media.File.ModifiedAt.UTC().Format(time.RFC3339Nano)
	if track.Path != "" {
		if info, err := os.Stat(track.Path); err == nil {
			value += fmt.Sprintf("|%d|%d", info.Size(), info.ModTime().UnixNano())
		}
	}
	return shortHash(value)
}

func shortHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:12])
}
func safeMediaID(value string) string {
	if value == "" {
		return "unknown"
	}
	return shortHash(value)
}
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func sanitizeFileName(value string) string {
	value = filepath.Base(strings.TrimSpace(value))
	var builder strings.Builder
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune(" ._-()[]", r) {
			builder.WriteRune(r)
		}
	}
	value = strings.TrimSpace(builder.String())
	if len(value) > 180 {
		value = value[:180]
	}
	return value
}

func uniquePath(directory, fileName string) string {
	candidate := filepath.Join(directory, fileName)
	if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
		return candidate
	}
	ext := filepath.Ext(fileName)
	stem := strings.TrimSuffix(fileName, ext)
	for index := 2; index < 1000; index++ {
		candidate = filepath.Join(directory, fmt.Sprintf("%s-%d%s", stem, index, ext))
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate
		}
	}
	return filepath.Join(directory, shortHash(fileName+time.Now().String())+ext)
}

func deduplicate(tracks []domain.SubtitleTrack) []domain.SubtitleTrack {
	seen := map[string]struct{}{}
	result := make([]domain.SubtitleTrack, 0, len(tracks))
	for _, track := range tracks {
		if _, ok := seen[track.ID]; ok {
			continue
		}
		seen[track.ID] = struct{}{}
		result = append(result, track)
	}
	return result
}

var _ ports.SubtitleStore = (*Store)(nil)
