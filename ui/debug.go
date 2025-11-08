package ui

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type debugModel struct {
	file     string
	viewport viewport.Model
}

func (m debugModel) content() string {
	content, _ := os.ReadFile(m.file)
	return string(content)
}

func newDebug() *debugModel {
	logDir := filepath.Join(os.TempDir(), "user")
	os.MkdirAll(logDir, 0755)
	debugLogFile := filepath.Join(logDir, "debug.log")
	f, err := os.OpenFile(debugLogFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		slog.Error("Failed to open debug file", "err", err)
		os.Exit(1)
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(f, nil)))
	return &debugModel{file: debugLogFile}
}

func (m *debugModel) Update(msg tea.Msg) (*debugModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport = viewport.New(msg.Width-2, msg.Height-5) // -2 because of the frame -5 because of the frame and the footer
	}

	m.viewport.SetContent(m.content())
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

var titleStyle = func() lipgloss.Style {
	b := lipgloss.RoundedBorder()
	b.Right = "├"
	return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
}()

var infoStyle = func() lipgloss.Style {
	b := lipgloss.RoundedBorder()
	b.Left = "┤"
	return titleStyle.BorderStyle(b)
}()

func (m *debugModel) View() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	footer := lipgloss.JoinHorizontal(lipgloss.Center, line, info)
	return lipgloss.JoinVertical(lipgloss.Top, m.viewport.View(), footer)
}
