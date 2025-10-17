package ui

import (
	"fmt"
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
	table          tableModel
	footer         *footerModel
	info string
}

func InitialModel() model {
	return model{
		table:  newTable(),
		footer: newFooter(),
		info: "Pkgmate v0.0.0",
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

	var tableCmd tea.Cmd
	m.table, tableCmd = m.table.update(msg)

	return m, tea.Batch(tableCmd, footerCmd)
}

func (m model) View() string {
	info := topTab.Render(m.info)
	header := lipgloss.JoinHorizontal(lipgloss.Center, info)

	content := lipgloss.JoinVertical(lipgloss.Bottom, header, m.table.View(), m.footer.View())

	content = frameStyle.Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
