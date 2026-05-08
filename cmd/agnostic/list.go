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
	listJSON   bool
	listExport bool
)

// listEntry represents a single package found in a backend.
type listEntry struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Backend string `json:"backend"`
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed packages",
	Long: `List installed packages from all backends or a specific one.

By default iterates over all backends (pacman, nix, flatpak) and
displays each package as "name  version  [backend]".

Use --backend to filter to a single backend.
Use --json to output a JSON array.
Use --export to output a YAML-compatible block.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := manager.NewAgnosticManager()

		// Determine which backends to query
		backendsToQuery := make(map[string]manager.PackageService)
		if cmd.Flags().Changed("backend") {
			svc, ok := mgr.Backends[backend]
			if !ok {
				return fmt.Errorf("backend '%s' not found — available: %s", backend, strings.Join(mgr.ListBackends(), ", "))
			}
			backendsToQuery[backend] = svc
		} else {
			backendsToQuery = mgr.Backends
		}

		// Collect entries from each backend
		var entries []listEntry
		for name, svc := range backendsToQuery {
			results, err := svc.List()
			if err != nil {
				// Backend not available — skip gracefully
				continue
			}
			for _, line := range results {
				entry := parseListLine(line, name)
				entries = append(entries, entry)
			}
		}

		// Sort by backend, then by name
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].Backend != entries[j].Backend {
				return entries[i].Backend < entries[j].Backend
			}
			return entries[i].Name < entries[j].Name
		})

		// JSON output
		if listJSON {
			enc := json.NewEncoder(cmd.OutOrStdout())
			return enc.Encode(entries)
		}

		// YAML export output
		if listExport {
			cmd.Println("packages:")
			for _, e := range entries {
				cmd.Printf("  - %s\n", e.Name)
			}
			return nil
		}

		// Default tabular output
		for _, e := range entries {
			cmd.Printf("%s  %s  [%s]\n", e.Name, e.Version, e.Backend)
		}
		return nil
	},
}

// parseListLine splits a raw List() line by the first whitespace
// to extract the package name and version.
func parseListLine(line, backendName string) listEntry {
	parts := strings.SplitN(line, " ", 2)
	name := strings.TrimSpace(parts[0])
	version := ""
	if len(parts) > 1 {
		version = strings.TrimSpace(parts[1])
	}
	return listEntry{Name: name, Version: version, Backend: backendName}
}

func init() {
	listCmd.Flags().StringVarP(&backend, "backend", "b", "pacman", "Backend to use (pacman, nix, flatpak)")
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output in JSON format")
	listCmd.Flags().BoolVar(&listExport, "export", false, "Output in YAML format for agnostic.yaml")
	rootCmd.AddCommand(listCmd)
}

// resetListFlags resets package-level variables to their defaults
// so that tests don't leak state between each other.
func resetListFlags() {
	listJSON = false
	listExport = false
	backend = "pacman"
}
