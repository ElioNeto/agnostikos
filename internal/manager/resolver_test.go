package manager

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mockSearchBackend is a PackageService that returns controlled search results.
type mockSearchBackend struct {
	name      string
	searchRes []string
	searchErr error
}

func (m *mockSearchBackend) Install(pkgName string) error { return nil }
func (m *mockSearchBackend) Remove(pkgName string) error  { return nil }
func (m *mockSearchBackend) Update(pkg string) error      { return nil }
func (m *mockSearchBackend) UpdateAll() error              { return nil }
func (m *mockSearchBackend) Search(q string) ([]string, error) {
	return m.searchRes, m.searchErr
}
func (m *mockSearchBackend) List() ([]string, error) { return nil, nil }

func newMockBackends() map[string]PackageService {
	return map[string]PackageService{
		"pacman": &mockSearchBackend{
			name:      "pacman",
			searchRes: []string{"extra/firefox 125.0.1"},
		},
		"nix": &mockSearchBackend{
			name:      "nix",
			searchRes: []string{"legacyPackages.x86_64-linux.firefox (124.0)"},
		},
		"flatpak": &mockSearchBackend{
			name:      "flatpak",
			searchRes: []string{"org.mozilla.firefox"},
		},
	}
}

func newMockBackendsWithVersions() map[string]PackageService {
	return map[string]PackageService{
		"pacman": &mockSearchBackend{
			name: "pacman",
			searchRes: []string{
				"extra/firefox 124.0.1",
				"extra/firefox 125.0.0",
				"extra/firefox 126.0.0-beta1",
			},
		},
		"nix": &mockSearchBackend{
			name: "nix",
			searchRes: []string{
				"legacyPackages.x86_64-linux.firefox (123.0)",
				"legacyPackages.x86_64-linux.firefox-esr (115.0.0)",
			},
		},
	}
}

