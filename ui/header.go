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
	info      string
	width     int
	tabs      []string
	activeTab int
}

func (m headerModel) newHeaderResizeEvent() tea.Msg {
	v := m.View()
	return HeaderResizeEvent{
		width:  lipgloss.Width(v),
		height: lipgloss.Height(v),
	}
}

func newHeader() headerModel {
	return headerModel{info: "Pkgmate v0.0.0", tabs: []string{"Direct Packages", "Dependency Packages", "All Packages"}}
}

func (m headerModel) Update(msg tea.Msg) (headerModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - 2
		cmd = m.newHeaderResizeEvent
	case ChangeTabEvent:
		m.activeTab += 1
		m.activeTab %= len(m.tabs)
	}
	return m, cmd
}

func (m headerModel) View() string {
	info := topRightTab.Render(m.info)
	tabs := make([]string, len(m.tabs))
	for i, v := range m.tabs {
		if i == m.activeTab {
			tab := topLeftTab.Bold(true).
				BorderStyle(lipgloss.ThickBorder()).
				BorderForeground(lipgloss.Color("#4355ff")).
				Render(v)
			tabs = append(tabs, tab)
			continue
		}
		tab := topLeftTab.Render(v)
		tabs = append(tabs, tab)

	}

	leftSection := lipgloss.JoinHorizontal(lipgloss.Bottom, tabs...)
	spacer := spaceStyle.Width(m.width - lipgloss.Width(leftSection) - lipgloss.Width(info)).Render()
	return lipgloss.JoinHorizontal(lipgloss.Center, leftSection, spacer, info)
}
