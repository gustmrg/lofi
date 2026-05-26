package components

import "charm.land/lipgloss/v2"

func Header(width int, logo, sub, status string) string {
	left := lipgloss.JoinVertical(lipgloss.Left, logo, sub)
	leftW := lipgloss.Width(left)
	statusW := lipgloss.Width(status)
	gap := width - leftW - statusW
	if gap < 1 {
		gap = 1
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, left, repeat(" ", gap), status)
}
