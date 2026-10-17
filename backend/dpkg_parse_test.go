package backend

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func readDpkgFixture(t *testing.T, name string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", "dpkg", name))
	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}

func TestParseDpkgStatus(t *testing.T) {
	// [PIN] multi-arch libc6 yields two entries that share one Name.
	want := []Package{
		{Name: "bash", Version: "5.1-6ubuntu1.1", Size: 1864 * 1024},
		{Name: "openssh-server", Version: "1:8.9p1-3ubuntu0.10", Size: 1425 * 1024},
		{Name: "ca-certificates", Version: "20240203~22.04.1", Size: 0},
		{Name: "libc6", Version: "2.35-0ubuntu3.8", Size: 13182 * 1024},
		{Name: "libc6", Version: "2.35-0ubuntu3.8", Size: 12000 * 1024},
	}

	got := parseDpkgStatus(readDpkgFixture(t, "status"))
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}

	if got := parseDpkgStatus(""); len(got) != 0 {
		t.Errorf("empty input: got %+v", got)
	}
}

func TestParseAptExtendedStates(t *testing.T) {
	want := map[string]bool{"libfoo1": true, "python3-minimal": true}

	got := parseAptExtendedStates(readDpkgFixture(t, "extended_states"))
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseAptListUpgradable(t *testing.T) {
	want := map[string]string{
		"curl":    "7.81.0-1ubuntu1.18",
		"libssl3": "3.0.2-0ubuntu1.18",
		"bash":    "5.1-6ubuntu1.2",
	}

	got := parseAptListUpgradable(readDpkgFixture(t, "apt-list-upgradable.txt"))
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseDpkgSelections(t *testing.T) {
	// [PIN] the multi-arch hold is keyed "libc6:i386", so it never matches a status Name.
	want := map[string]bool{"openssh-server": true, "libc6:i386": true}

	got := parseDpkgSelections(readDpkgFixture(t, "get-selections.txt"))
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseAptAutoremoveSim(t *testing.T) {
	want := map[string]bool{"libfoo1": true, "python3-minimal": true}

	got := parseAptAutoremoveSim(readDpkgFixture(t, "autoremove-sim.txt"))
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if got := parseAptAutoremoveSim("Reading package lists...\n"); len(got) != 0 {
		t.Errorf("no Remv lines: got %v", got)
	}
}
