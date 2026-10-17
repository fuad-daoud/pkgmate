package backend

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"reflect"
	"testing"
	"time"
)

type call struct{ argv []string }

type fakeRun struct {
	code int
	err  error
	out  string
}

// setupFakes installs fake lookPath/runCmd; runs are consumed in order.
func setupFakes(t *testing.T, missing map[string]bool, runs []fakeRun) *[]call {
	t.Helper()
	oldLook, oldRun := lookPath, runCmd
	t.Cleanup(func() { lookPath, runCmd = oldLook, oldRun })
	lookPath = func(name string) (string, error) {
		if missing[name] {
			return "", exec.ErrNotFound
		}
		return "/usr/bin/" + name, nil
	}
	var calls []call
	runCmd = func(argv []string, out io.Writer) (int, error) {
		i := len(calls)
		calls = append(calls, call{argv})
		if i >= len(runs) {
			t.Fatalf("unexpected runner call %d: %v", i, argv)
		}
		io.WriteString(out, runs[i].out)
		return runs[i].code, runs[i].err
	}
	return &calls
}

func readResult(t *testing.T, ch chan OperationResult) OperationResult {
	t.Helper()
	select {
	case r := <-ch:
		return r
	case <-time.After(time.Second):
		t.Fatal("no result sent")
		return OperationResult{}
	}
}

func TestEscalationArgv(t *testing.T) {
	evil := "; rm -rf /"
	args := []string{"-S", evil}
	cases := map[string][]string{
		"pkexec": {"pkexec", "/usr/bin/pacman", "-S", evil},
		"sudo":   {"sudo", "--", "/usr/bin/pacman", "-S", evil},
		"":       {"/usr/bin/pacman", "-S", evil},
	}
	for method, want := range cases {
		if got := escalationArgv(method, "/usr/bin/pacman", args); !reflect.DeepEqual(got, want) {
			t.Errorf("%q: got %v want %v", method, got, want)
		}
	}
}

func TestShouldFallback(t *testing.T) {
	for code, want := range map[int]bool{126: true, 127: true, 0: false, 1: false, 2: false} {
		if shouldFallback(code) != want {
			t.Errorf("shouldFallback(%d) != %v", code, want)
		}
	}
}

func TestPrivilegedPkexecSuccess(t *testing.T) {
	calls := setupFakes(t, nil, []fakeRun{{code: 0, out: "hello"}})
	f, ch := CreatePrivilegedCmd("op", "pacman", "-Syy")
	if err := f(); err != nil {
		t.Fatal(err)
	}
	r := readResult(t, ch)
	if len(*calls) != 1 || !reflect.DeepEqual((*calls)[0].argv, []string{"pkexec", "/usr/bin/pacman", "-Syy"}) {
		t.Fatalf("calls: %v", *calls)
	}
	if !r.Success || r.Error != nil || r.Logs != "hello" {
		t.Fatalf("result: %+v", r)
	}
}

func TestPrivilegedPkexecMissingUsesSudo(t *testing.T) {
	calls := setupFakes(t, map[string]bool{"pkexec": true}, []fakeRun{{code: 0}})
	f, ch := CreatePrivilegedCmd("op", "pacman", "-Syy")
	if err := f(); err != nil {
		t.Fatal(err)
	}
	r := readResult(t, ch)
	if len(*calls) != 1 || !reflect.DeepEqual((*calls)[0].argv, []string{"sudo", "--", "/usr/bin/pacman", "-Syy"}) {
		t.Fatalf("calls: %v", *calls)
	}
	if !r.Success {
		t.Fatalf("result: %+v", r)
	}
}

func TestPrivilegedFallbackOnAuthCodes(t *testing.T) {
	for _, code := range []int{126, 127} {
		calls := setupFakes(t, nil, []fakeRun{{code: code}, {code: 3}})
		f, ch := CreatePrivilegedCmd("op", "pacman", "-Syy")
		if err := f(); err != nil {
			t.Fatal(err)
		}
		r := readResult(t, ch)
		if len(*calls) != 2 || (*calls)[0].argv[0] != "pkexec" || (*calls)[1].argv[0] != "sudo" {
			t.Fatalf("code %d calls: %v", code, *calls)
		}
		if r.Success || r.Error == nil {
			t.Fatalf("code %d result should reflect sudo's 3: %+v", code, r)
		}
	}
}

func TestPrivilegedPkexecStartErrorFallsBack(t *testing.T) {
	calls := setupFakes(t, nil, []fakeRun{{err: errors.New("boom")}, {code: 0}})
	f, ch := CreatePrivilegedCmd("op", "pacman")
	if err := f(); err != nil {
		t.Fatal(err)
	}
	r := readResult(t, ch)
	if len(*calls) != 2 || (*calls)[1].argv[0] != "sudo" || !r.Success {
		t.Fatalf("calls %v result %+v", *calls, r)
	}
}

func TestPrivilegedPkexecExit1NoFallback(t *testing.T) {
	calls := setupFakes(t, nil, []fakeRun{{code: 1}})
	f, ch := CreatePrivilegedCmd("op", "pacman")
	if err := f(); err != nil {
		t.Fatalf("func should return nil, got %v", err)
	}
	r := readResult(t, ch)
	if len(*calls) != 1 || r.Success || r.Error == nil {
		t.Fatalf("calls %v result %+v", *calls, r)
	}
}

func TestPrivilegedBothStartErrors(t *testing.T) {
	setupFakes(t, nil, []fakeRun{{err: errors.New("a")}, {err: errors.New("b")}})
	f, ch := CreatePrivilegedCmd("op", "pacman")
	if err := f(); err == nil {
		t.Fatal("expected error")
	}
	r := readResult(t, ch)
	if r.Success || r.Error == nil {
		t.Fatalf("result %+v", r)
	}
}

func TestPrivilegedCommandNotFound(t *testing.T) {
	calls := setupFakes(t, map[string]bool{"nope": true}, nil)
	f, ch := CreatePrivilegedCmd("op", "nope")
	if err := f(); err == nil {
		t.Fatal("expected error")
	}
	r := readResult(t, ch)
	if len(*calls) != 0 || r.Success || r.Error == nil {
		t.Fatalf("calls %v result %+v", *calls, r)
	}
}

func TestNormalCmdArgv(t *testing.T) {
	calls := setupFakes(t, nil, []fakeRun{{code: 0, out: "ok"}})
	f, ch := createNormalCmd("op", "brew", "update")
	if err := f(); err != nil {
		t.Fatal(err)
	}
	r := readResult(t, ch)
	if len(*calls) != 1 || !reflect.DeepEqual((*calls)[0].argv, []string{"/usr/bin/brew", "update"}) {
		t.Fatalf("calls: %v", *calls)
	}
	if !r.Success || r.Logs != "ok" {
		t.Fatalf("result %+v", r)
	}
}

func TestNewResult(t *testing.T) {
	if r := newResult("x", 0, "l", nil); !r.Success || r.Error != nil || r.Logs != "l" {
		t.Fatalf("%+v", r)
	}
	if r := newResult("x", 2, "", nil); r.Success || r.Error == nil {
		t.Fatalf("%+v", r)
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("PKGMATE_HELPER") != "1" {
		return
	}
	os.Exit(7)
}

func TestResultRunExecExitCode(t *testing.T) {
	t.Setenv("PKGMATE_HELPER", "1")
	code, err := runExec([]string{os.Args[0], "-test.run=TestHelperProcess"}, io.Discard)
	if err != nil || code != 7 {
		t.Fatalf("code %d err %v", code, err)
	}
}
