package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func PlaceOverlay(base string, content string) string {
	base = stripAnsi(base)
	baseLines := strings.Split(base, "\n")
	contentLines := strings.Split(content, "\n")

	baseHeight := len(baseLines)
	baseWidth := 0
	for _, line := range baseLines {
		w := len([]rune(line))
		if w > baseWidth {
			baseWidth = w
		}
	}

	contentHeight := len(contentLines)
	contentWidth := lipgloss.Width(content)

	contentRowStart := (baseHeight - contentHeight) / 2
	contentRowEnd := contentRowStart + contentHeight
	contentColStart := (baseWidth - contentWidth) / 2

	var builder strings.Builder
	builder.Grow(len(base))

	dimStart := "\x1b[2m"
	dimEnd := "\x1b[22m"

	for row, baseLine := range baseLines {
		if row >= contentRowStart && row < contentRowEnd {
			baseRunes := []rune(baseLine)
			contentLine := contentLines[row-contentRowStart]

			builder.WriteString(dimStart)
			builder.WriteString(string(baseRunes[:contentColStart]))
			builder.WriteString(dimEnd)

			builder.WriteString(contentLine)

			builder.WriteString(dimStart)
			if contentColStart+contentWidth < len(baseRunes) {
				builder.WriteString(string(baseRunes[contentColStart+contentWidth:]))
			}
			builder.WriteString(dimEnd)
		} else {
			builder.WriteString(dimStart)
			builder.WriteString(baseLine)
			builder.WriteString(dimEnd)
		}

		if row < len(baseLines)-1 {
			builder.WriteRune('\n')
		}
	}

	return builder.String()
}
func stripAnsi(s string) string {
	var result strings.Builder
	runes := []rune(s)

	for i := 0; i < len(runes); i++ {
		if runes[i] == '\x1b' && i+1 < len(runes) {
			if runes[i+1] == '[' {
				// CSI sequence: skip until we find a letter
				i += 2
				for i < len(runes) && !isLetter(runes[i]) {
					i++
				}
				continue
			} else if runes[i+1] == ']' {
				// OSC sequence: skip until BEL or ESC backslash
				i += 2
				for i < len(runes) {
					if runes[i] == '\x07' || (runes[i] == '\x1b' && i+1 < len(runes) && runes[i+1] == '\\') {
						if runes[i] == '\x1b' {
							i++
						}
						break
					}
					i++
				}
				continue
			}
		}
		result.WriteRune(runes[i])
	}

	return result.String()
}

func isLetter(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}
