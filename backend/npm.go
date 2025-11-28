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
	"time"
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

func (n *NpmBackend) LoadPackages() (chan []Package, error) {
	start := time.Now()

	rootCmd := exec.Command("npm", "root", "-g")
	rootOutput, err := rootCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get npm root: %w", err)
	}
	globalPath := strings.TrimSpace(string(rootOutput))

	listCmd := exec.Command("npm", "list", "-g", "--json", "--depth=0")
	listOutput, err := listCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list npm packages: %w", err)
	}

	var npmList struct {
		Dependencies map[string]npmPackage `json:"dependencies"`
	}

	if err := json.Unmarshal(listOutput, &npmList); err != nil {
		return nil, fmt.Errorf("failed to parse npm list output: %w", err)
	}

	var wg sync.WaitGroup
	pkgsChan := make(chan []Package, 1)

	wg.Go(func() {
		packages := make([]Package, 0, len(npmList.Dependencies))
		results := make(chan Package, len(npmList.Dependencies))
		var loadWg sync.WaitGroup
		semaphore := make(chan struct{}, 8)

		for name, metadata := range npmList.Dependencies {
			loadWg.Go(func() {
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

				results <- Package{
					Name:     name,
					Version:  metadata.Version,
					Size:     size,
					Date:     info.ModTime(),
					IsDirect: true, // All global packages are direct installs
					IsFrozen: false,
					DB:       "npm",
				}
			})
		}

		go func() {
			loadWg.Wait()
			close(results)
		}()

		for pkg := range results {
			packages = append(packages, pkg)
		}

		wg.Go(func() { pkgsChan <- packages })
	})

	go func() {
		wg.Wait()
		close(pkgsChan)
		slog.Info("time to load npm packages", "time", time.Since(start))
	}()

	return pkgsChan, nil
}

func (n *NpmBackend) Update() (func() error, chan OperationResult) {
	// Not implemented - update functionality is currently disabled
	resultChan := make(chan OperationResult, 1)
	err := fmt.Errorf("npm update not implemented")

	return func() error {
		resultChan <- OperationResult{Error: err}
		return err
	}, resultChan
}

func (n *NpmBackend) GetOrphanPackages() ([]Package, error) {
	// npm doesn't have orphan packages - dependencies are bundled with their parent packages
	return []Package{}, nil
}
