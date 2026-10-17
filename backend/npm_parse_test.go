//go:build npm || all_backends

package backend

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseNpmList(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		want    map[string]npmPackage
		wantErr bool
	}{
		{
			name:    "global list with scoped, linked and unversioned entries",
			fixture: "list-global.json",
			want: map[string]npmPackage{
				"@anthropic-ai/claude-code": {Version: "2.0.14"},
				"corepack":                  {Version: "0.34.0"},
				"my-linked-tool":            {Version: "1.0.0"},
				"npm":                       {Version: "11.6.1"},
				"unversioned-pkg":           {Version: ""},
			},
		},
		{name: "no dependencies key", fixture: "list-global-empty.json", want: nil},
		{name: "malformed json", fixture: "list-global-malformed.json", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata", "npm", tt.fixture))
			if err != nil {
				t.Fatal(err)
			}

			got, err := parseNpmList(data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}

			if len(got) == 0 && len(tt.want) == 0 {
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
