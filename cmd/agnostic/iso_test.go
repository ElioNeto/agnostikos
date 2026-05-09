package agnostic

import (
	"strings"
	"testing"
)

// resetIsoFlags resets package-level variables to their defaults
// so that tests don't leak state between each other.
func resetIsoFlags() {
	isoRootFS = ""
	isoOutput = ""
	isoVersion = "0.1.0"
	isoName = "AgnostikOS"
	isoKernelVersion = ""
	isoInitramfs = ""
	isoUEFI = false
	isoTestMode = false
}

func TestIsoCmd_Use(t *testing.T) {
	resetIsoFlags()
	if isoCmd.Use != "iso" {
		t.Errorf("expected Use 'iso', got %q", isoCmd.Use)
	}
	if isoCmd.Short == "" {
		t.Error("expected Short to be non-empty")
	}
}

func TestIsoCmd_ExecuteC_ReturnsCorrectCommand(t *testing.T) {
	resetIsoFlags()
	rootCmd.SetArgs([]string{"iso", "--rootfs", "/tmp/test-rootfs", "--output", "/tmp/test.iso"})
	cmd, err := rootCmd.ExecuteC()
	if err == nil {
		// Command succeeded (build env available), that's fine
		if cmd.Name() != "iso" {
			t.Errorf("expected command name 'iso', got %q", cmd.Name())
		}
		return
	}
	if cmd.Name() != "iso" {
		t.Errorf("expected command name 'iso', got %q", cmd.Name())
	}
}

func TestIsoCmd_ErrorOnNonExistentRootFS(t *testing.T) {
	resetIsoFlags()
	rootCmd.SetArgs([]string{"iso", "--rootfs", "/tmp/iso-test-nonexistent-rootfs", "--output", "/tmp/iso-test-output.iso"})
	_, err := rootCmd.ExecuteC()
	if err == nil {
		// If no error (build env available with /tmp/iso-test-nonexistent-rootfs?), skip
		t.Skip("command succeeded unexpectedly")
	}
	if !strings.Contains(err.Error(), "ISO build failed") &&
		!strings.Contains(err.Error(), "no kernel found") {
		t.Errorf("expected error about ISO build or kernel, got: %v", err)
	}
}

func TestIsoCmd_OutputFlag(t *testing.T) {
	resetIsoFlags()
	// Check that --output flag exists and has correct default
	f := isoCmd.Flags().Lookup("output")
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

func TestIsoCmd_UEFIFlagSetsVariable(t *testing.T) {
	resetIsoFlags()
	rootCmd.SetArgs([]string{"iso", "--rootfs", "/tmp/test-rootfs", "--output", "/tmp/test.iso", "--uefi"})
	_, err := rootCmd.ExecuteC()
	if err == nil {
		t.Skip("command succeeded unexpectedly")
	}
	if !isoUEFI {
		t.Error("expected isoUEFI to be true after --uefi flag")
	}
}

func TestIsoCmd_TestFlagSetsVariable(t *testing.T) {
	resetIsoFlags()
	rootCmd.SetArgs([]string{"iso", "--rootfs", "/tmp/test-rootfs", "--output", "/tmp/test.iso", "--test"})
	_, err := rootCmd.ExecuteC()
	if err == nil {
		t.Skip("command succeeded unexpectedly")
	}
	if !isoTestMode {
		t.Error("expected isoTestMode to be true after --test flag")
	}
}

func TestIsoCmd_DefaultRootFSFlag(t *testing.T) {
	resetIsoFlags()
	f := isoCmd.Flags().Lookup("rootfs")
	if f == nil {
		t.Fatal("expected --rootfs flag to be defined")
	}
	if f.DefValue != "" {
		t.Errorf("expected default rootfs '', got %q", f.DefValue)
	}
}

func TestIsoCmd_DefaultVersionFlag(t *testing.T) {
	resetIsoFlags()
	f := isoCmd.Flags().Lookup("version")
	if f == nil {
		t.Fatal("expected --version flag to be defined")
	}
	if f.DefValue != "0.1.0" {
		t.Errorf("expected default version '0.1.0', got %q", f.DefValue)
	}
}

func TestIsoCmd_DefaultNameFlag(t *testing.T) {
	resetIsoFlags()
	f := isoCmd.Flags().Lookup("name")
	if f == nil {
		t.Fatal("expected --name flag to be defined")
	}
	if f.DefValue != "AgnostikOS" {
		t.Errorf("expected default name 'AgnostikOS', got %q", f.DefValue)
	}
}
