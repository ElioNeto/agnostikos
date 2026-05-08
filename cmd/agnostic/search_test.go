package agnostic

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// resetSearchFlags resets package-level variables to their defaults
// so that tests don't leak state between each other.
func resetSearchFlags() {
	searchLimit = 20
	searchJSON = false
	searchInstalled = false
	backend = "pacman"
}

func TestSearchCmd_InvalidBackend(t *testing.T) {
	resetSearchFlags()
	rootCmd.SetArgs([]string{"search", "neovim", "--backend", "xyz"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid backend, got nil")
	}
	if !strings.Contains(err.Error(), "backend 'xyz' not found") {
		t.Fatalf("expected error about backend not found, got: %v", err)
	}
}

func TestSearchCmd_ValidBackend(t *testing.T) {
	resetSearchFlags()
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"search", "firefox", "--backend", "pacman"})
	err := rootCmd.Execute()
	// May fail if pacman not installed, but should not return "backend not found"
	if err != nil && strings.Contains(err.Error(), "backend 'pacman' not found") {
		t.Fatal("backend 'pacman' should be registered")
	}
}

func TestSearchCmd_JSONOutput(t *testing.T) {
	resetSearchFlags()
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"search", "firefox", "--backend", "pacman", "--json"})
	err := rootCmd.Execute()
	if err != nil {
		// If pacman not available, error is expected in CI
		if strings.Contains(err.Error(), "backend 'pacman' not found") {
			t.Fatal("backend 'pacman' should be registered")
		}
		return
	}
	// Validate JSON array output
	var results []string
	if err := json.Unmarshal(buf.Bytes(), &results); err != nil {
		t.Fatalf("expected valid JSON array, got error: %v (output: %q)", err, buf.String())
	}
}

func TestSearchCmd_LimitFlag(t *testing.T) {
	resetSearchFlags()
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"search", "firefox", "--backend", "pacman", "--limit", "5"})
	err := rootCmd.Execute()
	if err != nil {
		// If pacman not available, error is expected in CI
		if strings.Contains(err.Error(), "backend 'pacman' not found") {
			t.Fatal("backend 'pacman' should be registered")
		}
		return
	}
	// If successful, check output does not exceed limit
	output := strings.TrimSpace(buf.String())
	if output == "" {
		return // no results is fine
	}
	lines := strings.Split(output, "\n")
	if len(lines) > 5 {
		t.Fatalf("expected at most 5 lines with --limit 5, got %d", len(lines))
	}
}

func TestSearchCmd_NoResultsMessage(t *testing.T) {
	resetSearchFlags()
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"search", "xyznonexistent12345", "--backend", "pacman"})
	err := rootCmd.Execute()
	if err != nil {
		// If pacman not available, error is expected in CI
		if strings.Contains(err.Error(), "backend 'pacman' not found") {
			t.Fatal("backend 'pacman' should be registered")
		}
		return
	}
	output := buf.String()
	if !strings.Contains(output, "No packages found") {
		t.Fatalf("expected 'No packages found' message in output, got: %q", output)
	}
}

func TestSearchCmd_InstalledFlag(t *testing.T) {
	resetSearchFlags()

	root := &cobra.Command{Use: "root"}
	root.AddCommand(searchCmd)
	root.SetArgs([]string{"search", "firefox", "--installed"})

	buf := new(strings.Builder)
	root.SetOut(buf)
	root.SetErr(buf)

	err := root.Execute()
	if err != nil {
		// If pacman not available, error is expected in CI
		if strings.Contains(err.Error(), "backend 'pacman' not found") {
			t.Fatal("backend 'pacman' should be registered")
		}
		return
	}

	output := buf.String()
	// Should contain search results or "No packages found" message
	if output == "" {
		t.Error("expected some output")
	}
}

func TestSearchCmd_JSONWithLimit(t *testing.T) {
	resetSearchFlags()
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"search", "firefox", "--backend", "pacman", "--json", "--limit", "3"})
	err := rootCmd.Execute()
	if err != nil {
		if strings.Contains(err.Error(), "backend 'pacman' not found") {
			t.Fatal("backend 'pacman' should be registered")
		}
		return
	}
	var results []string
	if err := json.Unmarshal(buf.Bytes(), &results); err != nil {
		t.Fatalf("expected valid JSON array, got error: %v (output: %q)", err, buf.String())
	}
	if len(results) > 3 {
		t.Fatalf("expected at most 3 results with --limit 3, got %d", len(results))
	}
}
