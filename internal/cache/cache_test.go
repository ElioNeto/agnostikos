package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// equalStringSlices compares two string slices for equality.
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestPackageCache_SetGet verifies that Set followed by Get returns the same
// results for matching backend, query, and version policy.
func TestPackageCache_SetGet(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	c := New(dir, 24*time.Hour, 1*time.Hour)

	// Set some data.
	c.Set("pacman", "firefox", []string{"extra/firefox 125.0.1"})

	tests := []struct {
		name     string
		backend  string
		query    string
		policy   string
		wantOK   bool
		wantRes  []string
	}{
		{
			name:    "stable policy finds cached entry",
			backend: "pacman",
			query:   "firefox",
			policy:  "stable",
			wantOK:  true,
			wantRes: []string{"extra/firefox 125.0.1"},
		},
		{
			name:    "latest policy finds cached entry",
			backend: "pacman",
			query:   "firefox",
			policy:  "latest",
			wantOK:  true,
			wantRes: []string{"extra/firefox 125.0.1"},
		},
		{
			name:    "default policy finds cached entry",
			backend: "pacman",
			query:   "firefox",
			policy:  "default",
			wantOK:  true,
			wantRes: []string{"extra/firefox 125.0.1"},
		},
		{
			name:    "empty policy finds cached entry",
			backend: "pacman",
			query:   "firefox",
			policy:  "",
			wantOK:  true,
			wantRes: []string{"extra/firefox 125.0.1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := c.Get(tt.backend, tt.query, tt.policy)
			if ok != tt.wantOK {
				t.Errorf("Get() ok = %v, want %v", ok, tt.wantOK)
			}
			if tt.wantOK && !equalStringSlices(got, tt.wantRes) {
				t.Errorf("Get() results = %v, want %v", got, tt.wantRes)
			}
		})
	}
}

// TestPackageCache_GetMiss verifies that Get returns false for a non-existent key.
func TestPackageCache_GetMiss(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	c := New(dir, 24*time.Hour, 1*time.Hour)

	got, ok := c.Get("pacman", "nonexistent", "stable")
	if ok {
		t.Fatal("expected cache miss, got hit")
	}
	if got != nil {
		t.Fatalf("expected nil results, got %v", got)
	}
}

// TestPackageCache_GetExpired verifies that an expired memory entry is not returned.
// Uses a 1-nanosecond TTL so the entry is expired immediately after Set.
func TestPackageCache_GetExpired(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	c := New(dir, 1*time.Nanosecond, 1*time.Nanosecond)

	c.Set("pacman", "firefox", []string{"extra/firefox 125.0.1"})

	// The entry should be expired immediately due to the nanosecond TTL.
	got, ok := c.Get("pacman", "firefox", "stable")
	if ok {
		t.Fatal("expected cache miss for expired entry, got hit")
	}
	if got != nil {
		t.Fatalf("expected nil results for expired entry, got %v", got)
	}
}

// TestPackageCache_Invalidate verifies that Invalidate clears all entries.
func TestPackageCache_Invalidate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	c := New(dir, 24*time.Hour, 1*time.Hour)

	// Insert entries for multiple backends.
	c.Set("pacman", "firefox", []string{"extra/firefox 125.0.1"})
	c.Set("nix", "firefox", []string{"legacyPackages.x86_64-linux.firefox (124.0)"})
	c.Set("flatpak", "firefox", []string{"org.mozilla.firefox"})

	// Verify they are cached.
	if _, ok := c.Get("pacman", "firefox", "stable"); !ok {
		t.Fatal("expected cache hit before invalidation")
	}

	c.Invalidate()

	// All backends should now miss.
	tests := []struct {
		name    string
		backend string
		query   string
	}{
		{name: "pacman miss after invalidate", backend: "pacman", query: "firefox"},
		{name: "nix miss after invalidate", backend: "nix", query: "firefox"},
		{name: "flatpak miss after invalidate", backend: "flatpak", query: "firefox"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := c.Get(tt.backend, tt.query, "stable"); ok {
				t.Errorf("expected cache miss after Invalidate() for %s/%s", tt.backend, tt.query)
			}
		})
	}
}

// TestPackageCache_InvalidateBackend verifies that InvalidateBackend clears
// only the specified backend's entries.
func TestPackageCache_InvalidateBackend(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	c := New(dir, 24*time.Hour, 1*time.Hour)

	c.Set("pacman", "firefox", []string{"extra/firefox 125.0.1"})
	c.Set("nix", "firefox", []string{"legacyPackages.x86_64-linux.firefox (124.0)"})

	c.InvalidateBackend("pacman")

	// Pacman should miss.
	if _, ok := c.Get("pacman", "firefox", "stable"); ok {
		t.Error("expected cache miss for pacman after InvalidateBackend")
	}

	// Nix should still be cached.
	if _, ok := c.Get("nix", "firefox", "stable"); !ok {
		t.Error("expected cache hit for nix after InvalidateBackend('pacman')")
	}
}

