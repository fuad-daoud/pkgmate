package backend

import (
	"fmt"
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

func (p Package) FormatSize() string {
	const unit = 1024
	if p.Size < unit {
		return fmt.Sprintf("%d B", p.Size)
	}
	div, exp := int64(unit), 0
	for n := p.Size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(p.Size)/float64(div), "KMGTPE"[exp])
}
