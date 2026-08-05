# mocho

An LLM-maintained wiki study app: a Go backend + React SPA in a single binary,
following the [Karpathy LLM-Wiki](https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f) pattern.

## v1 tracer bullet (issue #2)

A single binary serves the SPA and JSON API, scaffolds an empty wiki, and lets
you browse wiki pages in the browser.

### Layout

```
cmd/mocho/      server entrypoint
internal/config  wiki path / addr resolution (flag > env > config file > default)
internal/server  HTTP API + embedded SPA
internal/wiki    on-disk wiki scaffold, index.md parsing, page reads
web/            Vite + React + Tailwind SPA
```

### Build & run

```sh
# Frontend (builds the SPA into web/dist)
cd web && npm install && npm run build && cd ..

# Backend (embeds web/dist into one binary)
go run ./cmd/mocho
```

Point at a custom wiki path (flag > env > config file > default
`~/Work/dev/mocho-wiki`):

```sh
go run ./cmd/mocho -wiki ~/my-wiki -addr 127.0.0.1:7777
MOCHO_WIKI=~/my-wiki go run ./cmd/mocho
```

Then open <http://127.0.0.1:7777>.

### Using the wiki

The wiki is a plain directory of Markdown. It defaults to
`~/Work/dev/mocho-wiki` (override with `-wiki` / `MOCHO_WIKI`). On first run
with an empty directory, mocho scaffolds this layout:

```
AGENTS.md  agent conventions for the wiki (read this)
index.md   page catalog — your table of contents
log.md     append-only operation log
raw/       immutable ingested sources
concepts/  one .md per idea
courses/   one course page per course-shaped source (created on demand)
```

Latest functional state: wiki scaffold + browse (#2) and local-path ingest with
agent synthesis (#3). Not yet wired: boot.dev / web URL adapters (#4), Q&A
(#5), raw reader + search + lint (#6).

### How the pipeline works

1. **Submit a source.** Today: a local file or directory path
   (`POST /api/ingest`, `{"source": "/path/..."}`). Coming in #4: boot.dev lesson
   URLs and public web URLs, via the same `{"source": "..."}` field (dispatched
   by scheme: `http(s)://` → remote, otherwise → local). Each source type is
   handled by a pluggable *source adapter* behind a `Source` interface; adding a
   source type later means adding an adapter, nothing else.
2. **Acquire — Go-owned.** The adapter copies the material **verbatim** into
   `raw/<slug>/` plus a sidecar `<slug>.json` recording provenance (origin
   URL/path, source type, fetched-at). From this moment the raw artifact is
   immutable — nothing ever edits it again.
3. **Synthesize — agent-owned.** mocho spawns `opencode run` with the wiki as
   working directory and an ingest prompt. The agent reads `raw/` + `AGENTS.md`
   (the wiki schema) and does the LLM work: creates or updates **concept
   pages** (and a **course page** when the source is course-shaped — its
   provenance declares an ordered lesson list), interlinks them, and updates
   `index.md` + `log.md`.
4. **Study.** Browse the rendered pages in the SPA. Query and lint arrive in
   #5 and #6.

`index.md` and `log.md` are maintained by the agent on every wiki-writing
operation — there is no separate indexer.

#### Ingest a source (app-driven)

Requires the `opencode` CLI on your `$PATH`.

```sh
curl -X POST http://127.0.0.1:7777/api/ingest \
  -H 'Content-Type: application/json' \
  -d '{"source":"/path/to/your-source.md"}'
```

The response is a Server-Sent Events stream showing agent progress live.

#### Ingest a source (manually)

Everything the app does is reproducible by hand — the wiki is agent-neutral:

```sh
# 1. Copy the source into raw/ verbatim, with a provenance sidecar
mkdir -p ~/Work/dev/mocho-wiki/raw/my-source
cp /path/to/your-source.md ~/Work/dev/mocho-wiki/raw/my-source/
cat > ~/Work/dev/mocho-wiki/raw/my-source.json <<'EOF'
{"source": "/path/to/your-source.md", "type": "local", "fetchedAt": "2026-08-05T12:00:00Z"}
EOF

# 2. Ask any agent CLI to synthesize it, from the wiki root
cd ~/Work/dev/mocho-wiki
opencode run "Ingest the new raw source at raw/my-source/ into this wiki following AGENTS.md"
```

The agent updates concept/course pages, `index.md`, and `log.md` exactly as
the app-driven pipeline would. (This manual path is also the fallback while
adapters for new source types don't exist yet.)

#### Add a page manually

1. Create `concepts/<slug>.md` with an H1 at the top:

   ```markdown
   # Goroutines

   Lightweight concurrent execution units.
   ```

2. Append a catalog entry to `index.md`:

   ```
   - [Goroutines](concepts/goroutines.md) — lightweight concurrent execution units
   ```

3. Append a log entry to `log.md` (`## [YYYY-MM-DD] ingest | Goroutines`), then
   refresh <http://127.0.0.1:7777> in the browser.

### Frontend dev

```sh
cd web && npm run dev        # Vite on :5173, proxies /api -> Go on :7777
# in another shell
go run ./cmd/mocho -addr 127.0.0.1:7777
```

### API

- `GET /api/health` — `{ "ok": true }`
- `GET /api/pages` — `{ "pages": [{ "name","title","summary" }] }` (from `index.md`)
- `GET /api/pages/{name}` — `{ "name","title","markdown" }`
- `POST /api/ingest` — `{ "source" }` (path or URL), streams ingest progress as SSE

### Test

```sh
go test ./...
```

Black-box tests at the HTTP seam use real `t.TempDir()` wikis.
