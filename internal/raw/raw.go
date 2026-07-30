// Package raw owns the immutable raw layer of the wiki: the verbatim storage
// of ingested sources plus sidecar provenance metadata. Writes here are
// performed by Go, never the agent — immutability is determinstic and
// testable. Re-ingest of the same source produces a new artifact rather than
// mutating the previous one.
package raw

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// SourceType labels how a raw artifact was acquired. v1 ships local sources;
// remote boot.dev / web adapters (#4) extend this vocabulary.
type SourceType string

const SourceLocal SourceType = "local"

// Artifact describes a single stored raw source and its provenance.
type Artifact struct {
	Name       string     `json:"name"`       // unique allocated slug under raw/
	SourceType SourceType `json:"sourceType"` // how it was acquired
	SourcePath string     `json:"sourcePath"` // absolute path/URL of the origin
	FetchedAt  time.Time  `json:"fetchedAt"`  // when it was copied in
}

// Store is the raw/ directory of a wiki. Root is the absolute path to raw/.
type Store struct {
	Root string
}

// metaFileName is the sidecar written next to the artifact content.
const metaFileName = "meta.json"

// Ingest copies a local file or directory into raw/ verbatim and writes a
// sidecar meta.json recording provenance. The returned Artifact.Name uniquely
// identifies the stored content (a fresh slug per ingest) so re-ingesting the
// same source never overwrites a previous copy. The original source is never
// mutated.
func (s Store) Ingest(srcPath string) (Artifact, error) {
	abs, err := filepath.Abs(filepath.Clean(srcPath))
	if err != nil {
		return Artifact{}, fmt.Errorf("resolve source path: %w", err)
	}
	st, err := os.Stat(abs)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Artifact{}, fmt.Errorf("source not found: %s", abs)
		}
		return Artifact{}, fmt.Errorf("stat source: %w", err)
	}
	if err := os.MkdirAll(s.Root, 0o755); err != nil {
		return Artifact{}, fmt.Errorf("create raw dir: %w", err)
	}

	name, err := allocateName(s.Root, st)
	if err != nil {
		return Artifact{}, err
	}
	artDir := filepath.Join(s.Root, name)
	if err := os.Mkdir(artDir, 0o755); err != nil {
		return Artifact{}, fmt.Errorf("allocate artifact dir: %w", err)
	}

	if st.IsDir() {
		if err := copyDir(abs, artDir); err != nil {
			return Artifact{}, fmt.Errorf("copy dir: %w", err)
		}
	} else {
		if err := copyFile(abs, filepath.Join(artDir, st.Name())); err != nil {
			return Artifact{}, fmt.Errorf("copy file: %w", err)
		}
	}

	now := time.Now().UTC()
	art := Artifact{
		Name:       name,
		SourceType: SourceLocal,
		SourcePath: abs,
		FetchedAt:  now,
	}
	if err := s.writeMeta(artDir, art); err != nil {
		return Artifact{}, fmt.Errorf("write %s: %w", metaFileName, err)
	}
	return art, nil
}

// writeMeta serializes the provenance sidecar.
func (s Store) writeMeta(artDir string, art Artifact) error {
	body, err := json.MarshalIndent(art, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(artDir, metaFileName), body, 0o644)
}

// ReadMeta reads the sidecar for an existing artifact name.
func (s Store) ReadMeta(name string) (Artifact, error) {
	data, err := os.ReadFile(filepath.Join(s.Root, name, metaFileName))
	if err != nil {
		return Artifact{}, err
	}
	var art Artifact
	if err := json.Unmarshal(data, &art); err != nil {
		return Artifact{}, fmt.Errorf("parse %s: %w", metaFileName, err)
	}
	return art, nil
}

// copyDir copies src's contents verbatim into dst (dst must already exist).
func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(p, target)
	})
}

// copyFile copies a single regular file preserving bytes exactly.
func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	st, err := os.Stat(src)
	if err == nil {
		_ = os.Chmod(dst, st.Mode().Perm())
	}
	return nil
}

// allocateName returns a fresh unique slug under root, composed of the source
// base name, a UTC timestamp, and a short random suffix. Existence is checked
// so collisions (astronomically unlikely) retry with new randomness.
func allocateName(root string, st os.FileInfo) (string, error) {
	base := slug(filepath.Base(nameHint(st)))
	ts := time.Now().UTC().Format("20060102-150405")
	for i := 0; i < 8; i++ {
		suf, err := randomHex(2) // 4 hex chars
		if err != nil {
			return "", err
		}
		name := fmt.Sprintf("%s-%s-%s", base, ts, suf)
		if _, err := os.Stat(filepath.Join(root, name)); errors.Is(err, fs.ErrNotExist) {
			return name, nil
		}
	}
	return "", errors.New("could not allocate unique raw artifact name")
}

// nameHint picks the slug seed: the dir name for directories, the file name
// (minus extension) for files.
func nameHint(st os.FileInfo) string {
	if st.IsDir() {
		return st.Name()
	}
	return strings.TrimSuffix(st.Name(), filepath.Ext(st.Name()))
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// slug sanitizes an arbitrary string into a lowercase, hyphen-separated slug.
func slug(in string) string {
	s := nonSlug.ReplaceAllString(strings.ToLower(in), "-")
	return strings.Trim(s, "-")
}

// randomHex returns n random bytes as a hex string (2n hex chars).
func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
