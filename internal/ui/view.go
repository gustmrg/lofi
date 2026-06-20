package ui

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/gustmrg/lofi/internal/ui/components"
)

const (
	minWidth = 60
	hPad     = 2
)

func (m *Model) View() tea.View {
	bg := m.renderBackground()

	var v tea.View
	if m.mode == modeAddStation || m.mode == modeConfirmDelete || m.mode == modeNotice {
		var modal string

		if m.mode == modeAddStation {
			modal = components.AddStationModal(components.AddStationModalArgs{
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
		} else if m.mode == modeConfirmDelete {
			name := ""
			if m.activeIdx >= 0 && m.activeIdx < len(m.stations) {
				name = m.stations[m.activeIdx].Name
			}
			modal = components.ConfirmModal(components.ConfirmModalArgs{
				Title:        "DELETE STATION",
				Message:      "Delete " + name + "?",
				Hint:         "enter to delete  ·  esc to cancel",
				Loading:      m.removing,
				Error:        m.removeError,
				BoxStyle:     styleModalBox,
				TitleStyle:   styleModalTitle,
				MessageStyle: styleStationActive,
				HintStyle:    styleModalHint,
				StatusStyle:  styleModalStatus,
				ErrorStyle:   styleError,
			})
		} else {
			modal = components.ConfirmModal(components.ConfirmModalArgs{
				Title:        m.noticeTitle,
				Message:      m.noticeText,
				Hint:         "esc to close",
				BoxStyle:     styleModalBox,
				TitleStyle:   styleModalTitle,
				MessageStyle: styleStationActive,
				HintStyle:    styleModalHint,
				StatusStyle:  styleModalStatus,
				ErrorStyle:   styleError,
			})
		}
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
		statusBadge(m.playing, m.health, m.lastTick),
	)

	errLine := ""
	if m.lastError != "" {
		errLine = styleError.Render("! " + m.lastError)
	}

	var progressRow, loadingRow string
	switch {
	case m.loading:
		loadingRow = styleLoading.Render(animatedDots("Loading", m.lastTick))
	case !m.streamStarted:
		loadingRow = styleLoading.Render(animatedDots("Connecting", m.lastTick))
	default:
		progressRow = components.ProgressBar(contentWidth-4, m.elapsed, m.track.Duration, styleProgressFill, styleProgressBg, styleTime)
	}

	np := components.NowPlaying(components.NowPlayingArgs{
		Width:      contentWidth,
		Label:      styleNPLabel.Render("NOW PLAYING"),
		Title:      styleNPTitle.Render(m.track.Title),
		Artist:     styleNPArtist.Render("by " + m.track.Artist),
		Error:      errLine,
		Visualizer: components.Visualizer(m.visualizer[:], contentWidth-4, pal.Accent, pal.Blue, pal.Purple, pal.TextFaint),
		Progress:   progressRow,
		Loading:    loadingRow,
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

func statusBadge(playing bool, health connectionHealth, t time.Time) string {
	frame := statusFrame(t)
	if !playing {
		return styleStatusPaused.Render(statusBadgeText("- paused"))
	}
	switch health {
	case healthHealthy:
		return styleStatusHealthy.Render(statusBadgeText("ok healthy"))
	case healthUnstable:
		return styleStatusUnstable.Render(statusBadgeText(statusBadgeLabel(healthUnstable, frame)))
	case healthReconnecting:
		return styleStatusReconnecting.Render(statusBadgeText(statusBadgeLabel(healthReconnecting, frame)))
	case healthDisconnected:
		return styleStatusDisconnected.Render(statusBadgeText("! disconnected"))
	default:
		return styleStatusReconnecting.Render(statusBadgeText(statusBadgeLabel(healthReconnecting, frame)))
	}
}

func statusFrame(t time.Time) int {
	if t.IsZero() {
		return 0
	}
	return int(t.UnixNano() / int64(250*time.Millisecond))
}

func statusBadgeLabel(health connectionHealth, frame int) string {
	switch health {
	case healthUnstable:
		if frame%2 == 0 {
			return "~ unstable"
		}
		return "^ unstable"
	case healthReconnecting:
		switch frame % 3 {
		case 0:
			return ".   connecting"
		case 1:
			return "..  connecting"
		default:
			return "... connecting"
		}
	default:
		return ""
	}
}

func statusBadgeText(text string) string {
	const bodyWidth = 14
	if len(text) > bodyWidth {
		text = text[:bodyWidth]
	}
	return "[ " + text + strings.Repeat(" ", bodyWidth-len(text)) + " ]"
}

func animatedDots(base string, t time.Time) string {
	frame := statusFrame(t)
	dots := strings.Repeat(".", (frame%3)+1)
	return base + dots
}
