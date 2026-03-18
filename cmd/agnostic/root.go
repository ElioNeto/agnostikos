package agnostic

import (
	"fmt"
	"os"

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

var isolatedCmd = &cobra.Command{
	Use:   "isolated",
	Short: "Install a package in an isolated environment",
	Long: `Install a package in an isolated environment using a specified backend.

Usage:
  agnostic install <package-name> --backend <backend>
  agnostic install <package-name> --isolated`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(isolatedCmd)

	isolatedCmd.Flags().StringP("backend", "b", "", "Specify the package manager backend (pacman, nix, flatpak)")
	isolatedCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n%s\n", rootCmd.Long, isolatedCmd.Long)
	})
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}