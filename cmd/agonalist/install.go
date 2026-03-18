package agnostic

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install a service",
	Long: `Install a service on the target system.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Implementation of the install command
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}