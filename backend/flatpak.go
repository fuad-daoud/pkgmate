//go:build flatpak || all_backends

package backend

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type FlatpakBackend struct{}

func init() {
	RegisterBackend(&FlatpakBackend{})
}

func (f *FlatpakBackend) Name() string {
	return "flatpak"
}

func (f *FlatpakBackend) IsAvailable() bool {
	if _, err := exec.LookPath("flatpak"); err != nil {
		return false
	}
	cmd := exec.Command("flatpak", "list", "--app")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func (f *FlatpakBackend) LoadPackages() (chan []Package, error) {
	start := time.Now()

	var wg sync.WaitGroup
	pkgsChan := make(chan []Package, 1)

	wg.Go(func() {
		cmd := exec.Command("flatpak", "list", "--app", "--columns=application,name,version,size,installation")
		output, err := cmd.Output()
		if err != nil {
			slog.Warn("Failed to list flatpak packages", "err", err)
			return
		}

		packages := make([]Package, 0)
		lines := strings.SplitSeq(string(output), "\n")

		for line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			fields := strings.Fields(line)
			if len(fields) < 4 {
				continue
			}

			// Parse fields: application, name, version, size, installation
			var sizeIdx int
			for i := len(fields) - 1; i >= 0; i-- {
				if strings.HasSuffix(fields[i], "B") {
					sizeIdx = i - 1 // Size value is before unit
					break
				}
			}

			if sizeIdx < 3 {
				continue
			}

			version := fields[sizeIdx-1]
			sizeStr := fields[sizeIdx] + fields[sizeIdx+1]

			nameFields := fields[1 : sizeIdx-1]
			name := strings.Join(nameFields, " ")

			size := parseFlatpakSize(sizeStr)
			installDate := getFlatpakInstallDate(name)

			packages = append(packages, Package{
				Name:     name,
				Version:  version,
				Size:     size,
				Date:     installDate,
				IsDirect: true, // All flatpak apps are direct installs
				IsFrozen: false,
				DB:       "flatpak",
			})
		}

		wg.Go(func() { pkgsChan <- packages })
	})

	go func() {
		wg.Wait()
		close(pkgsChan)
		slog.Info("time to load flatpak packages", "time", time.Since(start))
	}()

	return pkgsChan, nil
}

func (f *FlatpakBackend) Update() (func() error, chan OperationResult) {
	resultChan := make(chan OperationResult, 1)
	err := fmt.Errorf("flatpak update not implemented")

	return func() error {
		resultChan <- OperationResult{Error: err}
		return err
	}, resultChan
}

func (f *FlatpakBackend) GetOrphanPackages() ([]Package, error) {
	return make([]Package, 0), nil
}

func parseFlatpakSize(sizeStr string) int64 {
	sizeStr = strings.ReplaceAll(sizeStr, " ", "")
	sizeStr = strings.ToUpper(sizeStr)

	var value float64
	var unit string

	if strings.Contains(sizeStr, "GB") {
		fmt.Sscanf(sizeStr, "%fGB", &value)
		unit = "GB"
	} else if strings.Contains(sizeStr, "MB") {
		fmt.Sscanf(sizeStr, "%fMB", &value)
		unit = "MB"
	} else if strings.Contains(sizeStr, "KB") {
		fmt.Sscanf(sizeStr, "%fKB", &value)
		unit = "KB"
	} else {
		return 0
	}

	switch unit {
	case "GB":
		return int64(value * 1024 * 1024 * 1024)
	case "MB":
		return int64(value * 1024 * 1024)
	case "KB":
		return int64(value * 1024)
	}

	return 0
}

func getFlatpakInstallDate(appID string) time.Time {
	paths := []string{
		filepath.Join("/var/lib/flatpak/app", appID),
		filepath.Join(os.Getenv("HOME"), ".local/share/flatpak/app", appID),
	}

	for _, basePath := range paths {
		if info, err := os.Stat(basePath); err == nil {
			return info.ModTime()
		}
	}

	return time.Now()
}
