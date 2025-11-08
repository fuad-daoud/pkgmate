package ui

import (
	"io"
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
	tab key.Binding
}

func (k displayKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.tab}
}

func (k displayKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.tab}}
}

type displayModel struct {
	keys         *displayKeyMap
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

	keys := displayKeyMap{
		tab: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "shift through tabs")),
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

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.tab):
			commands = append(commands, m.newChangeTabEvent)
		}

	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	commands = append(commands, cmd)
	return m, tea.Batch(commands...)
}

func (m displayModel) View() string {
	return m.table.View()
}

type PriviligedFunction struct {
	function func() error
}

func (pf PriviligedFunction) Run() error { return pf.function() }

func (PriviligedFunction) SetStdin(io.Reader)  {}
func (PriviligedFunction) SetStdout(io.Writer) {}
func (PriviligedFunction) SetStderr(io.Writer) {}
func callback(err error) tea.Msg {
	if err != nil {
		slog.Error("Failed updating database")
		return ErrUpdating
	}
	return Updated
}
func waitForResult(ch chan backend.OperationResult) tea.Cmd {
	return func() tea.Msg {
		result := <-ch
		if result.Success {
			return Updated
		}
		return ErrUpdating
	}
}

func Update() (tea.Cmd, chan backend.OperationResult) {
	authFunc, resultChan := backend.Update()
	return tea.Exec(PriviligedFunction{authFunc}, callback), resultChan
}

type PrivilegedCmdSuccess struct{}
type PrivilegedCmdError struct{ Err error }
