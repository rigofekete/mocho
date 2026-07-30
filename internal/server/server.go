// Package server wires the wiki into an HTTP API and serves the embedded SPA.
// It is the primary test seam: black-box tests exercise the running server
// against a real temp-dir wiki.
package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/rigofekete/mocho/internal/ingest"
	"github.com/rigofekete/mocho/internal/wiki"
	"github.com/rigofekete/mocho/web"
)

// App holds the dependencies the HTTP handlers read from.
type App struct {
	Wiki   wiki.Wiki
	Ingest *ingest.Service
}

// New returns an App bound to a wiki. The ingest service is nil; attach one
// with WithIngest to enable POST /api/ingest.
func New(w wiki.Wiki) *App {
	return &App{Wiki: w}
}

// WithIngest attaches the ingest pipeline and returns the app for chaining.
func (a *App) WithIngest(s *ingest.Service) *App {
	a.Ingest = s
	return a
}

// Handler builds the mux for the app. API routes live under /api; everything
// else falls through to the embedded SPA.
func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", a.handleHealth)
	mux.HandleFunc("GET /api/pages", a.handlePages)
	mux.HandleFunc("GET /api/pages/{name...}", a.handlePage)
	mux.HandleFunc("POST /api/ingest", a.handleIngest)
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

// handleIngest streams an ingest run to the client as Server-Sent Events.
// The handler is the seam where the agent backend (a fake in tests) is
// substituted, so black-box tests exercise the live pipeline without spawning
// a real opencode process.
func (a *App) handleIngest(w http.ResponseWriter, r *http.Request) {
	if a.Ingest == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "ingest not configured"})
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "path required"})
		return
	}

	// Once the request is validated, the response is an SSE stream for the
	// rest of the run. Both the raw-layer copy and the agent run are reported
	// as events, so ingest success/failure is always visible to the client:
	// an agent failure still surfaces the committed raw artifact before the
	// error event.
	flusher, _ := w.(http.Flusher)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	if flusher != nil {
		flusher.Flush()
	}

	res, err := a.Ingest.Ingest(r.Context(), req.Path)
	if err != nil {
		payload := map[string]string{"error": err.Error()}
		if res.Artifact.Name != "" {
			payload["artifact"] = res.Artifact.Name
		}
		sseEvent(w, flusher, "error", mustJSON(payload))
		return
	}
	sseEvent(w, flusher, "artifact", mustJSON(res.Artifact))

	scanner := bufio.NewScanner(res.Stream)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		sseData(w, flusher, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		// Non-zero agent exit surfaces here as a final-Read error.
		sseEvent(w, flusher, "error", mustJSON(map[string]string{"error": err.Error()}))
		_ = res.Stream.Close()
		return
	}
	_ = res.Stream.Close()
	sseEvent(w, flusher, "done", `{"ok":true}`)
}

// sseData writes a default-event data frame and flushes if possible.
func sseData(w io.Writer, flusher http.Flusher, line string) {
	fmt.Fprintf(w, "data: %s\n\n", line)
	if flusher != nil {
		flusher.Flush()
	}
}

// sseEvent writes a named event with the given json payload and flushes.
func sseEvent(w io.Writer, flusher http.Flusher, event, payload string) {
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, payload)
	if flusher != nil {
		flusher.Flush()
	}
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return `{"error":"marshal failed"}`
	}
	return string(b)
}
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
