//go:build dummy

package backend

import "time"

type DummyBackend struct{}

func init() {
	RegisterBackend(&DummyBackend{})
}

func (DummyBackend) Name() string {
	return "Dummy"
}
func (DummyBackend) IsAvailable() bool {
	return true
}

func (DummyBackend) LoadPackages() (chan []Package, error) {
	time.Sleep(30 * time.Second)
	return nil, nil
}

func (DummyBackend) GetOrphanPackages() ([]Package, error) {
	return nil, nil
}
func (DummyBackend) Update() (func() error, chan OperationResult) {
	return nil, nil
}
func (DummyBackend) String() string {
	return "Dummy"
}
