package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/gustmrg/lofi/internal/ui/components"
)

const (
	minWidth = 60
	hPad     = 2
)

func (m *Model) View() string {
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

