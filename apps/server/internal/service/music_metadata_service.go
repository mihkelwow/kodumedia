package service

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/platform/childprocess"
)

const (
	musicMetadataCacheSchema = 4
	fingerprintCacheSchema   = 1
	fpcalcVersion            = "1.6.0"
	fpcalcDownloadURL        = "https://github.com/acoustid/chromaprint/releases/download/v1.6.0/chromaprint-fpcalc-1.6.0-windows-x86_64.zip"
	fpcalcArchiveSHA256      = "30179d3d0dc4cc92f1a0995c1a2e523fb4867724c2ee6a6ceae474f8e4d6937a"
)

type MusicMetadataMatch struct {
	RecordingID    string   `json:"recordingId,omitempty"`
	ReleaseID      string   `json:"releaseId,omitempty"`
	ReleaseGroupID string   `json:"releaseGroupId,omitempty"`
	AcoustID       string   `json:"acoustId,omitempty"`
	Title          string   `json:"title,omitempty"`
	Artist         string   `json:"artist,omitempty"`
	Album          string   `json:"album,omitempty"`
	ReleaseStatus  string   `json:"releaseStatus,omitempty"`
	Genres         []string `json:"genres,omitempty"`
	Styles         []string `json:"styles,omitempty"`
	Moods          []string `json:"moods,omitempty"`
	ArtworkURL     string   `json:"artworkUrl,omitempty"`
	ArtworkSource  string   `json:"artworkSource,omitempty"`
	Source         string   `json:"source,omitempty"`
	SourceURL      string   `json:"sourceUrl,omitempty"`
	Method         string   `json:"method,omitempty"`
	Score          int      `json:"score,omitempty"`
	BPM            float64  `json:"bpm,omitempty"`
}

type musicMetadataCacheEntry struct {
	SchemaVersion int                 `json:"schemaVersion"`
	FetchedAt     time.Time           `json:"fetchedAt"`
	Miss          bool                `json:"miss,omitempty"`
	Match         *MusicMetadataMatch `json:"match,omitempty"`
}

type fingerprintCacheEntry struct {
	SchemaVersion int       `json:"schemaVersion"`
	CreatedAt     time.Time `json:"createdAt"`
	Duration      int       `json:"duration"`
	Fingerprint   string    `json:"fingerprint"`
}

