package embed

import (
	"net/http"
	"strings"
)

// ServeHTML returns a handler that serves one embedded HTML page.
func ServeHTML(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := ReadFile(name)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(b)
	}
}

// ServeAsset returns a handler for a single embedded asset with a fixed content type.
func ServeAsset(name, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := ReadFile(name)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		_, _ = w.Write(b)
	}
}

// ServePathPrefix serves files under embedPrefix at urlPrefix with the given content type.
func ServePathPrefix(embedPrefix, urlPrefix, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, urlPrefix)
		p = strings.TrimSpace(p)
		if p == "" || strings.Contains(p, "..") || strings.HasPrefix(p, "/") || strings.ContainsAny(p, "\\") {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		b, err := ReadFile(embedPrefix + p)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write(b)
	}
}

// ServeFontPrefix serves woff2 (and related) font files from embedui/fonts/.
func ServeFontPrefix(embedPrefix, urlPrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, urlPrefix)
		p = strings.TrimSpace(p)
		if p == "" || strings.Contains(p, "..") || strings.HasPrefix(p, "/") || strings.ContainsAny(p, "\\") {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		b, err := ReadFile(embedPrefix + p)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		ct := "application/octet-stream"
		switch {
		case strings.HasSuffix(strings.ToLower(p), ".woff2"):
			ct = "font/woff2"
		case strings.HasSuffix(strings.ToLower(p), ".woff"):
			ct = "font/woff"
		case strings.HasSuffix(strings.ToLower(p), ".ttf"):
			ct = "font/ttf"
		case strings.HasSuffix(strings.ToLower(p), ".js"):
			ct = "application/javascript; charset=utf-8"
		case strings.HasSuffix(strings.ToLower(p), ".txt"):
			ct = "text/plain; charset=utf-8"
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", ct)
		_, _ = w.Write(b)
	}
}
