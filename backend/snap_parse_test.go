package backend

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func readSnapFixture(t *testing.T, name string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", "snap", name))
	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}

func TestParseSnapList(t *testing.T) {
	want := []Package{
		{Name: "core22", Version: "20240111", IsDirect: false, DB: "snap"},
		{Name: "firefox", Version: "131.0.2-1", IsDirect: true, DB: "snap"},
		{Name: "gnome-42-2204", Version: "0+git.510a2d3", IsDirect: false, DB: "snap"},
		{Name: "gtk-common-themes", Version: "0.1-81-g442e511", IsDirect: false, DB: "snap"},
		{Name: "my-tool", Version: "x1", IsDirect: true, DB: "snap"},
		{Name: "snapd", Version: "2.63", IsDirect: false, DB: "snap"},
		{Name: "code", Version: "1.93.1", IsDirect: true, DB: "snap"},
	}

	got := parseSnapList(readSnapFixture(t, "list.txt"))
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}

	if got := parseSnapList(""); len(got) != 0 {
		t.Errorf("empty input: got %+v", got)
	}
}

func TestParseSnapRefreshList(t *testing.T) {
	want := map[string]string{"firefox": "132.0", "core22": "20241001"}

	got := parseSnapRefreshList(readSnapFixture(t, "refresh-list.txt"))
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// The snapd held-row format is unverified; this case is written from the parser.
func TestParseSnapHeld(t *testing.T) {
	want := map[string]bool{"core22": true}

	got := parseSnapHeld(readSnapFixture(t, "refresh-list.txt"))
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestIsDirectInstall(t *testing.T) {
	tests := []struct {
		name, notes string
		want        bool
	}{
		{"core22", "base", false},
		{"snapd", "snapd", false},
		{"gnome-42-2204", "-", false},
		{"kde-frameworks-5", "-", false},
		{"gtk-common-themes", "-", false},
		{"firefox", "-", true},
		{"code", "classic", true},
		{"Firefox", "", true},
		{"anything", "CORE", false},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/"+tt.notes, func(t *testing.T) {
			if got := isDirectInstall(tt.name, tt.notes); got != tt.want {
				t.Errorf("isDirectInstall(%q, %q) = %v, want %v", tt.name, tt.notes, got, tt.want)
			}
		})
	}
}
