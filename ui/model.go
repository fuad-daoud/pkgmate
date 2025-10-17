package ui

import (
	"log/slog"
	"os"
	"pkgmate/backend"
	"reflect"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	width          int
	height         int
	viewportHeight int
	viewportWidth  int
	header         headerModel
	displayModel        displayModel
	footer         footerModel
	showDebug      bool
}

func InitialModel() model {
	return model{
		header:    newHeader(),
		displayModel:   newDisplay(),
		footer:    newFooter(),
		showDebug: false,
	}
}

func fetchPackages() tea.Msg {
	pkgs, err := backend.LoadPackages()
	if err != nil {
		os.Exit(1)
	}
	return pkgs
}

func (m model) Init() tea.Cmd {
	return fetchPackages
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	slog.Info("New event", "type", reflect.TypeOf(msg))
	commands := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewportHeight = msg.Height - (msg.Height / 5)
		m.viewportWidth = msg.Width - (msg.Width / 5)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if !m.footer.search.Focused() {
				return m, tea.Quit
			}
		case "ctrl+d":
			m.showDebug = !m.showDebug
		}
	}
	var footerCmd tea.Cmd
	m.footer, footerCmd = m.footer.Update(msg)
	commands = append(commands, footerCmd)

	var displayCmd tea.Cmd
	m.displayModel, displayCmd = m.displayModel.Update(msg)
	commands = append(commands, displayCmd)

	var headerCmd tea.Cmd
	
	m.header, headerCmd = m.header.Update(msg)
	commands = append(commands, headerCmd)

	var debugCmd tea.Cmd
	debug, debugCmd = debug.Update(msg)
	commands = append(commands, debugCmd)

	return m, tea.Batch(commands...)
}

func (m model) View() string {
	if m.showDebug {
		content := frameStyle.Render(debug.View())
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}
	content := lipgloss.JoinVertical(lipgloss.Bottom, m.header.View(), m.displayModel.View(), m.footer.View())

	return frameStyle.Render(content)
}
