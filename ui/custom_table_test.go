package ui

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"pkgmate/backend"
)

func TestFilterColumn(t *testing.T) {
	t.Run("Match on the first column", func(t *testing.T) {
		table := newCustomTable("test")
		table.Columns = []string{"Name", "Version", "Size"}
		table.addRow("curl", []string{"curl", "7.0", "1MB"})
		table.addRow("wget", []string{"wget", "1.0", "2MB"})
		table.addRow("git", []string{"git", "2.0", "5MB"})

		table.filterColumn("Name", "curl")
		if len(table.Rows) != 1 || table.Rows[0].Cells[0] != "curl" {
			t.Fatalf("expected 1 row with 'curl', got %v", table.Rows)
		}
	})

	t.Run("Match on the last column", func(t *testing.T) {
		table := newCustomTable("test")
		table.Columns = []string{"Name", "Version", "Size"}
		table.addRow("curl", []string{"curl", "7.0", "1MB"})
		table.addRow("wget", []string{"wget", "1.0", "2MB"})
		table.addRow("git", []string{"git", "2.0", "5MB"})

		table.filterColumn("Size", "5mb")
		if len(table.Rows) != 1 || table.Rows[0].Cells[0] != "git" {
			t.Fatalf("expected 1 row with 'git', got %v", table.Rows)
		}
	})

	t.Run("No matches gives empty Rows", func(t *testing.T) {
		table := newCustomTable("test")
		table.Columns = []string{"Name", "Version", "Size"}
		table.addRow("curl", []string{"curl", "7.0", "1MB"})
		table.addRow("wget", []string{"wget", "1.0", "2MB"})

		table.filterColumn("Name", "nonexistent")
		if len(table.Rows) != 0 {
			t.Fatalf("expected empty Rows, got %v", table.Rows)
		}
	})

	t.Run("An empty term matches all rows", func(t *testing.T) {
		table := newCustomTable("test")
		table.Columns = []string{"Name", "Version", "Size"}
		table.addRow("curl", []string{"curl", "7.0", "1MB"})
		table.addRow("wget", []string{"wget", "1.0", "2MB"})
		table.addRow("git", []string{"git", "2.0", "5MB"})

		table.filterColumn("Name", "")
		if len(table.Rows) != 3 {
			t.Fatalf("expected 3 rows, got %d", len(table.Rows))
		}
	})

	t.Run("An unknown column name is a no-op", func(t *testing.T) {
		table := newCustomTable("test")
		table.Columns = []string{"Name", "Version", "Size"}
		table.addRow("curl", []string{"curl", "7.0", "1MB"})
		table.addRow("wget", []string{"wget", "1.0", "2MB"})
		table.cursor = 1
		origRows := table.Rows

		table.filterColumn("UnknownColumn", "curl")
		if !reflect.DeepEqual(table.Rows, origRows) {
			t.Fatalf("expected Rows to be unchanged, got %v", table.Rows)
		}
		if table.cursor != 1 {
			t.Fatalf("expected cursor to remain 1, got %d", table.cursor)
		}
	})

	t.Run("Empty Columns is a no-op", func(t *testing.T) {
		table := newCustomTable("test")
		table.addRow("curl", []string{"curl", "7.0", "1MB"})
		table.addRow("wget", []string{"wget", "1.0", "2MB"})
		table.Columns = []string{}
		table.cursor = 1
		origRows := table.Rows

		table.filterColumn("Name", "curl")
		if !reflect.DeepEqual(table.Rows, origRows) {
			t.Fatalf("expected Rows to be unchanged, got %v", table.Rows)
		}
		if table.cursor != 1 {
			t.Fatalf("expected cursor to remain 1, got %d", table.cursor)
		}
	})

	t.Run("A short row is skipped", func(t *testing.T) {
		table := newCustomTable("test")
		table.Columns = []string{"Name", "Version", "Size"}
		table.addRow("curl", []string{"curl", "7.0", "1MB"})
		table.addRow("short", []string{"short"})
		table.addRow("wget", []string{"wget", "1.0", "2MB"})

		table.filterColumn("Size", "mb")
		if len(table.Rows) != 2 {
			t.Fatalf("expected 2 rows, got %d (%v)", len(table.Rows), table.Rows)
		}
		if table.Rows[0].Cells[0] != "curl" || table.Rows[1].Cells[0] != "wget" {
			t.Fatalf("expected curl and wget, got %v", table.Rows)
		}
	})

	t.Run("Case-insensitivity: an upper-case cell matches a lower-case term", func(t *testing.T) {
		table := newCustomTable("test")
		table.Columns = []string{"Name", "Version", "Size"}
		table.addRow("RipGrep", []string{"RipGrep", "13.0", "3MB"})
		table.addRow("curl", []string{"curl", "7.0", "1MB"})

		table.filterColumn("Name", "ripgrep")
		if len(table.Rows) != 1 || table.Rows[0].Cells[0] != "RipGrep" {
			t.Fatalf("expected 1 row with 'RipGrep', got %v", table.Rows)
		}
	})
}

