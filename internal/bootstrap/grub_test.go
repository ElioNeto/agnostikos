package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallGRUB_CreatesGrubCfg(t *testing.T) {
	tmpDir := t.TempDir()

	err := InstallGRUB(context.Background(), tmpDir, false)
	if err != nil {
		t.Fatalf("InstallGRUB failed: %v", err)
	}

	grubCfgPath := filepath.Join(tmpDir, "boot", "grub", "grub.cfg")
	if _, err := os.Stat(grubCfgPath); os.IsNotExist(err) {
		t.Fatal("grub.cfg was not created")
	}

	data, err := os.ReadFile(grubCfgPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "set timeout=5") {
		t.Error("grub.cfg missing timeout setting")
	}
	if !strings.Contains(content, "set default=0") {
		t.Error("grub.cfg missing default setting")
	}
	if !strings.Contains(content, `menuentry "Agnostikos Linux"`) {
		t.Error("grub.cfg missing menuentry")
	}
	if !strings.Contains(content, "linux /boot/vmlinuz root=/dev/ram0 quiet") {
		t.Error("grub.cfg missing linux line")
	}
	if !strings.Contains(content, "initrd /boot/initramfs.img") {
		t.Error("grub.cfg missing initrd line")
	}
}

func TestInstallGRUB_BIOSMode(t *testing.T) {
	tmpDir := t.TempDir()

	err := InstallGRUB(context.Background(), tmpDir, false)
	if err != nil {
		t.Fatalf("InstallGRUB (BIOS) failed: %v", err)
	}

	// BIOS mode should NOT create EFI directory
	efiDir := filepath.Join(tmpDir, "boot", "efi")
	if _, err := os.Stat(efiDir); err == nil {
		t.Error("EFI directory should not exist in BIOS mode")
	}

	// But should create grub.cfg
	grubCfgPath := filepath.Join(tmpDir, "boot", "grub", "grub.cfg")
	if _, err := os.Stat(grubCfgPath); os.IsNotExist(err) {
		t.Error("grub.cfg should exist in BIOS mode")
	}
}

func TestInstallGRUB_UEFIMode(t *testing.T) {
	tmpDir := t.TempDir()

	err := InstallGRUB(context.Background(), tmpDir, true)
	if err != nil {
		t.Fatalf("InstallGRUB (UEFI) failed: %v", err)
	}

	// UEFI mode should create EFI directory
	efiDir := filepath.Join(tmpDir, "boot", "efi", "EFI", "BOOT")
	if _, err := os.Stat(efiDir); os.IsNotExist(err) {
		t.Error("EFI directory should exist in UEFI mode")
	}

	// Should create BOOTx64.EFI (placeholder)
	efiPath := filepath.Join(efiDir, "BOOTx64.EFI")
	if _, err := os.Stat(efiPath); os.IsNotExist(err) {
		t.Error("BOOTx64.EFI should exist in UEFI mode")
	}

	// Should still create grub.cfg
	grubCfgPath := filepath.Join(tmpDir, "boot", "grub", "grub.cfg")
	if _, err := os.Stat(grubCfgPath); os.IsNotExist(err) {
		t.Error("grub.cfg should exist in UEFI mode")
	}
}

func TestInstallGRUB_EmptyRootfs(t *testing.T) {
	err := InstallGRUB(context.Background(), "", false)
	if err == nil {
		t.Error("expected error for empty rootfs dir")
	}
	if !strings.Contains(err.Error(), "rootfs directory is required") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestInstallGRUB_GrubCfgContent(t *testing.T) {
	tmpDir := t.TempDir()

	err := InstallGRUB(context.Background(), tmpDir, true)
	if err != nil {
		t.Fatalf("InstallGRUB failed: %v", err)
	}

	grubCfgPath := filepath.Join(tmpDir, "boot", "grub", "grub.cfg")
	data, err := os.ReadFile(grubCfgPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)

	// Verify exact expected content
	expectedParts := []string{
		"set timeout=5",
		"set default=0",
		"menuentry \"Agnostikos Linux\"",
		"linux /boot/vmlinuz root=/dev/ram0 quiet",
		"initrd /boot/initramfs.img",
	}
	for _, part := range expectedParts {
		if !strings.Contains(content, part) {
			t.Errorf("grub.cfg missing expected content: %s", part)
		}
	}
}
