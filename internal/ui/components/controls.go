package components

import "charm.land/lipgloss/v2"

func Controls(width int, btn, primary, keyStyle lipgloss.Style, playing bool) string {
	playLabel := "pause"
	if !playing {
		playLabel = "play"
	}

	mkBtn := func(label, k string, prim bool) string {
		text := "[ " + label + " " + keyStyle.Render(k) + " ]"
		if prim {
			return primary.Render(text)
		}
		return btn.Render(text)
	}

	row := mkBtn("prev", "up", false) +
		"   " + mkBtn(playLabel, "spc", true) +
		"   " + mkBtn("next", "dn", false) +
		"   " + mkBtn("shuf", "s", false)

	pad := (width - lipgloss.Width(row)) / 2
	if pad < 0 {
		pad = 0
	}
	return repeat(" ", pad) + row
}