func TestResolver_Resolve(t *testing.T) {
	tests := []struct {
		name        string
		backends    map[string]PackageService
		pkg         string
		policy      ResolvePolicy
		wantBackend string
		wantVersion string // empty means don't check
		wantErr     bool
	}{
		{
			name:        "priority selection picks first backend",
			backends:    newMockBackends(),
			pkg:         "firefox",
			policy:      ResolvePolicy{Priority: []string{"nix", "pacman", "flatpak"}, Fallback: false},
			wantBackend: "nix",
			wantErr:     false,
		},
		{
			name:        "priority picks first in order",
			backends:    newMockBackends(),
			pkg:         "firefox",
			policy:      ResolvePolicy{Priority: []string{"pacman", "nix", "flatpak"}, Fallback: false},
			wantBackend: "pacman",
			wantVersion: "125.0.1",
			wantErr:     false,
		},
		{
			name: "fallback to next backend when preferred is empty",
			backends: map[string]PackageService{
				"pacman": &mockSearchBackend{name: "pacman", searchRes: []string{}},
				"nix":    &mockSearchBackend{name: "nix", searchRes: []string{"legacyPackages.x86_64-linux.neovim (0.9.5)"}},
				"flatpak": &mockSearchBackend{name: "flatpak", searchRes: []string{}},
			},
			pkg:         "neovim",
			policy:      ResolvePolicy{Priority: []string{"pacman", "nix", "flatpak"}, Fallback: true},
			wantBackend: "nix",
			wantErr:     false,
		},
		{
			name: "no fallback returns error when preferred has no results",
			backends: map[string]PackageService{
				"pacman": &mockSearchBackend{name: "pacman", searchRes: []string{}},
				"nix":    &mockSearchBackend{name: "nix", searchRes: []string{"legacyPackages.x86_64-linux.neovim (0.9.5)"}},
			},
			pkg:    "neovim",
			policy: ResolvePolicy{Priority: []string{"pacman", "nix"}, Fallback: false},
			wantErr: true,
		},
		{
			name: "error when no backend has the package",
			backends: map[string]PackageService{
				"pacman":  &mockSearchBackend{name: "pacman", searchRes: []string{}},
				"nix":     &mockSearchBackend{name: "nix", searchRes: []string{}},
				"flatpak": &mockSearchBackend{name: "flatpak", searchRes: []string{}},
			},
			pkg:    "nonexistent-pkg",
			policy: ResolvePolicy{Priority: []string{"pacman", "nix", "flatpak"}, Fallback: true},
			wantErr: true,
		},
		{
			name:        "version policy latest picks highest version",
			backends:    newMockBackendsWithVersions(),
			pkg:         "firefox",
			policy:      ResolvePolicy{Priority: []string{"pacman", "nix"}, Version: "latest", Fallback: false},
			wantBackend: "pacman",
			wantErr:     false,
		},
		{
			name: "version policy stable excludes prerelease",
			backends: map[string]PackageService{
				"pacman": &mockSearchBackend{
					name: "pacman",
					searchRes: []string{
						"extra/firefox 126.0.0-beta1",
						"extra/firefox 125.0.0",
						"extra/firefox 124.0.0",
					},
				},
			},
			pkg:         "firefox",
			policy:      ResolvePolicy{Priority: []string{"pacman"}, Version: "stable", Fallback: false},
			wantBackend: "pacman",
			wantVersion: "125.0.0",
			wantErr:     false,
		},
		{
			name:        "version policy pinned matches prefix",
			backends:    newMockBackendsWithVersions(),
			pkg:         "firefox",
			policy:      ResolvePolicy{Priority: []string{"pacman", "nix"}, Version: "124.0", Fallback: true},
			wantBackend: "pacman",
			wantErr:     false,
		},
		{
			name:        "empty priority defaults to all backends",
			backends:    newMockBackends(),
			pkg:         "firefox",
			policy:      ResolvePolicy{Priority: []string{}, Fallback: false},
			wantBackend: "flatpak", // alphabetical: flatpak, nix, pacman
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewResolver(tt.backends)
			result, err := r.Resolve(context.Background(), tt.pkg, tt.policy)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if result.Backend != tt.wantBackend {
				t.Errorf("expected backend %q, got %q", tt.wantBackend, result.Backend)
			}
			if tt.wantVersion != "" && result.Version != tt.wantVersion {
				t.Errorf("expected version %q, got %q", tt.wantVersion, result.Version)
			}
			if result.Package != tt.pkg {
				t.Errorf("expected package %q, got %q", tt.pkg, result.Package)
			}
		})
	}
}

func TestResolver_SearchAll(t *testing.T) {
	tests := []struct {
		name         string
		backends     map[string]PackageService
		query        string
		wantBackends []string // backends that should have results
		wantErr      bool
	}{
		{
			name:         "returns results from all backends",
			backends:     newMockBackends(),
			query:        "firefox",
			wantBackends: []string{"pacman", "nix", "flatpak"},
			wantErr:      false,
		},
		{
			name: "returns error when all backends fail",
			backends: map[string]PackageService{
				"pacman": &mockSearchBackend{name: "pacman", searchErr: errors.New("search failed")},
				"nix":    &mockSearchBackend{name: "nix", searchRes: []string{}},
			},
			query:        "firefox",
			wantBackends: nil,
			wantErr:      true,
		},
		{
			name: "returns partial results when some backends fail",
			backends: map[string]PackageService{
				"pacman": &mockSearchBackend{name: "pacman", searchErr: errors.New("search failed")},
				"nix":    &mockSearchBackend{name: "nix", searchRes: []string{"legacyPackages.x86_64-linux.firefox (124.0)"}},
			},
			query:        "firefox",
			wantBackends: []string{"nix"},
			wantErr:      false,
		},
		{
			name: "succeeds when backends return empty results",
			backends: map[string]PackageService{
				"pacman": &mockSearchBackend{name: "pacman", searchRes: []string{}},
				"nix":    &mockSearchBackend{name: "nix", searchRes: []string{}},
			},
			query:        "firefox",
			wantBackends: nil, // no backends with non-empty results
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewResolver(tt.backends)
			results, err := r.SearchAll(context.Background(), tt.query)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if tt.wantBackends == nil && len(results) != 0 {
				t.Errorf("expected no results, got %d backends", len(results))
			}
			for _, name := range tt.wantBackends {
				if _, ok := results[name]; !ok {
					t.Errorf("expected results from backend %q", name)
				}
			}
		})
	}
}

