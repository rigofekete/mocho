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
courses/   one hub .md per course
```

Latest functional state: manual page authoring and source ingestion. Query and
lint operations are not wired into the app yet.

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
3. Refresh <http://127.0.0.1:7777> in the browser.

#### Ingest a source

Requires the `opencode` CLI on your `$PATH`.

```sh
curl -X POST http://127.0.0.1:7777/api/ingest \
  -H 'Content-Type: application/json' \
  -d '{"path":"/path/to/your-source.md"}'
```

The response is a Server-Sent Events stream. mocho copies the source into
`raw/` verbatim (immutable, never modified afterward), then `opencode` reads
`raw/` + `AGENTS.md` and synthesizes concept/course pages, updates `index.md`,
and appends an entry to `log.md`.

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
- `POST /api/ingest` — `{ "path" }`, streams ingest progress as SSE

### Test

```sh
go test ./...
```

Black-box tests at the HTTP seam use real `t.TempDir()` wikis.
