package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Command struct {
	Icon        string
	Name        string
	Description string
	Action      tea.Cmd
}

type commandPaletteModel struct {
	visible  bool
	width    int
	height   int
	commands []Command
	cursor   int
	offset   int
}

func newCommandPalette() commandPaletteModel {
	commands := []Command{
		{Icon: "📦", Name: "Browse Packages", Description: "View all installed packages", Action: NewChangeDisplay(newTable())},
		// {Icon: "🔄", Name: "Update System", Description: "Refresh package databases", Action: NewChangeDisplay(NewUpdateCommandView())},
		{Icon: "📝", Name: "Debug Console", Description: "Show application logs", Action: NewChangeDisplay(newDebug())},
		// {Icon: "⬆️", Name: "Upgrade Packages", Description: "Upgrade all installed packages", Action: nil},
		// {Icon: "🗑️", Name: "Remove Package", Description: "Uninstall selected package", Action: nil},
		// {Icon: "🔍", Name: "Search Packages", Description: "Search for available packages", Action: nil},
		// {Icon: "📊", Name: "Package Stats", Description: "View package statistics", Action: nil},
		// {Icon: "⚙️", Name: "Settings", Description: "Configure application settings", Action: nil},
		// {Icon: "❓", Name: "Help", Description: "Show help information", Action: nil},
		{Icon: "🚪", Name: "Quit", Description: "Exit the application", Action: tea.Quit},
	}

	return commandPaletteModel{
		visible:  false,
		width:    100,
		height:   20,
		commands: commands,
		cursor:   0,
		offset:   0,
	}
}

func (m commandPaletteModel) Toggle() commandPaletteModel {
	m.visible = !m.visible
	if !m.visible {
		m.cursor = 0
		m.offset = 0
	}
	return m
}

func (m *commandPaletteModel) adjustOffset() {
	// Each item takes 4 lines: padding(1) + name(1) + desc(1) + margin(1)
	itemHeight := 4
	// Header: title(1) + newline(1) + subtitle(1) + margin(2) = 5 lines
	headerHeight := 5
	// Bottom padding
	footerHeight := 2

	availableHeight := m.height - headerHeight - footerHeight
	visibleItems := availableHeight / itemHeight

	if m.cursor < m.offset {
		m.offset = m.cursor
	} else if m.cursor >= m.offset+visibleItems {
		m.offset = m.cursor - visibleItems + 1
	}

	maxOffset := max(0, len(m.commands)-visibleItems)
	if m.offset > maxOffset {
		m.offset = maxOffset
	}
}

func (m commandPaletteModel) Update(msg tea.Msg) (commandPaletteModel, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case DisplayResizeEvent:
		m.width = msg.width - (msg.width / 3)
		m.height = msg.height - (msg.height / 2)

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.visible = false
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.adjustOffset()
			}
		case "down", "j":
			if m.cursor < len(m.commands)-1 {
				m.cursor++
				m.adjustOffset()
			}
		case "enter":
			if m.cursor < len(m.commands) && m.commands[m.cursor].Action != nil {
				m.visible = false
				action := m.commands[m.cursor].Action
				m.cursor = 0
				m.offset = 0
				return m, action
			}
		}
	}

	return m, nil
}

func (m commandPaletteModel) View() string {
	if !m.visible {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		MarginBottom(1)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#B0B0B0")).
		MarginBottom(2)

	selectedStyle := lipgloss.NewStyle().
		Background(selectedColor).
		Bold(true).
		Padding(1, 2).
		Width(m.width - 6).
		MarginBottom(1)

	normalStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Width(m.width - 6).
		MarginBottom(1)

	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))

	var content strings.Builder

	content.WriteString(titleStyle.Render("⌘ Command Palette"))
	content.WriteString("\n")
	content.WriteString(subtitleStyle.Render("Select a command to execute"))
	content.WriteString("\n")

	itemHeight := 4
	headerHeight := 5
	footerHeight := 2
	availableHeight := m.height - headerHeight - footerHeight
	visibleItems := availableHeight / itemHeight

	end := min(m.offset+visibleItems, len(m.commands))

	for i := m.offset; i < end; i++ {
		cmd := m.commands[i]

		icon := cmd.Icon + " "
		desc := descStyle.Render(cmd.Description)

		line := icon + lipgloss.JoinVertical(lipgloss.Left, cmd.Name, desc)

		if i == m.cursor {
			content.WriteString(selectedStyle.Render(line))
		} else {
			content.WriteString(normalStyle.Render(line))
		}
		content.WriteString("\n")
	}

	if len(m.commands) > visibleItems {
		scrollInfo := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#777777")).
			Render(strings.Repeat(" ", (m.width-30)/2) + "↑/↓ scroll for more commands")
		content.WriteString("\n" + scrollInfo)
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FFFFFF")).
		Padding(2, 2).
		Render(content.String())
}
