package backend

import (
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"sync"
	"time"
)

type Package struct {
	Name         string
	Version      string
	NewVersion   string
	Size         int64
	DB           string
	Date         time.Time
	IsDirect     bool
	IsFrozen     bool
	Dependencies []string
	IsOrphan     bool
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

type Backend interface {
	Name() string
	IsAvailable() bool
	LoadPackages(*sync.WaitGroup, func(Package))
	Update() (func() error, chan OperationResult)
}

var (
	registeredBackends = make([]Backend, 0)
	backendMutex       sync.RWMutex
)

func RegisterBackend(backend Backend) {
	backendMutex.Lock()
	defer backendMutex.Unlock()
	registeredBackends = append(registeredBackends, backend)
}

func GetAvailableBackends() []Backend {
	backendMutex.RLock()
	defer backendMutex.RUnlock()

	available := make([]Backend, 0)
	for _, backend := range registeredBackends {
		if backend.IsAvailable() {
			available = append(available, backend)
		}
	}
	return available
}

func LoadAllPackages() (chan Package, error) {
	backends := GetAvailableBackends()
	slog.Info("Available backends", "backends", backends)
	if len(backends) == 0 {
		return nil, nil
	}

	outChan := make(chan Package)
	start := time.Now()

	var wg sync.WaitGroup
	send := func(p Package) {outChan <- p}
	for _, backend := range backends {
		wg.Go(func() {
			backend.LoadPackages(&wg, send)
			slog.Info("loaded pacakges from", "backend", backend.Name(), "time", time.Since(start))
		})
	}

	go func() {
		wg.Wait()
		close(outChan)
		slog.Info("Loaded all packages from backend", "time", time.Since(start))
	}()

	return outChan, nil
}

func UpdateAll() (func() error, chan OperationResult) {
	panic("Not Implemented")
}

func calculateSize(path string) int64 {
	var size int64
	err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return fs.SkipDir
		}
		if !d.IsDir() {
			if info, err := d.Info(); err == nil {
				size += info.Size()
			}

		}
		return nil
	})
	if err != nil {
		slog.Warn("could not calculate size of path", "path", path)
	}
	return size
}
