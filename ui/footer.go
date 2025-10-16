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
	search          textinput.Model
	count           int
	cursor          int
	width           int
	previousTerm    string
	cachedCountStr  string
	cachedCursorStr string
	cachedView      string
	viewDirty       bool
}

func newFooter() *footerModel {
	ti := textinput.New()
	ti.Placeholder = "Search packages..."
	ti.CharLimit = 50
	ti.Width = 50
	ti.ShowSuggestions = false
	return &footerModel{search: ti, viewDirty: true, count: -1, cachedCountStr: bottomTab.Render("0"), cachedCursorStr: bottomTab.Render("1")}
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

func (m *footerModel) update(msg tea.Msg) (*footerModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - (msg.Width / 6)
		m.viewDirty = true
	case TableEvent:
		switch msg.event {
		case CursorChanged:
			if m.cursor != msg.cursor {
				m.cursor = msg.cursor
				m.cachedCursorStr = bottomTab.Render(strconv.Itoa(m.cursor + 1))
				m.viewDirty = true
			}
		case NewSummery:
			if m.count != msg.summary.count {
				m.count = msg.summary.count
				m.cachedCountStr = bottomTab.Render(strconv.Itoa(m.count))
				m.viewDirty = true
			}
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "/":
			if !m.search.Focused() {
				m.search.Focus()
				m.viewDirty = true
				return m, tea.Batch(m.focusSearch, textinput.Blink)
			}

		case "esc":
			if m.search.Focused() {
				m.search.Blur()
				m.search.Reset()
				m.search.SetValue("")
				m.viewDirty = true
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
		m.viewDirty = true

		return m, m.newSearchTermEvent
	}
	return m, cmd
}

func (m *footerModel) view() string {
	if !m.viewDirty && m.cachedView != "" {
		return m.cachedView
	}
	styledSearch := bottomTab.Render(m.search.View())
	m.viewDirty = false
	spacer := spaceStyle.Width(m.width - lipgloss.Width(styledSearch) - lipgloss.Width(m.cachedCountStr) - lipgloss.Width(m.cachedCursorStr)).Render()
	m.cachedView = lipgloss.JoinHorizontal(lipgloss.Top, styledSearch, spacer, m.cachedCursorStr, m.cachedCountStr)
	return m.cachedView

}
