package agnostic

import (
	"fmt"
	"strings"

	"github.com/ElioNeto/agnostikos/internal/config"
	"github.com/ElioNeto/agnostikos/internal/manager"
	"github.com/spf13/cobra"

	"github.com/ElioNeto/agnostikos/internal/isolation"
)

var (
	backend  string
	isolated bool
)

var installCmd = &cobra.Command{
	Use:   "install [package...]",
	Short: "Install a package or all packages from a config file",
	Long: `Install packages using the configured backend.

If --config is provided, installs all packages defined in the config file (base + extra).
Otherwise, install a single package specified as argument.`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If --config is set, install all packages from the config file
		if configFile != "" {
			cfg, err := config.Load(configFile)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("invalid config: %w", err)
			}

			mgr := manager.NewAgnosticManager()
			pkgs := append(cfg.Packages.Base, cfg.Packages.Extra...)
			if len(pkgs) == 0 {
				fmt.Println("No packages defined in config")
				return nil
			}

			svc, ok := mgr.Backends[cfg.Backends.Default]
			if !ok {
				return fmt.Errorf("backend '%s' not found — available: pacman, nix, flatpak", cfg.Backends.Default)
			}

			for _, pkg := range pkgs {
				fmt.Printf("📦 Installing '%s' via %s...\n", pkg, cfg.Backends.Default)
				if err := svc.Install(pkg); err != nil {
					return fmt.Errorf("installation of '%s' failed: %w", pkg, err)
				}
				fmt.Printf("✅ '%s' installed successfully\n", pkg)
			}
			return nil
		}

		// Fallback: install a single package from args
		if len(args) == 0 {
			return fmt.Errorf("requires a package name argument or --config flag")
		}

		mgr := manager.NewAgnosticManager()
		svc, ok := mgr.Backends[backend]
		if !ok {
			return fmt.Errorf("backend '%s' not found — available: pacman, nix, flatpak", backend)
		}
		fmt.Printf("📦 Installing '%s' via %s...\n", args[0], backend)
		if isolated {
			fmt.Println("🔒 Running in isolated namespace...")
			binArgs, err := backendInstallArgs(backend, args[0])
			if err != nil {
				return err
			}
			return isolation.RunIsolated(binArgs[0], binArgs[1:]...)
		}
		if err := svc.Install(args[0]); err != nil {
			return fmt.Errorf("installation failed: %w", err)
		}
		fmt.Printf("✅ '%s' installed successfully\n", args[0])
		return nil
	},
}

var removeCmd = &cobra.Command{
	Use:     "remove [package]",
	Aliases: []string{"rm", "uninstall"},
	Short:   "Remove a package via the specified backend",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := manager.NewAgnosticManager()
		svc, ok := mgr.Backends[backend]
		if !ok {
			return fmt.Errorf("backend '%s' not found", backend)
		}
		fmt.Printf("🗑️  Removing '%s' via %s...\n", args[0], backend)
		if err := svc.Remove(args[0]); err != nil {
			return fmt.Errorf("removal failed: %w", err)
		}
		fmt.Printf("✅ '%s' removed\n", args[0])
		return nil
	},
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update all packages in the specified backend",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := manager.NewAgnosticManager()
		svc, ok := mgr.Backends[backend]
		if !ok {
			return fmt.Errorf("backend '%s' not found", backend)
		}
		fmt.Printf("🔄 Updating via %s...\n", backend)
		if err := svc.Update(); err != nil {
			return fmt.Errorf("update failed: %w", err)
		}
		fmt.Println("✅ Update complete")
		return nil
	},
}

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for packages in the specified backend",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := manager.NewAgnosticManager()
		svc, ok := mgr.Backends[backend]
		if !ok {
			return fmt.Errorf("backend '%s' not found", backend)
		}
		fmt.Printf("🔍 Searching '%s' in %s...\n", args[0], backend)
		results, err := svc.Search(args[0])
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}
		for _, r := range results {
			fmt.Println(r)
		}
		return nil
	},
}

func init() {
	for _, cmd := range []*cobra.Command{installCmd, removeCmd, updateCmd, searchCmd} {
		cmd.Flags().StringVarP(&backend, "backend", "b", "pacman", "Backend to use (pacman, nix, flatpak)")
	}
	installCmd.Flags().BoolVarP(&isolated, "isolated", "i", false, "Run in isolated Linux namespace")
	rootCmd.AddCommand(installCmd, removeCmd, updateCmd, searchCmd)
}

func backendInstallArgs(backend, pkg string) ([]string, error) {
	switch backend {
	case "pacman":
		return []string{"pacman", "-S", "--noconfirm", pkg}, nil
	case "nix":
		if !strings.Contains(pkg, "#") {
			pkg = "nixpkgs#" + pkg
		}
		return []string{"nix", "profile", "install", pkg}, nil
	case "flatpak":
		return []string{"flatpak", "install", "--noninteractive", "-y", pkg}, nil
	default:
		return nil, fmt.Errorf("backend '%s' not supported for isolated mode", backend)
	}
}
