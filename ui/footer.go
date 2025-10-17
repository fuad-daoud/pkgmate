package ui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type FooterEvents int

type FooterEvent struct {
	term  string
	event FooterEvents
}

const (
	SearchBlured FooterEvents = iota
	SearchFocused
	SearchReseted
	NewSearchTerm
)

type footerModel struct {
	search       textinput.Model
	count        int
	cursor       int
	width        int
	previousTerm string
}

func newFooter() *footerModel {
	ti := textinput.New()
	ti.Placeholder = "Search packages..."
	ti.Prompt = ""
	ti.CharLimit = 50
	ti.Width = 50
	ti.ShowSuggestions = false
	return &footerModel{search: ti, count: -1}
}

func (m footerModel) blurSearch() tea.Msg {
	return FooterEvent{
		event: SearchBlured,
	}
}

func (m footerModel) focusSearch() tea.Msg {
	return FooterEvent{
		event: SearchFocused,
	}
}

func (m footerModel) resetSearch() tea.Msg {
	return FooterEvent{
		event: SearchReseted,
	}
}
func (m footerModel) newSearchTermEvent() tea.Msg {
	return FooterEvent{
		event: NewSearchTerm,
		term:  m.previousTerm,
	}
}

func (m *footerModel) Update(msg tea.Msg) (*footerModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - (msg.Width / 6)
	case TableEvent:
		switch msg.event {
		case CursorChanged:
			if m.cursor != msg.cursor {
				m.cursor = msg.cursor
			}
		case NewSummery:
			if m.count != msg.summary.count {
				m.count = msg.summary.count
			}
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "/":
			if !m.search.Focused() {
				m.search.Focus()
				return m, tea.Batch(m.focusSearch, textinput.Blink)
			}

		case "esc":
			if m.search.Focused() {
				m.search.Blur()
				m.search.Reset()
				m.search.SetValue("")
				return m, m.resetSearch
			}

		case "enter":
			if m.search.Focused() {
				m.search.Blur()
				return m, m.blurSearch
			}
			return m, cmd
		}
	}

	if m.search.Focused() {
		m.search, cmd = m.search.Update(msg)
		term := strings.ToLower(m.search.Value())
		if m.previousTerm == term {
			return m, cmd
		}
		m.previousTerm = term

		return m, m.newSearchTermEvent
	}
	return m, cmd
}

func (m *footerModel) View() string {
	cursor := bottomRightTab.Render(strconv.Itoa(m.cursor))
	count := bottomRightTab.Render(strconv.Itoa(m.count))
	rightSection := lipgloss.JoinHorizontal(lipgloss.Top, cursor, count)

	styledSearch := bottomLeftTab.Render(m.search.View())
	searchColumn := bottomLeftTab.Bold(true).Render("Name")
	leftSection := lipgloss.JoinHorizontal(lipgloss.Bottom, styledSearch, searchColumn)
	spacer := spaceStyle.Width(m.width - lipgloss.Width(leftSection) - lipgloss.Width(rightSection)).Render()
	return lipgloss.JoinHorizontal(lipgloss.Top, leftSection, spacer, rightSection)

}
