package backend

import (
	"time"

	"github.com/Jguer/go-alpm/v2"
)

type Package struct {
	Name    string
	Version string
	Size    int64
	Date    time.Time
}

func LoadPackages() ([]Package, error) {
	h, err := alpm.Initialize("/", "/var/lib/pacman/")
	if err != nil {
		return nil, err
	}
	defer h.Release()

	db, err := h.LocalDB()
	if err != nil {
		return nil, err
	}

	var packages []Package
	db.PkgCache().ForEach(func(p alpm.IPackage) error {
		// if p.Reason() != alpm.PkgReasonExplicit {
		// 	return nil
		// }
		packages = append(packages, Package{
			Name:    p.Name(),
			Version: p.Version(),
			Size:    p.ISize(),
			Date:    p.InstallDate(),
		})
		return nil
	})

	return packages, nil
}
