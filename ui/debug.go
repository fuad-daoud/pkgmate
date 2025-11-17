package ui

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

type debugModel struct {
	viewport viewport.Model
	rootPath string
	width    int
	height   int
}

func (m debugModel) content() string {
	root, err := os.OpenRoot(m.rootPath)
	if err != nil {
		slog.Error("Failed to open debug dir", "err", err)
		os.Exit(1)
	}
	content, err := root.ReadFile("debug.log")
	if err != nil {
		slog.Error("Failed to open debug file", "err", err)
		return ""
	}
	return string(content)
}
func (debugModel) Init() tea.Cmd { return nil }

func newDebug() debugModel {
	rootPath := filepath.Join(os.TempDir(), "user")
	err := os.MkdirAll(rootPath, 0750)
	if err != nil {
		slog.Error("Failed to create logs dir", "err", err)
		os.Exit(1)
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		slog.Error("Failed to open debug dir", "err", err)
		os.Exit(1)
	}
	f, err := root.OpenFile("debug.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		slog.Error("Failed to open debug file", "err", err)
		os.Exit(1)
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(f, nil)))
	slog.Info("pkgmate", "version", Version)
	slog.Info("pkgmate", "Built", BuildTime)
	slog.Info("pkgmate", "Commit", GitCommit)

	if info, ok := debug.ReadBuildInfo(); ok {
		slog.Info("pkgmate", "go version", info.GoVersion)
	}
	return debugModel{rootPath: rootPath}
}

func (m debugModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case DisplayResizeEvent:
		m.width = msg.width
		m.height = msg.height
		m.viewport = viewport.New(msg.width, msg.height)
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

func (m debugModel) View() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(1, m.viewport.Width-lipgloss.Width(info)))
	footer := lipgloss.JoinHorizontal(lipgloss.Center, line, info)
	content := lipgloss.JoinVertical(lipgloss.Top, m.viewport.View(), footer)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}
