package agnostic

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install a new resource",
	Long:  `Install a new resource to the cluster.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("resource name is required")
		}

		resourceName := args[0]
		resourcePath := filepath.Join("/path/to/resources", resourceName)

		if err := os.MkdirAll(resourcePath, 0755); err != nil {
			return fmt.Errorf("failed to create resource directory: %w", err)
		}

		fmt.Printf("Resource %s installed successfully\n", resourceName)
		return nil
	},
}

func Execute() error {
	return installCmd.Execute()
}