type mbTag struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type mbArtistCredit struct {
	Name       string `json:"name"`
	JoinPhrase string `json:"joinphrase"`
	Artist     struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"artist"`
}

type mbReleaseGroup struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	PrimaryType    string   `json:"primary-type"`
	SecondaryTypes []string `json:"secondary-types"`
	Genres         []mbTag  `json:"genres"`
	Tags           []mbTag  `json:"tags"`
}

type mbRelease struct {
	ID           string           `json:"id"`
	Title        string           `json:"title"`
	Status       string           `json:"status"`
	Date         string           `json:"date"`
	ArtistCredit []mbArtistCredit `json:"artist-credit"`
	ReleaseGroup mbReleaseGroup   `json:"release-group"`
}

type mbRecording struct {
	ID               string           `json:"id"`
	Title            string           `json:"title"`
	Length           int64            `json:"length"`
	Score            int              `json:"score"`
	FirstReleaseDate string           `json:"first-release-date"`
	ArtistCredit     []mbArtistCredit `json:"artist-credit"`
	Releases         []mbRelease      `json:"releases"`
	Genres           []mbTag          `json:"genres"`
	Tags             []mbTag          `json:"tags"`
}

type acoustIDArtist struct {
	Name string `json:"name"`
}

type acoustIDRecording struct {
	ID            string           `json:"id"`
	Title         string           `json:"title"`
	Duration      float64          `json:"duration"`
	Artists       []acoustIDArtist `json:"artists"`
	ReleaseGroups []struct {
		ID             string   `json:"id"`
		Title          string   `json:"title"`
		Type           string   `json:"type"`
		SecondaryTypes []string `json:"secondarytypes"`
	} `json:"releasegroups"`
}

type MusicMetadataOption func(*MusicMetadataProvider)

func WithMusicMetadataFFmpeg(resolve func() string) MusicMetadataOption {
	return func(provider *MusicMetadataProvider) { provider.ffmpeg = resolve }
}

type MusicMetadataProvider struct {
	settings  func() domain.MusicSettings
	cacheDir  string
	version   string
	client    *http.Client
	now       func() time.Time
	mbURL     string
	acoustURL string
	deezerURL string
	shazamURL string
	ffmpeg    func() string

	shazamLookup func(context.Context, domain.MediaItem) (MusicMetadataMatch, bool, error)

	fpcalcPath string
	fpcalcURL  string
	fpcalcSHA  string

	mbMu       sync.Mutex
	lastMBCall time.Time
	acoustMu   sync.Mutex
	lastAcoust time.Time
	deezerMu   sync.Mutex
	lastDeezer time.Time
	shazamMu   sync.Mutex
	lastShazam time.Time
	toolMu     sync.Mutex
}

func NewMusicMetadataProvider(settings func() domain.MusicSettings, cacheDir, version string, options ...MusicMetadataOption) *MusicMetadataProvider {
	provider := &MusicMetadataProvider{
		settings:  settings,
		cacheDir:  cacheDir,
		version:   version,
		client:    &http.Client{Timeout: 20 * time.Second},
		now:       time.Now,
		mbURL:     "https://musicbrainz.org/ws/2",
		acoustURL: "https://api.acoustid.org/v2",
		deezerURL: "https://api.deezer.com",
		shazamURL: "https://amp.shazam.com",
		fpcalcURL: fpcalcDownloadURL,
		fpcalcSHA: fpcalcArchiveSHA256,
	}
	for _, option := range options {
		option(provider)
	}
	return provider
}

func (p *MusicMetadataProvider) Available() bool {
	return p != nil && p.settings != nil && p.settings().MetadataEnabled
}

func (p *MusicMetadataProvider) Enrich(ctx context.Context, item domain.MediaItem) (domain.MediaItem, bool, error) {
	return p.enrich(ctx, item, false)
}

// EnrichDeep adds the normal catalog lookup and, when the catalog still cannot
// provide a useful classification, a short local audio analysis. The deep path
// is intentionally used by the explicit library enrichment job rather than by
// every lightweight API request.
func (p *MusicMetadataProvider) EnrichDeep(ctx context.Context, item domain.MediaItem) (domain.MediaItem, bool, error) {
	return p.enrich(ctx, item, true)
}

func (p *MusicMetadataProvider) enrich(ctx context.Context, item domain.MediaItem, deep bool) (domain.MediaItem, bool, error) {
	if !p.Available() || item.Type != domain.MediaTypeMusic {
		return item, false, nil
	}
	lookupCtx := ctx
	cancelLookup := func() {}
	if deep {
		// Keep enough of the per-track budget for local analysis even when a
		// public catalog is temporarily slow or unreachable.
		lookupCtx, cancelLookup = context.WithTimeout(ctx, 16*time.Second)
	}
	match, found, lookupErr := p.Lookup(lookupCtx, item)
	cancelLookup()
	if lookupErr != nil {
		// A public catalog being temporarily unavailable must not prevent local
		// inference or the rest of a large library job from completing.
		match = MusicMetadataMatch{}
	}
	if !found {
		match = MusicMetadataMatch{Method: "inferred", Source: "local-inference"}
	}

	updated := item
	changed := false
	if match.RecordingID != "" && updated.ExternalIDs.MusicBrainzRecording != match.RecordingID {
		updated.ExternalIDs.MusicBrainzRecording = match.RecordingID
		changed = true
	}
	if match.ReleaseID != "" && updated.ExternalIDs.MusicBrainzRelease != match.ReleaseID {
		updated.ExternalIDs.MusicBrainzRelease = match.ReleaseID
		changed = true
	}
	if match.ReleaseGroupID != "" && updated.ExternalIDs.MusicBrainzReleaseGroup != match.ReleaseGroupID {
		updated.ExternalIDs.MusicBrainzReleaseGroup = match.ReleaseGroupID
		changed = true
	}
	if match.AcoustID != "" && updated.ExternalIDs.AcoustID != match.AcoustID {
		updated.ExternalIDs.AcoustID = match.AcoustID
		changed = true
	}

	classification := domain.MusicClassification{}
	if updated.MusicClassification != nil {
		classification = *updated.MusicClassification
		classification.Genres = append([]string(nil), classification.Genres...)
		classification.Styles = append([]string(nil), classification.Styles...)
		classification.Moods = append([]string(nil), classification.Moods...)
	}
	originalClassificationGenres := append([]string(nil), classification.Genres...)

	seedGenres := append([]string(nil), classification.Genres...)
	seedGenres = append(seedGenres, match.Genres...)
	seedStyles := append([]string(nil), classification.Styles...)
	seedStyles = append(seedStyles, match.Styles...)
	classificationItem := item
	if strings.EqualFold(match.Source, "shazam") {
		classificationItem.Title = firstNonEmptyMusicText(match.Title, item.Title)
		classificationItem.Artist = firstNonEmptyMusicText(match.Artist, item.Artist)
		classificationItem.Album = firstNonEmptyMusicText(match.Album, item.Album)
	}
	inferred := inferMusicClassification(classificationItem, seedGenres, seedStyles, match.BPM)
	if deep && p.settings().AudioAnalysisEnabled && needsAudioClassification(classification, match, inferred) {
		if analysis, analysisErr := p.analyzeAudioClassification(ctx, item); analysisErr == nil {
			inferred = mergeInferredClassification(inferred, analysis)
		}
	}

	catalogSource := strings.TrimSpace(match.Source)
	if catalogSource == "" && match.RecordingID != "" {
		catalogSource = "musicbrainz"
	}
	genreFieldChanged := false
	if values, source, fieldChanged := chooseMusicClassificationField(
		classification.Genres, classification.GenreSource,
		match.Genres, catalogSource,
		inferred.Genres, inferred.Source,
	); fieldChanged {
		classification.Genres = values
		classification.GenreSource = source
		genreFieldChanged = true
		changed = true
	}
	if values, source, fieldChanged := chooseMusicClassificationField(
		classification.Styles, classification.StyleSource,
		match.Styles, catalogSource,
		inferred.Styles, inferred.Source,
	); fieldChanged {
		classification.Styles = values
		classification.StyleSource = source
		changed = true
	}
	if values, source, fieldChanged := chooseMusicClassificationField(
		classification.Moods, classification.MoodSource,
		match.Moods, catalogSource,
		inferred.Moods, inferred.Source,
	); fieldChanged {
		classification.Moods = values
		classification.MoodSource = source
		changed = true
	}
	if match.SourceURL != "" && (classification.SourceURL == "" || musicClassificationSourceRank(match.Source, true) > musicClassificationSourceRank(classification.Source, len(classification.Genres) > 0 || len(classification.Styles) > 0)) {
		classification.SourceURL = match.SourceURL
		changed = true
	}
	if classification.Confidence == "" && (len(classification.Genres) > 0 || len(classification.Styles) > 0) {
		confidence := strings.TrimSpace(match.Method)
		if confidence == "" || confidence == "inferred" {
			confidence = inferred.Confidence
		}
		classification.Confidence = confidence
		changed = true
	}
	newSource := classificationSummarySource(classification)
	if classification.Source != newSource {
		classification.Source = newSource
		changed = true
	}
	if len(classification.Genres) > 0 || len(classification.Styles) > 0 || len(classification.Moods) > 0 || classification.SourceURL != "" {
		updated.MusicClassification = &classification
	}
	if len(classification.Genres) > 0 && (len(updated.Metadata.Genres) == 0 || (genreFieldChanged && (equalFoldedMusicValues(updated.Metadata.Genres, originalClassificationGenres) || containsFold(updated.Metadata.Genres, "Other")))) {
		updated.Metadata.Genres = append([]string(nil), classification.Genres...)
		changed = true
	}
	if lookupErr != nil && !changed {
		return item, false, lookupErr
	}
	return updated, changed, nil
}

func chooseMusicClassificationField(current []string, currentSource string, catalog []string, catalogSource string, inferred []string, inferredSource string) ([]string, string, bool) {
	values := normalizeMusicValues(current)
	source := strings.TrimSpace(currentSource)
	rank := musicClassificationSourceRank(source, len(values) > 0)

	apply := func(candidate []string, candidateSource string) {
		candidate = removeOtherWhenSpecific(normalizeMusicValues(candidate))
		candidateSource = strings.TrimSpace(candidateSource)
		if len(candidate) == 0 || candidateSource == "" {
			return
		}
		candidateRank := musicClassificationSourceRank(candidateSource, true)
		if len(values) == 0 || candidateRank > rank {
			values = append([]string(nil), candidate...)
			source = candidateSource
			rank = candidateRank
		}
	}

	apply(catalog, catalogSource)
	apply(inferred, inferredSource)
	changed := !equalFoldedMusicValues(values, current) || !strings.EqualFold(source, strings.TrimSpace(currentSource))
	return values, source, changed
}

func musicClassificationSourceRank(source string, hasValues bool) int {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "sidecar":
		return 100
	case "embedded":
		return 90
	case "shazam":
		return 85
	case "musicbrainz":
		return 80
	case "deezer", "catalog":
		return 70
	case "audio-analysis":
		return 50
	case "local-inference":
		return 40
	case "fallback":
		return 10
	case "":
		// A legacy non-empty value without source metadata is safer to preserve
		// than to treat as an automatic fallback.
		if hasValues {
			return 85
		}
	}
	return 60
}

func equalFoldedMusicValues(left, right []string) bool {
	left = normalizeMusicValues(left)
	right = normalizeMusicValues(right)
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if !strings.EqualFold(left[index], right[index]) {
			return false
		}
	}
	return true
}

func classificationSummarySource(value domain.MusicClassification) string {
	sources := normalizeMusicValues([]string{value.GenreSource, value.StyleSource, value.MoodSource})
	if len(sources) == 1 {
		return sources[0]
	}
	if len(sources) > 1 {
		return "mixed"
	}
	return value.Source
}

func (p *MusicMetadataProvider) Lookup(ctx context.Context, item domain.MediaItem) (MusicMetadataMatch, bool, error) {
	if !p.Available() || item.Type != domain.MediaTypeMusic {
		return MusicMetadataMatch{}, false, nil
	}
	key := p.lookupKey(item)
	if entry, ok := p.readLookupCache(key); ok {
		if entry.Miss || entry.Match == nil {
			return MusicMetadataMatch{}, false, nil
		}
		return *entry.Match, true, nil
	}

	settings := p.settings()
	var lookupErr error
	if settings.ShazamEnabled {
		lookup := p.lookupShazam
		if p.shazamLookup != nil {
			lookup = p.shazamLookup
		}
		match, ok, shazamErr := lookup(ctx, item)
		if shazamErr != nil {
			lookupErr = shazamErr
		} else if ok {
			_ = p.writeLookupCache(key, musicMetadataCacheEntry{SchemaVersion: musicMetadataCacheSchema, FetchedAt: p.now().UTC(), Match: &match})
			return match, true, nil
		}
	}

	var recording mbRecording
	var acoustID string
	method := "matched"
	found := false
	if settings.FingerprintEnabled && strings.TrimSpace(settings.AcoustIDAPIKey) != "" {
		candidate, candidateAcoustID, ok, fingerprintErr := p.lookupByFingerprint(ctx, item, strings.TrimSpace(settings.AcoustIDAPIKey))
		if fingerprintErr != nil {
			lookupErr = fingerprintErr
		} else if ok {
			recording, acoustID, found, method = candidate, candidateAcoustID, true, "fingerprint"
		}
	}
	if !found {
		candidate, ok, searchErr := p.searchRecording(ctx, item)
		if searchErr != nil {
			lookupErr = searchErr
		} else {
			recording, found = candidate, ok
		}
	}
	if found && recording.ID != "" {
		details, err := p.fetchRecording(ctx, recording.ID)
		if err == nil && details.ID != "" {
			details.Score = recording.Score
			recording = details
		}
		match := p.buildMatch(ctx, item, recording, acoustID, method)
		if match.RecordingID != "" {
			_ = p.writeLookupCache(key, musicMetadataCacheEntry{SchemaVersion: musicMetadataCacheSchema, FetchedAt: p.now().UTC(), Match: &match})
			return match, true, nil
		}
	}

	if settings.CatalogFallbackEnabled {
		match, ok, catalogErr := p.lookupDeezer(ctx, item)
		if catalogErr != nil {
			lookupErr = catalogErr
		} else if ok {
			_ = p.writeLookupCache(key, musicMetadataCacheEntry{SchemaVersion: musicMetadataCacheSchema, FetchedAt: p.now().UTC(), Match: &match})
			return match, true, nil
		}
	}

	if lookupErr != nil {
		return MusicMetadataMatch{}, false, lookupErr
	}
	_ = p.writeLookupCache(key, musicMetadataCacheEntry{SchemaVersion: musicMetadataCacheSchema, FetchedAt: p.now().UTC(), Miss: true})
	return MusicMetadataMatch{}, false, nil
}

func (p *MusicMetadataProvider) lookupKey(item domain.MediaItem) string {
	return stableID(
		"music-metadata-v4",
		filepath.Clean(item.File.Path),
		strconv.FormatInt(item.File.SizeBytes, 10),
		item.File.ModifiedAt.UTC().Format(time.RFC3339Nano),
		normalizeMusicMatchText(item.Artist),
		normalizeMusicMatchText(item.Title),
		normalizeMusicMatchText(item.Album),
		strconv.FormatInt(item.Technical.DurationMS, 10),
	)
}

func (p *MusicMetadataProvider) lookupByFingerprint(ctx context.Context, item domain.MediaItem, apiKey string) (mbRecording, string, bool, error) {
	fingerprint, err := p.fingerprint(ctx, item)
	if err != nil {
		return mbRecording{}, "", false, err
	}
	p.acoustMu.Lock()
	defer p.acoustMu.Unlock()
	if wait := time.Until(p.lastAcoust.Add(350 * time.Millisecond)); wait > 0 {
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return mbRecording{}, "", false, ctx.Err()
		case <-timer.C:
		}
	}
	values := url.Values{
		"client":      {apiKey},
		"format":      {"json"},
		"meta":        {"recordings+releasegroups+compress"},
		"duration":    {strconv.Itoa(fingerprint.Duration)},
		"fingerprint": {fingerprint.Fingerprint},
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, p.acoustURL+"/lookup", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", p.userAgent())
	resp, err := p.client.Do(req)
	if err != nil {
		return mbRecording{}, "", false, err
	}
	defer resp.Body.Close()
	p.lastAcoust = p.now()
	if resp.StatusCode != http.StatusOK {
		return mbRecording{}, "", false, fmt.Errorf("AcoustID lookup returned %s", resp.Status)
	}
	var payload struct {
		Status string `json:"status"`
		Error  struct {
			Message string `json:"message"`
		} `json:"error"`
		Results []struct {
			ID         string              `json:"id"`
			Score      float64             `json:"score"`
			Recordings []acoustIDRecording `json:"recordings"`
		} `json:"results"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&payload); err != nil {
		return mbRecording{}, "", false, err
	}
	if payload.Status != "ok" {
		return mbRecording{}, "", false, fmt.Errorf("AcoustID lookup failed: %s", strings.TrimSpace(payload.Error.Message))
	}
	type candidate struct {
		recording mbRecording
		acoustID  string
		score     int
	}
	candidates := make([]candidate, 0, 8)
	for _, result := range payload.Results {
		if result.Score < 0.75 {
			continue
		}
		for _, remote := range result.Recordings {
			artist := joinAcoustIDArtists(remote.Artists)
			length := int64(remote.Duration * 1000)
			metadataScore := scoreMusicRecording(item, mbRecording{Title: remote.Title, Length: length, ArtistCredit: []mbArtistCredit{{Name: artist}}}, 100)
			if metadataScore == 0 && result.Score < 0.92 {
				continue
			}
			total := int(result.Score*70) + metadataScore/3
			candidates = append(candidates, candidate{recording: mbRecording{ID: remote.ID, Title: remote.Title, Length: length, ArtistCredit: []mbArtistCredit{{Name: artist}}}, acoustID: result.ID, score: total})
		}
	}
	if len(candidates) == 0 {
		return mbRecording{}, "", false, nil
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })
	if candidates[0].score < 70 || (len(candidates) > 1 && candidates[0].score-candidates[1].score < 4 && candidates[0].recording.ID != candidates[1].recording.ID) {
		return mbRecording{}, "", false, nil
	}
	best := candidates[0]
	best.recording.Score = best.score
	return best.recording, best.acoustID, true, nil
}

