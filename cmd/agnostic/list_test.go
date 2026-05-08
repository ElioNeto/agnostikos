package agnostic

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestListCmd_Default(t *testing.T) {
	resetListFlags()
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"list"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	output := buf.String()
	if output == "" {
		// No backends available in this environment — that is acceptable
		return
	}
	// Every line should end with [backendname]
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if !strings.HasSuffix(line, "]") {
			t.Errorf("expected line to end with [backend], got: %q", line)
		}
	}
}

func TestListCmd_BackendFilter(t *testing.T) {
	resetListFlags()
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"list", "--backend", "nix"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	output := buf.String()
	if output == "" {
		return // no packages from nix (not installed)
	}
	// Every line should end with [nix]
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if !strings.HasSuffix(line, "[nix]") {
			t.Errorf("expected line to end with [nix], got: %q", line)
		}
	}
}

func TestListCmd_BackendInvalid(t *testing.T) {
	resetListFlags()
	rootCmd.SetArgs([]string{"list", "--backend", "xyz"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid backend, got nil")
	}
	if !strings.Contains(err.Error(), "backend 'xyz' not found") {
		t.Fatalf("expected error about backend not found, got: %v", err)
	}
}

func TestListCmd_JSON(t *testing.T) {
	resetListFlags()
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"list", "--json"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// Validate JSON array output
	var entries []map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entries); err != nil {
		t.Fatalf("expected valid JSON array, got error: %v (output: %q)", err, buf.String())
	}
	// If there are entries, they must have the "backend" key
	for _, e := range entries {
		if _, ok := e["backend"]; !ok {
			t.Errorf("expected each entry to have a 'backend' key, got: %v", e)
		}
		if _, ok := e["name"]; !ok {
			t.Errorf("expected each entry to have a 'name' key, got: %v", e)
		}
	}
}

func TestListCmd_Export(t *testing.T) {
	resetListFlags()
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"list", "--export"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	output := buf.String()
	if !strings.HasPrefix(output, "packages:") {
		t.Fatalf("expected output to start with 'packages:', got: %q", output)
	}
}

func TestListCmd_JSONWithBackend(t *testing.T) {
	resetListFlags()
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"list", "--json", "--backend", "pacman"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	var entries []map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entries); err != nil {
		t.Fatalf("expected valid JSON array, got error: %v (output: %q)", err, buf.String())
	}
	// All entries should have backend == "pacman"
	for _, e := range entries {
		if e["backend"] != "pacman" {
			t.Errorf("expected all entries to have backend 'pacman', got: %v", e["backend"])
		}
	}
}

func TestListCmd_ExportWithBackend(t *testing.T) {
	resetListFlags()
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"list", "--export", "--backend", "nix"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	output := buf.String()
	if !strings.HasPrefix(output, "packages:") {
		t.Fatalf("expected output to start with 'packages:', got: %q", output)
	}
}
