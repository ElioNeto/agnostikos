package agnostic

import (
	"bytes"
	"strings"
	"testing"
)

// resetUpdateFlags resets package-level variables to their defaults
// so that tests don't leak state between each other.
func resetUpdateFlags() {
	updateAll = false
	updateDryRun = false
	backend = "pacman"
}

func TestUpdateCmd_InvalidBackend(t *testing.T) {
	resetUpdateFlags()
	rootCmd.SetArgs([]string{"update", "--backend", "xyz"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid backend, got nil")
	}
	if !strings.Contains(err.Error(), "backend 'xyz' not found") {
		t.Fatalf("expected error about backend not found, got: %v", err)
	}
}

func TestUpdateCmd_SpecificPackage_DryRun(t *testing.T) {
	skipIfNoBackend(t, "pacman")
	resetUpdateFlags()
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"update", "firefox", "--backend", "pacman", "--dry-run"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error for dry-run, got: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Would update") {
		t.Fatalf("expected dry-run message in output, got: %q", output)
	}
	if !strings.Contains(output, "firefox") {
		t.Fatalf("expected package name in output, got: %q", output)
	}
	if !strings.Contains(output, "dry-run") {
		t.Fatalf("expected 'dry-run' in output, got: %q", output)
	}
}

func TestUpdateCmd_All_DryRun(t *testing.T) {
	skipIfNoBackend(t, "pacman")
	resetUpdateFlags()
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"update", "--backend", "pacman", "--all", "--dry-run"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error for dry-run, got: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Would update all packages") {
		t.Fatalf("expected dry-run message for all packages in output, got: %q", output)
	}
	if !strings.Contains(output, "dry-run") {
		t.Fatalf("expected 'dry-run' in output, got: %q", output)
	}
}

func TestUpdateCmd_NoArgsDefaultToAll_DryRun(t *testing.T) {
	skipIfNoBackend(t, "pacman")
	resetUpdateFlags()
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"update", "--backend", "pacman", "--dry-run"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error for dry-run, got: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Would update all packages") {
		t.Fatalf("expected dry-run message for all (no args) in output, got: %q", output)
	}
}

func TestUpdateCmd_InvalidBackendWithPackage(t *testing.T) {
	resetUpdateFlags()
	rootCmd.SetArgs([]string{"update", "firefox", "--backend", "invalid"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid backend, got nil")
	}
	if !strings.Contains(err.Error(), "backend 'invalid' not found") {
		t.Fatalf("expected error about backend not found, got: %v", err)
	}
}

func TestUpdateCmd_TooManyArgs(t *testing.T) {
	resetUpdateFlags()
	rootCmd.SetArgs([]string{"update", "firefox", "git"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for too many args, got nil")
	}
	if !strings.Contains(err.Error(), "accepts at most 1 arg") {
		t.Fatalf("expected error about max args, got: %v", err)
	}
}
