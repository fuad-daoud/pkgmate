package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type HeaderResizeEvent struct {
	height int
	width  int
}

type headerModel struct {
	info string
}

func (m headerModel) newHeaderResizeEvent() tea.Msg {
	v := m.View()
	return HeaderResizeEvent{
		width:  lipgloss.Width(v),
		height: lipgloss.Height(v),
	}
}

func newHeader() headerModel {
	return headerModel{info: "Pkgmate v0.0.0"}
}

func (m headerModel) Update(msg tea.Msg) (headerModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg.(type) {
	case tea.WindowSizeMsg:
		cmd = m.newHeaderResizeEvent
	}
	return m, cmd
}

func (m headerModel) View() string {
	info := topTab.Render(m.info)

	return lipgloss.JoinHorizontal(lipgloss.Center, info)
}
