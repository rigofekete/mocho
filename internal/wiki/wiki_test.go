package wiki_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rigofekete/mocho/internal/wiki"
)

func scaffoldWiki(t *testing.T) (wiki.Wiki, string) {
	t.Helper()
	dir := t.TempDir()
	w := wiki.Wiki{Root: dir}
	if err := w.Scaffold(); err != nil {
		t.Fatalf("scaffold: %v", err)
	}
	return w, dir
}

func TestPagesParsesMixedSeparators(t *testing.T) {
	w, dir := scaffoldWiki(t)
	idx := "# Index\n\n" +
		"- [Alpha](concepts/alpha.md) — em-summary\n" +
		"* [Beta](concepts/beta.md) -- dash-summary\n" +
		"- [Gamma](concepts/gamma.md)\n" +
		"- plain prose line, not a page\n"
	if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte(idx), 0o644); err != nil {
		t.Fatal(err)
	}
	pages, err := w.Pages()
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 3 {
		t.Fatalf("expected 3 pages, got %d: %+v", len(pages), pages)
	}
	if pages[0].Summary != "em-summary" || pages[1].Summary != "dash-summary" {
		t.Errorf("summaries wrong: %+v", pages)
	}
	if pages[2].Summary != "" {
		t.Errorf("gamma should have empty summary, got %q", pages[2].Summary)
	}
}

func TestPagesHandlesMissingIndex(t *testing.T) {
	dir := t.TempDir()
	w := wiki.Wiki{Root: dir}
	_, err := w.Pages()
	if err == nil {
		t.Fatal("expected error reading missing index.md")
	}
}

func TestReadPageTitleFallback(t *testing.T) {
	w, dir := scaffoldWiki(t)
	plain := "No heading here, just body.\n"
	if err := os.WriteFile(filepath.Join(dir, "concepts", "lonely.md"), []byte(plain), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := w.ReadPage("concepts/lonely.md")
	if err != nil {
		t.Fatal(err)
	}
	if p.Title != "lonely" {
		t.Fatalf("title fallback = %q, want lonely", p.Title)
	}
}

func TestResolveRejectsNonMdAndTraversal(t *testing.T) {
	w, _ := scaffoldWiki(t)
	for _, name := range []string{"AGENTS", "concepts/alpha.txt"} {
		if _, err := w.ReadPage(name); err == nil {
			t.Errorf("expected rejection for %q", name)
		}
	}
	if _, err := w.ReadPage("../../etc/passwd"); err == nil {
		t.Error("expected traversal rejection")
	}
}

func TestScaffoldNoOpOnExisting(t *testing.T) {
	w, _ := scaffoldWiki(t)
	size, _ := os.ReadFile(filepath.Join(w.Root, "index.md"))
	// Mutate index so idempotency is detectable.
	if err := os.WriteFile(filepath.Join(w.Root, "index.md"), append([]byte("# mutated\n"), size...), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := w.Scaffold(); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(w.Root, "index.md"))
	if string(got) == string(size) {
		t.Error("scaffold overwrote a user-mutated existing wiki")
	}
}