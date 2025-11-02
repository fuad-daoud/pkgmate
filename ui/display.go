package ui

import (
	"log/slog"
	"pkgmate/backend"

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

type displayKeyMap struct {
	tab    key.Binding
	update key.Binding
}

func (k displayKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.tab}
}

func (k displayKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.tab, k.update}}
}

type displayModel struct {
	keys         *displayKeyMap
	table        tableModel
	selected     SelectedDisplay
	focused      bool
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

	keys := displayKeyMap{
		tab: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "shift through tabs")),

		update: key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "run updates (root)")),
	}
	m := displayModel{
		keys:         &keys,
		table:        newTable(),
		selected:     TableDisplay,
		headerHeight: -1,
		headerWidth:  -1,
		footerHeight: -1,
		footerWidth:  -1,
		baseHeight:   -1,
		baseWidth:    -1,
	}

	return m
}

func (m displayModel) Update(msg tea.Msg) (displayModel, tea.Cmd) {
	commands := make([]tea.Cmd, 0)

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.baseHeight = msg.Height - 8 // 4 for the frame space and 2 for safe resize rendering 2 for the help menu
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

	case SearchFocusedEvent:
		m.Blur()
		m.keys.update.SetEnabled(false)
	case SearchBluredEvent, SearchResetedEvent:
		m.Focus()
		m.keys.update.SetEnabled(true)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.tab):
			commands = append(commands, m.newChangeTabEvent)

		case key.Matches(msg, m.keys.update):
			m.keys.update.SetEnabled(false)
			commands = append(commands, update)
			commands = append(commands, func() tea.Msg { return Updating })
		}
	case UpdateStatus:
		switch msg {
		case Updated, ErrUpdating:
			m.keys.update.SetEnabled(true)
		}

	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	commands = append(commands, cmd)
	return m, tea.Batch(commands...)
}

func (m *displayModel) Blur() {
	m.keys.tab.SetEnabled(false)
	m.focused = false
}

func (m *displayModel) Focus() {
	m.keys.tab.SetEnabled(true)
	m.focused = true
}

func (m displayModel) View() string {
	return m.table.View()
}

func update() tea.Msg {
	err := backend.Update()
	if err != nil {
		slog.Error("Failed updating database")
		return ErrUpdating
	}
	return Updated
}
