package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

type bootModel struct{}

func (bootModel) Init() tea.Cmd { return nil }

func (m bootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.KeyMsg); ok {
		return m, tea.Quit
	}
	return m, nil
}

func (bootModel) View() string {
	return "lofi -- press any key to quit\n"
}

func main() {
	prog := tea.NewProgram(bootModel{}, tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "lofi: %v\n", err)
		os.Exit(1)
	}
}
