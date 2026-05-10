package agnostic

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/ElioNeto/agnostikos/internal/manager"
	"github.com/spf13/cobra"
)

var (
	searchLimit      int
	searchJSON       bool
	searchInstalled  bool
	searchRefresh    bool
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for packages in the specified backend or all backends",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := newManager()

		// If --refresh-cache is set, invalidate the entire cache before searching.
		if searchRefresh && mgr.Cache != nil {
			mgr.Cache.Invalidate()
		}

		// If no backend specified, search all backends
		if backend == "" {
			return searchAllBackends(cmd, mgr, args[0])
		}

		svc, ok := mgr.Backends[backend]
		if !ok {
			return fmt.Errorf("backend '%s' not found — available: %s", backend, strings.Join(mgr.ListBackends(), ", "))
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

// searchAllBackends searches all backends concurrently and aggregates results.
func searchAllBackends(cmd *cobra.Command, mgr *manager.AgnosticManager, query string) error {
	results, err := mgr.Resolver.SearchAll(cmd.Context(), query)
	if err != nil {
		return fmt.Errorf("search all backends failed: %w", err)
	}

	// Flatten and sort results
	type entry struct {
		line    string
		backend string
	}
	var entries []entry
	for b, lines := range results {
		for _, line := range lines {
			entries = append(entries, entry{line: line, backend: b})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].backend != entries[j].backend {
			return entries[i].backend < entries[j].backend
		}
		return entries[i].line < entries[j].line
	})

	// Apply limit
	if searchLimit > 0 && len(entries) > searchLimit {
		entries = entries[:searchLimit]
	}

	if searchJSON {
		// JSON: map of backend -> []string
		grouped := make(map[string][]string)
		for _, e := range entries {
			grouped[e.backend] = append(grouped[e.backend], e.line)
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		return enc.Encode(grouped)
	}

	if len(entries) == 0 {
		cmd.Printf("No packages found for %q in any backend\n", query)
		return nil
	}

	cmd.Printf("Results for %q across all backends:\n", query)
	currentBackend := ""
	for _, e := range entries {
		if e.backend != currentBackend {
			currentBackend = e.backend
			cmd.Printf("\n── %s ──\n", currentBackend)
		}
		cmd.Printf("  • %s\n", e.line)
	}
	return nil
}

func init() {
	searchCmd.Flags().StringVarP(&backend, "backend", "b", "", "Backend to use (pacman, nix, flatpak) — empty searches all backends")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "Output in JSON format")
	searchCmd.Flags().BoolVar(&searchInstalled, "installed", false, "Search only among installed packages")
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "n", 20, "Maximum number of results")
	searchCmd.Flags().BoolVar(&searchRefresh, "refresh-cache", false, "Invalidate cache before searching")
	rootCmd.AddCommand(searchCmd)
}
