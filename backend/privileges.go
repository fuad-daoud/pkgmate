//go:build !darwin

package backend

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type OperationResult struct {
	Success bool
	Logs    string
	Error   error
}

func createNormalCmd(operation, command string, args ...string) (func() error, chan OperationResult) {
	resultChan := make(chan OperationResult, 1)
	logPath := filepath.Join(os.TempDir(), "user", fmt.Sprintf("pkgmate-%s.log", operation))
	exitPath := filepath.Join(os.TempDir(), "user", fmt.Sprintf("pkgmate-%s.exit", operation))


	f := func() error {
		os.Remove(logPath)
		os.Remove(exitPath)

		fullCmd := fmt.Sprintf(
			"{ %s %s > %s 2>&1; echo $? > %s; } &",
			command,
			strings.Join(args, " "),
			logPath,
			exitPath,
		)
		cmd := exec.Command("sh", "-c", fullCmd)
		cmd.Stdin = os.Stdin
		err := cmd.Run()
		if err != nil {
			return err
		}
		go monitorCompletion(logPath, exitPath, resultChan)
		return nil
	}
	return f, resultChan

}

func CreatePrivilegedCmd(operation, command string, args ...string) (func() error, chan OperationResult) {
	resultChan := make(chan OperationResult, 1)
	logPath := filepath.Join(os.TempDir(), "user", fmt.Sprintf("pkgmate-%s.log", operation))
	exitPath := filepath.Join(os.TempDir(), "user", fmt.Sprintf("pkgmate-%s.exit", operation))

	authFunc := func() error {
		os.Remove(logPath)
		os.Remove(exitPath)

		// Run command and write exit code when done
		fullCmd := fmt.Sprintf(
			"{ %s %s > %s 2>&1; echo $? > %s; } &",
			command,
			strings.Join(args, " "),
			logPath,
			exitPath,
		)

		if _, err := exec.LookPath("pkexec"); err == nil {
			cmd := exec.Command("pkexec", "sh", "-c", fullCmd)
			cmd.Stdin = os.Stdin
			if err := cmd.Run(); err == nil {
				go monitorCompletion(logPath, exitPath, resultChan)
				return nil
			}
		}

		cmd := exec.Command("sudo", "sh", "-c", fullCmd)
		cmd.Stdin = os.Stdin
		err := cmd.Run()
		if err == nil {
			go monitorCompletion(logPath, exitPath, resultChan)
		}
		return err
	}

	return authFunc, resultChan
}

func monitorCompletion(logPath, exitPath string, resultChan chan OperationResult) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		exitCode, err := os.ReadFile(exitPath)
		if err != nil {
			continue // Exit file doesn't exist yet
		}

		logs, _ := os.ReadFile(logPath)
		code := strings.TrimSpace(string(exitCode))

		resultChan <- OperationResult{
			Success: code == "0",
			Logs:    string(logs),
		}
		return
	}
}
