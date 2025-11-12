package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type mainKeymap struct {
	quit  key.Binding
	debug key.Binding
}

func (k mainKeymap) ShortHelp() []key.Binding {
	return []key.Binding{k.quit}
}

func (k mainKeymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.quit, k.debug}}
}

type model struct {
	keys           mainKeymap
	width          int
	height         int
	viewportHeight int
	viewportWidth  int
	header         headerModel
	display        displayModel
	footer         footerModel
	help           helpModel
	debug          *debugModel
	showDebug      bool
	spin           spinner.Model
	showSpinner    bool
}

func InitialModel() model {
	var keys = mainKeymap{
		quit:  key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		debug: key.NewBinding(key.WithKeys("ctrl+p"), key.WithHelp("ctrl+p", "show debug file content")),
	}
	spin := spinner.New(spinner.WithSpinner(spinner.Monkey))
	m := model{
		keys:      keys,
		header:    newHeader(),
		display:   newDisplay(),
		footer:    newFooter(),
		debug:     newDebug(),
		showDebug: false,
		spin:      spin,
	}
	m.help = NewHelpModel()

	m.help = m.help.addKeys(m.keys)
	m.help = m.help.addKeys(m.display.table.keys)
	m.help = m.help.addKeys(m.display.keys)
	m.help = m.help.addKeys(m.footer.keys)

	return m
}

type ProgramInitEvent struct{}

func (m model) Init() tea.Cmd {
	return func() tea.Msg { return ProgramInitEvent{} }
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// slog.Info("New event", "type", reflect.TypeOf(msg))
	commands := make([]tea.Cmd, 0)
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.width <= 70 {
			m.showSpinner = true
			commands = append(commands, m.spin.Tick)
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)

		if m.width > 70 {
			m.showSpinner = false
		} else {
			commands = append(commands, cmd)
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.quit):
			if !m.footer.search.Focused() {
				return m, tea.Quit
			}
		case key.Matches(msg, m.keys.debug):
			m.showDebug = !m.showDebug
		}
	}
	var footerCmd tea.Cmd
	m.footer, footerCmd = m.footer.Update(msg)
	commands = append(commands, footerCmd)

	var displayCmd tea.Cmd
	m.display, displayCmd = m.display.Update(msg)
	commands = append(commands, displayCmd)

	var headerCmd tea.Cmd

	m.header, headerCmd = m.header.Update(msg)
	commands = append(commands, headerCmd)

	var debugCmd tea.Cmd
	m.debug, debugCmd = m.debug.Update(msg)
	commands = append(commands, debugCmd)

	var helpCmd tea.Cmd
	m.help, helpCmd = m.help.Update(msg)
	commands = append(commands, helpCmd)

	return m, tea.Batch(commands...)
}

func (m model) View() string {
	if m.showDebug {
		content := frameStyle.Render(m.debug.View())
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}
	if m.showSpinner {
		content := fmt.Sprintf("%s Terminal Width (%d) less the minimum width %d", m.spin.View(), m.width, 70)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	content := lipgloss.JoinVertical(lipgloss.Bottom, m.header.View(), m.display.View(), m.footer.View())

	title := " Pkgmate "
	frameWidth := lipgloss.Width(content)

	leftPadding := (frameWidth - lipgloss.Width(title)) / 2
	rightPadding := frameWidth - lipgloss.Width(title) - leftPadding

	topBorder := "╭" + strings.Repeat("─", leftPadding) + title + strings.Repeat("─", rightPadding) + "╮"

	// Add side borders to content
	lines := strings.Split(content, "\n")
	bordered := make([]string, 0, len(lines)+2)
	bordered = append(bordered, topBorder)
	for _, line := range lines {
		bordered = append(bordered, "│"+line+"│")
	}
	bordered = append(bordered, "╰"+strings.Repeat("─", frameWidth)+"╯")

	framedContent := strings.Join(bordered, "\n")
	content = lipgloss.JoinVertical(lipgloss.Top, framedContent, m.help.View())
	content = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Top, content)
	return content
}
