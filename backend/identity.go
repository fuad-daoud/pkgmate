package backend

import (
	"sort"
	"strings"
)

var flatpakAliases = map[string]string{
	// Spec required
	"com.spotify.Client":    "app:spotify",
	"org.telegram.desktop":  "app:telegram-desktop",
	"com.google.Chrome":     "app:google-chrome",
	"com.visualstudio.code": "app:code",
	// Fixture required
	"com.obsproject.Studio": "app:obs-studio",
	"org.gnome.Calculator":  "app:gnome-calculator",
	// Well-known apps whose last segment does not normalise to the common name
	"com.bitwarden.desktop":     "app:bitwarden",
	"com.mattermost.Desktop":    "app:mattermost-desktop",
	"io.github.shiftey.Desktop": "app:github-desktop",
	"org.signal.Signal":         "app:signal-desktop",
	"org.gnome.Boxes":           "app:gnome-boxes",
	"org.gnome.Calendar":        "app:gnome-calendar",
	"org.gnome.Characters":      "app:gnome-characters",
	"org.gnome.Contacts":        "app:gnome-contacts",
	"org.gnome.Maps":            "app:gnome-maps",
	"org.gnome.SystemMonitor":   "app:gnome-system-monitor",
	"org.gnome.TextEditor":      "app:gnome-text-editor",
	"org.gnome.Weather":         "app:gnome-weather",
}

func normalizeKeyName(s string) string {
	s = strings.ToLower(s)
	return strings.ReplaceAll(s, "_", "-")
}

