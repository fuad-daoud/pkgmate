package ui

import (
	"os"
	"pkgmate/backend"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	width          int
	height         int
	viewportHeight int
	viewportWidth  int
	header         headerModel
	display        displayModel
	footer         *footerModel
}

func InitialModel() model {
	return model{
		header:  newHeader(),
		display: newDisplay(),
		footer:  newFooter(),
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
		}
	}
	var footerCmd tea.Cmd
	m.footer, footerCmd = m.footer.Update(msg)
	commands = append(commands, footerCmd)

	var displayCmd tea.Cmd
	m.display, displayCmd = m.display.Update(msg)
	commands = append(commands, displayCmd)

	return m, tea.Batch(commands...)
}

func (m model) View() string {
	content := lipgloss.JoinVertical(lipgloss.Bottom, m.header.View(), m.display.View(), m.footer.View())

	content = frameStyle.Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}
