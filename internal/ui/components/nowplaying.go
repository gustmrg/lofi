package components

import "strings"

type NowPlayingArgs struct {
	Width      int
	Label      string
	Title      string
	Artist     string
	Error      string
	Visualizer string
	Progress   string
	Loading    string
	AccentBar  string
}

func NowPlaying(a NowPlayingArgs) string {
	rows := []string{
		a.Label,
		a.Title,
		a.Artist,
	}
	if a.Error != "" {
		rows = append(rows, a.Error)
	}
	rows = append(rows, "")

	progressRow := a.Progress
	if a.Loading != "" {
		progressRow = a.Loading
	}
	rows = append(rows, a.Visualizer, progressRow)

	prefix := a.AccentBar + " "
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = prefix + r
	}
	return strings.Join(out, "\n")
}