func joinAcoustIDArtists(values []acoustIDArtist) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		if name := strings.TrimSpace(value.Name); name != "" {
			parts = append(parts, name)
		}
	}
	return strings.Join(parts, ", ")
}

func (p *MusicMetadataProvider) searchRecording(ctx context.Context, item domain.MediaItem) (mbRecording, bool, error) {
	title := strings.TrimSpace(item.Title)
	if title == "" {
		return mbRecording{}, false, nil
	}
	parts := []string{`recording:"` + escapeMBQuery(title) + `"`}
	if artist := strings.TrimSpace(item.Artist); artist != "" {
		parts = append(parts, `artist:"`+escapeMBQuery(artist)+`"`)
	}
	endpoint := p.mbURL + "/recording/?fmt=json&limit=10&query=" + url.QueryEscape(strings.Join(parts, " AND "))
	var payload struct {
		Recordings []mbRecording `json:"recordings"`
	}
	if err := p.musicBrainzJSON(ctx, endpoint, &payload); err != nil {
		return mbRecording{}, false, err
	}
	type scored struct {
		recording mbRecording
		score     int
	}
	candidates := make([]scored, 0, len(payload.Recordings))
	for _, candidate := range payload.Recordings {
		score := scoreMusicRecording(item, candidate, candidate.Score)
		if score > 0 {
			candidates = append(candidates, scored{recording: candidate, score: score})
		}
	}
	if len(candidates) == 0 {
		return mbRecording{}, false, nil
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })
	if candidates[0].score < 80 || (len(candidates) > 1 && candidates[0].score-candidates[1].score < 7) {
		return mbRecording{}, false, nil
	}
	best := candidates[0].recording
	best.Score = candidates[0].score
	return best, true, nil
}

