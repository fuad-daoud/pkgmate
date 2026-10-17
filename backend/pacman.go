//go:build pacman || all_backends

package backend

import (
	"context"
	"errors"
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

// Seams for the fresh update check; tests replace them.
var (
	checkupdatesLookPath  = exec.LookPath
	checkupdatesRun       = runCheckupdates
	checkupdatesTimeout   = 30 * time.Second
	checkupdatesWaitDelay = 3 * time.Second
	checkupdatesAbsent    sync.Once
)

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

		// Fresh check runs concurrently; the first emission below does not wait for it.
		freshCh := make(chan map[string]string, 1)
		go func() { freshCh <- fetchFreshUpdates() }()

		aurPkgs := make([]Package, 0)
		repoPkgs := make([]Package, 0)
		for _, pkg := range parsePacmanQi(string(packagesParagraphOutput)) {
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

			if pkg.DB == "pacman" {
				repoPkgs = append(repoPkgs, pkg)
			}
			load(pkg)
		}

		loadAURNewVersions(aurPkgs, load)

		for _, pkg := range reemitFreshUpdates(repoPkgs, <-freshCh) {
			load(pkg)
		}
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

	for _, pkg := range parsePacmanQi(string(packagesParagraphOutput)) {
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

func parsePacmanQi(out string) []Package {
	var pkgs []Package

	paragraphs := strings.SplitSeq(out, "\n\n")
	for para := range paragraphs {
		if para == "" {
			continue
		}
		pkg := parseParagraph(para)

		if pkg.Name == "" {
			continue
		}

		pkgs = append(pkgs, pkg)
	}

	return pkgs
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

	return parsePacmanIgnorePkg(string(data))
}

func parsePacmanIgnorePkg(conf string) map[string]bool {
	pinned := make(map[string]bool)

	for line := range strings.SplitSeq(conf, "\n") {
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

	return parsePacmanQu(string(output))
}

// runCheckupdates runs checkupdates as the user. The exit code is returned
// separately; err is non-nil only when the process could not run to completion
// (failed to start, or the context ended).
func runCheckupdates(ctx context.Context) (out []byte, code int, err error) {
	cmd := exec.CommandContext(ctx, "checkupdates", "--nocolor")
	// A killed script's fakeroot/pacman children can keep stdout open.
	cmd.WaitDelay = checkupdatesWaitDelay
	out, err = cmd.Output()
	if err == nil {
		return out, 0, nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, -1, ctxErr
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return out, exitErr.ExitCode(), nil
	}
	return nil, -1, err
}

// fetchFreshUpdates returns the fresh update versions from checkupdates, or
// nil when it is absent, reports no updates, or fails (keeping the -Qu data).
func fetchFreshUpdates() map[string]string {
	if _, err := checkupdatesLookPath("checkupdates"); err != nil {
		checkupdatesAbsent.Do(func() {
			slog.Info("checkupdates not found, update markers use the local sync database; install pacman-contrib for fresh ones")
		})
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), checkupdatesTimeout)
	defer cancel()

	out, code, err := checkupdatesRun(ctx)
	if err != nil {
		slog.Warn("checkupdates did not complete, keeping local update data", "err", err)
		return nil
	}
	switch code {
	case 0:
		return parsePacmanQu(string(out))
	case 2: // no updates available
		return nil
	default:
		slog.Warn("checkupdates failed, keeping local update data", "exitCode", code)
		return nil
	}
}

// reemitFreshUpdates returns the repo packages whose fresh new version is
// non-empty and differs from the one already emitted. Markers are never removed.
func reemitFreshUpdates(emitted []Package, fresh map[string]string) []Package {
	var out []Package
	for _, pkg := range emitted {
		if pkg.DB != "pacman" {
			continue
		}
		newVer := fresh[pkg.Name]
		if newVer == "" || newVer == pkg.NewVersion {
			continue
		}
		pkg.NewVersion = newVer
		out = append(out, pkg)
	}
	return out
}

func parsePacmanQu(out string) map[string]string {
	upgradable := make(map[string]string)

	for line := range strings.SplitSeq(out, "\n") {
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

	return time.Time{}
}

func (p PacmanBackend) String() string {
	return p.Name()
}
