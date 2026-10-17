//go:build pacman || all_backends

package backend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestCheckAURVersions(t *testing.T) {
	origCache := cache
	origURL := aurRPCURL
	origClient := aurHTTPClient

	t.Cleanup(func() {
		cache = origCache
		aurRPCURL = origURL
		aurHTTPClient = origClient
	})

	resetEnv := func() {
		cache = &aurCache{
			data:     make(map[string]cacheEntry),
			ttl:      1 * time.Minute,
			filePath: filepath.Join(t.TempDir(), "aur_cache.json"),
		}
		aurHTTPClient = &http.Client{
			Timeout: 10 * time.Second,
		}
	}

	t.Run("success_200_and_cached", func(t *testing.T) {
		resetEnv()

		var callCount atomic.Int32
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(aurResponse{
				Results: []aurPackage{
					{Name: "pkg-a", Version: "1.2.3"},
					{Name: "pkg-b", Version: "2.0.0"},
				},
			})
		}))
		defer ts.Close()

		aurRPCURL = ts.URL

		// First call - queries server
		res1, err := CheckAURVersions([]string{"pkg-a", "pkg-b"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res1) != 2 || res1["pkg-a"] != "1.2.3" || res1["pkg-b"] != "2.0.0" {
			t.Fatalf("unexpected results: %+v", res1)
		}
		if count := callCount.Load(); count != 1 {
			t.Fatalf("expected 1 server call, got %d", count)
		}

		// Verify cached in cache
		if cached, ok := cache.Get(CACHE_KEY); !ok {
			t.Fatal("expected cache to have CACHE_KEY")
		} else {
			cachedMap := cached.(map[string]string)
			if cachedMap["pkg-a"] != "1.2.3" || cachedMap["pkg-b"] != "2.0.0" {
				t.Fatalf("unexpected cached map: %+v", cachedMap)
			}
		}

		// Second call - should hit cache and not query server
		res2, err := CheckAURVersions([]string{"pkg-a"})
		if err != nil {
			t.Fatalf("unexpected error on second call: %v", err)
		}
		if res2["pkg-a"] != "1.2.3" {
			t.Fatalf("unexpected result from cache: %+v", res2)
		}
		if count := callCount.Load(); count != 1 {
			t.Fatalf("expected call count to remain 1, got %d", count)
		}
	})

	t.Run("non_200_returns_error_and_caches_nothing", func(t *testing.T) {
		resetEnv()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
		}))
		defer ts.Close()

		aurRPCURL = ts.URL

		res, err := CheckAURVersions([]string{"pkg-a"})
		if err == nil {
			t.Fatal("expected error for non-200 response, got nil")
		}
		if !strings.Contains(err.Error(), strconv.Itoa(http.StatusBadGateway)) {
			t.Fatalf("expected error to contain status code %d, got %v", http.StatusBadGateway, err)
		}
		if res != nil {
			t.Fatalf("expected nil result on error, got %+v", res)
		}

		// Verify nothing cached
		if _, ok := cache.Get(CACHE_KEY); ok {
			t.Fatal("expected cache to be empty after non-200 response")
		}
	})

	t.Run("slow_handler_trips_timeout", func(t *testing.T) {
		resetEnv()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-time.After(500 * time.Millisecond):
				w.WriteHeader(http.StatusOK)
			case <-r.Context().Done():
				return
			}
		}))
		defer ts.Close()

		aurRPCURL = ts.URL
		aurHTTPClient = &http.Client{
			Timeout: 20 * time.Millisecond,
		}

		res, err := CheckAURVersions([]string{"pkg-a"})
		if err == nil {
			t.Fatal("expected error due to timeout, got nil")
		}
		if res != nil {
			t.Fatalf("expected nil result on timeout error, got %+v", res)
		}

		// Verify nothing cached
		if _, ok := cache.Get(CACHE_KEY); ok {
			t.Fatal("expected cache to be empty after timeout error")
		}
	})
}
