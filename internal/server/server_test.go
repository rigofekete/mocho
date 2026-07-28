package server_test

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rigofekete/mocho/internal/server"
	"github.com/rigofekete/mocho/internal/wiki"
)

// newWiki builds a fresh empty wiki in a temp dir, scaffolds it, and returns a
// running server bound to it. Tests exercise the HTTP API against the real
// filesystem, never a fake.
func newWiki(t *testing.T) (dir string, ts *httptest.Server) {
	t.Helper()
	dir = t.TempDir()
	w := wiki.Wiki{Root: dir}
	if err := w.Scaffold(); err != nil {
		t.Fatalf("scaffold: %v", err)
	}
	app := server.New(w)
	ts = httptest.NewServer(app.Handler())
	t.Cleanup(ts.Close)
	return dir, ts
}

func getJSON(t *testing.T, ts *httptest.Server, path string, target any) (status int, body string) {
	t.Helper()
	res, err := ts.Client().Get(ts.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if target != nil && res.StatusCode < 300 {
		if err := json.Unmarshal(raw, target); err != nil {
			t.Fatalf("decode %s: %v (body=%q)", path, err, raw)
		}
	}
	return res.StatusCode, string(raw)
}

func TestScaffoldCreatesWikiStructure(t *testing.T) {
	dir, _ := newWiki(t)
	for _, rel := range []string{"AGENTS.md", "index.md", "log.md"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("expected %s to exist: %v", rel, err)
		}
	}
	for _, sub := range []string{"raw", "concepts", "courses"} {
		st, err := os.Stat(filepath.Join(dir, sub))
		if err != nil || !st.IsDir() {
			t.Errorf("expected dir %s to exist: %v", sub, err)
		}
	}
}

func TestScaffoldIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	w := wiki.Wiki{Root: dir}
	if err := w.Scaffold(); err != nil {
		t.Fatalf("first scaffold: %v", err)
	}
	original, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err := w.Scaffold(); err != nil {
		t.Fatalf("second scaffold: %v", err)
	}
	again, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if string(original) != string(again) {
		t.Error("scaffold overwrote existing wiki files")
	}
}

func TestHealth(t *testing.T) {
	_, ts := newWiki(t)
	var got map[string]bool
	status, _ := getJSON(t, ts, "/api/health", &got)
	if status != 200 || !got["ok"] {
		t.Fatalf("health = status %d, body %v", status, got)
	}
}

func TestEmptyWikiHasNoPages(t *testing.T) {
	_, ts := newWiki(t)
	status, raw := getJSON(t, ts, "/api/pages", nil)
	if status != 200 {
		t.Fatalf("status = %d, body = %s", status, raw)
	}
	if !strings.Contains(raw, `"pages":[]`) && !strings.Contains(raw, `"pages":null`) {
		t.Fatalf("expected empty pages list, got %s", raw)
	}
}

func TestPagesListedFromIndex(t *testing.T) {
	dir := t.TempDir()
	w := wiki.Wiki{Root: dir}
	if err := w.Scaffold(); err != nil {
		t.Fatal(err)
	}
	idx := "# Index\n\n- [Goroutines](concepts/goroutines.md) — lightweight concurrent units\n- [Channels](concepts/channels.md) -- typed conduits\n"
	if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte(idx), 0o644); err != nil {
		t.Fatal(err)
	}
	app := server.New(w)
	ts := httptest.NewServer(app.Handler())
	defer ts.Close()

	var got struct {
		Pages []struct {
			Name    string `json:"name"`
			Title   string `json:"title"`
			Summary string `json:"summary"`
		} `json:"pages"`
	}
	if status, body := getJSON(t, ts, "/api/pages", &got); status != 200 {
		t.Fatalf("status=%d body=%s", status, body)
	}
	if len(got.Pages) != 2 {
		t.Fatalf("expected 2 pages, got %d (%+v)", len(got.Pages), got.Pages)
	}
	if got.Pages[0].Name != "concepts/goroutines.md" || got.Pages[0].Title != "Goroutines" {
		t.Errorf("page[0] = %+v", got.Pages[0])
	}
	if got.Pages[1].Summary != "typed conduits" {
		t.Errorf("page[1].Summary = %q", got.Pages[1].Summary)
	}
}

func TestReadPageReturnsMarkdown(t *testing.T) {
	dir := t.TempDir()
	w := wiki.Wiki{Root: dir}
	if err := w.Scaffold(); err != nil {
		t.Fatal(err)
	}
	page := "# Goroutines\n\nGoroutines are lightweight threads managed by the Go runtime.\n\nSee [channels](concepts/channels.md).\n"
	if err := os.WriteFile(filepath.Join(dir, "concepts", "goroutines.md"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	app := server.New(w)
	ts := httptest.NewServer(app.Handler())
	defer ts.Close()

	var got struct {
		Name     string `json:"name"`
		Title    string `json:"title"`
		Markdown string `json:"markdown"`
	}
	if status, body := getJSON(t, ts, "/api/pages/concepts/goroutines.md", &got); status != 200 {
		t.Fatalf("status=%d body=%s", status, body)
	}
	if got.Title != "Goroutines" {
		t.Errorf("title = %q, want Goroutines", got.Title)
	}
	if !strings.Contains(got.Markdown, "lightweight threads") {
		t.Errorf("missing body content: %q", got.Markdown)
	}
}

func TestReadMissingPageIs404(t *testing.T) {
	_, ts := newWiki(t)
	status, _ := getJSON(t, ts, "/api/pages/concepts/missing.md", nil)
	if status != 404 {
		t.Fatalf("expected 404, got %d", status)
	}
}

func TestReadPageWithoutMdExtensionIsRejected(t *testing.T) {
	_, ts := newWiki(t)
	status, _ := getJSON(t, ts, "/api/pages/AGENTS", nil)
	if status == 200 {
		t.Fatalf("expected non-200 for non-.md page, got %d", status)
	}
}

func TestSPAServesIndex(t *testing.T) {
	_, ts := newWiki(t)
	res, err := ts.Client().Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("GET / status = %d", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), "mocho") && !strings.Contains(string(body), "<div id=\"root\">") {
		t.Errorf("SPA index did not contain root container or mocho: %s", body)
	}
}

// TestReadPageRejectsTraversal verifies at the wiki layer that a page name
// containing parent-directory traversal cannot escape the wiki root. (The HTTP
// server path-cleans such requests before dispatch, so the security guarantee
// is enforced at resolve.)
func TestReadPageRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	w := wiki.Wiki{Root: dir}
	if err := os.WriteFile(filepath.Join(dir, "secret.md"), []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := w.ReadPage("../secret.md"); err == nil {
		t.Fatal("expected traversal to be rejected")
	}
}