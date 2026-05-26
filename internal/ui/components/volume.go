package components

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func Volume(width, vol int, muted bool, label, fill, bg, mutedStyle, value lipgloss.Style) string {
	labelStr := label.Render("vol")
	var valStr string
	if muted {
		valStr = mutedStyle.Render("mute")
	} else {
		valStr = value.Render(fmt.Sprintf("%3d%%", vol))
	}

	barWidth := width - lipgloss.Width(labelStr) - lipgloss.Width(valStr) - 4
	if barWidth < 4 {
		barWidth = 4
	}
	if barWidth > 40 {
		barWidth = 40
	}

	displayVol := vol
	if muted {
		displayVol = 0
	}
	filled := barWidth * displayVol / 100
	if filled > barWidth {
		filled = barWidth
	}

	bar := fill.Render(strings.Repeat("=", filled)) + bg.Render(strings.Repeat("-", barWidth-filled))
	return labelStr + " " + bar + "  " + valStr
}
