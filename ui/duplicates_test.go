package ui

import (
	"strings"
	"testing"

	"pkgmate/backend"
)

// newLoadedModel builds a table model with the shared columns set and every
// package fed through PackageLoadedMsg. It never calls Init, so Program is
// untouched.
func newLoadedModel(t *testing.T, pkgs ...backend.Package) tableModel {
	t.Helper()
	m := newTable()
	if got := len(m.tables); got != 5 {
		t.Fatalf("expected 5 tables, got %d", got)
	}
	updated, _ := m.Update(TableInitEvent{})
	m, ok := updated.(tableModel)
	if !ok {
		t.Fatalf("expected tableModel, got %T", updated)
	}
	for _, pkg := range pkgs {
		updated, _ = m.Update(PackageLoadedMsg{pkgs: pkg})
		m = updated.(tableModel)
	}
	updated, _ = m.Update(DisplayResizeEvent{width: 200, height: 40})
	return updated.(tableModel)
}

// cell finds the row with the given key and returns its cells.
func cell(t *testing.T, rows []Row, key string) []string {
	t.Helper()
	for _, row := range rows {
		if row.Key == key {
			return row.Cells
		}
	}
	t.Fatalf("no row with key %q in %v", key, rows)
	return nil
}

// pacmanFirefox and flatpakFirefox form a duplicate group: both key to
// app:firefox, the pacman one is direct, so the group has two families.
func pacmanFirefox() backend.Package {
	return backend.Package{Name: "firefox", Version: "1.0", Size: 1024, DB: "pacman", IsDirect: true}
}

func flatpakFirefox() backend.Package {
	return backend.Package{Name: "Firefox", ID: "org.mozilla.firefox", Version: "2.0", Size: 2048, DB: "flatpak", IsDirect: true}
}

func TestDuplicatesTabFromFinishedLoad(t *testing.T) {
	m := newLoadedModel(t,
		pacmanFirefox(),
		flatpakFirefox(),
		backend.Package{Name: "curl", Version: "8.0", Size: 512, DB: "pacman", IsDirect: true},
	)
	updated, _ := m.Update(FinishedLoadingPackagesMsg{})
	m = updated.(tableModel)

	duplicates := m.tables[duplicatesTabIndex]
	if len(duplicates.Rows) != 2 {
		t.Fatalf("expected 2 duplicate rows, got %d: %v", len(duplicates.Rows), duplicates.Rows)
	}
	firefoxPacman := RowIdentityKey("pacman", "firefox")
	firefoxFlatpak := RowIdentityKey("flatpak", "Firefox")
	if !rowsAdjacent(duplicates.Rows, firefoxPacman, firefoxFlatpak) {
		t.Fatalf("expected the two firefox rows adjacent, got %v", duplicates.Rows)
	}

	all := m.tables[2]
	got := cell(t, all.Rows, firefoxPacman)[managerColumnIndex]
	if want := "pacman (+flatpak)"; got != want {
		t.Errorf("pacman manager cell = %q, want %q", got, want)
	}
	// A package outside any group is untouched.
	if got := cell(t, all.Rows, RowIdentityKey("pacman", "curl"))[managerColumnIndex]; got != "pacman" {
		t.Errorf("ungrouped manager cell = %q, want %q", got, "pacman")
	}
}

func rowsAdjacent(rows []Row, a, b string) bool {
	for i := 0; i+1 < len(rows); i++ {
		if rows[i].Key == a && rows[i+1].Key == b {
			return true
		}
		if rows[i].Key == b && rows[i+1].Key == a {
			return true
		}
	}
	return false
}

