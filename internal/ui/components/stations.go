package components

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/gustmrg/lofi/internal/provider"
)

func Stations(
	width int,
	stations []provider.Station,
	activeIdx int,
	section, sectionRule, active, idle, desc lipgloss.Style,
	accent, faint color.Color,
) string {
	header := sectionHeader(width, section, sectionRule, "STATIONS", fmt.Sprintf("%d available", len(stations)))
	rows := make([]string, 0, len(stations)+1)
	rows = append(rows, header)

	for i, s := range stations {
		rows = append(rows, stationRow(width, i+1, i == activeIdx, s, active, idle, desc, accent, faint))
	}
	return strings.Join(rows, "\n")
}

func sectionHeader(width int, section, rule lipgloss.Style, title, count string) string {
	left := section.Render(title)
	right := section.Render(count)
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	line := left + repeat(" ", gap) + right
	underline := rule.Render(repeat("-", width))
	return line + "\n" + underline
}

func stationRow(
	width int,
	num int,
	isActive bool,
	s provider.Station,
	active, idle, desc lipgloss.Style,
	accent, faint color.Color,
) string {
	indicator := lipgloss.NewStyle().Foreground(faint).Render("[ ]")
	nameStyle := idle
	if isActive {
		indicator = lipgloss.NewStyle().Foreground(accent).Render("[*]")
		nameStyle = active
	}

	prefix := fmt.Sprintf("%s %d. ", indicator, num)
	name := nameStyle.Render(s.Name)
	description := desc.Render(s.Description)

	first := prefix + name
	if w := lipgloss.Width(first); w < width {
		first += repeat(" ", width-w)
	}
	second := repeat(" ", lipgloss.Width(prefix)) + description
	if w := lipgloss.Width(second); w < width {
		second += repeat(" ", width-w)
	}
	return first + "\n" + second
}