func scoreMusicRecording(item domain.MediaItem, recording mbRecording, remoteSearchScore int) int {
	localTitle := normalizeMusicMatchText(item.Title)
	remoteTitle := normalizeMusicMatchText(recording.Title)
	if localTitle == "" || remoteTitle == "" {
		return 0
	}
	score := 0
	similarity := musicTokenSimilarity(localTitle, remoteTitle)
	switch {
	case localTitle == remoteTitle:
		score += 55
	case similarity >= 0.92:
		score += 48
	case similarity >= 0.82 && strings.Contains(remoteTitle, localTitle):
		score += 40
	default:
		return 0
	}
	localArtist := normalizeMusicMatchText(item.Artist)
	remoteArtist := normalizeMusicMatchText(mbArtistCreditName(recording.ArtistCredit))
	if localArtist != "" {
		artistSimilarity := musicTokenSimilarity(localArtist, remoteArtist)
		switch {
		case localArtist == remoteArtist:
			score += 25
		case artistSimilarity >= 0.88:
			score += 20
		case remoteArtist != "" && (strings.Contains(remoteArtist, localArtist) || strings.Contains(localArtist, remoteArtist)):
			score += 14
		default:
			return 0
		}
	}
	if item.Technical.DurationMS > 0 && recording.Length > 0 {
		diff := item.Technical.DurationMS - recording.Length
		if diff < 0 {
			diff = -diff
		}
		ratio := float64(diff) / float64(item.Technical.DurationMS)
		switch {
		case diff <= 2500:
			score += 20
		case diff <= 5000:
			score += 14
		case ratio <= 0.03:
			score += 9
		case ratio > 0.08:
			return 0
		}
	}
	if remoteSearchScore > 0 {
		if remoteSearchScore > 100 {
			remoteSearchScore = 100
		}
		score += remoteSearchScore / 10
	}
	return score
}

