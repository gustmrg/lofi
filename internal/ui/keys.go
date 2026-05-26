package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	PlayPause key.Binding
	Next      key.Binding
	Prev      key.Binding
	Shuffle   key.Binding
	VolUp     key.Binding
	VolDown   key.Binding
	Rain      key.Binding
	Quit      key.Binding
	Stations  []key.Binding
}

func defaultKeys() keyMap {
	stations := make([]key.Binding, 5)
	for i := 0; i < 5; i++ {
		stations[i] = key.NewBinding(key.WithKeys(string(rune('1' + i))))
	}
	return keyMap{
		PlayPause: key.NewBinding(key.WithKeys(" ", "space")),
		Next:      key.NewBinding(key.WithKeys("down", "j")),
		Prev:      key.NewBinding(key.WithKeys("up", "k")),
		Shuffle:   key.NewBinding(key.WithKeys("s")),
		VolUp:     key.NewBinding(key.WithKeys("right", "l")),
		VolDown:   key.NewBinding(key.WithKeys("left", "h")),
		Rain:      key.NewBinding(key.WithKeys("r")),
		Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c", "esc")),
		Stations:  stations,
	}
}
