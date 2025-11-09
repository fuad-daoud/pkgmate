package main

import (
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"

	"pkgmate/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	isPrivileged := os.Geteuid() == 0
	if isPrivileged {
		slog.Error("can't run in root")
		os.Exit(1)
	}
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("pkgmate %s\n", ui.Version)
		fmt.Printf("Built: %s\n", ui.BuildTime)
		fmt.Printf("Commit: %s\n", ui.GitCommit)

		if info, ok := debug.ReadBuildInfo(); ok {
			fmt.Printf("Go: %s\n", info.GoVersion)
		}
		os.Exit(0)
	}

	p := tea.NewProgram(ui.InitialModel(), tea.WithAltScreen(), tea.WithFPS(24))
	if _, err := p.Run(); err != nil {
		slog.Error("Failed to run", "err", err)
		os.Exit(1)
	}
}