func (p *MusicMetadataProvider) fetchRecording(ctx context.Context, recordingID string) (mbRecording, error) {
	endpoint := p.mbURL + "/recording/" + url.PathEscape(recordingID) + "?fmt=json&inc=artist-credits+releases+genres+tags"
	var result mbRecording
	if err := p.musicBrainzJSON(ctx, endpoint, &result); err != nil {
		return mbRecording{}, err
	}
	return result, nil
}

func (p *MusicMetadataProvider) fetchReleaseGroup(ctx context.Context, releaseGroupID string) (mbReleaseGroup, error) {
	if strings.TrimSpace(releaseGroupID) == "" {
		return mbReleaseGroup{}, domain.ErrNotFound
	}
	endpoint := p.mbURL + "/release-group/" + url.PathEscape(releaseGroupID) + "?fmt=json&inc=artist-credits+genres+tags"
	var result mbReleaseGroup
	if err := p.musicBrainzJSON(ctx, endpoint, &result); err != nil {
		return mbReleaseGroup{}, err
	}
	return result, nil
}

func (p *MusicMetadataProvider) musicBrainzJSON(ctx context.Context, endpoint string, target any) error {
	p.mbMu.Lock()
	defer p.mbMu.Unlock()
	if wait := time.Until(p.lastMBCall.Add(1100 * time.Millisecond)); wait > 0 {
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", p.userAgent())
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	p.lastMBCall = p.now()
	if resp.StatusCode == http.StatusNotFound {
		return domain.ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("MusicBrainz returned %s", resp.Status)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 12<<20)).Decode(target); err != nil {
		return err
	}
	return nil
}

func (p *MusicMetadataProvider) userAgent() string {
	version := strings.TrimSpace(p.version)
	if version == "" {
		version = "dev"
	}
	return "KoduMedia/" + version + " (self-hosted music metadata client)"
}

func (p *MusicMetadataProvider) buildMatch(ctx context.Context, item domain.MediaItem, recording mbRecording, acoustID, method string) MusicMetadataMatch {
	release := chooseMusicBrainzRelease(item, recording.Releases)
	releaseGroup := release.ReleaseGroup
	if releaseGroup.ID != "" && len(releaseGroup.Genres) == 0 && len(releaseGroup.Tags) == 0 {
		if details, err := p.fetchReleaseGroup(ctx, releaseGroup.ID); err == nil {
			if details.Title == "" {
				details.Title = releaseGroup.Title
			}
			releaseGroup = details
		}
	}
	genres, styles := musicBrainzClassification(recording.Genres, recording.Tags, releaseGroup)
	return MusicMetadataMatch{
		RecordingID:    recording.ID,
		ReleaseID:      release.ID,
		ReleaseGroupID: releaseGroup.ID,
		AcoustID:       acoustID,
		Title:          strings.TrimSpace(recording.Title),
		Artist:         mbArtistCreditName(recording.ArtistCredit),
		Album:          strings.TrimSpace(release.Title),
		ReleaseStatus:  strings.TrimSpace(release.Status),
		Genres:         genres,
		Styles:         styles,
		ArtworkSource:  "cover-art-archive",
		Source:         "musicbrainz",
		SourceURL:      "https://musicbrainz.org/recording/" + recording.ID,
		Method:         method,
		Score:          recording.Score,
	}
}

func chooseMusicBrainzRelease(item domain.MediaItem, releases []mbRelease) mbRelease {
	if len(releases) == 0 {
		return mbRelease{}
	}
	type scored struct {
		release mbRelease
		score   int
	}
	localAlbum := normalizeMusicMatchText(item.Album)
	localArtist := normalizeMusicMatchText(item.Artist)
	values := make([]scored, 0, len(releases))
	for _, release := range releases {
		score := 0
		album := normalizeMusicMatchText(release.Title)
		groupTitle := normalizeMusicMatchText(release.ReleaseGroup.Title)
		if localAlbum != "" {
			similarity := maxFloat(musicTokenSimilarity(localAlbum, album), musicTokenSimilarity(localAlbum, groupTitle))
			switch {
			case localAlbum == album || localAlbum == groupTitle:
				score += 60
			case similarity >= 0.9:
				score += 48
			case similarity >= 0.78:
				score += 28
			default:
				score -= 20
			}
		} else if musicTokenSimilarity(normalizeMusicMatchText(item.Title), album) >= 0.88 {
			score += 24
		}
		if localArtist != "" {
			credit := normalizeMusicMatchText(mbArtistCreditName(release.ArtistCredit))
			if credit == "" || credit == localArtist || musicTokenSimilarity(credit, localArtist) >= 0.85 {
				score += 10
			}
		}
		switch strings.ToLower(strings.TrimSpace(release.Status)) {
		case "official":
			score += 8
		case "promotion":
			score += 5
		case "bootleg":
			if containsVersionWord(item.Title, "bootleg") || containsVersionWord(item.Album, "bootleg") {
				score += 14
			}
		}
		values = append(values, scored{release: release, score: score})
	}
	sort.SliceStable(values, func(i, j int) bool { return values[i].score > values[j].score })
	if localAlbum != "" && values[0].score < 30 {
		return mbRelease{}
	}
	return values[0].release
}

