package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

type palette struct {
	Bg        color.Color
	BgPanel   color.Color
	BgActive  color.Color
	Border    color.Color
	Text      color.Color
	TextDim   color.Color
	TextFaint color.Color
	Accent    color.Color
	AccentDim color.Color
	Green     color.Color
	Red       color.Color
	Blue      color.Color
	Purple    color.Color
}

var pal = palette{
	Bg:        lipgloss.Color("#0d0f12"),
	BgPanel:   lipgloss.Color("#13161b"),
	BgActive:  lipgloss.Color("#1f2430"),
	Border:    lipgloss.Color("#252a35"),
	Text:      lipgloss.Color("#c5cdd8"),
	TextDim:   lipgloss.Color("#6b7a8d"),
	TextFaint: lipgloss.Color("#3d4a5c"),
	Accent:    lipgloss.Color("#e8a86e"),
	AccentDim: lipgloss.Color("#b07a45"),
	Green:     lipgloss.Color("#7ec49d"),
	Red:       lipgloss.Color("#d46a6a"),
	Blue:      lipgloss.Color("#6ea5d4"),
	Purple:    lipgloss.Color("#a47ed4"),
}

var (
	styleLogo = lipgloss.NewStyle().
			Foreground(pal.Accent).
			Bold(true)

	styleLogoSub = lipgloss.NewStyle().
			Foreground(pal.TextFaint)

	styleStatusHealthy       = lipgloss.NewStyle().Foreground(pal.Green)
	styleStatusUnstable      = lipgloss.NewStyle().Foreground(pal.Accent)
	styleStatusReconnecting  = lipgloss.NewStyle().Foreground(pal.Blue)
	styleStatusDisconnected = lipgloss.NewStyle().Foreground(pal.Red).Bold(true)
	styleStatusPaused        = lipgloss.NewStyle().Foreground(pal.TextDim)

	styleSection = lipgloss.NewStyle().
			Foreground(pal.TextFaint).
			Bold(true)

	styleSectionRule = lipgloss.NewStyle().Foreground(pal.Border)

	styleNPLabel = lipgloss.NewStyle().
			Foreground(pal.TextFaint)

	styleNPTitle = lipgloss.NewStyle().
			Foreground(pal.Text).
			Bold(true)

	styleNPArtist = lipgloss.NewStyle().
			Foreground(pal.AccentDim)

	styleTime = lipgloss.NewStyle().
			Foreground(pal.TextFaint)

	styleProgressFill = lipgloss.NewStyle().Foreground(pal.Accent)
	styleProgressBg   = lipgloss.NewStyle().Foreground(pal.Border)

	styleVolFill = lipgloss.NewStyle().Foreground(pal.Green)
	styleVolBg   = lipgloss.NewStyle().Foreground(pal.Border)
	styleMuted   = lipgloss.NewStyle().Foreground(pal.Red).Bold(true)
	styleError   = lipgloss.NewStyle().Foreground(pal.Red)

	styleModalBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(pal.Accent).
			Padding(1, 3).
			Width(64)

	styleModalTitle  = lipgloss.NewStyle().Foreground(pal.Accent).Bold(true)
	styleModalHint   = lipgloss.NewStyle().Foreground(pal.TextFaint)
	styleModalStatus = lipgloss.NewStyle().Foreground(pal.TextDim)

	styleCtrlBtn = lipgloss.NewStyle().Foreground(pal.TextDim)

	styleCtrlBtnPrimary = lipgloss.NewStyle().Foreground(pal.Accent).Bold(true)

	styleCtrlKey = lipgloss.NewStyle().Foreground(pal.TextFaint)

	styleStationActive = lipgloss.NewStyle().
				Foreground(pal.Text).
				Bold(true)

	styleStationIdle = lipgloss.NewStyle().Foreground(pal.Text)

	styleStationDesc = lipgloss.NewStyle().Foreground(pal.TextFaint)

	styleStationMetaListeners = lipgloss.NewStyle().Foreground(pal.TextDim)
	styleStationMetaBitrate   = lipgloss.NewStyle().Foreground(pal.TextFaint)

	styleAccentBar = lipgloss.NewStyle().Foreground(pal.Accent)

	styleFooterKey = lipgloss.NewStyle().Foreground(pal.TextDim).Bold(true)

	styleFooterLabel = lipgloss.NewStyle().Foreground(pal.TextFaint)
)
