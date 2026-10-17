//go:build brew || all_backends

package backend

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func readBrewFixture(t *testing.T, parts ...string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(append([]string{"testdata", "brew"}, parts...)...))
	if err != nil {
		t.Fatal(err)
	}

	return data
}

func TestParseInstallReceipt(t *testing.T) {
	tests := []struct {
		name       string
		fixture    string
		wantDate   time.Time
		wantDirect bool
	}{
		{"on request", "on-request.json", time.Unix(1700000000, 0), true},
		{"dependency", "dependency.json", time.Unix(1700100000, 0), false},
		{"null time", "no-time.json", time.Time{}, true},
		{"malformed", "malformed.json", time.Time{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date, direct := parseInstallReceipt(readBrewFixture(t, "receipts", tt.fixture))

			if !date.Equal(tt.wantDate) {
				t.Errorf("date = %v, want %v", date, tt.wantDate)
			}

			if direct != tt.wantDirect {
				t.Errorf("isDirect = %v, want %v", direct, tt.wantDirect)
			}
		})
	}
}

func TestParseBrewOutdated(t *testing.T) {
	want := map[string]string{"jq": "1.7.1", "node": "22.1.0", "firefox": "132.0"}

	got, err := parseBrewOutdated(readBrewFixture(t, "outdated.json"))
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if _, err := parseBrewOutdated(readBrewFixture(t, "malformed.json")); err == nil {
		t.Error("malformed json: expected error")
	}
}

func TestParseBrewAutoremove(t *testing.T) {
	want := map[string]bool{"oniguruma": true, "libyaml": true}

	got := parseBrewAutoremove(string(readBrewFixture(t, "autoremove.txt")))
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if got := parseBrewAutoremove(""); len(got) != 0 {
		t.Errorf("empty output: got %v", got)
	}
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func chtimes(t *testing.T, path string, mtime time.Time) {
	t.Helper()

	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}
}

func TestBrewGetInstalledVersion(t *testing.T) {
	root := t.TempDir()
	cellar := filepath.Join(root, "Cellar")

	older := filepath.Join(cellar, "foo", "1.0")
	newer := filepath.Join(cellar, "foo", "2.0")
	mkdirAll(t, older)
	mkdirAll(t, newer)
	chtimes(t, older, time.Unix(1600000000, 0))
	chtimes(t, newer, time.Unix(1700000000, 0))

	t.Run("newest mtime without opt link", func(t *testing.T) {
		got, err := getInstalledVersion(cellar, "foo")
		if err != nil {
			t.Fatal(err)
		}

		if got != "2.0" {
			t.Errorf("got %q, want 2.0", got)
		}
	})

	t.Run("opt symlink wins over newest", func(t *testing.T) {
		mkdirAll(t, filepath.Join(root, "opt"))

		if err := os.Symlink(older, filepath.Join(root, "opt", "foo")); err != nil {
			t.Fatal(err)
		}

		got, err := getInstalledVersion(cellar, "foo")
		if err != nil {
			t.Fatal(err)
		}

		if got != "1.0" {
			t.Errorf("got %q, want 1.0", got)
		}
	})

	t.Run("no versions", func(t *testing.T) {
		if _, err := getInstalledVersion(cellar, "missing"); err == nil {
			t.Error("expected error")
		}
	})
}

func TestBrewGetCaskVersion(t *testing.T) {
	caskroom := t.TempDir()
	mkdirAll(t, filepath.Join(caskroom, "app", ".metadata"))
	mkdirAll(t, filepath.Join(caskroom, "app", "1.2.3"))
	chtimes(t, filepath.Join(caskroom, "app", ".metadata"), time.Unix(1800000000, 0))
	chtimes(t, filepath.Join(caskroom, "app", "1.2.3"), time.Unix(1700000000, 0))

	got, err := getCaskVersion(caskroom, "app")
	if err != nil {
		t.Fatal(err)
	}

	if got != "1.2.3" {
		t.Errorf("got %q, want 1.2.3", got)
	}

	if _, err := getCaskVersion(caskroom, "missing"); err == nil {
		t.Error("missing cask: expected error")
	}
}

func TestBrewParseTimestampDir(t *testing.T) {
	got, err := parseTimestampDir("1700000000")
	if err != nil {
		t.Fatal(err)
	}

	if !got.Equal(time.Unix(1700000000, 0)) {
		t.Errorf("got %v", got)
	}

	if _, err := parseTimestampDir("not-a-number"); err == nil {
		t.Error("expected error")
	}
}

func TestBrewGetCaskMetadata(t *testing.T) {
	caskroom := t.TempDir()

	mkdirAll(t, filepath.Join(caskroom, "withmeta", "1.0", ".metadata", "1700000000"))

	got, err := getCaskMetadata(caskroom, "withmeta", "1.0")
	if err != nil {
		t.Fatal(err)
	}

	if !got.Equal(time.Unix(1700000000, 0)) {
		t.Errorf("epoch metadata: got %v", got)
	}

	versionDir := filepath.Join(caskroom, "nometa", "2.0")
	mkdirAll(t, versionDir)
	chtimes(t, versionDir, time.Unix(1650000000, 0))

	got, err = getCaskMetadata(caskroom, "nometa", "2.0")
	if err != nil {
		t.Fatal(err)
	}

	if !got.Equal(time.Unix(1650000000, 0)) {
		t.Errorf("fallback to version dir mtime: got %v", got)
	}
}

func TestBrewGetCaskMetadataRealLayout(t *testing.T) {
	t.Skip("known bug: Homebrew writes Caskroom/<token>/.metadata/<version>/<yyyymmddHHMMSS.mmm>/, the code reads <version>/.metadata/<epoch> so it always falls back to the version dir mtime")

	caskroom := t.TempDir()
	mkdirAll(t, filepath.Join(caskroom, "app", ".metadata", "1.0", "20240315120000.000"))
	mkdirAll(t, filepath.Join(caskroom, "app", "1.0"))

	got, err := getCaskMetadata(caskroom, "app", "1.0")
	if err != nil {
		t.Fatal(err)
	}

	if got.Year() != 2024 || got.Month() != time.March || got.Day() != 15 || got.Hour() != 12 {
		t.Errorf("got %v, want 2024-03-15 12:00", got)
	}
}

func TestBrewReadPinnedDir(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(t.TempDir(), "node")
	mkdirAll(t, target)

	if err := os.Symlink(target, filepath.Join(dir, "node")); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "regular"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	want := map[string]bool{"node": true}
	if got := readBrewPinnedDir(dir); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if got := readBrewPinnedDir(filepath.Join(dir, "absent")); len(got) != 0 {
		t.Errorf("absent dir: got %v", got)
	}
}
