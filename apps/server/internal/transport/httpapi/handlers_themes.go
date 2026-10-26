package httpapi

import (
	"errors"
	"net/http"
	"os"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

func (a *API) getThemeSettings(w http.ResponseWriter, _ *http.Request) {
	if a.dependencies.Themes == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"seriesPlayback": "off",
			"volume":         0.15,
			"fadeInMs":       2000,
			"fadeOutMs":      2000,
		})
		return
	}
	writeJSON(w, http.StatusOK, a.dependencies.Themes.PlaybackConfiguration())
}

func (a *API) updateThemeSettings(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Settings == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	var input domain.ThemeMusicSettings
	if !decodeJSON(w, r, &input) {
		return
	}
	if _, err := a.dependencies.Settings.UpdateThemeMusic(r.Context(), input); err != nil {
		writeError(w, err)
		return
	}
	if a.dependencies.Themes == nil {
		writeJSON(w, http.StatusOK, domain.ThemePlaybackConfiguration{
			SeriesPlayback: input.SeriesPlayback,
			Volume:         domain.SeriesThemeVolume,
			FadeInMS:       domain.SeriesThemeFadeInMS,
			FadeOutMS:      domain.SeriesThemeFadeOutMS,
		})
		return
	}
	writeJSON(w, http.StatusOK, a.dependencies.Themes.PlaybackConfiguration())
}

func (a *API) serveSeriesTheme(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Themes == nil {
		writeError(w, domain.ErrNotFound)
		return
	}
	asset, err := a.dependencies.Themes.ResolveSeriesTheme(r.Context(), r.PathValue("id"))
	if errors.Is(err, domain.ErrPending) {
		w.Header().Set("Retry-After", "2")
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "preparing"})
		return
	}
	if err != nil {
		if errors.Is(err, domain.ErrUnavailable) {
			w.Header().Set("Retry-After", "30")
		}
		writeError(w, err)
		return
	}
	file, err := os.Open(asset.Path)
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
	w.Header().Set("Content-Type", "audio/mp4")
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, asset.Name, info.ModTime(), file)
}