func musicBrainzClassification(recordingGenres, recordingTags []mbTag, releaseGroup mbReleaseGroup) ([]string, []string) {
	genres := rankedMBValues(recordingGenres, 3, nil)
	if len(genres) == 0 {
		genres = rankedMBValues(releaseGroup.Genres, 3, nil)
	}
	genreSet := map[string]struct{}{}
	for _, value := range genres {
		genreSet[strings.ToLower(value)] = struct{}{}
	}
	styleTags := append([]mbTag(nil), recordingTags...)
	styleTags = append(styleTags, releaseGroup.Tags...)
	styles := rankedMBValues(styleTags, 5, func(value string) bool {
		key := strings.ToLower(strings.TrimSpace(value))
		if _, duplicate := genreSet[key]; duplicate {
			return false
		}
		if key == "" || isGenericMusicTag(key) {
			return false
		}
		return true
	})
	for _, value := range releaseGroup.SecondaryTypes {
		if normalized := strings.TrimSpace(value); normalized != "" && !containsFold(styles, normalized) {
			switch strings.ToLower(normalized) {
			case "remix", "dj-mix", "mixtape/street", "live":
				styles = append(styles, normalized)
			}
		}
	}
	if len(styles) > 5 {
		styles = styles[:5]
	}
	return normalizeMusicValues(genres), normalizeMusicValues(styles)
}

func rankedMBValues(values []mbTag, limit int, accept func(string) bool) []string {
	copyValues := append([]mbTag(nil), values...)
	sort.SliceStable(copyValues, func(i, j int) bool {
		if copyValues[i].Count != copyValues[j].Count {
			return copyValues[i].Count > copyValues[j].Count
		}
		return strings.ToLower(copyValues[i].Name) < strings.ToLower(copyValues[j].Name)
	})
	result := make([]string, 0, limit)
	seen := map[string]struct{}{}
	for _, value := range copyValues {
		name := strings.TrimSpace(value.Name)
		key := strings.ToLower(name)
		if name == "" {
			continue
		}
		if accept != nil && !accept(name) {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, name)
		if len(result) >= limit {
			break
		}
	}
	return result
}

func isGenericMusicTag(value string) bool {
	switch value {
	case "music", "seen live", "favorites", "favourite", "favorite", "albums i own", "spotify", "official", "single", "album", "compilation", "male vocalists", "female vocalists", "instrumental":
		return true
	}
	return false
}

func mbArtistCreditName(values []mbArtistCredit) string {
	var result strings.Builder
	for _, value := range values {
		name := strings.TrimSpace(value.Name)
		if name == "" {
			name = strings.TrimSpace(value.Artist.Name)
		}
		result.WriteString(name)
		result.WriteString(value.JoinPhrase)
	}
	return strings.TrimSpace(result.String())
}

func escapeMBQuery(value string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value)
}

func normalizeMusicMatchText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	parts := strings.FieldsFunc(value, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	filtered := parts[:0]
	for _, part := range parts {
		switch part {
		case "official", "audio", "video", "hq", "hd", "remastered":
			continue
		}
		filtered = append(filtered, part)
	}
	return strings.Join(filtered, " ")
}

func musicTokenSimilarity(left, right string) float64 {
	leftSet, rightSet := map[string]struct{}{}, map[string]struct{}{}
	for _, value := range strings.Fields(left) {
		leftSet[value] = struct{}{}
	}
	for _, value := range strings.Fields(right) {
		rightSet[value] = struct{}{}
	}
	if len(leftSet) == 0 || len(rightSet) == 0 {
		return 0
	}
	intersection := 0
	for value := range leftSet {
		if _, ok := rightSet[value]; ok {
			intersection++
		}
	}
	union := len(leftSet) + len(rightSet) - intersection
	return float64(intersection) / float64(union)
}

func maxFloat(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}

func containsVersionWord(value, word string) bool {
	for _, token := range strings.Fields(normalizeMusicMatchText(value)) {
		if token == word {
			return true
		}
	}
	return false
}

