//go:build pacman || all_backends

package backend

import (
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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

func (p *PacmanBackend) LoadPackages(wg *sync.WaitGroup, load func(Package)) {

	cmd := exec.Command("pacman", "-Qi")
	packagesParagraphOutput, err := cmd.Output()
	if err != nil {
		slog.Error("Failed to get package info", "err", err)
		return
	}

	pinnedPkgs := getPinnedPackages()
	foreignPkgs := getForeignPackages()

	wg.Go(func() {
		explicitPkgs := getExplicitPackages()
		upgradable := getUpgradablePackages()

		aurPkgs := make([]Package, 0)
		paragraphs := strings.SplitSeq(string(packagesParagraphOutput), "\n\n")
		for para := range paragraphs {
			if para == "" {
				continue
			}
			pkg := parseParagraph(para)

			if pkg.Name == "" {
				continue
			}

			pkg.IsDirect = explicitPkgs[pkg.Name]
			pkg.IsFrozen = pinnedPkgs[pkg.Name]

			if newVer, ok := upgradable[pkg.Name]; ok {
				pkg.NewVersion = newVer
			}

			if foreignPkgs[pkg.Name] {
				pkg.DB = "pacman/AUR"
				aurPkgs = append(aurPkgs, pkg)
			} else {
				pkg.DB = "pacman"
			}

			// If no install date, try to get from filesystem
			if pkg.Date.IsZero() {
				pkgPath := filepath.Join("/var/lib/pacman/local", pkg.Name+"-"+pkg.Version)
				if info, err := os.Stat(pkgPath); err == nil {
					pkg.Date = info.ModTime()
				}
			}

			load(pkg)
		}

		loadAURNewVersions(aurPkgs, load)
	})

	cmd = exec.Command("pacman", "-Qtdq")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			slog.Error("could not orphan packages", "backend", p.Name(), "err", err)
			return
		}
		slog.Error("could not orphan packages", "backend", p.Name(), "err", err)
		return
	}

	orphanNames := strings.Fields(strings.TrimSpace(string(output)))
	if len(orphanNames) == 0 {
		return
	}

	paragraphs := strings.SplitSeq(string(packagesParagraphOutput), "\n\n")
	for para := range paragraphs {
		if para == "" {
			continue
		}
		pkg := parseParagraph(para)

		if pkg.Name == "" {
			continue
		}
		if !slices.Contains(orphanNames, pkg.Name) {
			continue
		}

		pkg.IsOrphan = true
		pkg.IsDirect = false
		pkg.IsFrozen = pinnedPkgs[pkg.Name]

		if foreignPkgs[pkg.Name] {
			pkg.DB = "pacman/AUR"
		} else {
			pkg.DB = "pacman"
		}

		// If no install date, try to get from filesystem
		if pkg.Date.IsZero() {
			pkgPath := filepath.Join("/var/lib/pacman/local", pkg.Name+"-"+pkg.Version)
			if info, err := os.Stat(pkgPath); err == nil {
				pkg.Date = info.ModTime()
			}
		}

		load(pkg)
	}
}

func loadAURNewVersions(pkgs []Package, load func(Package)) {
	pkgNames := make([]string, 0)
	for _, p := range pkgs {
		pkgNames = append(pkgNames, p.Name)
	}

	newVersions, err := CheckAURVersions(pkgNames)
	if err != nil {
		slog.Warn("Could not check AUR versions", "err", err)
		return
	}
	for i := range pkgs {
		if newVer, ok := newVersions[pkgs[i].Name]; ok && newVer != pkgs[i].Version {
			pkgs[i].NewVersion = newVer
			load(pkgs[i])
		}
	}

}

func parseParagraph(para string) Package {
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
		}
	}
	return pkg
}

func (p *PacmanBackend) Update() (func() error, chan OperationResult) {
	return CreatePrivilegedCmd(p.Name()+"-update", "pacman", "-Syy", "--noconfirm")
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

func (p PacmanBackend) String() string {
	return p.Name()
}
