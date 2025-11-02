package backend

import (
	"fmt"
	"time"
)

type Package struct {
	Name       string
	Version    string
	NewVersion string
	Size       int64
	DB         string
	Date       time.Time
	IsDirect   bool
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
func (p Package) FormatVersion() string {
	if p.NewVersion != "" {
		return fmt.Sprintf("%s->%s", p.Version, p.NewVersion)
	}
	return p.Version
}
