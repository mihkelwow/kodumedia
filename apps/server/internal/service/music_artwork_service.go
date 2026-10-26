package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/platform/childprocess"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

type ArtworkResult struct{ Path, ContentType string }

type MusicArtworkOption func(*MusicArtworkService)

func WithMusicMetadataArtwork(provider *MusicMetadataProvider) MusicArtworkOption {
	return func(service *MusicArtworkService) { service.metadata = provider }
}

type MusicArtworkService struct {
	catalog  ports.Catalog
	ffmpeg   func() string
	cacheDir string
	metadata *MusicMetadataProvider
	locksMu  sync.Mutex
	locks    map[string]*sync.Mutex
}

func NewMusicArtworkService(catalog ports.Catalog, ffmpeg func() string, cacheDir, _ string, options ...MusicArtworkOption) *MusicArtworkService {
	service := &MusicArtworkService{catalog: catalog, ffmpeg: ffmpeg, cacheDir: cacheDir, locks: map[string]*sync.Mutex{}}
	for _, option := range options {
		option(service)
	}
	return service
}

func (s *MusicArtworkService) Resolve(ctx context.Context, mediaID string) (ArtworkResult, error) {
	return s.resolve(ctx, mediaID, false)
}

// Refresh repeats the external artwork chain even when a generated fallback
// already exists. It is used by the explicit whole-library enrichment job so
// a temporary fallback can later be upgraded to verified artwork.
func (s *MusicArtworkService) Refresh(ctx context.Context, mediaID string) (ArtworkResult, error) {
	return s.resolve(ctx, mediaID, true)
}

func (s *MusicArtworkService) resolve(ctx context.Context, mediaID string, force bool) (ArtworkResult, error) {
	item, err := s.catalog.GetMedia(ctx, mediaID)
	if err != nil {
		return ArtworkResult{}, err
	}
	if item.Type != domain.MediaTypeMusic {
		return ArtworkResult{}, domain.ErrNotFound
	}
	if err := os.MkdirAll(s.cacheDir, 0o755); err != nil {
		return ArtworkResult{}, err
	}

	// Embedded covers and generated fallbacks belong to one physical audio
	// file. Album tags are often generic (for example "Music" / "2026") in
	// download folders, so sharing these cache entries by artist+album can leak
	// the first track's cover onto unrelated files.
	fileKey := musicFileArtworkKey(item)
	embeddedJPG := filepath.Join(s.cacheDir, fileKey+"-embedded.jpg")
	generatedSVG := filepath.Join(s.cacheDir, fileKey+"-generated.svg")
	missing := filepath.Join(s.cacheDir, fileKey+".missing")
	generatedEnabled := s.metadata != nil && s.metadata.settings != nil && s.metadata.settings().GeneratedArtworkEnabled
	preferCatalog := preferCatalogArtwork(item)

	if result, ok := localArtwork(item); ok {
		return result, nil
	}
	if !preferCatalog {
		if result, ok := cachedEmbeddedArtwork(embeddedJPG); ok {
			return result, nil
		}
	}
	if !force && generatedEnabled && fileExists(generatedSVG) {
		return ArtworkResult{Path: generatedSVG, ContentType: "image/svg+xml"}, nil
	}

	keyLock := s.lockFor(fileKey)
	keyLock.Lock()
	defer keyLock.Unlock()
	if result, ok := localArtwork(item); ok {
		return result, nil
	}
	if !preferCatalog {
		if result, ok := cachedEmbeddedArtwork(embeddedJPG); ok {
			return result, nil
		}
	}
	if !force && generatedEnabled && fileExists(generatedSVG) {
		return ArtworkResult{Path: generatedSVG, ContentType: "image/svg+xml"}, nil
	}
	if !preferCatalog {
		if result, ok := s.extractOrCachedEmbedded(ctx, item, embeddedJPG); ok {
			return result, nil
		}
	}

	if info, statErr := os.Stat(missing); statErr == nil && time.Since(info.ModTime()) < 24*time.Hour && !force {
		if preferCatalog {
			if result, ok := s.extractOrCachedEmbedded(ctx, item, embeddedJPG); ok {
				return result, nil
			}
		}
		if generatedEnabled {
			if err := generateMusicArtworkSVG(item, generatedSVG); err == nil && fileExists(generatedSVG) {
				return ArtworkResult{Path: generatedSVG, ContentType: "image/svg+xml"}, nil
			}
		}
		return ArtworkResult{}, domain.ErrNotFound
	}
	if s.metadata == nil || !s.metadata.Available() {
		if preferCatalog {
			if result, ok := s.extractOrCachedEmbedded(ctx, item, embeddedJPG); ok {
				return result, nil
			}
		}
		if generatedEnabled {
			if err := generateMusicArtworkSVG(item, generatedSVG); err == nil {
				return ArtworkResult{Path: generatedSVG, ContentType: "image/svg+xml"}, nil
			}
		}
		return ArtworkResult{}, domain.ErrNotFound
	}

	updated, changed, enrichErr := s.metadata.Enrich(ctx, item)
	if enrichErr == nil {
		item = updated
		if changed {
			_ = s.catalog.UpdateMedia(ctx, item)
		}
	}

	externalKey := musicCatalogArtworkKey(item, fileKey)
	externalJPG := filepath.Join(s.cacheDir, externalKey+"-catalog.jpg")
	if !force && fileExists(externalJPG) {
		return ArtworkResult{Path: externalJPG, ContentType: "image/jpeg"}, nil
	}
	if _, artworkErr := s.metadata.FetchArtwork(ctx, item, externalJPG); artworkErr == nil && fileExists(externalJPG) {
		_ = os.Remove(missing)
		_ = os.Remove(generatedSVG)
		return ArtworkResult{Path: externalJPG, ContentType: "image/jpeg"}, nil
	}
	if preferCatalog {
		if result, ok := s.extractOrCachedEmbedded(ctx, item, embeddedJPG); ok {
			return result, nil
		}
	}
	if generatedEnabled {
		if err := generateMusicArtworkSVG(item, generatedSVG); err == nil && fileExists(generatedSVG) {
			_ = os.Remove(missing)
			return ArtworkResult{Path: generatedSVG, ContentType: "image/svg+xml"}, nil
		}
	}
	_ = os.WriteFile(missing, []byte(time.Now().UTC().Format(time.RFC3339)), 0o600)
	return ArtworkResult{}, domain.ErrNotFound
}

