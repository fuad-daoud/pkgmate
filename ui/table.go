package ui

import (
	"pkgmate/backend"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

type TableEvents int

type TableEvent struct {
	cursor  int
	event   TableEvents
	summary TableSummery
}
type TableSummery struct {
	count int
}

const (
	CursorChanged TableEvents = iota
	NewSummery
)

type tableModel struct {
	table      table.Model
	height     int
	width      int
	rows       []table.Row
	newRows    []table.Row
	lastCursor int
	event      TableEvent
}

func (m tableModel) newSummeryEvent() tea.Msg {
	return TableEvent{
		event: NewSummery,
		summary: TableSummery{
			count: len(m.table.Rows()),
		},
	}
}
func (m tableModel) newCursorChangedEvent() tea.Msg {
	return TableEvent{event: CursorChanged, cursor: m.table.Cursor()}
}

func newTable() tableModel {
	t := table.New()
	t.KeyMap = table.DefaultKeyMap()
	t.KeyMap.PageUp = key.NewBinding(
		key.WithKeys("pgup", "ctrl+u"),
		key.WithHelp("ctrl+u", "page up"),
	)
	t.KeyMap.PageDown = key.NewBinding(
		key.WithKeys("pgdown", "ctrl+d"),
		key.WithHelp("ctrl+d", "page down"),
	)
	t.KeyMap.HalfPageUp = t.KeyMap.PageUp
	t.KeyMap.HalfPageDown = t.KeyMap.PageDown
	t.Focus()

	t.SetStyles(tableStyles)
	return tableModel{table: t}
}

func (m tableModel) update(msg tea.Msg) (tableModel, tea.Cmd) {
	var commands []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height - (msg.Height / 5)
		m.width = msg.Width - (msg.Width / 5)

		m.table.SetHeight(m.height)
		const column_count = 3
		columnWidth := m.width / column_count

		columns := []table.Column{
			{Title: "Name", Width: columnWidth},
			{Title: "Version", Width: columnWidth},
			{Title: "Size", Width: columnWidth},
		}

		m.table.SetColumns(columns)

	case []backend.Package:
		if len(msg) == 0 {
			break
		}
		rows := []table.Row{}
		for _, pkg := range msg {
			row := table.Row{pkg.Name, pkg.Version, formatSize(pkg.Size)}
			rows = append(rows, row)
		}
		m.rows = rows
		m.newRows = make([]table.Row, len(rows))
		m.table.SetRows(rows)
		commands = append(commands, m.newSummeryEvent)

	case FooterEvent:
		switch msg.event {
		case SearchFocused:
			m.table.Blur()
		case SearchBlured:
			m.table.Focus()
		case SearchReseted:
			m.table.Blur()
			m.table.SetCursor(0)
			m.table.SetRows(m.rows)
			m.table.Focus()

		case NewSearchTerm:
			m.table.SetCursor(0)
			m.filterPackages(msg.term)
			commands = append(commands, m.newSummeryEvent)
		}
	}
	var newCmd tea.Cmd

	m.table, newCmd = m.table.Update(msg)
	if m.lastCursor != m.table.Cursor() {
		m.lastCursor = m.table.Cursor()
		commands = append(commands, m.newCursorChangedEvent)
	}
	commands = append(commands, newCmd)
	return m, tea.Batch(commands...)
}

func (m *tableModel) filterPackages(term string) {
	index := 0
	for _, row := range m.rows {
		if !strings.Contains(strings.ToLower(row[0]), term) {
			continue
		}
		m.newRows[index] = row
		index++
	}

	m.table.SetRows(m.newRows[0:index])
}

func (m tableModel) View() string {
	return m.table.View()
}
