package tmdb

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestProviderUsesOfficialMovieTitleAndMetadata(t *testing.T) {
	provider := New("secret-token", "et-EE")
	provider.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Header.Get("Authorization") != "Bearer secret-token" {
			t.Fatalf("unexpected authorization header: %q", request.Header.Get("Authorization"))
		}
		switch request.URL.Path {
		case "/3/search/movie":
			if got := request.URL.Query().Get("query"); got != "Blade Runner" {
				t.Fatalf("unexpected query: %q", got)
			}
			if got := request.URL.Query().Get("year"); got != "1982" {
				t.Fatalf("unexpected year: %q", got)
			}
			return jsonResponse(`{"results":[{"id":7,"title":"Blade Runner 2049","release_date":"2017-10-04","popularity":100},{"id":42,"title":"Blade Runner","original_title":"Blade Runner","release_date":"1982-06-25","popularity":20}]}`), nil
		case "/3/movie/42":
			if got := request.URL.Query().Get("append_to_response"); got != "credits,external_ids,images,videos" {
				t.Fatalf("unexpected append_to_response: %q", got)
			}
			if got := request.URL.Query().Get("include_image_language"); got != "et,en,null" {
				t.Fatalf("unexpected image language filter: %q", got)
			}
			return jsonResponse(`{"id":42,"imdb_id":"tt0083658","title":"Blade Runner","overview":"A blade runner must pursue replicants.","tagline":"Man has made his match.","release_date":"1982-06-25","poster_path":"/poster.jpg","backdrop_path":"/backdrop.jpg","genres":[{"name":"Science Fiction"}],"credits":{"cast":[{"name":"Harrison Ford","character":"Rick Deckard","profile_path":"/ford.jpg"}],"crew":[{"name":"Ridley Scott","job":"Director","profile_path":"/scott.jpg"}]},"external_ids":{},"images":{"logos":[{"file_path":"/blade-en.png","iso_639_1":"en","aspect_ratio":2.8,"width":1000,"vote_average":9},{"file_path":"/blade-et.png","iso_639_1":"et","aspect_ratio":2.7,"width":800,"vote_average":7}]},"videos":{"results":[{"key":"teaser42","site":"YouTube","type":"Teaser","official":true,"iso_639_1":"et","size":1080,"name":"Official teaser"},{"key":"trailer42","site":"YouTube","type":"Trailer","official":true,"iso_639_1":"en","size":1080,"name":"Official Trailer"}]}}`), nil
		default:
			t.Fatalf("unexpected TMDb path: %s", request.URL.Path)
			return nil, nil
		}
	})}

	result, err := provider.Match(context.Background(), domain.MetadataMatchQuery{
		Type:     domain.MediaTypeMovie,
		Title:    "Blade Runner",
		Year:     1982,
		Language: "et-EE",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Title != "Blade Runner" || result.Year != 1982 {
		t.Fatalf("unexpected official title/year: %+v", result)
	}
	if result.ExternalIDs.TMDb != "42" || result.ExternalIDs.IMDb != "tt0083658" {
		t.Fatalf("unexpected external ids: %+v", result.ExternalIDs)
	}
	if result.Metadata.MetadataSource != "tmdb" || result.Metadata.Summary == "" || result.Metadata.PosterURL == "" {
		t.Fatalf("metadata was not mapped: %+v", result.Metadata)
	}
	if len(result.Metadata.Cast) != 1 || result.Metadata.Cast[0].Name != "Harrison Ford" {
		t.Fatalf("cast was not mapped: %+v", result.Metadata.Cast)
	}
	if len(result.Metadata.Directors) != 1 || result.Metadata.Directors[0].Name != "Ridley Scott" {
		t.Fatalf("director was not mapped: %+v", result.Metadata.Directors)
	}
	if result.Metadata.LogoURL != "https://image.tmdb.org/t/p/w500/blade-et.png" {
		t.Fatalf("localized logo was not selected: %+v", result.Metadata)
	}
	if result.Metadata.TrailerYouTubeKey != "trailer42" || result.Metadata.TrailerName != "Official Trailer" {
		t.Fatalf("official trailer was not selected: %+v", result.Metadata)
	}
	if len(result.Metadata.TrailerCandidates) != 2 || result.Metadata.TrailerCandidates[0].YouTubeKey != "trailer42" || result.Metadata.TrailerCandidates[1].YouTubeKey != "teaser42" {
		t.Fatalf("ranked trailer candidates were not retained: %+v", result.Metadata.TrailerCandidates)
	}
	if result.Metadata.HeroAssetsVersion != 2 {
		t.Fatalf("hero asset version was not marked: %+v", result.Metadata)
	}
}

func TestProviderUsesParentSeriesIDForOfficialEpisodeTitle(t *testing.T) {
	searchRequests := 0
	provider := New("secret-token", "en-US")
	provider.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if strings.HasPrefix(request.URL.Path, "/3/search/") {
			searchRequests++
		}
		if request.URL.Path != "/3/tv/123/season/2/episode/3" {
			t.Fatalf("unexpected TMDb path: %s", request.URL.Path)
		}
		return jsonResponse(`{"id":987,"name":"Who Is Alive?","overview":"The team searches for answers.","air_date":"2025-01-31","still_path":"/still.jpg","credits":{"cast":[],"crew":[]},"external_ids":{"imdb_id":"tt0000987"}}`), nil
	})}

	result, err := provider.Match(context.Background(), domain.MetadataMatchQuery{
		Type:        domain.MediaTypeEpisode,
		SeriesTitle: "Severance",
		Season:      2,
		Episode:     3,
		ParentExternalIDs: domain.ExternalIDs{
			TMDb: "123",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if searchRequests != 0 {
		t.Fatalf("episode matching repeated series search %d times", searchRequests)
	}
	if result.Title != "Who Is Alive?" || result.ExternalIDs.TMDb != "987" || result.ExternalIDs.IMDb != "tt0000987" {
		t.Fatalf("unexpected episode result: %+v", result)
	}
	if result.Year != 2025 || result.Metadata.MetadataSource != "tmdb" {
		t.Fatalf("episode metadata was not mapped: %+v", result)
	}
}

func TestProviderTriesRankedFilenameCandidates(t *testing.T) {
	queries := make([]string, 0, 2)
	provider := New("secret-token", "en-US")
	provider.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Path {
		case "/3/search/movie":
			query := request.URL.Query().Get("query")
			queries = append(queries, query)
			switch query {
			case "Anime Time Suzume No Tojimari":
				return jsonResponse(`{"results":[]}`), nil
			case "Suzume No Tojimari":
				return jsonResponse(`{"results":[{"id":916224,"title":"Suzume","original_title":"すずめの戸締まり","release_date":"2022-11-11","poster_path":"/suzume.jpg","popularity":80}]}`), nil
			default:
				t.Fatalf("unexpected query: %q", query)
				return nil, nil
			}
		case "/3/movie/916224":
			return jsonResponse(`{"id":916224,"imdb_id":"tt16428256","title":"Suzume","overview":"A teenager helps close mysterious doors.","release_date":"2022-11-11","poster_path":"/suzume.jpg","backdrop_path":"/suzume-bg.jpg","genres":[],"credits":{"cast":[],"crew":[]},"external_ids":{}}`), nil
		default:
			t.Fatalf("unexpected TMDb path: %s", request.URL.Path)
			return nil, nil
		}
	})}

	result, err := provider.Match(context.Background(), domain.MetadataMatchQuery{
		Type: domain.MediaTypeMovie,
		Candidates: []domain.MetadataTitleCandidate{
			{Title: "Anime Time Suzume No Tojimari", Year: 2022},
			{Title: "Suzume No Tojimari", Year: 2022},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(queries) < 2 || queries[len(queries)-1] != "Suzume No Tojimari" {
		t.Fatalf("candidate fallback was not used: %#v", queries)
	}
	if result.Title != "Suzume" || result.ExternalIDs.TMDb != "916224" || result.Metadata.PosterURL == "" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestProviderFillsMissingLocalizedDescriptionAndArtworkFromEnglish(t *testing.T) {
	provider := New("secret-token", "et-EE")
	provider.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path == "/3/search/movie" {
			return jsonResponse(`{"results":[{"id":916224,"title":"Suzume","release_date":"2022-11-11","poster_path":"/search.jpg"}]}`), nil
		}
		if request.URL.Path != "/3/movie/916224" {
			t.Fatalf("unexpected TMDb path: %s", request.URL.Path)
		}
		switch request.URL.Query().Get("language") {
		case "et-EE":
			return jsonResponse(`{"id":916224,"title":"Suzume","overview":"","release_date":"2022-11-11","poster_path":"","backdrop_path":"","genres":[],"credits":{"cast":[],"crew":[]},"external_ids":{}}`), nil
		case "en-US":
			return jsonResponse(`{"id":916224,"title":"Suzume (English)","overview":"English fallback summary.","release_date":"2022-11-11","poster_path":"/poster.jpg","backdrop_path":"/backdrop.jpg","genres":[{"name":"Animation"}],"credits":{"cast":[],"crew":[]},"external_ids":{}}`), nil
		default:
			t.Fatalf("unexpected language: %q", request.URL.Query().Get("language"))
			return nil, nil
		}
	})}

	result, err := provider.Match(context.Background(), domain.MetadataMatchQuery{Type: domain.MediaTypeMovie, Title: "Suzume", Year: 2022})
	if err != nil {
		t.Fatal(err)
	}
	if result.Title != "Suzume" {
		t.Fatalf("localized title should be retained, got %q", result.Title)
	}
	if result.Metadata.Summary != "English fallback summary." || result.Metadata.PosterURL == "" || result.Metadata.BackdropURL == "" {
		t.Fatalf("fallback metadata was not filled: %+v", result.Metadata)
	}
}