// TestPackageCache_ConcurrentAccess verifies thread safety under concurrent
// reads and writes. Run with `go test -race` to detect data races.
func TestPackageCache_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	c := New(dir, 24*time.Hour, 1*time.Hour)

	var wg sync.WaitGroup
	n := 50

	// Concurrent writers.
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			backend := "pacman"
			query := "pkg" + itoa(i)
			results := []string{"result" + itoa(i)}
			c.Set(backend, query, results)
		}(i)
	}

	// Concurrent readers.
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			backend := "pacman"
			query := "pkg" + itoa(i)
			c.Get(backend, query, "stable")
		}(i)
	}

	wg.Wait()

	// Verify all written entries are retrievable.
	for i := range n {
		backend := "pacman"
		query := "pkg" + itoa(i)
		got, ok := c.Get(backend, query, "stable")
		if !ok {
			t.Errorf("expected cache hit for %s/%s", backend, query)
			continue
		}
		if len(got) != 1 || got[0] != "result"+itoa(i) {
			t.Errorf("unexpected results for %s/%s: %v", backend, query, got)
		}
	}
}

// itoa is a small integer-to-string helper for tests.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	s := ""
	for i > 0 {
		s = string(rune('0'+i%10)) + s
		i /= 10
	}
	return s
}

// TestPackageCache_DiskPersistence verifies that entries survive cache
// instance recreation (i.e. they are persisted to disk and reloaded).
func TestPackageCache_DiskPersistence(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// First cache instance: write data.
	c1 := New(dir, 24*time.Hour, 1*time.Hour)
	c1.Set("pacman", "firefox", []string{"extra/firefox 125.0.1"})
	c1.Set("nix", "firefox", []string{"legacyPackages.x86_64-linux.firefox (124.0)"})
	c1.Close()

	// Second cache instance: same directory, should read from disk.
	c2 := New(dir, 24*time.Hour, 1*time.Hour)

	tests := []struct {
		name    string
		backend string
		query   string
		wantRes []string
	}{
		{
			name:    "pacman entry persists",
			backend: "pacman",
			query:   "firefox",
			wantRes: []string{"extra/firefox 125.0.1"},
		},
		{
			name:    "nix entry persists",
			backend: "nix",
			query:   "firefox",
			wantRes: []string{"legacyPackages.x86_64-linux.firefox (124.0)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := c2.Get(tt.backend, tt.query, "stable")
			if !ok {
				t.Fatalf("expected cache hit for %s/%s from disk", tt.backend, tt.query)
			}
			if !equalStringSlices(got, tt.wantRes) {
				t.Errorf("got %v, want %v", got, tt.wantRes)
			}
		})
	}
}

// TestPackageCache_DifferentTTL verifies that "stable" and "latest" policies
// have different TTLs. A short latestTTL causes latest entries to expire
// while stable entries (with longer TTL) remain valid.
func TestPackageCache_DifferentTTL(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Stable TTL = 1 hour, Latest TTL = 1 nanosecond (effectively expired).
	stableTTL := 1 * time.Hour
	latestTTL := 1 * time.Nanosecond

	c := New(dir, stableTTL, latestTTL)

	c.Set("pacman", "firefox", []string{"extra/firefox 125.0.1"})

	// Stable policy should hit (TTL is long).
	if _, ok := c.Get("pacman", "firefox", "stable"); !ok {
		t.Error("expected cache hit with stable policy (long TTL)")
	}

	// Latest policy should miss (TTL expired).
	if _, ok := c.Get("pacman", "firefox", "latest"); ok {
		t.Error("expected cache miss with latest policy (short TTL)")
	}
}

// TestPackageCache_EmptyMissingDirectory verifies that creating a cache with
// an empty or non-existent directory does not panic and falls back to
// memory-only mode.
func TestPackageCache_EmptyMissingDirectory(t *testing.T) {
	t.Parallel()

	// Use a deep non-existent path to ensure MkdirAll may fail.
	// On most systems this will succeed (MkdirAll creates parents),
	// but we test that it doesn't panic regardless.
	badDir := filepath.Join(os.TempDir(), "agnostikos-test-nonexistent-"+itoa(time.Now().Nanosecond()))
	c := New(badDir, 24*time.Hour, 1*time.Hour)

	// Even if disk is disabled, memory cache should work.
	c.Set("pacman", "firefox", []string{"extra/firefox 125.0.1"})
	got, ok := c.Get("pacman", "firefox", "stable")
	if !ok {
		t.Fatal("expected memory cache hit even with potentially disabled disk cache")
	}
	if !equalStringSlices(got, []string{"extra/firefox 125.0.1"}) {
		t.Fatalf("unexpected results: %v", got)
	}

	// Clean up.
	os.RemoveAll(badDir)
}

