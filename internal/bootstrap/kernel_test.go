package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKernelConfigFragment_HasRequiredOptions(t *testing.T) {
	required := []string{
		"CONFIG_BLK_DEV_INITRD=y",
		"CONFIG_DEVTMPFS=y",
		"CONFIG_DEVTMPFS_MOUNT=y",
		"CONFIG_SERIAL_8250_CONSOLE=y",
		"CONFIG_EXT4_FS=y",
	}
	for _, opt := range required {
		if !strings.Contains(kernelConfigFragment, opt) {
			t.Errorf("kernel config fragment missing required option: %s", opt)
		}
	}
}

func TestApplyKernelConfigMerge_UsesMergeConfigScript(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal .config with some default values
	baseConfig := `#
# Automatically generated file; DO NOT EDIT.
# Linux/x86_64 Kernel Configuration
#
CONFIG_INITRAMFS_SOURCE=""
# CONFIG_BLK_DEV_INITRD is not set
CONFIG_DEVTMPFS=n
# CONFIG_SERIAL_8250_CONSOLE is not set
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".config"), []byte(baseConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Create mock scripts/kconfig/merge_config.sh
	scriptsDir := filepath.Join(tmpDir, "scripts", "kconfig")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Mock merge_config.sh: appends fragment lines to .config (simulating what the real script does)
	mockScript := `#!/bin/bash
# Mock merge_config.sh for unit testing
# Usage: merge_config.sh -m -O <srctree> <base_config> <fragment>
# We read the fragment file and append its contents to .config
# ignoring -m and -O flags for simplicity
FRAGMENT="${@: -1}"
if [ -f "$FRAGMENT" ]; then
	cat "$FRAGMENT" >> .config
fi
`
	mockPath := filepath.Join(scriptsDir, "merge_config.sh")
	if err := os.WriteFile(mockPath, []byte(mockScript), 0755); err != nil {
		t.Fatal(err)
	}

	// Apply the config fragment
	if err := applyKernelConfigFragment(tmpDir, "x86_64"); err != nil {
		t.Fatalf("applyKernelConfigFragment failed: %v", err)
	}

	// Verify .config now has the expected options
	data, err := os.ReadFile(filepath.Join(tmpDir, ".config"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "CONFIG_BLK_DEV_INITRD=y") {
		t.Error("CONFIG_BLK_DEV_INITRD=y not found in .config after merge")
	}
	if !strings.Contains(content, "CONFIG_DEVTMPFS=y") {
		t.Error("CONFIG_DEVTMPFS=y not found in .config after merge")
	}
	if !strings.Contains(content, "CONFIG_SERIAL_8250_CONSOLE=y") {
		t.Error("CONFIG_SERIAL_8250_CONSOLE=y not found in .config after merge")
	}
}

func TestKernelArch(t *testing.T) {
	tests := []struct {
		arch          string
		wantKarch     string
		wantDefconfig string
		wantImagePath string
	}{
		{arch: "amd64", wantKarch: "x86_64", wantDefconfig: "x86_64_defconfig", wantImagePath: "arch/x86/boot/bzImage"},
		{arch: "arm64", wantKarch: "arm64", wantDefconfig: "defconfig", wantImagePath: "arch/arm64/boot/Image"},
	}
	for _, tt := range tests {
		t.Run(tt.arch, func(t *testing.T) {
			gotKarch, gotDefconfig, gotImagePath := kernelArch(tt.arch)
			if gotKarch != tt.wantKarch {
				t.Errorf("kernelArch(%q) karch = %q; want %q", tt.arch, gotKarch, tt.wantKarch)
			}
			if gotDefconfig != tt.wantDefconfig {
				t.Errorf("kernelArch(%q) defconfig = %q; want %q", tt.arch, gotDefconfig, tt.wantDefconfig)
			}
			if gotImagePath != tt.wantImagePath {
				t.Errorf("kernelArch(%q) imagePath = %q; want %q", tt.arch, gotImagePath, tt.wantImagePath)
			}
		})
	}
}

func TestAutoDetectArch_Custom(t *testing.T) {
	got := autoDetectArch(KernelConfig{Arch: "arm64"})
	if got != "arm64" {
		t.Errorf("autoDetectArch({Arch: 'arm64'}) = %q; want 'arm64'", got)
	}
}

func TestAutoDetectArch_Empty(t *testing.T) {
	got := autoDetectArch(KernelConfig{})
	// On a real system, this will match runtime.GOARCH
	if got == "" {
		t.Error("autoDetectArch({}) returned empty")
	}
}

func TestKernelArch_Default(t *testing.T) {
	// Default case (any unknown arch) should use x86_64
	gotKarch, gotDefconfig, gotImagePath := kernelArch("unknown")
	if gotKarch != "x86_64" {
		t.Errorf("kernelArch('unknown') karch = %q; want 'x86_64'", gotKarch)
	}
	if gotDefconfig != "x86_64_defconfig" {
		t.Errorf("kernelArch('unknown') defconfig = %q; want 'x86_64_defconfig'", gotDefconfig)
	}
	if gotImagePath != "arch/x86/boot/bzImage" {
		t.Errorf("kernelArch('unknown') imagePath = %q; want 'arch/x86/boot/bzImage'", gotImagePath)
	}
}

func TestApplyKernelConfigMerge_WriteFragmentError(t *testing.T) {
	tmpDir := t.TempDir()

	// Make kernel-config-minimal.config a directory so WriteFile fails
	fragmentPath := filepath.Join(tmpDir, "kernel-config-minimal.config")
	if err := os.MkdirAll(fragmentPath, 0755); err != nil {
		t.Fatal(err)
	}

	// .config not needed because write fragment fails first
	err := applyKernelConfigFragment(tmpDir, "x86_64")
	if err == nil {
		t.Fatal("expected error when fragment path is a directory")
	}
	if !strings.Contains(err.Error(), "write config fragment") {
		t.Errorf("expected 'write config fragment' error, got: %v", err)
	}
}

func TestApplyKernelConfigMerge_MissingMergeScript(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .config but DON'T create scripts/kconfig/merge_config.sh
	if err := os.WriteFile(filepath.Join(tmpDir, ".config"), []byte("CONFIG_EXPERT=y\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Should fail because merge_config.sh doesn't exist
	err := applyKernelConfigFragment(tmpDir, "x86_64")
	if err == nil {
		t.Fatal("expected error when merge_config.sh is missing")
	}
	if !strings.Contains(err.Error(), "merge config fragment") {
		t.Errorf("expected 'merge config fragment' error, got: %v", err)
	}
}

func TestApplyKernelConfigMerge_WritesFragmentFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .config
	if err := os.WriteFile(filepath.Join(tmpDir, ".config"), []byte("CONFIG_EXPERT=y\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create mock scripts/kconfig/merge_config.sh
	scriptsDir := filepath.Join(tmpDir, "scripts", "kconfig")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatal(err)
	}
	mockScript := `#!/bin/bash
# Mock merge_config.sh
FRAGMENT="${@: -1}"
if [ -f "$FRAGMENT" ]; then
	cat "$FRAGMENT" >> .config
fi
`
	if err := os.WriteFile(filepath.Join(scriptsDir, "merge_config.sh"), []byte(mockScript), 0755); err != nil {
		t.Fatal(err)
	}

	if err := applyKernelConfigFragment(tmpDir, "x86_64"); err != nil {
		t.Fatalf("applyKernelConfigFragment failed: %v", err)
	}

	// Verify fragment file was created
	fragmentPath := filepath.Join(tmpDir, "kernel-config-minimal.config")
	if _, err := os.Stat(fragmentPath); os.IsNotExist(err) {
		t.Error("kernel config fragment file was not created")
	}

	// Verify fragment content matches the constant
	data, err := os.ReadFile(fragmentPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != kernelConfigFragment {
		t.Error("fragment file content does not match kernelConfigFragment constant")
	}
}
