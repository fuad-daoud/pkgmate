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
		lines := strings.Split(para, "\n")

		for _, line := range lines {
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
