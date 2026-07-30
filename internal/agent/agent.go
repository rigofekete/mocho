// Package agent abstracts the LLM agent CLI (opencode by default) behind an
// interface so tests substitute a scripted fake and a Claude Code adapter can
// be added later without refactoring. A run takes a prompt, a working
// directory, a query mode (light/wiki), and the model to use for that mode —
// the model is resolved from config per mode, not chosen by the backend.
package agent

import (
	"context"
	"io"
)

// Mode selects the query posture against the wiki.
type Mode string

const (
	// ModeLight is a read-only lookup backed by a cheap model. Write/edit
	// tools are disabled by the backend's restricted tool profile so the
	// wiki cannot be modified in light mode.
	ModeLight Mode = "light"
	// ModeWiki is the full Karpathy loop back: ingest synthesis, query
	// write-back, and lint run a write-capable model against the wiki.
	ModeWiki Mode = "wiki"
)

// AgentBackend runs an agent invocation and streams its stdout. The caller
// (the app) owns prompt templates and resolves the model per mode from
// config; the backend just runs. The returned reader is closed by the caller;
// ctx cancellation should kill the run. A non-nil error on the final Read
// (other than io.EOF) signals an agent run failure (non-zero exit).
type AgentBackend interface {
	Run(ctx context.Context, prompt, workdir string, mode Mode, model string) (io.ReadCloser, error)
}
