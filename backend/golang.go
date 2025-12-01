package backend

import (
	"debug/buildinfo"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type GoBackend struct{}

func init() {
	RegisterBackend(&GoBackend{})
}

func (g *GoBackend) Name() string {
	return "go"
}

func (g *GoBackend) IsAvailable() bool {
	if _, err := exec.LookPath("go"); err != nil {
		return false
	}
	return true
}

func (g *GoBackend) LoadPackages() (chan []Package, error) {
	start := time.Now()

	binDir := getGoBinDir()
	if binDir == "" {
		return nil, fmt.Errorf("could not determine Go binary directory")
	}

	if _, err := os.Stat(binDir); err != nil {
		// Directory doesn't exist, no packages installed
		pkgsChan := make(chan []Package, 1)
		close(pkgsChan)
		return pkgsChan, nil
	}

	var wg sync.WaitGroup
	pkgsChan := make(chan []Package, 1)

	wg.Go(func() {
		entries, err := os.ReadDir(binDir)
		if err != nil {
			slog.Warn("Failed to read Go bin directory", "dir", binDir, "err", err)
			return
		}

		packages := make([]Package, 0)
		results := make(chan Package, len(entries))
		var loadWg sync.WaitGroup
		semaphore := make(chan struct{}, 8)

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			loadWg.Go(func() {
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				binaryPath := filepath.Join(binDir, entry.Name())

				bi, err := buildinfo.ReadFile(binaryPath)
				if err != nil {
					// Not a Go binary or no build info
					slog.Debug("Could not read build info", "binary", entry.Name(), "err", err)
					return
				}

				info, err := os.Stat(binaryPath)
				if err != nil {
					slog.Warn("Could not stat binary", "binary", entry.Name(), "err", err)
					return
				}

				packageName := bi.Main.Path
				if packageName == "" {
					packageName = entry.Name()
				}

				version := bi.Main.Version
				if version == "" || version == "(devel)" {
					version = "unknown"
				}

				results <- Package{
					Name:     packageName,
					Version:  version,
					Size:     info.Size(),
					Date:     info.ModTime(),
					IsDirect: true, // All Go installed binaries are direct installs
					IsFrozen: false,
					DB:       "golang",
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
		slog.Info("time to load Go packages", "time", time.Since(start))
	}()

	return pkgsChan, nil
}

func (g *GoBackend) Update() (func() error, chan OperationResult) {
	resultChan := make(chan OperationResult, 1)
	err := fmt.Errorf("go update not implemented")

	return func() error {
		resultChan <- OperationResult{Error: err}
		return err
	}, resultChan
}

func (g *GoBackend) GetOrphanPackages() ([]Package, error) {
	return []Package{}, nil
}

func getGoBinDir() string {
	if gobin := os.Getenv("GOBIN"); gobin != "" {
		return gobin
	}

	if gopath := os.Getenv("GOPATH"); gopath != "" {
		return filepath.Join(gopath, "bin")
	}

	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, "go", "bin")
	}

	return ""
}
