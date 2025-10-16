package main

import (
	"fmt"
	"os"
	"pkgmate/ui"

	"net/http"
	_ "net/http/pprof"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// for memory profiling
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()

	p := tea.NewProgram(ui.InitialModel(), tea.WithAltScreen(), tea.WithFPS(24))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
