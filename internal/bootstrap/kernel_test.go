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
