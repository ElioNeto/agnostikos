package agnostic

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// resetBuildFlags resets package-level variables to their defaults
// so that tests don't leak state between each other.
func resetBuildFlags() {
	buildOutput = ""
	buildTarget = "/mnt/lfs"
}

func TestBuildCmd_Use(t *testing.T) {
	resetBuildFlags()
	if buildCmd.Use != "build [recipe.yaml]" {
		t.Errorf("expected Use 'build [recipe.yaml]', got %q", buildCmd.Use)
	}
	if buildCmd.Short == "" {
		t.Error("expected Short to be non-empty")
	}
}

func TestBuildCmd_MissingArg(t *testing.T) {
	resetBuildFlags()
	rootCmd.SetArgs([]string{"build"})
	_, err := rootCmd.ExecuteC()
	if err == nil {
		t.Fatal("expected error for missing arg, got nil")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg") &&
		!strings.Contains(err.Error(), "requires exactly 1 arg") &&
		!strings.Contains(err.Error(), "requires at least 1 arg") {
		t.Errorf("expected error about arg count, got: %v", err)
	}
}

func TestBuildCmd_InvalidRecipePath(t *testing.T) {
	resetBuildFlags()
	rootCmd.SetArgs([]string{"build", "/tmp/nonexistent-recipe-12345.yaml"})
	_, err := rootCmd.ExecuteC()
	if err == nil {
		t.Fatal("expected error for invalid path, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read recipe") {
		t.Errorf("expected error about reading recipe, got: %v", err)
	}
}

func TestBuildCmd_MockRecipe(t *testing.T) {
	resetBuildFlags()

	// Create a temporary recipe file
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "target")
	outputISO := filepath.Join(tmpDir, "output.iso")
	recipePath := filepath.Join(tmpDir, "recipe.yaml")

	recipeContent := []byte(`name: TestOS
version: 1.0.0
arch: amd64
description: "Test recipe"
packages: []
build:
  output_iso: "` + outputISO + `"
`)
	if err := os.WriteFile(recipePath, recipeContent, 0644); err != nil {
		t.Fatalf("failed to write recipe: %v", err)
	}

	rootCmd.SetArgs([]string{"build", recipePath, "--target", targetDir, "--output", outputISO})
	_, err := rootCmd.ExecuteC()
	if err == nil {
		t.Fatal("expected error (CreateRootFS or GenerateISO), got nil")
	}

	// The error should NOT be about recipe parsing — it should be about
	// rootfs creation or ISO generation (since the recipe was valid)
	if strings.Contains(err.Error(), "failed to read recipe") {
		t.Errorf("recipe should have been parsed successfully, got: %v", err)
	}
	if strings.Contains(err.Error(), "failed to parse recipe") {
		t.Errorf("recipe should have been parsed successfully, got: %v", err)
	}
}

func TestBuildCmd_DeprecatedAnnotation(t *testing.T) {
	resetBuildFlags()
	if buildCmd.Deprecated == "" {
		t.Error("expected buildCmd to be marked as deprecated with a message")
	}
}

func TestBuildCmd_TargetFlagDefault(t *testing.T) {
	resetBuildFlags()
	f := buildCmd.Flags().Lookup("target")
	if f == nil {
		t.Fatal("expected --target flag to be defined")
	}
	if f.DefValue != "/mnt/lfs" {
		t.Errorf("expected default target '/mnt/lfs', got %q", f.DefValue)
	}
	if f.Shorthand != "t" {
		t.Errorf("expected shorthand 't', got %q", f.Shorthand)
	}
}
