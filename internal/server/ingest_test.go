package server_test

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rigofekete/mocho/internal/agent"
	"github.com/rigofekete/mocho/internal/ingest"
	"github.com/rigofekete/mocho/internal/server"
	"github.com/rigofekete/mocho/internal/wiki"
)

// ingestHarness wraps a running server bound to a wiki + injectable agent fake.
type ingestHarness struct {
	*httptest.Server
	Dir  string
	Fake *agent.Fake
}

// newIngestWiki scaffolds a fresh wiki and binds a server with a defaulted fake.
func newIngestWiki(t *testing.T) ingestHarness {
	t.Helper()
	dir := t.TempDir()
	w := wiki.Wiki{Root: dir}
	if err := w.Scaffold(); err != nil {
		t.Fatalf("scaffold: %v", err)
	}
	fake := &agent.Fake{Output: "reading raw source\nsynthesizing concepts\nwriting concepts/goroutines.md\n"}
	h := newIngestHarness(t, dir, fake)
	return h
}

// newIngestHarness builds a server for an existing wiki dir + a caller-supplied fake.
func newIngestHarness(t *testing.T, dir string, fake *agent.Fake) ingestHarness {
	t.Helper()
	w := wiki.Wiki{Root: dir}
	svc := ingest.New(dir, fake, "opencode/wiki-test")
	app := server.New(w).WithIngest(svc)
	srv := httptest.NewServer(app.Handler())
	t.Cleanup(srv.Close)
	return ingestHarness{Server: srv, Dir: dir, Fake: fake}
}

// postIngest issues POST /api/ingest and returns status + full body.
func (h ingestHarness) postIngest(t *testing.T, path string) (status int, body string) {
	t.Helper()
	b, _ := json.Marshal(map[string]string{"path": path})
	res, err := h.Client().Post(h.URL+"/api/ingest", "application/json", strings.NewReader(string(b)))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(raw)
}

func artifactNameFromRaw(t *testing.T, dir string) string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(dir, "raw"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one raw artifact dir, got %d", len(entries))
	}
	return entries[0].Name()
}

