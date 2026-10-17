//go:build pip || all_backends

package backend

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func readPipFixture(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", "pip", name))
	if err != nil {
		t.Fatal(err)
	}

	return data
}

func sortPipPackages(pkgs []pipPackage) {
	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].Name < pkgs[j].Name })
}

func TestParsePipShowLocation(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"editable line does not match", string(readPipFixture(t, "show-pip.txt")), "/usr/lib/python3.12/site-packages"},
		{"only editable line", "Editable project location: /x\n", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parsePipShowLocation(tt.in); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParsePipList(t *testing.T) {
	want := []pipPackage{
		{Name: "PyYAML", Version: "6.0.1"},
		{Name: "requests", Version: "2.31.0"},
		{Name: "my-editable", Version: "0.1.0"},
		{Name: "pip", Version: "24.0"},
	}

	got, err := parsePipList(readPipFixture(t, "list.json"))
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}

	if _, err := parsePipList(readPipFixture(t, "malformed.json")); err == nil {
		t.Error("malformed json: expected error")
	}
}

func TestParsePipOutdated(t *testing.T) {
	want := map[string]string{"requests": "2.32.3", "PyYAML": "6.0.2"}

	got, err := parsePipOutdated(readPipFixture(t, "outdated.json"))
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if _, err := parsePipOutdated(readPipFixture(t, "malformed.json")); err == nil {
		t.Error("malformed json: expected error")
	}
}

func TestParsePipxListJSON(t *testing.T) {
	want := []pipPackage{{Name: "black", Version: "24.2.0"}, {Name: "httpie", Version: "3.2.2"}}

	got, err := parsePipxListJSON(readPipFixture(t, "pipx-list.json"))
	if err != nil {
		t.Fatal(err)
	}

	sortPipPackages(got)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}

	if _, err := parsePipxListJSON(readPipFixture(t, "malformed.json")); err == nil {
		t.Error("malformed json: expected error")
	}
}

func TestParsePipxListPlain(t *testing.T) {
	// [PIN] the version is on the line but the parser reports "unknown".
	want := []pipPackage{{Name: "black", Version: "unknown"}, {Name: "httpie", Version: "unknown"}}

	got := parsePipxListPlain(string(readPipFixture(t, "pipx-list.txt")))
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestFindDistInfoDir(t *testing.T) {
	sp := filepath.Join("testdata", "pip", "site-packages")

	tests := []struct {
		name, pkg, version string
		want               string
	}{
		{"exact match", "requests", "2.31.0", "requests-2.31.0.dist-info"},
		{"case-insensitive match", "PyYAML", "6.0.1", "PyYAML-6.0.1.dist-info"},
		{"module without package dir", "typing_extensions", "4.9.0", "typing_extensions-4.9.0.dist-info"},
		{"absent package", "nonexistent", "1.0", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findDistInfoDir(sp, tt.pkg, tt.version)

			want := ""
			if tt.want != "" {
				want = filepath.Join(sp, tt.want)
			}

			if got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}

	t.Run("hyphen name with underscore dist-info", func(t *testing.T) {
		t.Skip("known bug: python-dateutil is not matched to python_dateutil-*.dist-info, so the package is dropped")

		want := filepath.Join(sp, "python_dateutil-2.9.0.dist-info")
		if got := findDistInfoDir(sp, "python-dateutil", "2.9.0"); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestProcessPipPackage(t *testing.T) {
	sp := filepath.Join("testdata", "pip", "site-packages")

	tests := []struct {
		name, pkg, version string
		found              bool
	}{
		{"exact match", "requests", "2.31.0", true},
		{"case-insensitive match", "PyYAML", "6.0.1", true},
		{"module without package dir", "typing_extensions", "4.9.0", true},
		{"absent package", "nonexistent", "1.0", false},
		{"distro installer skipped", "debian_pkg", "1.0.0", false},
		{"missing installer skipped", "no_installer_pkg", "1.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processPipPackage(tt.pkg, tt.version, sp)

			if !tt.found {
				if !reflect.DeepEqual(got, Package{}) {
					t.Errorf("got %+v, want zero Package", got)
				}

				return
			}

			if got.Name != tt.pkg || got.Version != tt.version || got.DB != "pip" || !got.IsDirect {
				t.Errorf("unexpected package: %+v", got)
			}

			if got.Size <= 0 {
				t.Errorf("Size = %d, want > 0", got.Size)
			}

			if got.Date.IsZero() {
				t.Error("Date is zero")
			}
		})
	}

	t.Run("hyphen name with underscore dist-info", func(t *testing.T) {
		t.Skip("known bug: python-dateutil is dropped because its dist-info uses underscores")

		got := processPipPackage("python-dateutil", "2.9.0", sp)
		if got.Name != "python-dateutil" {
			t.Errorf("got %+v, want python-dateutil", got)
		}
	})
}
