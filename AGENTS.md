# lofi — codebase guide

A terminal TUI that streams lofi audio from YouTube live streams. Requires `mpv` and `yt-dlp` in PATH.

## Build & run

```bash
go build -o lofi ./cmd/lofi   # build
go vet ./...                   # vet
go test -short ./...           # tests
./lofi                         # run (youtube provider + mpv)
./lofi -provider=mock          # run with in-memory mock (no network/mpv)
```

## Key dependencies

| Package | Purpose |
|---|---|
| `charm.land/bubbletea/v2` | TUI event loop (Elm-style Model/Update/View) |
| `charm.land/lipgloss/v2` | ANSI styling + Compositor for modal overlay |
| `charm.land/bubbles/v2` | `textinput` (add-station form) and `key` (keybinding helpers) |
| `mpv` (system binary) | Audio playback over IPC socket |
| `yt-dlp` (system binary) | Stream URL resolution from YouTube |

## Module path

`github.com/gustmrg/lofi` (Go module); the charm.land packages are the v2 vanity domain for charmbracelet/lipgloss, bubbletea, and bubbles.

## Directory layout

```
cmd/lofi/main.go               entry point — wires provider, player, and tea.Program
internal/
  player/
    player.go                  Player interface + Noop stub (used in tests)
    mpv/mpv.go                 mpv IPC implementation (unix socket, JSON protocol)
  provider/
    provider.go                Provider + StationManager interfaces; Station + Track types
    youtube/youtube.go         YouTube provider — yt-dlp for resolve/add, store for persistence
    mock/mock.go               In-memory provider for tests
  store/store.go               ~/.lofi/stations.json read/write (atomic rename)
  ui/
    model.go                   tea.Model — state, Init, Update, all commands
    view.go                    View() tea.View — renders background + compositor modal overlay
    styles.go                  Palette (image/color fields) + all lipgloss.Style vars
    keys.go                    keyMap struct + defaultKeys()
    model_test.go              Unit tests for Update logic
    components/
      controls.go              Play/pause/next/prev/shuffle button row
      footer.go                Keybinding legend, wraps to multiple lines
      header.go                Logo + status badge
      modal.go                 Add-station bordered box (no placement — caller uses Compositor)
      nowplaying.go            Track title/artist/visualizer/progress assembly
      progress.go              Progress bar with elapsed/total timestamps
      stations.go              Station list with active indicator
      util.go                  repeat() helper
      visualizer.go            ASCII bar visualizer (blocks ▁▂▃…▇)
      volume.go                Volume bar with mute display
```

## Architecture notes

**TUI loop:** `model.go` holds all mutable state. `Update` dispatches on message type:
- `tea.WindowSizeMsg` — stores terminal dimensions
- `tickMsg` — advances elapsed time and steps the visualizer (80 ms interval)
- `trackResolvedMsg` / `trackErrorMsg` — async yt-dlp resolve results
- `stationAddedMsg` / `stationRemovedMsg` / `addErrorMsg` / `removeErrorMsg` — station CRUD results
- `tea.KeyPressMsg` — routed to `handleKey` (normal mode) or `handleAddKey` (add-station mode)

**Modal overlay:** `view.go` renders the full background unconditionally via `renderBackground()`. When `mode == modeAddStation`, the bare modal box from `components.AddStationModal` is composited on top using `lipgloss.NewCompositor` at the computed center position. `v.AltScreen = true` is set on the returned `tea.View`; there is no `tea.WithAltScreen()` at program startup.

**Key matching:** `charm.land/bubbles/v2/key.Matches` is generic over `fmt.Stringer`. `tea.KeyPressMsg.String()` returns the key's text (for printable keys) or its name (`"space"`, `"up"`, `"ctrl+c"`, etc.) via the ultraviolet key name table.

**Station persistence:** `internal/store` reads/writes `~/.lofi/stations.json`. The YouTube provider loads it at startup; if absent, seeds with four default stations and writes the file. Add/remove operations mutate the in-memory slice then call `store.Save` atomically (write to `.tmp`, then rename).

**Player IPC:** `mpv` is launched once at startup with `--idle=yes --input-ipc-server=<socket>`. Play/pause/volume commands are sent as JSON over the unix socket.

## Color palette

All colors defined in `internal/ui/styles.go` as `image/color.Color` values (created via `lipgloss.Color("#hex")`). The palette variable `pal` is package-level; component functions that need per-element coloring (visualizer, stations indicator) accept `color.Color` parameters directly.

## Tests

`internal/ui/model_test.go` covers key-driven state transitions (play/pause, browse, station select, volume clamp, mute, elapsed reset). Helpers:
- `sendSpecial(t, m, tea.KeySpace/Up/Down/Left/Right)` — sends a special key
- `sendString(t, m, "a")` — sends a printable character key

`internal/provider/youtube/youtube_test.go` covers URL parsing and video ID extraction.
