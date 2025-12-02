//go:build pacman || all_backends

package backend

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type cacheEntry struct {
	Data       map[string]string `json:"data"`
	LastUpdate time.Time         `json:"last_update"`
}

type aurCache struct {
	mu       sync.RWMutex
	data     map[string]cacheEntry
	ttl      time.Duration
	filePath string
}

var cache = newCache()

func newCache() *aurCache {
	cacheDir := os.TempDir()
	if userCache, err := os.UserCacheDir(); err == nil {
		cacheDir = filepath.Join(userCache, "pkgmate")
		err := os.MkdirAll(cacheDir, 0750)
		if err != nil {
			slog.Warn("could not create cache dir", "err", err)
		}
	}

	c := &aurCache{
		data:     make(map[string]cacheEntry),
		ttl:      1 * time.Minute,
		filePath: filepath.Join(cacheDir, "aur_cache.json"),
	}
	c.load()
	return c
}

func (c *aurCache) load() {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(c.filePath)
	if err != nil {
		return // File doesn't exist or can't be read
	}

	err = json.Unmarshal(data, &c.data)
	if err != nil {
		slog.Warn("could not parse json cache")
	}
}

func (c *aurCache) save() {
	data, err := json.Marshal(c.data)
	if err != nil {
		return
	}
	err = os.WriteFile(c.filePath, data, 0600)
	if err != nil {
		slog.Warn("could not write to cache file", "filePath", c.filePath)
	}
}

func (c *aurCache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.data[key]
	if !ok || time.Since(entry.LastUpdate) > c.ttl {
		return nil, false
	}
	return entry.Data, true
}

func (c *aurCache) Set(key string, data map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = cacheEntry{
		Data:       data,
		LastUpdate: time.Now(),
	}
	c.save()
}

func (c *aurCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]cacheEntry)
	err := os.Remove(c.filePath)
	if err != nil {
		slog.Warn("could not remove cache file", "err", err)
	}
}
