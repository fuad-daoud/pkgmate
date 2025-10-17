package ui

import "github.com/charmbracelet/lipgloss"

type headerModel struct {
	info string
}

func newHeader() headerModel {
	return headerModel{info: "Pkgmate v0.0.0"}
}

func (m headerModel) View() string {
	info := topTab.Render(m.info)

	return lipgloss.JoinHorizontal(lipgloss.Center, info)
}
