package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func Volume(width, vol int, label, fill, bg, value lipgloss.Style) string {
	labelStr := label.Render("vol")
	valStr := value.Render(fmt.Sprintf("%3d%%", vol))

	barWidth := width - lipgloss.Width(labelStr) - lipgloss.Width(valStr) - 4
	if barWidth < 4 {
		barWidth = 4
	}
	if barWidth > 40 {
		barWidth = 40
	}
	filled := barWidth * vol / 100
	if filled > barWidth {
		filled = barWidth
	}
	bar := fill.Render(strings.Repeat("=", filled)) + bg.Render(strings.Repeat("-", barWidth-filled))

	return labelStr + " " + bar + "  " + valStr
}
