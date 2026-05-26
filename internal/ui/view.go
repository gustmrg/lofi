package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	tea "charm.land/bubbletea/v2"

	"github.com/gustmrg/lofi/internal/ui/components"
)

const (
	minWidth = 60
	hPad     = 2
)

func (m *Model) View() tea.View {
	bg := m.renderBackground()

	var v tea.View
	if m.mode == modeAddStation {
		modal := components.AddStationModal(components.AddStationModalArgs{
			Title:       "ADD STATION",
			Input:       m.input.View(),
			Loading:     m.adding,
			Error:       m.addError,
			BoxStyle:    styleModalBox,
			TitleStyle:  styleModalTitle,
			HintStyle:   styleModalHint,
			StatusStyle: styleModalStatus,
			ErrorStyle:  styleError,
		})
		modalW := lipgloss.Width(modal)
		modalH := lipgloss.Height(modal)
		x := (m.width - modalW) / 2
		y := (m.height - modalH) / 2
		composed := lipgloss.NewCompositor(
			lipgloss.NewLayer(bg),
			lipgloss.NewLayer(modal).X(x).Y(y).Z(1),
		).Render()
		v.SetContent(composed)
	} else {
		v.SetContent(bg)
	}
	v.AltScreen = true
	return v
}

func (m *Model) renderBackground() string {
	width := m.width
	if width == 0 {
		width = minWidth + hPad*2
	}
	contentWidth := width - hPad*2
	if contentWidth < minWidth {
		contentWidth = minWidth
	}

	header := components.Header(contentWidth,
		styleLogo.Render("lofi"),
		styleLogoSub.Render("terminal beats player"),
		styleStatusBadge.Render(statusText(m.playing)),
	)

	errLine := ""
	if m.lastError != "" {
		errLine = styleError.Render("! " + m.lastError)
	}
	np := components.NowPlaying(components.NowPlayingArgs{
		Width:      contentWidth,
		Label:      styleNPLabel.Render("NOW PLAYING"),
		Title:      styleNPTitle.Render(m.track.Title),
		Artist:     styleNPArtist.Render("by " + m.track.Artist),
		Error:      errLine,
		Visualizer: components.Visualizer(m.visualizer[:], contentWidth-4, pal.Accent, pal.Blue, pal.Purple, pal.TextFaint),
		Progress:   components.ProgressBar(contentWidth-4, m.elapsed, m.track.Duration, styleProgressFill, styleProgressBg, styleTime),
		AccentBar:  styleAccentBar.Render("|"),
	})

	controls := components.Controls(contentWidth,
		styleCtrlBtn, styleCtrlBtnPrimary, styleCtrlKey, m.playing)

	stationList := components.Stations(contentWidth, m.stations, m.activeIdx,
		styleSection, styleSectionRule,
		styleStationActive, styleStationIdle, styleStationDesc,
		styleStationMetaListeners, styleStationMetaBitrate,
		pal.Accent, pal.TextFaint)

	vol := components.Volume(contentWidth, m.volume, m.muted,
		styleFooterLabel, styleVolFill, styleVolBg, styleMuted, styleStationMetaListeners)

	footer := components.Footer(contentWidth, styleFooterKey, styleFooterLabel)

	body := strings.Join([]string{
		header,
		"",
		np,
		"",
		controls,
		"",
		stationList,
		"",
		vol,
		"",
		footer,
	}, "\n")

	return lipgloss.NewStyle().Padding(1, hPad).Render(body)
}

func statusText(playing bool) string {
	if playing {
		return "[ * streaming ]"
	}
	return "[ - paused    ]"
}
