package components

import (
	"charm.land/lipgloss/v2"
)

type AddStationModalArgs struct {
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

	return a.BoxStyle.Render(body)
}
