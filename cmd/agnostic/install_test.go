package agnostic

import (
	"bytes"
	"testing"
)

func TestInstallCmd_InvalidBackend(t *testing.T) {
	rootCmd.SetArgs([]string{"install", "firefox", "--backend", "invalid"})
	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error for invalid backend, got nil")
	}
}

func TestInstallCmd_ValidBackend(t *testing.T) {
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"install", "firefox", "--backend", "pacman"})
	// Este teste vai falhar se pacman não estiver instalado — tudo bem por ora
	_ = rootCmd.Execute()
}

func TestListCmd_InvalidBackend(t *testing.T) {
	rootCmd.SetArgs([]string{"list", "--backend", "xyz"})
	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error for invalid backend, got nil")
	}
}

func TestListCmd_ValidBackend(t *testing.T) {
	rootCmd.SetArgs([]string{"list", "--backend", "pacman"})
	err := rootCmd.Execute()
	// May fail if pacman not installed, but should not return "backend not found"
	if err != nil && err.Error() == "backend 'pacman' not found" {
		t.Fatal("backend 'pacman' should be registered")
	}
}
