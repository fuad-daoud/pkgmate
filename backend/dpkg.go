//go:build dpkg

package backend

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

func LoadPackages() (chan []Package, error) {
	start := time.Now()

	data, err := os.ReadFile("/var/lib/dpkg/status")
	if err != nil {
		return nil, fmt.Errorf("failed to read dpkg status: %w", err)
	}

	autoInstalled := getAutoInstalledPackages()

	pkgsChan := make(chan []Package, 1)

	var wg sync.WaitGroup
	wg.Go(func() {
		heldPkgs := getHeldPackages()
		packages := make([]Package, 0)
		paragraphs := strings.SplitSeq(string(data), "\n\n")

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
				listFile := fmt.Sprintf("/var/lib/dpkg/info/%s.list", pkg.Name)
				if info, err := os.Stat(listFile); err == nil {
					pkg.Date = info.ModTime()
				}

				_, isAuto := autoInstalled[pkg.Name]
				pkg.IsDirect = !isAuto
				pkg.DB = "dpkg"
				pkg.IsFrozen = heldPkgs[pkg.Name]

				packages = append(packages, pkg)
			}

		}

		wg.Go(func() { pkgsChan <- packages })

		upgradable := getUpgradableVersions()
		for i := range packages {
			if newVer, ok := upgradable[packages[i].Name]; ok && newVer != packages[i].Version {
				packages[i].NewVersion = newVer
			}
		}

		wg.Go(func() { pkgsChan <- packages })
	})

	go func() {
		wg.Wait()
		close(pkgsChan)
		_ = time.Since(start)
	}()

	return pkgsChan, nil
}

func getAutoInstalledPackages() map[string]bool {
	autoInstalled := make(map[string]bool)

	data, err := os.ReadFile("/var/lib/apt/extended_states")
	if err != nil {
		return autoInstalled
	}

	paragraphs := strings.SplitSeq(string(data), "\n\n")
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

	for line := range strings.SplitSeq(string(output), "\n") {
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

	for line := range strings.SplitSeq(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == "hold" {
			held[fields[0]] = true
		}
	}

	return held
}

func Update() (func() error, chan OperationResult) {
	if _, err := exec.LookPath("apt-get"); err == nil {
		return CreatePrivilegedCmd("update", "apt-get", "update")
	}

	if _, err := exec.LookPath("apt"); err == nil {
		return CreatePrivilegedCmd("update", "apt", "update")
	}

	resultChan := make(chan OperationResult, 1)
	err := fmt.Errorf("neither apt-get nor apt found")

	return func() error {
		resultChan <- OperationResult{Error: err}
		return err
	}, resultChan
}
