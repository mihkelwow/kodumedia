package httpapi

import (
	"fmt"
	"net/http"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

func (a *API) getStorage(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Storage == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	report, err := a.dependencies.Storage.Report(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, report)
}

func (a *API) manageStorage(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Storage == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	if !isLoopbackRequest(r) || !isLoopbackHost(r.Host) || r.Header.Get("X-KoduMedia-Storage") != "manage" {
		writeError(w, fmt.Errorf("%w: kettahooldus on lubatud ainult serveriarvuti localhost-brauserist", domain.ErrForbidden))
		return
	}
	var input domain.StorageCleanupRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := a.dependencies.Storage.Clear(r.Context(), input.Action)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, result)
}
