package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func ProgressBar(width int, elapsed, total time.Duration, fill, bg, timeStyle lipgloss.Style) string {
	elapsedStr := formatDur(elapsed)
	totalStr := formatDur(total)
	timeLen := lipgloss.Width(elapsedStr) + lipgloss.Width(totalStr) + 2 // two spaces
	barWidth := width - timeLen
	if barWidth < 4 {
		barWidth = 4
	}

	ratio := 0.0
	if total > 0 {
		ratio = float64(elapsed) / float64(total)
		if ratio > 1 {
			ratio = 1
		}
		if ratio < 0 {
			ratio = 0
		}
	}
	filled := int(float64(barWidth) * ratio)
	if filled > barWidth {
		filled = barWidth
	}

	bar := fill.Render(strings.Repeat("=", filled)) + bg.Render(strings.Repeat("-", barWidth-filled))
	return timeStyle.Render(elapsedStr) + " " + bar + " " + timeStyle.Render(totalStr)
}

func formatDur(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	secs := int(d.Seconds())
	return fmt.Sprintf("%d:%02d", secs/60, secs%60)
}
