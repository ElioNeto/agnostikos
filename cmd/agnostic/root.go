package agnostic

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/spf13/cobra"
)

var (
	Version = "0.1.0"
	Commit  = "dev"
)

var rootCmd = &cobra.Command{
	Use:   "agnostic",
	Short: "AgnosticOS Hybrid Package Manager",
	Long: `AgnosticOS - A meta-wrapper package manager that unifies multiple backends.

Supported backends:
  - Pacman (Arch Linux)
  - Nix   (NixOS)
  - Flatpak (Universal)

Examples:
  agnostic install firefox --backend pacman
  agnostic search neovim  --backend nix
  agnostic update         --backend flatpak`,
	Version: fmt.Sprintf("%s (commit: %s)", Version, Commit),
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

func TestRootCmd(t *testing.T) {
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"--version"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRootCmd_Help(t *testing.T) {
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected help output, got empty")
	}
}
