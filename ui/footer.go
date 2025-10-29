package ui

import (
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type FooterResizeEvent struct {
	width  int
	height int
}
type NewSearchTermEvent struct {
	term string
}
type SearchBluredEvent struct{}
type SearchFocusedEvent struct{}
type SearchResetedEvent struct{}

type footerKeyMap struct {
	search key.Binding
	reset  key.Binding
	submit key.Binding
}

func (k footerKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.search}
}

func (k footerKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.search, k.reset, k.submit}}
}

type footerModel struct {
	keys         *footerKeyMap
	search       textinput.Model
	count        int
	cursor       int
	width        int
	previousTerm string
}

func newFooter() footerModel {
	ti := textinput.New()
	ti.Placeholder = "Search packages..."
	ti.Prompt = ""
	ti.CharLimit = 50
	ti.Width = 50
	ti.ShowSuggestions = false
	keys := footerKeyMap{
		search: key.NewBinding(key.WithKeys("/", "ctrl+f"), key.WithHelp("//ctrl+f", "focus search box")),
		reset:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "reset search"), key.WithDisabled()),
		submit: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit search"), key.WithDisabled()),
	}
	return footerModel{keys: &keys, search: ti, count: -1}
}

func (m footerModel) blurSearch() tea.Msg {
	return SearchBluredEvent{}
}

func (m footerModel) focusSearch() tea.Msg {
	return SearchFocusedEvent{}
}

func (m footerModel) resetSearch() tea.Msg {
	return SearchResetedEvent{}
}
func (m footerModel) newSearchTermEvent() tea.Msg {
	return NewSearchTermEvent{
		term: m.previousTerm,
	}
}
func (m footerModel) newFooterResizeEvent() tea.Msg {
	v := m.View()
	return FooterResizeEvent{
		width:  lipgloss.Width(v),
		height: lipgloss.Height(v),
	}
}

func (m footerModel) Update(msg tea.Msg) (footerModel, tea.Cmd) {
	commands := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - 2
		commands = append(commands, m.newFooterResizeEvent)
	case TableEvent:
		switch msg.event {
		case CursorChanged:
			if m.cursor != msg.cursor {
				m.cursor = msg.cursor
			}
		case NewSummary:
			if m.count != msg.summary.count {
				m.count = msg.summary.count
			}
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.search):
			m.search.Focus()

			m.keys.reset.SetEnabled(true)
			m.keys.submit.SetEnabled(true)
			m.keys.search.SetEnabled(false)

			return m, tea.Batch(m.focusSearch, textinput.Blink)

		case key.Matches(msg, m.keys.reset):
			m.search.Blur()
			m.search.Reset()
			m.search.SetValue("")

			m.keys.reset.SetEnabled(false)
			m.keys.submit.SetEnabled(false)
			m.keys.search.SetEnabled(true)

			return m, m.resetSearch

		case key.Matches(msg, m.keys.submit):
			m.search.Blur()

			m.keys.reset.SetEnabled(true)
			m.keys.submit.SetEnabled(false)
			m.keys.search.SetEnabled(true)
			return m, m.blurSearch
		default:
			if m.search.Focused() && !slices.Contains(msg.Runes, '?') {
				var cmd tea.Cmd
				m.search, cmd = m.search.Update(msg)
				term := strings.ToLower(m.search.Value())
				if m.previousTerm == term {
					return m, cmd
				}
				m.previousTerm = term

				return m, m.newSearchTermEvent
			}

		}

	}

	return m, tea.Batch(commands...)
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
