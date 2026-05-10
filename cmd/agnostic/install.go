package agnostic

import (
	"context"
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
			policy := policyFromConfig(nil)
			for _, pkg := range pp.Packages {
				b, err := resolveBackend(cmd.Context(), mgr, pkg, policy)
				if err != nil {
					return err
				}
				fmt.Printf("📦 Installing '%s' via %s...\n", pkg, b)
				if err := mgr.Backends[b].Install(pkg); err != nil {
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

			policy := policyFromConfig(cfg)
			for _, pkg := range pkgs {
				b, err := resolveBackend(cmd.Context(), mgr, pkg, policy)
				if err != nil {
					return err
				}
				fmt.Printf("📦 Installing '%s' via %s...\n", pkg, b)
				if err := mgr.Backends[b].Install(pkg); err != nil {
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
		policy := policyFromConfig(nil)

		// Determine which backend to use
		b := backend
		if b == "" {
			// Auto-resolve using the resolver
			var err error
			b, err = resolveBackend(cmd.Context(), mgr, args[0], policy)
			if err != nil {
				return err
			}
		}

		svc, ok := mgr.Backends[b]
		if !ok {
			return fmt.Errorf("backend '%s' not found — available: pacman, nix, flatpak", b)
		}
		fmt.Printf("📦 Installing '%s' via %s...\n", args[0], b)
		if isolated {
			fmt.Println("🔒 Running in isolated namespace...")
			binArgs, err := backendInstallArgs(b, args[0])
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
	installCmd.Flags().StringVarP(&backend, "backend", "b", "", "Backend to use (pacman, nix, flatpak) — empty uses auto-resolution")
	installCmd.Flags().BoolVarP(&isolated, "isolated", "i", false, "Run in isolated Linux namespace")
	installCmd.Flags().StringVarP(&profile, "profile", "p", "", "Profile to install (minimal, desktop, server, dev)")
	rootCmd.AddCommand(installCmd)
}

// resolveBackend uses the Resolver to find the best backend for a package.
func resolveBackend(ctx context.Context, mgr *manager.AgnosticManager, pkg string, policy manager.ResolvePolicy) (string, error) {
	result, err := mgr.ResolvePackage(ctx, pkg, policy)
	if err != nil {
		return "", fmt.Errorf("could not resolve backend for %q: %w", pkg, err)
	}
	return result.Backend, nil
}

// policyFromConfig builds a ResolvePolicy from a config file (or uses defaults).
func policyFromConfig(cfg *config.Config) manager.ResolvePolicy {
	policy := manager.ResolvePolicy{
		Priority: []string{"pacman", "nix", "flatpak"},
		Version:  "latest",
		Fallback: true,
	}
	if cfg != nil {
		if len(cfg.Backends.Priority) > 0 {
			policy.Priority = cfg.Backends.Priority
		}
		if cfg.Backends.Version != "" {
			policy.Version = cfg.Backends.Version
		}
		policy.Fallback = cfg.Backends.FallbackEnabled
	}
	return policy
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
