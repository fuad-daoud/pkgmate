//go:build arch

package backend

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/Jguer/go-alpm/v2"
)

type AlpmBackend struct{}

func init() {
	RegisterBackend(&AlpmBackend{})
}

func (a *AlpmBackend) Name() string {
	return "alpm"
}

func (a *AlpmBackend) IsAvailable() bool {
	if _, err := os.Stat("/var/lib/pacman/"); err != nil {
		return false
	}
	if _, err := exec.LookPath("pacman"); err != nil {
		return false
	}
	return true
}

func (a *AlpmBackend) LoadPackages() (chan []Package, error) {
	var wg sync.WaitGroup
	pkgsChan := make(chan []Package, 2)
	h, err := alpm.Initialize("/", "/var/lib/pacman/")
	if err != nil {
		return nil, err
	}
	start := time.Now()
	wg.Go(func() {

		if _, err := h.RegisterSyncDB("core", 0); err != nil {
			slog.Warn("Failed to register core", "err", err)
		}
		if _, err := h.RegisterSyncDB("extra", 0); err != nil {
			slog.Warn("Failed to register extra", "err", err)
		}
		if _, err := h.RegisterSyncDB("multilib", 0); err != nil {
			slog.Warn("Failed to register multilib", "err", err)
		}

		syncDBs, err := h.SyncDBs()
		if err != nil {
			return
		}

		localDB, err := h.LocalDB()
		if err != nil {
			return
		}
		pkgs := localDB.PkgCache().Slice()
		aurPackages := make([]Package, 0)
		aurPackagesNames := make([]string, 0)
		packages := make([]Package, 0)

		pinnedPkgs, err := getPinnedPackages(h)
		if err != nil {
			slog.Warn("could not get forzen packages", "err", err)
		}
		for _, p := range pkgs {
			pkg := Package{
				Name:     p.Name(),
				Version:  p.Version(),
				Size:     p.ISize(),
				DB:       p.DB().Name(),
				Date:     p.InstallDate(),
				IsFrozen: pinnedPkgs[p.Name()],
				IsDirect: p.Reason() == alpm.PkgReasonExplicit,
			}

			if newPkg := p.SyncNewVersion(syncDBs); newPkg != nil {
				pkg.NewVersion = newPkg.Version()
			} else if isAURPackage(p.Name(), syncDBs) {
				pkg.DB = "AUR"
				aurPackages = append(aurPackages, pkg)
				aurPackagesNames = append(aurPackagesNames, pkg.Name)
				continue
			}
			packages = append(packages, pkg)
		}
		wg.Go(func() { pkgsChan <- packages })
		newVersions, err := CheckAURVersions(aurPackagesNames)
		if err != nil {
			slog.Warn("Could not update aur packages")
		} else {
			for i, pkg := range aurPackages {
				if aurPackages[i].Version != newVersions[pkg.Name] {
					aurPackages[i].NewVersion = newVersions[pkg.Name]
				}
			}
		}

		wg.Go(func() { pkgsChan <- aurPackages })
	})
	go func() {
		defer h.Release()
		wg.Wait()
		close(pkgsChan)
		slog.Info("time to map all pacakges and sync with aur", "time", time.Since(start))
	}()

	return pkgsChan, nil
}

func (a *AlpmBackend) Update() (func() error, chan OperationResult) {
	return CreatePrivilegedCmd(a.Name()+"-update", "pacman", "-Syy", "--noconfirm")
}

func (a *AlpmBackend) GetOrphanPackages() ([]Package, error) {
	cmd := exec.Command("pacman", "-Qtdq")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return []Package{}, nil
		}
		return nil, fmt.Errorf("failed to get orphans: %w", err)
	}

	h, err := alpm.Initialize("/", "/var/lib/pacman/")
	if err != nil {
		return nil, err
	}
	defer h.Release()

	localDB, err := h.LocalDB()
	if err != nil {
		return nil, err
	}

	pinnedPkgs, _ := getPinnedPackages(h)

	orphans := make([]Package, 0)
	for name := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
		if name == "" {
			continue
		}

		pkg := localDB.Pkg(name)
		if pkg == nil {
			continue
		}

		orphans = append(orphans, Package{
			Name:     pkg.Name(),
			Version:  pkg.Version(),
			Size:     pkg.ISize(),
			DB:       pkg.DB().Name(),
			Date:     pkg.InstallDate(),
			IsDirect: false,
			IsFrozen: pinnedPkgs[pkg.Name()],
		})
	}

	return orphans, nil
}

func getPinnedPackages(h *alpm.Handle) (map[string]bool, error) {
	pinned := make(map[string]bool)

	ignorePkgs, err := h.IgnorePkgs()
	if err != nil {
		return pinned, err
	}
	s := ignorePkgs.Slice()
	for _, pkg := range s {
		pinned[pkg] = true
	}
	data, err := os.ReadFile("/etc/pacman.conf")
	if err != nil {
		return pinned, err
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

	return pinned, nil
}

func isAURPackage(pkgName string, syncDBs alpm.IDBList) bool {
	found := false
	syncDBs.ForEach(func(db alpm.IDB) error {
		if db.Pkg(pkgName) != nil {
			found = true
		}
		return nil
	})
	return !found
}
