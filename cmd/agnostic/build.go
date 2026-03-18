package agnostic

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ElioNeto/agnostikos/internal/bootstrap"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func TestBuildCmd(t *testing.T) {
	// Create a temporary recipe file
	tempRecipe := filepath.Join(os.TempDir(), "test_recipe.yaml")
	recipeContent := `name: test
version: 1.0
arch: amd64
packages:
- package1
- package2
build:
  kernel_version: 5.10
  output_iso: custom.iso
  uefi: true`
	if err := os.WriteFile(tempRecipe, []byte(recipeContent), 0644); err != nil {
		t.Fatalf("Failed to create temporary recipe file: %v", err)
	}
	defer os.Remove(tempRecipe)

	// Set up the environment for the test
	oldOut := os.Stdout
	r := bytes.NewBuffer(nil)
	os.Stdout = r

	// Execute the build command
	buildCmd.RunE(buildCmd, []string{tempRecipe})

	// Restore the original output
	os.Stdout = oldOut

	// Check if the expected output is present
	expectedOutput := "🏗️  Building test v1.0 (amd64)\n✅ Build complete: custom.iso\n"
	if r.String() != expectedOutput {
		t.Errorf("Unexpected output:\n%s", r.String())
	}
}

func TestBuildCmdWithOverride(t *testing.T) {
	// Create a temporary recipe file
	tempRecipe := filepath.Join(os.TempDir(), "test_recipe.yaml")
	recipeContent := `name: test
version: 1.0
arch: amd64
packages:
- package1
- package2`
	if err := os.WriteFile(tempRecipe, []byte(recipeContent), 0644); err != nil {
		t.Fatalf("Failed to create temporary recipe file: %v", err)
	}
	defer os.Remove(tempRecipe)

	// Set up the environment for the test
	oldOut := os.Stdout
	r := bytes.NewBuffer(nil)
	os.Stdout = r

	// Execute the build command with an output override
	buildCmd.Flags().String("output", "custom_output.iso", "")
	buildCmd.RunE(buildCmd, []string{tempRecipe})

	// Restore the original output
	os.Stdout = oldOut

	// Check if the expected output is present
	expectedOutput := "🏗️  Building test v1.0 (amd64)\n✅ Build complete: custom_output.iso\n"
	if r.String() != expectedOutput {
		t.Errorf("Unexpected output:\n%s", r.String())
	}
}

func TestBuildCmdWithKernel(t *testing.T) {
	// Create a temporary recipe file
	tempRecipe := filepath.Join(os.TempDir(), "test_recipe.yaml")
	recipeContent := `name: test
version: 1.0
arch: amd64
packages:
- package1
- package2
build:
  kernel_version: 5.10`
	if err := os.WriteFile(tempRecipe, []byte(recipeContent), 0644); err != nil {
		t.Fatalf("Failed to create temporary recipe file: %v", err)
	}
	defer os.Remove(tempRecipe)

	// Set up the environment for the test
	oldOut := os.Stdout
	r := bytes.NewBuffer(nil)
	os.Stdout = r

	// Execute the build command with a kernel version specified
	buildCmd.RunE(buildCmd, []string{tempRecipe})

	// Restore the original output
	os.Stdout = oldOut

	// Check if the expected output is present
	expectedOutput := "🏗️  Building test v1.0 (amd64)\n✅ Build complete: custom.iso\n"
	if r.String() != expectedOutput {
		t.Errorf("Unexpected output:\n%s", r.String())
	}
}