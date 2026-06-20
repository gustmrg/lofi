package components

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// blocks indexed 0..7, lower-block elements producing a smooth gradient
var blocks = []string{" ", "▁", "▂", "▃", "▄", "▅", "▆", "▇"}

func Visualizer(heights []int, maxWidth int, low, high color.Color) string {
	if maxWidth <= 0 {
		maxWidth = len(heights)
	}
	bars := heights
	if len(bars) > maxWidth {
		bars = bars[:maxWidth]
	}

	var b strings.Builder
	for _, h := range bars {
		if h < 0 {
			h = 0
		}
		if h >= len(blocks) {
			h = len(blocks) - 1
		}
		ch := blocks[h]
		t := float64(h) / float64(len(blocks)-1)
		c := blendColor(low, high, t)
		b.WriteString(lipgloss.NewStyle().Foreground(c).Render(ch))
	}
	return b.String()
}

func blendColor(a, b color.Color, t float64) color.Color {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	ar, ag, ab, _ := a.RGBA()
	br, bg, bb, _ := b.RGBA()
	r := lerpU16(ar, br, t)
	g := lerpU16(ag, bg, t)
	bl := lerpU16(ab, bb, t)
	return color.RGBA64{R: r, G: g, B: bl, A: 0xffff}
}

func lerpU16(a, b uint32, t float64) uint16 {
	return uint16(float64(a) + (float64(b)-float64(a))*t)
}
