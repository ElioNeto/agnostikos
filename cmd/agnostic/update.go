package agnostic

import (
	"fmt"

	"github.com/ElioNeto/agnostikos/internal/manager"
	"github.com/spf13/cobra"
)

var (
	updateAll    bool
	updateDryRun bool
)

var updateCmd = &cobra.Command{
	Use:   "update [package]",
	Short: "Update packages in the specified backend",
	Long: `Update a specific package or all packages using the configured backend.

If --all is provided or no package argument is given, updates all packages.
Otherwise, updates the specified package.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if updateDryRun {
			if len(args) > 0 {
				cmd.Printf("Would update '%s' via %s (dry-run)\n", args[0], backend)
			} else {
				cmd.Printf("Would update all packages via %s (dry-run)\n", backend)
			}
			return nil
		}

		mgr := manager.NewAgnosticManager()
		svc, ok := mgr.Backends[backend]
		if !ok {
			return fmt.Errorf("backend '%s' not found — available: pacman, nix, flatpak", backend)
		}

		if updateAll || len(args) == 0 {
			cmd.Printf("🔄 Updating all packages via %s...\n", backend)
			if err := svc.UpdateAll(); err != nil {
				return fmt.Errorf("update failed: %w", err)
			}
			cmd.Println("✅ Update complete")
			return nil
		}

		pkg := args[0]
		cmd.Printf("🔄 Updating '%s' via %s...\n", pkg, backend)
		if err := svc.Update(pkg); err != nil {
			return fmt.Errorf("update of '%s' failed: %w", pkg, err)
		}
		cmd.Printf("✅ '%s' updated\n", pkg)
		return nil
	},
}

func init() {
	updateCmd.Flags().BoolVarP(&updateAll, "all", "a", false, "Update all packages")
	updateCmd.Flags().BoolVar(&updateDryRun, "dry-run", false, "Simulate update without executing")
	updateCmd.Flags().StringVarP(&backend, "backend", "b", "pacman", "Backend to use (pacman, nix, flatpak)")
	rootCmd.AddCommand(updateCmd)
}
