package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Column struct {
	Title string
}

type customTable struct {
	Columns       []string
	OriginalRows  [][]string
	Rows          [][]string
	NewRows       [][]string
	cursor        int
	offset        int
	Height        int
	Width         int
	Focused       bool
	selectedRow   int
	headerStyle   lipgloss.Style
	selectedStyle lipgloss.Style
	cellStyle     lipgloss.Style
}

func newCustomTable() *customTable {
	return &customTable{
		Columns:       []string{},
		Rows:          [][]string{},
		NewRows:       [][]string{},
		cursor:        0,
		offset:        0,
		Focused:       true,
		headerStyle:   lipgloss.NewStyle().Bold(true).Padding(0, 1).BorderStyle(lipgloss.NormalBorder()).BorderBottom(true),
		selectedStyle: lipgloss.NewStyle().Bold(false).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#4355ff")).Padding(0, 1),
		cellStyle:     lipgloss.NewStyle().Padding(0, 1),
	}
}

func (m customTable) Update(msg tea.Msg) (customTable, tea.Cmd) {
	if !m.Focused {
		return m, nil
	}

	if len(m.Rows) == 0 {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			m.updateCursor(-1)
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			m.updateCursor(1)
		case key.Matches(msg, key.NewBinding(key.WithKeys("pgup", "ctrl+u"))):
			m.updateCursor(10 - m.Height)
		case key.Matches(msg, key.NewBinding(key.WithKeys("pgdown", "ctrl+d"))):
			m.updateCursor(m.Height - 10)
		case key.Matches(msg, key.NewBinding(key.WithKeys("home", "g"))):
			m.updateCursor(0)
		case key.Matches(msg, key.NewBinding(key.WithKeys("end", "G"))):
			m.cursor = len(m.Rows) - 1
		}
	}

	return m, nil
}

func (m *customTable) updateCursor(n int) {
	m.cursor = max(0, min(m.cursor+n, len(m.Rows)))
	m.adjustOffset()
}

func (m *customTable) Reset() {
	m.cursor = 0
	m.Focused = true
	m.Rows = m.OriginalRows

}
func (t *customTable) adjustOffset() {
	visibleRows := t.Height - 1 // -1 for header
	if visibleRows <= 0 {
		return
	}

	if t.cursor < t.offset {
		t.offset = t.cursor
	} else if t.cursor >= t.offset+visibleRows {
		t.offset = t.cursor - visibleRows + 1
	}

	maxOffset := max(0, len(t.Rows)-visibleRows)
	if t.offset > maxOffset {
		t.offset = maxOffset
	}
}

// func (t *customTable) recalculateColumnWidths() {
// if len(t.Columns) == 0 || t.Width == 0 {
// 	return
// }

// totalBorder := len(t.Columns) * 2 // padding on each side
// availableWidth := t.Width - totalBorder

// equalWidth := availableWidth / len(t.Columns)
// for i := range t.Columns {
// t.Columns[i].width = equalWidth
// }
// }

func (m *customTable) View() string {
	if len(m.Columns) == 0 {
		return ""
	}

	var b strings.Builder

	headerCells := make([]string, len(m.Columns))
	for i, col := range m.Columns {
		content := truncate(col, m.Width/len(m.Columns))
		headerCells[i] = m.headerStyle.Width(m.Width / len(m.Columns)).Render(content)
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, headerCells...))
	b.WriteString("\n")

	visibleRows := m.Height - 1
	endIdx := min(m.offset+visibleRows, len(m.Rows))

	for i := m.offset; i < endIdx; i++ {
		row := m.Rows[i]
		rowCells := make([]string, len(m.Columns))

		for j := range m.Columns {
			cellContent := ""
			if j < len(row) {
				cellContent = truncate(row[j], m.Width/len(m.Columns))
			}

			style := m.cellStyle
			if i == m.cursor && m.Focused {
				style = m.selectedStyle
			}

			rowCells[j] = style.Width(m.Width / len(m.Columns)).Render(cellContent)
		}

		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, rowCells...))
		if i < endIdx-1 {
			b.WriteString("\n")
		}
	}
	currentHeight := strings.Count(b.String(), "\n")
	for currentHeight < m.Height {
		b.WriteString("\n")
		currentHeight++
	}

	return b.String()
}

func (m *customTable) filterColumn(column, term string) {

	index := 0
	columnIndex := 0
	for m.Columns[columnIndex] != column {
		columnIndex++
	}
	for _, row := range m.OriginalRows {
		if !strings.Contains(strings.ToLower(row[columnIndex]), term) {
			continue
		}
		m.NewRows[index] = row
		index++
	}

	m.cursor = 0
	m.adjustOffset()
	m.Rows = m.NewRows[0:index]
}

func truncate(s string, width int) string {
	if len(s) <= width-2 {
		return s
	}
	return s[:max(1, width-5)] + "..."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
