package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// blocks indexed 0..7, lower-block elements producing a smooth gradient
var blocks = []string{" ", "▁", "▂", "▃", "▄", "▅", "▆", "▇"}

func Visualizer(heights []int, maxWidth int, accent, blue, purple, faint lipgloss.Color) string {
	if maxWidth <= 0 {
		maxWidth = len(heights)
	}
	bars := heights
	if len(bars) > maxWidth {
		bars = bars[:maxWidth]
	}

	var b strings.Builder
	for i, h := range bars {
		if h < 0 {
			h = 0
		}
		if h >= len(blocks) {
			h = len(blocks) - 1
		}
		ch := blocks[h]
		color := accent
		switch {
		case i%5 == 0:
			color = purple
		case i%3 == 0:
			color = blue
		case i%2 == 1:
			color = faint
		}
		b.WriteString(lipgloss.NewStyle().Foreground(color).Render(ch))
	}
	return b.String()
}
