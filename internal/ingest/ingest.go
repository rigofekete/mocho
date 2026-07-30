// Package ingest wires the ingestion pipeline: Go copies a local source into
// the immutable raw/ layer, then the agent CLI synthesizes the wiki from it
// (concept/course pages, index/log updates per AGENTS.md). The agent output
// is streamed to the caller so the UI can show ingest progress live.
package ingest

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/rigofekete/mocho/internal/agent"
	"github.com/rigofekete/mocho/internal/raw"
)

// Service is the ingest pipeline bound to a wiki, an agent backend, and the
// wiki-mode model resolved from config.
type Service struct {
	WikiRoot  string // absolute wiki root; agent runs here
	Raw       raw.Store
	Agent     agent.AgentBackend
	WikiModel string
}

// New builds a Service for a wiki root. The raw store lives at WikiRoot/raw.
func New(wikiRoot string, be agent.AgentBackend, wikiModel string) *Service {
	return &Service{
		WikiRoot:  wikiRoot,
		Raw:       raw.Store{Root: filepath.Join(wikiRoot, "raw")},
		Agent:     be,
		WikiModel: wikiModel,
	}
}

// Result is the outcome of the synchronous part of an ingest: the stored raw
// artifact plus a live stream of the agent's synthesis output. The caller
// drains and closes Stream to complete the run.
type Result struct {
	Artifact raw.Artifact
	Stream   io.ReadCloser
}

// Ingest performs the raw-layer copy (Go-owned) then kicks off the agent
// synthesis (wiki mode) and returns the streaming agent output. The raw
// artifact is committed before the agent runs; an agent failure leaves the
// raw artifact in place for re-ingest.
func (s *Service) Ingest(ctx context.Context, srcPath string) (Result, error) {
	art, err := s.Raw.Ingest(srcPath)
	if err != nil {
		return Result{}, err
	}
	stream, err := s.Agent.Run(ctx, ingestPrompt(art), s.WikiRoot, agent.ModeWiki, s.WikiModel)
	if err != nil {
		return Result{Artifact: art}, fmt.Errorf("start agent: %w", err)
	}
	return Result{Artifact: art, Stream: stream}, nil
}

// ingestPrompt is the app-owned ingest-synthesize prompt. The agent reads the
// fresh raw artifact and updates the wiki per AGENTS.md.
func ingestPrompt(art raw.Artifact) string {
	return fmt.Sprintf(
		"Ingest the new raw source at raw/%s/ into this wiki following AGENTS.md.\n"+
			"Source provenance: sourceType=%s sourcePath=%s fetchedAt=%s.\n"+
			"Read the source verbatim, extract atomic concepts, create or update "+
			"concept pages and any course hub pages, add cross-references across "+
			"the wiki, update index.md so every new page is cataloged, and append "+
			"a '## [%s] ingest | <title>' entry to log.md.\n"+
			"Work only inside this wiki directory. Never modify files under raw/.",
		art.Name, art.SourceType, art.SourcePath,
		art.FetchedAt.Format("2006-01-02T15:04:05Z07:00"),
		art.FetchedAt.Format("2006-01-02"),
	)
}