func TestColumnWidth(t *testing.T) {
	tests := []struct {
		name     string
		cols     []string
		width    int
		expected int
	}{
		{"5 columns at width 100", []string{"1", "2", "3", "4", "5"}, 100, 20},
		{"5 columns at width 9", []string{"1", "2", "3", "4", "5"}, 9, 1},
		{"5 columns at width 0", []string{"1", "2", "3", "4", "5"}, 0, 0},
		{"5 columns at negative width", []string{"1", "2", "3", "4", "5"}, -10, 0},
		{"0 columns", []string{}, 100, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tbl := newCustomTable("test")
			tbl.Columns = tt.cols
			tbl.Width = tt.width
			got := tbl.columnWidth()
			if got != tt.expected {
				t.Fatalf("expected columnWidth %d, got %d", tt.expected, got)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	t.Run("Empty string at width -1, 0, 1, 2", func(t *testing.T) {
		widths := []int{-1, 0, 1, 2}
		for _, w := range widths {
			got := truncate("", w)
			if got != "" {
				t.Fatalf("truncate(\"\", %d) = %q; want \"\"", w, got)
			}
		}
	})

	t.Run("A short string that fits", func(t *testing.T) {
		s := "abc"
		got := truncate(s, 10)
		if got != "abc" {
			t.Fatalf("truncate(%q, 10) = %q; want %q", s, got, "abc")
		}
	})

	t.Run("A long string at width 10", func(t *testing.T) {
		s := "longstringhere"
		expected := s[:5] + "..."
		got := truncate(s, 10)
		if got != expected {
			t.Fatalf("truncate(%q, 10) = %q; want %q", s, got, expected)
		}
	})

	t.Run("Width 0 and width 1 on a non-empty string", func(t *testing.T) {
		s := "nonempty"
		expected := s[:1] + "..."
		for _, w := range []int{0, 1} {
			got := truncate(s, w)
			if got != expected {
				t.Fatalf("truncate(%q, %d) = %q; want %q", s, w, got, expected)
			}
		}
	})
}

func TestCustomTableView(t *testing.T) {
	widths := []int{0, 9}
	for _, w := range widths {
		t.Run(fmt.Sprintf("Width %d with empty cell does not panic", w), func(t *testing.T) {
			tbl := newCustomTable("test")
			tbl.Columns = []string{"Name", "Version", "Size", "Date", "Status"}
			tbl.addRow("curl", []string{"curl", "", "1MB", "2024-01-01", "installed"})
			tbl.Width = w
			tbl.Height = 10
			tbl.SetLoading(false)
			out := tbl.View()
			_ = out
		})
	}
}

func TestTablePackageLoadedDate(t *testing.T) {
	tblModel := newTable()
	p1 := backend.Package{Name: "pkg1"}
	updated, _ := tblModel.Update(PackageLoadedMsg{pkgs: p1})
	tblModel = updated.(tableModel)
	if got := tblModel.tables[2].Rows[0].Cells[4]; got != "unknown" {
		t.Fatalf("expected 'unknown' for zero date, got %q", got)
	}

	p2 := backend.Package{Name: "pkg2", Date: time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)}
	updated, _ = tblModel.Update(PackageLoadedMsg{pkgs: p2})
	tblModel = updated.(tableModel)
	if got := tblModel.tables[2].Rows[1].Cells[4]; got != "2024-05-01" {
		t.Fatalf("expected '2024-05-01', got %q", got)
	}
}

func TestRowIdentity(t *testing.T) {
	t.Run("same name across different managers creates separate rows", func(t *testing.T) {
		tblModel := newTable()
		p1 := backend.Package{Name: "typescript", DB: "npm", Version: "5.0.0"}
		p2 := backend.Package{Name: "typescript", DB: "pacman", Version: "5.0.0"}

		updated, _ := tblModel.Update(PackageLoadedMsg{pkgs: p1})
		tblModel = updated.(tableModel)
		updated, _ = tblModel.Update(PackageLoadedMsg{pkgs: p2})
		tblModel = updated.(tableModel)

		// (a) npm and pacman typescript both present in tables[2]
		allTable := tblModel.tables[2]
		if len(allTable.Rows) != 2 {
			t.Fatalf("expected 2 rows in tables[2], got %d", len(allTable.Rows))
		}
		foundNPM := false
		foundPacman := false
		for _, r := range allTable.Rows {
			if r.Cells[0] == "typescript" && r.Cells[2] == "npm" {
				foundNPM = true
			}
			if r.Cells[0] == "typescript" && r.Cells[2] == "pacman" {
				foundPacman = true
			}
		}
		if !foundNPM || !foundPacman {
			t.Fatalf("expected both npm and pacman typescript rows, got: %+v", allTable.Rows)
		}

		// (b) a re-sent pacman typescript with a NewVersion replaces its row (count unchanged, version cell updated)
		p3 := backend.Package{Name: "typescript", DB: "pacman", Version: "5.0.0", NewVersion: "5.1.0"}
		updated, _ = tblModel.Update(PackageLoadedMsg{pkgs: p3})
		tblModel = updated.(tableModel)

		allTable = tblModel.tables[2]
		if len(allTable.Rows) != 2 {
			t.Fatalf("expected row count unchanged (2), got %d", len(allTable.Rows))
		}
		var pacmanVersion string
		for _, r := range allTable.Rows {
			if r.Cells[0] == "typescript" && r.Cells[2] == "pacman" {
				pacmanVersion = r.Cells[1]
			}
		}
		if pacmanVersion != "5.0.0->5.1.0" {
			t.Fatalf("expected version '5.0.0->5.1.0', got %q", pacmanVersion)
		}

		// (c) that update style is registered for the pacman key and not for the npm key
		pacmanKey := RowIdentityKey("pacman", "typescript")
		npmKey := RowIdentityKey("npm", "typescript")
		if _, ok := allTable.StyledRows[pacmanKey]; !ok {
			t.Errorf("expected update style registered for pacman key %q", pacmanKey)
		}
		if _, ok := allTable.StyledRows[npmKey]; ok {
			t.Errorf("update style should not be registered for npm key %q", npmKey)
		}
	})

	t.Run("row replaced while filter is active shows new cells in Rows", func(t *testing.T) {
		// (d) on a bare customTable, a row replaced while a filter is active shows its new cells in Rows
		ct := newCustomTable("test")
		ct.Columns = []string{"Name", "Version", "Package Manager"}
		ct.addRow(RowIdentityKey("pacman", "typescript"), []string{"typescript", "5.0.0", "pacman"})
		ct.addRow(RowIdentityKey("pacman", "curl"), []string{"curl", "8.0.0", "pacman"})

		ct.filterColumn("Name", "typescript")
		if len(ct.Rows) != 1 || ct.Rows[0].Cells[1] != "5.0.0" {
			t.Fatalf("expected filtered to 1 row with version 5.0.0, got %+v", ct.Rows)
		}

		// Re-add / replace typescript with updated version
		ct.addRow(RowIdentityKey("pacman", "typescript"), []string{"typescript", "5.1.0", "pacman"})

		if len(ct.Rows) != 1 {
			t.Fatalf("expected 1 row in Rows, got %d", len(ct.Rows))
		}
		if ct.Rows[0].Cells[1] != "5.1.0" {
			t.Fatalf("expected updated cells '5.1.0' in Rows, got %q", ct.Rows[0].Cells[1])
		}
		if len(ct.OriginalRows) != 2 {
			t.Fatalf("expected 2 rows in OriginalRows, got %d", len(ct.OriginalRows))
		}
	})
}
