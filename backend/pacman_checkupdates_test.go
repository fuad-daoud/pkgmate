//go:build pacman || all_backends

package backend

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os/exec"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeCheckupdates struct {
	present bool
	out     string
	code    int
	err     error
	block   bool
}

func setupCheckupdates(t *testing.T, f fakeCheckupdates) {
	t.Helper()

	origLook, origRun, origTimeout := checkupdatesLookPath, checkupdatesRun, checkupdatesTimeout
	t.Cleanup(func() {
		checkupdatesLookPath, checkupdatesRun, checkupdatesTimeout = origLook, origRun, origTimeout
	})

	checkupdatesLookPath = func(file string) (string, error) {
		if !f.present {
			return "", exec.ErrNotFound
		}
		return "/usr/bin/" + file, nil
	}
	checkupdatesRun = func(ctx context.Context) ([]byte, int, error) {
		if f.block {
			<-ctx.Done()
			return nil, -1, ctx.Err()
		}
		return []byte(f.out), f.code, f.err
	}
}

func TestCheckupdatesExitCodes(t *testing.T) {
	const out = "linux 6.11.8.arch1-1 -> 6.12.2.arch1-1\n"

	tests := []struct {
		name string
		fake fakeCheckupdates
		want map[string]string
	}{
		{"absent", fakeCheckupdates{present: false}, nil},
		{"exit0 parsed", fakeCheckupdates{present: true, out: out}, map[string]string{"linux": "6.12.2.arch1-1"}},
		{"exit1 failure", fakeCheckupdates{present: true, out: out, code: 1}, nil},
		{"exit2 no updates", fakeCheckupdates{present: true, code: 2}, nil},
		{"other code", fakeCheckupdates{present: true, out: out, code: 127}, nil},
		{"start error", fakeCheckupdates{present: true, code: -1, err: errors.New("fork/exec: permission denied")}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupCheckupdates(t, tt.fake)
			if got := fetchFreshUpdates(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckupdatesTimeout(t *testing.T) {
	setupCheckupdates(t, fakeCheckupdates{present: true, block: true})
	checkupdatesTimeout = 20 * time.Millisecond

	done := make(chan map[string]string, 1)
	go func() { done <- fetchFreshUpdates() }()

	select {
	case got := <-done:
		if got != nil {
			t.Errorf("got %v, want nil", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("fetchFreshUpdates did not return after the timeout")
	}
}

func TestCheckupdatesAbsentLogsInfoOnce(t *testing.T) {
	setupCheckupdates(t, fakeCheckupdates{present: false})
	checkupdatesAbsent = sync.Once{}

	var buf bytes.Buffer
	orig := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	t.Cleanup(func() { slog.SetDefault(orig) })

	fetchFreshUpdates()
	fetchFreshUpdates()

	if n := strings.Count(buf.String(), "checkupdates not found"); n != 1 {
		t.Errorf("Info logged %d times, want 1; log: %q", n, buf.String())
	}
}

func TestPacmanReemitFreshUpdates(t *testing.T) {
	date := time.Date(2024, 11, 1, 0, 0, 0, 0, time.UTC)
	emitted := []Package{
		{Name: "linux", Version: "6.11.8", NewVersion: "6.12.1", DB: "pacman", IsDirect: true, IsFrozen: true, Date: date},
		{Name: "firefox", Version: "132.0-1", NewVersion: "132.0.1-1", DB: "pacman"},
		{Name: "bash", Version: "5.2.032-1", DB: "pacman", Date: date},
		{Name: "vim", Version: "9.0-1", NewVersion: "9.1-1", DB: "pacman"},
		{Name: "yay-bin", Version: "12.4.1-1", DB: "pacman/AUR"},
	}
	fresh := map[string]string{
		"linux":   "6.12.2",    // differs: re-emit
		"firefox": "132.0.1-1", // equal: skip
		"bash":    "5.2.037-2", // newly marked: re-emit
		"vim":     "",          // empty: skip
		"yay-bin": "12.4.2-1",  // foreign: skip
		// "absent" packages have no fresh entry: skip
	}

	got := reemitFreshUpdates(emitted, fresh)
	want := []Package{
		{Name: "linux", Version: "6.11.8", NewVersion: "6.12.2", DB: "pacman", IsDirect: true, IsFrozen: true, Date: date},
		{Name: "bash", Version: "5.2.032-1", NewVersion: "5.2.037-2", DB: "pacman", Date: date},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if emitted[0].NewVersion != "6.12.1" {
		t.Errorf("input mutated: %+v", emitted[0])
	}
	if got := reemitFreshUpdates(emitted, nil); len(got) != 0 {
		t.Errorf("nil fresh re-emitted %+v", got)
	}
}
