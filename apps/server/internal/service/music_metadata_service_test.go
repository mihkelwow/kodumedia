package service

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

func TestMusicMetadataEnrichesOnlyMissingClassificationAndCachesMatch(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/recording/":
			fmt.Fprint(w, `{"recordings":[{"id":"recording-id","title":"Night Drive (Bootleg)","length":210000,"score":100,"artist-credit":[{"name":"Artist","joinphrase":""}]}]}`)
		case "/recording/recording-id":
			fmt.Fprint(w, `{"id":"recording-id","title":"Night Drive (Bootleg)","length":210000,"artist-credit":[{"name":"Artist","joinphrase":""}],"releases":[{"id":"release-id","title":"Unreleased Edits","status":"Bootleg","artist-credit":[{"name":"Artist","joinphrase":""}],"release-group":{"id":"group-id","title":"Unreleased Edits","secondary-types":["Remix"]}}]}`)
		case "/release-group/group-id":
			fmt.Fprint(w, `{"id":"group-id","title":"Unreleased Edits","genres":[{"name":"Electronic","count":8}],"tags":[{"name":"Progressive House","count":6},{"name":"seen live","count":99}],"secondary-types":["Remix"]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider := NewMusicMetadataProvider(
		func() domain.MusicSettings { return domain.MusicSettings{MetadataEnabled: true} },
		t.TempDir(),
		"test",
	)
	provider.mbURL = server.URL
	provider.now = func() time.Time { return time.Unix(1_700_000_000, 0) }

	item := domain.MediaItem{
		Type:   domain.MediaTypeMusic,
		Title:  "Night Drive (Bootleg)",
		Artist: "Artist",
		Album:  "Unreleased Edits",
		File: domain.MediaFile{
			Path:       filepath.Join(t.TempDir(), "Night Drive.flac"),
			SizeBytes:  42_000_000,
			ModifiedAt: time.Unix(1_699_000_000, 0),
		},
		Technical: domain.TechnicalMetadata{DurationMS: 210000},
		MusicClassification: &domain.MusicClassification{
			Genres:      []string{"Electronic"},
			GenreSource: "embedded",
			Source:      "embedded",
			Confidence:  "tagged",
		},
	}

	enriched, changed, err := provider.Enrich(context.Background(), item)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected missing style and identifiers to be enriched")
	}
	if enriched.ExternalIDs.MusicBrainzRecording != "recording-id" || enriched.ExternalIDs.MusicBrainzRelease != "release-id" || enriched.ExternalIDs.MusicBrainzReleaseGroup != "group-id" {
		t.Fatalf("unexpected external identifiers: %+v", enriched.ExternalIDs)
	}
	classification := enriched.MusicClassification
	if classification == nil || len(classification.Genres) != 1 || classification.Genres[0] != "Electronic" {
		t.Fatalf("embedded genre was not preserved: %+v", classification)
	}
	if classification.StyleSource != "musicbrainz" || classification.Source != "mixed" || !containsFold(classification.Styles, "Progressive House") || !containsFold(classification.Styles, "Remix") {
		t.Fatalf("unexpected enriched classification: %+v", classification)
	}
	if containsFold(classification.Styles, "seen live") {
		t.Fatalf("generic MusicBrainz tag leaked into styles: %+v", classification.Styles)
	}

	firstRequestCount := requests.Load()
	if firstRequestCount != 3 {
		t.Fatalf("expected one search and two exact lookups, got %d requests", firstRequestCount)
	}
	if _, _, err := provider.Enrich(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != firstRequestCount {
		t.Fatal("cached match should avoid repeated MusicBrainz requests")
	}
}

func TestMusicMetadataNegativeCacheAvoidsRepeatedSearch(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"recordings":[]}`)
	}))
	defer server.Close()

	provider := NewMusicMetadataProvider(func() domain.MusicSettings {
		return domain.MusicSettings{MetadataEnabled: true}
	}, t.TempDir(), "test")
	provider.mbURL = server.URL
	provider.now = func() time.Time { return time.Unix(1_700_000_000, 0) }
	item := domain.MediaItem{
		Type:  domain.MediaTypeMusic,
		Title: "Unknown Edit",
		File:  domain.MediaFile{Path: filepath.Join(t.TempDir(), "unknown.flac"), ModifiedAt: time.Unix(1_699_000_000, 0)},
	}

	for range 2 {
		if _, found, err := provider.Lookup(context.Background(), item); err != nil || found {
			t.Fatalf("unexpected lookup result: found=%v err=%v", found, err)
		}
	}
	if requests.Load() != 1 {
		t.Fatalf("negative cache should make one request, got %d", requests.Load())
	}
}

