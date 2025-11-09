package ui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type HeaderResizeEvent struct {
	height int
	width  int
}

type UpdateStatus int

const (
	Updating UpdateStatus = iota
	IDLE
	ErrUpdating
	Updated
)

type headerModel struct {
	version      string
	width        int
	tabs         []string
	activeTab    int
	updateStatus UpdateStatus
	spin         spinner.Model
}

func (m headerModel) newHeaderResizeEvent() tea.Msg {
	v := m.View()
	return HeaderResizeEvent{
		width:  lipgloss.Width(v),
		height: lipgloss.Height(v),
	}
}

func newHeader() headerModel {
	spin := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	return headerModel{spin: spin, updateStatus: IDLE, version: "Pkgmate " + Version, tabs: []string{"Direct Packages", "Dependency Packages", "All Packages"}}
}

func (m headerModel) Update(msg tea.Msg) (headerModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - 2
		cmd = m.newHeaderResizeEvent

	case UpdateStatus:
		switch msg {
		case Updating:
			m.updateStatus = Updating
			cmd = m.spin.Tick
		case Updated, ErrUpdating:
			m.updateStatus = msg
		}

	case spinner.TickMsg:
		m.spin, cmd = m.spin.Update(msg)
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
				BorderForeground(selectedColor).
				Render(v)
			tabs = append(tabs, tab)
			continue
		}
		tab := topLeftTab.Render(v)
		tabs = append(tabs, tab)

	}
	var modeIndicator string
	var updateIndicator string
	updateIndicatorTab := topRightTab.
		BorderStyle(lipgloss.ThickBorder())

	switch m.updateStatus {
	case Updating:
		updateIndicator = updateIndicatorTab.BorderForeground(loadingColor).
			Render(m.spin.View() + " Updating")
	case Updated:
		updateIndicator = updateIndicatorTab.BorderForeground(allGoodColor).
			Render("✅ Updated")
	case ErrUpdating:
		updateIndicator = updateIndicatorTab.BorderForeground(dangerColor).
			Render("❌ Failed updating")
	}

	leftSection := lipgloss.JoinHorizontal(lipgloss.Bottom, tabs...)
	rightSection := lipgloss.JoinHorizontal(lipgloss.Bottom, updateIndicator, modeIndicator, version)
	spacer := spaceStyle.Width(m.width - lipgloss.Width(leftSection) - lipgloss.Width(rightSection)).Render()
	return lipgloss.JoinHorizontal(lipgloss.Center, leftSection, spacer, rightSection)
}
