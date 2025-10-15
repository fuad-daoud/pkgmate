package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"pkgmate/ui"
)

func main() {
	p := tea.NewProgram(ui.InitialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