func TestMusicMetadataUsesCatalogFallbackAndCachesArtworkCandidate(t *testing.T) {
	var deezerRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/mb/recording/":
			fmt.Fprint(w, `{"recordings":[]}`)
		case "/deezer/search/track":
			deezerRequests.Add(1)
			fmt.Fprint(w, `{"data":[{"id":42,"title":"Night Drive (Deep House Remix)","title_short":"Night Drive","title_version":"Deep House Remix","duration":210,"rank":900000,"artist":{"id":1,"name":"Artist"},"album":{"id":7,"title":"Night Drive Remixes","cover_xl":"https://images.example/cover.jpg"}}]}`)
		case "/deezer/track/42":
			deezerRequests.Add(1)
			fmt.Fprint(w, `{"id":42,"title":"Night Drive (Deep House Remix)","title_short":"Night Drive","title_version":"Deep House Remix","duration":210,"bpm":124,"link":"https://www.deezer.com/track/42","artist":{"id":1,"name":"Artist"},"album":{"id":7,"title":"Night Drive Remixes","cover_xl":"https://images.example/cover.jpg"}}`)
		case "/deezer/album/7":
			deezerRequests.Add(1)
			fmt.Fprint(w, `{"id":7,"title":"Night Drive Remixes","cover_xl":"https://images.example/cover.jpg","genres":{"data":[{"id":106,"name":"Dance"}]}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider := NewMusicMetadataProvider(func() domain.MusicSettings {
		return domain.MusicSettings{MetadataEnabled: true, CatalogFallbackEnabled: true}
	}, t.TempDir(), "test")
	provider.mbURL = server.URL + "/mb"
	provider.deezerURL = server.URL + "/deezer"
	provider.now = func() time.Time { return time.Unix(1_700_000_000, 0) }
	item := domain.MediaItem{
		Type: domain.MediaTypeMusic, Title: "Night Drive (Deep House Remix)", Artist: "Artist",
		File:      domain.MediaFile{Path: filepath.Join(t.TempDir(), "night-drive.flac"), SizeBytes: 42_000_000, ModifiedAt: time.Unix(1_699_000_000, 0)},
		Technical: domain.TechnicalMetadata{DurationMS: 210000},
	}

	match, found, err := provider.Lookup(context.Background(), item)
	if err != nil || !found {
		t.Fatalf("expected Deezer fallback match, found=%v err=%v", found, err)
	}
	if match.Source != "deezer" || match.ArtworkSource != "deezer" || match.ArtworkURL != "https://images.example/cover.jpg" {
		t.Fatalf("unexpected fallback match: %+v", match)
	}
	if !containsFold(match.Genres, "Electronic") || len(match.Styles) != 0 {
		t.Fatalf("expected normalized catalog genre without a falsely attributed style: %+v", match)
	}
	firstCount := deezerRequests.Load()
	if firstCount != 3 {
		t.Fatalf("expected search, track and album requests, got %d", firstCount)
	}
	if _, found, err := provider.Lookup(context.Background(), item); err != nil || !found {
		t.Fatalf("unexpected cached lookup result, found=%v err=%v", found, err)
	}
	if deezerRequests.Load() != firstCount {
		t.Fatal("catalog fallback match should be cached")
	}
	enriched, changed, err := provider.Enrich(context.Background(), item)
	if err != nil || !changed || enriched.MusicClassification == nil {
		t.Fatalf("expected catalog genre and locally inferred style, changed=%v err=%v item=%+v", changed, err, enriched)
	}
	if enriched.MusicClassification.GenreSource != "deezer" || enriched.MusicClassification.StyleSource != "local-inference" || !containsFold(enriched.MusicClassification.Styles, "Deep House") {
		t.Fatalf("unexpected source attribution: %+v", enriched.MusicClassification)
	}
}

func TestInferredClassificationUsesFilenameAndNeverLeavesFieldsEmpty(t *testing.T) {
	item := domain.MediaItem{
		Type:  domain.MediaTypeMusic,
		Title: "Sunrise (Liquid DnB Bootleg)",
		File:  domain.MediaFile{Name: "Sunrise (Liquid DnB Bootleg).flac", Path: filepath.Join("Music", "Drum and Bass", "Sunrise.flac")},
	}
	classification := inferMusicClassification(item, nil, nil, 0)
	if !containsFold(classification.Genres, "Electronic") || !containsFold(classification.Styles, "Liquid Drum & Bass") {
		t.Fatalf("unexpected filename inference: %+v", classification)
	}

	unknown := inferMusicClassification(domain.MediaItem{Type: domain.MediaTypeMusic, Title: "Untitled 001"}, nil, nil, 0)
	if len(unknown.Genres) == 0 || len(unknown.Styles) == 0 || unknown.Source != "fallback" || unknown.Confidence != "low" {
		t.Fatalf("unknown tracks must receive an explicit low-confidence fallback: %+v", unknown)
	}
}

func TestAudioAnalysisCanReplaceOnlyBroadInferredStyle(t *testing.T) {
	item := domain.MediaItem{Type: domain.MediaTypeMusic, Title: "Night Drive"}
	base := inferMusicClassification(item, []string{"Electronic"}, nil, 0)
	if !isBroadInferredStyle(base) || !needsAudioClassification(domain.MusicClassification{Genres: []string{"Electronic"}}, MusicMetadataMatch{}, base) {
		t.Fatalf("broad parent style should request audio analysis: %+v", base)
	}
	audio := inferredMusicClassification{Genres: []string{"Electronic"}, Styles: []string{"House"}, Source: "audio-analysis", Confidence: "estimated"}
	merged := mergeInferredClassification(base, audio)
	if len(merged.Styles) != 1 || merged.Styles[0] != "House" || merged.Source != "audio-analysis" {
		t.Fatalf("audio analysis should replace a broad default style: %+v", merged)
	}
}

func TestAutomaticFallbackClassificationCanBeUpgradedWithoutOverwritingTags(t *testing.T) {
	values, source, changed := chooseMusicClassificationField(
		[]string{"Other"}, "fallback",
		[]string{"Electronic"}, "deezer",
		[]string{"House"}, "audio-analysis",
	)
	if !changed || source != "deezer" || len(values) != 1 || values[0] != "Electronic" {
		t.Fatalf("catalog metadata should upgrade fallback values: values=%v source=%q changed=%v", values, source, changed)
	}

	values, source, changed = chooseMusicClassificationField(
		[]string{"Other"}, "fallback",
		nil, "",
		[]string{"House"}, "audio-analysis",
	)
	if !changed || source != "audio-analysis" || len(values) != 1 || values[0] != "House" {
		t.Fatalf("audio analysis should upgrade fallback values: values=%v source=%q changed=%v", values, source, changed)
	}

	values, source, changed = chooseMusicClassificationField(
		[]string{"Progressive House"}, "embedded",
		[]string{"Electronic"}, "deezer",
		[]string{"House"}, "audio-analysis",
	)
	if changed || source != "embedded" || len(values) != 1 || values[0] != "Progressive House" {
		t.Fatalf("embedded tags must remain protected: values=%v source=%q changed=%v", values, source, changed)
	}
}

func TestShazamFingerprintRunsBeforeCatalogLookupsAndIsCached(t *testing.T) {
	var shazamCalls atomic.Int32
	var catalogCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		catalogCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"recordings":[]}`)
	}))
	defer server.Close()

	provider := NewMusicMetadataProvider(func() domain.MusicSettings {
		return domain.MusicSettings{MetadataEnabled: true, ShazamEnabled: true, CatalogFallbackEnabled: true}
	}, t.TempDir(), "test")
	provider.mbURL = server.URL
	provider.deezerURL = server.URL
	provider.shazamLookup = func(context.Context, domain.MediaItem) (MusicMetadataMatch, bool, error) {
		shazamCalls.Add(1)
		return MusicMetadataMatch{
			Title: "Night Drive", Artist: "Artist", Genres: []string{"Electronic"},
			Source: "shazam", Method: "shazam-fingerprint", ArtworkURL: "https://example.test/cover.jpg",
		}, true, nil
	}
	provider.now = func() time.Time { return time.Unix(1_700_000_000, 0) }
	item := domain.MediaItem{
		Type: domain.MediaTypeMusic, Title: "Unknown filename", Artist: "Unknown",
		File: domain.MediaFile{Path: filepath.Join(t.TempDir(), "track.flac"), SizeBytes: 1234, ModifiedAt: time.Unix(1_699_000_000, 0)},
	}

	for range 2 {
		match, found, err := provider.Lookup(context.Background(), item)
		if err != nil || !found || match.Source != "shazam" {
			t.Fatalf("unexpected Shazam lookup result: found=%v err=%v match=%+v", found, err, match)
		}
	}
	if shazamCalls.Load() != 1 {
		t.Fatalf("Shazam result should be cached, got %d recognizer calls", shazamCalls.Load())
	}
	if catalogCalls.Load() != 0 {
		t.Fatalf("catalog fallback must not run after a Shazam match, got %d calls", catalogCalls.Load())
	}
}

