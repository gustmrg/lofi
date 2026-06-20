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

type ConfirmModalArgs struct {
	Title   string
	Message string
	Hint    string

	Loading bool
	Error   string

	BoxStyle     lipgloss.Style
	TitleStyle   lipgloss.Style
	MessageStyle lipgloss.Style
	HintStyle    lipgloss.Style
	StatusStyle  lipgloss.Style
	ErrorStyle   lipgloss.Style
}

func ConfirmModal(a ConfirmModalArgs) string {
	status := a.HintStyle.Render(a.Hint)
	if a.Loading {
		status = a.StatusStyle.Render("deleting…")
	}
	if a.Error != "" {
		status = a.ErrorStyle.Render("! " + a.Error)
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		a.TitleStyle.Render(a.Title),
		"",
		a.MessageStyle.Render(a.Message),
		"",
		status,
	)

	return a.BoxStyle.Render(body)
}
