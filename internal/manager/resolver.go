package manager

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ElioNeto/agnostikos/internal/cache"
	"golang.org/x/sync/errgroup"
)

// ResolvePolicy defines how the Resolver selects a backend for a package.
type ResolvePolicy struct {
	// Priority is an ordered list of backend names to try.
	// If empty, all backends are tried in non-deterministic order.
	Priority []string `json:"priority,omitempty" yaml:"priority,omitempty"`

	// Version specifies the version policy: "latest", "stable", or a semver pin.
	// "latest" prefers the highest version.
	// "stable" prefers the latest non-prerelease version.
	// A specific semver (e.g. "1.2.3") filters to exact match.
	// Empty means "any version".
	Version string `json:"version,omitempty" yaml:"version,omitempty"`

	// Fallback enables automatic fallback to the next backend in priority
	// when the preferred backend does not have the package.
	Fallback bool `json:"fallback" yaml:"fallback_enabled"`
}

// ResolveResult holds the outcome of a successful resolution.
type ResolveResult struct {
	Backend  string `json:"backend"`
	Package  string `json:"package"`
	Version  string `json:"version,omitempty"`
	FullLine string `json:"full_line,omitempty"`
}

// SearchResult represents a single search hit from a backend with parsed version info.
type SearchResult struct {
	Backend  string
	Version  string
	FullLine string
}

// Resolver searches for packages across all backends and selects one
// based on the resolution policy.
type Resolver interface {
	// Resolve finds a package across backends and returns the best match.
	Resolve(ctx context.Context, pkg string, policy ResolvePolicy) (ResolveResult, error)

	// SearchAll searches for a query in all backends concurrently.
	// Returns a map of backend name -> search results.
	SearchAll(ctx context.Context, query string) (map[string][]string, error)
}

// resolver is the concrete implementation of Resolver.
type resolver struct {
	backends      map[string]PackageService
	searchTimeout time.Duration
	cache         *cache.PackageCache
}

// ResolverOption configures a Resolver.
type ResolverOption func(*resolver)

// WithSearchTimeout sets the per-backend search timeout.
// Default is 5 seconds.
func WithSearchTimeout(d time.Duration) ResolverOption {
	return func(r *resolver) {
		r.searchTimeout = d
	}
}

// withCache configures a Resolver to use the given PackageCache for caching
// search results. When set, SearchAll and Resolve check the cache before
// querying backends and store results after successful queries.
// It is unexported because the public API is the manager-level WithCache,
// which wires the cache into the resolver internally.
func withCache(c *cache.PackageCache) ResolverOption {
	return func(r *resolver) {
		r.cache = c
	}
}

