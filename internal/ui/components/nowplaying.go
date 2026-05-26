package components

import "strings"

type NowPlayingArgs struct {
	Width       int
	Label       string
	Title       string
	AmbienceTag string
	Artist      string
	Visualizer  string
	Progress    string
	AccentBar   string
}

func NowPlaying(a NowPlayingArgs) string {
	titleLine := a.Title
	if a.AmbienceTag != "" {
		titleLine = a.Title + "  " + a.AmbienceTag
	}

	rows := []string{
		a.Label,
		titleLine,
		a.Artist,
		"",
		a.Visualizer,
		a.Progress,
	}

	prefix := a.AccentBar + " "
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = prefix + r
	}
	return strings.Join(out, "\n")
}
