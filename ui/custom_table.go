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

type customTableKeyMap struct {
	up       key.Binding
	down     key.Binding
	pageup   key.Binding
	pagedown key.Binding
	first    key.Binding
	last     key.Binding
}

func (k customTableKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.up, k.down}
}

func (k customTableKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.up, k.down, k.pageup, k.pagedown}, {k.first, k.last}}
}

type customTable struct {
	keys          *customTableKeyMap
	Columns       []string
	mp            map[string]int
	OriginalRows  [][]string
	StyledRows    map[string]lipgloss.Style
	Rows          [][]string
	NewRows       [][]string
	cursor        int
	offset        int
	Height        int
	Width         int
	focused       bool
	headerStyle   lipgloss.Style
	selectedStyle lipgloss.Style
	cellStyle     lipgloss.Style
}

func newCustomTable() *customTable {
	keys := customTableKeyMap{
		up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "move up")),
		down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "move down")),
		pageup:   key.NewBinding(key.WithKeys("pgup", "ctrl+u"), key.WithHelp("pageup/ctrl+u", "jump up")),
		pagedown: key.NewBinding(key.WithKeys("pgdown", "ctrl+d"), key.WithHelp("pagedown/ctrl+d", "jump down")),
		first:    key.NewBinding(key.WithKeys("home", "ctrl+g"), key.WithHelp("home/ctrl+g", "move to first")),
		last:     key.NewBinding(key.WithKeys("end", "G"), key.WithHelp("end/G", "move to last")),
	}
	return &customTable{
		keys:          &keys,
		mp:            make(map[string]int),
		Columns:       []string{},
		Rows:          [][]string{},
		NewRows:       [][]string{},
		cursor:        0,
		offset:        0,
		focused:       true,
		headerStyle:   lipgloss.NewStyle().Bold(true).Padding(0, 1).BorderStyle(lipgloss.NormalBorder()).BorderBottom(true),
		selectedStyle: lipgloss.NewStyle().Bold(false).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#4355ff")).Padding(0, 1),
		cellStyle:     lipgloss.NewStyle().Padding(0, 1),
		StyledRows:    map[string]lipgloss.Style{},
	}
}

func (m *customTable) Blur() {
	m.focused = false
	m.keys.up.SetEnabled(false)
	m.keys.down.SetEnabled(false)
	m.keys.pageup.SetEnabled(false)
	m.keys.pagedown.SetEnabled(false)
	m.keys.first.SetEnabled(false)
	m.keys.last.SetEnabled(false)
}

func (m *customTable) Focus() {
	m.focused = true
	m.keys.up.SetEnabled(true)
	m.keys.down.SetEnabled(true)
	m.keys.pageup.SetEnabled(true)
	m.keys.pagedown.SetEnabled(true)
	m.keys.first.SetEnabled(true)
	m.keys.last.SetEnabled(true)
}

func (m customTable) Focused() bool {
	return m.focused
}

func (m customTable) Update(msg tea.Msg) (customTable, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	if len(m.Rows) == 0 {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.up):
			m.updateCursor(-1)
		case key.Matches(msg, m.keys.down):
			m.updateCursor(1)
		case key.Matches(msg, m.keys.pageup):
			m.updateCursor(10 - m.Height)
		case key.Matches(msg, m.keys.pagedown):
			m.updateCursor(m.Height - 10)
		case key.Matches(msg, m.keys.first):
			m.updateCursor(len(m.Rows) * -1)
		case key.Matches(msg, m.keys.last):
			m.updateCursor(len(m.Rows))
		}
	}

	return m, nil
}

func (m *customTable) updateCursor(n int) {
	m.cursor = max(0, min(m.cursor+n, len(m.Rows)-1))
	m.adjustOffset()
}

func (m *customTable) Reset() {
	m.cursor = 0
	m.Focus()
	m.Rows = m.OriginalRows
}

func (m *customTable) adjustOffset() {
	visibleRows := m.Height - 1 // -1 for header
	if visibleRows <= 0 {
		return
	}

	if m.cursor < m.offset {
		m.offset = m.cursor
	} else if m.cursor >= m.offset+visibleRows {
		m.offset = m.cursor - visibleRows + 1
	}

	maxOffset := max(0, len(m.Rows)-visibleRows)
	if m.offset > maxOffset {
		m.offset = maxOffset
	}
}

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
			if i == m.cursor && m.Focused() {
				style = m.selectedStyle
			}

			rowCells[j] = style.Width(m.Width / len(m.Columns)).Render(cellContent)
		}

		curruentRow := lipgloss.JoinHorizontal(lipgloss.Top, rowCells...)
		rowStyle, ok := m.StyledRows[row[0]]
		if !ok {
			rowStyle = lipgloss.NewStyle()
		}
		b.WriteString(rowStyle.Render(curruentRow))
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

func (m *customTable) addRow(row []string) {
	if index, ok := m.mp[row[0]]; ok {
		m.OriginalRows[index] = row
		return
	}
	m.OriginalRows = append(m.OriginalRows, row)
	m.mp[row[0]] = len(m.OriginalRows) - 1
	m.Rows = m.OriginalRows
	m.NewRows = make([][]string, len(m.OriginalRows))
	m.adjustOffset()
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