func TestProviderRepairsMissingMetadataUsingExistingTMDbID(t *testing.T) {
	searchRequests := 0
	provider := New("secret-token", "en-US")
	provider.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if strings.HasPrefix(request.URL.Path, "/3/search/") {
			searchRequests++
			t.Fatalf("existing TMDb ID should avoid search")
		}
		if request.URL.Path != "/3/movie/916224" {
			t.Fatalf("unexpected TMDb path: %s", request.URL.Path)
		}
		return jsonResponse(`{"id":916224,"imdb_id":"tt16428256","title":"Suzume","overview":"Repaired summary.","release_date":"2022-11-11","poster_path":"/poster.jpg","backdrop_path":"/backdrop.jpg","genres":[],"credits":{"cast":[],"crew":[]},"external_ids":{}}`), nil
	})}

	result, err := provider.Match(context.Background(), domain.MetadataMatchQuery{
		Type:        domain.MediaTypeMovie,
		Title:       "Noisy local filename",
		ExternalIDs: domain.ExternalIDs{TMDb: "916224"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if searchRequests != 0 || result.Title != "Suzume" || result.Metadata.Summary != "Repaired summary." {
		t.Fatalf("unexpected repaired result: %+v (searches=%d)", result, searchRequests)
	}
}

func TestChooseBestResultDoesNotTreatSiloAsSubstringOfSilom(t *testing.T) {
	results := []searchResult{
		{ID: 999, Name: "Love of Silom", FirstAirDate: "2026-01-10", PosterPath: "/silom.jpg", Popularity: 120},
		{ID: 66732, Name: "Silo", FirstAirDate: "2023-05-04", PosterPath: "/silo.jpg", Popularity: 55},
	}
	best, score := chooseBestResult(results, "Silo", 2026)
	if best.ID != 66732 {
		t.Fatalf("expected exact word title Silo, got %#v (score %.2f)", best, score)
	}
}

func TestContainsTokenSequenceUsesWholeWords(t *testing.T) {
	if containsTokenSequence("love of silom", "silo") {
		t.Fatal("Silo must not match inside Silom")
	}
	if !containsTokenSequence("the silo story", "silo") {
		t.Fatal("Silo should match as a complete title token")
	}
}

func TestChooseTrailersDeprioritizesLikelyRestrictedCuts(t *testing.T) {
	values := []videoAsset{
		{Key: "red", Site: "YouTube", Type: "Trailer", Official: true, Language: "en", Size: 2160, Name: "Official Red Band Trailer"},
		{Key: "normal", Site: "YouTube", Type: "Trailer", Official: true, Language: "en", Size: 1080, Name: "Official Trailer"},
	}

	trailers := chooseTrailers(values, "en-US", 8)
	if len(trailers) != 2 {
		t.Fatalf("unexpected trailer count: %d", len(trailers))
	}
	if trailers[0].Key != "normal" || trailers[1].Key != "red" {
		t.Fatalf("restricted-looking trailer was not deprioritized: %+v", trailers)
	}
}
