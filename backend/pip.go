//go:build pip || all_backends

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

type PipBackend struct{}

type pipPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func init() {
	RegisterBackend(&PipBackend{})
}

func (p *PipBackend) Name() string {
	return "pip"
}

func (p *PipBackend) IsAvailable() bool {
	for _, cmd := range []string{"pip3", "pip"} {
		if _, err := exec.LookPath(cmd); err == nil {
			testCmd := exec.Command(cmd, "list", "--format=json")
			if err := testCmd.Run(); err == nil {
				return true
			}
		}
	}
	return false
}

func (p *PipBackend) LoadPackages(wg *sync.WaitGroup, load func(Package)) {
	pipCmd := getPipCommand()
	if pipCmd == "" {
		slog.Error("no pip command available")
		return
	}

	sitePackages := getSitePackagesPath(pipCmd)
	if sitePackages == "" {
		slog.Error("could not determine site-packages location")
		return
	}

	pipPackages := getPipList(pipCmd)

	semaphore := make(chan struct{}, 16)
	wg.Go(func() {
		for _, pkgInfo := range pipPackages {
			wg.Go(func() {
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				pkg := processPipPackage(pkgInfo.Name, pkgInfo.Version, sitePackages)
				if pkg.Name != "" {
					load(pkg)
				}
			})
		}
	})

	outdated := getPipOutdated(pipCmd)
	for _, pkgInfo := range pipPackages {
		wg.Go(func() {
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			pkg := processPipPackage(pkgInfo.Name, pkgInfo.Version, sitePackages)
			if pkg.Name != "" {
				if newVer, ok := outdated[pkg.Name]; ok && newVer != pkg.Version {
					pkg.NewVersion = newVer
					load(pkg)
				}
			}
		})
	}

	if _, err := exec.LookPath("pipx"); err == nil {
		pipxPackages := getPipxList()
		for _, pkgInfo := range pipxPackages {
			wg.Go(func() {
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				pkg := processPipxPackage(pkgInfo.Name, pkgInfo.Version)
				if pkg.Name != "" {
					load(pkg)
				}
			})
		}
	}
}

func (p *PipBackend) Update() (func() error, chan OperationResult) {
	resultChan := make(chan OperationResult, 1)
	err := fmt.Errorf("pip update not implemented")

	return func() error {
		resultChan <- OperationResult{Error: err}
		return err
	}, resultChan
}

func getPipCommand() string {
	for _, cmd := range []string{"pip3", "pip"} {
		if _, err := exec.LookPath(cmd); err == nil {
			return cmd
		}
	}
	return ""
}

func getSitePackagesPath(pipCmd string) string {
	cmd := exec.Command(pipCmd, "show", "pip")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	for line := range strings.SplitSeq(string(output), "\n") {
		if after, ok := strings.CutPrefix(line, "Location: "); ok {
			return after
		}
	}

	return ""
}

func getPipList(pipCmd string) []pipPackage {
	cmd := exec.Command(pipCmd, "list", "--format=json")
	output, err := cmd.Output()
	if err != nil {
		slog.Warn("Failed to list pip packages", "err", err)
		return []pipPackage{}
	}

	var packages []pipPackage
	if err := json.Unmarshal(output, &packages); err != nil {
		slog.Warn("Failed to parse pip list output", "err", err)
		return []pipPackage{}
	}

	return packages
}

