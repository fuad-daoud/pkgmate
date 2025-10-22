package backend

import (
	"fmt"
	"time"
)


type Package struct {
	Name    string
	Version string
	Size    int64
	Date    time.Time
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
