//go:build npm || all_backends

package backend

import (
	"encoding/json"
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

	dependencies, err := parseNpmList(listOutput)
	if err != nil {
		slog.Error("failed to parse npm list output", "err", err)
		return
	}

	semaphore := make(chan struct{}, 8)

	for name, metadata := range dependencies {
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

func parseNpmList(out []byte) (map[string]npmPackage, error) {
	var npmList struct {
		Dependencies map[string]npmPackage `json:"dependencies"`
	}

	if err := json.Unmarshal(out, &npmList); err != nil {
		return nil, err
	}

	return npmList.Dependencies, nil
}

func (n NpmBackend) String() string {
	return n.Name()
}
