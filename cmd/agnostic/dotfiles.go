package agnostic

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ElioNeto/agnostikos/internal/dotfiles"
	"github.com/spf13/cobra"
)

var (
	dotfilesFrom  string
	dotfilesForce bool
)

var dotfilesCmd = &cobra.Command{
	Use:   "dotfiles",
	Short: "Manage dotfiles (symlink configuration files)",
	Long: `Manage dotfiles — symlink configuration files from the configs/ directory
to their XDG-standard locations in the home directory.

Supports:

  apply    Create symlinks for all managed dotfiles.
           Use --from to clone an external git repo of dotfiles.
           Use --force to overwrite existing files (no backup).

  list     List available dotfiles in the configs/ directory.

  diff     Compare dotfiles in configs/ with those in the home directory.

Examples:
  agnostic dotfiles apply
  agnostic dotfiles apply --from https://github.com/user/dotfiles.git
  agnostic dotfiles apply --force
  agnostic dotfiles list
  agnostic dotfiles diff`,
}

var dotfilesApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply dotfiles by creating symlinks",
	Long: `Create symlinks for all managed dotfiles from the configs/ directory
to their XDG-standard locations.

Flags:
  --from    Git URL or local path to clone dotfiles from
  --force   Overwrite existing files without creating backups

The configs/ directory is resolved relative to the agnostic.yaml config file
location, falling back to "./configs/" relative to the working directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configsDir := resolveConfigsDir()
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("getting home directory: %w", err)
		}

		mgr := dotfiles.New(dotfilesFrom)
		if err := mgr.Apply(configsDir, homeDir, dotfilesForce); err != nil {
			return fmt.Errorf("applying dotfiles: %w", err)
		}
		fmt.Println("✅ Dotfiles applied successfully")
		return nil
	},
}

var dotfilesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available dotfiles",
	Long:  `List all available dotfiles in the configs/ directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configsDir := resolveConfigsDir()
		mgr := dotfiles.New("")
		list, err := mgr.List(configsDir)
		if err != nil {
			return fmt.Errorf("listing dotfiles: %w", err)
		}

		if len(list) == 0 {
			fmt.Println("No dotfiles found in", configsDir)
			return nil
		}

		fmt.Println("Available dotfiles:")
		for _, item := range list {
			fmt.Printf("  📄 %s\n", item)
		}
		return nil
	},
}

var dotfilesDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare dotfiles with home directory",
	Long:  `Compare dotfiles in configs/ with those currently in the home directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configsDir := resolveConfigsDir()
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("getting home directory: %w", err)
		}

		mgr := dotfiles.New("")
		diffs, err := mgr.Diff(configsDir, homeDir)
		if err != nil {
			return fmt.Errorf("diffing dotfiles: %w", err)
		}

		if len(diffs) == 0 {
			fmt.Println("No dotfiles found.")
			return nil
		}

		fmt.Println("Dotfiles status:")
		for _, d := range diffs {
			prefix := "  ✅"
			if strings.Contains(d, "MISSING") {
				prefix = "  ❌"
			} else if strings.Contains(d, "DIFFERENT") {
				prefix = "  ⚠️ "
			}
			fmt.Printf("%s %s\n", prefix, d)
		}
		return nil
	},
}

func init() {
	dotfilesApplyCmd.Flags().StringVarP(&dotfilesFrom, "from", "", "", "Git URL or local path to clone dotfiles from")
	dotfilesApplyCmd.Flags().BoolVarP(&dotfilesForce, "force", "f", false, "Overwrite existing files without backup")
	dotfilesCmd.AddCommand(dotfilesApplyCmd)
	dotfilesCmd.AddCommand(dotfilesListCmd)
	dotfilesCmd.AddCommand(dotfilesDiffCmd)
	rootCmd.AddCommand(dotfilesCmd)
}

// resolveConfigsDir resolves the configs/ directory path.
// It checks for a config file flag first, then falls back to "./configs/".
func resolveConfigsDir() string {
	// If a config file was provided, look for configs/ next to it
	if configFile != "" {
		cfgDir := filepath.Dir(configFile)
		candidate := filepath.Join(cfgDir, "configs")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	// Fallback to ./configs/ relative to working directory
	candidate := "./configs"
	if info, err := os.Stat(candidate); err == nil && info.IsDir() {
		abs, _ := filepath.Abs(candidate)
		return abs
	}
	return candidate
}
