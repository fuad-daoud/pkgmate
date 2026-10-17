package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestPlaceOverlay(t *testing.T) {
	t.Run("A base of width 0 with content abc", func(t *testing.T) {
		out := PlaceOverlay("", "abc")
		lines := strings.Split(out, "\n")
		if len(lines) != 1 {
			t.Fatalf("expected 1 line, got %d", len(lines))
		}
		if !strings.Contains(out, "abc") {
			t.Fatalf("expected output to contain 'abc', got %q", out)
		}
	})

	t.Run("Base width 1", func(t *testing.T) {
		out := PlaceOverlay("x", "abc")
		lines := strings.Split(out, "\n")
		if len(lines) != 1 {
			t.Fatalf("expected 1 line, got %d", len(lines))
		}
		if !strings.Contains(out, "abc") {
			t.Fatalf("expected output to contain 'abc', got %q", out)
		}
	})

	t.Run("Content narrower than the base: byte-identity with old behaviour", func(t *testing.T) {
		base := "1234567890\n1234567890\n1234567890"
		content := "foo"
		expected := "\x1b[2m1234567890\x1b[22m\n\x1b[2m123\x1b[22mfoo\x1b[2m7890\x1b[22m\n\x1b[2m1234567890\x1b[22m"
		out := PlaceOverlay(base, content)
		if out != expected {
			t.Fatalf("byte-identity mismatch:\ngot  %q\nwant %q", out, expected)
		}
	})

	t.Run("Content the same width as the base", func(t *testing.T) {
		base := "12345\n12345\n12345"
		content := "abcde"
		out := PlaceOverlay(base, content)
		plain := stripAnsi(out)
		lines := strings.Split(plain, "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines, got %d", len(lines))
		}
		if lines[1] != "abcde" {
			t.Fatalf("expected middle line 'abcde', got %q", lines[1])
		}
	})

	t.Run("Content wider than the base", func(t *testing.T) {
		base := "12\n12\n12"
		content := "wider content"
		out := PlaceOverlay(base, content)
		plain := stripAnsi(out)
		lines := strings.Split(plain, "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines, got %d", len(lines))
		}
		if !strings.Contains(lines[1], "wider content") {
			t.Fatalf("expected middle line to contain 'wider content', got %q", lines[1])
		}
	})

	t.Run("Content taller than the base", func(t *testing.T) {
		base := "12345"
		content := "a\nb\nc"
		out := PlaceOverlay(base, content)
		lines := strings.Split(out, "\n")
		if len(lines) != 1 {
			t.Fatalf("expected 1 line (clipped to base height), got %d", len(lines))
		}
	})

	t.Run("Ragged base where overlaid line is shorter than column start: space padding", func(t *testing.T) {
		base := "1234567890\n12\n1234567890"
		content := "foo"
		// baseWidth = 10, contentWidth = 3 -> contentColStart = 3
		// line 1 has length 2, shorter than colStart 3.
		// Expected: line 1 padded with 1 space so "foo" starts at index 3.
		out := PlaceOverlay(base, content)
		plain := stripAnsi(out)
		lines := strings.Split(plain, "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines, got %d", len(lines))
		}
		if lines[1] != "12 foo" {
			t.Fatalf("expected line 1 to be '12 foo' with padding, got %q", lines[1])
		}
	})

	t.Run("Multi-byte base: rune-correct splice", func(t *testing.T) {
		base := "ééééé\nééééé\nééééé"
		content := "ab"
		// baseWidth = 5 runes, contentWidth = 2 runes -> colStart = 1
		out := PlaceOverlay(base, content)
		plain := stripAnsi(out)
		lines := strings.Split(plain, "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines, got %d", len(lines))
		}
		if lines[1] != "éabéé" {
			t.Fatalf("expected 'éabéé', got %q", lines[1])
		}

		// Box runes test
		boxBase := "┌───┐\n│   │\n└───┘"
		boxOut := PlaceOverlay(boxBase, "X")
		boxPlain := stripAnsi(boxOut)
		boxLines := strings.Split(boxPlain, "\n")
		if boxLines[1] != "│ X │" {
			t.Fatalf("expected '│ X │', got %q", boxLines[1])
		}
	})

	t.Run("ANSI-styled base: stripped before splicing", func(t *testing.T) {
		base := "\x1b[31mhello\x1b[0m\n\x1b[32mworld\x1b[0m\n\x1b[33mextra\x1b[0m"
		content := "X"
		out := PlaceOverlay(base, content)
		if strings.Contains(out, "\x1b[31m") || strings.Contains(out, "\x1b[32m") || strings.Contains(out, "\x1b[33m") {
			t.Fatalf("base ANSI was not stripped: %q", out)
		}
		plain := stripAnsi(out)
		lines := strings.Split(plain, "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines, got %d", len(lines))
		}
	})

	t.Run("ANSI-styled content: width measured in cells and escapes preserved", func(t *testing.T) {
		base := "1234567890\n1234567890\n1234567890"
		styledContent := lipgloss.NewStyle().Bold(true).Render("hi")
		out := PlaceOverlay(base, styledContent)
		if !strings.Contains(out, styledContent) {
			t.Fatalf("expected content escapes to be preserved, got %q", out)
		}
		plain := stripAnsi(out)
		lines := strings.Split(plain, "\n")
		if lines[1] != "1234hi7890" {
			t.Fatalf("expected plain line '1234hi7890', got %q", lines[1])
		}
	})
}
