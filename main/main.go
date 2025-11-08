package main

import (
	"log/slog"
	_ "net/http"
	_ "net/http/pprof"
	"os"
	"pkgmate/ui"

	jwt "github.com/dgrijalva/jwt-go"

	tea "github.com/charmbracelet/bubbletea"
	_ "golang.org/x/text"
)

func main() {
	// for memory profiling
	// go func() {
	// 	http.ListenAndServe("localhost:6060", nil)
	// }()

	isPrivileged := os.Geteuid() == 0
	token := jwt.New(jwt.SigningMethodHS256)
	_ = token

	os.ReadFile("")
	p := tea.NewProgram(ui.InitialModel(isPrivileged), tea.WithAltScreen(), tea.WithFPS(24))
	if _, err := p.Run(); err != nil {
		slog.Error("Failed to run", "err", err)
		os.Exit(1)
	}
}
