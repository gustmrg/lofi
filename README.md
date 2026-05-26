# lofi

A terminal music player for lofi hip hop streams. Streams audio from YouTube Live directly in your terminal with a minimal, keyboard-driven interface.

```
lofi                           terminal beats player             [ * streaming ]

| NOW PLAYING
| lo-fi hip hop radio - beats to relax/study to
| by Lofi Girl
|
| ▁▂▄▃▅▂▄▁▃▅▂▁▄▃▂▅▄▁▃▂▄▅▁▃▂▄▁▃▅▂▁▄▃▂▅▄▁▃▂▄▅▁▃▂
| 12:04 ============================================ 3:14:00

          [ prev up ]   [ pause spc ]   [ next dn ]   [ shuf s ]

STATIONS                                                        4 available
--------------------------------------------------------------------------------
[*] 1. lofi girl - beats to study                                   0 listeners
       lofi hip hop radio . beats to relax/study to                        128k
[ ] 2. lofi girl - sleep                                            0 listeners
[ ] 3. chillhop radio                                               0 listeners
[ ] 4. lofi daily                                                   0 listeners

vol =======================================   72%

spc play/pause   up/dn browse stations   l/r volume   m mute   s shuffle
1-5 station   a add   d delete   q quit
```

## Requirements

- [mpv](https://mpv.io) — audio playback
- [yt-dlp](https://github.com/yt-dlp/yt-dlp) — stream URL resolution

Both must be available in your `PATH`.

## Install

```bash
go install github.com/gustmrg/lofi/cmd/lofi@latest
```

Or build from source:

```bash
git clone https://github.com/gustmrg/lofi
cd lofi
go build -o lofi ./cmd/lofi
```

## Usage

```bash
lofi
```

On first run, four default stations are written to `~/.lofi/stations.json`. You can add or remove stations from inside the app.

## Keybindings

| Key | Action |
|---|---|
| `space` | Play / pause |
| `↑` / `k` | Previous station |
| `↓` / `j` | Next station |
| `→` / `l` | Volume up |
| `←` / `h` | Volume down |
| `m` | Mute / unmute |
| `s` | Shuffle |
| `1`–`5` | Jump to station |
| `a` | Add station by YouTube URL |
| `d` | Delete current station |
| `q` / `ctrl+c` | Quit |

## Stations

Stations are stored in `~/.lofi/stations.json`. Press `a` to add any YouTube video or live stream URL, and `d` to remove the current one. Changes persist across sessions.

## How it works

`yt-dlp` resolves the stream URL for the active YouTube video. `mpv` plays the audio stream via a local IPC socket. The TUI is built with [Bubble Tea v2](https://charm.land/bubbletea/v2) and [Lipgloss v2](https://charm.land/lipgloss/v2) — the add-station modal floats over the background using the Lipgloss compositor.

## License

MIT
