package httpapi

import (
	"net/http"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

func (a *API) listNotifications(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Notifications == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	page, err := a.dependencies.Notifications.List(
		r.Context(),
		integerQuery(r.URL.Query().Get("limit")),
		integerQuery(r.URL.Query().Get("offset")),
	)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (a *API) markNotificationsRead(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Notifications == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	if err := a.dependencies.Notifications.MarkAllRead(r.Context()); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