func TestShazamResponseProvidesGenreArtworkAndAlbum(t *testing.T) {
	var payload shazamResponse
	if err := json.Unmarshal([]byte(`{
		"matches":[{"id":"match"}],
		"track":{
			"key":"123",
			"title":"Recognized Song",
			"subtitle":"Recognized Artist",
			"genres":{"primary":"Hip-Hop/Rap"},
			"images":{"coverarthq":"https://example.test/cover-hq.jpg"},
			"sections":[{"type":"SONG","metadata":[{"title":"Album","text":"Recognized Album"}]}]
		}
	}`), &payload); err != nil {
		t.Fatal(err)
	}
	match, found, err := musicMatchFromShazamResponse(payload)
	if err != nil || !found {
		t.Fatalf("expected parsed Shazam match, found=%v err=%v", found, err)
	}
	if match.Source != "shazam" || match.Method != "shazam-fingerprint" || match.Album != "Recognized Album" || match.ArtworkURL != "https://example.test/cover-hq.jpg" {
		t.Fatalf("unexpected parsed Shazam metadata: %+v", match)
	}
	if len(match.Genres) != 1 || match.Genres[0] != "Hip-Hop" {
		t.Fatalf("compound Shazam genre should normalize without a stray Rap value: %+v", match.Genres)
	}
}

