# mocho-opentui — TUI bake-off candidate

Prototype for [issue #8](https://github.com/rigofekete/mocho/issues/8). The bake-off
decision is **under reconsideration** (see the issue): OpenTUI's structural impact on
the single-binary architecture is being rethought, so no winner is asserted here yet.
Bubbletea's candidate lives in `../bubbletea/`; the comparison is in `../COMPARISON.md`.

This candidate is a **React spike**: the UI is written as TSX on `@opentui/react`
(the React reconciler over the native Zig core), demonstrating that the TUI and the
browser SPA can share hooks/logic/state and a component architecture. It also proves
**image rendering** in the terminal — a jimp-generated PNG graph of the wiki is drawn
and displayed with `<image>` (kitty/sixel when the terminal supports it, unicode-blocks
otherwise).

## Prerequisite

The TUI is a pure HTTP client: it reads the wiki through a running `mocho` server,
so start one first (from the repo root):

```
go run ./cmd/mocho    # serves the wiki at the default addr 127.0.0.1:7777
```

## Dependencies

Requires [bun](https://bun.sh) (>= 1.3). Install once:

```
cd tui-bakeoff/opentui && bun install
```

(`node_modules` and `dist/` are gitignored; `bun.lock` is committed.)

## Run (dev)

```
MOCHO_API=http://127.0.0.1:7777 bun run dev
```

`MOCHO_API` is optional — it defaults to `http://127.0.0.1:7777`, the server's
default. Only set it if the server runs elsewhere.

## Build a single binary

```
bun run build     # -> dist/mocho-tui
./dist/mocho-tui  # self-contained; no bun/node_modules needed (native core bundled)
```

`bun build --compile` bundles the JS + the OpenTUI native Zig core into one
executable (tested from a clean dir with no `node_modules`).

## Keys

- `j` / `k` or `up` / `down` — move selection through the page list
- `/` — focus the search filter; typing filters the list by title/name/summary
- `enter` — leave the search filter
- `g` — toggle the graph view (renders a generated PNG of the wiki network)
- `q` / `ctrl+c` — quit

## Rendering

- Pages render with `<markdown>` (`MarkdownRenderable`): tree-sitter syntax
  highlighting (GitHub-dark theme), OSC8 hyperlinks for interlinks, tables.
- Graph view renders a jimp-generated PNG via `<image>` — `protocol="auto"` picks
  kitty/sixel where available, else unicode blocks.
- The API client is decoupled from the UI (`src/client.ts`) and TDD'd against a
  `Bun.serve` stub.

### Tests / typecheck

```
cd tui-bakeoff/opentui
bun test          # 5 client tests
bun run typecheck # tsc --noEmit (jsx: @opentui/react)
```
