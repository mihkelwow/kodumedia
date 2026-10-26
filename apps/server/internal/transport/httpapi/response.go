package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

type errorResponse struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if value == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("write JSON response", "error", err)
	}
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "internal_error"
	message := "Serveri sisemine viga."

	switch {
	case errors.Is(err, domain.ErrNotFound):
		status = http.StatusNotFound
		code = "not_found"
		message = "Soovitud ressurssi ei leitud."
	case errors.Is(err, domain.ErrForbidden):
		status = http.StatusForbidden
		code = "forbidden"
		message = err.Error()
	case errors.Is(err, domain.ErrInvalidInput):
		status = http.StatusBadRequest
		code = "invalid_input"
		message = err.Error()
	case errors.Is(err, domain.ErrConflict):
		status = http.StatusConflict
		code = "conflict"
		message = err.Error()
	case errors.Is(err, domain.ErrUnsupported):
		status = http.StatusUnprocessableEntity
		code = "unsupported"
		message = err.Error()
	case errors.Is(err, domain.ErrUnavailable):
		status = http.StatusServiceUnavailable
		code = "unavailable"
		message = err.Error()
	default:
		slog.Error("HTTP handler failed", "error", err)
	}

	writeJSON(w, status, errorResponse{Error: apiError{Code: code, Message: message}})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, domain.ErrInvalidInput)
		return false
	}
	return true
}
