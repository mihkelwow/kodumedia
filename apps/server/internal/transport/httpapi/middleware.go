package httpapi

import (
	"log/slog"
	"net/http"
	"path"
	"runtime/debug"
	"strings"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(data)
}

func withMiddleware(next http.Handler, allowedOrigins []string) http.Handler {
	return recoveryMiddleware(loggingMiddleware(corsMiddleware(securityHeaders(next), allowedOrigins)))
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler, allowedOrigins []string) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	allowAny := false
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin == "*" {
			allowAny = true
		}
		if origin != "" {
			allowed[origin] = struct{}{}
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		_, isAllowed := allowed[origin]
		if origin != "" && (allowAny || isAllowed) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		duration := time.Since(started)
		attributes := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"duration", duration,
			"remote", r.RemoteAddr,
		}
		switch {
		case status >= http.StatusInternalServerError:
			slog.Error("http request failed", attributes...)
		case status >= http.StatusBadRequest:
			slog.Warn("http request rejected", attributes...)
		case duration >= 750*time.Millisecond:
			slog.Warn("slow http request", attributes...)
		case r.Method != http.MethodGet && r.Method != http.MethodHead && !noisyRequestPath(r.URL.Path):
			slog.Info("http request", attributes...)
		}
	})
}

func noisyRequestPath(value string) bool {
	extension := strings.ToLower(path.Ext(value))
	switch extension {
	case ".m3u8", ".m4s", ".ts", ".mp4", ".css", ".js", ".map", ".ico", ".png", ".jpg", ".jpeg", ".webp", ".svg", ".woff", ".woff2":
		return true
	}
	return strings.Contains(value, "/playback/sessions/")
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.Error("panic in HTTP handler", "panic", recovered, "stack", string(debug.Stack()))
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: apiError{Code: "internal_error", Message: "Serveri sisemine viga."}})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
