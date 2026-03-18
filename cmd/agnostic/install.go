package agnostic

import (
	"fmt"

	"github.com/ElioNeto/agnostikos/internal/manager"
	"github.com/spf13/cobra"
)

var (
	backend  string
	isolated bool
)

var installCmd = &cobra.Command{
	Use:   "install [package]",
	Short: "Install a package via the specified backend",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := manager.NewAgnosticManager()
		svc, ok := mgr.Backends[backend]
		if !ok {
			return fmt.Errorf("backend '%s' not found — available: pacman, nix, flatpak", backend)
		}
		fmt.Printf("📦 Installing '%s' via %s...\n", args[0], backend)
		if isolated {
			fmt.Println("🔒 Running in isolated namespace...")
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