// NewResolver creates a new Resolver with the given backends and optional settings.
func NewResolver(backends map[string]PackageService, opts ...ResolverOption) Resolver {
	r := &resolver{
		backends:      backends,
		searchTimeout: 5 * time.Second,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// SearchAll runs Search on all backends concurrently and returns aggregated results.
// Each backend search is bounded by the resolver's searchTimeout.
// Uses errgroup.Group for proper context cancellation propagation.
func (r *resolver) SearchAll(ctx context.Context, query string) (map[string][]string, error) {
	// Save original context for cancellation detection (errgroup cancels its
	// derived context when Wait() returns, so we cannot check that one)
	origCtx := ctx

	type result struct {
		name  string
		lines []string
		err   error
	}

	g, ctx := errgroup.WithContext(ctx)
	results := make(chan result, len(r.backends))

	for name, svc := range r.backends {
		name, svc := name, svc
		g.Go(func() error {
			// Check cache first (fast path).
			if r.cache != nil {
				if lines, ok := r.cache.Get(name, query, ""); ok {
					results <- result{name: name, lines: lines}
					return nil
				}
			}

			// Apply per-backend timeout
			searchCtx, searchCancel := context.WithTimeout(ctx, r.searchTimeout)
			defer searchCancel()

			// Check for context cancellation before starting
			select {
			case <-searchCtx.Done():
				results <- result{name: name, err: searchCtx.Err()}
				return nil // individual errors don't cancel the group
			default:
			}

			lines, err := svc.Search(query)
			if err != nil {
				results <- result{name: name, err: err}
				return nil // individual search errors don't fail the group
			}

			// Store results in cache for future lookups.
			if r.cache != nil {
				r.cache.Set(name, query, lines)
			}

			results <- result{name: name, lines: lines}
			return nil
		})
	}

	// Wait for all goroutines
	_ = g.Wait()
	close(results)

	// Check if the original context was cancelled (before errgroup wrapping)
	if origCtx.Err() != nil {
		return nil, origCtx.Err()
	}

	// Collect results, collecting errors when backends fail
	aggregated := make(map[string][]string)
	var errs []error
	for res := range results {
		if res.err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", res.name, res.err))
			continue
		}
		if len(res.lines) > 0 {
			aggregated[res.name] = res.lines
		}
	}

	// If no backends returned results and there were errors, report them
	if len(aggregated) == 0 && len(errs) > 0 {
		return aggregated, fmt.Errorf("search errors: %w", errors.Join(errs...))
	}

	return aggregated, nil
}

// Resolve finds the best backend for a package based on the policy.
// It searches all backends in parallel using errgroup and cancels
// remaining searches as soon as the preferred backend returns a match.
// Each backend search is bounded by the resolver's searchTimeout.
//
// Search results are streamed through a channel as backends complete,
// so the resolver can make a decision without waiting for slow backends.
func (r *resolver) Resolve(ctx context.Context, pkg string, policy ResolvePolicy) (ResolveResult, error) {
	// Determine the order of backends to try
	backendsToTry := r.resolvePriorityOrder(policy)

	// Save original context reference for cancellation propagation
	origCtx := ctx

	// Create a cancellable context for early termination
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// errgroup with the cancellable context for propagation
	g, ctx := errgroup.WithContext(ctx)

	type candidate struct {
		name  string
		lines []string
		err   error
	}
	ch := make(chan candidate, len(backendsToTry))

	// Fan-out: launch search for each backend concurrently
	for _, name := range backendsToTry {
		name := name
		backend, ok := r.backends[name]
		if !ok {
			continue
		}
		g.Go(func() error {
			// Check cache first (fast path).
			if r.cache != nil {
				if lines, ok := r.cache.Get(name, pkg, policy.Version); ok {
					select {
					case ch <- candidate{name: name, lines: lines}:
					case <-ctx.Done():
					}
					return nil
				}
			}

			// Per-backend timeout (pre-flight check)
			searchCtx, searchCancel := context.WithTimeout(ctx, r.searchTimeout)
			defer searchCancel()

			// If the context is already done (cancelled or timed out), report it.
			// We send directly to the buffered channel — the buffer is large
			// enough to hold all results, so this never blocks.
			if searchCtx.Err() != nil {
				ch <- candidate{name: name, err: searchCtx.Err()}
				return nil
			}

			lines, err := backend.Search(pkg)

			// Store results in cache on success.
			if err == nil && r.cache != nil {
				r.cache.Set(name, pkg, lines)
			}

			select {
			case ch <- candidate{name: name, lines: lines, err: err}:
			case <-ctx.Done():
				// Context cancelled (e.g. preferred backend already found)
			}
			return nil
		})
	}

	// Close channel when all goroutines complete
	go func() {
		_ = g.Wait()
		close(ch)
	}()

	// Streaming fan-in: process results as they arrive from backends.
	// Track which backends have responded and what their results were.
	allResults := make(map[string][]string)
	responded := make(map[string]bool)

	for c := range ch {
		responded[c.name] = true
		if c.err == nil && len(c.lines) > 0 {
			allResults[c.name] = c.lines
		}

		// Check if the original context was cancelled by the caller
		if origCtx.Err() != nil {
			return ResolveResult{}, fmt.Errorf("resolve failed: %w", origCtx.Err())
		}

		// Walk the priority list in order. For each backend in priority order:
		//   - If it hasn't responded yet, we cannot decide — wait for more results.
		//   - If it has results and passes version filter, this is our match.
		//   - If it has no results (or version mismatch) and fallback is on, skip it.
		//   - If it has no results and fallback is off, return error.
		for _, name := range backendsToTry {
			if !responded[name] {
				break // can't decide yet, wait for more results
			}
			lines, found := allResults[name]
			if !found || len(lines) == 0 {
				if policy.Fallback {
					continue // try next backend in priority
				}
				return ResolveResult{}, fmt.Errorf("package %q not found in backend %q", pkg, name)
			}
			// Filter by version policy
			filtered := filterByVersion(lines, policy.Version)
			if len(filtered) == 0 {
				if policy.Fallback {
					continue
				}
				return ResolveResult{}, fmt.Errorf("package %q found in backend %q but no version matches policy %q", pkg, name, policy.Version)
			}
			// Pick the best match from filtered results
			best := pickBestMatch(filtered, policy.Version)
			version := extractVersion(best)

			// Cancel remaining searches — we found a match
			cancel()

			return ResolveResult{
				Backend:  name,
				Package:  pkg,
				Version:  version,
				FullLine: best,
			}, nil
		}
	}

	// All backends have responded, none had a suitable match.
	// Build list of tried backends for error message.
	tried := make([]string, 0, len(backendsToTry))
	tried = append(tried, backendsToTry...)
	return ResolveResult{}, fmt.Errorf("package %q not found in any backend (tried: %s)", pkg, strings.Join(tried, ", "))
}

// resolvePriorityOrder returns the ordered list of backends to try.
func (r *resolver) resolvePriorityOrder(policy ResolvePolicy) []string {
	if len(policy.Priority) > 0 {
		// Return a defensive copy to prevent caller from mutating the policy slice
		return append([]string{}, policy.Priority...)
	}
	// Default priority: alphabetical for determinism
	names := make([]string, 0, len(r.backends))
	for name := range r.backends {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// filterByVersion filters search result lines based on the version policy.
func filterByVersion(lines []string, version string) []string {
	if version == "" {
		return lines
	}

	var filtered []string
	for _, line := range lines {
		v := extractVersion(line)
		if v == "" {
			// Lines without version info are included when no specific filter is needed
			if version == "latest" || version == "stable" {
				filtered = append(filtered, line)
			}
			continue
		}
		switch version {
		case "latest":
			filtered = append(filtered, line)
		case "stable":
			if !isPrerelease(v) {
				filtered = append(filtered, line)
			}
		default:
			// Treat as exact semver match (or prefix match)
			if strings.HasPrefix(v, version) {
				filtered = append(filtered, line)
			}
		}
	}
	return filtered
}

// pickBestMatch selects the best result from a list of filtered search lines.
func pickBestMatch(lines []string, version string) string {
	if len(lines) == 0 {
		return ""
	}
	if len(lines) == 1 {
		return lines[0]
	}

	// For version policies that prefer newest, sort by version desc
	if version == "latest" || version == "stable" {
		sort.Slice(lines, func(i, j int) bool {
			return compareVersions(extractVersion(lines[i]), extractVersion(lines[j])) > 0
		})
	}

	return lines[0]
}

// extractVersion attempts to parse a version string from a search result line.
// It looks for common patterns like "(1.2.3)" or "1.2.3" after the package name.
var versionRegex = regexp.MustCompile(`\(?(\d+\.\d+(?:\.\d+)*(?:[-.+][a-zA-Z0-9.]+)?)\)?`)

func extractVersion(line string) string {
	matches := versionRegex.FindStringSubmatch(line)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// isPrerelease checks if a version string is a prerelease (contains alpha, beta, rc, dev, etc.)
func isPrerelease(v string) bool {
	lower := strings.ToLower(v)
	for _, tag := range []string{"alpha", "beta", "rc", "dev", "pre", "test", "snapshot", "nightly"} {
		if strings.Contains(lower, tag) {
			return true
		}
	}
	return false
}

// numericPrefix extracts the leading numeric portion of a string.
// For example, "0-alpha" returns 0, "123beta" returns 123, "456" returns 456.
// Returns 0 if the string has no numeric prefix.
func numericPrefix(s string) int {
	for i, c := range s {
		if c < '0' || c > '9' {
			n, _ := strconv.Atoi(s[:i])
			return n
		}
	}
	n, _ := strconv.Atoi(s)
	return n
}

// compareVersions compares two semantic version strings.
// Returns >0 if v1 > v2, <0 if v1 < v2, 0 if equal.
func compareVersions(v1, v2 string) int {
	if v1 == "" && v2 == "" {
		return 0
	}
	if v1 == "" {
		return -1
	}
	if v2 == "" {
		return 1
	}

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		if i < len(parts1) {
			// Extract numeric prefix from version part (strip non-numeric suffix)
			n1 = numericPrefix(parts1[i])
		}
		if i < len(parts2) {
			n2 = numericPrefix(parts2[i])
		}
		if n1 != n2 {
			return n1 - n2
		}
	}
	return 0
}

// linesToSearchResults converts search result lines to SearchResult objects
// with parsed version information.
func linesToSearchResults(backend string, lines []string) []SearchResult {
	results := make([]SearchResult, len(lines))
	for i, line := range lines {
		results[i] = SearchResult{
			Backend:  backend,
			Version:  extractVersion(line),
			FullLine: line,
		}
	}
	return results
}

// applyVersionPolicy dispatches to the appropriate filter based on the version policy.
func applyVersionPolicy(results []SearchResult, version string) []SearchResult {
	switch version {
	case "":
		if len(results) > 0 {
			return results[:1]
		}
		return nil
	case "latest":
		return latestFilter(results)
	case "stable":
		return stableFilter(results)
	default:
		return pinnedFilter(results, version)
	}
}

// latestFilter returns the single result with the highest version.
// If multiple results have the same version, the first one is returned.
// Results without a parseable version sort last.
func latestFilter(results []SearchResult) []SearchResult {
	if len(results) == 0 {
		return nil
	}
	if len(results) == 1 {
		return results
	}
	sorted := make([]SearchResult, len(results))
	copy(sorted, results)
	sort.SliceStable(sorted, func(i, j int) bool {
		return compareVersions(sorted[i].Version, sorted[j].Version) > 0
	})
	return sorted[:1]
}

// stableFilter excludes pre-release versions (alpha, beta, rc, dev, etc.)
// and returns the highest remaining stable version.
// If all results are pre-release, returns nil.
func stableFilter(results []SearchResult) []SearchResult {
	var filtered []SearchResult
	for _, r := range results {
		if !isPrerelease(r.Version) {
			filtered = append(filtered, r)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return latestFilter(filtered)
}

// pinnedFilter returns results that exactly match the given pinVersion string.
// A result matches if its version equals pinVersion or has pinVersion as a
// proper prefix (e.g. pin "1.2" matches version "1.2.3" but not "1.20.0").
// Returns the first matching result or nil if none match.
func pinnedFilter(results []SearchResult, pinVersion string) []SearchResult {
	for _, r := range results {
		if r.Version == pinVersion || strings.HasPrefix(r.Version, pinVersion+".") {
			return []SearchResult{r}
		}
	}
	return nil
}
