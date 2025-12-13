package backend

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type SnapBackend struct{}

func init() {
	RegisterBackend(&SnapBackend{})
}

func (s *SnapBackend) Name() string {
	return "snap"
}

func (s *SnapBackend) IsAvailable() bool {
	if _, err := exec.LookPath("snap"); err != nil {
		return false
	}
	cmd := exec.Command("snap", "list")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func (s *SnapBackend) LoadPackages(wg *sync.WaitGroup, load func(Package)) {
	start := time.Now()

	wg.Go(func() {
		cmd := exec.Command("snap", "list")
		output, err := cmd.Output()
		if err != nil {
			slog.Warn("Failed to list snap packages", "err", err)
			return
		}

		heldPkgs := getHeldSnaps()
		updatable := getUpdatableSnaps()

		lines := strings.SplitSeq(string(output), "\n")

		first := true
		for line := range lines {
			if first {
				first = false
				continue
			}

			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			fields := strings.Fields(line)
			if len(fields) < 3 {
				continue
			}

			name := fields[0]
			version := fields[1]

			notes := ""
			if len(fields) >= 6 {
				notes = fields[5]
			}

			isDirect := isDirectInstall(name, notes)
			size := getSnapSize(name)
			installDate := getSnapInstallDate(name)

			pkg := Package{
				Name:     name,
				Version:  version,
				Size:     size,
				Date:     installDate,
				IsDirect: isDirect,
				IsFrozen: heldPkgs[name],
				IsOrphan: false,
				DB:       "snap",
			}

			if newVer, ok := updatable[pkg.Name]; ok && newVer != pkg.Version {
				pkg.NewVersion = newVer
			}

			load(pkg)
		}

		slog.Info("loaded packages from snap", "time", time.Since(start))
	})
}

func (s *SnapBackend) Update() (func() error, chan OperationResult) {
	resultChan := make(chan OperationResult, 1)
	err := fmt.Errorf("snap update not implemented")

	return func() error {
		resultChan <- OperationResult{Error: err}
		return err
	}, resultChan
}

func isDirectInstall(name, notes string) bool {
	notes = strings.ToLower(notes)
	name = strings.ToLower(name)

	if notes == "base" || notes == "snapd" || notes == "core" {
		return false
	}

	dependencyPrefixes := []string{"gnome-", "kde-", "gtk-common-", "core", "bare", "snapd"}
	for _, prefix := range dependencyPrefixes {
		if strings.HasPrefix(name, prefix) {
			return false
		}
	}

	return true
}

func getSnapSize(name string) int64 {
	snapPath := filepath.Join("/var/lib/snapd/snaps", name+"_*.snap")
	matches, err := filepath.Glob(snapPath)
	if err != nil || len(matches) == 0 {
		return 0
	}

	var latestSnap string
	var latestTime time.Time
	for _, match := range matches {
		if info, err := os.Stat(match); err == nil {
			if info.ModTime().After(latestTime) {
				latestTime = info.ModTime()
				latestSnap = match
			}
		}
	}

	if latestSnap == "" {
		return 0
	}

	info, err := os.Stat(latestSnap)
	if err != nil {
		return 0
	}

	return info.Size()
}

func getSnapInstallDate(name string) time.Time {
	snapPath := filepath.Join("/var/lib/snapd/snaps", name+"_*.snap")
	matches, err := filepath.Glob(snapPath)
	if err != nil || len(matches) == 0 {
		return time.Now()
	}

	var latestTime time.Time
	for _, match := range matches {
		if info, err := os.Stat(match); err == nil {
			if info.ModTime().After(latestTime) {
				latestTime = info.ModTime()
			}
		}
	}

	if latestTime.IsZero() {
		return time.Now()
	}

	return latestTime
}

func getHeldSnaps() map[string]bool {
	held := make(map[string]bool)

	cmd := exec.Command("snap", "refresh", "--list")
	output, err := cmd.Output()
	if err != nil {
		return held
	}

	lines := strings.SplitSeq(string(output), "\n")
	for line := range lines {
		if strings.Contains(line, "held") || strings.Contains(line, "hold") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				held[fields[0]] = true
			}
		}
	}

	return held
}

func getUpdatableSnaps() map[string]string {
	updatable := make(map[string]string)

	cmd := exec.Command("snap", "refresh", "--list")
	output, err := cmd.Output()
	if err != nil {
		return updatable
	}

	lines := strings.SplitSeq(string(output), "\n")

	first := true
	for line := range lines {
		if first {
			first = false
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		name := fields[0]
		newVersion := fields[1]
		updatable[name] = newVersion
	}

	return updatable
}
func (s SnapBackend) String() string {
	return s.Name()
}