// TestPackageCache_Close verifies that Close returns no error.
func TestPackageCache_Close(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	c := New(dir, 24*time.Hour, 1*time.Hour)

	if err := c.Close(); err != nil {
		t.Fatalf("Close() returned error: %v", err)
	}
}

// TestPackageCache_SanitizeQuery verifies that query sanitization produces
// safe filesystem names.
func TestPackageCache_SanitizeQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{input: "firefox", want: "firefox"},
		{input: "firefox-123", want: "firefox_123"},
		{input: "neovim/stable", want: "neovim_stable"},
		{input: "a.b.c", want: "a_b_c"},
		{input: "foo bar baz", want: "foo_bar_baz"},
		{input: "", want: ""},
		{input: "...", want: "___"},
	}

	c := New(t.TempDir(), 24*time.Hour, 1*time.Hour)

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeQuery(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeQuery(%q) = %q, want %q", tt.input, got, tt.want)
			}
			// Verify the result is safe for filesystem use:
			// diskPath should not contain any path separators or special chars
			// (other than the expected directory structure).
			path := c.diskPath("test", tt.input)
			if filepath.Base(path) != got+".json" {
				t.Errorf("diskPath base = %q, want %q.json", filepath.Base(path), got)
			}
		})
	}
}

// TestPackageCache_DiskFileFormat verifies the JSON format of disk cache files.
func TestPackageCache_DiskFileFormat(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	c := New(dir, 24*time.Hour, 1*time.Hour)

	results := []string{"extra/firefox 125.0.1", "extra/firefox 126.0.0-beta1"}
	c.Set("pacman", "firefox", results)

	// Read the disk file directly.
	diskPath := c.diskPath("pacman", "firefox")
	data, err := os.ReadFile(diskPath)
	if err != nil {
		t.Fatalf("failed to read disk cache file: %v", err)
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("failed to unmarshal disk cache JSON: %v", err)
	}

	if !equalStringSlices(entry.Results, results) {
		t.Errorf("disk results = %v, want %v", entry.Results, results)
	}

	if entry.FetchedAt.IsZero() {
		t.Error("expected FetchedAt to be set, got zero time")
	}
}

// TestPackageCache_KeyFormat verifies the cache key format.
func TestPackageCache_KeyFormat(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	c := New(dir, 24*time.Hour, 1*time.Hour)

	c.Set("pacman", "firefox", []string{"extra/firefox 125.0.1"})

	// Verify key structure via Get.
	got, ok := c.Get("pacman", "firefox", "stable")
	if !ok || !equalStringSlices(got, []string{"extra/firefox 125.0.1"}) {
		t.Fatal("expected cache hit with correct key format")
	}

	// Same backend, different query should miss.
	if _, ok := c.Get("pacman", "chromium", "stable"); ok {
		t.Error("expected cache miss for different query")
	}

	// Different backend, same query should miss.
	if _, ok := c.Get("nix", "firefox", "stable"); ok {
		t.Error("expected cache miss for different backend")
	}
}

// TestPackageCache_DefaultTTL verifies that the default TTL is the minimum
// of stable and latest TTLs.
func TestPackageCache_DefaultTTL(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	c := New(dir, 10*time.Second, 5*time.Second)

	if c.defaultTTL != 5*time.Second {
		t.Errorf("defaultTTL = %v, want 5s (min of 10s and 5s)", c.defaultTTL)
	}

	c2 := New(dir, 3*time.Second, 7*time.Second)
	if c2.defaultTTL != 3*time.Second {
		t.Errorf("defaultTTL = %v, want 3s (min of 3s and 7s)", c2.defaultTTL)
	}
}

// TestPackageCache_GetAfterSetUpdatesDisk verifies that Set writes to disk
// immediately and a new cache instance reads the updated data.
func TestPackageCache_GetAfterSetUpdatesDisk(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	c1 := New(dir, 24*time.Hour, 1*time.Hour)
	c1.Set("pacman", "firefox", []string{"version1"})
	c1.Close()

	// New instance: should read version1 from disk.
	c2 := New(dir, 24*time.Hour, 1*time.Hour)
	got, ok := c2.Get("pacman", "firefox", "stable")
	if !ok || !equalStringSlices(got, []string{"version1"}) {
		t.Fatalf("expected [version1] from disk, got %v (ok=%v)", got, ok)
	}

	// Overwrite with new data and verify.
	c2.Set("pacman", "firefox", []string{"version2"})
	c2.Close()

	c3 := New(dir, 24*time.Hour, 1*time.Hour)
	got, ok = c3.Get("pacman", "firefox", "stable")
	if !ok || !equalStringSlices(got, []string{"version2"}) {
		t.Fatalf("expected [version2] from disk, got %v (ok=%v)", got, ok)
	}
}

// TestPackageCache_String verifies that String returns without panicking.
func TestPackageCache_String(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	c := New(dir, 24*time.Hour, 1*time.Hour)

	s := c.String()
	if s == "" {
		t.Error("expected non-empty String() output")
	}
}