func localArtwork(item domain.MediaItem) (ArtworkResult, bool) {
	// A user-provided local sidecar or safe album image must be able to replace
	// an older extracted cover without requiring cache deletion.
	if local, contentType := localMusicArtwork(item); local != "" {
		return ArtworkResult{Path: local, ContentType: contentType}, true
	}
	return ArtworkResult{}, false
}

func cachedEmbeddedArtwork(path string) (ArtworkResult, bool) {
	if fileExists(path) {
		return ArtworkResult{Path: path, ContentType: "image/jpeg"}, true
	}
	return ArtworkResult{}, false
}

func (s *MusicArtworkService) extractOrCachedEmbedded(ctx context.Context, item domain.MediaItem, target string) (ArtworkResult, bool) {
	if result, ok := cachedEmbeddedArtwork(target); ok {
		return result, true
	}
	if s.extractEmbedded(ctx, item.File.Path, target) == nil && fileExists(target) {
		return ArtworkResult{Path: target, ContentType: "image/jpeg"}, true
	}
	return ArtworkResult{}, false
}

// preferCatalogArtwork lets fingerprint/catalog artwork outrank embedded art
// when the local tags strongly resemble a loose downloads collection. This
// handles batches where every audio file was stamped with the same playlist
// image while preserving embedded covers for normal, well-tagged albums.
func preferCatalogArtwork(item domain.MediaItem) bool {
	artist := normalizeMusicMatchText(item.Artist)
	album := normalizeMusicMatchText(item.Album)
	if !genericMusicArtistLabel(artist) && !genericMusicAlbumLabel(album) {
		return false
	}
	return musicFolderHasMultipleAudioFiles(filepath.Dir(item.File.Path)) && !albumFolderArtworkSafe(item)
}

func genericMusicArtistLabel(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "", "music", "muusika", "audio", "downloads", "download", "unknown", "unknown artist", "tundmatu", "tundmatu esitaja", "various", "various artists":
		return true
	default:
		return false
	}
}

// musicFileArtworkKey is deliberately tied to the physical file. It prevents
// duplicate/generic tags from making unrelated tracks share embedded artwork,
// generated cards or negative-cache entries. The v2 namespace also makes
// installations stop reading cache files created by the old album-level logic.
func musicFileArtworkKey(item domain.MediaItem) string {
	return stableID(
		"music-artwork-file-v2",
		filepath.Clean(item.File.Path),
		strconv.FormatInt(item.File.SizeBytes, 10),
		item.File.ModifiedAt.UTC().Format(time.RFC3339Nano),
		strings.ToLower(strings.TrimSpace(item.Artist)),
		strings.ToLower(strings.TrimSpace(item.Title)),
		strings.ToLower(strings.TrimSpace(item.Album)),
		strconv.FormatInt(item.Technical.DurationMS, 10),
	)
}