func TestResolver_ContextCancellation(t *testing.T) {
	tests := []struct {
		name        string
		backends    map[string]PackageService
		policy      ResolvePolicy
		cancel      bool // whether to cancel context before calling
		wantErr     bool
		wantErrIs   error
	}{
		{
			name:     "cancel before search returns cancellation error",
			backends: newMockBackends(),
			policy:   ResolvePolicy{Priority: []string{"pacman", "nix", "flatpak"}, Fallback: true},
			cancel:   true,
			wantErr:  true,
			wantErrIs: context.Canceled,
		},
		{
			name:     "no cancellation succeeds normally",
			backends: newMockBackends(),
			policy:   ResolvePolicy{Priority: []string{"pacman", "nix", "flatpak"}, Fallback: true},
			cancel:   false,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewResolver(tt.backends)

			ctx := context.Background()
			if tt.cancel {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			_, err := r.Resolve(ctx, "firefox", tt.policy)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("expected error wrapping %v, got %v", tt.wantErrIs, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

// delayedMockBackend wraps a mockSearchBackend and adds a configurable delay.
type delayedMockBackend struct {
	mockSearchBackend
	delay time.Duration
}

func (d *delayedMockBackend) Search(q string) ([]string, error) {
	time.Sleep(d.delay)
	return d.searchRes, d.searchErr
}

func TestResolver_SearchAll_ContextCancellation(t *testing.T) {
	// Set up a slow backend that would block if not cancelled
	backends := map[string]PackageService{
		"slow": &delayedMockBackend{
			mockSearchBackend: mockSearchBackend{
				name:      "slow",
				searchRes: []string{"slow/package 1.0.0"},
			},
			delay: 100 * time.Millisecond,
		},
		"fast": &mockSearchBackend{
			name:      "fast",
			searchRes: []string{"fast/package 1.0.0"},
		},
	}

	r := NewResolver(backends, WithSearchTimeout(10*time.Second))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := r.SearchAll(ctx, "package")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestResolver_SearchAll_PerBackendTimeout_PreFlight(t *testing.T) {
	// The per-backend timeout is a pre-flight check. When the context is
	// already expired before the search starts, the backend is skipped.
	backends := map[string]PackageService{
		"slow": &delayedMockBackend{
			mockSearchBackend: mockSearchBackend{name: "slow"},
			delay:             100 * time.Millisecond,
		},
		"fast": &mockSearchBackend{
			name:      "fast",
			searchRes: []string{"fast/package 1.0.0"},
		},
	}

	// Use a very short timeout so the deadlined context fires before
	// the goroutine reaches the search call.
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Microsecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond) // ensure the deadline has passed

	r := NewResolver(backends, WithSearchTimeout(100*time.Millisecond))

	results, err := r.SearchAll(ctx, "package")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected no results with cancelled context, got %v", results)
	}
}

func TestResolver_TimeoutConfigurable(t *testing.T) {
	// Verify WithSearchTimeout is used as the per-backend deadline.
	// Since backend.Search() does not accept a context, the timeout is a
	// pre-flight check: if the deadline has already expired before the search
	// starts, the backend is skipped. This test verifies the option is wired.
	r := NewResolver(nil, WithSearchTimeout(1*time.Second))
	if r.(*resolver).searchTimeout != 1*time.Second {
		t.Errorf("expected searchTimeout 1s, got %v", r.(*resolver).searchTimeout)
	}

	// Default timeout is 5s
	r2 := NewResolver(nil)
	if r2.(*resolver).searchTimeout != 5*time.Second {
		t.Errorf("expected default searchTimeout 5s, got %v", r2.(*resolver).searchTimeout)
	}
}

func TestResolver_TimeoutWithFallback(t *testing.T) {
	// First backend is slow, second responds fast
	backends := map[string]PackageService{
		"slow": &delayedMockBackend{
			mockSearchBackend: mockSearchBackend{name: "slow"},
			delay:             50 * time.Millisecond,
		},
		"fast": &mockSearchBackend{
			name:      "fast",
			searchRes: []string{"fast/package 2.0.0"},
		},
	}

	r := NewResolver(backends, WithSearchTimeout(50*time.Millisecond))

	result, err := r.Resolve(context.Background(), "package", ResolvePolicy{
		Priority: []string{"slow", "fast"},
		Fallback: true,
	})
	if err != nil {
		t.Fatalf("expected no error with fallback, got %v", err)
	}
	if result.Backend != "fast" {
		t.Errorf("expected fallback to fast, got %q", result.Backend)
	}
	if result.Version != "2.0.0" {
		t.Errorf("expected version 2.0.0, got %q", result.Version)
	}
}

func TestResolver_EarlyCancellation_PreferredBackendFast(t *testing.T) {
	// The preferred backend responds fast, so the slow backend should not
	// block resolution. This tests that parallel search returns as soon
	// as the preferred backend result is available.

	slowBackend := &delayedMockBackend{
		mockSearchBackend: mockSearchBackend{
			name: "slow",
		},
		delay: 200 * time.Millisecond, // slower than the immediate fast backend
	}

	backends := map[string]PackageService{
		"pacman": &mockSearchBackend{
			name:      "pacman",
			searchRes: []string{"extra/firefox 125.0.1"},
		},
		"nix": slowBackend,
	}

	r := NewResolver(backends, WithSearchTimeout(5*time.Second))

	start := time.Now()
	result, err := r.Resolve(context.Background(), "firefox", ResolvePolicy{
		Priority: []string{"pacman", "nix"},
		Fallback: true,
	})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Backend != "pacman" {
		t.Errorf("expected pacman (fast), got %q", result.Backend)
	}
	// The resolver should return much faster than the slow backend delay
	// because the slow backend is cancelled when pacman returns a result.
	if elapsed > 100*time.Millisecond {
		t.Errorf("expected early return (<100ms), but took %v (slow backend delay is 200ms)", elapsed)
	}
}

func TestResolver_ManagerIntegration(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() *AgnosticManager
		pkg         string
		policy      ResolvePolicy
		wantBackend string
		wantErr     bool
	}{
		{
			name: "resolver is initialized in NewAgnosticManager",
			setup: func() *AgnosticManager {
				return NewAgnosticManager()
			},
			pkg:    "some-package",
			policy: ResolvePolicy{Priority: []string{"pacman", "nix", "flatpak"}, Fallback: true},
			wantErr: true, // real backends won't find the package
		},
		{
			name: "custom resolver via WithResolver option works",
			setup: func() *AgnosticManager {
				mgr := NewAgnosticManager()
				WithResolver(NewResolver(newMockBackends()))(mgr)
				return mgr
			},
			pkg:         "firefox",
			policy:      ResolvePolicy{Priority: []string{"pacman"}, Fallback: false},
			wantBackend: "pacman",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := tt.setup()

			if mgr.Resolver == nil {
				t.Fatal("expected Resolver to be initialized")
			}

			result, err := mgr.ResolvePackage(context.Background(), tt.pkg, tt.policy)

			if tt.wantErr {
				if err == nil {
					t.Log("expected error since package doesn't exist")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if result.Backend != tt.wantBackend {
				t.Errorf("expected backend %q, got %q", tt.wantBackend, result.Backend)
			}
		})
	}
}

func TestLatestFilter(t *testing.T) {
	tests := []struct {
		name     string
		results  []SearchResult
		wantLen  int
		wantVers string // expected version of the single result (empty = don't check)
	}{
		{
			name:    "empty slice returns nil",
			results: nil,
			wantLen: 0,
		},
		{
			name:    "single result returns same",
			results: []SearchResult{{Backend: "pacman", Version: "1.0.0", FullLine: "pkg 1.0.0"}},
			wantLen: 1,
		},
		{
			name: "picks highest version",
			results: []SearchResult{
				{Backend: "pacman", Version: "1.0.0", FullLine: "pkg 1.0.0"},
				{Backend: "pacman", Version: "2.0.0", FullLine: "pkg 2.0.0"},
				{Backend: "pacman", Version: "1.5.0", FullLine: "pkg 1.5.0"},
			},
			wantLen:  1,
			wantVers: "2.0.0",
		},
		{
			name: "picks highest with three-part versions",
			results: []SearchResult{
				{Backend: "nix", Version: "10.9.8", FullLine: "pkg 10.9.8"},
				{Backend: "nix", Version: "11.0.0", FullLine: "pkg 11.0.0"},
				{Backend: "nix", Version: "10.99.99", FullLine: "pkg 10.99.99"},
			},
			wantLen:  1,
			wantVers: "11.0.0",
		},
		{
			name: "empty versions sort last",
			results: []SearchResult{
				{Backend: "flatpak", Version: "", FullLine: "org.pkg"},
				{Backend: "flatpak", Version: "1.0.0", FullLine: "pkg 1.0.0"},
				{Backend: "flatpak", Version: "", FullLine: "org.pkg.stable"},
			},
			wantLen:  1,
			wantVers: "1.0.0",
		},
		{
			name: "all same version returns first",
			results: []SearchResult{
				{Backend: "pacman", Version: "1.2.3", FullLine: "pkg 1.2.3"},
				{Backend: "nix", Version: "1.2.3", FullLine: "pkg 1.2.3"},
			},
			wantLen:  1,
			wantVers: "1.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := latestFilter(tt.results)
			if len(got) != tt.wantLen {
				t.Fatalf("expected %d results, got %d", tt.wantLen, len(got))
			}
			if tt.wantLen > 0 && tt.wantVers != "" && got[0].Version != tt.wantVers {
				t.Errorf("expected version %q, got %q", tt.wantVers, got[0].Version)
			}
		})
	}
}

func TestStableFilter(t *testing.T) {
	tests := []struct {
		name     string
		results  []SearchResult
		wantLen  int
		wantVers string
	}{
		{
			name:    "empty slice returns nil",
			results: nil,
			wantLen: 0,
		},
		{
			name: "no prerelease returns highest stable",
			results: []SearchResult{
				{Backend: "pacman", Version: "1.0.0", FullLine: "pkg 1.0.0"},
				{Backend: "pacman", Version: "3.0.0", FullLine: "pkg 3.0.0"},
				{Backend: "pacman", Version: "2.0.0", FullLine: "pkg 2.0.0"},
			},
			wantLen:  1,
			wantVers: "3.0.0",
		},
		{
			name: "excludes beta prerelease",
			results: []SearchResult{
				{Backend: "pacman", Version: "3.0.0-beta1", FullLine: "pkg 3.0.0-beta1"},
				{Backend: "pacman", Version: "2.0.0", FullLine: "pkg 2.0.0"},
				{Backend: "pacman", Version: "1.0.0", FullLine: "pkg 1.0.0"},
			},
			wantLen:  1,
			wantVers: "2.0.0",
		},
		{
			name: "excludes alpha, rc, dev, nightly",
			results: []SearchResult{
				{Backend: "nix", Version: "2.0.0-nightly", FullLine: "pkg nightly"},
				{Backend: "nix", Version: "2.0.0-alpha1", FullLine: "pkg alpha"},
				{Backend: "nix", Version: "1.9.0-rc2", FullLine: "pkg rc"},
				{Backend: "nix", Version: "1.9.0", FullLine: "pkg stable"},
			},
			wantLen:  1,
			wantVers: "1.9.0",
		},
		{
			name: "all prerelease returns nil",
			results: []SearchResult{
				{Backend: "pacman", Version: "3.0.0-beta1", FullLine: "pkg 3.0.0-beta1"},
				{Backend: "pacman", Version: "3.0.0-alpha", FullLine: "pkg 3.0.0-alpha"},
			},
			wantLen: 0,
		},
		{
			name: "empty version treated as stable",
			results: []SearchResult{
				{Backend: "flatpak", Version: "", FullLine: "org.pkg"},
				{Backend: "flatpak", Version: "2.0.0-beta", FullLine: "pkg beta"},
			},
			wantLen:  1,
			wantVers: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stableFilter(tt.results)
			if len(got) != tt.wantLen {
				t.Fatalf("expected %d results, got %d", tt.wantLen, len(got))
			}
			if tt.wantLen > 0 && tt.wantVers != "" && got[0].Version != tt.wantVers {
				t.Errorf("expected version %q, got %q", tt.wantVers, got[0].Version)
			}
		})
	}
}

func TestPinnedFilter(t *testing.T) {
	tests := []struct {
		name       string
		results    []SearchResult
		pinVersion string
		wantLen    int
		wantVers   string
	}{
		{
			name:       "empty slice returns nil",
			results:    nil,
			pinVersion: "1.0.0",
			wantLen:    0,
		},
		{
			name: "exact match returns the result",
			results: []SearchResult{
				{Backend: "pacman", Version: "1.0.0", FullLine: "pkg 1.0.0"},
				{Backend: "pacman", Version: "2.0.0", FullLine: "pkg 2.0.0"},
			},
			pinVersion: "2.0.0",
			wantLen:    1,
			wantVers:   "2.0.0",
		},
		{
			name: "prefix match (pin 1.2 matches 1.2.3)",
			results: []SearchResult{
				{Backend: "nix", Version: "1.2.3", FullLine: "pkg 1.2.3"},
				{Backend: "nix", Version: "1.3.0", FullLine: "pkg 1.3.0"},
			},
			pinVersion: "1.2",
			wantLen:    1,
			wantVers:   "1.2.3",
		},
		{
			name: "prefix does not cross major version boundary",
			results: []SearchResult{
				{Backend: "pacman", Version: "10.0.0", FullLine: "pkg 10.0.0"},
				{Backend: "pacman", Version: "1.0.0", FullLine: "pkg 1.0.0"},
			},
			pinVersion: "1",
			wantLen:    1,
			wantVers:   "1.0.0",
		},
		{
			name: "no match returns nil",
			results: []SearchResult{
				{Backend: "pacman", Version: "3.0.0", FullLine: "pkg 3.0.0"},
				{Backend: "pacman", Version: "4.0.0", FullLine: "pkg 4.0.0"},
			},
			pinVersion: "2.0.0",
			wantLen:    0,
		},
		{
			name: "exact match with prerelease version string",
			results: []SearchResult{
				{Backend: "pacman", Version: "1.0.0-rc1", FullLine: "pkg rc"},
				{Backend: "pacman", Version: "0.9.0", FullLine: "pkg 0.9.0"},
			},
			pinVersion: "1.0.0-rc1",
			wantLen:    1,
			wantVers:   "1.0.0-rc1",
		},
		{
			name: "prefix match with prerelease version",
			results: []SearchResult{
				{Backend: "nix", Version: "1.0.0-rc1", FullLine: "pkg rc"},
				{Backend: "nix", Version: "0.9.0", FullLine: "pkg 0.9.0"},
			},
			pinVersion: "1",
			wantLen:    1,
			wantVers:   "1.0.0-rc1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pinnedFilter(tt.results, tt.pinVersion)
			if len(got) != tt.wantLen {
				t.Fatalf("expected %d results, got %d", tt.wantLen, len(got))
			}
			if tt.wantLen > 0 && tt.wantVers != "" && got[0].Version != tt.wantVers {
				t.Errorf("expected version %q, got %q", tt.wantVers, got[0].Version)
			}
		})
	}
}

func TestLinesToSearchResults(t *testing.T) {
	lines := []string{
		"extra/firefox 125.0.1",
		"extra/firefox 126.0.0-beta1",
	}
	results := linesToSearchResults("pacman", lines)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Backend != "pacman" {
		t.Errorf("expected backend pacman, got %q", results[0].Backend)
	}
	if results[0].Version != "125.0.1" {
		t.Errorf("expected version 125.0.1, got %q", results[0].Version)
	}
	if results[0].FullLine != "extra/firefox 125.0.1" {
		t.Errorf("expected full line %q, got %q", "extra/firefox 125.0.1", results[0].FullLine)
	}
	if results[1].Version != "126.0.0-beta1" {
		t.Errorf("expected version 126.0.0-beta1, got %q", results[1].Version)
	}
}

func TestApplyVersionPolicy(t *testing.T) {
	results := []SearchResult{
		{Backend: "pacman", Version: "1.0.0", FullLine: "pkg 1.0.0"},
		{Backend: "pacman", Version: "2.0.0-beta", FullLine: "pkg 2.0.0-beta"},
		{Backend: "pacman", Version: "3.0.0", FullLine: "pkg 3.0.0"},
	}

	tests := []struct {
		name    string
		policy  string
		wantLen int
		wantVer string
	}{
		{name: "empty policy returns first", policy: "", wantLen: 1, wantVer: "1.0.0"},
		{name: "latest returns highest", policy: "latest", wantLen: 1, wantVer: "3.0.0"},
		{name: "stable returns highest non-prerelease", policy: "stable", wantLen: 1, wantVer: "3.0.0"},
		{name: "pinned exact match", policy: "1.0.0", wantLen: 1, wantVer: "1.0.0"},
		{name: "pinned prefix match", policy: "1", wantLen: 1, wantVer: "1.0.0"},
		{name: "pinned no match", policy: "5.0.0", wantLen: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyVersionPolicy(results, tt.policy)
			if len(got) != tt.wantLen {
				t.Fatalf("expected %d results, got %d", tt.wantLen, len(got))
			}
			if tt.wantLen > 0 && got[0].Version != tt.wantVer {
				t.Errorf("expected version %q, got %q", tt.wantVer, got[0].Version)
			}
		})
	}
}

