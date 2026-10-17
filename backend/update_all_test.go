package backend

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeBackend struct {
	name        string
	isAvailable bool
	updateFn    func() (func() error, chan OperationResult)
}

func (f *fakeBackend) Name() string                                        { return f.name }
func (f *fakeBackend) IsAvailable() bool                                   { return f.isAvailable }
func (f *fakeBackend) LoadPackages(wg *sync.WaitGroup, load func(Package)) {}
func (f *fakeBackend) Update() (func() error, chan OperationResult) {
	if f.updateFn != nil {
		return f.updateFn()
	}
	return nil, nil
}

// (a) all succeed -> Success, Error == nil, every name under Updated
func TestUpdateBackends_AllSucceed(t *testing.T) {
	b1 := &fakeBackend{
		name: "b1",
		updateFn: func() (func() error, chan OperationResult) {
			ch := make(chan OperationResult, 1)
			return func() error {
				ch <- OperationResult{Success: true, Logs: "b1 output log\n"}
				return nil
			}, ch
		},
	}
	b2 := &fakeBackend{
		name: "b2",
		updateFn: func() (func() error, chan OperationResult) {
			ch := make(chan OperationResult, 1)
			return func() error {
				ch <- OperationResult{Success: true, Logs: "b2 output log\n"}
				return nil
			}, ch
		},
	}

	var progress bytes.Buffer
	exec, resChan := updateBackends([]Backend{b1, b2}, &progress)
	if err := exec(); err != nil {
		t.Fatalf("expected exec() to return nil, got: %v", err)
	}

	select {
	case res := <-resChan:
		if !res.Success {
			t.Errorf("expected res.Success == true, got false")
		}
		if res.Error != nil {
			t.Errorf("expected res.Error == nil, got %v", res.Error)
		}
		if !strings.Contains(res.Logs, "Updated: b1, b2") {
			t.Errorf("expected Updated: b1, b2 in logs, got:\n%s", res.Logs)
		}
		if !strings.Contains(res.Logs, "==> b1\nb1 output log") {
			t.Errorf("expected b1 logs section, got:\n%s", res.Logs)
		}
		if !strings.Contains(res.Logs, "==> b2\nb2 output log") {
			t.Errorf("expected b2 logs section, got:\n%s", res.Logs)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for result")
	}

	progStr := progress.String()
	if !strings.Contains(progStr, "pkgmate: updating b1…") || !strings.Contains(progStr, "pkgmate: b1 updated") {
		t.Errorf("expected b1 progress lines, got:\n%s", progStr)
	}
	if !strings.Contains(progStr, "pkgmate: updating b2…") || !strings.Contains(progStr, "pkgmate: b2 updated") {
		t.Errorf("expected b2 progress lines, got:\n%s", progStr)
	}
}

// (b) one fails -> !Success, Error mentions that backend, the others still ran
func TestUpdateBackends_OneFails(t *testing.T) {
	b3Ran := false
	b1 := &fakeBackend{
		name: "b1",
		updateFn: func() (func() error, chan OperationResult) {
			ch := make(chan OperationResult, 1)
			return func() error {
				ch <- OperationResult{Success: true, Logs: "b1 ok\n"}
				return nil
			}, ch
		},
	}
	b2 := &fakeBackend{
		name: "b2",
		updateFn: func() (func() error, chan OperationResult) {
			ch := make(chan OperationResult, 1)
			return func() error {
				ch <- OperationResult{Success: false, Error: errors.New("something went wrong"), Logs: "b2 failed log\n"}
				return nil
			}, ch
		},
	}
	b3 := &fakeBackend{
		name: "b3",
		updateFn: func() (func() error, chan OperationResult) {
			ch := make(chan OperationResult, 1)
			return func() error {
				b3Ran = true
				ch <- OperationResult{Success: true, Logs: "b3 ok\n"}
				return nil
			}, ch
		},
	}

	exec, resChan := updateBackends([]Backend{b1, b2, b3}, nil)
	if err := exec(); err != nil {
		t.Fatalf("expected exec() to return nil, got: %v", err)
	}

	select {
	case res := <-resChan:
		if res.Success {
			t.Errorf("expected res.Success == false")
		}
		if res.Error == nil || !strings.Contains(res.Error.Error(), "b2") {
			t.Errorf("expected Error mentioning b2, got: %v", res.Error)
		}
		if !strings.Contains(res.Logs, "Updated: b1, b3") {
			t.Errorf("expected b1 and b3 under Updated, got:\n%s", res.Logs)
		}
		if !strings.Contains(res.Logs, "Failed: b2: something went wrong") {
			t.Errorf("expected b2 under Failed, got:\n%s", res.Logs)
		}
		if !b3Ran {
			t.Errorf("expected b3 to have run even after b2 failed")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for result")
	}
}

// (c) some skipped (sentinel + nil func) alongside successes -> Success, the skipped names are listed as skipped and not as failures
func TestUpdateBackends_Skipped(t *testing.T) {
	b1 := &fakeBackend{
		name: "b1",
		updateFn: func() (func() error, chan OperationResult) {
			ch := make(chan OperationResult, 1)
			return func() error {
				ch <- OperationResult{Success: true}
				return nil
			}, ch
		},
	}
	// nil func
	b2 := &fakeBackend{
		name: "b2",
		updateFn: func() (func() error, chan OperationResult) {
			return nil, nil
		},
	}
	// sentinel error
	b3 := &fakeBackend{
		name: "b3",
		updateFn: func() (func() error, chan OperationResult) {
			ch := make(chan OperationResult, 1)
			sentinelErr := fmt.Errorf("b3 %w", ErrUpdateNotImplemented)
			ch <- OperationResult{Error: sentinelErr}
			return func() error {
				return sentinelErr
			}, ch
		},
	}

	exec, resChan := updateBackends([]Backend{b1, b2, b3}, nil)
	if err := exec(); err != nil {
		t.Fatalf("expected exec() to return nil, got: %v", err)
	}

	select {
	case res := <-resChan:
		if !res.Success {
			t.Errorf("expected res.Success == true, got false")
		}
		if res.Error != nil {
			t.Errorf("expected res.Error == nil, got: %v", res.Error)
		}
		if !strings.Contains(res.Logs, "Updated: b1") {
			t.Errorf("expected Updated: b1 in logs, got:\n%s", res.Logs)
		}
		if !strings.Contains(res.Logs, "Skipped: b2, b3") {
			t.Errorf("expected Skipped: b2, b3 in logs, got:\n%s", res.Logs)
		}
		if strings.Contains(res.Logs, "Failed:") {
			t.Errorf("logs should not contain Failed:, got:\n%s", res.Logs)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for result")
	}
}

// (d) an f() error that sends nothing on its channel does not hang (run the test under -timeout)
func TestUpdateBackends_NoHangOnAuthError(t *testing.T) {
	b1 := &fakeBackend{
		name: "b1",
		updateFn: func() (func() error, chan OperationResult) {
			ch := make(chan OperationResult, 1)
			// Channel will never have anything sent to it
			return func() error {
				return errors.New("auth cancelled")
			}, ch
		},
	}

	exec, resChan := updateBackends([]Backend{b1}, nil)
	if err := exec(); err != nil {
		t.Fatalf("expected exec() to return nil, got: %v", err)
	}

	select {
	case res := <-resChan:
		if res.Success {
			t.Errorf("expected res.Success == false")
		}
		if res.Error == nil || !strings.Contains(res.Error.Error(), "auth cancelled") {
			t.Errorf("expected error mentioning auth cancelled, got: %v", res.Error)
		}
		if !strings.Contains(res.Logs, "Failed: b1: auth cancelled") {
			t.Errorf("expected Failed: b1: auth cancelled, got:\n%s", res.Logs)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("hung waiting on channel when f() returned error")
	}
}

// (e) zero backends -> exactly one result, Success
func TestUpdateBackends_ZeroBackends(t *testing.T) {
	exec, resChan := updateBackends([]Backend{}, nil)
	if err := exec(); err != nil {
		t.Fatalf("expected exec() to return nil, got: %v", err)
	}

	select {
	case res := <-resChan:
		if !res.Success {
			t.Errorf("expected res.Success == true")
		}
		if res.Error != nil {
			t.Errorf("expected res.Error == nil, got: %v", res.Error)
		}
		if !strings.Contains(strings.ToLower(res.Logs), "no available backends") {
			t.Errorf("expected 'no available backends' in logs, got:\n%s", res.Logs)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for result")
	}

	// Verify exactly one result
	select {
	case extra := <-resChan:
		t.Fatalf("unexpected extra result: %+v", extra)
	default:
	}
}

// (f) ordering: backend n+1's Update/f is not invoked before backend n's result has been delivered
func TestUpdateBackends_Ordering(t *testing.T) {
	var mu sync.Mutex
	var events []string
	record := func(ev string) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, ev)
	}

	b1 := &fakeBackend{
		name: "b1",
		updateFn: func() (func() error, chan OperationResult) {
			record("b1.Update")
			ch := make(chan OperationResult, 1)
			return func() error {
				record("b1.f_start")
				go func() {
					time.Sleep(30 * time.Millisecond)
					record("b1.deliver")
					ch <- OperationResult{Success: true, Logs: "b1 done\n"}
				}()
				return nil
			}, ch
		},
	}

	b2 := &fakeBackend{
		name: "b2",
		updateFn: func() (func() error, chan OperationResult) {
			record("b2.Update")
			ch := make(chan OperationResult, 1)
			return func() error {
				record("b2.f_start")
				go func() {
					time.Sleep(10 * time.Millisecond)
					record("b2.deliver")
					ch <- OperationResult{Success: true, Logs: "b2 done\n"}
				}()
				return nil
			}, ch
		},
	}

	exec, resChan := updateBackends([]Backend{b1, b2}, nil)
	if err := exec(); err != nil {
		t.Fatalf("expected exec() to return nil, got: %v", err)
	}

	select {
	case <-resChan:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for result")
	}

	mu.Lock()
	defer mu.Unlock()

	// Verify order: b1.deliver MUST precede b2.Update and b2.f_start
	deliverB1Idx := -1
	updateB2Idx := -1
	for i, ev := range events {
		if ev == "b1.deliver" {
			deliverB1Idx = i
		}
		if ev == "b2.Update" {
			updateB2Idx = i
		}
	}

	if deliverB1Idx == -1 {
		t.Fatalf("b1.deliver not recorded in events: %v", events)
	}
	if updateB2Idx == -1 {
		t.Fatalf("b2.Update not recorded in events: %v", events)
	}
	if deliverB1Idx > updateB2Idx {
		t.Fatalf("expected b1.deliver (%d) before b2.Update (%d) in events: %v", deliverB1Idx, updateB2Idx, events)
	}
}

func TestUpdateAll(t *testing.T) {
	exec, ch := UpdateAll()
	if exec == nil {
		t.Fatal("expected exec != nil")
	}
	if ch == nil {
		t.Fatal("expected ch != nil")
	}
}