func TestIngestStreamsAgentOutputAndStoresRawArtifact(t *testing.T) {
	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "lesson.md")
	if err := os.WriteFile(src, []byte("# Go concurrency\n\ngoroutines content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := newIngestWiki(t)

	status, body := h.postIngest(t, src)
	if status != 200 {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if !strings.Contains(body, "data: reading raw source") {
		t.Errorf("missing streamed output line: %s", body)
	}
	if !strings.Contains(body, "data: writing concepts/goroutines.md") {
		t.Errorf("missing synthesized output line: %s", body)
	}
	if !strings.Contains(body, "event: done") || !strings.Contains(body, `"ok":true`) {
		t.Errorf("missing done event: %s", body)
	}
	if !strings.Contains(body, "event: artifact") {
		t.Errorf("missing artifact event: %s", body)
	}

	last := h.Fake.Last()
	if last.Mode != agent.ModeWiki {
		t.Errorf("agent mode = %q, want wiki", last.Mode)
	}
	if last.Model != "opencode/wiki-test" {
		t.Errorf("agent model = %q", last.Model)
	}
	if last.WorkDir != h.Dir {
		t.Errorf("agent workdir = %q, want %q", last.WorkDir, h.Dir)
	}
	if !strings.HasPrefix(last.Prompt, "Ingest the new raw source at raw/") {
		t.Errorf("prompt did not reference the raw artifact: %q", last.Prompt)
	}

	name := artifactNameFromRaw(t, h.Dir)
	copied, err := os.ReadFile(filepath.Join(h.Dir, "raw", name, "lesson.md"))
	if err != nil {
		t.Fatalf("missing copied source: %v", err)
	}
	if !strings.Contains(string(copied), "goroutines content") {
		t.Errorf("verbatim content not copied: %q", copied)
	}
	if _, err := os.Stat(filepath.Join(h.Dir, "raw", name, "meta.json")); err != nil {
		t.Errorf("missing meta.json: %v", err)
	}
}

// TestIngestAgentExitFailureSurfacesErrorNotDone verifies a non-zero agent exit
// is reported as an error event (not a success "done"), with the committed raw
// artifact already announced. This is the spec's "ingest failure is visible".
func TestIngestAgentExitFailureSurfacesErrorNotDone(t *testing.T) {
	dir := t.TempDir()
	w := wiki.Wiki{Root: dir}
	if err := w.Scaffold(); err != nil {
		t.Fatal(err)
	}
	fake := &agent.Fake{Output: "starting synthesis\n", EndErr: runErr("agent exited 1")}
	h := newIngestHarness(t, dir, fake)

	src := filepath.Join(t.TempDir(), "s.md")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	status, body := h.postIngest(t, src)
	if status != 200 {
		t.Fatalf("expected streamed 200, got %d: %s", status, body)
	}
	if strings.Contains(body, "event: done") {
		t.Fatalf("agent failure must not emit done: %s", body)
	}
	if !strings.Contains(body, "event: error") {
		t.Fatalf("agent failure must emit error event: %s", body)
	}
	if !strings.Contains(body, "agent exited 1") {
		t.Fatalf("exit error text not surfaced: %s", body)
	}
	// Partial streamed output was visible before the failure.
	if !strings.Contains(body, "data: starting synthesis") {
		t.Errorf("missing partial stream output: %s", body)
	}
	// Raw artifact still committed and announced.
	if !strings.Contains(body, "event: artifact") {
		t.Errorf("missing artifact event before error: %s", body)
	}
	artifactNameFromRaw(t, dir)
}

// TestIngestAgentStartFailureStillCommitsRawArtifact verifies an agent that
// cannot start still commits the raw artifact and reports the error over SSE.
func TestIngestAgentStartFailureStillCommitsRawArtifact(t *testing.T) {
	dir := t.TempDir()
	w := wiki.Wiki{Root: dir}
	if err := w.Scaffold(); err != nil {
		t.Fatal(err)
	}
	fake := &agent.Fake{Err: runErr("opencode not installed")}
	h := newIngestHarness(t, dir, fake)

	src := filepath.Join(t.TempDir(), "s.md")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	status, body := h.postIngest(t, src)
	if status != 200 {
		t.Fatalf("expected streamed 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, "event: error") || !strings.Contains(body, "opencode not installed") {
		t.Fatalf("missing error event: %s", body)
	}
	// The committed raw artifact is named in the error payload so the client
	// knows the source was ingested even though synthesis did not run.
	name := artifactNameFromRaw(t, dir)
	if !strings.Contains(body, `"artifact":`) || !strings.Contains(body, name) {
		t.Errorf("error event should name committed artifact %q: %s", name, body)
	}
}

func TestIngestMissingSourceIsErrorEvent(t *testing.T) {
	h := newIngestWiki(t)
	status, body := h.postIngest(t, filepath.Join(t.TempDir(), "nope.md"))
	if status != 200 {
		t.Fatalf("valid requests stream as 200 SSE; got %d: %s", status, body)
	}
	if !strings.Contains(body, "event: error") {
		t.Fatalf("missing error event: %s", body)
	}
	if strings.Contains(body, "event: artifact") {
		t.Errorf("raw failure must not announce an artifact: %s", body)
	}
}

func TestIngestEmptyPathIsRejected(t *testing.T) {
	dir := t.TempDir()
	w := wiki.Wiki{Root: dir}
	if err := w.Scaffold(); err != nil {
		t.Fatal(err)
	}
	fake := &agent.Fake{}
	h := newIngestHarness(t, dir, fake)
	status, body := h.postIngest(t, "")
	if status != 400 {
		t.Fatalf("expected 400 for empty path, got %d: %s", status, body)
	}
}

func TestIngestWithoutServiceIs503(t *testing.T) {
	dir := t.TempDir()
	w := wiki.Wiki{Root: dir}
	if err := w.Scaffold(); err != nil {
		t.Fatal(err)
	}
	app := server.New(w) // no WithIngest
	srv := httptest.NewServer(app.Handler())
	defer srv.Close()
	h := ingestHarness{Server: srv}
	status, _ := h.postIngest(t, "/tmp/whatever")
	if status != 503 {
		t.Fatalf("expected 503 when ingest unset, got %d", status)
	}
}

type runErr string

func (e runErr) Error() string { return string(e) }
