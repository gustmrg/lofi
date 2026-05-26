package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/gustmrg/lofi/internal/ui/components"
)

func (m *Model) View() string {
	contentWidth := panelWidth - 4 // panel padding 2 on each side

	title := components.TitleBar(contentWidth, "lofi -- ~/music/streams", "v0.1.0",
		pal.Red, pal.Accent, pal.Green, pal.TextDim, pal.TextFaint)

	header := components.Header(contentWidth,
		styleLogo.Render("lofi"),
		styleLogoSub.Render("terminal beats player"),
		styleStatusBadge.Render(statusText(m.playing)),
	)

	np := components.NowPlaying(components.NowPlayingArgs{
		Width:       contentWidth,
		Label:       styleNPLabel.Render("NOW PLAYING"),
		Title:       styleNPTitle.Render(m.track.Title),
		AmbienceTag: ambienceTag(m.rainOn),
		Artist:      styleNPArtist.Render("by " + m.track.Artist),
		Visualizer:  components.Visualizer(m.visualizer[:], contentWidth-4, pal.Accent, pal.Blue, pal.Purple, pal.TextFaint),
		Progress:    components.ProgressBar(contentWidth-4, m.elapsed, m.track.Duration, styleProgressFill, styleProgressBg, styleTime),
		AccentBar:   styleAccentBar.Render("|"),
	})

	controls := components.Controls(contentWidth,
		styleCtrlBtn, styleCtrlBtnPrimary, styleCtrlKey, m.playing)

	stationList := components.Stations(contentWidth, m.stations, m.activeIdx,
		styleSection, styleSectionRule,
		styleStationActive, styleStationIdle, styleStationDesc,
		styleStationMetaListeners, styleStationMetaBitrate,
		pal.Accent, pal.TextFaint)

	vol := components.Volume(contentWidth, m.volume,
		styleFooterLabel, styleVolFill, styleVolBg, styleStationMetaListeners)

	footer := components.Footer(contentWidth, styleFooterKey, styleFooterLabel)

	body := strings.Join([]string{
		title,
		"",
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

	panel := stylePanel.Render(body)

	if m.width == 0 || m.height == 0 {
		return panel
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel)
}

func statusText(playing bool) string {
	if playing {
		return "[ * streaming ]"
	}
	return "[ - paused    ]"
}

func ambienceTag(on bool) string {
	if !on {
		return ""
	}
	return styleAmbience.Render("[ rain ]")
}

