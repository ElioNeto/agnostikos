package agnostic

import (
	"bytes"
	"fmt"
	"os/exec"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCmd_Initialization(t *testing.T) {
	rootCmd := &cobra.Command{
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

	if rootCmd.Use != "agnostic" {
		t.Errorf("Expected Use to be 'agnostic', got '%s'", rootCmd.Use)
	}
	if rootCmd.Short != "AgnosticOS Hybrid Package Manager" {
		t.Errorf("Expected Short to be 'AgnosticOS Hybrid Package Manager', got '%s'", rootCmd.Short)
	}
	if rootCmd.Long != `AgnosticOS - A meta-wrapper package manager that unifies multiple backends.

Supported backends:
  - Pacman (Arch Linux)
  - Nix   (NixOS)
  - Flatpak (Universal)

Examples:
  agnostic install firefox --backend pacman
  agnostic search neovim  --backend nix
  agnostic update         --backend flatpak` {
		t.Errorf("Expected Long to be the provided text, got '%s'", rootCmd.Long)
	}
	if rootCmd.Version != fmt.Sprintf("%s (commit: %s)", Version, Commit) {
		t.Errorf("Expected Version to be '%s', got '%s'", fmt.Sprintf("%s (commit: %s)", Version, Commit), rootCmd.Version)
	}
}

func TestRootCmd_Execute(t *testing.T) {
	rootCmd := &cobra.Command{
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

	output := &bytes.Buffer{}
	rootCmd.SetOut(output)
	rootCmd.Execute()

	expectedOutput := `AgnosticOS Hybrid Package Manager

Supported backends:
  - Pacman (Arch Linux)
  - Nix   (NixOS)
  - Flatpak (Universal)

Examples:
  agnostic install firefox --backend pacman
  agnostic search neovim  --backend nix
  agnostic update         --backend flatpak
agnostic version: 0.1.0 (commit: dev)
`
	if output.String() != expectedOutput {
		t.Errorf("Expected output to be '%s', got '%s'", expectedOutput, output.String())
	}
}

func TestRootCmd_InvalidCommand(t *testing.T) {
	rootCmd := &cobra.Command{
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

	output := &bytes.Buffer{}
	rootCmd.SetOut(output)
	rootCmd.ExecuteCommand(cobra.Command{Use: "invalid"}, []string{"invalid"})

	expectedOutput := `Error: unknown command "invalid"

Usage:
  agnostic [flags]
  agnostic [command]

Available Commands:
  help        Help about any command
  install     Install a package using a specified backend
  search      Search for a package using a specified backend

Flags:
  -h, --help   help for agnostic

Use "agnostic [command] --help" for more information about a command.
`
	if output.String() != expectedOutput {
		t.Errorf("Expected output to be '%s', got '%s'", expectedOutput, output.String())
	}
}

func TestRootCmd_Version(t *testing.T) {
	rootCmd := &cobra.Command{
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

	output := &bytes.Buffer{}
	rootCmd.SetOut(output)
	rootCmd.ExecuteCommand(cobra.Command{Use: "version"}, []string{"version"})

	expectedOutput := fmt.Sprintf("agnostic version: %s (commit: %s)\n", Version, Commit)
	if output.String() != expectedOutput {
		t.Errorf("Expected output to be '%s', got '%s'", expectedOutput, output.String())
	}
}