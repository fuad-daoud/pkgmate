package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type mainKeymap struct {
	quit key.Binding
}

func (k mainKeymap) ShortHelp() []key.Binding {
	return []key.Binding{k.quit}
}

func (k mainKeymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.quit}}
}

type model struct {
	keys           mainKeymap
	width          int
	height         int
	viewportHeight int
	viewportWidth  int
	display        displayModel
	help           helpModel
	spin           spinner.Model
	showSpinner    bool
	shuttingDown   bool
}

func InitialModel() model {
	var keys = mainKeymap{
		quit: key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
	}
	spin := spinner.New(spinner.WithSpinner(spinner.Monkey))
	m := model{
		keys:    keys,
		display: newDisplay(),
		spin:    spin,
		help:    NewHelpModel(),
	}
	return m
}

type ProgramInitEvent struct{}

func (m model) Init() tea.Cmd {
	return func() tea.Msg { return ProgramInitEvent{} }
}

type ShutdownDelayMsg struct{}

func shutdownDelay() tea.Cmd {
	return tea.Tick(600*time.Millisecond, func(t time.Time) tea.Msg {
		return ShutdownDelayMsg{}
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	commands := make([]tea.Cmd, 0)
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.width <= 70 {
			m.showSpinner = true
			commands = append(commands, m.spin.Tick)
		}
	case ShutdownDelayMsg:
		return m, tea.Sequence(tea.ClearScreen, tea.Quit)
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
			m.shuttingDown = true
			return m, shutdownDelay()
		}
	}

	var displayCmd tea.Cmd
	m.display, displayCmd = m.display.Update(msg)
	commands = append(commands, displayCmd)

	return m, tea.Batch(commands...)
}

func (m model) View() string {
	if m.shuttingDown {
		shutdownMsg := lipgloss.NewStyle().
			Bold(true).
			Render("Shutdown Successfully ✅")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, shutdownMsg)
	}
	if m.showSpinner {
		content := fmt.Sprintf("%s Terminal Width (%d) less the minimum width %d", m.spin.View(), m.width, 70)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	content := lipgloss.JoinVertical(lipgloss.Bottom, m.display.View())

	title := " Pkgmate "
	frameWidth := lipgloss.Width(content)

	leftPadding := (frameWidth - lipgloss.Width(title)) / 2
	rightPadding := frameWidth - lipgloss.Width(title) - leftPadding

	topBorder := "╭" + strings.Repeat("─", leftPadding) + title + strings.Repeat("─", rightPadding) + "╮"

	lines := strings.Split(content, "\n")
	bordered := make([]string, 0, len(lines)+2)
	bordered = append(bordered, topBorder)
	for _, line := range lines {
		bordered = append(bordered, "│"+line+"│")
	}
	bordered = append(bordered, "╰"+strings.Repeat("─", frameWidth)+"╯")

	framedContent := strings.Join(bordered, "\n")
	content = lipgloss.JoinVertical(lipgloss.Top, framedContent)
	content = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Top, content)

	return content
}
