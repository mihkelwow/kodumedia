package httpapi

import (
	"net/http"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

func (a *API) getJob(w http.ResponseWriter, r *http.Request) {
	job, found := a.dependencies.Jobs.Get(r.PathValue("id"))
	if !found {
		writeError(w, domain.ErrNotFound)
		return
	}
	writeJSON(w, http.StatusOK, job)
}
