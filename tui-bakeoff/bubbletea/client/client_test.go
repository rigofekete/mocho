package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rigofekete/mocho-tui-bakeoff/bubbletea/client"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func newAPIServer(t *testing.T, pages map[string][]byte, listBody []byte, listErr string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/pages", func(w http.ResponseWriter, r *http.Request) {
		if listErr != "" {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": listErr})
			return
		}
		if listBody != nil {
			_, _ = w.Write(listBody)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"pages": []any{}})
	})
	mux.HandleFunc("GET /api/pages/{name...}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		body, ok := pages[name]
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "page not found"})
			return
		}
		_, _ = w.Write(body)
	})
	s := httptest.NewServer(mux)
	t.Cleanup(s.Close)
	return s
}

func TestListPagesReturnsPages(t *testing.T) {
	list, _ := json.Marshal(map[string]any{
		"pages": []map[string]string{
			{"name": "concepts/goroutines.md", "title": "Goroutines", "summary": "lightweight threads"},
			{"name": "courses/boot.md", "title": "Boot", "summary": "course hub"},
		},
	})
	s := newAPIServer(t, nil, list, "")

	c := client.New(s.URL)
	pages, err := c.ListPages(context.Background())
	if err != nil {
		t.Fatalf("ListPages: %v", err)
	}
	if len(pages) != 2 || pages[0].Name != "concepts/goroutines.md" || pages[0].Title != "Goroutines" || pages[0].Summary != "lightweight threads" {
		t.Fatalf("unexpected pages: %+v", pages)
	}
}

func TestListPagesEmptyReturnsEmptySlice(t *testing.T) {
	s := newAPIServer(t, nil, nil, "")
	c := client.New(s.URL)
	pages, err := c.ListPages(context.Background())
	if err != nil {
		t.Fatalf("ListPages: %v", err)
	}
	if len(pages) != 0 {
		t.Fatalf("expected empty slice, got %v", pages)
	}
}

func TestListPagesServerError(t *testing.T) {
	s := newAPIServer(t, nil, nil, "boom")
	c := client.New(s.URL)
	if _, err := c.ListPages(context.Background()); err == nil {
		t.Fatal("expected error on 500")
	}
}

func TestReadPageResolvesThroughWildcardMux(t *testing.T) {
	pageJSON, _ := json.Marshal(map[string]string{
		"name":     "concepts/goroutines.md",
		"title":    "Goroutines",
		"markdown": "# Goroutines\n\nbody",
	})
	s := newAPIServer(t, map[string][]byte{"concepts/goroutines.md": pageJSON}, nil, "")

	c := client.New(s.URL)
	p, err := c.ReadPage(context.Background(), "concepts/goroutines.md")
	if err != nil {
		t.Fatalf("ReadPage: %v", err)
	}
	if p.Title != "Goroutines" || p.Markdown != "# Goroutines\n\nbody" {
		t.Fatalf("unexpected page: %+v", p)
	}
}

func TestReadPageNotFoundReturnsErrorMessage(t *testing.T) {
	s := newAPIServer(t, map[string][]byte{}, nil, "")
	c := client.New(s.URL)
	_, err := c.ReadPage(context.Background(), "missing.md")
	if err == nil || !strings.Contains(err.Error(), "page not found") {
		t.Fatalf("expected 'page not found' error, got %v", err)
	}
}