package agnostic

import (
	"strings"
	"testing"
)

// resetBootstrapFlags resets package-level variables to their defaults
// so that tests don't leak state between each other.
func resetBootstrapFlags() {
	bootstrapTarget = ""
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

func TestBootstrapCmd_Use(t *testing.T) {
	resetBootstrapFlags()
	if bootstrapCmd.Use != "bootstrap" {
		t.Errorf("expected Use 'bootstrap', got %q", bootstrapCmd.Use)
	}
	if bootstrapCmd.Short == "" {
		t.Error("expected Short to be non-empty")
	}
}

func TestBootstrapCmd_SkipToolchainFlag(t *testing.T) {
	resetBootstrapFlags()
	f := bootstrapCmd.Flags().Lookup("skip-toolchain")
	if f == nil {
		t.Fatal("expected --skip-toolchain flag to be defined")
	}
	if f.DefValue != "false" {
		t.Errorf("expected default 'false', got %q", f.DefValue)
	}
}

func TestBootstrapCmd_JobsFlag(t *testing.T) {
	resetBootstrapFlags()
	f := bootstrapCmd.Flags().Lookup("jobs")
	if f == nil {
		t.Fatal("expected --jobs flag to be defined")
	}
	if f.DefValue != "" {
		t.Errorf("expected default '', got %q", f.DefValue)
	}
}

func TestBootstrapCmd_TargetFlag(t *testing.T) {
	resetBootstrapFlags()
	f := bootstrapCmd.Flags().Lookup("target")
	if f == nil {
		t.Fatal("expected --target flag to be defined")
	}
	if f.DefValue != "" {
		t.Errorf("expected default '', got %q", f.DefValue)
	}
}

func TestBootstrapCmd_TargetFlagShort(t *testing.T) {
	resetBootstrapFlags()
	f := bootstrapCmd.Flags().Lookup("target")
	if f == nil {
		t.Fatal("expected --target flag to be defined")
	}
	if f.Shorthand != "t" {
		t.Errorf("expected shorthand 't', got %q", f.Shorthand)
	}
}

func TestBootstrapCmd_RecipeFlag(t *testing.T) {
	resetBootstrapFlags()
	f := bootstrapCmd.Flags().Lookup("recipe")
	if f == nil {
		t.Fatal("expected --recipe flag to be defined")
	}
	if f.DefValue != "" {
		t.Errorf("expected default '', got %q", f.DefValue)
	}
}

func TestBootstrapCmd_RecipeFlagLocal(t *testing.T) {
	resetBootstrapFlags()
	// --recipe should be a local flag, not persistent
	if bootstrapCmd.PersistentFlags().Lookup("recipe") != nil {
		t.Error("expected --recipe to be a local flag, not persistent")
	}
}

func TestBootstrapCmd_RecipeFlagInvalidPath(t *testing.T) {
	resetBootstrapFlags()
	// This error occurs before BootstrapAll is called (just os.ReadFile)
	rootCmd.SetArgs([]string{"bootstrap", "--recipe", "/tmp/nonexistent-recipe-12345.yaml"})
	_, err := rootCmd.ExecuteC()
	if err == nil {
		t.Fatal("expected error for invalid recipe path, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read recipe") {
		t.Errorf("expected error about reading recipe, got: %v", err)
	}
}

func TestBootstrapCmd_SkipToolchainFlagCanBeParsed(t *testing.T) {
	resetBootstrapFlags()
	// Just verify the --skip-toolchain flag can be parsed without reaching BootstrapAll.
	// This will fail at ReadFile (recipe does not exist) before BootstrapAll is called.
	rootCmd.SetArgs([]string{"bootstrap", "--skip-toolchain", "--recipe", "/tmp/nonexistent-recipe-12345.yaml"})
	_, err := rootCmd.ExecuteC()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Error should be about recipe, not about the flag
	if strings.Contains(err.Error(), "flag") {
		t.Errorf("flag parsing should succeed, got: %v", err)
	}
}
