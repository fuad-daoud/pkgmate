package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	selectedColor = lipgloss.Color("#4355ff")
	dangerColor   = lipgloss.Color("#AA000B")
	loadingColor  = lipgloss.Color("#FFB86C")
	allGoodColor  = lipgloss.Color("#43AA22")

	bottomRightTab = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder(), false, false, true, true).
			MarginTop(1)

	bottomLeftTab = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder(), false, true, true, false).
			MarginTop(1)

	topRightTab = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder(), false, false, true, true).
			PaddingLeft(1).
			PaddingRight(1).
			MarginBottom(0)

	topLeftTab = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder(), false, true, true, false).
			MarginBottom(0)
	frameStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
	spaceStyle = lipgloss.NewStyle()
	frozenRow  = lipgloss.NewStyle().
			Background(lipgloss.Color("#1A1F2E")).
			Foreground(lipgloss.Color("#7B8FD3")).
			Italic(true)
	updateAvailableRow = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFB86C"))
)
