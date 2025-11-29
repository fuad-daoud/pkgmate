package backend

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type PacmanBackend struct{}

func init() {
	RegisterBackend(&PacmanBackend{})
}

func (p *PacmanBackend) Name() string {
	return "pacman"
}

func (p *PacmanBackend) IsAvailable() bool {
	if _, err := os.Stat("/var/lib/pacman/"); err != nil {
		return false
	}
	if _, err := exec.LookPath("pacman"); err != nil {
		return false
	}
	return true
}

func (p *PacmanBackend) LoadPackages() (chan []Package, error) {
	start := time.Now()

	var wg sync.WaitGroup
	pkgsChan := make(chan []Package, 2)

	wg.Go(func() {
		cmd := exec.Command("pacman", "-Qi")
		output, err := cmd.Output()
		if err != nil {
			slog.Error("Failed to get package info", "err", err)
			return
		}

		explicitPkgs := getExplicitPackages()
		pinnedPkgs := getPinnedPackages()
		upgradable := getUpgradablePackages()
		foreignPkgs := getForeignPackages()

		packages := parsePackageInfo(string(output), explicitPkgs, pinnedPkgs, upgradable, foreignPkgs)
		regularPkgs := make([]Package, 0)
		aurPkgs := make([]Package, 0)
		aurPkgNames := make([]string, 0)

		for _, pkg := range packages {
			if pkg.DB == "AUR" {
				aurPkgs = append(aurPkgs, pkg)
				aurPkgNames = append(aurPkgNames, pkg.Name)
			} else {
				pkg.DB = "pacman"
				regularPkgs = append(regularPkgs, pkg)
			}
		}

		wg.Go(func() { pkgsChan <- regularPkgs })

		// Check AUR versions
		newVersions, err := CheckAURVersions(aurPkgNames)
		if err != nil {
			slog.Warn("Could not check AUR versions", "err", err)
		} else {
			for i := range aurPkgs {
				if newVer, ok := newVersions[aurPkgs[i].Name]; ok && newVer != aurPkgs[i].Version {
					aurPkgs[i].NewVersion = newVer
				}
			}
		}

		wg.Go(func() { pkgsChan <- aurPkgs })
	})

	go func() {
		wg.Wait()
		close(pkgsChan)
		slog.Info("time to map all packages and sync with aur", "time", time.Since(start))
	}()

	return pkgsChan, nil
}

func (p *PacmanBackend) Update() (func() error, chan OperationResult) {
	return CreatePrivilegedCmd(p.Name()+"-update", "pacman", "-Syy", "--noconfirm")
}

func (p *PacmanBackend) GetOrphanPackages() ([]Package, error) {
	cmd := exec.Command("pacman", "-Qtdq")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return []Package{}, nil
		}
		return nil, fmt.Errorf("failed to get orphans: %w", err)
	}

	orphanNames := strings.Fields(strings.TrimSpace(string(output)))
	if len(orphanNames) == 0 {
		return []Package{}, nil
	}

	// Get detailed info for orphan packages
	args := append([]string{"-Qi"}, orphanNames...)
	cmd = exec.Command("pacman", args...)
	output, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get orphan details: %w", err)
	}

	pinnedPkgs := getPinnedPackages()
	packages := parsePackageInfo(string(output), make(map[string]bool), pinnedPkgs, make(map[string]string), make(map[string]bool))

	// Mark all as dependencies
	for i := range packages {
		packages[i].IsDirect = false
	}

	return packages, nil
}

func getExplicitPackages() map[string]bool {
	explicit := make(map[string]bool)

	cmd := exec.Command("pacman", "-Qeq")
	output, err := cmd.Output()
	if err != nil {
		slog.Warn("Failed to get explicit packages", "err", err)
		return explicit
	}

	for name := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
		if name != "" {
			explicit[name] = true
		}
	}

	return explicit
}