func isVersionSuffix(s string) bool {
	if len(s) < 2 || s[0] != 'v' {
		return false
	}
	for i := 1; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// IdentityKeys returns the duplicate-detection keys for a package.
func IdentityKeys(p Package) []string {
	if p.Name == "" {
		return nil
	}

	switch p.DB {
	case "pacman", "pacman/AUR":
		norm := normalizeKeyName(p.Name)
		if norm == "" {
			return nil
		}
		if strings.HasPrefix(norm, "python-") {
			name := strings.TrimPrefix(norm, "python-")
			if name == "" {
				return nil
			}
			return []string{"py:" + name}
		}
		if strings.HasSuffix(norm, "-bin") {
			norm = strings.TrimSuffix(norm, "-bin")
		} else if strings.HasSuffix(norm, "-git") {
			norm = strings.TrimSuffix(norm, "-git")
		} else if strings.HasSuffix(norm, "-appimage") {
			norm = strings.TrimSuffix(norm, "-appimage")
		}
		if norm == "" {
			return nil
		}
		return []string{"app:" + norm}

	case "apt/dpkg":
		norm := normalizeKeyName(p.Name)
		if norm == "" {
			return nil
		}
		if strings.HasPrefix(norm, "python3-") {
			name := strings.TrimPrefix(norm, "python3-")
			if name == "" {
				return nil
			}
			return []string{"py:" + name}
		}
		return []string{"app:" + norm}

	case "brew", "brew/Cask", "snap":
		norm := normalizeKeyName(p.Name)
		if norm == "" {
			return nil
		}
		return []string{"app:" + norm}

	case "pip":
		norm := normalizeKeyName(p.Name)
		if norm == "" {
			return nil
		}
		return []string{"py:" + norm}

	case "pipx":
		norm := normalizeKeyName(p.Name)
		if norm == "" {
			return nil
		}
		return []string{"py:" + norm, "app:" + norm}

	case "npm":
		norm := normalizeKeyName(p.Name)
		if norm == "" {
			return nil
		}
		return []string{"app:" + norm}

	case "golang":
		cleanPath := strings.Trim(p.Name, "/")
		if cleanPath == "" {
			return nil
		}
		parts := strings.Split(cleanPath, "/")
		last := parts[len(parts)-1]
		if len(parts) > 1 && isVersionSuffix(last) {
			last = parts[len(parts)-2]
		}
		norm := normalizeKeyName(last)
		if norm == "" {
			return nil
		}
		return []string{"app:" + norm}

	case "flatpak":
		if p.ID == "" {
			return nil
		}
		if alias, ok := flatpakAliases[p.ID]; ok {
			return []string{alias}
		}
		lastSegment := p.ID
		if idx := strings.LastIndex(p.ID, "."); idx >= 0 {
			lastSegment = p.ID[idx+1:]
		}
		norm := normalizeKeyName(lastSegment)
		switch norm {
		case "client", "desktop", "app", "application", "":
			return nil
		default:
			return []string{"app:" + norm}
		}

	default:
		return nil
	}
}

// ManagerFamily returns the canonical manager family for a backend DB name.
func ManagerFamily(db string) string {
	switch db {
	case "pacman/AUR":
		return "pacman"
	case "brew/Cask":
		return "brew"
	default:
		return db
	}
}

// DuplicateGroup represents a group of packages installed across multiple package managers.
type DuplicateGroup struct {
	Key      string
	Packages []Package
	Size     int64
}

// FindDuplicates finds groups of packages installed across more than one package manager.
func FindDuplicates(pkgs []Package) []DuplicateGroup {
	if len(pkgs) == 0 {
		return nil
	}

	parent := make([]int, len(pkgs))
	for i := range parent {
		parent[i] = i
	}

	var find func(int) int
	find = func(i int) int {
		root := i
		for parent[root] != root {
			root = parent[root]
		}
		for parent[i] != root {
			next := parent[i]
			parent[i] = root
			i = next
		}
		return root
	}

	union := func(i, j int) {
		rootI := find(i)
		rootJ := find(j)
		if rootI != rootJ {
			parent[rootJ] = rootI
		}
	}

	keyToIndices := make(map[string][]int)
	pkgKeys := make([][]string, len(pkgs))
	for i, p := range pkgs {
		keys := IdentityKeys(p)
		pkgKeys[i] = keys
		for _, k := range keys {
			keyToIndices[k] = append(keyToIndices[k], i)
		}
	}

	for _, indices := range keyToIndices {
		if len(indices) > 1 {
			first := indices[0]
			for _, idx := range indices[1:] {
				union(first, idx)
			}
		}
	}

	components := make(map[int][]int)
	for i := range pkgs {
		root := find(i)
		components[root] = append(components[root], i)
	}

	var groups []DuplicateGroup

	for _, indices := range components {
		if len(indices) < 2 {
			continue
		}

		families := make(map[string]bool)
		hasDirect := false
		for _, idx := range indices {
			p := pkgs[idx]
			families[ManagerFamily(p.DB)] = true
			if p.IsDirect {
				hasDirect = true
			}
		}

		if len(families) < 2 || !hasDirect {
			continue
		}

		keyCounts := make(map[string]int)
		for _, idx := range indices {
			seen := make(map[string]bool)
			for _, k := range pkgKeys[idx] {
				if !seen[k] {
					seen[k] = true
					keyCounts[k]++
				}
			}
		}

		repKey := ""
		for k, count := range keyCounts {
			if count >= 2 {
				if repKey == "" || k < repKey {
					repKey = k
				}
			}
		}

		groupPkgs := make([]Package, len(indices))
		var totalSize int64
		for i, idx := range indices {
			p := pkgs[idx]
			groupPkgs[i] = p
			totalSize += p.Size
		}

		sort.Slice(groupPkgs, func(i, j int) bool {
			if groupPkgs[i].DB != groupPkgs[j].DB {
				return groupPkgs[i].DB < groupPkgs[j].DB
			}
			return groupPkgs[i].Name < groupPkgs[j].Name
		})

		groups = append(groups, DuplicateGroup{
			Key:      repKey,
			Packages: groupPkgs,
			Size:     totalSize,
		})
	}

	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Size != groups[j].Size {
			return groups[i].Size > groups[j].Size
		}
		return groups[i].Key < groups[j].Key
	})

	return groups
}
