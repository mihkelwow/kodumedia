package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

func (a *API) continueWatching(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Watch == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := a.dependencies.Watch.ContinueWatching(r.Context(), limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) updateWatchState(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Watch == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	var input domain.WatchUpdate
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeError(w, domain.ErrInvalidInput)
		return
	}
	state, err := a.dependencies.Watch.Update(r.Context(), r.PathValue("id"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}
