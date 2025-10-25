//go:build brew

package backend

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type installReceipt struct {
	HomebrewVersion       string   `json:"homebrew_version"`
	UsedOptions           []string `json:"used_options"`
	UnusedOptions         []string `json:"unused_options"`
	BuiltAsBottle         bool     `json:"built_as_bottle"`
	InstalledOnRequest    bool     `json:"installed_on_request"`
	InstalledAsDependency bool     `json:"installed_as_dependency"`
	Time                  int64    `json:"time"`
}

func LoadPackages() ([]Package, error) {
	cellarCmd := exec.Command("brew", "--cellar")
	cellarOutput, err := cellarCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get cellar path: %w", err)
	}
	cellarPath := strings.TrimSpace(string(cellarOutput))

	entries, err := os.ReadDir(cellarPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cellar: %w", err)
	}

	packages := make([]Package, 0, len(entries))
	results := make(chan Package, len(entries))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 8)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		wg.Add(1)
		go func(pkgName string) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			pkgPath := filepath.Join(cellarPath, pkgName)

			versions, err := os.ReadDir(pkgPath)
			if err != nil {
				return
			}

			for _, verEntry := range versions {
				if !verEntry.IsDir() {
					continue
				}
				installedVersion, err := getInstalledVersion(cellarPath, pkgName)
				if err != nil {
					panic("AAAAAAAAAAAAAAAAH")
				}
				if installedVersion != verEntry.Name() {
					continue
				}

				verPath := filepath.Join(pkgPath, verEntry.Name())

				receiptPath := filepath.Join(verPath, "INSTALL_RECEIPT.json")
				receiptData, err := os.ReadFile(receiptPath)

				var installDate time.Time
				if err == nil {
					var receipt installReceipt
					if json.Unmarshal(receiptData, &receipt) == nil && receipt.Time > 0 {
						installDate = time.Unix(receipt.Time, 0)
					}
				}

				if installDate.IsZero() {
					installDate = time.Now()
				}

				size := calculateSize(verPath)

				results <- Package{
					Name:    pkgName,
					Version: verEntry.Name(),
					Size:    size,
					Date:    installDate,
				}

				break
			}
		}(entry.Name())
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for pkg := range results {
		packages = append(packages, pkg)
	}

	return packages, nil
}
func getInstalledVersion(cellarPath, pkgName string) (string, error) {
	optPath := strings.Replace(cellarPath, "/Cellar", "/opt", 1)
	linkPath := filepath.Join(optPath, pkgName)

	target, err := os.Readlink(linkPath)
	if err != nil {
		return getNewestVersion(filepath.Join(cellarPath, pkgName))
	}

	return filepath.Base(target), nil
}

func getNewestVersion(pkgPath string) (string, error) {
	versions, _ := os.ReadDir(pkgPath)
	if len(versions) == 0 {
		return "", fmt.Errorf("no versions found")
	}

	var newest string
	var newestTime time.Time

	for _, v := range versions {
		info, _ := v.Info()
		if info.ModTime().After(newestTime) {
			newestTime = info.ModTime()
			newest = v.Name()
		}
	}
	return newest, nil
}

func calculateSize(path string) int64 {
	var size int64
	filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return fs.SkipDir
		}
		if !d.IsDir() {
			if info, err := d.Info(); err == nil {
				size += info.Size()
			}
		}
		return nil
	})
	return size
}
