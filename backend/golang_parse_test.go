//go:build golang || all_backends

package backend

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"
)

func TestGoNameVersion(t *testing.T) {
	tests := []struct {
		name        string
		fixture     string
		fileName    string
		wantName    string
		wantVersion string
	}{
		{"tagged module binary", "gopls.buildinfo", "gopls", "golang.org/x/tools/gopls", "v0.18.1"},
		{"devel version is unknown", "devel.buildinfo", "tool", "example.com/tool", "unknown"},
		{"empty main path uses file name", "command-line-arguments.buildinfo", "mytool", "mytool", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata", "golang", tt.fixture))
			if err != nil {
				t.Fatal(err)
			}

			bi, err := debug.ParseBuildInfo(string(data))
			if err != nil {
				t.Fatalf("ParseBuildInfo: %v", err)
			}

			gotName, gotVersion := goNameVersion(bi, tt.fileName)
			if gotName != tt.wantName || gotVersion != tt.wantVersion {
				t.Errorf("goNameVersion = (%q, %q), want (%q, %q)", gotName, gotVersion, tt.wantName, tt.wantVersion)
			}
		})
	}
}
