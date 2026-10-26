package httpapi

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

func (a *API) listMedia(w http.ResponseWriter, r *http.Request) {
	queryValues := r.URL.Query()
	query := domain.MediaQuery{
		LibraryID: strings.TrimSpace(queryValues.Get("libraryId")),
		SeriesID:  strings.TrimSpace(queryValues.Get("seriesId")),
		Season:    integerQuery(queryValues.Get("season")),
		Search:    strings.TrimSpace(queryValues.Get("search")),
		YearFrom:  integerQuery(queryValues.Get("yearFrom")),
		YearTo:    integerQuery(queryValues.Get("yearTo")),
		Sort:      domain.MediaSort(queryValues.Get("sort")),
		Order:     domain.SortOrder(queryValues.Get("order")),
		Limit:     integerQuery(queryValues.Get("limit")),
		Offset:    integerQuery(queryValues.Get("offset")),
	}
	if types := strings.TrimSpace(queryValues.Get("type")); types != "" {
		for _, value := range strings.Split(types, ",") {
			query.Types = append(query.Types, domain.MediaType(strings.TrimSpace(value)))
		}
	}
	if genres := strings.TrimSpace(queryValues.Get("genre")); genres != "" {
		for _, value := range strings.Split(genres, ",") {
			if genre := strings.TrimSpace(value); genre != "" {
				query.Genres = append(query.Genres, genre)
			}
		}
	}
	page, err := a.dependencies.Media.List(r.Context(), query)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (a *API) getMedia(w http.ResponseWriter, r *http.Request) {
	item, err := a.dependencies.Media.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) getPostPlay(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.PostPlay == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	result, err := a.dependencies.PostPlay.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) streamMedia(w http.ResponseWriter, r *http.Request) {
	item, err := a.dependencies.Media.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if item.Type == domain.MediaTypeSeries || strings.TrimSpace(item.File.Path) == "" {
		writeError(w, fmt.Errorf("%w: this media item has no playable file", domain.ErrUnsupported))
		return
	}
	file, err := os.Open(item.File.Path)
	if errors.Is(err, os.ErrNotExist) {
		writeError(w, domain.ErrNotFound)
		return
	}
	if err != nil {
		writeError(w, err)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		writeError(w, err)
		return
	}
	contentType := mediaContentType(filepath.Ext(item.File.Name))
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("Accept-Ranges", "bytes")
	// Google Cast's receiver runs on a different origin. Direct-play media needs
	// the same read-only CORS allowance as generated HLS playlists and segments.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	http.ServeContent(w, r, item.File.Name, info.ModTime(), file)
}

func (a *API) refreshMetadata(w http.ResponseWriter, r *http.Request) {
	item, err := a.dependencies.Metadata.Refresh(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) getSeriesDetails(w http.ResponseWriter, r *http.Request) {
	details, err := a.dependencies.Media.GetSeriesDetails(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, details)
}

func integerQuery(value string) int {
	parsed, _ := strconv.Atoi(strings.TrimSpace(value))
	return parsed
}

func mediaContentType(extension string) string {
	switch strings.ToLower(strings.TrimSpace(extension)) {
	case ".flac":
		return "audio/flac"
	case ".wav":
		return "audio/wav"
	case ".aiff", ".aif", ".aifc":
		return "audio/aiff"
	case ".mp3", ".mp2", ".mpa":
		return "audio/mpeg"
	case ".m4a", ".m4b", ".aac", ".adts":
		return "audio/mp4"
	case ".ogg", ".oga", ".opus":
		return "audio/ogg"
	case ".wma":
		return "audio/x-ms-wma"
	}
	return mime.TypeByExtension(extension)
}
