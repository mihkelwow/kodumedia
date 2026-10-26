package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

func (a *API) getMusicLanding(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Music == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	result, err := a.dependencies.Music.Landing(r.Context(), strings.TrimSpace(r.URL.Query().Get("libraryId")))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) getMusicArtist(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Music == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	result, err := a.dependencies.Music.Artist(r.Context(), strings.TrimSpace(r.URL.Query().Get("libraryId")), r.PathValue("artist"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) getMusicAlbum(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Music == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	result, err := a.dependencies.Music.Album(r.Context(), strings.TrimSpace(r.URL.Query().Get("libraryId")), r.PathValue("albumId"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) streamLosslessAudio(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Audio == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("Cache-Control", "no-store")
	if err := a.dependencies.Audio.StreamLossless(r.Context(), r.PathValue("id"), w); err != nil {
		// Once streaming has started a JSON error cannot safely replace the WAV
		// response. The connection simply terminates and the player reports it.
		return
	}
}

func (a *API) serveMusicArtwork(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.MusicArtwork == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	result, err := a.dependencies.MusicArtwork.Resolve(r.Context(), r.PathValue("id"))
	if errors.Is(err, domain.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		writeError(w, err)
		return
	}
	file, err := os.Open(result.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		writeError(w, err)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", result.ContentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if result.ContentType == "image/svg+xml" {
		// Generated cards are safe, self-contained SVGs. Revalidate them often
		// so a later enrichment job can visibly upgrade the same URL to verified
		// artwork without waiting a full day for the browser cache to expire.
		w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'; style-src 'unsafe-inline'")
		w.Header().Set("Cache-Control", "public, max-age=300, must-revalidate")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=86400")
	}
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}

func (a *API) getMusicTracks(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Music == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	values := r.URL.Query()
	result, err := a.dependencies.Music.Tracks(r.Context(), domain.MusicTrackQuery{
		LibraryID: strings.TrimSpace(values.Get("libraryId")),
		Search:    strings.TrimSpace(values.Get("search")),
		Artist:    strings.TrimSpace(values.Get("artist")),
		Album:     strings.TrimSpace(values.Get("album")),
		Genre:     strings.TrimSpace(values.Get("genre")),
		Style:     strings.TrimSpace(values.Get("style")),
		Year:      integerQuery(values.Get("year")),
		Codec:     strings.TrimSpace(values.Get("codec")),
		Quality:   strings.TrimSpace(values.Get("quality")),
		Sort:      strings.TrimSpace(values.Get("sort")),
		Order:     domain.SortOrder(values.Get("order")),
		Limit:     integerQuery(values.Get("limit")),
		Offset:    integerQuery(values.Get("offset")),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) enrichMusic(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.MusicEnrichment == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	var input struct {
		LibraryID string `json:"libraryId"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&input); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, domain.ErrInvalidInput)
		return
	}
	job, err := a.dependencies.MusicEnrichment.Start(r.Context(), strings.TrimSpace(input.LibraryID))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}
