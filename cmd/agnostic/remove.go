package agnostic

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	removeDryRun bool
	removeYes    bool
)

// stdinReader allows tests to mock stdin for the interactive confirmation prompt.
var stdinReader *bufio.Reader

var removeCmd = &cobra.Command{
	Use:     "remove [package]",
	Aliases: []string{"rm", "uninstall"},
	Short:   "Remove a package via the specified backend",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := newManager()
		svc, ok := mgr.Backends[backend]
		if !ok {
			return fmt.Errorf("backend '%s' not found — available: pacman, nix, flatpak", backend)
		}

		pkg := args[0]

		if removeDryRun {
			cmd.Printf("Would remove '%s' via %s (dry-run)\n", pkg, backend)
			return nil
		}

		if !removeYes {
			reader := stdinReader
			if reader == nil {
				reader = bufio.NewReader(os.Stdin)
			}
			cmd.Printf("Remove '%s' via %s? [y/N] ", pkg, backend)
			input, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("reading confirmation: %w", err)
			}
			input = strings.TrimSpace(strings.ToLower(input))
			if input != "y" && input != "yes" {
				cmd.Println("Aborted.")
				return nil
			}
		}

		cmd.Printf("🗑️  Removing '%s' via %s...\n", pkg, backend)
		if err := svc.Remove(pkg); err != nil {
			return fmt.Errorf("removal failed: %w", err)
		}
		cmd.Printf("✅ '%s' removed\n", pkg)
		return nil
	},
}

func init() {
	removeCmd.Flags().BoolVarP(&removeYes, "yes", "y", false, "Skip confirmation prompt")
	removeCmd.Flags().BoolVar(&removeDryRun, "dry-run", false, "Print what would be removed without actually removing")
	removeCmd.Flags().StringVarP(&backend, "backend", "b", "pacman", "Backend to use (pacman, nix, flatpak)")
	rootCmd.AddCommand(removeCmd)
}
