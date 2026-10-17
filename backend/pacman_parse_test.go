//go:build pacman || all_backends

package backend

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func readPacmanFixture(t *testing.T, name string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", "pacman", name))
	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}

func pacmanBytes(v float64, mult int64) int64 {
	return int64(v * float64(mult))
}

func TestParsePacmanQi(t *testing.T) {
	got := parsePacmanQi(readPacmanFixture(t, "qi.txt"))

	// [PIN] an en_GB date (no AM/PM) does not parse and gives a zero Date;
	// LoadPackages then falls back to the mtime.
	want := []Package{
		{
			Name: "bash", Version: "5.2.037-2", Size: pacmanBytes(9.59, 1<<20),
			Date: time.Date(2024, time.November, 28, 10, 30, 45, 0, time.UTC), IsDirect: false,
		},
		{
			Name: "vim", Version: "9.1.0861-1", Size: pacmanBytes(4.08, 1<<20),
			Date: time.Date(2024, time.November, 7, 10, 30, 45, 0, time.UTC), IsDirect: true,
		},
		{Name: "tiny-config", Version: "1.0-1", Size: pacmanBytes(512.5, 1<<10), IsDirect: true},
		{
			Name: "filesystem", Version: "2024.11.21-1", Size: 892,
			Date: time.Date(2024, time.November, 28, 10, 30, 45, 0, time.UTC), IsDirect: true,
		},
		{
			Name: "zoned-pkg", Version: "3.1-1", Size: pacmanBytes(1.5, 1<<30),
			Date: time.Date(2026, time.September, 20, 14, 5, 15, 0, time.UTC), IsDirect: false,
		},
	}

	if len(got) != len(want) {
		t.Fatalf("got %d packages, want %d: %+v", len(got), len(want), got)
	}

	for i := range want {
		if !got[i].Date.Equal(want[i].Date) {
			t.Errorf("package %d date: got %v, want %v", i, got[i].Date, want[i].Date)
		}

		g, w := got[i], want[i]
		g.Date, w.Date = time.Time{}, time.Time{}

		if !reflect.DeepEqual(g, w) {
			t.Errorf("package %d:\n got  %+v\n want %+v", i, g, w)
		}
	}
}

func TestParsePacmanQiEmpty(t *testing.T) {
	for _, in := range []string{"", "\n\n", "garbage without colon\n\n"} {
		if got := parsePacmanQi(in); len(got) != 0 {
			t.Errorf("parsePacmanQi(%q) = %+v, want none", in, got)
		}
	}
}

func TestParsePacmanQu(t *testing.T) {
	want := map[string]string{
		"linux":   "6.12.1.arch1-1",
		"firefox": "132.0.1-1",
		"yay-bin": "12.4.2-1",
		"bash":    "5.2.037-2",
	}

	got := parsePacmanQu(readPacmanFixture(t, "qu.txt"))
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParsePacmanIgnorePkg(t *testing.T) {
	t.Run("ignore lines", func(t *testing.T) {
		want := map[string]bool{"linux": true, "linux-headers": true, "nvidia-utils": true}

		got := parsePacmanIgnorePkg(readPacmanFixture(t, "pacman.conf"))
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("inline comment", func(t *testing.T) {
		t.Skip("known bug: inline comment `IgnorePkg = linux # keep lts` pins `#`, `keep` and `lts`")

		want := map[string]bool{"linux": true}

		got := parsePacmanIgnorePkg(readPacmanFixture(t, "pacman-inline-comment.conf"))
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})
}

func swapAURForTest(t *testing.T, url string) {
	t.Helper()

	origCache := cache
	origURL := aurRPCURL
	origClient := aurHTTPClient

	t.Cleanup(func() {
		cache = origCache
		aurRPCURL = origURL
		aurHTTPClient = origClient
	})

	cache = &aurCache{
		data:     make(map[string]cacheEntry),
		ttl:      1 * time.Minute,
		filePath: filepath.Join(t.TempDir(), "aur_cache.json"),
	}
	aurRPCURL = url
	aurHTTPClient = &http.Client{Timeout: 10 * time.Second}
}

func serveAURFixture(t *testing.T, name string) *httptest.Server {
	t.Helper()

	body := readPacmanFixture(t, name)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(ts.Close)

	return ts
}

func TestPacmanAURInfoFixture(t *testing.T) {
	ts := serveAURFixture(t, "aur_info.json")
	swapAURForTest(t, ts.URL)

	want := map[string]string{"yay-bin": "12.4.2-1", "example-git": "r42.abc1234-1"}

	got, err := CheckAURVersions([]string{"yay-bin", "example-git"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestPacmanAURErrorFixture(t *testing.T) {
	ts := serveAURFixture(t, "aur_error.json")
	swapAURForTest(t, ts.URL)

	// [PIN] a type:"error" response with HTTP 200 gives an empty map, nil error,
	// and the empty map is cached.
	got, err := CheckAURVersions([]string{"yay-bin"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("got %v, want empty map", got)
	}

	if _, ok := cache.Get(CACHE_KEY); !ok {
		t.Error("expected the empty result to be cached")
	}
}
