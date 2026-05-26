package components

import "github.com/charmbracelet/lipgloss"

func TitleBar(width int, center, right string, red, yellow, green, dim, faint lipgloss.Color) string {
	dot := func(c lipgloss.Color) string {
		return lipgloss.NewStyle().Foreground(c).Render("o")
	}
	dots := dot(red) + " " + dot(yellow) + " " + dot(green)
	centerStyle := lipgloss.NewStyle().Foreground(dim)
	rightStyle := lipgloss.NewStyle().Foreground(faint)

	dotsW := lipgloss.Width(dots)
	rightW := lipgloss.Width(right)
	centerStr := centerStyle.Render(center)
	centerW := lipgloss.Width(centerStr)

	leftGap := (width-centerW)/2 - dotsW
	if leftGap < 1 {
		leftGap = 1
	}
	rightGap := width - dotsW - leftGap - centerW - rightW
	if rightGap < 1 {
		rightGap = 1
	}

	return dots +
		lipgloss.NewStyle().Render(repeat(" ", leftGap)) +
		centerStr +
		lipgloss.NewStyle().Render(repeat(" ", rightGap)) +
		rightStyle.Render(right)
}

func repeat(s string, n int) string {
	if n <= 0 {
		return ""
	}
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
