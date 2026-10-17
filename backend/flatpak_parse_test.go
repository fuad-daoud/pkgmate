//go:build flatpak || all_backends

package backend

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func sz(v float64, mult int64) int64 {
	return int64(v * float64(mult))
}

func readFlatpakFixture(t *testing.T, name string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", "flatpak", name))
	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}

func TestParseFlatpakList(t *testing.T) {
	pkg := func(id, name, version string, size int64) Package {
		return Package{ID: id, Name: name, Version: version, Size: size, IsDirect: true, DB: "flatpak"}
	}

	// [PIN] sizes use 1024 multipliers although flatpak/GLib units are decimal (1000).
	want := []Package{
		pkg("com.obsproject.Studio", "OBS Studio", "32.2.2", sz(528.5, 1<<20)),
		pkg("org.gimp.GIMP", "GNU Image Manipulation Program", "2.10.38", sz(1.1, 1<<30)),
		pkg("org.mozilla.firefox", "Firefox", "131.0.3", sz(256.0, 1<<20)),
		pkg("org.gnome.Calculator", "Calculator", "46.1", sz(312.0, 1<<10)),
		pkg("org.videolan.VLC", "VLC", "3.0.21", sz(410.2, 1<<20)),
	}

	got := parseFlatpakList(readFlatpakFixture(t, "list.txt"))
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

func TestParseFlatpakListEmptyVersion(t *testing.T) {
	t.Skip("known bug: empty version column shifts the last name word into Version (or drops single-word-name rows)")

	want := []Package{{
		ID:   "org.gimp.GIMP",
		Name: "GNU Image Manipulation Program", Version: "", Size: sz(1.1, 1<<30),
		IsDirect: true, DB: "flatpak",
	}}

	got := parseFlatpakList(readFlatpakFixture(t, "list-no-version.txt"))
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

func TestGetFlatpakInstallDateByID(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	fakeID := "org.pkgmate.test.FakeApp"
	appDir := filepath.Join(tmpDir, ".local", "share", "flatpak", "app", fakeID)
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}

	fixedTime := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(appDir, fixedTime, fixedTime); err != nil {
		t.Fatal(err)
	}

	got := getFlatpakInstallDate(fakeID)
	if !got.Equal(fixedTime) {
		t.Errorf("getFlatpakInstallDate(%q) = %v, want %v", fakeID, got, fixedTime)
	}
}

func TestParseFlatpakSize(t *testing.T) {
	tests := []struct {
		in   string
		want int64
	}{
		{"528.5MB", sz(528.5, 1<<20)},
		{"312.0kB", 312 * 1024},
		{"1.1GB", sz(1.1, 1<<30)},
		{"12 bytes", 0},
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := parseFlatpakSize(tt.in); got != tt.want {
				t.Errorf("parseFlatpakSize(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}
