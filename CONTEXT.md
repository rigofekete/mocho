# mocho

mocho is a personal study app implementing Karpathy's LLM-wiki pattern: it ingests study material into immutable raw sources and maintains an LLM-curated, interlinked markdown wiki that the user browses, reads, and questions.

## Language

### Wiki structure

**Wiki**:
The curated body of interlinked markdown pages built on top of raw sources; the thing the user studies from. A plain directory of markdown, ownable by any agent, independent of the app.
_Avoid_: notes, knowledge base, KB

**Raw source**:
An immutable, verbatim copy of one ingested piece of study material, stored with its provenance. Never edited after acquisition; wiki pages trace back to it.
_Avoid_: source file, import, dump

**Provenance**:
The metadata linking a raw source to where it came from: origin URL or path, source type, acquisition time.

**Wiki page**:
A single markdown file in the wiki — the unit the user studies from. The two page types in v1 are concept page and course page. Wiki pages are interconnected by markdown links and cataloged in the index.
_Avoid_: page (use the base term), doc

**Concept page**:
A wiki page about one atomic idea, heavily interlinked to other wiki pages. The universal page type — every source produces these. A concept may be linked from many course pages; concepts are not owned by any course.
_Avoid_: topic page, note

**Course page**:
A wiki page created only for course-shaped sources (raw sources whose provenance declares an ordered lesson list): it lists the source's lessons in order as sections and links the concept pages drawn from each. One-off sources (articles, single documents) produce concept pages only. There are no per-lesson pages.
_Avoid_: course hub, lesson page

**Index**:
The catalog of all wiki pages, maintained as part of every wiki-writing operation. Enables navigation without search infrastructure.
_Avoid_: index.md-as-filename in prose, TOC

**Log**:
The append-only, grep-able audit trail of wiki operations (`## [date] op | title`).

**Schema**:
The agent-neutral conventions file living inside the wiki that governs how any agent reads and writes it. The wiki's constitution.
_Avoid_: conventions file, config (it is not app config)

### Operations

**Ingest**:
The operation that acquires material into a new raw source and synthesizes it into wiki pages. Re-ingesting the same origin creates a new raw source; immutability is never violated.
_Avoid_: import

**Query**:
Asking the wiki a question; good answers are filed back into the wiki as pages, so questioning grows the wiki.
_Avoid_: chat, ask

**Lint**:
The operation that checks the wiki against its schema and reports violations.

### Modes

**Light mode**:
The read-only query mode for locating and summarizing material already in the wiki, backed by a cheap/free model. Incapable of writing, enforced by tool restriction.
_Avoid_: search mode, browse mode

**Wiki mode**:
The write-capable query mode, backed by a frontier model: reasons, dialogues, and files answers back into the wiki. The pattern's core loop.
_Avoid_: full mode, smart mode

### Actors

**Source adapter**:
The pluggable acquisition path for one source type (boot.dev, local path, web URL). Adding a source type means adding an adapter and nothing else.
_Avoid_: importer, fetcher

**Agent backend**:
The external agent CLI that performs wiki synthesis, query reasoning, and lint — the only writer of the wiki. Swappable behind an interface.
_Avoid_: the LLM, the model (the backend _runs_ a model; it is not one)