func TestAudioAnalysisCannotContradictFingerprintGenre(t *testing.T) {
	base := inferredMusicClassification{
		Genres: []string{"Electronic"}, Styles: []string{"Electronic"},
		Source: "local-inference", Confidence: "estimated",
	}
	audio := inferredMusicClassification{
		Genres: []string{"Hip-Hop"}, Styles: []string{"Hip-Hop"},
		Source: "audio-analysis", Confidence: "estimated",
	}
	merged := mergeInferredClassification(base, audio)
	if !equalFoldedMusicValues(merged.Genres, []string{"Electronic"}) || !equalFoldedMusicValues(merged.Styles, []string{"Electronic"}) || merged.Source != "local-inference" {
		t.Fatalf("conflicting low-confidence audio result must be ignored: %+v", merged)
	}
}

func TestShazamSignatureEncodingWritesHeaderSizeAndCRC(t *testing.T) {
	signature := shazamSignature{SampleRate: shazamSampleRate, NumberSamples: shazamSampleRate * 10}
	signature.Bands[1] = []shazamFrequencyPeak{{FFTPassNumber: 4, PeakMagnitude: 6200, CorrectedPeakFrequencyBin: 5000}}
	uri, err := encodeShazamSignatureURI(signature)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(uri, shazamDataURIPrefix) {
		t.Fatalf("unexpected signature URI prefix: %q", uri)
	}
	payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(uri, shazamDataURIPrefix))
	if err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint32(payload[0:4]) != 0xcafe2580 {
		t.Fatalf("unexpected signature magic: %x", payload[:4])
	}
	if got, want := binary.LittleEndian.Uint32(payload[8:12]), uint32(len(payload)-48); got != want {
		t.Fatalf("unexpected signature size: got %d want %d", got, want)
	}
	if got, want := binary.LittleEndian.Uint32(payload[4:8]), crc32.ChecksumIEEE(payload[8:]); got != want {
		t.Fatalf("unexpected signature CRC: got %08x want %08x", got, want)
	}
}
