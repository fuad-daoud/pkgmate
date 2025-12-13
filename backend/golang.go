//go:build golang || all_backends

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

func (g *GoBackend) LoadPackages(wg *sync.WaitGroup, load func(Package)) {
	start := time.Now()

	binDir := getGoBinDir()
	if binDir == "" {
		slog.Error("could not determine Go binary directory")
		return
	}

	if _, err := os.Stat(binDir); err != nil {
		return
	}

	wg.Go(func() {
		entries, err := os.ReadDir(binDir)
		if err != nil {
			slog.Warn("Failed to read Go bin directory", "dir", binDir, "err", err)
			return
		}

		semaphore := make(chan struct{}, 8)

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			wg.Go(func() {
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				binaryPath := filepath.Join(binDir, entry.Name())

				bi, err := buildinfo.ReadFile(binaryPath)
				if err != nil {
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

				load(Package{
					Name:     packageName,
					Version:  version,
					Size:     info.Size(),
					Date:     info.ModTime(),
					IsDirect: true,
					IsFrozen: false,
					IsOrphan: false,
					DB:       "golang",
				})
			})
		}

		slog.Info("loaded packages from go", "time", time.Since(start))
	})
}

func (g *GoBackend) Update() (func() error, chan OperationResult) {
	resultChan := make(chan OperationResult, 1)
	err := fmt.Errorf("go update not implemented")

	return func() error {
		resultChan <- OperationResult{Error: err}
		return err
	}, resultChan
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
func (g GoBackend) String() string {
	return g.Name()
}