func TestApplyDuplicatesGroupOrderAndMarker(t *testing.T) {
	zedPacman := backend.Package{Name: "zed", DB: "pacman", IsDirect: true}
	zedFlatpak := backend.Package{Name: "zed", DB: "flatpak", IsDirect: true}
	antPacman := backend.Package{Name: "ant", DB: "pacman", IsDirect: true}
	antBrew := backend.Package{Name: "ant", DB: "brew", IsDirect: true}
	antNpm := backend.Package{Name: "ant", DB: "npm", IsDirect: true}

	m := newLoadedModel(t, zedPacman, zedFlatpak, antPacman, antBrew, antNpm)

	m.applyDuplicates([]backend.DuplicateGroup{
		{Key: "app:zed", Packages: []backend.Package{zedFlatpak, zedPacman}},
		{Key: "app:ant", Packages: []backend.Package{antBrew, antNpm, antPacman}},
	})

	rows := m.tables[duplicatesTabIndex].Rows
	wantOrder := []string{
		RowIdentityKey("flatpak", "zed"),
		RowIdentityKey("pacman", "zed"),
		RowIdentityKey("brew", "ant"),
		RowIdentityKey("npm", "ant"),
		RowIdentityKey("pacman", "ant"),
	}
	if len(rows) != len(wantOrder) {
		t.Fatalf("expected %d rows, got %d: %v", len(wantOrder), len(rows), rows)
	}
	for i, key := range wantOrder {
		if rows[i].Key != key {
			t.Fatalf("row %d = %q, want %q (full order %v)", i, rows[i].Key, key, rows)
		}
	}

	// The three-family group marks each package with the other families in
	// alphabetical order (flag 2).
	for _, tc := range []struct {
		key  string
		want string
	}{
		{RowIdentityKey("pacman", "ant"), "pacman (+brew, npm)"},
		{RowIdentityKey("brew", "ant"), "brew (+npm, pacman)"},
		{RowIdentityKey("npm", "ant"), "npm (+brew, pacman)"},
		{RowIdentityKey("pacman", "zed"), "pacman (+flatpak)"},
	} {
		if got := cell(t, m.tables[2].Rows, tc.key)[managerColumnIndex]; got != tc.want {
			t.Errorf("marker for %q = %q, want %q", tc.key, got, tc.want)
		}
	}

	// The Duplicates tab itself keeps the plain DB in the Manager cell.
	if got := cell(t, rows, RowIdentityKey("pacman", "ant"))[managerColumnIndex]; got != "pacman" {
		t.Errorf("duplicates tab manager cell = %q, want %q", got, "pacman")
	}

	if len(m.duplicates) != 2 {
		t.Errorf("expected 2 groups on the model, got %d", len(m.duplicates))
	}
}

func TestDuplicatesTabEmptyStateAndLoading(t *testing.T) {
	m := newLoadedModel(t, pacmanFirefox(), flatpakFirefox())
	duplicates := &m.tables[duplicatesTabIndex]
	duplicates.Columns = m.columns
	duplicates.Width = 120
	duplicates.Height = 20

	if got := duplicates.View(); !strings.Contains(got, "Loading packages...") {
		t.Errorf("before the finish message the Duplicates tab should show the loading text, got:\n%s", got)
	}

	updated, _ := m.Update(FinishedLoadingPackagesMsg{})
	m = updated.(tableModel)
	if len(m.duplicates) != 1 {
		t.Fatalf("expected 1 group, got %d", len(m.duplicates))
	}

	// A load with no groups at all.
	empty := newLoadedModel(t, backend.Package{Name: "curl", DB: "pacman", IsDirect: true})
	updated, _ = empty.Update(FinishedLoadingPackagesMsg{})
	empty = updated.(tableModel)

	duplicates = &empty.tables[duplicatesTabIndex]
	duplicates.Columns = empty.columns
	duplicates.Width = 120
	duplicates.Height = 20
	if got := duplicates.View(); !strings.Contains(got, "No duplicates found") {
		t.Errorf("expected %q in:\n%s", "No duplicates found", got)
	}

	// Groups present but a search that matches nothing keeps the usual text.
	filtered := newLoadedModel(t, pacmanFirefox(), flatpakFirefox())
	updated, _ = filtered.Update(FinishedLoadingPackagesMsg{})
	filtered = updated.(tableModel)
	filtered.tables[duplicatesTabIndex].filterColumn("Name", "nothing-matches-this")
	got := filtered.tables[duplicatesTabIndex].View()
	if !strings.Contains(got, "No results found") {
		t.Errorf("expected %q in:\n%s", "No results found", got)
	}
	if strings.Contains(got, "No duplicates found") {
		t.Errorf("a filtered table should not show the empty state, got:\n%s", got)
	}
}

