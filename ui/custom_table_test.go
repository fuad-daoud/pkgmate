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
		table.addRow([]string{"curl", "7.0", "1MB"})
		table.addRow([]string{"wget", "1.0", "2MB"})
		table.addRow([]string{"git", "2.0", "5MB"})

		table.filterColumn("Name", "curl")
		if len(table.Rows) != 1 || table.Rows[0][0] != "curl" {
			t.Fatalf("expected 1 row with 'curl', got %v", table.Rows)
		}
	})

	t.Run("Match on the last column", func(t *testing.T) {
		table := newCustomTable("test")
		table.Columns = []string{"Name", "Version", "Size"}
		table.addRow([]string{"curl", "7.0", "1MB"})
		table.addRow([]string{"wget", "1.0", "2MB"})
		table.addRow([]string{"git", "2.0", "5MB"})

		table.filterColumn("Size", "5mb")
		if len(table.Rows) != 1 || table.Rows[0][0] != "git" {
			t.Fatalf("expected 1 row with 'git', got %v", table.Rows)
		}
	})

	t.Run("No matches gives empty Rows", func(t *testing.T) {
		table := newCustomTable("test")
		table.Columns = []string{"Name", "Version", "Size"}
		table.addRow([]string{"curl", "7.0", "1MB"})
		table.addRow([]string{"wget", "1.0", "2MB"})

		table.filterColumn("Name", "nonexistent")
		if len(table.Rows) != 0 {
			t.Fatalf("expected empty Rows, got %v", table.Rows)
		}
	})

	t.Run("An empty term matches all rows", func(t *testing.T) {
		table := newCustomTable("test")
		table.Columns = []string{"Name", "Version", "Size"}
		table.addRow([]string{"curl", "7.0", "1MB"})
		table.addRow([]string{"wget", "1.0", "2MB"})
		table.addRow([]string{"git", "2.0", "5MB"})

		table.filterColumn("Name", "")
		if len(table.Rows) != 3 {
			t.Fatalf("expected 3 rows, got %d", len(table.Rows))
		}
	})

	t.Run("An unknown column name is a no-op", func(t *testing.T) {
		table := newCustomTable("test")
		table.Columns = []string{"Name", "Version", "Size"}
		table.addRow([]string{"curl", "7.0", "1MB"})
		table.addRow([]string{"wget", "1.0", "2MB"})
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
		table.addRow([]string{"curl", "7.0", "1MB"})
		table.addRow([]string{"wget", "1.0", "2MB"})
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
		table.addRow([]string{"curl", "7.0", "1MB"})
		table.addRow([]string{"short"})
		table.addRow([]string{"wget", "1.0", "2MB"})

		table.filterColumn("Size", "mb")
		if len(table.Rows) != 2 {
			t.Fatalf("expected 2 rows, got %d (%v)", len(table.Rows), table.Rows)
		}
		if table.Rows[0][0] != "curl" || table.Rows[1][0] != "wget" {
			t.Fatalf("expected curl and wget, got %v", table.Rows)
		}
	})

	t.Run("Case-insensitivity: an upper-case cell matches a lower-case term", func(t *testing.T) {
		table := newCustomTable("test")
		table.Columns = []string{"Name", "Version", "Size"}
		table.addRow([]string{"RipGrep", "13.0", "3MB"})
		table.addRow([]string{"curl", "7.0", "1MB"})

		table.filterColumn("Name", "ripgrep")
		if len(table.Rows) != 1 || table.Rows[0][0] != "RipGrep" {
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
			tbl.addRow([]string{"curl", "", "1MB", "2024-01-01", "installed"})
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
	if got := tblModel.tables[2].Rows[0][4]; got != "unknown" {
		t.Fatalf("expected 'unknown' for zero date, got %q", got)
	}

	p2 := backend.Package{Name: "pkg2", Date: time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)}
	updated, _ = tblModel.Update(PackageLoadedMsg{pkgs: p2})
	tblModel = updated.(tableModel)
	if got := tblModel.tables[2].Rows[1][4]; got != "2024-05-01" {
		t.Fatalf("expected '2024-05-01', got %q", got)
	}
}
