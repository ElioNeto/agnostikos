// Package cache provides a disk-backed, in-memory cache for package metadata
// search results. It uses a two-level caching strategy:
//   - In-memory: fast path using sync.RWMutex-protected map
//   - Disk: JSON files stored under ~/.cache/agnostikos/<backend>/<query>.json
//
// TTL varies by version policy: 24h for "stable", 1h for "latest", and the
// minimum of both for any other policy.
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

// cacheEntry represents a single cached result set with its fetch timestamp.
type cacheEntry struct {
	Results   []string  `json:"results"`
	FetchedAt time.Time `json:"fetched_at"`
}

// PackageCache provides a thread-safe cache for package search results
// with both in-memory and disk-backed storage.
type PackageCache struct {
	mu          sync.RWMutex
	mem         map[string]cacheEntry
	cacheDir    string
	stableTTL   time.Duration
	latestTTL   time.Duration
	defaultTTL  time.Duration
	diskEnabled bool
}

// New creates a new PackageCache with the given cache directory and TTLs.
// The cache directory is created if it does not exist. If directory creation
// fails, disk caching is silently disabled and only in-memory caching is used.
//
// stableTTL and latestTTL are the TTLs for the "stable" and "latest" version
// policies respectively. defaultTTL is computed as the minimum of the two.
func New(dir string, stableTTL, latestTTL time.Duration) *PackageCache {
	defaultTTL := latestTTL
	if stableTTL < latestTTL {
		defaultTTL = stableTTL
	}

	c := &PackageCache{
		mem:        make(map[string]cacheEntry),
		cacheDir:   dir,
		stableTTL:  stableTTL,
		latestTTL:  latestTTL,
		defaultTTL: defaultTTL,
	}

	// Best-effort: if the directory cannot be created, disk cache is disabled.
	if err := os.MkdirAll(dir, 0755); err == nil {
		c.diskEnabled = true
	}

	return c
}

// cacheKey builds the key string for a given backend and query.
// Format: "<backend>:<query>"
func cacheKey(backend, query string) string {
	return backend + ":" + query
}

// nonAlphanumericRegex matches any character that is not a letter or digit.
var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9]`)

// sanitizeQuery replaces non-alphanumeric characters with underscores
// to produce a safe filesystem name.
func sanitizeQuery(query string) string {
	return nonAlphanumericRegex.ReplaceAllString(query, "_")
}

// diskPath returns the filesystem path for a given backend and query.
// Format: <cacheDir>/<backend>/<sanitized_query>.json
func (c *PackageCache) diskPath(backend, query string) string {
	return filepath.Join(c.cacheDir, backend, sanitizeQuery(query)+".json")
}

// ttlForPolicy returns the TTL appropriate for the given version policy.
func (c *PackageCache) ttlForPolicy(versionPolicy string) time.Duration {
	switch versionPolicy {
	case "stable":
		return c.stableTTL
	case "latest":
		return c.latestTTL
	default:
		return c.defaultTTL
	}
}

// Get retrieves cached results for a given backend, query, and version policy.
// It checks the in-memory cache first (fast path), then the disk cache.
// Returns the results and true if found and not expired; nil and false otherwise.
// Expired entries are removed from both memory and disk.
func (c *PackageCache) Get(backend, query, versionPolicy string) ([]string, bool) {
	key := cacheKey(backend, query)
	ttl := c.ttlForPolicy(versionPolicy)

	// Fast path: check in-memory cache first.
	c.mu.RLock()
	entry, found := c.mem[key]
	c.mu.RUnlock()

	if found {
		if time.Since(entry.FetchedAt) <= ttl {
			return entry.Results, true
		}
		// Entry expired: remove from memory.
		c.mu.Lock()
		delete(c.mem, key)
		c.mu.Unlock()
	}

	// Slow path: check disk cache.
	if !c.diskEnabled {
		return nil, false
	}

	path := c.diskPath(backend, query)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false // best-effort: file not found or unreadable
	}

	var diskEntry cacheEntry
	if err := json.Unmarshal(data, &diskEntry); err != nil {
		return nil, false // best-effort: corrupt file
	}

	if time.Since(diskEntry.FetchedAt) > ttl {
		// Stale disk entry: remove it.
		_ = os.Remove(path) // best-effort
		return nil, false
	}

	// Promote disk entry to memory for future fast access.
	c.mu.Lock()
	c.mem[key] = diskEntry
	c.mu.Unlock()

	return diskEntry.Results, true
}

// Set stores results in both memory and disk caches, tagged with the current
// timestamp. The in-memory store is always updated; the disk store is
// best-effort (failures are silently ignored).
func (c *PackageCache) Set(backend, query string, results []string) {
	key := cacheKey(backend, query)
	entry := cacheEntry{
		Results:   results,
		FetchedAt: time.Now(),
	}

	// Always store in memory.
	c.mu.Lock()
	c.mem[key] = entry
	c.mu.Unlock()

	// Best-effort disk write.
	if !c.diskEnabled {
		return
	}

	path := c.diskPath(backend, query)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return
	}
}

// Invalidate clears all cached entries from both memory and disk.
// After calling Invalidate, every Get will return a cache miss until new
// data is stored via Set.
func (c *PackageCache) Invalidate() {
	// Clear memory.
	c.mu.Lock()
	c.mem = make(map[string]cacheEntry)
	c.mu.Unlock()

	// Clear disk.
	if !c.diskEnabled {
		return
	}

	// Best-effort: remove entire cache directory and recreate.
	if err := os.RemoveAll(c.cacheDir); err != nil {
		return
	}
	_ = os.MkdirAll(c.cacheDir, 0755) // best-effort
}

// InvalidateBackend clears all cached entries for a specific backend from
// both memory and disk. Other backends' entries are preserved.
func (c *PackageCache) InvalidateBackend(name string) {
	prefix := name + ":"

	// Clear memory entries for this backend.
	c.mu.Lock()
	for key := range c.mem {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.mem, key)
		}
	}
	c.mu.Unlock()

	// Clear disk directory for this backend.
	if !c.diskEnabled {
		return
	}

	backendDir := filepath.Join(c.cacheDir, name)
	_ = os.RemoveAll(backendDir) // best-effort
}

// Close finalises the cache. Currently a no-op because all writes are
// synchronous. This method exists for future use (e.g. flush buffers, close
// file handles) and to satisfy cleanup patterns.
func (c *PackageCache) Close() error {
	return nil
}

// String returns a human-readable description of the cache.
func (c *PackageCache) String() string {
	c.mu.RLock()
	size := len(c.mem)
	c.mu.RUnlock()
	return fmt.Sprintf("PackageCache{dir: %s, entries: %d, disk: %v, stableTTL: %s, latestTTL: %s}",
		c.cacheDir, size, c.diskEnabled, c.stableTTL, c.latestTTL)
}
