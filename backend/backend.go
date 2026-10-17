package backend

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var ErrUpdateNotImplemented = errors.New("update not implemented")

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
	send := func(p Package) { outChan <- p }
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
	return updateBackends(GetAvailableBackends(), os.Stdout)
}

func updateBackends(backends []Backend, progress io.Writer) (func() error, chan OperationResult) {
	if progress == nil {
		progress = io.Discard
	}
	resultChan := make(chan OperationResult, 1)

	execFunc := func() error {
		var updated []string
		var skipped []string
		type failedBackend struct {
			name string
			err  error
		}
		var failed []failedBackend
		type backendLog struct {
			name string
			logs string
		}
		var logs []backendLog

		for _, b := range backends {
			name := b.Name()
			fmt.Fprintf(progress, "pkgmate: updating %s…\n", name)

			f, ch := b.Update()
			if f == nil {
				skipped = append(skipped, name)
				fmt.Fprintf(progress, "pkgmate: %s skipped\n", name)
				continue
			}

			err := f()
			if err != nil {
				if errors.Is(err, ErrUpdateNotImplemented) {
					skipped = append(skipped, name)
					fmt.Fprintf(progress, "pkgmate: %s skipped\n", name)
					continue
				}
				failed = append(failed, failedBackend{name: name, err: err})
				fmt.Fprintf(progress, "pkgmate: %s failed\n", name)
				continue
			}

			res := <-ch
			if res.Logs != "" {
				logs = append(logs, backendLog{name: name, logs: res.Logs})
			}

			if res.Success {
				updated = append(updated, name)
				fmt.Fprintf(progress, "pkgmate: %s updated\n", name)
			} else {
				resErr := res.Error
				if resErr == nil {
					resErr = errors.New("update failed")
				}
				failed = append(failed, failedBackend{name: name, err: resErr})
				fmt.Fprintf(progress, "pkgmate: %s failed\n", name)
			}
		}

		var finalErr error
		if len(failed) > 0 {
			errs := make([]error, len(failed))
			for i, f := range failed {
				errs[i] = fmt.Errorf("%s: %w", f.name, f.err)
			}
			finalErr = errors.Join(errs...)
		}

		var sb strings.Builder
		if len(backends) == 0 {
			sb.WriteString("No available backends\n")
		} else {
			if len(updated) > 0 {
				sb.WriteString("Updated: " + strings.Join(updated, ", ") + "\n")
			}
			if len(skipped) > 0 {
				sb.WriteString("Skipped: " + strings.Join(skipped, ", ") + "\n")
			}
			if len(failed) > 0 {
				failedStrs := make([]string, len(failed))
				for i, f := range failed {
					failedStrs[i] = fmt.Sprintf("%s: %v", f.name, f.err)
				}
				sb.WriteString("Failed: " + strings.Join(failedStrs, ", ") + "\n")
			}
			for _, l := range logs {
				sb.WriteString("\n==> " + l.name + "\n")
				sb.WriteString(l.logs)
				if !strings.HasSuffix(l.logs, "\n") {
					sb.WriteString("\n")
				}
			}
		}

		resultChan <- OperationResult{
			Success: len(failed) == 0,
			Error:   finalErr,
			Logs:    sb.String(),
		}

		return nil
	}

	return execFunc, resultChan
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
