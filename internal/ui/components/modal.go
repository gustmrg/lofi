package components

import (
	"github.com/charmbracelet/lipgloss"
)

type AddStationModalArgs struct {
	Width  int
	Height int

	Title string
	Input string

	Loading bool
	Error   string

	BoxStyle    lipgloss.Style
	TitleStyle  lipgloss.Style
	HintStyle   lipgloss.Style
	StatusStyle lipgloss.Style
	ErrorStyle  lipgloss.Style
}

func AddStationModal(a AddStationModalArgs) string {
	status := a.HintStyle.Render("enter to confirm  ·  esc to cancel")
	if a.Loading {
		status = a.StatusStyle.Render("loading…")
	}
	if a.Error != "" {
		status = a.ErrorStyle.Render("! " + a.Error)
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		a.TitleStyle.Render(a.Title),
		"",
		a.Input,
		"",
		status,
	)

	box := a.BoxStyle.Render(body)

	w := a.Width
	if w <= 0 {
		w = lipgloss.Width(box)
	}
	h := a.Height
	if h <= 0 {
		h = lipgloss.Height(box)
	}
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, box)
}
