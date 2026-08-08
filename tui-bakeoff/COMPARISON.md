# TUI Bake-off — Bubbletea vs OpenTUI

Prototype bake-off for [issue #8](https://github.com/rigofekete/mocho/issues/8).
Both prototypes implement the **same minimal slice** against the live mocho HTTP API
(`GET /api/pages` + `GET /api/pages/{name}`): a **page list**, a **rendered-markdown
page view**, and a **search input** that filters the list. No production code was
touched; prototypes live in this directory, outside the shipped binary.

> **Both prototypes are kept here for side-by-side comparison — the choice is yours.**
> This doc records the evaluation and a *recommendation*; nothing is deleted until you
> pick a winner. See the "Recommendation" section at the bottom.

## Layout

- `bubbletea/` — Go prototype (separate `go.mod`, so it never lands in the mocho
  module). API client TDD'd with `httptest`; UI uses bubbletea + lipgloss + glamour.
- `opentui/` — bun + TypeScript prototype. API client TDD'd with `bun test` against
  a `Bun.serve` stub; UI uses `@opentui/core`'s imperative API + `MarkdownRenderable`
  (GitHub-dark theme, tree-sitter highlighting, OSC8 hyperlinks, search `InputRenderable`).

Run either against a running mocho server:

```
mocho -wiki <path> -addr 127.0.0.1:7777   # server
./bubbletea http://127.0.0.1:7777          # Bubbletea (build: go build -o mocho-bt . from bubbletea/)
MOCHO_API=http://127.0.0.1:7777 bun opentui/src/index.ts   # OpenTUI
```

Both were exercised against a real temp wiki (`concepts/`, `courses/`) served by the
mocho binary on `127.0.0.1:7788` and confirmed to do all three: list pages, navigate
the list, and render the selected page's markdown. Concrete run evidence:

- **OpenTUI** (run under a pty, `script`): rendered the list pane (Channels,
  Boot.dev Course Hub, with the ▸ cursor on Goroutines) **and** the rendered page in
  the same frame — `MarkdownRenderable` produced styled headings, a bulleted list,
  and the `[channels](concepts/channels.md)` link as an **OSC8 hyperlink**; status bar
  showed `concepts/goroutines.md`. GitHub-dark theme applied.
- **Bubbletea** (run under a pty, `script`): rendered the bordered two-pane layout —
  `Pages` list with the `▸` cursor and all three titles, the `Page` pane, the search
  hint bar `/ search  j/k move  q quit`. (Headless capture under a non-interactive pty
  occasionally missed the async page-load repaint; the API client is covered by
  `httptest` tests that drive the **real `http.ServeMux` wildcard route**
  `GET /api/pages/{name...}` to prove the `%2F`-encoded path resolves and decodes to
  the page name — i.e. wire-compatible with the live server.)

## Evaluation against the PRD criteria

| Criterion | Bubbletea (charm) | OpenTUI (@opentui/core) |
|---|---|---|
| **Markdown rendering quality** | Glamour — mature, clean GFM, ANSI styles, word-wrap. Re-render into a `viewport` per selection; links render as underlined text. | `MarkdownRenderable` — **richer**: tree-sitter syntax highlighting, multiple themes, OSC8 hyperlinks, tables with auto-alignment, conceal mode, and a built-in **streaming** mode (sticky bottom-scroll). Visually the better renderer for chat-style prose. |
| **DX / iteration speed** | `go run`/`go build` <1s after cache; same language & toolchain as the server; static types catch errors early. Strong. | bun hot-restart is instant; TS types catch errors (`tsc --noEmit`); but introduces a second toolchain and a separate process boundary. Slightly faster feedback, more moving parts. |
| **Layout capability for the future Q&A streaming view** | Manual: `lipgloss` join + `viewport`; streaming = incremental re-render / append. Doable, you build it yourself. | Flexbox-ish layout (`flexGrow`/`flexShrink`/`flexDirection`, absolute pos, z-index, opacity), `ScrollBoxRenderable`, and a `MarkdownRenderable` literally designed for **streaming assistant output** with sticky scroll. Purpose-built for exactly this view. |
| **Single-binary Go story** | **Ships inside the mocho binary as `mocho tui`** — zero extra runtime, matches the PRD's "one self-contained binary" goal. | Requires **bun installed**; TUI is a separate frontend process speaking to the API. Breaks the single-binary deployment story. |
| **Maintenance / community momentum** | Charm ecosystem is large, mature, stable, broadly used, long track record. | Younger (0.4.x), single-org stewardship (anomalyco/opencode), fast-moving, smaller community, likely breaking changes; but powers opencode in production (good signal). |

## Recommendation: Bubbletea (decision pending)

> Both prototypes are kept in this tree so the decision is yours to make. The notes
> below are a *recommendation*, not a done deal.

The case for **Bubbletea** turns on the deployment story and maturity, because the
OpenTUI-only advantages (streaming markdown, richer layout) are **implementable** in
Bubbletea, whereas OpenTUI's **bun runtime as a separate process** is **structural**
and directly conflicts with a core PRD implementation decision: *"SPA built and served
by Go via `go:embed` — one self-contained binary"* and *"TUI ... ships inside the mocho
binary as `mocho tui`."* Introducing a bun dependency just for the terminal frontend
makes running mocho meaningfully harder and is hard to reverse later.

The case for **OpenTUI** is the richer, purpose-built renderer: tree-sitter syntax
highlighting, OSC8 hyperlinks, auto-aligning tables,themes, and a `MarkdownRenderable`
with a built-in **streaming** mode (sticky bottom-scroll) that is essentially tailor-made
for the future Q&A streaming view — plus flexbox-style layout (`flexGrow`/`flexShrink`/
`flexDirection`, absolute pos, z-index, opacity). If you weight the chat/streaming UX and
visual polish above the single-binary story, OpenTUI wins on those axes.

Bubbletea also shares the server's language and toolchain (no runtime to install, no
native Zig core to ship per-platform in a second package), and the Charm ecosystem is
far more mature with a large community — lower long-term maintenance risk for a
hobby/student app maintained by one person.

If Bubbletea is chosen: the Q&A streaming view will be a `glamour`-rendered `viewport`
with incremental append + a manual sticky-bottom behavior, instead of OpenTUI's turnkey
streaming `MarkdownRenderable`. This is a known, bounded amount of extra work.

Per the bake-off acceptance criteria, the **losing** prototype should be deleted once
you've picked a winner. Until then, both live here for comparison and the `mocho tui`
follow-up ticket (#9) is blocked on your decision.