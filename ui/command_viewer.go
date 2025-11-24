package ui

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pkgmate/backend"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func NewUpdateCommandView() commandExecModel {
	return newCommandExecView(
		"System Update",
		"pkgmate-update.log",
		backend.UpdateAll,
	)
}

type CommandExecutor func() (func() error, chan backend.OperationResult)

type commandExecModel struct {
	viewport    viewport.Model
	rootPath    string
	logFile     string
	running     bool
	resultChan  chan backend.OperationResult
	lastLogSize int
	commandName string
	commandFunc CommandExecutor
}

func (commandExecModel) Init() tea.Cmd { return nil }

type LogTickMsg time.Time

// newCommandExecView creates a new command execution view
// Parameters:
//   - commandName: Display name shown in the UI (e.g., "System Update", "Package Install")
//   - logFile: Name of the log file to monitor (e.g., "pkgmate-update.log")
//   - cmdFunc: The command executor function that will be called when user presses Enter
func newCommandExecView(commandName, logFile string, cmdFunc CommandExecutor) commandExecModel {
	rootPath := filepath.Join(os.TempDir(), "user")
	err := os.MkdirAll(rootPath, 0750)
	if err != nil {
		slog.Error("Failed to create command logs dir", "err", err)
	}

	return commandExecModel{
		rootPath:    rootPath,
		logFile:     logFile,
		commandName: commandName,
		commandFunc: cmdFunc,
		running:     false,
	}
}

func (m *commandExecModel) startCommand() tea.Cmd {
	m.running = true
	execFunc, resultChan := m.commandFunc()
	m.resultChan = resultChan
	m.lastLogSize = 0

	return tea.Batch(
		tea.Exec(PrivilegedFunction{execFunc}, func(err error) tea.Msg {
			if err != nil {
				slog.Error("Failed to start command", "command", m.commandName, "err", err)
			}
			return nil
		}),
		m.tickLog(),
		m.waitForResult(),
	)
}

func (m *commandExecModel) tickLog() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return LogTickMsg(t)
	})
}

func (m *commandExecModel) waitForResult() tea.Cmd {
	return func() tea.Msg {
		result := <-m.resultChan
		return result
	}
}

func (m *commandExecModel) content() string {
	logPath := filepath.Join(m.rootPath, m.logFile)
	content, err := os.ReadFile(logPath)
	if err != nil {
		if m.running {
			return "Waiting for " + m.commandName + " to start...\n"
		}
		return "No logs available. Press Enter to run " + m.commandName + ".\n"
	}

	logs := string(content)
	if logs == "" && m.running {
		return m.commandName + " started...\n"
	}
	return logs
}

type PrivilegedFunction struct {
	function func() error
}

func (pf PrivilegedFunction) Run() error { return pf.function() }

func (PrivilegedFunction) SetStdin(io.Reader)  {}
func (PrivilegedFunction) SetStdout(io.Writer) {}
func (PrivilegedFunction) SetStderr(io.Writer) {}

type PrivilegedCmdSuccess struct{}
type PrivilegedCmdError struct{ Err error }

func (m commandExecModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case DisplayResizeEvent:
		m.viewport = viewport.New(msg.width, msg.height)
		m.viewport.SetContent(m.content())

	case LogTickMsg:
		// Only tick if command is still running
		if m.running {
			content := m.content()
			newSize := len(content)
			// Update viewport only if content changed
			if newSize != m.lastLogSize {
				m.viewport.SetContent(content)
				m.viewport.GotoBottom() // Auto-scroll to see latest logs
				m.lastLogSize = newSize
			}
			cmds = append(cmds, m.tickLog())
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			// Only start if not already running
			if !m.running {
				cmds = append(cmds, m.startCommand())
			}
		}

	case backend.OperationResult:
		// Command completed
		m.running = false
		m.viewport.SetContent(m.content())
		m.viewport.GotoBottom()

		if msg.Success {
			slog.Info("Command completed successfully", "command", m.commandName)
		} else {
			slog.Error("Command failed", "command", m.commandName, "err", msg.Error)
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m commandExecModel) View() string {
	// Build the header with status on the right side
	var statusText string
	var statusColor lipgloss.Color

	if m.running {
		statusText = "● Running"
		statusColor = loadingColor
	} else {
		statusText = "○ Ready"
		statusColor = allGoodColor
	}

	// Style the status indicator
	statusIndicator := lipgloss.NewStyle().
		Foreground(statusColor).
		Bold(true).
		Render(statusText)

	// Create the header line
	title := m.commandName
	spacerWidth := max(0, m.viewport.Width-len(title)-lipgloss.Width(statusIndicator)-2)
	spacer := strings.Repeat(" ", spacerWidth)

	header := lipgloss.NewStyle().
		Bold(true).
		Render(title) + spacer + statusIndicator

	separator := strings.Repeat("─", m.viewport.Width)

	// Build help text at the bottom
	var helpText string
	if m.running {
		helpText = "Command is running... Logs updating in real-time"
	} else {
		helpText = "Press Enter to execute • ↑/↓ to scroll"
	}

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render(helpText)

	helpSpacer := strings.Repeat(" ", max(0, m.viewport.Width-len(helpText)))
	footer := help + helpSpacer

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		separator,
		m.viewport.View(),
		footer,
	)
}
