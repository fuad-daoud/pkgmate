package ui

import (
	"fmt"
	"os"
	"strings"

	"pkgmate/backend"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	packages       []backend.Pkg
	filtered       []backend.Pkg
	textInput      textinput.Model
	cursor         int
	offset         int
	width          int
	height         int
	viewportHeight int
	viewportWidth  int
	table          table.Model
}

func InitialModel() model {
	packages, err := backend.LoadPackages()
	if err != nil {
		fmt.Printf("Error loading packages: %v\n", err)
		os.Exit(1)
	}

	ti := textinput.New()
	ti.Placeholder = "Search packages..."
	ti.CharLimit = 50
	ti.Width = 50
	t := table.New()

	t.Focus()

	return model{
		packages:  packages,
		filtered:  packages,
		textInput: ti,
		cursor:    0,
		offset:    0,
		table:     t,
	}
}

func fetchPackages() tea.Msg {
	pkgs, err := backend.LoadPackages()
	if err != nil {
		os.Exit(1)
	}
	return pkgs

}

func (m model) Init() tea.Cmd {
	return fetchPackages
}

func (m *model) filterPackages() {
	query := strings.ToLower(m.textInput.Value())
	if query == "" {
		m.filtered = m.packages
		return
	}

	m.filtered = []backend.Pkg{}
	for _, p := range m.packages {
		if strings.Contains(strings.ToLower(p.Name), query) {
			m.filtered = append(m.filtered, p)
		}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewportHeight = msg.Height - 10
		m.viewportWidth = msg.Width - 10

		columnWidth := max(0, m.viewportWidth/3-10)
		columns := []table.Column{
			{Title: "Name", Width: columnWidth},
			{Title: "Version", Width: columnWidth},
			{Title: "Size", Width: columnWidth},
		}

		m.table.SetColumns(columns)
		m.table.SetHeight(m.viewportHeight)
		return m, nil
	case []backend.Pkg:
		pkgs := msg

		m.packages = pkgs

		rows := []table.Row{}
		for _, pkg := range pkgs {
			row := table.Row{pkg.Name, pkg.Version, formatSize(pkg.Size)}
			rows = append(rows, row)
		}
		m.table.SetRows(rows)
		return m, cmd

	case tea.KeyMsg:
		switch msg.String() {
		case "/":
			m.textInput.Focus()
			return m, textinput.Blink

		case "esc":
			if m.textInput.Focused() {
				m.textInput.Blur()
				return m, nil
			}
			return m, cmd

		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.offset {
					m.offset--
				}
			}

		case "down", "j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
				if m.cursor >= m.offset+m.viewportHeight {
					m.offset++
				}
			}

		case "pgup", "ctrl+u":
			m.cursor -= m.viewportHeight
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.offset = m.cursor

		case "pgdown", "ctrl+d":
			m.cursor += m.viewportHeight
			if m.cursor >= len(m.filtered) {
				m.cursor = len(m.filtered) - 1
			}
			if m.cursor >= m.offset+m.viewportHeight {
				m.offset = m.cursor - m.viewportHeight + 1
			}

		case "home", "0":
			m.cursor = 0
			m.offset = 0

		case "end", "shift+4":
			m.cursor = len(m.filtered) - 1
			m.offset = max(0, len(m.filtered)-m.viewportHeight)

		default:
			m.textInput, cmd = m.textInput.Update(msg)
			m.filterPackages()
			m.cursor = 0
			m.offset = 0
			return m, cmd
		}
	}
	m.table, cmd = m.table.Update(msg)

	return m, cmd
}
