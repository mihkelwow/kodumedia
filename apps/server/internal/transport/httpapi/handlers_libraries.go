package httpapi

import (
	"net/http"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/service"
)

type createLibraryRequest struct {
	Name string             `json:"name"`
	Kind domain.LibraryKind `json:"kind"`
}

type updateLibraryRequest struct {
	Name *string             `json:"name"`
	Kind *domain.LibraryKind `json:"kind"`
}

type addSourceRequest struct {
	Path string `json:"path"`
}

func (a *API) listLibraries(w http.ResponseWriter, r *http.Request) {
	libraries, err := a.dependencies.Libraries.List(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": libraries})
}

func (a *API) getLibrary(w http.ResponseWriter, r *http.Request) {
	library, err := a.dependencies.Libraries.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, library)
}

func (a *API) createLibrary(w http.ResponseWriter, r *http.Request) {
	var request createLibraryRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	library, err := a.dependencies.Libraries.Create(r.Context(), service.CreateLibraryInput{Name: request.Name, Kind: request.Kind})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, library)
}

func (a *API) updateLibrary(w http.ResponseWriter, r *http.Request) {
	var request updateLibraryRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	library, err := a.dependencies.Libraries.Update(r.Context(), r.PathValue("id"), service.UpdateLibraryInput{Name: request.Name, Kind: request.Kind})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, library)
}

func (a *API) deleteLibrary(w http.ResponseWriter, r *http.Request) {
	if err := a.dependencies.Libraries.Delete(r.Context(), r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) addLibrarySource(w http.ResponseWriter, r *http.Request) {
	var request addSourceRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	library, err := a.dependencies.Libraries.AddSource(r.Context(), r.PathValue("id"), request.Path)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, library)
}

func (a *API) removeLibrarySource(w http.ResponseWriter, r *http.Request) {
	library, err := a.dependencies.Libraries.RemoveSource(r.Context(), r.PathValue("id"), r.PathValue("sourceId"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, library)
}

func (a *API) scanLibrary(w http.ResponseWriter, r *http.Request) {
	job, err := a.dependencies.Scanner.Start(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}
