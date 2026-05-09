package agnostic

import (
	"fmt"
	"os"

	"github.com/ElioNeto/agnostikos/internal/manager"
	"github.com/ElioNeto/agnostikos/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive TUI (Terminal User Interface)",
	Long: `Launch a Bubble Tea interactive terminal interface for browsing
and managing packages across all configured backends.

Screens:
  - Backend selection (pacman, nix, flatpak)
  - Package search with text input
  - Package detail with install/remove actions
  - List installed packages
  - Build AgnosticOS ISO

Use arrow keys to navigate, Enter to select, Esc to go back.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := manager.NewAgnosticManager()
		model := tui.InitialModel(mgr)

		p := tea.NewProgram(model, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
			return fmt.Errorf("TUI error: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