func (p *MusicMetadataProvider) fingerprint(ctx context.Context, item domain.MediaItem) (fingerprintCacheEntry, error) {
	key := stableID("music-fingerprint", filepath.Clean(item.File.Path), strconv.FormatInt(item.File.SizeBytes, 10), item.File.ModifiedAt.UTC().Format(time.RFC3339Nano))
	path := filepath.Join(p.cacheDir, "fingerprints", key+".json")
	if payload, err := os.ReadFile(path); err == nil {
		var cached fingerprintCacheEntry
		if json.Unmarshal(payload, &cached) == nil && cached.SchemaVersion == fingerprintCacheSchema && cached.Duration > 0 && cached.Fingerprint != "" {
			return cached, nil
		}
	}
	binary, err := p.resolveFpcalc(ctx)
	if err != nil {
		return fingerprintCacheEntry{}, err
	}
	command := exec.CommandContext(ctx, binary, "-json", item.File.Path)
	childprocess.Hide(command)
	output, err := command.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fingerprintCacheEntry{}, fmt.Errorf("fpcalc failed: %w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return fingerprintCacheEntry{}, fmt.Errorf("fpcalc failed: %w", err)
	}
	var result struct {
		Duration    float64 `json:"duration"`
		Fingerprint string  `json:"fingerprint"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return fingerprintCacheEntry{}, fmt.Errorf("decode fpcalc output: %w", err)
	}
	duration := int(result.Duration + 0.5)
	if duration <= 0 && item.Technical.DurationMS > 0 {
		duration = int(item.Technical.DurationMS / 1000)
	}
	if duration <= 0 || strings.TrimSpace(result.Fingerprint) == "" {
		return fingerprintCacheEntry{}, errors.New("fpcalc returned an empty fingerprint")
	}
	entry := fingerprintCacheEntry{SchemaVersion: fingerprintCacheSchema, CreatedAt: p.now().UTC(), Duration: duration, Fingerprint: strings.TrimSpace(result.Fingerprint)}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err == nil {
		if payload, marshalErr := json.MarshalIndent(entry, "", "  "); marshalErr == nil {
			_ = os.WriteFile(path, append(payload, '\n'), 0o600)
		}
	}
	return entry, nil
}

func (p *MusicMetadataProvider) resolveFpcalc(ctx context.Context) (string, error) {
	if configured := strings.TrimSpace(p.fpcalcPath); configured != "" {
		if fileExists(configured) {
			return configured, nil
		}
		return "", fmt.Errorf("configured fpcalc was not found: %s", configured)
	}
	if path, err := exec.LookPath("fpcalc"); err == nil {
		return path, nil
	}
	if runtime.GOOS != "windows" {
		return "", errors.New("fpcalc is not installed")
	}
	return p.ensureWindowsFpcalc(ctx)
}

func (p *MusicMetadataProvider) ensureWindowsFpcalc(ctx context.Context) (string, error) {
	p.toolMu.Lock()
	defer p.toolMu.Unlock()
	target := filepath.Join(p.cacheDir, "tools", "chromaprint-"+fpcalcVersion, "fpcalc.exe")
	if fileExists(target) {
		return target, nil
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, p.fpcalcURL, nil)
	req.Header.Set("User-Agent", p.userAgent())
	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download fpcalc: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download fpcalc returned %s", resp.Status)
	}
	archive, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(archive)
	if !strings.EqualFold(hex.EncodeToString(sum[:]), p.fpcalcSHA) {
		return "", errors.New("fpcalc archive checksum did not match the pinned official release")
	}
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return "", err
	}
	var executable []byte
	var license []byte
	for _, file := range reader.File {
		base := strings.ToLower(filepath.Base(file.Name))
		if base != "fpcalc.exe" && !strings.Contains(base, "license") && !strings.Contains(base, "copying") {
			continue
		}
		stream, openErr := file.Open()
		if openErr != nil {
			continue
		}
		data, readErr := io.ReadAll(io.LimitReader(stream, 16<<20))
		_ = stream.Close()
		if readErr != nil {
			continue
		}
		if base == "fpcalc.exe" {
			executable = data
		} else if len(license) == 0 {
			license = data
		}
	}
	if len(executable) == 0 {
		return "", errors.New("official Chromaprint archive did not contain fpcalc.exe")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}
	temporary := target + ".tmp"
	if err := os.WriteFile(temporary, executable, 0o700); err != nil {
		return "", err
	}
	if err := os.Rename(temporary, target); err != nil {
		_ = os.Remove(temporary)
		return "", err
	}
	if len(license) > 0 {
		_ = os.WriteFile(filepath.Join(filepath.Dir(target), "LICENSE.txt"), license, 0o600)
	}
	return target, nil
}

func (p *MusicMetadataProvider) readLookupCache(key string) (musicMetadataCacheEntry, bool) {
	payload, err := os.ReadFile(filepath.Join(p.cacheDir, "lookups", key+".json"))
	if err != nil {
		return musicMetadataCacheEntry{}, false
	}
	var entry musicMetadataCacheEntry
	if json.Unmarshal(payload, &entry) != nil || entry.SchemaVersion != musicMetadataCacheSchema {
		return musicMetadataCacheEntry{}, false
	}
	ttl := 180 * 24 * time.Hour
	if entry.Miss {
		ttl = 7 * 24 * time.Hour
	}
	if p.now().Sub(entry.FetchedAt) > ttl {
		return musicMetadataCacheEntry{}, false
	}
	return entry, true
}

func (p *MusicMetadataProvider) writeLookupCache(key string, entry musicMetadataCacheEntry) error {
	directory := filepath.Join(p.cacheDir, "lookups")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(directory, key+".json"), append(payload, '\n'), 0o600)
}

func (p *MusicMetadataProvider) FetchArtwork(ctx context.Context, item domain.MediaItem, target string) (MusicMetadataMatch, error) {
	match := MusicMetadataMatch{
		RecordingID:    item.ExternalIDs.MusicBrainzRecording,
		ReleaseID:      item.ExternalIDs.MusicBrainzRelease,
		ReleaseGroupID: item.ExternalIDs.MusicBrainzReleaseGroup,
		AcoustID:       item.ExternalIDs.AcoustID,
	}
	if match.ReleaseID == "" && match.ReleaseGroupID == "" {
		lookupCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
		resolved, found, err := p.Lookup(lookupCtx, item)
		cancel()
		if err != nil || !found {
			if err != nil {
				return MusicMetadataMatch{}, err
			}
			return MusicMetadataMatch{}, domain.ErrNotFound
		}
		match = resolved
	}
	endpoints := make([]string, 0, 2)
	if match.ReleaseID != "" {
		endpoints = append(endpoints, "https://coverartarchive.org/release/"+url.PathEscape(match.ReleaseID))
	}
	if match.ReleaseGroupID != "" {
		endpoints = append(endpoints, "https://coverartarchive.org/release-group/"+url.PathEscape(match.ReleaseGroupID))
	}
	var lastErr error = domain.ErrNotFound
	for _, endpoint := range endpoints {
		imageURL, err := p.coverArtURL(ctx, endpoint)
		if err != nil {
			lastErr = err
			continue
		}
		if err := p.downloadArtwork(ctx, imageURL, target); err != nil {
			lastErr = err
			continue
		}
		match.ArtworkSource = "cover-art-archive"
		return match, nil
	}
	if match.ArtworkURL != "" {
		if err := p.downloadArtwork(ctx, match.ArtworkURL, target); err == nil {
			return match, nil
		} else {
			lastErr = err
		}
	}
	if p.settings().CatalogFallbackEnabled && match.Source != "deezer" {
		if catalogMatch, found, err := p.lookupDeezer(ctx, item); err == nil && found && catalogMatch.ArtworkURL != "" {
			if downloadErr := p.downloadArtwork(ctx, catalogMatch.ArtworkURL, target); downloadErr == nil {
				return catalogMatch, nil
			} else {
				lastErr = downloadErr
			}
		} else if err != nil {
			lastErr = err
		}
	}
	return MusicMetadataMatch{}, lastErr
}

func (p *MusicMetadataProvider) coverArtURL(ctx context.Context, endpoint string) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", p.userAgent())
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", domain.ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Cover Art Archive returned %s", resp.Status)
	}
	var payload struct {
		Images []struct {
			Front      bool              `json:"front"`
			Image      string            `json:"image"`
			Thumbnails map[string]string `json:"thumbnails"`
		} `json:"images"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&payload); err != nil {
		return "", err
	}
	for _, image := range payload.Images {
		if !image.Front {
			continue
		}
		for _, key := range []string{"500", "1200", "large"} {
			if value := strings.TrimSpace(image.Thumbnails[key]); value != "" {
				return value, nil
			}
		}
		if value := strings.TrimSpace(image.Image); value != "" {
			return value, nil
		}
	}
	return "", domain.ErrNotFound
}

