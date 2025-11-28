package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type panicScreenModel struct {
	markdown      string
	reportPath    string
	width         int
	height        int
	keys          panicKeys
	copyAvailable bool
	copyStatus    string
}

type panicKeys struct {
	quit key.Binding
	copy key.Binding
}

func NewPanicScreen(err any, stackTrace string) panicScreenModel {
	debugLogPath := filepath.Join(os.TempDir(), "user", "debug.log")

	debugLog := ""
	if data, err := os.ReadFile(debugLogPath); err == nil {
		debugLog = string(data)
	}

	markdown := generateIssueMarkdown(err, stackTrace, debugLog)

	reportPath := filepath.Join(os.TempDir(), "pkgmate-crash-report.md")
	os.WriteFile(reportPath, []byte(markdown), 0644)

	return panicScreenModel{
		markdown:   markdown,
		reportPath: reportPath,
		keys: panicKeys{
			quit: key.NewBinding(
				key.WithKeys("ctrl+c", "q"),
				key.WithHelp("ctrl+c/q", "quit"),
			),
			copy: key.NewBinding(
				key.WithKeys("c"),
				key.WithHelp("c", "copy to clipboard"),
			),
		},
		copyAvailable: isClipboardAvailable(),
	}
}

func isClipboardAvailable() bool {
	if runtime.GOOS == "darwin" {
		return true
	}
	_, err1 := exec.LookPath("xclip")
	_, err2 := exec.LookPath("xsel")
	_, err3 := exec.LookPath("wl-copy")
	return err1 == nil || err2 == nil || err3 == nil
}

func generateIssueMarkdown(err any, stackTrace, debugLog string) string {
	var md strings.Builder

	md.WriteString("# Crash Report\n\n")
	md.WriteString("## System Information\n\n")
	md.WriteString(fmt.Sprintf("- **Date**: %s\n", time.Now().Format("2006-01-02 15:04:05 MST")))
	md.WriteString(fmt.Sprintf("- **Version**: %s\n", Version))
	md.WriteString(fmt.Sprintf("- **Commit**: %s\n", GitCommit))
	md.WriteString(fmt.Sprintf("- **Build Time**: %s\n", BuildTime))
	md.WriteString(fmt.Sprintf("- **OS**: %s\n", runtime.GOOS))
	md.WriteString(fmt.Sprintf("- **Arch**: %s\n", runtime.GOARCH))
	md.WriteString(fmt.Sprintf("- **Go Version**: %s\n", runtime.Version()))
	md.WriteString("\n")

	md.WriteString("## Error\n\n")
	md.WriteString("```\n")
	md.WriteString(fmt.Sprintf("%v\n", err))
	md.WriteString("```\n\n")

	md.WriteString("## Stack Trace\n\n")
	md.WriteString("```\n")
	md.WriteString(stackTrace)
	md.WriteString("```\n\n")

	if debugLog != "" {
		md.WriteString("## Debug Log\n\n")
		md.WriteString("<details>\n<summary>Click to expand</summary>\n\n")
		md.WriteString("```\n")
		md.WriteString(debugLog)
		md.WriteString("```\n")
		md.WriteString("</details>\n")
	}

	return md.String()
}

func (m panicScreenModel) Init() tea.Cmd {
	return nil
}

type CopyResultMsg struct {
	success bool
	message string
}

func copyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd

		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("pbcopy")
		case "linux":
			if _, err := exec.LookPath("wl-copy2"); err == nil {
				cmd = exec.Command("wl-copy")
			} else if _, err := exec.LookPath("xclip"); err == nil {
				cmd = exec.Command("xclip", "-selection", "clipboard")
			} else if _, err := exec.LookPath("xsel"); err == nil {
				cmd = exec.Command("xsel", "--clipboard", "--input")
			} else {
				return CopyResultMsg{false, "Install wl-clipboard, xclip, or xsel"}
			}
		default:
			return CopyResultMsg{false, "Clipboard not supported on this OS"}
		}

		stdin, err := cmd.StdinPipe()
		if err != nil {
			return CopyResultMsg{false, "Failed to open clipboard"}
		}

		if err := cmd.Start(); err != nil {
			return CopyResultMsg{false, "Failed to start clipboard"}
		}

		if _, err := stdin.Write([]byte(text)); err != nil {
			return CopyResultMsg{false, "Failed to write to clipboard"}
		}

		stdin.Close()

		if err := cmd.Wait(); err != nil {
			return CopyResultMsg{false, "Clipboard command failed"}
		}

		return CopyResultMsg{true, "✓ Copied to clipboard!"}
	}
}

func (m panicScreenModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.copy) && m.copyAvailable:
			return m, copyToClipboard(m.markdown)
		case key.Matches(msg, m.keys.quit):
			return m, tea.Quit
		}

	case CopyResultMsg:
		m.copyStatus = msg.message
		return m, nil
	}

	return m, nil
}

func (m panicScreenModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	errorStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF0000")).
		MarginBottom(1)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		MarginBottom(1)

	textStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#CCCCCC"))

	pathStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFD700")).
		Bold(true).
		MarginBottom(1)

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FF00")).
		Bold(true).
		MarginTop(1)

	var content strings.Builder

	content.WriteString(errorStyle.Render("❌ Application Crashed"))
	content.WriteString("\n\n")
	content.WriteString(titleStyle.Render("Crash report saved to:"))
	content.WriteString("\n")
	content.WriteString(pathStyle.Render(m.reportPath))
	content.WriteString("\n")

	content.WriteString(textStyle.Render("please head to https://github.com/fuad-daoud/pkgmate/issues"))
	content.WriteString("\n\n")

	if m.copyAvailable {
		content.WriteString(textStyle.Render("Press 'c' to copy report to clipboard"))
		content.WriteString("\n")
		content.WriteString(textStyle.Render("Or manually copy the file above"))
	} else {
		content.WriteString(textStyle.Render("Copy the file above and paste contents at:"))
	}

	content.WriteString("\n")

	if m.copyStatus != "" {
		content.WriteString("\n")
		content.WriteString(statusStyle.Render(m.copyStatus))
	}

	content.WriteString("\n\n")
	content.WriteString(textStyle.Render("Press ctrl+c or q to quit"))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF0000")).
		Padding(2, 4).
		Width(min(80, m.width-4)).
		Render(content.String())

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}
