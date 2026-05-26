package components

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type keybind struct {
	key, label string
}

var bindings = []keybind{
	{"spc", "play/pause"},
	{"up/dn", "browse stations"},
	{"l/r", "volume"},
	{"m", "mute"},
	{"s", "shuffle"},
	{"1-5", "station"},
	{"a", "add"},
	{"d", "delete"},
	{"q", "quit"},
}

func Footer(width int, keyStyle, labelStyle lipgloss.Style) string {
	rule := lipgloss.NewStyle().Foreground(lipgloss.Color("#252a35")).Render(strings.Repeat("-", width))

	parts := make([]string, 0, len(bindings))
	for _, b := range bindings {
		parts = append(parts, keyStyle.Render(b.key)+" "+labelStyle.Render(b.label))
	}

	// Wrap parts onto multiple lines if too wide.
	var lines []string
	var cur strings.Builder
	curW := 0
	for i, p := range parts {
		pw := lipgloss.Width(p)
		sep := "   "
		sepW := 3
		if i == 0 {
			sep = ""
			sepW = 0
		}
		if curW+sepW+pw > width {
			lines = append(lines, cur.String())
			cur.Reset()
			curW = 0
			sep = ""
			sepW = 0
		}
		cur.WriteString(sep)
		cur.WriteString(p)
		curW += sepW + pw
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}

	return rule + "\n" + strings.Join(lines, "\n")
}