func (p *MusicMetadataProvider) downloadArtwork(ctx context.Context, source, target string) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	req.Header.Set("Accept", "image/*")
	req.Header.Set("User-Agent", p.userAgent())
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("artwork download returned %s", resp.Status)
	}
	if contentType := strings.ToLower(resp.Header.Get("Content-Type")); contentType != "" && !strings.HasPrefix(contentType, "image/") {
		return errors.New("artwork response was not an image")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	temporary := target + ".tmp"
	file, err := os.OpenFile(temporary, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(file, io.LimitReader(resp.Body, 12<<20+1))
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(temporary)
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	}
	if written > 12<<20 {
		_ = os.Remove(temporary)
		return errors.New("artwork response was larger than 12 MiB")
	}
	normalized := target + ".normalized.jpg"
	_ = os.Remove(normalized)
	defer os.Remove(normalized)
	binary := "ffmpeg"
	if p.ffmpeg != nil && strings.TrimSpace(p.ffmpeg()) != "" {
		binary = strings.TrimSpace(p.ffmpeg())
	}
	command := exec.CommandContext(
		ctx, binary,
		"-v", "error", "-i", temporary, "-frames:v", "1",
		"-vf", "scale=800:800:force_original_aspect_ratio=decrease",
		"-q:v", "3", "-y", normalized,
	)
	childprocess.Hide(command)
	if output, normalizeErr := command.CombinedOutput(); normalizeErr != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("normalize artwork: %w: %s", normalizeErr, strings.TrimSpace(string(output)))
	}
	_ = os.Remove(temporary)
	if err := os.Rename(normalized, target); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	return nil
}

func (p *MusicMetadataProvider) MaintainItems(ctx context.Context, items []domain.MediaItem, maximumBytes int64) int64 {
	if p == nil {
		return 0
	}
	before := directorySizeForMaintenance(p.cacheDir)
	validLookups := map[string]struct{}{}
	validFingerprints := map[string]struct{}{}
	validAnalysis := map[string]struct{}{}
	for _, item := range items {
		if err := ctx.Err(); err != nil {
			break
		}
		if item.Type != domain.MediaTypeMusic {
			continue
		}
		validLookups[p.lookupKey(item)+".json"] = struct{}{}
		fingerprintKey := stableID("music-fingerprint", filepath.Clean(item.File.Path), strconv.FormatInt(item.File.SizeBytes, 10), item.File.ModifiedAt.UTC().Format(time.RFC3339Nano))
		validFingerprints[fingerprintKey+".json"] = struct{}{}
		analysisKey := stableID("music-audio-analysis-v1", filepath.Clean(item.File.Path), strconv.FormatInt(item.File.SizeBytes, 10), item.File.ModifiedAt.UTC().Format(time.RFC3339Nano))
		validAnalysis[analysisKey+".json"] = struct{}{}
	}
	pruneNamedCache(filepath.Join(p.cacheDir, "lookups"), validLookups, true)
	pruneNamedCache(filepath.Join(p.cacheDir, "fingerprints"), validFingerprints, false)
	pruneNamedCache(filepath.Join(p.cacheDir, "audio-analysis"), validAnalysis, false)
	enforceMusicMetadataLimit(p.cacheDir, maximumBytes)
	after := directorySizeForMaintenance(p.cacheDir)
	if after >= before {
		return 0
	}
	return before - after
}

func pruneNamedCache(root string, valid map[string]struct{}, expireLookups bool) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name())
		if _, keep := valid[entry.Name()]; !keep {
			_ = os.Remove(path)
			continue
		}
		if expireLookups {
			if info, infoErr := entry.Info(); infoErr == nil && now.Sub(info.ModTime()) > 180*24*time.Hour {
				_ = os.Remove(path)
			}
		}
	}
}

func enforceMusicMetadataLimit(root string, maximumBytes int64) {
	if maximumBytes <= 0 || directorySizeForMaintenance(root) <= maximumBytes {
		return
	}
	type candidate struct {
		path string
		size int64
		time time.Time
	}
	var files []candidate
	var total int64
	for _, name := range []string{"lookups", "audio-analysis", "fingerprints"} {
		directory := filepath.Join(root, name)
		entries, _ := os.ReadDir(directory)
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if info, err := entry.Info(); err == nil {
				total += info.Size()
				files = append(files, candidate{filepath.Join(directory, entry.Name()), info.Size(), info.ModTime()})
			}
		}
	}
	sort.SliceStable(files, func(i, j int) bool { return files[i].time.Before(files[j].time) })
	for _, file := range files {
		if total <= maximumBytes {
			break
		}
		if os.Remove(file.path) == nil {
			total -= file.size
		}
	}
}