func TestFooterDuplicateCount(t *testing.T) {
	m := newLoadedModel(t, pacmanFirefox(), flatpakFirefox())
	m.tables[0].Width = 240
	if got := m.footerView(); strings.Contains(got, "installed more than once") {
		t.Errorf("no groups means no footer count, got:\n%s", got)
	}

	updated, _ := m.Update(FinishedLoadingPackagesMsg{})
	m = updated.(tableModel)
	if got := m.footerView(); !strings.Contains(got, "1 app installed more than once") {
		t.Errorf("expected %q in:\n%s", "1 app installed more than once", got)
	}

	two := newLoadedModel(t,
		pacmanFirefox(), flatpakFirefox(),
		backend.Package{Name: "curl", DB: "pacman", IsDirect: true},
		backend.Package{Name: "curl", DB: "brew", IsDirect: true},
	)
	two.tables[0].Width = 240
	updated, _ = two.Update(FinishedLoadingPackagesMsg{})
	two = updated.(tableModel)
	if got := two.footerView(); !strings.Contains(got, "2 apps installed more than once") {
		t.Errorf("expected %q in:\n%s", "2 apps installed more than once", got)
	}
}

func TestMarkerSurvivesManagerFilter(t *testing.T) {
	m := newLoadedModel(t, pacmanFirefox(), flatpakFirefox())
	updated, _ := m.Update(FinishedLoadingPackagesMsg{})
	m = updated.(tableModel)

	all := &m.tables[2]
	// Both marked cells mention "pacman" (flag 9), so the assertion is that the
	// pacman row is still matched, not that it is the only one.
	all.filterColumn("Package Manager", "pacman")
	if len(all.Rows) == 0 {
		t.Fatalf("expected the pacman row to survive the filter, got %v", all.Rows)
	}
	if got := cell(t, all.Rows, RowIdentityKey("pacman", "firefox"))[managerColumnIndex]; got != "pacman (+flatpak)" {
		t.Errorf("filtered manager cell = %q, want %q", got, "pacman (+flatpak)")
	}

	// Flag 9: the marker text also makes the pacman row match the other manager.
	all.filterColumn("Package Manager", "flatpak")
	if len(all.Rows) != 2 {
		t.Errorf("expected both rows to match %q, got %v", "flatpak", all.Rows)
	}
}

func TestMarkerVisibleInFilteredRows(t *testing.T) {
	pacman := backend.Package{Name: "firefox", DB: "pacman", IsDirect: true}
	flatpak := backend.Package{Name: "Firefox", ID: "org.mozilla.firefox", DB: "flatpak", IsDirect: true}

	m := newLoadedModel(t, pacman, flatpak)

	all := &m.tables[2]
	all.filterColumn("Package Manager", "pacman")
	before := cell(t, all.Rows, RowIdentityKey("pacman", "firefox"))[managerColumnIndex]
	if before != "pacman" {
		t.Fatalf("precondition: manager cell = %q, want %q", before, "pacman")
	}

	m.applyDuplicates([]backend.DuplicateGroup{{
		Key:      "app:firefox",
		Packages: []backend.Package{flatpak, pacman},
	}})

	after := cell(t, all.Rows, RowIdentityKey("pacman", "firefox"))[managerColumnIndex]
	if after != "pacman (+flatpak)" {
		t.Errorf("marker applied under an active filter must be visible in Rows, got %q", after)
	}
}
