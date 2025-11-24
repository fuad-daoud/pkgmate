package ui

import (
	"log/slog"
	"slices"
	"strconv"
	"strings"

	"pkgmate/backend"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	HeaderHeight = 2
	FooterHeight = 2
)

type TableInitEvent struct{}
type LoadOrphans struct{}
type LoadPackages struct{}

func NewLoadOrphans() tea.Msg {
	return LoadOrphans{}
}
func NewLoadPackages() tea.Msg {
	return LoadPackages{}
}

type tableKeys struct {
	customTableKey *customTableKeyMap

	tab            key.Binding
	prevTab        key.Binding
	remove         key.Binding
	exitSelectMode key.Binding
	search         key.Binding
	reset          key.Binding
	submit         key.Binding
}

func (k tableKeys) ShortHelp() []key.Binding {
	help := k.customTableKey.ShortHelp()
	help = append(help, k.remove)
	return help
}

func (k tableKeys) FullHelp() [][]key.Binding {
	help := k.customTableKey.FullHelp()
	help = append(help, []key.Binding{k.remove})
	return help
}

type tableModel struct {
	keys         tableKeys
	tables       []customTable
	search       textinput.Model
	previousTerm string
	lastCursor   int
	activeTable  int
	pkgStream    chan []backend.Package
	orphanStream chan []backend.Package
}

func (m *tableModel) table() *customTable {
	return &m.tables[m.activeTable]
}

type CursorChangedEvent struct {
	cursor int
}

func newTable() tableModel {
	keys := tableKeys{
		remove:         key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "select packages to delete (root)"), key.WithDisabled()),
		exitSelectMode: key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "clear selected packages"), key.WithDisabled()),
		customTableKey: newCustomTable("Direct Packages").keys,
		search:         key.NewBinding(key.WithKeys("/", "ctrl+f"), key.WithHelp("//ctrl+f", "focus search box")),
		reset:          key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "reset search"), key.WithDisabled()),
		submit:         key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit search"), key.WithDisabled()),
		tab:            key.NewBinding(key.WithKeys("tab", "ctrl+l", "ctrl+right"), key.WithHelp("tab", "shift through tabs")),
		prevTab:        key.NewBinding(key.WithKeys("shift+tab", "ctrl+h", "ctrl+left"), key.WithHelp("shift+tab", "previous tab")),
	}

	ti := textinput.New()
	ti.Placeholder = "Search packages..."
	ti.Prompt = ""
	ti.CharLimit = 50
	ti.Width = 50
	ti.ShowSuggestions = false
	return tableModel{keys: keys, tables: []customTable{*newCustomTable("Direct Packages"), *newCustomTable("Dependency Packages"), *newCustomTable("All Packages"), *newCustomTable("Orphan Packages")}, search: ti}
}

func (m tableModel) Init() tea.Cmd {
	return func() tea.Msg { return TableInitEvent{} }
}

type PackageStreamMsg struct {
	done bool
	pkgs []backend.Package
}

type OrphanStreamMsg struct {
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
func (m tableModel) listenOrphans() tea.Msg {
	data, ok := <-m.orphanStream
	if !ok {
		slog.Info("orphan channel is closed")
		return OrphanStreamMsg{done: true}
	}
	return OrphanStreamMsg{pkgs: data}
}

func (m tableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var commands []tea.Cmd
	switch msg := msg.(type) {
	case TableInitEvent:
		columns := []string{
			"Name",
			"Version",
			"Size",
			"Installed",
		}
		var initCmds []tea.Cmd
		for i := range m.tables {
			m.tables[i].Columns = columns
			initCmds = append(initCmds, m.tables[i].Init())
		}
		commands = append(commands, NewLoadPackages, NewLoadOrphans)
		commands = append(commands, initCmds...)
	case LoadPackages:
		pkgChan, err := backend.LoadAllPackages()
		if err != nil {
			slog.Error("could not load packages", "err", err)
			break
		}
		m.pkgStream = pkgChan
		commands = append(commands, m.listen)

	case LoadOrphans:
		orphanStream, err := backend.GetAllOrphanPackages()
		if err != nil {
			slog.Error("could not load orphans", "err", err)
			break
		}
		m.orphanStream = orphanStream
		commands = append(commands, m.listenOrphans)

	case DisplayResizeEvent:
		for i := range m.tables {
			m.tables[i].Height = msg.height - HeaderHeight - FooterHeight
			m.tables[i].Width = msg.width
		}

	case PackageStreamMsg:
		if len(msg.pkgs) > 0 {
			for i := range m.tables[:3] {
				m.tables[i].SetLoading(false)
			}
		}

		pkgs := msg.pkgs
		for _, pkg := range pkgs {
			row := table.Row{pkg.Name, pkg.FormatVersion(), pkg.FormatSize(), pkg.Date.Format("2006-01-02")}
			if pkg.IsDirect {
				m.tables[0].addRow(row)
				if pkg.NewVersion != "" {
					m.tables[0].addStyleRow(pkg.Name, updateAvailableRow)
				}
				if pkg.IsFrozen {
					m.tables[0].addStyleRow(pkg.Name, frozenRowStyle)
				}
			} else {
				m.tables[1].addRow(row)
				if pkg.NewVersion != "" {
					m.tables[1].addStyleRow(pkg.Name, updateAvailableRow)
				}
				if pkg.IsFrozen {
					m.tables[1].addStyleRow(pkg.Name, frozenRowStyle)
				}
			}
			m.tables[2].addRow(row)
			if pkg.NewVersion != "" {
				m.tables[2].addStyleRow(pkg.Name, updateAvailableRow)
			}
			if pkg.IsFrozen {
				m.tables[2].addStyleRow(pkg.Name, frozenRowStyle)
			}
		}
		if !msg.done {
			commands = append(commands, m.listen)
		}
	case OrphanStreamMsg:
		if len(msg.pkgs) > 0 {
			m.tables[3].SetLoading(false)
		}

		for _, pkg := range msg.pkgs {
			row := table.Row{pkg.Name, pkg.FormatVersion(), pkg.FormatSize(), pkg.Date.Format("2006-01-02")}
			m.tables[3].addRow(row)
			if pkg.NewVersion != "" {
				m.tables[3].addStyleRow(pkg.Name, updateAvailableRow)
			}
			if pkg.IsFrozen {
				m.tables[3].addStyleRow(pkg.Name, frozenRowStyle)
			}
		}

		if !msg.done {
			commands = append(commands, m.listenOrphans)
		}
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.remove):
			m.keys.remove.SetEnabled(false)
			m.keys.exitSelectMode.SetEnabled(true)
			m.table().EnterSelectMode(dangerRow, cursorAndDangerRow)
		case key.Matches(msg, m.keys.exitSelectMode):
			m.keys.exitSelectMode.SetEnabled(false)
			m.keys.remove.SetEnabled(true)
			m.table().ExitSelectMode()

		case key.Matches(msg, m.keys.prevTab):
			m.keys.exitSelectMode.SetEnabled(false)
			m.table().ExitSelectMode()
			m.activeTable -= 1
			m.activeTable += len(m.tables)
			m.activeTable %= len(m.tables)
		case key.Matches(msg, m.keys.tab):
			m.keys.exitSelectMode.SetEnabled(false)
			m.table().ExitSelectMode()
			m.activeTable += 1
			m.activeTable %= len(m.tables)
		}

	}
	var newCmd tea.Cmd
	for index, table := range m.tables {
		m.tables[index], newCmd = table.Update(msg)
		commands = append(commands, newCmd)
	}
	if m.lastCursor != m.table().cursor {
		m.lastCursor = m.table().cursor
	}

	m, newCmd = m.footerUpdates(msg)
	commands = append(commands, newCmd)

	return m, tea.Batch(commands...)
}

