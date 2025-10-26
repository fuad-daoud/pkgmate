package ui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

var (
	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F8F8F2"))

	versionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#50FA7B"))

	sizeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFB86C"))

	tableStyles = table.Styles{
		Header: lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#FFFFFF")).
			BorderBottom(true).
			MarginTop(1).
			Bold(false),
		Selected: lipgloss.NewStyle().
			Bold(false).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#4355ff")),
		Cell: lipgloss.NewStyle().Padding(0, 1),
	}

	bottomRightTab = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder(), false, false, true, true).
			MarginTop(1)

	bottomLeftTab = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder(), false, true, true, false).
			MarginTop(1)

	topRightTab = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder(), false, false, true, true).
			MarginBottom(0)

	topLeftTab = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder(), false, true, true, false).
			MarginBottom(0)
	frameStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
	spaceStyle = lipgloss.NewStyle()
)
