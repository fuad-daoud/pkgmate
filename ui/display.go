package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type SelectedDisplay int

const (
	TableDisplay SelectedDisplay = iota
	// UpdateDisplay
)

type DisplayResizeEvent struct {
	height int
	width  int
}

type displayModel struct {
	table       tableModel
	// updateModel *updateViewModel
	selected    SelectedDisplay
	height      int
	width       int
}

func (m displayModel) newDisplayResizeEvent() tea.Msg {
	return DisplayResizeEvent{
		width:  m.width,
		height: m.height,
	}
}

func newDisplay() displayModel {
	m := displayModel{
		table:       newTable(),
		// updateModel: newUpdateView(),
		selected:    TableDisplay,
	}

	return m
}

func (m displayModel) Update(msg tea.Msg) (displayModel, tea.Cmd) {
	commands := make([]tea.Cmd, 0)

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.height = msg.Height - 6 // 4 for the frame space 2 for better visual space
		m.width = msg.Width - 2
		commands = append(commands, m.newDisplayResizeEvent)

	case SelectedDisplay:
		m.selected = msg
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	commands = append(commands, cmd)

	// var updateViewCmd tea.Cmd
	// m.updateModel, updateViewCmd = m.updateModel.Update(msg)
	// commands = append(commands, updateViewCmd)

	return m, tea.Batch(commands...)
}

func (m displayModel) View() string {
	// if m.selected == UpdateDisplay {
	// 	return m.updateModel.View()
	// }
	return m.table.View()
}
