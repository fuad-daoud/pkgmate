package ui

import (
	"log/slog"
	"slices"
	"sort"
	"strconv"
	"strings"

	"pkgmate/backend"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	HeaderHeight = 2
	FooterHeight = 2
)

// managerColumnIndex is the index of the "Package Manager" column, the cell that
// carries the duplicate marker.
const managerColumnIndex = 2

// duplicatesTabIndex is the index of the Duplicates tab, appended after the four
// loading tabs.
const duplicatesTabIndex = 4

type TableInitEvent struct{}

type tableKeys struct {
	customTableKey *customTableKeyMap

	tab     key.Binding
	prevTab key.Binding
	search  key.Binding
	reset   key.Binding
	submit  key.Binding
}

func (k tableKeys) ShortHelp() []key.Binding {
	help := k.customTableKey.ShortHelp()
	return help
}

func (k tableKeys) FullHelp() [][]key.Binding {
	help := k.customTableKey.FullHelp()
	return help
}

type tableModel struct {
	keys              tableKeys
	columns           []string
	tables            []customTable
	search            textinput.Model
	previousTerm      string
	lastCursor        int
	activeTable       int
	activeSearchField int
	orphanStream      chan []backend.Package
	// packages holds the latest copy of every loaded package, keyed by
	// RowIdentityKey. It feeds the duplicate computation once loading finishes.
	packages map[string]backend.Package
	// duplicates holds the groups computed on the last finished load.
	duplicates []backend.DuplicateGroup
}

func (m *tableModel) table() *customTable {
	return &m.tables[m.activeTable]
}

type CursorChangedEvent struct {
	cursor int
}

func newTable() tableModel {
	keys := tableKeys{
		customTableKey: newCustomTable("Direct Packages").keys,
		search:         key.NewBinding(key.WithKeys("/", "ctrl+f"), key.WithHelp("//ctrl+f", "focus search box")),
		reset:          key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "reset search"), key.WithDisabled()),
		submit:         key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit search"), key.WithDisabled()),
		tab:            key.NewBinding(key.WithKeys("tab", "ctrl+l", "ctrl+right"), key.WithHelp("tab", "shift through tabs")),
		prevTab:        key.NewBinding(key.WithKeys("shift+tab", "ctrl+h", "ctrl+left"), key.WithHelp("shift+tab", "previous tab")),
	}

	columns := []string{
		"Name",
		"Version",
		"Package Manager",
		"Size",
		"Installed",
	}

	ti := textinput.New()
	ti.Placeholder = "Search packages..."
	ti.Prompt = ""
	ti.CharLimit = 50
	ti.Width = 50
	ti.ShowSuggestions = false
	duplicatesTab := newCustomTable("Duplicates")
	duplicatesTab.emptyState = "No duplicates found"
	return tableModel{keys: keys, columns: columns, tables: []customTable{*newCustomTable("Direct Packages"), *newCustomTable("Dependency Packages"), *newCustomTable("All Packages"), *newCustomTable("Orphan Packages"), *duplicatesTab}, search: ti, packages: make(map[string]backend.Package)}
}

func (m tableModel) Init() tea.Cmd {
	return func() tea.Msg {
		go func() {
			pkgChan, err := backend.LoadAllPackages()
			if err != nil {
				slog.Error("could not load packages", "err", err)
				return
			}
			for data := range pkgChan {
				Program.Send(PackageLoadedMsg{pkgs: data})
			}
			Program.Send(FinishedLoadingPackagesMsg{})
			slog.Info("channel is closed loaded all packages")
		}()
		return TableInitEvent{}
	}
}

// RowIdentityKey builds a collision-free row identity key from package manager and name.
func RowIdentityKey(manager, name string) string {
	return manager + "\x00" + name
}

