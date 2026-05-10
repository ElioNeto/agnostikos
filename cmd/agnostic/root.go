package agnostic

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ElioNeto/agnostikos/internal/cache"
	"github.com/ElioNeto/agnostikos/internal/config"
	"github.com/ElioNeto/agnostikos/internal/manager"
	"github.com/spf13/cobra"
)

var (
	Version    = "0.1.0"
	Commit     = "dev"
	configFile string
	backend    string
	noSandbox  bool
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

var validateCmd = &cobra.Command{
	Use:   "validate <config-file>",
	Short: "Validate an agnostic.yaml configuration file",
	Long: `Validates an agnostic.yaml configuration file and reports all issues found.

Checks:
  - version is set
  - locale format (e.g. pt_BR.UTF-8)
  - timezone format (e.g. America/Sao_Paulo)
  - backend values (must be pacman, nix, or flatpak)
  - package names are not empty
  - user.shell is an absolute path when set

Exit code 0 = valid, 1 = invalid.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(args[0])
		if err != nil {
			// config.Load already calls Validate, so this handles all errors
			return fmt.Errorf("❌ config validation failed:\n%w", err)
		}
		fmt.Printf("✅ Config file '%s' is valid\n", args[0])
		fmt.Printf("  Version:  %s\n", cfg.Version)
		fmt.Printf("  Locale:   %s\n", cfg.Locale)
		fmt.Printf("  Timezone: %s\n", cfg.Timezone)
		fmt.Printf("  Backend:  %s", cfg.Backends.Default)
		if cfg.Backends.Fallback != "" {
			fmt.Printf(" (fallback: %s)", cfg.Backends.Fallback)
		}
		fmt.Println()
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// newManager creates a manager.AgnosticManager respecting the --no-sandbox flag
// and initialises the package metadata cache with config or default settings.
//
// Cache defaults:
//   - Directory: ~/.cache/agnostikos
//   - Stable TTL: 24h
//   - Latest TTL: 1h
//
// If a config file is provided via --config/-c, cache settings from the
// config override these defaults.
func newManager() *manager.AgnosticManager {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	cacheDir = filepath.Join(cacheDir, "agnostikos")

	stableTTL := 24 * time.Hour
	latestTTL := 1 * time.Hour

	// Load config if provided to apply cache settings.
	if configFile != "" {
		if cfg, cfgErr := config.Load(configFile); cfgErr == nil {
			if cfg.Cache.Dir != "" {
				if filepath.IsAbs(cfg.Cache.Dir) {
					cacheDir = cfg.Cache.Dir
				} else {
					// Resolve relative path from config file directory.
					configDir := filepath.Dir(configFile)
					if absPath, absErr := filepath.Abs(filepath.Join(configDir, cfg.Cache.Dir)); absErr == nil {
						cacheDir = absPath
					}
				}
			}
			if cfg.Cache.StableTTL > 0 {
				stableTTL = cfg.Cache.StableTTL
			}
			if cfg.Cache.LatestTTL > 0 {
				latestTTL = cfg.Cache.LatestTTL
			}
		}
	}

	pkgCache := cache.New(cacheDir, stableTTL, latestTTL)

	opts := []func(*manager.AgnosticManager){manager.WithCache(pkgCache)}
	if noSandbox {
		opts = append(opts, manager.WithNoSandbox())
	}
	return manager.NewAgnosticManager(opts...)
}

// RootCmd returns the root command for use by external tooling (e.g. doc generation).
func RootCmd() *cobra.Command {
	return rootCmd
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Path to agnostic.yaml config file")
	rootCmd.PersistentFlags().BoolVar(&noSandbox, "no-sandbox", false, "Disable Linux namespace isolation for backend commands")
	rootCmd.AddCommand(validateCmd)
}

func TestRootCmd(t *testing.T) {
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"--version"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRootCmdHelp(t *testing.T) {
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
