package agnostic

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ElioNeto/agnostikos/internal/bootstrap"
	"github.com/spf13/cobra"
)

func TestBuildCmd(t *testing.T) {
	tests := []struct {
		name        string
		recipeFile  string
		output      string
		expected    string
	}{
		{
			name:       "default output",
			recipeFile: filepath.Join(os.TempDir(), "test_recipe.yaml"),
			output:     "",
			expected:   "🏗️  Building test v1.0 (amd64)\n✅ Build complete: custom.iso\n",
		},
		{
			name:       "custom output",
			recipeFile: filepath.Join(os.TempDir(), "test_recipe.yaml"),
			output:     "custom_output.iso",
			expected:   "🏗️  Building test v1.0 (amd64)\n✅ Build complete: custom_output.iso\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.WriteFile(tt.recipeFile, []byte(`name: test
version: 1.0
arch: amd64
packages:
- package1
- package2`), 0644); err != nil {
				t.Fatalf("Failed to create temporary recipe file: %v", err)
			}
			defer os.Remove(tt.recipeFile)

			oldOut := os.Stdout
			r := bytes.NewBuffer(nil)
			os.Stdout = r

			if tt.output != "" {
				buildCmd.Flags().String("output", tt.output, "")
			}
			buildCmd.RunE(buildCmd, []string{tt.recipeFile})

			os.Stdout = oldOut

			if r.String() != tt.expected {
				t.Errorf("Unexpected output:\n%s", r.String())
			}
		})
	}
}

func TestBuildCmdWithKernel(t *testing.T) {
	tests := []struct {
		name        string
		recipeFile  string
		expected    string
	}{
		{
			name:       "kernel version specified",
			recipeFile: filepath.Join(os.TempDir(), "test_recipe.yaml"),
			expected:   "🏗️  Building test v1.0 (amd64)\n✅ Build complete: custom.iso\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.WriteFile(tt.recipeFile, []byte(`name: test
version: 1.0
arch: amd64
packages:
- package1
- package2
build:
  kernel_version: 5.10`), 0644); err != nil {
				t.Fatalf("Failed to create temporary recipe file: %v", err)
			}
			defer os.Remove(tt.recipeFile)

			oldOut := os.Stdout
			r := bytes.NewBuffer(nil)
			os.Stdout = r

			buildCmd.RunE(buildCmd, []string{tt.recipeFile})

			os.Stdout = oldOut

			if r.String() != tt.expected {
				t.Errorf("Unexpected output:\n%s", r.String())
			}
		})
	}
}