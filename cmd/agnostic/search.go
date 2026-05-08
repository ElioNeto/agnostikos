package agnostic

import (
	"encoding/json"
	"fmt"

	"github.com/ElioNeto/agnostikos/internal/manager"
	"github.com/spf13/cobra"
)

var (
	searchLimit     int
	searchJSON      bool
	searchInstalled bool
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for packages in the specified backend",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := manager.NewAgnosticManager()
		svc, ok := mgr.Backends[backend]
		if !ok {
			return fmt.Errorf("backend '%s' not found — available: pacman, nix, flatpak", backend)
		}

		results, err := svc.Search(args[0])
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}

		// Filter to installed packages if --installed flag is set
		if searchInstalled {
			installed, err := svc.List()
			if err != nil {
				return fmt.Errorf("listing installed packages: %w", err)
			}
			installedSet := make(map[string]struct{}, len(installed))
			for _, p := range installed {
				installedSet[p] = struct{}{}
			}
			filtered := make([]string, 0, len(results))
			for _, r := range results {
				if _, ok := installedSet[r]; ok {
					filtered = append(filtered, r)
				}
			}
			results = filtered
		}

		// Apply limit
		if searchLimit > 0 && len(results) > searchLimit {
			results = results[:searchLimit]
		}

		if searchJSON {
			enc := json.NewEncoder(cmd.OutOrStdout())
			return enc.Encode(results)
		}

		// Standard format
		if len(results) == 0 {
			cmd.Printf("No packages found for %q in %s\n", args[0], backend)
			return nil
		}

		for _, r := range results {
			cmd.Printf("  • %s  [%s]\n", r, backend)
		}
		return nil
	},
}

func init() {
	searchCmd.Flags().StringVarP(&backend, "backend", "b", "pacman", "Backend to use (pacman, nix, flatpak)")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "Output in JSON format")
	searchCmd.Flags().BoolVar(&searchInstalled, "installed", false, "Search only among installed packages")
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "n", 20, "Maximum number of results")
	rootCmd.AddCommand(searchCmd)
}