func musicCatalogArtworkKey(item domain.MediaItem, fileKey string) string {
	// Keep catalog artwork track-specific too. A stale or incorrect external ID
	// must never make another file inherit this track's image. The verified
	// release IDs remain part of the key so a corrected match invalidates the
	// previous cached result without requiring manual cache deletion.
	return stableID(
		"music-artwork-catalog-v2",
		fileKey,
		strings.TrimSpace(item.ExternalIDs.MusicBrainzRelease),
		strings.TrimSpace(item.ExternalIDs.MusicBrainzReleaseGroup),
		strings.TrimSpace(item.ExternalIDs.MusicBrainzRecording),
	)
}

func (s *MusicArtworkService) lockFor(key string) *sync.Mutex {
	s.locksMu.Lock()
	defer s.locksMu.Unlock()
	if lock := s.locks[key]; lock != nil {
		return lock
	}
	lock := &sync.Mutex{}
	s.locks[key] = lock
	return lock
}

func (s *MusicArtworkService) extractEmbedded(ctx context.Context, source, target string) error {
	binary := "ffmpeg"
	if s.ffmpeg != nil && strings.TrimSpace(s.ffmpeg()) != "" {
		binary = strings.TrimSpace(s.ffmpeg())
	}
	command := exec.CommandContext(
		ctx, binary,
		"-v", "error", "-i", source, "-map", "0:v:0", "-frames:v", "1",
		"-vf", "scale=800:800:force_original_aspect_ratio=decrease",
		"-q:v", "3", "-y", target,
	)
	childprocess.Hide(command)
	if output, err := command.CombinedOutput(); err != nil {
		_ = os.Remove(target)
		return fmt.Errorf("extract embedded artwork: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// Maintain removes cache entries that cannot belong to a current music file,
// deduplicates identical normalized images with hard links when the filesystem
// supports them, and then applies an LRU size cap. Local sidecar artwork is
// never touched.
func (s *MusicArtworkService) Maintain(ctx context.Context, maximumBytes int64) int64 {
	if s == nil || s.catalog == nil {
		return 0
	}
	before := directorySizeForMaintenance(s.cacheDir)
	valid := map[string]struct{}{}
	for offset := 0; ; offset += 500 {
		page, err := s.catalog.ListMedia(ctx, domain.MediaQuery{Types: []domain.MediaType{domain.MediaTypeMusic}, Limit: 500, Offset: offset})
		if err != nil {
			return 0
		}
		for _, item := range page.Items {
			fileKey := musicFileArtworkKey(item)
			for _, name := range []string{fileKey + "-embedded.jpg", fileKey + "-generated.svg", fileKey + ".missing"} {
				valid[name] = struct{}{}
			}
			valid[musicCatalogArtworkKey(item, fileKey)+"-catalog.jpg"] = struct{}{}
		}
		if offset+len(page.Items) >= page.Total {
			break
		}
	}
	entries, _ := os.ReadDir(s.cacheDir)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if _, keep := valid[entry.Name()]; !keep {
			_ = os.Remove(filepath.Join(s.cacheDir, entry.Name()))
		}
	}
	deduplicateArtworkFiles(s.cacheDir)
	enforceMaintenanceLimit(s.cacheDir, maximumBytes)
	after := directorySizeForMaintenance(s.cacheDir)
	if after >= before {
		return 0
	}
	return before - after
}

func deduplicateArtworkFiles(root string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	canonical := map[string]string{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".jpg") {
			continue
		}
		path := filepath.Join(root, entry.Name())
		file, openErr := os.Open(path)
		if openErr != nil {
			continue
		}
		hash := sha256.New()
		_, copyErr := io.Copy(hash, file)
		closeErr := file.Close()
		if copyErr != nil || closeErr != nil {
			continue
		}
		key := hex.EncodeToString(hash.Sum(nil))
		original := canonical[key]
		if original == "" {
			canonical[key] = path
			continue
		}
		temporary := path + ".dedupe"
		_ = os.Remove(temporary)
		if os.Link(original, temporary) != nil {
			continue
		}
		if os.Remove(path) != nil || os.Rename(temporary, path) != nil {
			_ = os.Remove(temporary)
		}
	}
}

type maintenanceFile struct {
	path     string
	size     int64
	modified time.Time
}

func enforceMaintenanceLimit(root string, maximumBytes int64) {
	if maximumBytes <= 0 {
		return
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	files := make([]maintenanceFile, 0, len(entries))
	var total int64
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}
		total += info.Size()
		files = append(files, maintenanceFile{
			path: filepath.Join(root, entry.Name()), size: info.Size(), modified: info.ModTime(),
		})
	}
	sort.SliceStable(files, func(i, j int) bool { return files[i].modified.Before(files[j].modified) })
	for _, file := range files {
		if total <= maximumBytes {
			break
		}
		if os.Remove(file.path) == nil {
			total -= file.size
		}
	}
}

