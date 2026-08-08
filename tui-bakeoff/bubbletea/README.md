# mocho-bubbletea — TUI bake-off candidate (backup)

Prototype for [issue #8](https://github.com/rigofekete/mocho/issues/8), kept in the
repo. The bake-off chose **OpenTUI**; this Bubbletea prototype is kept as a backup /
for potential later reuse. See `../COMPARISON.md` for the full evaluation.

## Prerequisite

The TUI is a pure HTTP client: it reads the wiki through a running `mocho` server,
so start one first (from the repo root):

```
go run ./cmd/mocho    # serves the wiki at the default addr 127.0.0.1:7777
```

## Run

```
go run .                    # uses the default API base http://127.0.0.1:7777
MOCHO_API=http://127.0.0.1:7777 go run .   # or set the env
go run . http://127.0.0.1:7777             # or pass it as the first argument
```

Only override the host/port if the server runs elsewhere.

## Keys

- `j` / `k` (arrows, pgup/pgdown) — move selection through the page list
- `/` — focus the search filter; typing filters by title/name/summary
- `enter` / `esc` — leave the search filter
- `q` or `ctrl+c` — quit

## Why a separate module

`go.mod` lives here (not in the mocho module) so the prototype's dependencies
(bubbletea, lipgloss, glamour) never enter the shipped binary, and `go test ./...`
from the repo root ignores this directory.

### Tests

```
cd tui-bakeoff/bubbletea && go test ./client/
go vet ./...   # builds + vets
```