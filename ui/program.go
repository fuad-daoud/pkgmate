package ui

import (
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"

	tea "github.com/charmbracelet/bubbletea"
)

var Program *tea.Program

func CrashApp() {
	Program.ReleaseTerminal()
	if r := recover(); r != nil {
		stackTrace := string(debug.Stack())

		slog.Error("Application crashed", "error", r, "stack", stackTrace)

		Program = tea.NewProgram(
			NewPanicScreen(r, stackTrace),
			tea.WithAltScreen(),
			tea.WithFPS(24),
		)

		if _, err := Program.Run(); err != nil {
			slog.Error("Failed to run", "err", err)
			os.Exit(1)
		}

		os.Exit(0)
	}

}

func Run() error {
	defer CrashApp()
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("pkgmate %s\n", Version)
		fmt.Printf("Built: %s\n", BuildTime)
		fmt.Printf("Commit: %s\n", GitCommit)

		if info, ok := debug.ReadBuildInfo(); ok {
			fmt.Printf("Go: %s\n", info.GoVersion)
		}
	}

	Program = tea.NewProgram(InitialModel(), tea.WithAltScreen(), tea.WithFPS(24), tea.WithoutCatchPanics())
	if _, err := Program.Run(); err != nil {
		slog.Error("Failed to run", "err", err)
		return err
	}
	return nil
}
