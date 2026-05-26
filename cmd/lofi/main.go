package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/gustmrg/lofi/internal/provider/mock"
	"github.com/gustmrg/lofi/internal/ui"
)

func main() {
	p := mock.New()
	model, err := ui.NewModel(p)
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
