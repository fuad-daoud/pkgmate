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
	mode      AppMode
	version   string
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

var Version = "dev"

func newHeader(mode AppMode) headerModel {
	return headerModel{mode: mode, version: "Pkgmate " + Version, tabs: []string{"Direct Packages", "Dependency Packages", "All Packages"}}
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
	version := topRightTab.Render(m.version)
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
	var modeIndicator string
	modeTab := topRightTab.PaddingRight(1).PaddingLeft(1).Bold(true).Italic(true)
	if m.mode == ModePrivileged {
		modeStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AA000B")).
			Bold(true)
		modeIndicator = modeTab.Render(modeStyle.Render("⚠️ PRIVILEGED"))
	} else {
		modeStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFB86C")).
			Bold(true)
		modeIndicator = modeTab.Render(modeStyle.Render("◆ Normal"))
	}

	leftSection := lipgloss.JoinHorizontal(lipgloss.Bottom, tabs...)
	rightSection := lipgloss.JoinHorizontal(lipgloss.Bottom, modeIndicator, version)
	spacer := spaceStyle.Width(m.width - lipgloss.Width(leftSection) - lipgloss.Width(rightSection)).Render()
	return lipgloss.JoinHorizontal(lipgloss.Center, leftSection, spacer, rightSection)
}