func directorySizeForMaintenance(root string) int64 {
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

func localMusicArtwork(item domain.MediaItem) (string, string) {
	audioPath := item.File.Path
	dir := filepath.Dir(audioPath)
	stem := strings.TrimSuffix(filepath.Base(audioPath), filepath.Ext(audioPath))

	// A sidecar named after the audio file is always track-specific and safe,
	// even inside a large downloads folder.
	for _, name := range []string{
		stem + ".jpg", stem + ".jpeg", stem + ".png",
		stem + "-cover.jpg", stem + "-cover.jpeg", stem + "-cover.png",
	} {
		if candidate, contentType := existingMusicImage(filepath.Join(dir, name)); candidate != "" {
			return candidate, contentType
		}
	}

	// Generic cover.jpg/folder.jpg files are album-level. Use them only when the
	// directory actually looks like an album folder; otherwise one stray image
	// in a loose-track collection would be assigned to every song.
	if !albumFolderArtworkSafe(item) {
		return "", ""
	}
	for _, name := range []string{"cover.jpg", "Cover.jpg", "folder.jpg", "Folder.jpg", "front.jpg", "album.jpg", "cover.png", "folder.png", "front.png", "album.png"} {
		if candidate, contentType := existingMusicImage(filepath.Join(dir, name)); candidate != "" {
			return candidate, contentType
		}
	}
	return "", ""
}

func existingMusicImage(candidate string) (string, string) {
	if !fileExists(candidate) {
		return "", ""
	}
	if strings.EqualFold(filepath.Ext(candidate), ".png") {
		return candidate, "image/png"
	}
	return candidate, "image/jpeg"
}

func albumFolderArtworkSafe(item domain.MediaItem) bool {
	dir := filepath.Dir(item.File.Path)
	if !musicFolderHasMultipleAudioFiles(dir) {
		return true
	}

	album := normalizeMusicMatchText(item.Album)
	if album == "" || genericMusicAlbumLabel(album) {
		return false
	}
	folder := normalizeMusicMatchText(filepath.Base(dir))
	parent := normalizeMusicMatchText(filepath.Base(filepath.Dir(dir)))
	return musicFolderMatchesAlbum(folder, album) || musicFolderMatchesAlbum(parent, album)
}

func musicFolderHasMultipleAudioFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	audioCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if isMusicAudioExtension(filepath.Ext(entry.Name())) {
			audioCount++
			if audioCount > 1 {
				return true
			}
		}
	}
	return false
}

func musicFolderMatchesAlbum(folder, album string) bool {
	if folder == "" || album == "" {
		return false
	}
	if folder == album || strings.Contains(folder, album) || strings.Contains(album, folder) {
		return true
	}
	return musicTokenSimilarity(folder, album) >= 0.72
}

func genericMusicAlbumLabel(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return true
	}
	if len(value) == 4 {
		if year, err := strconv.Atoi(value); err == nil && year >= 1900 && year <= 2200 {
			return true
		}
	}
	switch value {
	case "music", "muusika", "audio", "downloads", "download", "singles", "single", "tracks", "track", "songs", "song", "unknown", "unknown album", "tundmatu", "tundmatu album", "unreleased", "bootlegs", "bootleg", "remixes", "remix", "various", "various artists":
		return true
	default:
		return false
	}
}

func isMusicAudioExtension(extension string) bool {
	switch strings.ToLower(strings.TrimSpace(extension)) {
	case ".mp3", ".flac", ".wav", ".m4a", ".aac", ".ogg", ".opus", ".wma", ".aiff", ".aif", ".ape", ".alac":
		return true
	default:
		return false
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Size() > 0
}
