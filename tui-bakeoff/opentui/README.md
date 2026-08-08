# mocho-opentui — TUI bake-off winner

Prototype for [issue #8](https://github.com/rigofekete/mocho/issues/8). The bake-off
chose **OpenTUI** (`@opentui/core`, bun + TypeScript, native Zig core) as the
framework for the future `mocho tui`. Bubbletea remains as a backup in
`../bubbletea/`. See `../COMPARISON.md` for the evaluation.

## Prerequisite

The TUI is a pure HTTP client: it reads the wiki through a running `mocho` server,
so start one first (from the repo root):

```
go run ./cmd/mocho    # serves the wiki at the default addr 127.0.0.1:7777
```

## Dependencies

Requires [bun](https://bun.sh) (>= 1.3). Install the dependencies once:

```
cd tui-bakeoff/opentui && bun install
```

(`node_modules` and the lockfile live here and are gitignored.)

## Run

```
MOCHO_API=http://127.0.0.1:7777 bun src/index.ts
```

`MOCHO_API` is optional — it defaults to `http://127.0.0.1:7777`, the server's
default. Only set it if the server runs elsewhere.

## Keys

- `j` / `k` or `up` / `down` — move selection through the page list
- `/` — focus the search filter; typing filters the list by title/name/summary
- `q` / `ctrl+c` — quit

## Rendering

Pages render with `MarkdownRenderable`: tree-sitter syntax highlighting (GitHub-dark
theme), OSC8 hyperlinks for interlinks, auto-aligning tables. The API client is
decoupled from the UI (`src/client.ts`) and TDD'd against a `Bun.serve` stub.

### Tests / typecheck

```
cd tui-bakeoff/opentui
bun test          # 5 client tests
bunx tsc --noEmit # typecheck
```