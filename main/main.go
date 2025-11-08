package main

import (
	"log/slog"
	"os"
	"pkgmate/ui"

	jwt "github.com/dgrijalva/jwt-go"

	tea "github.com/charmbracelet/bubbletea"
	_ "golang.org/x/text"
)

func main() {
	token := jwt.New(jwt.SigningMethodHS256)
	_ = token
	isPrivileged := os.Geteuid() == 0
	p := tea.NewProgram(ui.InitialModel(isPrivileged), tea.WithAltScreen(), tea.WithFPS(24))
	if _, err := p.Run(); err != nil {
		slog.Error("Failed to run", "err", err)
		os.Exit(1)
	}
}
