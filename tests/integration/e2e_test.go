package integration

import (
	"errors"
	"strings"
	"testing"

	"github.com/ElioNeto/agnostikos/internal/config"
	"github.com/ElioNeto/agnostikos/internal/manager"
)

// mockErrorBackend implements manager.PackageService and always fails Install.
type mockErrorBackend struct {
	manager.PackageService
}

func (m *mockErrorBackend) Install(_ string) error {
	return errors.New("primary backend failure")
}

// TestE2E_PackageLifecycle tests the full lifecycle of a package:
// search → install → list → update → remove.
func TestE2E_PackageLifecycle(t *testing.T) {
	mgr := manager.NewAgnosticManager()
	mockSvc := NewMockPackageService()
	mgr.RegisterBackend("mock", mockSvc)

	t.Log("=== Search for a known package ===")
	results, err := mockSvc.Search("htop")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected Search to return results for 'htop'")
	}
	found := false
	for _, r := range results {
		if r == "htop" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected 'htop' in search results, got %v", results)
	}
	t.Logf("Search results: %v", results)

	t.Log("=== Install the package ===")
	if err := mockSvc.Install("htop"); err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	t.Log("=== List installed packages ===")
	list, err := mockSvc.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if !containsPackage(list, "htop") {
		t.Fatalf("expected 'htop' in list, got %v", list)
	}
	t.Logf("Installed packages: %v", list)

	t.Log("=== Update the package ===")
	if err := mockSvc.Update("htop"); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	t.Log("=== Remove the package ===")
	if err := mockSvc.Remove("htop"); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	t.Log("=== Verify package is gone ===")
	list, err = mockSvc.List()
	if err != nil {
		t.Fatalf("List after remove failed: %v", err)
	}
	if containsPackage(list, "htop") {
		t.Fatalf("expected 'htop' to be removed, but list still contains it: %v", list)
	}
	t.Log("Package lifecycle test completed successfully")
}

// TestE2E_ConfigInstall tests loading a config file and installing all packages
// from it using the mock backend.
func TestE2E_ConfigInstall(t *testing.T) {
	mockSvc := NewMockPackageService()

	mgr := manager.NewAgnosticManager()
	mgr.RegisterBackend("pacman", mockSvc)

	t.Log("=== Load and validate config ===")
	cfg, err := config.Load("testdata/agnostic_e2e.yaml")
	if err != nil {
		t.Fatalf("config.Load failed: %v", err)
	}
	t.Logf("Config loaded: profile=%s, default backend=%s", cfg.Profile, cfg.Backends.Default)

	t.Log("=== Install packages from config ===")
	allPkgs := append(cfg.Packages.Base, cfg.Packages.Extra...)
	for _, pkg := range allPkgs {
		svc, ok := mgr.Backends[cfg.Backends.Default]
		if !ok {
			t.Fatalf("backend %q not registered", cfg.Backends.Default)
		}
		if err := svc.Install(pkg); err != nil {
			t.Fatalf("Install(%q) failed: %v", pkg, err)
		}
		t.Logf("  installed %q", pkg)
	}

	t.Log("=== Verify all packages appear in list ===")
	list, err := mockSvc.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	for _, pkg := range allPkgs {
		if !containsPackage(list, pkg) {
			t.Fatalf("expected %q in list after config install, got %v", pkg, list)
		}
	}
	t.Logf("All %d packages installed and verified", len(allPkgs))
}

// TestE2E_BackendFallback demonstrates fallback behavior when a primary backend fails.
//
// NOTE: The current AgnosticManager does not implement automatic backend fallback.
// This test documents the expected behavior: when the primary backend fails,
// the user can retry with a fallback backend. Automatic retry/fallback is a
// future enhancement.
func TestE2E_BackendFallback(t *testing.T) {
	mockSvc := NewMockPackageService()
	primary := &mockErrorBackend{PackageService: mockSvc}

	mgr := manager.NewAgnosticManager()
	mgr.RegisterBackend("primary", primary)
	mgr.RegisterBackend("fallback", mockSvc)

	pkg := "htop"

	t.Log("=== Attempt install with primary backend (always fails) ===")
	err := mgr.Backends["primary"].Install(pkg)
	if err == nil {
		t.Fatal("expected primary backend to fail")
	}
	t.Logf("Primary backend failed as expected: %v", err)

	t.Log("=== Retry with fallback backend ===")
	if err := mgr.Backends["fallback"].Install(pkg); err != nil {
		t.Fatalf("Fallback install failed: %v", err)
	}

	t.Log("=== Verify package was installed by fallback ===")
	list, err := mockSvc.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if !containsPackage(list, pkg) {
		t.Fatalf("expected %q in list after fallback install, got %v", pkg, list)
	}
	t.Logf("Package %q successfully installed via fallback backend", pkg)
}

// containsPackage checks if a package name appears in a list of "name version" strings.
func containsPackage(list []string, name string) bool {
	for _, entry := range list {
		fields := strings.Fields(entry)
		if len(fields) > 0 && fields[0] == name {
			return true
		}
	}
	return false
}
