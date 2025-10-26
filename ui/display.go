package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type SelectedDisplay int

const (
	TableDisplay SelectedDisplay = iota
)

type DisplayResizeEvent struct {
	height int
	width  int
}

type ChangeTabEvent struct{}

type displayModel struct {
	table        tableModel
	selected     SelectedDisplay
	headerHeight int
	headerWidth  int
	footerHeight int
	footerWidth  int
	baseHeight   int
	baseWidth    int
	height       int
	width        int
}

func (m displayModel) newDisplayResizeEvent() tea.Msg {
	return DisplayResizeEvent{
		width:  m.width,
		height: m.height,
	}
}

func (displayModel) newChangeTabEvent() tea.Msg {
	return ChangeTabEvent{}
}

func newDisplay() displayModel {
	return displayModel{
		table:        newTable(),
		selected:     TableDisplay,
		headerHeight: -1,
		headerWidth:  -1,
		footerHeight: -1,
		footerWidth:  -1,
		baseHeight:   -1,
		baseWidth:    -1,
	}
}

func (m displayModel) Update(msg tea.Msg) (displayModel, tea.Cmd) {
	commands := make([]tea.Cmd, 0)

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.baseHeight = msg.Height - 6 // 4 for the frame space and 2 for safe resize rendering
		m.baseWidth = msg.Width
		m.width = m.baseWidth

	case FooterResizeEvent:
		m.footerHeight = msg.height
		m.footerWidth = msg.width

		if m.headerHeight != -1 && m.headerWidth != -1 {
			m.height = m.baseHeight - m.footerHeight - m.headerHeight
			commands = append(commands, m.newDisplayResizeEvent)
		}

	case HeaderResizeEvent:
		m.headerHeight = msg.height
		m.headerWidth = msg.width

		if m.footerHeight != -1 && m.footerWidth != -1 {
			m.height = m.baseHeight - m.footerHeight - m.headerHeight
			commands = append(commands, m.newDisplayResizeEvent)
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
			if m.focused() {
				commands = append(commands, m.newChangeTabEvent)
			}
		}
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	commands = append(commands, cmd)
	return m, tea.Batch(commands...)
}

func (m displayModel) focused() bool {
	return m.table.table.Focused
}

func (m displayModel) View() string {
	return m.table.View()
}