func (m tableModel) footerUpdates(msg tea.Msg) (tableModel, tea.Cmd) {
	commands := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.search):
			m.search.Focus()

			m.keys.reset.SetEnabled(true)
			m.keys.submit.SetEnabled(true)
			m.keys.search.SetEnabled(false)

			m.table().Blur()
			return m, textinput.Blink

		case key.Matches(msg, m.keys.reset):
			m.search.Blur()
			m.search.Reset()
			m.search.SetValue("")
			m.previousTerm = ""

			m.keys.reset.SetEnabled(false)
			m.keys.submit.SetEnabled(false)
			m.keys.search.SetEnabled(true)

			m.table().Reset()
			return m, nil

		case key.Matches(msg, m.keys.submit):
			m.search.Blur()

			m.keys.reset.SetEnabled(true)
			m.keys.submit.SetEnabled(false)
			m.keys.search.SetEnabled(true)

			m.table().Focus()

			return m, nil
		default:
			if m.search.Focused() && !slices.Contains(msg.Runes, '?') {
				var cmd tea.Cmd
				m.search, cmd = m.search.Update(msg)
				term := strings.ToLower(m.search.Value())
				if m.previousTerm == term {
					return m, cmd
				}
				m.previousTerm = term

				m.table().filterColumn("Name", m.previousTerm)
				return m, nil
			}

		}

	}

	return m, tea.Batch(commands...)
}

func (m tableModel) View() string {
	return lipgloss.JoinVertical(lipgloss.Bottom, m.tabsView(), m.table().View(), m.footerView())
}

func (m tableModel) tabsView() string {
	tabs := make([]string, len(m.tables))
	for i, v := range m.tables {
		if i == m.activeTable {
			tab := topLeftTab.Bold(true).
				BorderStyle(lipgloss.ThickBorder()).
				BorderForeground(selectedColor).
				Render(v.label + "|" + strconv.Itoa(len(v.OriginalRows)))
			tabs = append(tabs, tab)
			continue
		}
		tab := topLeftTab.Render(v.label + "|" + strconv.Itoa(len(v.OriginalRows)))
		tabs = append(tabs, tab)

	}

	leftSection := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	spacer := spaceStyle.Width(m.table().Width - lipgloss.Width(leftSection)).Render()
	return lipgloss.JoinHorizontal(lipgloss.Bottom, leftSection, spacer)
}

func (m tableModel) footerView() string {
	cursor := bottomRightTab.Render(strconv.Itoa(m.table().cursor + 1))
	count := bottomRightTab.Render(strconv.Itoa(len(m.table().Rows)))
	rightSection := lipgloss.JoinHorizontal(lipgloss.Top, cursor, count)

	styledSearch := bottomLeftTab.Render(m.search.View())
	searchColumn := bottomLeftTab.Bold(true).Render("Name")
	leftSection := lipgloss.JoinHorizontal(lipgloss.Bottom, styledSearch, searchColumn)
	spacer := spaceStyle.Width(m.table().Width - lipgloss.Width(leftSection) - lipgloss.Width(rightSection)).Render()
	return lipgloss.JoinHorizontal(lipgloss.Top, leftSection, spacer, rightSection)
}
