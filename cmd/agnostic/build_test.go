package agnostic_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/ElioNeto/agnostikos/internal/bootstrap"
	"github.com/spf13/cobra"
)

// TestBuildCmd tests the build command with various scenarios.
func TestBuildCmd(t *testing.T) {
	testCases := []struct {
		name          string
		recipeFile    string
		outputPath    string
		targetDir     string
		expectedError bool
	}{
		{
			name:         "Valid recipe, default output",
			recipeFile:   "valid_recipe.yaml",
			outputPath:   "",
			targetDir:    "/mnt/lfs",
			expectedError: false,
		},
		{
			name:          "Valid recipe, custom output",
			recipeFile:   "valid_recipe.yaml",
			outputPath:   "custom.iso",
			targetDir:    "/mnt/lfs",
			expectedError: false,
		},
		{
			name:          "Invalid recipe file",
			recipeFile:   "nonexistent_recipe.yaml",
			outputPath:   "",
			targetDir:    "/mnt/lfs",
			expectedError: true,
		},
		{
			name:         "Empty recipe content",
			recipeFile:   "empty_recipe.yaml",
			outputPath:   "",
			targetDir:    "/mnt/lfs",
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a temporary recipe file for testing
			tmpRecipeFile, err := ioutil.TempFile("", "recipe.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp recipe file: %v", err)
			}
			defer os.Remove(tmpRecipeFile.Name())

			// Write the test recipe content to the temporary file
			content := []byte{}
			switch tc.recipeFile {
			case "valid_recipe.yaml":
				content = []byte(`
name: AgnosticOS
version: 1.0
arch: amd64
description: A minimalist operating system
packages:
- coreutils
build:
  kernel_version: ""
  output_iso: ""
  uefi: false
`)
			case "nonexistent_recipe.yaml":
				content = []byte("Invalid content")
			case "empty_recipe.yaml":
				content = []byte{}
			}

			if _, err := tmpRecipeFile.Write(content); err != nil {
				t.Fatalf("Failed to write recipe content: %v", err)
			}
			tmpRecipeFile.Close()

			// Set the global build output and target variables
			buildOutput = tc.outputPath
			buildTarget = tc.targetDir

			// Run the build command
			cmd := &cobra.Command{}
			cmd.SetArgs([]string{tmpRecipeFile.Name()})
			err = buildCmd.RunE(cmd, []string{})

			if tc.expectedError && err == nil {
				t.Fatalf("Expected error but got nil")
			} else if !tc.expectedError && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
		})
	}
}