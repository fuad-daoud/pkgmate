package backend

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
)

type OperationResult struct {
	Success bool
	Logs    string
	Error   error
}

// Seams swapped in tests.
var lookPath = exec.LookPath

// cmdRunner runs argv to completion. err is non-nil only when the process
// could not be started or waited on; a non-zero exit is (code, nil).
type cmdRunner func(argv []string, out io.Writer) (exitCode int, err error)

var runCmd cmdRunner = runExec

func runExec(argv []string, out io.Writer) (int, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = out
	cmd.Stderr = out
	err := cmd.Run()
	if err == nil {
		return 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), nil
	}
	return -1, err
}

// escalationArgv builds the argv for method "pkexec", "sudo" or "" (plain).
// path is the resolved absolute path, so neither escalator does a PATH lookup
// and the command can never be read as an option.
func escalationArgv(method, path string, args []string) []string {
	var argv []string
	switch method {
	case "pkexec":
		argv = []string{"pkexec", path}
	case "sudo":
		argv = []string{"sudo", "--", path}
	default:
		argv = []string{path}
	}
	return append(argv, args...)
}

// shouldFallback reports whether a pkexec exit code means authentication did
// not happen (126: dialog dismissed, 127: not authorized or no agent).
// Ambiguity: a program that itself exits 126 or 127 would trigger a sudo
// retry; pacman and apt do not use those codes.
func shouldFallback(exitCode int) bool {
	return exitCode == 126 || exitCode == 127
}

// newResult builds the OperationResult for a finished run.
func newResult(command string, exitCode int, logs string, err error) OperationResult {
	res := OperationResult{Success: err == nil && exitCode == 0, Logs: logs}
	switch {
	case err != nil:
		res.Error = err
	case exitCode != 0:
		res.Error = fmt.Errorf("%s exited with status %d", command, exitCode)
	}
	return res
}

func runLogged(operation, method string, argv []string, out io.Writer) (int, error) {
	slog.Info("running privileged command", "operation", operation, "method", method, "argv", argv)
	code, err := runCmd(argv, out)
	slog.Info("command finished", "operation", operation, "method", method, "exit", code, "err", err)
	return code, err
}

// runEscalated runs path with pkexec, falling back to sudo (logged).
func runEscalated(operation, path string, args []string, out io.Writer) (exitCode int, method string, err error) {
	if _, lerr := lookPath("pkexec"); lerr != nil {
		slog.Info("pkexec not found, using sudo", "operation", operation)
	} else {
		code, rerr := runLogged(operation, "pkexec", escalationArgv("pkexec", path, args), out)
		switch {
		case rerr != nil:
			slog.Warn("pkexec could not be started, falling back to sudo", "operation", operation, "err", rerr)
		case shouldFallback(code):
			slog.Warn("pkexec did not authorize, falling back to sudo", "operation", operation, "exit", code)
		default:
			return code, "pkexec", nil
		}
	}
	code, err := runLogged(operation, "sudo", escalationArgv("sudo", path, args), out)
	return code, "sudo", err
}

// execute resolves command, runs it via run, and sends exactly one result.
func execute(command string, resultChan chan OperationResult,
	run func(path string, out io.Writer) (int, error)) error {
	path, err := lookPath(command)
	if err != nil {
		err = fmt.Errorf("%s: %w", command, err)
		resultChan <- newResult(command, -1, "", err)
		return err
	}
	var buf bytes.Buffer
	out := io.MultiWriter(os.Stdout, &buf)
	code, err := run(path, out)
	resultChan <- newResult(command, code, buf.String(), err)
	return err
}

func createNormalCmd(operation, command string, args ...string) (func() error, chan OperationResult) {
	resultChan := make(chan OperationResult, 1)
	f := func() error {
		return execute(command, resultChan, func(path string, out io.Writer) (int, error) {
			return runLogged(operation, "", escalationArgv("", path, args), out)
		})
	}
	return f, resultChan
}

func CreatePrivilegedCmd(operation, command string, args ...string) (func() error, chan OperationResult) {
	resultChan := make(chan OperationResult, 1)
	f := func() error {
		return execute(command, resultChan, func(path string, out io.Writer) (int, error) {
			code, _, err := runEscalated(operation, path, args, out)
			return code, err
		})
	}
	return f, resultChan
}