func getPinnedPackages() map[string]bool {
	pinned := make(map[string]bool)

	data, err := os.ReadFile("/etc/pacman.conf")
	if err != nil {
		return pinned
	}

	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "IgnorePkg") && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				pkgs := strings.FieldsSeq(strings.TrimSpace(parts[1]))
				for pkg := range pkgs {
					pinned[pkg] = true
				}
			}
		}
	}

	return pinned
}

func getUpgradablePackages() map[string]string {
	upgradable := make(map[string]string)

	cmd := exec.Command("pacman", "-Qu")
	output, err := cmd.Output()
	if err != nil {
		// Exit code 1 means no updates available
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return upgradable
		}
		slog.Warn("Failed to check for upgradable packages", "err", err)
		return upgradable
	}

	for line := range strings.SplitSeq(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			// Format: package_name old_version -> new_version
			pkgName := fields[0]
			newVersion := fields[3]
			upgradable[pkgName] = newVersion
		}
	}

	return upgradable
}

func getForeignPackages() map[string]bool {
	foreign := make(map[string]bool)

	cmd := exec.Command("pacman", "-Qmq")
	output, err := cmd.Output()
	if err != nil {
		// Exit code 1 means no foreign packages
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return foreign
		}
		slog.Warn("Failed to get foreign packages", "err", err)
		return foreign
	}

	for name := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
		if name != "" {
			foreign[name] = true
		}
	}

	return foreign
}

func parsePackageInfo(output string, explicitPkgs, pinnedPkgs map[string]bool, upgradable map[string]string, foreignPkgs map[string]bool) []Package {
	packages := make([]Package, 0)
	paragraphs := strings.SplitSeq(output, "\n\n")

	for para := range paragraphs {
		if para == "" {
			continue
		}

		var pkg Package
		lines := strings.SplitSeq(para, "\n")

		for line := range lines {
			if strings.HasPrefix(line, " ") {
				continue // Skip continuation lines
			}

			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "Name":
				pkg.Name = value
			case "Version":
				pkg.Version = value
			case "Installed Size":
				pkg.Size = parseSize(value)
			case "Install Date":
				pkg.Date = parseDate(value)
			case "Install Reason":
				pkg.IsDirect = value == "Explicitly installed"
			case "Repository":
				pkg.DB = value
			}
		}

		if pkg.Name == "" {
			continue
		}

		// Override IsDirect with explicit package list (more reliable)
		pkg.IsDirect = explicitPkgs[pkg.Name]
		pkg.IsFrozen = pinnedPkgs[pkg.Name]

		// Check for updates
		if newVer, ok := upgradable[pkg.Name]; ok {
			pkg.NewVersion = newVer
		}

		// Check if AUR package
		if foreignPkgs[pkg.Name] {
			pkg.DB = "pacman/AUR"
		}

		// If no install date, try to get from filesystem
		if pkg.Date.IsZero() {
			pkgPath := filepath.Join("/var/lib/pacman/local", pkg.Name+"-"+pkg.Version)
			if info, err := os.Stat(pkgPath); err == nil {
				pkg.Date = info.ModTime()
			}
		}

		packages = append(packages, pkg)
	}

	return packages
}

func parseSize(sizeStr string) int64 {
	// Format examples: "123.45 KiB", "12.34 MiB", "1.23 GiB"
	fields := strings.Fields(sizeStr)
	if len(fields) < 2 {
		return 0
	}

	value, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}

	unit := fields[1]
	switch unit {
	case "B":
		return int64(value)
	case "KiB":
		return int64(value * 1024)
	case "MiB":
		return int64(value * 1024 * 1024)
	case "GiB":
		return int64(value * 1024 * 1024 * 1024)
	default:
		return 0
	}
}

func parseDate(dateStr string) time.Time {
	// Format: "Mon 28 Nov 2024 10:30:45 AM UTC"
	layouts := []string{
		"Mon 02 Jan 2006 03:04:05 PM MST",
		"Mon 2 Jan 2006 03:04:05 PM MST",
		time.ANSIC,
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t
		}
	}

	return time.Now()
}
