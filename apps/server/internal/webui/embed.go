package webui

import (
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

//go:embed dist/*
var embedded embed.FS

func Handler() http.Handler {
	content, err := fs.Sub(embedded, "dist")
	if err != nil {
		panic(err)
	}
	index, err := fs.ReadFile(content, "index.html")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(content))

	serveIndex := func(w http.ResponseWriter) {
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(index)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" || path == "index.html" {
			serveIndex(w)
			return
		}
		info, statErr := fs.Stat(content, path)
		if statErr != nil || info.IsDir() {
			serveIndex(w)
			return
		}

		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		if contentType := mime.TypeByExtension(filepath.Ext(path)); contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		clone := r.Clone(r.Context())
		clone.URL.Path = "/" + path
		fileServer.ServeHTTP(w, clone)
	})
}
