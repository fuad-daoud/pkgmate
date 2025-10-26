//go:build dpkg

package backend

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func LoadPackages() ([]Package, error) {
	data, err := os.ReadFile("/var/lib/dpkg/status")
	if err != nil {
		return nil, fmt.Errorf("failed to read dpkg status: %w", err)
	}

	var packages []Package
	paragraphs := strings.SplitSeq(string(data), "\n\n")

	for para := range paragraphs {
		if para == "" {
			continue
		}

		// Only include installed packages
		if !strings.Contains(para, "Status: install ok installed") {
			continue
		}

		var pkg Package
		lines := strings.SplitSeq(para, "\n")

		for line := range lines {
			// Handle continuation lines (start with space)
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
					pkg.Size = size * 1024 // Convert KB to bytes
				}
			}
		}

		// Get install date from .list file
		if pkg.Name != "" {
			listFile := fmt.Sprintf("/var/lib/dpkg/info/%s.list", pkg.Name)
			if info, err := os.Stat(listFile); err == nil {
				pkg.Date = info.ModTime()
			}

			packages = append(packages, pkg)
		}
	}

	return packages, nil
}

func LoadDirectPackages() ([]Package, error) {
	autoInstalled := getAutoInstalledPackages()

	data, err := os.ReadFile("/var/lib/dpkg/status")
	if err != nil {
		return nil, fmt.Errorf("failed to read dpkg status: %w", err)
	}

	var packages []Package
	paragraphs := strings.SplitSeq(string(data), "\n\n")

	for para := range paragraphs {
		if para == "" {
			continue
		}

		if !strings.Contains(para, "Status: install ok installed") {
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
			// Only include manually installed packages
			if _, isAuto := autoInstalled[pkg.Name]; isAuto {
				continue
			}

			listFile := fmt.Sprintf("/var/lib/dpkg/info/%s.list", pkg.Name)
			if info, err := os.Stat(listFile); err == nil {
				pkg.Date = info.ModTime()
			}

			packages = append(packages, pkg)
		}
	}

	return packages, nil
}

func LoadDepedencyPackages() ([]Package, error) {
	autoInstalled := getAutoInstalledPackages()

	data, err := os.ReadFile("/var/lib/dpkg/status")
	if err != nil {
		return nil, fmt.Errorf("failed to read dpkg status: %w", err)
	}

	var packages []Package
	paragraphs := strings.SplitSeq(string(data), "\n\n")

	for para := range paragraphs {
		if para == "" {
			continue
		}

		if !strings.Contains(para, "Status: install ok installed") {
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
			// Only include auto-installed packages
			if _, isAuto := autoInstalled[pkg.Name]; !isAuto {
				continue
			}

			listFile := fmt.Sprintf("/var/lib/dpkg/info/%s.list", pkg.Name)
			if info, err := os.Stat(listFile); err == nil {
				pkg.Date = info.ModTime()
			}

			packages = append(packages, pkg)
		}
	}

	return packages, nil
}

// getAutoInstalledPackages reads the APT extended_states file to determine
// which packages were automatically installed as dependencies
func getAutoInstalledPackages() map[string]bool {
	autoInstalled := make(map[string]bool)

	data, err := os.ReadFile("/var/lib/apt/extended_states")
	if err != nil {
		// If file doesn't exist, assume no packages are auto-installed
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
			if after, ok :=strings.CutPrefix(line, "Package: "); ok  {
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