// packageRow builds the identity key, the cells and the styles for a package.
// It is shared by the loading path and the Duplicates tab so both render a
// package the same way.
func packageRow(pkg backend.Package) (key string, cells []string, styles []lipgloss.Style) {
	dateStr := "unknown"
	if !pkg.Date.IsZero() {
		dateStr = pkg.Date.Format("2006-01-02")
	}
	cells = []string{pkg.Name, pkg.FormatVersion(), pkg.DB, pkg.FormatSize(), dateStr}
	if pkg.NewVersion != "" {
		styles = append(styles, updateAvailableRow)
	}
	if pkg.IsFrozen {
		styles = append(styles, frozenRowStyle)
	}
	return RowIdentityKey(pkg.DB, pkg.Name), cells, styles
}

// addPackageRow adds a package row and its styles to a table under key.
func addPackageRow(t *customTable, key string, cells []string, styles []lipgloss.Style) {
	if t.loading {
		t.SetLoading(false)
	}
	t.addRow(key, cells)
	for _, style := range styles {
		t.addStyleRow(key, style)
	}
}

type PackageLoadedMsg struct {
	pkgs backend.Package
}
type FinishedLoadingPackagesMsg struct{}

func (m tableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var commands []tea.Cmd
	switch msg := msg.(type) {
	case TableInitEvent:
		var initCmds []tea.Cmd
		for i := range m.tables {
			m.tables[i].Columns = m.columns
			initCmds = append(initCmds, m.tables[i].Init())
		}
		commands = append(commands, initCmds...)
	case DisplayResizeEvent:
		for i := range m.tables {
			m.tables[i].Height = msg.height - HeaderHeight - FooterHeight
			m.tables[i].Width = msg.width
		}
	case FinishedLoadingPackagesMsg:
		for i := range m.tables {
			m.tables[i].SetLoading(false)
		}
		loaded := make([]backend.Package, 0, len(m.packages))
		for _, pkg := range m.packages {
			loaded = append(loaded, pkg)
		}
		m.applyDuplicates(backend.FindDuplicates(loaded))
		return m, tea.Batch(commands...)

	case PackageLoadedMsg:
		pkg := msg.pkgs
		key, cells, styles := packageRow(pkg)
		if m.packages == nil {
			m.packages = make(map[string]backend.Package)
		}
		m.packages[key] = pkg
		addToTable := func(i int) {
			addPackageRow(&m.tables[i], key, cells, styles)
		}
		addToTable(2)
		if pkg.IsDirect {
			addToTable(0)
		} else {
			addToTable(1)
		}
		if pkg.IsOrphan {
			addToTable(3)
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.prevTab):
			if m.search.Focused() {
				break
			}
			m.activeTable -= 1
			m.activeTable += len(m.tables)
			m.activeTable %= len(m.tables)
		case key.Matches(msg, m.keys.tab):
			if m.search.Focused() {
				break
			}
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

// duplicateManagerCell renders the Package Manager cell for a package that
// belongs to a duplicate group: its own DB, then the other manager families in
// the group, deduplicated through ManagerFamily and sorted alphabetically.
func duplicateManagerCell(pkg backend.Package, group []backend.Package) string {
	own := backend.ManagerFamily(pkg.DB)
	seen := map[string]bool{}
	others := make([]string, 0, len(group))
	for _, other := range group {
		family := backend.ManagerFamily(other.DB)
		if family == own || seen[family] {
			continue
		}
		seen[family] = true
		others = append(others, family)
	}
	if len(others) == 0 {
		return pkg.DB
	}
	sort.Strings(others)
	return pkg.DB + " (+" + strings.Join(others, ", ") + ")"
}

// setManagerCell rewrites the Package Manager cell of an existing row in place,
// so the row stays visible through an active filter.
func setManagerCell(t *customTable, key, cell string) {
	if managerColumnIndex >= len(t.Columns) || t.Columns[managerColumnIndex] != "Package Manager" {
		return
	}
	for _, row := range t.OriginalRows {
		if row.Key != key {
			continue
		}
		if managerColumnIndex >= len(row.Cells) {
			return
		}
		cells := slices.Clone(row.Cells)
		cells[managerColumnIndex] = cell
		t.addRow(key, cells)
		return
	}
}

// applyDuplicates fills the Duplicates tab with the group's packages and marks
// the Package Manager cell of every grouped package in the four loading tabs.
func (m *tableModel) applyDuplicates(groups []backend.DuplicateGroup) {
	m.duplicates = groups

	markers := make(map[string]string)
	for _, group := range groups {
		for _, pkg := range group.Packages {
			markers[RowIdentityKey(pkg.DB, pkg.Name)] = duplicateManagerCell(pkg, group.Packages)
		}
	}

	if len(groups) == 0 {
		return
	}

	if duplicatesTabIndex < len(m.tables) {
		duplicates := &m.tables[duplicatesTabIndex]
		for _, group := range groups {
			for _, pkg := range group.Packages {
				key, cells, styles := packageRow(pkg)
				addPackageRow(duplicates, key, cells, styles)
			}
		}
	}

	for i := 0; i < len(m.tables) && i != duplicatesTabIndex; i++ {
		for key, cell := range markers {
			setManagerCell(&m.tables[i], key, cell)
		}
	}
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
		case key.Matches(msg, m.keys.tab):
			if !m.search.Focused() {
				break
			}

			m.activeSearchField += 1
			m.activeSearchField %= len(m.columns)

		case key.Matches(msg, m.keys.prevTab):
			if !m.search.Focused() {
				break
			}

			m.activeSearchField -= 1
			m.activeSearchField += len(m.columns)
			m.activeSearchField %= len(m.columns)

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

				m.table().filterColumn(m.columns[m.activeSearchField], m.previousTerm)
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
		tab := topLeftTab.BorderForeground(mutedColor).Foreground(mutedColor).Render(v.label + "|" + strconv.Itoa(len(v.OriginalRows)))
		tabs = append(tabs, tab)

	}

	leftSection := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	spacer := spaceStyle.Width(m.table().Width - lipgloss.Width(leftSection)).Render()
	return lipgloss.JoinHorizontal(lipgloss.Bottom, leftSection, spacer)
}

func (m tableModel) footerView() string {
	cursor := bottomRightTab.Render(strconv.Itoa(m.table().cursor + 1))
	count := bottomRightTab.Render(strconv.Itoa(len(m.table().Rows)))
	parts := []string{cursor, count}
	if len(m.duplicates) > 0 {
		noun := "apps"
		if len(m.duplicates) == 1 {
			noun = "app"
		}
		parts = append(parts, bottomRightTab.Foreground(mutedColor).Render(strconv.Itoa(len(m.duplicates))+" "+noun+" installed more than once"))
	}
	rightSection := lipgloss.JoinHorizontal(lipgloss.Top, parts...)

	styledSearch := bottomLeftTab.Render(m.search.View())
	leftSectionTabs := make([]string, 0)
	leftSectionTabs = append(leftSectionTabs, styledSearch)
	for i, column := range m.columns {
		if i == m.activeSearchField {
			searchField := bottomLeftTab.Bold(true).BorderStyle(lipgloss.ThickBorder()).BorderForeground(selectedColor).Render(m.columns[m.activeSearchField])
			leftSectionTabs = append(leftSectionTabs, searchField)
			continue
		}
		field := bottomLeftTab.Foreground(mutedColor).BorderForeground(mutedColor).Render(column)
		leftSectionTabs = append(leftSectionTabs, field)
	}
	leftSection := lipgloss.JoinHorizontal(lipgloss.Bottom, leftSectionTabs...)
	spacer := spaceStyle.Width(m.table().Width - lipgloss.Width(leftSection) - lipgloss.Width(rightSection)).Render()
	return lipgloss.JoinHorizontal(lipgloss.Top, leftSection, spacer, rightSection)
}
