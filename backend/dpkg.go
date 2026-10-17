package backend

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type DpkgBackend struct{}

func init() {
	RegisterBackend(&DpkgBackend{})
}

func (d *DpkgBackend) Name() string {
	return "dpkg"
}

func (d *DpkgBackend) IsAvailable() bool {
	if _, err := os.Stat("/var/lib/dpkg/status"); err != nil {
		return false
	}
	if _, err := exec.LookPath("apt"); err == nil {
		return true
	}
	if _, err := exec.LookPath("apt-get"); err == nil {
		return true
	}
	return false
}

func (d *DpkgBackend) LoadPackages(wg *sync.WaitGroup, load func(Package)) {
	start := time.Now()

	data, err := os.ReadFile("/var/lib/dpkg/status")
	if err != nil {
		slog.Error("failed to read dpkg status", "err", err)
		return
	}

	autoInstalled := getAutoInstalledPackages()
	heldPkgs := getHeldPackages()
	upgradable := getUpgradableVersions()
	orphanNames := getOrphanPackageNames()

	wg.Go(func() {
		for _, pkg := range parseDpkgStatus(string(data)) {
			if pkg.Name != "" {
				listFile := fmt.Sprintf("/var/lib/dpkg/info/%s.list", pkg.Name)
				if info, err := os.Stat(listFile); err == nil {
					pkg.Date = info.ModTime()
				}

				_, isAuto := autoInstalled[pkg.Name]
				pkg.IsDirect = !isAuto
				pkg.DB = "apt/dpkg"
				pkg.IsFrozen = heldPkgs[pkg.Name]
				pkg.IsOrphan = orphanNames[pkg.Name]

				if newVer, ok := upgradable[pkg.Name]; ok && newVer != pkg.Version {
					pkg.NewVersion = newVer
				}

				load(pkg)
			}
		}

		slog.Info("loaded packages from dpkg", "time", time.Since(start))
	})
}

func parseDpkgStatus(data string) []Package {
	var pkgs []Package

	paragraphs := strings.SplitSeq(data, "\n\n")

	for para := range paragraphs {
		if para == "" {
			continue
		}

		if !strings.Contains(para, "Status: install ok installed") && !strings.Contains(para, "Status: hold ok installed") {
			continue
		}

		var pkg Package
		lines := strings.SplitSeq(para, "\n")

		for line := range lines {
			if strings.HasPrefix(line, " ") {
				continue
			}

			parts := strings.SplitN(line, ": ", 2)
			if len(parts) != 2 {
				continue
			}

			key := parts[0]
			value := parts[1]

			switch key {
			case "Package":
				pkg.Name = value
			case "Version":
				pkg.Version = value
			case "Installed-Size":
				if size, err := strconv.ParseInt(value, 10, 64); err == nil {
					pkg.Size = size * 1024
				}
			}
		}

		if pkg.Name != "" {
			pkgs = append(pkgs, pkg)
		}
	}

	return pkgs
}

func getAutoInstalledPackages() map[string]bool {
	autoInstalled := make(map[string]bool)

	data, err := os.ReadFile("/var/lib/apt/extended_states")
	if err != nil {
		return autoInstalled
	}

	return parseAptExtendedStates(string(data))
}

func parseAptExtendedStates(data string) map[string]bool {
	autoInstalled := make(map[string]bool)

	paragraphs := strings.SplitSeq(data, "\n\n")
	for para := range paragraphs {
		if para == "" {
			continue
		}

		var packageName string
		var isAuto bool

		lines := strings.SplitSeq(para, "\n")
		for line := range lines {
			if after, ok := strings.CutPrefix(line, "Package: "); ok {
				packageName = after
			} else if strings.HasPrefix(line, "Auto-Installed: 1") {
				isAuto = true
			}
		}

		if packageName != "" && isAuto {
			autoInstalled[packageName] = true
		}
	}

	return autoInstalled
}

func getUpgradableVersions() map[string]string {
	available := make(map[string]string)

	cmd := exec.Command("apt", "list", "--upgradable")
	output, err := cmd.Output()
	if err != nil {
		return available
	}

	return parseAptListUpgradable(string(output))
}

func parseAptListUpgradable(out string) map[string]string {
	available := make(map[string]string)

	for line := range strings.SplitSeq(out, "\n") {
		if line == "" || strings.HasPrefix(line, "Listing") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		name := strings.Split(fields[0], "/")[0]
		version := fields[1]
		available[name] = version
	}

	return available
}

func getHeldPackages() map[string]bool {
	held := make(map[string]bool)

	cmd := exec.Command("dpkg", "--get-selections")
	output, err := cmd.Output()
	if err != nil {
		return held
	}

	return parseDpkgSelections(string(output))
}

func parseDpkgSelections(out string) map[string]bool {
	held := make(map[string]bool)

	for line := range strings.SplitSeq(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == "hold" {
			held[fields[0]] = true
		}
	}

	return held
}

func getOrphanPackageNames() map[string]bool {
	orphans := make(map[string]bool)

	cmd := exec.Command("apt-get", "--simulate", "autoremove")
	output, err := cmd.Output()
	if err != nil {
		return orphans
	}

	return parseAptAutoremoveSim(string(output))
}

func parseAptAutoremoveSim(out string) map[string]bool {
	orphans := make(map[string]bool)

	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Remv ") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		orphans[fields[1]] = true
	}

	return orphans
}
func (d DpkgBackend) String() string {
	return d.Name()
}
