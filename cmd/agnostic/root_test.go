package agnostic

import (
	"bytes"
	"os/exec"
	"testing"

	"github.com/spf13/cobra"
)

func TestExecute(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name: "test version command",
			args: []string{"--version"},
			expected: fmt.Sprintf("AgnosticOS Hybrid Package Manager v%s (commit: %s)\n", Version, Commit),
		},
		{
			name: "test help command",
			args: []string{"--help"},
			expected: `AgnosticOS - A meta-wrapper package manager that unifies multiple backends.

Supported backends:
  - Pacman (Arch Linux)
  - Nix   (NixOS)
  - Flatpak (Universal)

Examples:
  agnostic install firefox --backend pacman
  agnostic search neovim  --backend nix
  agnostic update         --backend flatpak

Usage:
  agnostic [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell (currently zsh, bash, powershell, and fish)
  help        Help about any command
  install     Install a package from a specific backend
  search      Search for packages in a specific backend
  update      Update packages in a specific backend

Flags:
  -h, --help   help for agnostic

Use "agnostic [command] --help" for more information about a command.
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("go", "run", ".", "--"+strings.Join(tt.args, " "))
			var out bytes.Buffer
			cmd.Stdout = &out
			err := cmd.Run()
			if err != nil {
				t.Errorf("Execute() error = %v", err)
			}
			if out.String() != tt.expected {
				t.Errorf("Execute() output = %s; want %s", out.String(), tt.expected)
			}
		})
	}
}