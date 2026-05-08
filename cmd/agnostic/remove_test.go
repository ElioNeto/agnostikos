package agnostic

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

// resetRemoveFlags resets package-level variables to their defaults
// so that tests don't leak state between each other.
func resetRemoveFlags() {
	removeDryRun = false
	removeYes = false
	backend = "pacman"
	stdinReader = nil
}

func TestRemoveCmd_InvalidBackend(t *testing.T) {
	resetRemoveFlags()
	rootCmd.SetArgs([]string{"remove", "firefox", "--backend", "xyz"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid backend, got nil")
	}
	if !strings.Contains(err.Error(), "backend 'xyz' not found") {
		t.Fatalf("expected error about backend not found, got: %v", err)
	}
}

func TestRemoveCmd_MissingArgs(t *testing.T) {
	resetRemoveFlags()
	rootCmd.SetArgs([]string{"remove"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing args, got nil")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Fatalf("expected error about arg count, got: %v", err)
	}
}

func TestRemoveCmd_DryRun(t *testing.T) {
	resetRemoveFlags()
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"remove", "firefox", "--backend", "pacman", "--dry-run"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error for dry-run, got: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Would remove") {
		t.Fatalf("expected dry-run message in output, got: %q", output)
	}
	if !strings.Contains(output, "dry-run") {
		t.Fatalf("expected 'dry-run' in output, got: %q", output)
	}
}

func TestRemoveCmd_YesFlag(t *testing.T) {
	resetRemoveFlags()
	rootCmd.SetArgs([]string{"remove", "firefox", "--backend", "pacman", "--yes"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected an error (pacman not available in CI), got nil")
	}
	// Must NOT be about backend not found
	if strings.Contains(err.Error(), "backend 'pacman' not found") {
		t.Fatal("--yes should skip confirmation and attempt execution, not fail on backend resolution")
	}
	// Must NOT be about aborting
	if strings.Contains(err.Error(), "Aborted") {
		t.Fatal("--yes should skip confirmation, not abort")
	}
}

func TestRemoveCmd_YesFlagShort(t *testing.T) {
	resetRemoveFlags()
	rootCmd.SetArgs([]string{"remove", "firefox", "--backend", "pacman", "-y"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected an error (pacman not available), got nil")
	}
	if strings.Contains(err.Error(), "backend 'pacman' not found") {
		t.Fatal("-y should skip confirmation and attempt execution, not fail on backend resolution")
	}
}

func TestRemoveCmd_ConfirmYes(t *testing.T) {
	resetRemoveFlags()
	stdinReader = bufio.NewReader(strings.NewReader("y\n"))
	t.Cleanup(func() { stdinReader = nil })

	rootCmd.SetArgs([]string{"remove", "firefox", "--backend", "pacman"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected an error (pacman not available), got nil")
	}
	if strings.Contains(err.Error(), "Aborted") {
		t.Fatal("confirmation with 'y' should proceed, not abort")
	}
}

func TestRemoveCmd_ConfirmYesWord(t *testing.T) {
	resetRemoveFlags()
	stdinReader = bufio.NewReader(strings.NewReader("yes\n"))
	t.Cleanup(func() { stdinReader = nil })

	rootCmd.SetArgs([]string{"remove", "firefox", "--backend", "pacman"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected an error (pacman not available), got nil")
	}
	if strings.Contains(err.Error(), "Aborted") {
		t.Fatal("confirmation with 'yes' should proceed, not abort")
	}
}

func TestRemoveCmd_ConfirmNo(t *testing.T) {
	resetRemoveFlags()
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	stdinReader = bufio.NewReader(strings.NewReader("n\n"))
	t.Cleanup(func() { stdinReader = nil })

	rootCmd.SetArgs([]string{"remove", "firefox", "--backend", "pacman"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error when user aborts, got: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Aborted") {
		t.Fatalf("expected 'Aborted' message in output, got: %q", output)
	}
}

func TestRemoveCmd_UseAndAliases(t *testing.T) {
	if removeCmd.Use != "remove [package]" {
		t.Errorf("expected Use 'remove [package]', got %q", removeCmd.Use)
	}
	if len(removeCmd.Aliases) != 2 {
		t.Errorf("expected 2 aliases, got %d", len(removeCmd.Aliases))
	}
	if removeCmd.Aliases[0] != "rm" || removeCmd.Aliases[1] != "uninstall" {
		t.Errorf("expected aliases [rm uninstall], got %v", removeCmd.Aliases)
	}
}
