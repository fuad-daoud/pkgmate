//go:build brew

package backend

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type brewInfo struct {
	Formulae []struct {
		Name      string `json:"name"`
		Installed []struct {
			Version   string `json:"version"`
			OnRequest bool   `json:"installed_on_request"`
			Time      int64  `json:"time"`
		} `json:"installed"`
	} `json:"formulae"`
	Casks []struct {
		Token   string `json:"token"`
		Version string `json:"version"`
	} `json:"casks"`
}

func LoadPackages() ([]Package, error) {
	cmd := exec.Command("brew", "info", "--json=v2", "--installed")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run brew info: %w", err)
	}

	var info brewInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return nil, fmt.Errorf("failed to parse brew output: %w", err)
	}

	var packages []Package

	// Process formulae (CLI tools, libraries)
	for _, formula := range info.Formulae {
		if len(formula.Installed) == 0 {
			continue
		}

		installed := formula.Installed[0]
		size := getPackageSize(formula.Name)

		installDate := time.Now()
		if installed.Time > 0 {
			installDate = time.Unix(installed.Time, 0)
		}

		packages = append(packages, Package{
			Name:    formula.Name,
			Version: installed.Version,
			Size:    size,
			Date:    installDate,
		})
	}

	// Process casks (GUI applications)
	for _, cask := range info.Casks {
		size := getPackageSize(cask.Token)

		packages = append(packages, Package{
			Name:    cask.Token,
			Version: cask.Version,
			Size:    size,
			Date:    time.Now(),
		})
	}

	return packages, nil
}

func getPackageSize(name string) int64 {
	// Get brew prefix
	prefixCmd := exec.Command("brew", "--prefix", name)
	prefixOutput, err := prefixCmd.Output()
	if err != nil {
		return 0
	}

	prefix := strings.TrimSpace(string(prefixOutput))

	// Get directory size
	cmd := exec.Command("du", "-sk", prefix)
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	parts := strings.Fields(string(output))
	if len(parts) == 0 {
		return 0
	}

	size, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0
	}

	return size * 1024 // Convert KB to bytes
}
