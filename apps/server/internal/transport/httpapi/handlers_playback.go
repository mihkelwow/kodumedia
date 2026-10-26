package httpapi

import (
	"context"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

type playbackRequest struct {
	MediaID      string                     `json:"mediaId"`
	Capabilities domain.ClientCapabilities  `json:"capabilities"`
	Preferences  domain.PlaybackPreferences `json:"preferences"`
}

func (a *API) playbackOptions(w http.ResponseWriter, r *http.Request) {
	options, err := a.dependencies.Playback.Options(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, options)
}

func (a *API) planPlayback(w http.ResponseWriter, r *http.Request) {
	var request playbackRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if strings.TrimSpace(request.MediaID) == "" {
		writeError(w, domain.ErrInvalidInput)
		return
	}
	plan, err := a.dependencies.Playback.Plan(r.Context(), request.MediaID, request.Capabilities, request.Preferences)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (a *API) startPlayback(w http.ResponseWriter, r *http.Request) {
	var request playbackRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if strings.TrimSpace(request.MediaID) == "" {
		writeError(w, domain.ErrInvalidInput)
		return
	}
	session, err := a.dependencies.Playback.Start(r.Context(), request.MediaID, request.Capabilities, request.Preferences)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, session)
}

func (a *API) playbackStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if a.dependencies.Transcoder == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	stats, err := a.dependencies.Transcoder.Stats(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (a *API) stopPlayback(w http.ResponseWriter, r *http.Request) {
	if err := a.dependencies.Playback.Stop(r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) servePlaybackFile(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Transcoder == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	file, err := a.dependencies.Transcoder.Open(r.Context(), r.PathValue("id"), r.PathValue("file"))
	if err != nil {
		writeError(w, err)
		return
	}
	defer file.Close()

	statFile, ok := file.(interface{ Stat() (fs.FileInfo, error) })
	if !ok {
		writeError(w, domain.ErrUnsupported)
		return
	}
	info, err := statFile.Stat()
	if err != nil {
		writeError(w, err)
		return
	}
	seeker, ok := file.(io.ReadSeeker)
	if !ok {
		writeError(w, domain.ErrUnsupported)
		return
	}

	extension := strings.ToLower(filepath.Ext(info.Name()))
	contentType := mime.TypeByExtension(extension)
	switch extension {
	case ".m3u8":
		contentType = "application/vnd.apple.mpegurl"
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	case ".m4s":
		contentType = "video/iso.segment"
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	case ".ts":
		contentType = "video/mp2t"
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	case ".mp4":
		contentType = "video/mp4"
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	http.ServeContent(w, r, info.Name(), info.ModTime(), seeker)
}

func (a *API) serveSubtitle(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Subtitles == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	ctx, cancel := contextWithTimeout(r, 90*time.Second)
	defer cancel()
	path, err := a.dependencies.Subtitles.WebVTT(ctx, r.PathValue("id"), r.PathValue("subtitleId"))
	if err != nil {
		writeError(w, err)
		return
	}
	file, err := os.Open(path)
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
	w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}

func (a *API) searchSubtitles(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Subtitles == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	languages := splitQueryValues(r.URL.Query()["language"])
	if value := strings.TrimSpace(r.URL.Query().Get("languages")); value != "" {
		languages = append(languages, strings.Split(value, ",")...)
	}
	results, err := a.dependencies.Subtitles.Search(r.Context(), r.PathValue("id"), languages)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": results})
}

func (a *API) downloadSubtitle(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Subtitles == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	var request domain.SubtitleDownloadRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	track, err := a.dependencies.Subtitles.Download(r.Context(), r.PathValue("id"), request)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, track)
}

func splitQueryValues(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, strings.Split(value, ",")...)
	}
	return result
}

func contextWithTimeout(r *http.Request, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), timeout)
}
