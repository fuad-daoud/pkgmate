package ui

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#FFFFFF")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#4355ff")).
		Bold(false)
	m.table.SetStyles(s)
	bordredTable := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Render(m.table.View())


	count := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Render(strconv.Itoa(len(m.packages)))


	content := bordredTable + "\n" + count

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
