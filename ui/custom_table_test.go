package ui

import (
	"reflect"
	"testing"
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
