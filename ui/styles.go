package ui

import "github.com/charmbracelet/lipgloss"

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(lipgloss.Color("#383838"))

	searchStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF79C6"))

	columnHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#BD93F9")).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderTop(true).
				BorderForeground(lipgloss.Color("#383838"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F8F8F2"))

	versionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#50FA7B"))

	sizeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFB86C"))

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4")).
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(lipgloss.Color("#383838"))
)
