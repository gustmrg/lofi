package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/gustmrg/lofi/internal/player"
	"github.com/gustmrg/lofi/internal/player/mpv"
	"github.com/gustmrg/lofi/internal/provider"
	"github.com/gustmrg/lofi/internal/provider/mock"
	"github.com/gustmrg/lofi/internal/provider/youtube"
	"github.com/gustmrg/lofi/internal/ui"
)

func main() {
	providerFlag := flag.String("provider", "youtube", "provider: youtube|mock")
	flag.Parse()

	prov, pl, err := build(*providerFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lofi: %v\n", err)
		os.Exit(1)
	}
	defer pl.Close()

	model, err := ui.NewModel(prov, pl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lofi: %v\n", err)
		os.Exit(1)
	}

	prog := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "lofi: %v\n", err)
		os.Exit(1)
	}
}

func build(name string) (provider.Provider, player.Player, error) {
	switch name {
	case "mock":
		return mock.New(), player.Noop{}, nil
	case "youtube":
		yp, err := youtube.New()
		if err != nil {
			return nil, nil, fmt.Errorf("youtube provider: %w", err)
		}
		mp, err := mpv.New()
		if err != nil {
			return nil, nil, fmt.Errorf("mpv player: %w", err)
		}
		return yp, mp, nil
	default:
		return nil, nil, fmt.Errorf("unknown provider %q (want: youtube|mock)", name)
	}
}
