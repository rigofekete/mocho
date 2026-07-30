# mocho-bubbletea (TUI bake-off winner)

Throwaway prototype for issue #8. Seeds the future `mocho tui` (see `../COMPARISON.md`).

## Run

```
go run . http://127.0.0.1:7777        # argument = API base URL
MOCHO_API=http://127.0.0.1:7777 go run .   # or env
```

## Keys

- `j` / `k` (or arrows / pgup-pgdown) — move selection
- `/` — focus the search filter; typing filters the page list by title/name/summary
- `enter` / `esc` — leave search
- `q` — quit

## Layout

This is a separate Go module (`go.mod`) so its dependencies never enter the mocho
module, and `go test ./...` from the repo root does not pick it up.

### Tests

```
cd tui-bakeoff/bubbletea && go test ./client/
```