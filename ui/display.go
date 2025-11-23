package ui

import (
	"reflect"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type DisplayResizeEvent struct {
	height int
	width  int
}

type ChangeDisplay struct {
	display tea.Model
}

func NewChangeDisplay(m tea.Model) tea.Cmd {
	return func() tea.Msg { return ChangeDisplay{display: m} }

}

type displayModel struct {
	currentDisplay tea.Model
	commandPalette commandPaletteModel
	height         int
	width          int
}

func (m displayModel) newDisplayResizeEvent() tea.Msg {
	return DisplayResizeEvent{
		width:  m.width,
		height: m.height,
	}
}

func newDisplay() displayModel {
	m := displayModel{
		currentDisplay: newTable(),
		commandPalette: newCommandPalette(),
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

	case ProgramInitEvent:
		cmd := m.currentDisplay.Init()
		commands = append(commands, cmd)

	case ChangeDisplay:
		m.currentDisplay = msg.display
		cmd := m.currentDisplay.Init()
		commands = append(commands, cmd)
		commands = append(commands, m.newDisplayResizeEvent)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+k", "ctrl+p"))):
			m.commandPalette = m.commandPalette.Toggle()
			commands = append(commands, m.newDisplayResizeEvent)
		}
	}

	if !m.commandPalette.visible {
		var cmd tea.Cmd
		m.currentDisplay, cmd = m.currentDisplay.Update(msg)
		commands = append(commands, cmd)
	} else {
		var cmd tea.Cmd
		m.commandPalette, cmd = m.commandPalette.Update(msg)
		commands = append(commands, cmd)
	}
	if reflect.TypeOf(msg).Name() == "DisplayResizeEvent" {
		var cmd tea.Cmd
		m.currentDisplay, cmd = m.currentDisplay.Update(msg)
		commands = append(commands, cmd)

		m.commandPalette, cmd = m.commandPalette.Update(msg)
		commands = append(commands, cmd)
	}

	return m, tea.Batch(commands...)
}

func (m displayModel) View() string {
	content := m.currentDisplay.View()

	if m.commandPalette.visible {
		content = PlaceOverlay(content, m.commandPalette.View())
	}
	return content
}
