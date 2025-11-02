package ui

import (
	"log/slog"
	"os"

	"pkgmate/backend"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

type TableSummaryEvent struct {
	count int
}

type FetchDataEvent struct{}

type tableModel struct {
	tables      []customTable
	lastCursor  int
	activeTable int
	pkgStream   chan []backend.Package
}

func (m *tableModel) table() *customTable {
	return &m.tables[m.activeTable]
}

func (m tableModel) newSummaryEvent() tea.Msg {
	return TableSummaryEvent{
		count: len(m.table().Rows),
	}
}

type CursorChangedEvent struct {
	cursor int
}

func (m tableModel) newCursorChangedEvent() tea.Msg {
	return CursorChangedEvent{cursor: m.table().cursor + 1}
}

func newTable() tableModel {
	return tableModel{tables: []customTable{*newCustomTable(), *newCustomTable(), *newCustomTable()}}
}

type PackageStreamMsg struct {
	done bool
	pkgs []backend.Package
}

func (m tableModel) listen() tea.Msg {
	data, ok := <-m.pkgStream
	if !ok {
		slog.Info("channel is closed")
		return PackageStreamMsg{done: true}
	}
	return PackageStreamMsg{pkgs: data}
}

func (m tableModel) Update(msg tea.Msg) (tableModel, tea.Cmd) {
	var commands []tea.Cmd
	switch msg := msg.(type) {
	case ProgramInitEvent:
		columns := []string{
			"Name",
			"Version",
			"Size",
			"Installed",
		}
		for i := range m.tables {
			m.tables[i].Columns = columns
		}
		pkgChan, err := backend.LoadPackages()
		if err != nil {
			slog.Error("could not load packages", "err", err)
			os.Exit(1)
		}
		m.pkgStream = pkgChan
		commands = append(commands, m.listen)

	case DisplayResizeEvent:
		slog.Info("got display resize message", "msg", msg)
		for i := range m.tables {
			m.tables[i].Height = msg.height
			m.tables[i].Width = msg.width - 2
		}

	case SearchFocusedEvent:
		m.table().Blur()
	case SearchBluredEvent:
		m.table().Focus()
	case SearchResetedEvent:
		m.table().Reset()
		commands = append(commands, m.newSummaryEvent)

	case NewSearchTermEvent:
		m.table().filterColumn("Name", msg.term)
		commands = append(commands, m.newSummaryEvent)
	case PackageStreamMsg:
		pkgs := msg.pkgs
		for _, pkg := range pkgs {
			row := table.Row{pkg.Name, pkg.FormatVersion(), pkg.FormatSize(), pkg.Date.Format("2006-01-02")}
			if pkg.IsDirect {
				m.tables[0].addRow(row)
			} else {
				m.tables[1].addRow(row)
			}
			m.tables[2].addRow(row)
		}
		commands = append(commands, m.newSummaryEvent, m.newCursorChangedEvent)
		if !msg.done {
			commands = append(commands, m.listen)
		}

	case ChangeTabEvent:
		m.activeTable += 1
		m.activeTable %= len(m.tables)
		commands = append(commands, m.newSummaryEvent, m.newCursorChangedEvent)
	}
	var newCmd tea.Cmd

	m.tables[m.activeTable], newCmd = m.table().Update(msg)
	if m.lastCursor != m.table().cursor {
		m.lastCursor = m.table().cursor
		commands = append(commands, m.newCursorChangedEvent)
	}
	commands = append(commands, newCmd)
	return m, tea.Batch(commands...)
}

func (m tableModel) View() string {
	return m.table().View()
}
