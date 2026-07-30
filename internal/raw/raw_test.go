package raw_test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/rigofekete/mocho/internal/raw"
)

// writeFile is a small helper creating a file with content and 0644 perms.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestIngestFileCopiesVerbatimWithMeta(t *testing.T) {
	srcDir := t.TempDir()
	content := "# Gator lesson\n\nSome verbatim source text.\n"
	src := filepath.Join(srcDir, "notes.md")
	writeFile(t, src, content)

	rawRoot := t.TempDir()
	store := raw.Store{Root: rawRoot}
	art, err := store.Ingest(src)
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}

	if art.SourceType != raw.SourceLocal {
		t.Fatalf("source type = %q, want local", art.SourceType)
	}
	if !strings.HasSuffix(art.SourcePath, "notes.md") {
		t.Fatalf("source path = %q", art.SourcePath)
	}
	if art.FetchedAt.IsZero() {
		t.Fatal("fetched-at is zero")
	}
	if !strings.HasPrefix(art.Name, "notes-") {
		t.Fatalf("artifact name %q should start with slug 'notes-'", art.Name)
	}

	// Content copied verbatim.
	got, err := os.ReadFile(filepath.Join(rawRoot, art.Name, "notes.md"))
	if err != nil {
		t.Fatalf("read copied content: %v", err)
	}
	if string(got) != content {
		t.Fatalf("content mismatch: got %q", got)
	}

	// Sidecar meta.json present and parseable.
	meta, err := store.ReadMeta(art.Name)
	if err != nil {
		t.Fatalf("read meta: %v", err)
	}
	if meta.Name != art.Name || meta.SourceType != raw.SourceLocal {
		t.Fatalf("meta mismatch: %+v", meta)
	}
}

func TestIngestDirectoryCopiesNestedVerbatim(t *testing.T) {
	src := t.TempDir()
	writeFile(t, filepath.Join(src, "README.md"), "# readme\n")
	writeFile(t, filepath.Join(src, "sub", "deep", "one.md"), "# one\n")

	rawRoot := t.TempDir()
	store := raw.Store{Root: rawRoot}
	art, err := store.Ingest(src)
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}

	if !strings.HasPrefix(art.Name, filepath.Base(src)+"-") {
		t.Fatalf("dir artifact name %q should start with dir base", art.Name)
	}
	// Both nested files preserved verbatim.
	for _, rel := range []string{"README.md", filepath.Join("sub", "deep", "one.md")} {
		data, err := os.ReadFile(filepath.Join(rawRoot, art.Name, rel))
		if err != nil {
			t.Fatalf("missing nested %s: %v", rel, err)
		}
		if len(data) == 0 {
			t.Fatalf("empty nested %s", rel)
		}
	}
	_, err = store.ReadMeta(art.Name)
	if err != nil {
		t.Fatalf("missing dir meta.json: %v", err)
	}
}

func TestReIngestCreatesNewArtifact(t *testing.T) {
	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "doc.md")
	writeFile(t, src, "first body\n")

	rawRoot := t.TempDir()
	store := raw.Store{Root: rawRoot}
	a1, err := store.Ingest(src)
	if err != nil {
		t.Fatal(err)
	}
	// Force the clock to advance so the timestamp suffix differs even on
	// coarse-resolution clocks.
	time.Sleep(1100 * time.Millisecond)
	content2 := "second body — the source changed but old artifact stays\n"
	writeFile(t, src, content2)
	a2, err := store.Ingest(src)
	if err != nil {
		t.Fatal(err)
	}
	if a1.Name == a2.Name {
		t.Fatalf("re-ingest reused artifact name %q (immutability violated)", a1.Name)
	}

	// First artifact unchanged.
	old, err := os.ReadFile(filepath.Join(rawRoot, a1.Name, "doc.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(old) != "first body\n" {
		t.Fatalf("old artifact mutated: %q", old)
	}
	// Second artifact holds the new body.
	got, err := os.ReadFile(filepath.Join(rawRoot, a2.Name, "doc.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != content2 {
		t.Fatalf("new artifact body mismatch: %q", got)
	}

	// Listing raw/ shows two artifact dirs (+ their files).
	entries, err := os.ReadDir(rawRoot)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	if len(names) != 2 {
		t.Fatalf("expected 2 artifact dirs, got %d: %v", len(names), names)
	}
}

func TestIngestMissingSourceErrors(t *testing.T) {
	rawRoot := t.TempDir()
	store := raw.Store{Root: rawRoot}
	if _, err := store.Ingest(filepath.Join(t.TempDir(), "nope.md")); err == nil {
		t.Fatal("expected error for missing source")
	}
}

func TestSlugSanitizesUnsafeBase(t *testing.T) {
	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "Weird Lesson 101!?.md")
	writeFile(t, src, "x")
	rawRoot := t.TempDir()
	store := raw.Store{Root: rawRoot}
	art, err := store.Ingest(src)
	if err != nil {
		t.Fatal(err)
	}
	// The slug prefix must be lowercase, hyphen-separated, no punctuation.
	prefixEnd := strings.IndexByte(art.Name, '-')
	for _, r := range art.Name[:prefixEnd] {
		if !(r == '-' || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
			t.Fatalf("name has unsafe char %q in slug: %q", r, art.Name)
		}
	}
}