func getPipxList() []pipPackage {
	cmd := exec.Command("pipx", "list", "--json")
	output, err := cmd.Output()
	if err != nil {
		return getPipxListPlain()
	}

	var result struct {
		Venvs map[string]struct {
			Metadata struct {
				MainPackage struct {
					PackageVersion string `json:"package_version"`
				} `json:"main_package"`
			} `json:"metadata"`
		} `json:"venvs"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		slog.Warn("Failed to parse pipx list output", "err", err)
		return []pipPackage{}
	}

	packages := make([]pipPackage, 0)
	for name, venv := range result.Venvs {
		packages = append(packages, pipPackage{
			Name:    name,
			Version: venv.Metadata.MainPackage.PackageVersion,
		})
	}

	return packages
}

func getPipxListPlain() []pipPackage {
	cmd := exec.Command("pipx", "list")
	output, err := cmd.Output()
	if err != nil {
		return []pipPackage{}
	}

	packages := make([]pipPackage, 0)
	for line := range strings.SplitSeq(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				packages = append(packages, pipPackage{
					Name:    fields[1],
					Version: "unknown",
				})
			}
		}
	}

	return packages
}

func processPipPackage(name, version, sitePackages string) Package {
	distInfoDir := findDistInfoDir(sitePackages, name, version)
	if distInfoDir == "" {
		return Package{}
	}

	pkgDir := getPackageDir(sitePackages, name)
	size := calculateSize(pkgDir)
	size += calculateSize(distInfoDir)

	info, err := os.Stat(distInfoDir)
	installDate := time.Now()
	if err == nil {
		installDate = info.ModTime()
	}

	return Package{
		Name:     name,
		Version:  version,
		Size:     size,
		Date:     installDate,
		IsDirect: true,
		IsFrozen: false,
		IsOrphan: false,
		DB:       "pip",
	}
}

func processPipxPackage(name, version string) Package {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return Package{}
	}

	pipxVenvPath := filepath.Join(homeDir, ".local", "share", "pipx", "venvs", name)
	if _, err := os.Stat(pipxVenvPath); err != nil {
		return Package{}
	}

	size := calculateSize(pipxVenvPath)

	info, err := os.Stat(pipxVenvPath)
	installDate := time.Now()
	if err == nil {
		installDate = info.ModTime()
	}

	return Package{
		Name:     name,
		Version:  version,
		Size:     size,
		Date:     installDate,
		IsDirect: true,
		IsFrozen: false,
		IsOrphan: false,
		DB:       "pipx",
	}
}

func findDistInfoDir(sitePackages, name, version string) string {
	normalizedName := strings.ReplaceAll(strings.ToLower(name), "_", "-")
	distInfo := filepath.Join(sitePackages, fmt.Sprintf("%s-%s.dist-info", normalizedName, version))
	if _, err := os.Stat(distInfo); err == nil {
		return distInfo
	}

	pattern := filepath.Join(sitePackages, "*.dist-info")
	matches, _ := filepath.Glob(pattern)

	for _, match := range matches {
		baseName := filepath.Base(match)
		baseName = strings.TrimSuffix(baseName, ".dist-info")
		parts := strings.Split(baseName, "-")
		if len(parts) >= 2 {
			pkgName := strings.Join(parts[:len(parts)-1], "-")
			if strings.EqualFold(pkgName, normalizedName) ||
				strings.EqualFold(strings.ReplaceAll(pkgName, "-", "_"), name) {
				return match
			}
		}
	}

	return ""
}

func getPackageDir(sitePackages, name string) string {
	candidates := []string{
		name,
		strings.ReplaceAll(name, "-", "_"),
		strings.ReplaceAll(name, "_", "-"),
	}

	for _, candidate := range candidates {
		pkgPath := filepath.Join(sitePackages, candidate)
		if info, err := os.Stat(pkgPath); err == nil && info.IsDir() {
			return pkgPath
		}
	}

	return ""
}

func getPipOutdated(pipCmd string) map[string]string {
	outdated := make(map[string]string)

	cmd := exec.Command(pipCmd, "list", "--outdated", "--format=json")
	output, err := cmd.Output()
	if err != nil {
		return outdated
	}

	var packages []struct {
		Name          string `json:"name"`
		LatestVersion string `json:"latest_version"`
	}

	if err := json.Unmarshal(output, &packages); err != nil {
		slog.Warn("Failed to parse pip outdated output", "err", err)
		return outdated
	}

	for _, pkg := range packages {
		outdated[pkg.Name] = pkg.LatestVersion
	}

	return outdated
}

func (p PipBackend) String() string {
	return p.Name()
}
