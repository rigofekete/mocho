// Package server wires the wiki into an HTTP API and serves the embedded SPA.
// It is the primary test seam: black-box tests exercise the running server
// against a real temp-dir wiki.
package server

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	"github.com/rigofekete/mocho/internal/wiki"
	"github.com/rigofekete/mocho/web"
)

// App holds the dependencies the HTTP handlers read from.
type App struct {
	Wiki wiki.Wiki
}

// New returns an App bound to a wiki.
func New(w wiki.Wiki) *App {
	return &App{Wiki: w}
}

// Handler builds the mux for the app. API routes live under /api; everything
// else falls through to the embedded SPA.
func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", a.handleHealth)
	mux.HandleFunc("GET /api/pages", a.handlePages)
	mux.HandleFunc("GET /api/pages/{name...}", a.handlePage)
	mux.Handle("/", spaHandler(web.FS()))
	return mux
}

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) handlePages(w http.ResponseWriter, r *http.Request) {
	pages, err := a.Wiki.Pages()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if pages == nil {
		pages = []wiki.PageRef{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"pages": pages})
}

func (a *App) handlePage(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	page, err := a.Wiki.ReadPage(name)
	if err != nil {
		if isNotExist(err) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "page not found"})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// spaHandler serves embedded SPA assets and falls back to index.html for
// client-side routing on unknown paths.
func spaHandler(dist fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(dist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := strings.TrimPrefix(r.URL.Path, "/")
		if clean == "" {
			clean = "index.html"
		}
		if _, err := fs.Stat(dist, clean); err != nil {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}