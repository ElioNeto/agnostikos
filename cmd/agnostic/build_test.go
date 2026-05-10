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
	buildTarget = ""

	// Also reset bootstrap flags that the build command now shares
	bootstrapDevice = ""
	bootstrapEFIPartition = ""
	bootstrapKernelVer = "generic"
	bootstrapBusyboxVer = "1.36.1"
	bootstrapArch = ""
	bootstrapUEFI = false
	bootstrapJobs = ""
	bootstrapSkipToolchain = false
	bootstrapSkipKernel = false
	bootstrapSkipBusybox = false
	bootstrapSkipInitramfs = false
	bootstrapSkipGRUB = false
	bootstrapForce = false
	bootstrapDotfilesApply = false
	bootstrapDotfilesSource = ""
	bootstrapConfigsDir = ""
	bootstrapAutologinUser = ""
	bootstrapRecipe = ""
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

func TestBuildCmd_NoArgs(t *testing.T) {
	resetBuildFlags()
	// 'build' with no args should NOT produce an arg-count error.
	// It may fail later (BootstrapAll needs a real system), but cobra validation
	// should pass since we allow 0 or 1 args.
	rootCmd.SetArgs([]string{"build"})
	_, err := rootCmd.ExecuteC()
	if err == nil {
		return // success is OK (though unlikely without real system)
	}
	// If there IS an error, it should NOT be about argument count
	if strings.Contains(err.Error(), "accepts") ||
		strings.Contains(err.Error(), "requires exactly") {
		t.Errorf("expected no arg-count error for 'build' with no args, got: %v", err)
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

	// Pass --skip-* flags to avoid heavy toolchain download in unit tests.
	// The test validates that recipe parsing succeeds and the error comes
	// from BootstrapAll or GenerateISO execution.
	rootCmd.SetArgs([]string{
		"build", recipePath,
		"--target", targetDir,
		"--output", outputISO,
		"--skip-toolchain",
		"--skip-kernel",
		"--skip-busybox",
		"--skip-initramfs",
		"--skip-grub",
	})
	_, err := rootCmd.ExecuteC()
	if err == nil {
		t.Fatal("expected error (BootstrapAll or GenerateISO), got nil")
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

// TestBuildCmd_NotDeprecated verifies that build is no longer marked as deprecated.
func TestBuildCmd_NotDeprecated(t *testing.T) {
	resetBuildFlags()
	if buildCmd.Deprecated != "" {
		t.Errorf("expected buildCmd to NOT be deprecated, got Deprecated message: %q", buildCmd.Deprecated)
	}
}

func TestBuildCmd_TargetFlagDefault(t *testing.T) {
	resetBuildFlags()
	f := buildCmd.Flags().Lookup("target")
	if f == nil {
		t.Fatal("expected --target flag to be defined")
	}
	if f.DefValue != "" {
		t.Errorf("expected default target empty string (auto-detect), got %q", f.DefValue)
	}
	if f.Shorthand != "t" {
		t.Errorf("expected shorthand 't', got %q", f.Shorthand)
	}
}

func TestBuildCmd_OutputFlag(t *testing.T) {
	resetBuildFlags()
	f := buildCmd.Flags().Lookup("output")
	if f == nil {
		t.Fatal("expected --output flag to be defined")
	}
	if f.DefValue != "" {
		t.Errorf("expected default output '', got %q", f.DefValue)
	}
	if f.Shorthand != "o" {
		t.Errorf("expected shorthand 'o', got %q", f.Shorthand)
	}
}

// Bootstrap flag tests — verify the unified build command exposes all pipeline flags.

func TestBuildCmd_KernelVersionFlag(t *testing.T) {
	resetBuildFlags()
	f := buildCmd.Flags().Lookup("kernel-version")
	if f == nil {
		t.Fatal("expected --kernel-version flag to be defined")
	}
	if f.DefValue != "generic" {
		t.Errorf("expected default 'generic', got %q", f.DefValue)
	}
}

func TestBuildCmd_BusyboxVersionFlag(t *testing.T) {
	resetBuildFlags()
	f := buildCmd.Flags().Lookup("busybox-version")
	if f == nil {
		t.Fatal("expected --busybox-version flag to be defined")
	}
	if f.DefValue != "1.36.1" {
		t.Errorf("expected default '1.36.1', got %q", f.DefValue)
	}
}

func TestBuildCmd_ArchFlag(t *testing.T) {
	resetBuildFlags()
	f := buildCmd.Flags().Lookup("arch")
	if f == nil {
		t.Fatal("expected --arch flag to be defined")
	}
	if f.DefValue != "" {
		t.Errorf("expected default '', got %q", f.DefValue)
	}
}

func TestBuildCmd_UEFIFlag(t *testing.T) {
	resetBuildFlags()
	f := buildCmd.Flags().Lookup("uefi")
	if f == nil {
		t.Fatal("expected --uefi flag to be defined")
	}
	if f.DefValue != "false" {
		t.Errorf("expected default 'false', got %q", f.DefValue)
	}
}

func TestBuildCmd_SkipToolchainFlag(t *testing.T) {
	resetBuildFlags()
	f := buildCmd.Flags().Lookup("skip-toolchain")
	if f == nil {
		t.Fatal("expected --skip-toolchain flag to be defined")
	}
	if f.DefValue != "false" {
		t.Errorf("expected default 'false', got %q", f.DefValue)
	}
}

func TestBuildCmd_SkipKernelFlag(t *testing.T) {
	resetBuildFlags()
	f := buildCmd.Flags().Lookup("skip-kernel")
	if f == nil {
		t.Fatal("expected --skip-kernel flag to be defined")
	}
}

func TestBuildCmd_SkipBusyboxFlag(t *testing.T) {
	resetBuildFlags()
	f := buildCmd.Flags().Lookup("skip-busybox")
	if f == nil {
		t.Fatal("expected --skip-busybox flag to be defined")
	}
}

func TestBuildCmd_SkipInitramfsFlag(t *testing.T) {
	resetBuildFlags()
	f := buildCmd.Flags().Lookup("skip-initramfs")
	if f == nil {
		t.Fatal("expected --skip-initramfs flag to be defined")
	}
}

func TestBuildCmd_SkipGRUBFlag(t *testing.T) {
	resetBuildFlags()
	f := buildCmd.Flags().Lookup("skip-grub")
	if f == nil {
		t.Fatal("expected --skip-grub flag to be defined")
	}
}

func TestBuildCmd_ForceFlag(t *testing.T) {
	resetBuildFlags()
	f := buildCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("expected --force flag to be defined")
	}
	if f.DefValue != "false" {
		t.Errorf("expected default 'false', got %q", f.DefValue)
	}
}

func TestBuildCmd_JobsFlag(t *testing.T) {
	resetBuildFlags()
	f := buildCmd.Flags().Lookup("jobs")
	if f == nil {
		t.Fatal("expected --jobs flag to be defined")
	}
}

func TestBuildCmd_RecipeFlag(t *testing.T) {
	resetBuildFlags()
	f := buildCmd.Flags().Lookup("recipe")
	if f == nil {
		t.Fatal("expected --recipe flag to be defined")
	}
}

func TestBuildCmd_RecipeFlagInvalidPath(t *testing.T) {
	resetBuildFlags()
	rootCmd.SetArgs([]string{"build", "--recipe", "/tmp/nonexistent-recipe-build-12345.yaml"})
	_, err := rootCmd.ExecuteC()
	if err == nil {
		t.Fatal("expected error for invalid recipe path, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read recipe") {
		t.Errorf("expected error about reading recipe, got: %v", err)
	}
}

func TestBuildCmd_DeviceFlag(t *testing.T) {
	resetBuildFlags()
	f := buildCmd.Flags().Lookup("device")
	if f == nil {
		t.Fatal("expected --device flag to be defined")
	}
}

func TestBuildCmd_EFIPartitionFlag(t *testing.T) {
	resetBuildFlags()
	f := buildCmd.Flags().Lookup("efi-partition")
	if f == nil {
		t.Fatal("expected --efi-partition flag to be defined")
	}
}
