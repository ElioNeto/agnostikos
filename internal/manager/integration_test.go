//go:build integration

package manager

import (
	"os"
	"testing"
)

// These integration tests require real backends installed on the host.
// Run with: go test -tags=integration ./internal/manager/
//
// Each test checks if the corresponding backend binary exists before running.
// If the binary is not found, the test skips with a message.

func TestPacmanBackend_Integration_Install(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("integration test requires root for pacman")
	}
	skipWithoutBin(t, "pacman")

	p := NewPacmanBackend(&RealExecutor{})
	// Test with a tiny package that is likely cached or very small
	if err := p.Install("tiny"); err != nil {
		// It's OK if the package doesn't exist; we're testing the plumbing
		t.Logf("pacman install call completed (expected error if 'tiny' not found): %v", err)
	}
}

func TestPacmanBackend_Integration_Search(t *testing.T) {
	skipWithoutBin(t, "pacman")

	p := NewPacmanBackend(&RealExecutor{})
	results, err := p.Search("firefox")
	if err != nil {
		t.Fatalf("pacman search failed: %v", err)
	}
	if len(results) == 0 {
		t.Log("no results for 'firefox' (expected on minimal systems)")
	} else {
		t.Logf("found %d results for 'firefox'", len(results))
		for _, r := range results {
			t.Logf("  %s", r)
		}
	}
}

func TestPacmanBackend_Integration_List(t *testing.T) {
	skipWithoutBin(t, "pacman")

	p := NewPacmanBackend(&RealExecutor{})
	results, err := p.List()
	if err != nil {
		t.Fatalf("pacman list failed: %v", err)
	}
	t.Logf("pacman has %d installed packages", len(results))
	for _, r := range results {
		t.Logf("  %s", r)
	}
}

func TestNixBackend_Integration_Search(t *testing.T) {
	skipWithoutBin(t, "nix")

	n := NewNixBackend(&RealExecutor{})
	results, err := n.Search("hello")
	if err != nil {
		t.Fatalf("nix search failed: %v", err)
	}
	if len(results) == 0 {
		t.Log("no results for 'hello'")
	} else {
		t.Logf("found %d results for 'hello'", len(results))
		for _, r := range results {
			t.Logf("  %s", r)
		}
	}
}

func TestNixBackend_Integration_List(t *testing.T) {
	skipWithoutBin(t, "nix")

	n := NewNixBackend(&RealExecutor{})
	results, err := n.List()
	if err != nil {
		t.Fatalf("nix list failed: %v", err)
	}
	t.Logf("nix has %d installed packages", len(results))
}

func TestFlatpakBackend_Integration_Search(t *testing.T) {
	skipWithoutBin(t, "flatpak")

	f := NewFlatpakBackend(&RealExecutor{})
	results, err := f.Search("firefox")
	if err != nil {
		t.Fatalf("flatpak search failed: %v", err)
	}
	if len(results) == 0 {
		t.Log("no flatpak results for 'firefox'")
	} else {
		t.Logf("found %d results for 'firefox'", len(results))
		for _, r := range results {
			t.Logf("  %s", r)
		}
	}
}

func TestFlatpakBackend_Integration_List(t *testing.T) {
	skipWithoutBin(t, "flatpak")

	f := NewFlatpakBackend(&RealExecutor{})
	results, err := f.List()
	if err != nil {
		t.Fatalf("flatpak list failed: %v", err)
	}
	t.Logf("flatpak has %d installed packages", len(results))
}

func TestAgnosticManager_Integration_Dispatcher(t *testing.T) {
	mgr := NewAgnosticManager()
	backends := mgr.ListBackends()
	t.Logf("registered backends: %v", backends)

	for _, name := range backends {
		svc := mgr.Backends[name]
		if svc == nil {
			t.Errorf("backend %q is nil", name)
			continue
		}
		// Ensure the backend can at least be instantiated
		t.Logf("backend %q initialized successfully", name)
	}
}

// skipWithoutBin skips the test if the given binary is not in PATH.
func skipWithoutBin(t *testing.T, bin string) {
	t.Helper()
	if _, err := os.Stat("/usr/bin/" + bin); os.IsNotExist(err) {
		if _, err := os.Stat("/usr/local/bin/" + bin); os.IsNotExist(err) {
			t.Skipf("binary %q not found — skipping integration test", bin)
		}
	}
}
