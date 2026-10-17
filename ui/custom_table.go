package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Column struct {
	Title string
}

// defaultEmptyState is the text shown for a table with no rows after loading.
const defaultEmptyState = "🔍 No results found"

type Row struct {
	Key   string
	Cells []string
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
	label        string
	keys         *customTableKeyMap
	Columns      []string
	mp           map[string]int
	OriginalRows []Row
	StyledRows   map[string]lipgloss.Style
	Rows         []Row
	NewRows      []Row
	cursor       int
	offset       int
	Height       int
	Width        int
	focused      bool
	loading      bool
	spinner      spinner.Model
	headerStyle  lipgloss.Style
	cursorStyler Styler
	cellStyle    lipgloss.Style
	// emptyState is shown by View when the table holds no rows at all after
	// loading. It defaults to the generic no-results text.
	emptyState string
}

func newCustomTable(label string) *customTable {
	keys := customTableKeyMap{
		up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "move up")),
		down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "move down")),
		pageup:   key.NewBinding(key.WithKeys("pgup", "ctrl+u"), key.WithHelp("pageup/ctrl+u", "jump up")),
		pagedown: key.NewBinding(key.WithKeys("pgdown", "ctrl+d"), key.WithHelp("pagedown/ctrl+d", "jump down")),
		first:    key.NewBinding(key.WithKeys("home", "ctrl+g"), key.WithHelp("home/ctrl+g", "move to first")),
		last:     key.NewBinding(key.WithKeys("end", "G"), key.WithHelp("end/G", "move to last")),
	}

	s := spinner.New(spinner.WithSpinner(spinner.Dot))

	return &customTable{
		label:        label,
		keys:         &keys,
		mp:           make(map[string]int),
		Columns:      []string{},
		OriginalRows: []Row{},
		Rows:         []Row{},
		NewRows:      []Row{},
		cursor:       0,
		offset:       0,
		focused:      true,
		loading:      true,
		spinner:      s,
		headerStyle:  lipgloss.NewStyle().Bold(true).Padding(0, 1).BorderStyle(lipgloss.NormalBorder()).BorderBottom(true),
		cursorStyler: cursorRowStyler,
		cellStyle:    lipgloss.NewStyle().Padding(0, 1),
		StyledRows:   make(map[string]lipgloss.Style),
		emptyState:   defaultEmptyState,
	}
}

func (m *customTable) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *customTable) SetLoading(loading bool) {
	m.loading = loading
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
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			if m.loading {
				return m, cmd
			}
		}
		return m, nil
	}
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if m.loading {
			cmds = append(cmds, cmd)
		}

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

	return m, tea.Batch(cmds...)
}

func (m *customTable) addStyleRow(key string, style lipgloss.Style) {
	m.StyledRows[key] = style
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

func (m *customTable) columnWidth() int {
	if len(m.Columns) == 0 {
		return 0
	}
	return max(0, m.Width/len(m.Columns))
}

func (m *customTable) View() string {
	if len(m.Columns) == 0 {
		return ""
	}

	if len(m.Rows) == 0 {
		if m.loading {
			// Show spinner while loading
			loadingStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888"))

			content := lipgloss.JoinHorizontal(
				lipgloss.Left,
				m.spinner.View(),
				loadingStyle.Render(" Loading packages..."),
			)

			return lipgloss.Place(
				m.Width,
				m.Height+1,
				lipgloss.Center,
				lipgloss.Center,
				content,
			)
		}
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true)

		// A table that never held a row shows its own empty state; a table whose
		// rows were all filtered away shows the generic no-results text.
		text := defaultEmptyState
		if len(m.OriginalRows) == 0 && m.emptyState != "" {
			text = m.emptyState
		}

		content := emptyStyle.Render(text)

		emptyView := lipgloss.Place(
			m.Width,
			m.Height+1,
			lipgloss.Center,
			lipgloss.Center,
			content,
		)

		return emptyView
	}

	var b strings.Builder

	colWidth := m.columnWidth()
	headerCells := make([]string, len(m.Columns))
	for i, col := range m.Columns {
		content := truncate(col, colWidth)
		headerCells[i] = m.headerStyle.Width(colWidth).Render(content)
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
			if j < len(row.Cells) {
				cellContent = truncate(row.Cells[j], colWidth)
			}

			rowCells[j] = lipgloss.NewStyle().Width(colWidth).Render(cellContent)
		}

		currentRow := lipgloss.JoinHorizontal(lipgloss.Top, rowCells...)
		rowStyle, ok := m.StyledRows[row.Key]
		if !ok {
			rowStyle = lipgloss.NewStyle()
		}
		if i == m.cursor && m.Focused() {
			rowStyle = m.cursorStyler(rowStyle)
		}

		b.WriteString(rowStyle.Render(currentRow))
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

func (m *customTable) addRow(key string, cells []string) {
	if index, ok := m.mp[key]; ok {
		m.OriginalRows[index] = Row{Key: key, Cells: cells}
		for i := range m.Rows {
			if m.Rows[i].Key == key {
				m.Rows[i] = Row{Key: key, Cells: cells}
				break
			}
		}
		return
	}
	m.OriginalRows = append(m.OriginalRows, Row{Key: key, Cells: cells})
	m.mp[key] = len(m.OriginalRows) - 1
	m.Rows = m.OriginalRows
	m.NewRows = make([]Row, len(m.OriginalRows))
	m.adjustOffset()
}

func (m *customTable) filterColumn(column, term string) {
	if len(m.Columns) == 0 {
		return
	}
	columnIndex := -1
	for i, col := range m.Columns {
		if col == column {
			columnIndex = i
			break
		}
	}
	if columnIndex == -1 {
		return
	}

	index := 0
	for _, row := range m.OriginalRows {
		if columnIndex >= len(row.Cells) {
			continue
		}
		if !strings.Contains(strings.ToLower(row.Cells[columnIndex]), term) {
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
	if s == "" {
		return ""
	}
	if len(s) <= width-2 {
		return s
	}
	return s[:max(1, width-5)] + "..."
}
