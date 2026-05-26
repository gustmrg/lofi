package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/gustmrg/lofi/internal/provider"
)

func Stations(
	width int,
	stations []provider.Station,
	activeIdx int,
	section, sectionRule, active, idle, desc, metaListeners, metaBitrate lipgloss.Style,
	accent, faint lipgloss.Color,
) string {
	header := sectionHeader(width, section, sectionRule, "STATIONS", fmt.Sprintf("%d available", len(stations)))
	rows := make([]string, 0, len(stations)+1)
	rows = append(rows, header)

	for i, s := range stations {
		rows = append(rows, stationRow(width, i+1, i == activeIdx, s, active, idle, desc, metaListeners, metaBitrate, accent, faint))
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
	active, idle, desc, metaListeners, metaBitrate lipgloss.Style,
	accent, faint lipgloss.Color,
) string {
	indicator := lipgloss.NewStyle().Foreground(faint).Render("[ ]")
	nameStyle := idle
	if isActive {
		indicator = lipgloss.NewStyle().Foreground(accent).Render("[*]")
		nameStyle = active
	}

	prefix := fmt.Sprintf("%s %d. ", indicator, num)
	name := nameStyle.Render(s.Name)
	description := desc.Render("  " + s.Description)

	listeners := metaListeners.Render(fmt.Sprintf("%d listeners", s.Listeners))
	bitrate := metaBitrate.Render(s.Bitrate)
	meta := listeners + "  " + bitrate

	leftBlock := prefix + name + "\n" + repeat(" ", lipgloss.Width(prefix)) + description

	leftW := maxLineWidth(leftBlock)
	metaW := lipgloss.Width(meta)
	gap := width - leftW - metaW
	if gap < 1 {
		gap = 1
	}

	lines := strings.Split(leftBlock, "\n")
	lines[0] = lines[0] + repeat(" ", width-lipgloss.Width(lines[0])-metaW) + meta
	return strings.Join(lines, "\n")
}

func maxLineWidth(s string) int {
	max := 0
	for _, ln := range strings.Split(s, "\n") {
		if w := lipgloss.Width(ln); w > max {
			max = w
		}
	}
	return max
}
