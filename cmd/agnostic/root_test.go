package agnostic

import (
	"bytes"
	"os/exec"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCmd_Initialization(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"Use", "agnostic"},
		{"Short", "AgnosticOS Hybrid Package Manager"},
		{"Long", `AgnosticOS - A meta-wrapper package manager that unifies multiple backends.

Supported backends:
  - Pacman (Arch Linux)
  - Nix   (NixOS)
  - Flatpak (Universal)

Examples:
  agnostic install firefox --backend pacman
  agnostic search neovim  --backend nix
  agnostic update         --backend flatpak`},
		{"Version", fmt.Sprintf("%s (commit: %s)", Version, Commit)},
	}

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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if rootCmd.Use != tt.expected && tt.name == "Use" ||
				rootCmd.Short != tt.expected && tt.name == "Short" ||
				rootCmd.Long != tt.expected && tt.name == "Long" ||
				rootCmd.Version != tt.expected && tt.name == "Version" {
				t.Errorf("Expected %s to be '%s', got '%s'", tt.name, tt.expected, rootCmd.Use)
			}
		})
	}
}

func TestRootCmd_Execute(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"Output", `AgnosticOS Hybrid Package Manager

Supported backends:
  - Pacman (Arch Linux)
  - Nix   (NixOS)
  - Flatpak (Universal)

Examples:
  agnostic install firefox --backend pacman
  agnostic search neovim  --backend nix
  agnostic update         --backend flatpak
agnostic version: 0.1.0 (commit: dev)`},
	}

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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := &bytes.Buffer{}
			rootCmd.SetOut(output)
			rootCmd.Execute()

			if output.String() != tt.expected {
				t.Errorf("Expected output to be '%s', got '%s'", tt.expected, output.String())
			}
		})
	}
}

func TestRootCmd_InvalidCommand(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"Output", `Error: unknown command "invalid"

Usage:
  agnostic [flags]
  agnostic [command]

Available Commands:
  help        Help about any command
  install     Install a package using a specified backend
  search      Search for a package using a specified backend

Flags:
  -h, --help   help for agnostic

Use "agnostic [command] --help" for more information about a command.`},
	}

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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := &bytes.Buffer{}
			rootCmd.SetOut(output)
			rootCmd.ExecuteCommand(cobra.Command{Use: "invalid"}, []string{"invalid"})

			if output.String() != tt.expected {
				t.Errorf("Expected output to be '%s', got '%s'", tt.expected, output.String())
			}
		})
	}
}

func TestRootCmd_Version(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"Output", fmt.Sprintf("agnostic version: %s (commit: %s)\n", Version, Commit)},
	}

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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := &bytes.Buffer{}
			rootCmd.SetOut(output)
			rootCmd.ExecuteCommand(cobra.Command{Use: "version"}, []string{"version"})

			if output.String() != tt.expected {
				t.Errorf("Expected output to be '%s', got '%s'", tt.expected, output.String())
			}
		})
	}
}