//go:build brew || all_backends

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

type BrewBackend struct{}

type installReceipt struct {
	HomebrewVersion       string   `json:"homebrew_version"`
	UsedOptions           []string `json:"used_options"`
	UnusedOptions         []string `json:"unused_options"`
	BuiltAsBottle         bool     `json:"built_as_bottle"`
	InstalledOnRequest    bool     `json:"installed_on_request"`
	InstalledAsDependency bool     `json:"installed_as_dependency"`
	Time                  int64    `json:"time"`
}

func init() {
	RegisterBackend(&BrewBackend{})
}

func (b *BrewBackend) Name() string {
	return "brew"
}

func (b *BrewBackend) IsAvailable() bool {
	if _, err := exec.LookPath("brew"); err != nil {
		return false
	}
	cmd := exec.Command("brew", "--cellar")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func (b *BrewBackend) LoadPackages(wg *sync.WaitGroup, load func(Package)) {
	cellarCmd := exec.Command("brew", "--cellar")
	cellarOutput, err := cellarCmd.Output()
	if err != nil {
		slog.Error("failed to get cellar path", "err", err)
		return
	}
	cellarPath := strings.TrimSpace(string(cellarOutput))

	entries, err := os.ReadDir(cellarPath)
	if err != nil {
		slog.Error("failed to read cellar", "err", err)
		return
	}

	wg.Go(func() {
		pinnedFormulae := getPinnedFormulae()
		outdated := getOutdatedVersions()
		orphanNames := getOrphanNames()

		semaphore := make(chan struct{}, 8)

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			wg.Go(func() {
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

					root, err := os.OpenRoot(pkgPath)

					if err != nil {
						slog.Warn("Could not open pkgPath for", "pkgPath", pkgPath, "err", err)
						continue
					}
					receiptPath := filepath.Join(verEntry.Name(), "INSTALL_RECEIPT.json")
					receiptData, err := root.ReadFile(receiptPath)
					if err != nil {
						slog.Warn("could not read data for package", "package", pkgName, "err", err)
						continue
					}

					var installDate time.Time
					var receipt installReceipt
					isDirect := true

					if json.Unmarshal(receiptData, &receipt) == nil {
						if receipt.Time > 0 {
							installDate = time.Unix(receipt.Time, 0)
						}
						isDirect = receipt.InstalledOnRequest
					}

					if installDate.IsZero() {
						installDate = time.Now()
					}

					verPath := filepath.Join(pkgPath, verEntry.Name())
					size := calculateSize(verPath)

					pkg := Package{
						Name:     pkgName,
						Version:  verEntry.Name(),
						Size:     size,
						Date:     installDate,
						IsDirect: isDirect,
						IsFrozen: pinnedFormulae[pkgName],
						IsOrphan: orphanNames[pkgName],
						DB:       "brew",
					}

					if newVer, ok := outdated[pkg.Name]; ok && newVer != pkg.Version {
						pkg.NewVersion = newVer
					}

					load(pkg)
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

				wg.Go(func() {
					caskName := entry.Name()
					semaphore <- struct{}{}
					defer func() { <-semaphore }()

					version, _ := getCaskVersion(caskroomPath, caskName)
					size := getCaskSize(caskroomPath, caskName, version)
					installedDate, _ := getCaskMetadata(caskroomPath, caskName, version)

					pkg := Package{
						Name:     caskName,
						Version:  version,
						Size:     size,
						Date:     installedDate,
						IsDirect: true,
						IsFrozen: false,
						IsOrphan: false,
						DB:       "brew/Cask",
					}

					if newVer, ok := outdated[pkg.Name]; ok && newVer != pkg.Version {
						pkg.NewVersion = newVer
					}

					load(pkg)
				})
			}
		}
	})
}

func (b *BrewBackend) Update() (func() error, chan OperationResult) {
	return createNormalCmd("update", "brew", "update")
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

		if info.Mode()&os.ModeSymlink == 0 {
			size += calculateSize(appPath)
			break
		}
	}

	return size
}

func getCaskMetadata(caskroomPath, caskName, version string) (time.Time, error) {
	metadataPath := filepath.Join(caskroomPath, caskName, version, ".metadata")

	entries, err := os.ReadDir(metadataPath)
	if err == nil && len(entries) > 0 {
		if timestamp, err := parseTimestampDir(entries[0].Name()); err == nil {
			return timestamp, nil
		}
	}

	if info, err := os.Stat(filepath.Join(caskroomPath, caskName, version)); err == nil {
		return info.ModTime(), nil
	}

	return time.Now(), nil
}

func parseTimestampDir(dirName string) (time.Time, error) {
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
			Name             string `json:"name"`
			AvailableVersion string `json:"current_version"`
		} `json:"formulae"`
		Casks []struct {
			Name             string `json:"name"`
			AvailableVersion string `json:"current_version"`
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

func getPinnedFormulae() map[string]bool {
	pinned := make(map[string]bool)

	cmd := exec.Command("brew", "--prefix")
	output, err := cmd.Output()
	if err != nil {
		slog.Warn("could not get brew prefix", "err", err)
		return pinned
	}

	prefix := strings.TrimSpace(string(output))
	pinnedDir := filepath.Join(prefix, "var", "homebrew", "pinned")

	entries, err := os.ReadDir(pinnedDir)
	if err != nil {
		slog.Warn("could not read pinned dir", "err", err)
		return pinned
	}

	for _, entry := range entries {
		if info, err := entry.Info(); err == nil && info.Mode().Type() == os.ModeSymlink {
			pinned[entry.Name()] = true
		}
	}

	return pinned
}

func getOrphanNames() map[string]bool {
	orphans := make(map[string]bool)

	cmd := exec.Command("brew", "autoremove", "--dry-run")
	output, err := cmd.Output()
	if err != nil {
		return orphans
	}

	for line := range strings.SplitSeq(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, "Would remove:") {
			if strings.HasPrefix(line, "==>") {
				continue
			}
			if line == "" {
				continue
			}

			pkgName := strings.Fields(line)[0]
			if pkgName != "" {
				orphans[pkgName] = true
			}
		}
	}

	return orphans
}
func (b BrewBackend) String() string {
	return b.Name()
}
