//go:build all_backends

package backend

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"strings"
	"testing"
)

func TestDuplicateDetectionE2E(t *testing.T) {
	var pkgs []Package

	// 1. pacman
	for _, p := range parsePacmanQi(readPacmanFixture(t, "qi.txt")) {
		p.DB = "pacman"
		pkgs = append(pkgs, p)
	}

	// 2. dpkg
	autoMap := parseAptExtendedStates(readDpkgFixture(t, "extended_states"))
	for _, p := range parseDpkgStatus(readDpkgFixture(t, "status")) {
		p.DB = "apt/dpkg"
		_, isAuto := autoMap[p.Name]
		p.IsDirect = !isAuto
		pkgs = append(pkgs, p)
	}

	// 3. snap
	pkgs = append(pkgs, parseSnapList(readSnapFixture(t, "list.txt"))...)

	// 4. flatpak
	pkgs = append(pkgs, parseFlatpakList(readFlatpakFixture(t, "list.txt"))...)

	// 5. npm
	npmData, err := os.ReadFile(filepath.Join("testdata", "npm", "list-global.json"))
	if err != nil {
		t.Fatal(err)
	}
	npmMap, err := parseNpmList(npmData)
	if err != nil {
		t.Fatal(err)
	}
	for name, meta := range npmMap {
		pkgs = append(pkgs, Package{
			Name:     name,
			Version:  meta.Version,
			DB:       "npm",
			IsDirect: true,
		})
	}

	// 6. pip & pipx
	pipList, err := parsePipList(readPipFixture(t, "list.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range pipList {
		pkgs = append(pkgs, Package{
			Name:     p.Name,
			Version:  p.Version,
			DB:       "pip",
			IsDirect: true,
		})
	}

	pipxList, err := parsePipxListJSON(readPipFixture(t, "pipx-list.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range pipxList {
		pkgs = append(pkgs, Package{
			Name:     p.Name,
			Version:  p.Version,
			DB:       "pipx",
			IsDirect: true,
		})
	}

	// 7. golang
	goEntries, err := filepath.Glob(filepath.Join("testdata", "golang", "*.buildinfo"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range goEntries {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		bi, err := debug.ParseBuildInfo(string(data))
		if err != nil {
			t.Fatalf("ParseBuildInfo(%s): %v", path, err)
		}
		base := strings.TrimSuffix(filepath.Base(path), ".buildinfo")
		name, ver := goNameVersion(bi, base)
		pkgs = append(pkgs, Package{
			Name:     name,
			Version:  ver,
			DB:       "golang",
			IsDirect: true,
		})
	}

	groups := FindDuplicates(pkgs)

	// The exact set of groups the fixtures must produce, keyed by the
	// representative key (the lexicographically smallest key shared by two or
	// more members). Members are listed in the order FindDuplicates sorts them:
	// by DB, then Name.
	want := []struct {
		key     string
		members []string
	}{
		{"app:vlc", []string{"flatpak:VLC", "pacman:vlc"}},
		{"app:firefox", []string{"flatpak:Firefox", "snap:firefox"}},
		{"app:gopls", []string{"golang:golang.org/x/tools/gopls", "pacman:gopls"}},
		{"app:bash", []string{"apt/dpkg:bash", "pacman:bash"}},
		{"py:black", []string{"pacman:python-black", "pipx:black"}},
		{"py:requests", []string{"apt/dpkg:python3-requests", "pip:requests"}},
	}

	if len(groups) != len(want) {
		t.Fatalf("got %d groups, want %d:\n got  %s\n want %s",
			len(groups), len(want), dumpGroups(groups), dumpWant(want))
	}

	for i, w := range want {
		g := groups[i]
		if g.Key != w.key {
			t.Errorf("group %d: Key = %q, want %q (groups: %s)", i, g.Key, w.key, dumpGroups(groups))
			continue
		}
		if got := memberStrings(g.Packages); !reflect.DeepEqual(got, w.members) {
			t.Errorf("group %d (%s): members = %v, want %v", i, g.Key, got, w.members)
		}
	}

	// Groups are sorted by Size descending, then Key ascending.
	for i := 1; i < len(groups); i++ {
		if groups[i-1].Size < groups[i].Size {
			t.Errorf("groups out of Size-descending order at %d: %+v then %+v", i, groups[i-1], groups[i])
		}
	}

	// Packages that must never appear in any group.
	forbidden := []string{
		"apt/dpkg:libc6", // two dpkg copies of the same package: one family
		"snap:code",      // no alias bridges snap code to a flatpak ID in these fixtures
		"npm:npm",
		"npm:corepack",
		"npm:my-linked-tool",
		"npm:unversioned-pkg",
		"npm:@anthropic-ai/claude-code",
		"pip:pip",
		"pip:PyYAML",
		"pip:my-editable",
		"pipx:httpie", // pipx alone, both of its keys are unique to it
		"flatpak:GNU Image Manipulation Program",
		"flatpak:Calculator", // generic segment rule / alias, no second manager
		"flatpak:OBS Studio",
	}
	for _, name := range forbidden {
		for _, g := range groups {
			for _, p := range g.Packages {
				if p.DB+":"+p.Name == name {
					t.Errorf("package %q must not be in any group, found in group %q (%s)",
						name, g.Key, strings.Join(memberStrings(g.Packages), ", "))
				}
			}
		}
	}
}

func memberStrings(pkgs []Package) []string {
	out := make([]string, 0, len(pkgs))
	for _, p := range pkgs {
		out = append(out, p.DB+":"+p.Name)
	}
	return out
}

func dumpGroups(groups []DuplicateGroup) string {
	parts := make([]string, 0, len(groups))
	for _, g := range groups {
		parts = append(parts, fmt.Sprintf("%s %v", g.Key, memberStrings(g.Packages)))
	}
	return strings.Join(parts, " | ")
}

func dumpWant(want []struct {
	key     string
	members []string
},
) string {
	parts := make([]string, 0, len(want))
	for _, w := range want {
		parts = append(parts, fmt.Sprintf("%s %v", w.key, w.members))
	}
	return strings.Join(parts, " | ")
}
