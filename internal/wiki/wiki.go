// Package wiki owns the on-disk wiki structure: scaffolding an empty wiki,
// enumerating pages from index.md, and reading a page's markdown source.
package wiki

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Wiki is a directory of markdown governed by an agent-neutral schema.
// The path is configured at startup; Scaffold materializes the structure.
type Wiki struct {
	Root string
}

// PageRef is a catalog entry parsed from index.md.
type PageRef struct {
	Name    string `json:"name"`    // path relative to wiki root, with .md
	Title   string `json:"title"`   // link text from index.md
	Summary string `json:"summary"` // one-line description following " — "
}

// Page is the full content of a single wiki page.
type Page struct {
	Name     string `json:"name"`
	Title    string `json:"title"`
	Markdown string `json:"markdown"`
}

// Exists reports whether the wiki already has its scaffold marker (AGENTS.md).
func (w Wiki) Exists() bool {
	_, err := os.Stat(filepath.Join(w.Root, "AGENTS.md"))
	return err == nil
}

// Scaffold creates an empty wiki at the configured path. It is a no-op if the
// wiki already exists. Required layout: AGENTS.md, raw/, concepts/, courses/,
// index.md, log.md.
func (w Wiki) Scaffold() error {
	if w.Root == "" {
		return errors.New("wiki path is empty")
	}
	if err := os.MkdirAll(w.Root, 0o755); err != nil {
		return fmt.Errorf("create wiki dir: %w", err)
	}
	if w.Exists() {
		return nil
	}
	for _, sub := range []string{"raw", "concepts", "courses"} {
		if err := os.MkdirAll(filepath.Join(w.Root, sub), 0o755); err != nil {
			return fmt.Errorf("create %s: %w", sub, err)
		}
	}
	writes := map[string]string{
		"AGENTS.md": agentsSchema,
		"index.md":  indexTemplate,
		"log.md":    logHeader,
	}
	for name, content := range writes {
		if err := os.WriteFile(filepath.Join(w.Root, name), []byte(content), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}
	return nil
}

// indexLineRe matches a markdown list entry linking to a .md page.
var indexLineRe = regexp.MustCompile(`^\s*[-*]\s+\[([^\]]*)\]\(([^)]+\.md)\)(?:\s+(?:—|-{1,2})\s+(.*))?$`)

// Pages enumerates the wiki catalog by parsing index.md. Each entry is a
// list item of the form: `- [Title](path/to.md) — summary`.
func (w Wiki) Pages() ([]PageRef, error) {
	data, err := os.ReadFile(filepath.Join(w.Root, "index.md"))
	if err != nil {
		return nil, fmt.Errorf("read index.md: %w", err)
	}
	var pages []PageRef
	for _, line := range strings.Split(string(data), "\n") {
		m := indexLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		ref := PageRef{Name: m[2], Title: m[1], Summary: strings.TrimSpace(m[3])}
		pages = append(pages, ref)
	}
	return pages, nil
}

// ReadPage returns a single page's content. name is a path relative to the
// wiki root with a .md extension. It returns os.ErrNotExist-style errors for
// missing pages and rejects paths that escape the wiki root.
func (w Wiki) ReadPage(name string) (Page, error) {
	abs, err := w.resolve(name)
	if err != nil {
		return Page{}, err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return Page{}, err
	}
	return Page{
		Name:     name,
		Title:    titleOf(name, string(data)),
		Markdown: string(data),
	}, nil
}

// resolve cleans name and confirms it stays inside the wiki root.
func (w Wiki) resolve(name string) (string, error) {
	if name == "" || strings.Contains(name, "..") {
		return "", errors.New("invalid page name")
	}
	if !strings.HasSuffix(name, ".md") {
		return "", errors.New("page name must end with .md")
	}
	abs := filepath.Clean(filepath.Join(w.Root, filepath.ToSlash(name)))
	rel, err := filepath.Rel(w.Root, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", errors.New("page escapes wiki root")
	}
	return abs, nil
}

// titleOf prefers the first H1 in the body, else derives from the file name.
func titleOf(name string, body string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	base := filepath.Base(name)
	return strings.TrimSuffix(base, ".md")
}