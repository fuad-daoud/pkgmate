//go:build npm || all_backends

package backend

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type NpmBackend struct{}

type npmPackage struct {
	Version string `json:"version"`
}

func init() {
	RegisterBackend(&NpmBackend{})
}

func (n *NpmBackend) Name() string {
	return "npm"
}

func (n *NpmBackend) IsAvailable() bool {
	if _, err := exec.LookPath("npm"); err != nil {
		return false
	}

	cmd := exec.Command("npm", "root", "-g")
	if err := cmd.Run(); err != nil {
		return false
	}

	return true
}

func (n *NpmBackend) LoadPackages(wg *sync.WaitGroup, load func(Package)) {
	rootCmd := exec.Command("npm", "root", "-g")
	rootOutput, err := rootCmd.Output()
	if err != nil {
		slog.Error("failed to get npm root", "err", err)
		return
	}
	globalPath := strings.TrimSpace(string(rootOutput))

	listCmd := exec.Command("npm", "list", "-g", "--json", "--depth=0")
	listOutput, err := listCmd.Output()
	if err != nil {
		slog.Error("failed to list npm packages", "err", err)
		return
	}

	var npmList struct {
		Dependencies map[string]npmPackage `json:"dependencies"`
	}

	if err := json.Unmarshal(listOutput, &npmList); err != nil {
		slog.Error("failed to parse npm list output", "err", err)
		return
	}

	semaphore := make(chan struct{}, 8)

	for name, metadata := range npmList.Dependencies {
		wg.Go(func() {
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			pkgPath := filepath.Join(globalPath, name)

			if _, err := os.Stat(pkgPath); err != nil {
				slog.Warn("Package directory not found", "package", name, "path", pkgPath)
				return
			}

			info, err := os.Stat(pkgPath)
			if err != nil {
				slog.Warn("Failed to stat package", "package", name, "err", err)
				return
			}

			size := calculateSize(pkgPath)

			load(Package{
				Name:     name,
				Version:  metadata.Version,
				Size:     size,
				Date:     info.ModTime(),
				IsDirect: true,
				IsFrozen: false,
				IsOrphan: false,
				DB:       "npm",
			})
		})
	}
}

func (n *NpmBackend) Update() (func() error, chan OperationResult) {
	resultChan := make(chan OperationResult, 1)
	err := fmt.Errorf("npm update not implemented")

	return func() error {
		resultChan <- OperationResult{Error: err}
		return err
	}, resultChan
}
func (n NpmBackend) String() string {
	return n.Name()
}
