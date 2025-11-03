//go:build arch

package backend

import (
	"log/slog"
	"os/exec"
	"sync"
	"time"

	"github.com/Jguer/go-alpm/v2"
)

const PACKAGES_CACHE_KEY = "cache_key"

func LoadPackages() (chan []Package, error) {
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
		for _, p := range pkgs {
			pkg := Package{
				Name:     p.Name(),
				Version:  p.Version(),
				Size:     p.ISize(),
				DB:       p.DB().Name(),
				Date:     p.InstallDate(),
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
				aurPackages[i].NewVersion = newVersions[pkg.Name]
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

func Update() error {
	return exec.Command("pacman", "-Sy").Run()
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
