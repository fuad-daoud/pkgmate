package ui

import tea "github.com/charmbracelet/bubbletea"

type SelectedDisplay int

const (
	TableDisplay SelectedDisplay = iota
)

type displayModel struct {
	table    tableModel
	selected SelectedDisplay
}

func newDisplay() displayModel {
	return displayModel{
		table:    newTable(),
		selected: TableDisplay,
	}
}

func (m displayModel) Update(msg tea.Msg) (displayModel, tea.Cmd) {
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m displayModel) View() string {
	return m.table.View()
}
