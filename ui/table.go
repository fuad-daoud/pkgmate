package ui

import (
	"log/slog"
	"os"
	"pkgmate/backend"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

type TableEvent struct {
	cursor  int
	event   TableEvents
	summary TableSummary
}
type TableSummary struct {
	count int
}

type TableEvents int

const (
	CursorChanged TableEvents = iota
	NewSummary
)

type tableModel struct {
	table         customTable
	rows          []table.Row
	newRows       []table.Row
	lastCursor    int
	event         TableEvent
	fetchers      []fetcher
	activeFetcher int
}

type fetcher func() ([]backend.Package, error)

func (m tableModel) newSummaryEvent() tea.Msg {
	return TableEvent{
		event: NewSummary,
		summary: TableSummary{
			count: len(m.table.Rows),
		},
	}
}
func (m tableModel) newCursorChangedEvent() tea.Msg {
	return TableEvent{event: CursorChanged, cursor: m.table.cursor + 1}
}

func newTable() tableModel {
	return tableModel{table: *newCustomTable(), fetchers: []fetcher{backend.LoadDirectPackages, backend.LoadDepedencyPackages, backend.LoadPackages}}
}

func (m tableModel) Update(msg tea.Msg) (tableModel, tea.Cmd) {
	var commands []tea.Cmd
	switch msg := msg.(type) {
	case ProgramInitEvent:
		pkgs, err := m.fetchers[m.activeFetcher]()
		if err != nil {
			os.Exit(1)
		}

		rows := [][]string{}
		for _, pkg := range pkgs {
			row := table.Row{pkg.Name, pkg.Version, pkg.FormatSize(), pkg.Date.Format("2006-01-02")}
			rows = append(rows, row)
		}
		m.table.updateRows(rows)
		commands = append(commands, m.newSummaryEvent, m.newCursorChangedEvent)

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
	case SearchFocusedEvent:
		m.table.Focused = false
	case SearchBluredEvent:
		m.table.Focused = true
	case SearchResetedEvent:
		m.table.Reset()
		commands = append(commands, m.newSummaryEvent)

	case NewSearchTermEvent:
		m.table.filterColumn("Name", msg.term)
		commands = append(commands, m.newSummaryEvent)

	case ChangeTabEvent:
		m.activeFetcher += 1
		m.activeFetcher %= len(m.fetchers)

		pkgs, err := m.fetchers[m.activeFetcher]()
		if err != nil {
			os.Exit(1)
		}

		rows := [][]string{}
		for _, pkg := range pkgs {
			row := table.Row{pkg.Name, pkg.Version, pkg.FormatSize(), pkg.Date.Format("2006-01-02")}
			rows = append(rows, row)
		}
		m.table.updateRows(rows)
		commands = append(commands, m.newSummaryEvent, m.newCursorChangedEvent)
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
