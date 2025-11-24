package backend

import (
	"fmt"
	"log/slog"
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
	LoadPackages() (chan []Package, error)
	Update() (func() error, chan OperationResult)
	GetOrphanPackages() ([]Package, error)
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

func LoadAllPackages() (chan []Package, error) {
	backends := GetAvailableBackends()
	if len(backends) == 0 {
		return nil, nil
	}

	outChan := make(chan []Package, len(backends)*2)

	var wg sync.WaitGroup
	for _, backend := range backends {
		wg.Go(func() {
			pkgChan, err := backend.LoadPackages()
			if err != nil {
				return
			}
			for pkgs := range pkgChan {
				outChan <- pkgs
			}
		})
	}

	go func() {
		wg.Wait()
		close(outChan)
	}()

	return outChan, nil
}

func UpdateAll() (func() error, chan OperationResult) {
	panic("Not Implemented")
}

func GetAllOrphanPackages() (chan []Package, error) {
	backends := GetAvailableBackends()
	pkgsChan := make(chan []Package, len(backends))

	var wg sync.WaitGroup

	for _, backend := range backends {
		wg.Go(func() {
			orphans, err := backend.GetOrphanPackages()
			if err != nil {
				slog.Warn("Failed to get orphans", "backend", backend.Name(), "err", err)
				return
			}
			if len(orphans) > 0 {
				pkgsChan <- orphans
			}
		})
	}

	go func() {
		wg.Wait()
		close(pkgsChan)
	}()

	return pkgsChan, nil
}
