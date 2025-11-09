package main

import (
	"log/slog"
	"os"
	"pkgmate/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	isPrivileged := os.Geteuid() == 0
	if isPrivileged {
		slog.Error("can't run in root")
		os.Exit(1)
	}

	p := tea.NewProgram(ui.InitialModel(), tea.WithAltScreen(), tea.WithFPS(24))
	if _, err := p.Run(); err != nil {
		slog.Error("Failed to run", "err", err)
		os.Exit(1)
	}
}
