//go:build brew

package backend

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
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

func LoadPackages() (chan []Package, error) {
	start := time.Now()

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

	var wg sync.WaitGroup
	pkgsChan := make(chan []Package, 2)

	wg.Go(func() {
		packages := make([]Package, 0, len(entries))
		results := make(chan Package, len(entries)*2) // *2 to account for casks
		var loadWg sync.WaitGroup
		semaphore := make(chan struct{}, 8)

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			loadWg.Go(func() {
				pkgName := entry.Name()

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
						return
					}
					if installedVersion != verEntry.Name() {
						continue
					}

					verPath := filepath.Join(pkgPath, verEntry.Name())

					receiptPath := filepath.Join(verPath, "INSTALL_RECEIPT.json")
					receiptData, err := os.ReadFile(receiptPath)

					var installDate time.Time
					var receipt installReceipt
					isDirect := true // Default to direct if no receipt

					if err == nil && json.Unmarshal(receiptData, &receipt) == nil {
						if receipt.Time > 0 {
							installDate = time.Unix(receipt.Time, 0)
						}
						isDirect = receipt.InstalledOnRequest
					}

					if installDate.IsZero() {
						installDate = time.Now()
					}

					size := calculateSize(verPath)

					results <- Package{
						Name:     pkgName,
						Version:  verEntry.Name(),
						Size:     size,
						Date:     installDate,
						IsDirect: isDirect,
						DB:       "Homebrew",
					}

					break
				}
			})
		}

		caskroomPath := strings.Replace(cellarPath, "/Cellar", "/Caskroom", 1)
		if caskEntries, err := os.ReadDir(caskroomPath); err == nil {
			for _, entry := range caskEntries {
				if !entry.IsDir() {
					continue
				}

				loadWg.Go(func() {
					caskName := entry.Name()
					semaphore <- struct{}{}
					defer func() { <-semaphore }()

					version, _ := getCaskVersion(caskroomPath, caskName)
					size := getCaskSize(caskroomPath, caskName, version)
					installedDate, _ := getCaskMetadata(caskroomPath, caskName, version)

					results <- Package{
						Name:     caskName,
						Version:  version,
						Size:     size,
						Date:     installedDate,
						IsDirect: true, // Casks are always direct installs
						DB:       "Cask",
					}
				})
			}
		}

		go func() {
			loadWg.Wait()
			close(results)
		}()

		for pkg := range results {
			packages = append(packages, pkg)
		}

		wg.Go(func() { pkgsChan <- packages })

		outdated := getOutdatedVersions()
		for i := range packages {
			if newVer, ok := outdated[packages[i].Name]; ok && newVer != packages[i].Version {
				packages[i].NewVersion = newVer
			}
		}

		wg.Go(func() { pkgsChan <- packages })
	})

	go func() {
		wg.Wait()
		close(pkgsChan)
		slog.Info("time to map all packages and check updates", "time", time.Since(start))
	}()

	return pkgsChan, nil
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

func getCaskSize(caskroomPath, caskName, version string) int64 {
	caskPath := filepath.Join(caskroomPath, caskName, version)
	size := calculateSize(caskPath)

	appPaths := []string{
		filepath.Join("/Applications", caskName+".app"),
		filepath.Join("/Applications", strings.Title(caskName)+".app"),
	}

	for _, appPath := range appPaths {
		info, err := os.Lstat(appPath)
		if err != nil {
			continue
		}

		// Only count if it's NOT a symlink (real copy)
		if info.Mode()&os.ModeSymlink == 0 {
			size += calculateSize(appPath)
			break
		}
	}

	return size
}

func getCaskMetadata(caskroomPath, caskName, version string) (time.Time, error) {
	metadataPath := filepath.Join(caskroomPath, caskName, version, ".metadata")

	// Check if .metadata is a directory with timestamped subdirs
	entries, err := os.ReadDir(metadataPath)
	if err == nil && len(entries) > 0 {
		// Use first (usually only one) timestamp directory
		if timestamp, err := parseTimestampDir(entries[0].Name()); err == nil {
			return timestamp, nil
		}
	}

	// Fallback: check directory mod time
	if info, err := os.Stat(filepath.Join(caskroomPath, caskName, version)); err == nil {
		return info.ModTime(), nil
	}

	return time.Now(), nil
}

func parseTimestampDir(dirName string) (time.Time, error) {
	// Directory name is usually a Unix timestamp like "1234567890"
	var timestamp int64
	_, err := fmt.Sscanf(dirName, "%d", &timestamp)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(timestamp, 0), nil
}

func getCaskVersion(caskroomPath, caskName string) (string, error) {
	caskPath := filepath.Join(caskroomPath, caskName)
	versions, err := os.ReadDir(caskPath)
	if err != nil {
		return "", err
	}

	// Usually only one version installed, pick the newest by mod time
	var newestVer string
	var newestTime time.Time

	for _, v := range versions {
		if !v.IsDir() {
			continue
		}
		if v.Name() == ".metadata" {
			continue
		}
		info, _ := v.Info()
		if info.ModTime().After(newestTime) {
			newestTime = info.ModTime()
			newestVer = v.Name()
		}
	}

	if newestVer == "" && len(versions) > 0 {
		// Fallback: use first directory
		newestVer = versions[0].Name()
	}

	return newestVer, nil
}

func getOutdatedVersions() map[string]string {
	outdated := make(map[string]string)

	cmd := exec.Command("brew", "outdated", "--json=v2")
	output, err := cmd.Output()
	if err != nil {
		slog.Warn("Failed to get outdated packages", "err", err)
		return outdated
	}

	var result struct {
		Formulae []struct {
			Name              string   `json:"name"`
			AvailableVersion  string   `json:"current_version"`
		} `json:"formulae"`
		Casks []struct {
			Name              string   `json:"name"`
			AvailableVersion  string   `json:"current_version"`
		} `json:"casks"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		slog.Warn("Failed to parse outdated JSON", "err", err)
		return outdated
	}

	for _, formula := range result.Formulae {
		outdated[formula.Name] = formula.AvailableVersion
	}

	for _, cask := range result.Casks {
		outdated[cask.Name] = cask.AvailableVersion
	}

	return outdated
}

func Update() error {
	return exec.Command("brew", "update").Run()
}
