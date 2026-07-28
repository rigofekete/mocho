package wiki

// agentsSchema is the agent-neutral schema written to AGENTS.md inside the
// wiki on first scaffold. Any agent CLI (opencode, Claude Code) reads this to
// learn how the wiki is laid out and how ingest/query/lint operate.
const agentsSchema = `# AGENTS.md — mocho wiki schema

This directory is an LLM-maintained wiki following the Karpathy LLM-Wiki
pattern. A markdown wiki sits between you (the reader) and immutable raw
sources. The LLM writes and maintains the wiki; the human curates sources and
asks questions.

## Layout

    <wiki>/
    ├── AGENTS.md     this schema — agent operating manual
    ├── raw/          immutable source artifacts + sidecar .json metadata
    ├── concepts/     atomic concept pages (one idea per file)
    ├── courses/      one hub page per course
    ├── index.md      catalog of all wiki pages
    └── log.md        append-only chronological operation log

## Layers

- Raw sources (raw/): immutable. Agents read but never modify them. Every
  artifact has a sidecar <name>.json recording provenance (source URL/path,
  fetched-at time, source type).
- The wiki (concepts/, courses/, and any other pages): markdown the LLM owns.
  Pages are atomic, interlinked, and kept current as new sources arrive.
- The schema (this file): co-evolved by human + LLM. Higher scrutiny than page
  edits — every future agent run reads it.

## Page conventions

- One idea per file. A concept page lives at concepts/<slug>.md.
- A course hub lives at courses/<course-slug>.md; lesson-level notes are
  sections on the course page, not separate files.
- Title each page with a single H1: # Title
- Interlink with markdown links to the .md file, e.g.
  [goroutines](concepts/goroutines.md). Links must point at real files so
  they work in any markdown viewer.
- Keep page summaries to one line.

## index.md

index.md is the content-oriented catalog — the entry point for navigation.
List every page as a list item:

    - [Goroutines](concepts/goroutines.md) — lightweight concurrent execution units
    - [Channels](concepts/channels.md) — typed conduits between goroutines

Update index.md on every ingest, query filing, and page creation so the
catalog stays complete.

## log.md

log.md is the chronological, append-only audit trail. Append one entry per
operation, prefixed so the log is grep-able:

    ## [2026-04-02] ingest | Boot.dev — Go Concurrency
    ## [2026-04-02] query | How do goroutines yield?
    ## [2026-04-02] lint | found 2 orphans

"grep ^## \[ log.md | tail -5" shows the last 5 operations.

## Operations

- Ingest: read a new raw source, extract the key information, create or
  update concept and course pages, add cross-references across the wiki,
  append to log.md, and update index.md. A single source may touch many pages.
- Query: read index.md first to find relevant pages, drill into them, answer
  with citations to the wiki pages you drew from. Good answers get filed back
  into the wiki as new pages — questioning compounds the wiki.
- Lint: health-check the wiki. Surface contradictions between pages, stale
  claims superseded by newer sources, orphan pages with no inbound links,
  mentioned-but-missing concepts, and missing cross-references. Report
  findings and file any structural fixes as a reviewed change.

## Invariants the agent must never violate

1. Never modify files under raw/.
2. Every new page must appear in index.md.
3. Every operation must append to log.md.
4. Keep interlinks pointing at real files.
`

const indexTemplate = `# Index

Catalog of every wiki page. Updated on every ingest.

<!-- pages -->

_Pages will be listed here as sources are ingested._
`

const logHeader = `# Log

Append-only chronological record of wiki operations. One entry per operation:

    ## [YYYY-MM-DD] ingest | <title>
    ## [YYYY-MM-DD] query | <title>
    ## [YYYY-MM-DD] lint | <summary>
`