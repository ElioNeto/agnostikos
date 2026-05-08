package agnostic

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ElioNeto/agnostikos/internal/config"
	"github.com/ElioNeto/agnostikos/internal/manager"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/ElioNeto/agnostikos/internal/isolation"
)

var (
	isolated bool
	profile  string
)

// profilePackages represents the package list in a profile YAML file.
type profilePackages struct {
	Packages []string `yaml:"packages"`
}

var installCmd = &cobra.Command{
	Use:   "install [package...]",
	Short: "Install a package or all packages from a config file",
	Long: `Install packages using the configured backend.

If --config is provided, installs all packages defined in the config file (base + extra).
Otherwise, install a single package specified as argument.`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If --profile is set, install packages from the profile file
		if profile != "" {
			if !config.SupportedProfiles[profile] {
				return fmt.Errorf("profile %q is not supported — must be one of: minimal, desktop, server, dev", profile)
			}

			profilePath := filepath.Join("configs", "profiles", profile+".yaml")
			data, err := os.ReadFile(profilePath)
			if err != nil {
				return fmt.Errorf("reading profile %q: %w", profile, err)
			}

			var pp profilePackages
			if err := yaml.Unmarshal(data, &pp); err != nil {
				return fmt.Errorf("parsing profile %q: %w", profile, err)
			}

			if len(pp.Packages) == 0 {
				fmt.Printf("No packages defined in profile %q\n", profile)
				return nil
			}

			fmt.Printf("📋 Profile %q — %d packages to install:\n", profile, len(pp.Packages))
			for _, pkg := range pp.Packages {
				fmt.Printf("   • %s\n", pkg)
			}

			mgr := manager.NewAgnosticManager()
			svc, ok := mgr.Backends[backend]
			if !ok {
				return fmt.Errorf("backend '%s' not found — available: pacman, nix, flatpak", backend)
			}

			for _, pkg := range pp.Packages {
				fmt.Printf("📦 Installing '%s' via %s...\n", pkg, backend)
				if err := svc.Install(pkg); err != nil {
					return fmt.Errorf("installation of '%s' failed: %w", pkg, err)
				}
				fmt.Printf("✅ '%s' installed successfully\n", pkg)
			}
			return nil
		}

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
			pkgs = append(pkgs, cfg.Packages.Dev...)
			pkgs = append(pkgs, cfg.Packages.Desktop...)

			// If profile is set in config, load profile packages too
			if cfg.Profile != "" {
				if !config.SupportedProfiles[cfg.Profile] {
					return fmt.Errorf("profile %q is not supported — must be one of: minimal, desktop, server, dev", cfg.Profile)
				}
				profilePath := filepath.Join("configs", "profiles", cfg.Profile+".yaml")
				data, err := os.ReadFile(profilePath)
				if err != nil {
					return fmt.Errorf("reading profile %q: %w", cfg.Profile, err)
				}
				var pp profilePackages
				if err := yaml.Unmarshal(data, &pp); err != nil {
					return fmt.Errorf("parsing profile %q: %w", cfg.Profile, err)
				}
				pkgs = append(pkgs, pp.Packages...)
			}

			// Deduplicate packages to prevent double-installation
			seen := make(map[string]struct{}, len(pkgs))
			unique := make([]string, 0, len(pkgs))
			for _, pkg := range pkgs {
				if _, ok := seen[pkg]; !ok {
					seen[pkg] = struct{}{}
					unique = append(unique, pkg)
				}
			}
			pkgs = unique

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
			return errors.New("requires a package name argument or --config flag")
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

func init() {
	installCmd.Flags().StringVarP(&backend, "backend", "b", "pacman", "Backend to use (pacman, nix, flatpak)")
	installCmd.Flags().BoolVarP(&isolated, "isolated", "i", false, "Run in isolated Linux namespace")
	installCmd.Flags().StringVarP(&profile, "profile", "p", "", "Profile to install (minimal, desktop, server, dev)")
	rootCmd.AddCommand(installCmd)
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
