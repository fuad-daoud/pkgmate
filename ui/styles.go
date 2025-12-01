package ui

import (
	"github.com/charmbracelet/lipgloss"
)

type Styler func(lipgloss.Style) lipgloss.Style

var (
	selectedColor = lipgloss.Color("#4355ff")
	dangerColor   = lipgloss.Color("#AA220B")
	loadingColor  = lipgloss.Color("#FFB86C")
	allGoodColor  = lipgloss.Color("#43AA22")
	mutedColor  = lipgloss.Color("#6C7086")

	noStyler = func(s lipgloss.Style) lipgloss.Style {
		return s
	}

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
	dangerRow  = func(style lipgloss.Style) lipgloss.Style {
		return style.Background(dangerColor).
			Bold(true)
	}
	cursorRowStyler = func(s lipgloss.Style) lipgloss.Style {
		return s.Bold(false).Foreground(lipgloss.Color("#FFFFFF")).Background(selectedColor)
	}

	cursorAndDangerRow = func(s lipgloss.Style) lipgloss.Style {
		return s.
			Background(lipgloss.Color("#BB4488")).
			Bold(true)
	}
	frozenRowStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1A1F2E")).
			Foreground(lipgloss.Color("#7B8FD3")).
			Italic(true)
	updateAvailableRow = lipgloss.NewStyle().
				Bold(true).
				Foreground(loadingColor)
)
