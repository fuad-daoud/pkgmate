package ui

import (
	"log/slog"
	"pkgmate/backend"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

type TableEvent struct {
	cursor  int
	event   TableEvents
	summary TableSummery
}
type TableSummery struct {
	count int
}

type TableEvents int

const (
	CursorChanged TableEvents = iota
	NewSummery
)

type tableModel struct {
	table      customTable
	rows       []table.Row
	newRows    []table.Row
	lastCursor int
	event      TableEvent
}

func (m tableModel) newSummeryEvent() tea.Msg {
	return TableEvent{
		event: NewSummery,
		summary: TableSummery{
			count: len(m.table.Rows),
		},
	}
}
func (m tableModel) newCursorChangedEvent() tea.Msg {
	return TableEvent{event: CursorChanged, cursor: m.table.cursor}
}

func newTable() tableModel {
	return tableModel{table: *newCustomTable()}
}

func (m tableModel) Update(msg tea.Msg) (tableModel, tea.Cmd) {
	var commands []tea.Cmd
	switch msg := msg.(type) {
	case DisplayResizeEvent:
		slog.Info("got window resize message", "msg", msg)
		m.table.Height = msg.height
		m.table.Width = msg.width - 2

		columns := []string{
			"Name",
			"Version",
			"Size",
			"Installed",
		}

		m.table.Columns = columns

	case []backend.Package:
		rows := [][]string{}
		for _, pkg := range msg {
			row := table.Row{pkg.Name, pkg.Version, pkg.FormatSize(), pkg.Date.Format("2006-01-02")}
			rows = append(rows, row)
		}
		m.table.Rows = rows
		m.table.OriginalRows = rows
		m.table.NewRows = make([][]string, len(rows))
		commands = append(commands, m.newSummeryEvent)

	case SearchFocusedEvent:
		m.table.Focused = false
	case SearchBluredEvent:
		m.table.Focused = true
	case SearchResetedEvent:
		m.table.Reset()

	case NewSearchTermEvent:
		m.table.filterColumn("Name", msg.term)
		commands = append(commands, m.newSummeryEvent)
	}
	var newCmd tea.Cmd

	m.table, newCmd = m.table.Update(msg)
	if m.lastCursor != m.table.cursor {
		m.lastCursor = m.table.cursor
		commands = append(commands, m.newCursorChangedEvent)
	}
	commands = append(commands, newCmd)
	return m, tea.Batch(commands...)
}

func (m tableModel) View() string {
	return m.table.View()
}