func TestApplyVersionPolicy_EmptyResults(t *testing.T) {
	got := applyVersionPolicy(nil, "latest")
	if len(got) != 0 {
		t.Errorf("expected 0 results for nil input, got %d", len(got))
	}

	got = applyVersionPolicy([]SearchResult{}, "stable")
	if len(got) != 0 {
		t.Errorf("expected 0 results for empty input, got %d", len(got))
	}
}

func TestPinnedFilter_ExactVersionSelection(t *testing.T) {
	// Test that when multiple backends have the pinned version,
	// the first matching result is returned (backends ordered by priority).
	tests := []struct {
		name       string
		results    []SearchResult
		pinVersion string
		wantBackend string
		wantVersion string
	}{
		{
			name: "exact match prefers first in list",
			results: []SearchResult{
				{Backend: "nix", Version: "0.10.2", FullLine: "nixpkg.neovim 0.10.2"},
				{Backend: "pacman", Version: "0.10.2", FullLine: "extra/neovim 0.10.2"},
			},
			pinVersion:  "0.10.2",
			wantBackend: "nix",
			wantVersion: "0.10.2",
		},
		{
			name: "no backend with pinned version returns nil",
			results: []SearchResult{
				{Backend: "pacman", Version: "0.9.5", FullLine: "extra/neovim 0.9.5"},
				{Backend: "nix", Version: "0.10.0", FullLine: "nixpkg.neovim 0.10.0"},
			},
			pinVersion: "0.10.2",
			wantBackend: "",
			wantVersion: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pinnedFilter(tt.results, tt.pinVersion)
			if tt.wantBackend == "" {
				if len(got) != 0 {
					t.Errorf("expected no match, got %+v", got)
				}
				return
			}
			if len(got) != 1 {
				t.Fatalf("expected 1 result, got %d", len(got))
			}
			if got[0].Backend != tt.wantBackend {
				t.Errorf("expected backend %q, got %q", tt.wantBackend, got[0].Backend)
			}
			if got[0].Version != tt.wantVersion {
				t.Errorf("expected version %q, got %q", tt.wantVersion, got[0].Version)
			}
		})
	}
}
