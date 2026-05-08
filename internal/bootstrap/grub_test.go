package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// biosCfg retorna um GRUBConfig BIOS seguro para testes unitários.
// Strict: false garante que falha do grub-install (sem root/device real) vira warn.
func biosCfg(rootfsDir string) GRUBConfig {
	return GRUBConfig{RootfsDir: rootfsDir, Device: "/dev/sda", UEFI: false, Strict: false}
}

func TestInstallGRUB_CreatesGrubCfg(t *testing.T) {
	tmpDir := t.TempDir()

	if err := InstallGRUB(context.Background(), biosCfg(tmpDir)); err != nil {
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
	if !strings.Contains(content, "linux /boot/vmlinuz root=/dev/ram0 console=ttyS0,115200") {
		t.Error("grub.cfg missing linux line with serial console")
	}
	if !strings.Contains(content, "initrd /boot/initramfs.img") {
		t.Error("grub.cfg missing initrd line")
	}
}

func TestInstallGRUB_BIOSMode(t *testing.T) {
	tmpDir := t.TempDir()

	if err := InstallGRUB(context.Background(), biosCfg(tmpDir)); err != nil {
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

	err := InstallGRUB(context.Background(), GRUBConfig{
		RootfsDir: tmpDir,
		UEFI:      true,
	})
	if err != nil {
		t.Fatalf("InstallGRUB (UEFI) failed: %v", err)
	}

	efiDir := filepath.Join(tmpDir, "boot", "efi", "EFI", "BOOT")
	if _, err := os.Stat(efiDir); os.IsNotExist(err) {
		t.Error("EFI directory should exist in UEFI mode")
	}

	efiPath := filepath.Join(efiDir, "BOOTx64.EFI")
	if _, err := os.Stat(efiPath); os.IsNotExist(err) {
		t.Error("BOOTx64.EFI should exist in UEFI mode")
	}

	grubCfgPath := filepath.Join(tmpDir, "boot", "grub", "grub.cfg")
	if _, err := os.Stat(grubCfgPath); os.IsNotExist(err) {
		t.Error("grub.cfg should exist in UEFI mode")
	}
}

func TestInstallGRUB_EmptyRootfs(t *testing.T) {
	err := InstallGRUB(context.Background(), GRUBConfig{
		RootfsDir: "",
		Device:    "/dev/sda",
		UEFI:      false,
	})
	if err == nil {
		t.Error("expected error for empty rootfs dir")
	}
	if !strings.Contains(err.Error(), "rootfs directory is required") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestInstallGRUB_MissingDeviceBIOS(t *testing.T) {
	tmpDir := t.TempDir()

	err := InstallGRUB(context.Background(), GRUBConfig{
		RootfsDir: tmpDir,
		Device:    "",
		UEFI:      false,
	})
	if err == nil {
		t.Error("expected error for missing device in BIOS mode")
	}
	if !strings.Contains(err.Error(), "device is required") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestInstallGRUB_StrictBIOSFails documenta o comportamento de Strict: true.
// O dispositivo fake /dev/nonexistent garante falha controlada do grub-install.
func TestInstallGRUB_StrictBIOSFails(t *testing.T) {
	if !hasGrubInstall() {
		t.Skip("grub-install not available")
	}
	tmpDir := t.TempDir()

	err := InstallGRUB(context.Background(), GRUBConfig{
		RootfsDir: tmpDir,
		Device:    "/dev/nonexistent",
		UEFI:      false,
		Strict:    true,
	})
	if err == nil {
		t.Error("expected error for Strict BIOS install with nonexistent device")
	}
}

func TestInstallGRUB_GrubCfgContent(t *testing.T) {
	tmpDir := t.TempDir()

	err := InstallGRUB(context.Background(), GRUBConfig{
		RootfsDir: tmpDir,
		UEFI:      true,
	})
	if err != nil {
		t.Fatalf("InstallGRUB failed: %v", err)
	}

	grubCfgPath := filepath.Join(tmpDir, "boot", "grub", "grub.cfg")
	data, err := os.ReadFile(grubCfgPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	expectedParts := []string{
		"set timeout=5",
		"set default=0",
		"menuentry \"Agnostikos Linux\"",
		"linux /boot/vmlinuz root=/dev/ram0 console=ttyS0,115200",
		"initrd /boot/initramfs.img",
	}
	for _, part := range expectedParts {
		if !strings.Contains(content, part) {
			t.Errorf("grub.cfg missing expected content: %s", part)
		}
	}
}
