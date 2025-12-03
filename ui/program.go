package ui

import (
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
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
	initColorProfile()

	Program = tea.NewProgram(InitialModel(), tea.WithAltScreen(), tea.WithFPS(24), tea.WithoutCatchPanics())
	if _, err := Program.Run(); err != nil {
		slog.Error("Failed to run", "err", err)
		return err
	}
	return nil
}

func initColorProfile() {
	// Allow user override via environment variable
	if forceProfile := os.Getenv("PKGMATE_COLOR_PROFILE"); forceProfile != "" {
		switch forceProfile {
		case "truecolor", "24bit":
			lipgloss.SetColorProfile(termenv.TrueColor)
		case "256", "ansi256":
			lipgloss.SetColorProfile(termenv.ANSI256)
		case "ansi", "16":
			lipgloss.SetColorProfile(termenv.ANSI)
		case "ascii", "none":
			lipgloss.SetColorProfile(termenv.Ascii)
		}
		return
	}

	// Smart detection: check TERM and COLORTERM
	term := os.Getenv("TERM")
	colorterm := os.Getenv("COLORTERM")

	// Most modern terminals set COLORTERM=truecolor or COLORTERM=24bit
	if colorterm == "truecolor" || colorterm == "24bit" {
		lipgloss.SetColorProfile(termenv.TrueColor)
		return
	}

	// Known TrueColor terminals (even if they don't advertise it)
	trueColorTerms := []string{
		"xterm-256color", "screen-256color", "tmux-256color",
		"alacritty", "kitty", "wezterm", "iterm", "iterm2",
		"vscode", "konsole", "gnome", "terminator",
	}

	for _, t := range trueColorTerms {
		if strings.Contains(term, t) {
			lipgloss.SetColorProfile(termenv.TrueColor)
			return
		}
	}

	// AWS CloudShell, browser terminals often use basic xterm
	// but support TrueColor - default to 256 color as safe middle ground
	if term == "xterm" {
		lipgloss.SetColorProfile(termenv.ANSI256)
		return
	}

	// Otherwise use lipgloss auto-detection (fallback)
}
