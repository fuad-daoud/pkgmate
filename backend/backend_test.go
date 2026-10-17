package backend

import (
	"sync"
	"testing"
)

type fakeNestedBackend struct {
	name     string
	pkgCount int
}

func (f *fakeNestedBackend) Name() string {
	return f.name
}

func (f *fakeNestedBackend) IsAvailable() bool {
	return true
}

func (f *fakeNestedBackend) LoadPackages(wg *sync.WaitGroup, send func(Package)) {
	wg.Go(func() {
		for i := 0; i < f.pkgCount; i++ {
			wg.Go(func() {
				send(Package{
					Name: f.name,
				})
			})
		}
	})
}

func (f *fakeNestedBackend) Update() (func() error, chan OperationResult) {
	return nil, nil
}

func TestLoadAllPackages(t *testing.T) {
	backendMutex.Lock()
	origBackends := registeredBackends
	registeredBackends = []Backend{
		&fakeNestedBackend{name: "fake1", pkgCount: 10},
		&fakeNestedBackend{name: "fake2", pkgCount: 15},
	}
	backendMutex.Unlock()

	t.Cleanup(func() {
		backendMutex.Lock()
		registeredBackends = origBackends
		backendMutex.Unlock()
	})

	outChan, err := LoadAllPackages()
	if err != nil {
		t.Fatalf("unexpected error from LoadAllPackages: %v", err)
	}

	count := 0
	for range outChan {
		count++
	}

	expected := 25
	if count != expected {
		t.Fatalf("expected %d packages, got %d", expected, count)
	}
}
