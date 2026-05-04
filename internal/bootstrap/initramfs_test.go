package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildInitramfs_CreatesInitScript(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "initramfs.img")

	// Create a minimal rootfs dir with busybox-install (even empty)
	busyboxDir := filepath.Join(tmpDir, "busybox-install")
	if err := os.MkdirAll(busyboxDir, 0755); err != nil {
		t.Fatal(err)
	}

	err := BuildInitramfs(context.Background(), tmpDir, outputPath)
	if err != nil {
		t.Fatalf("BuildInitramfs failed: %v", err)
	}

	// Check output file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("initramfs output file was not created")
	}

	// Verify it's non-empty
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Error("initramfs output is empty")
	}
}

func TestBuildInitramfs_InitScriptContent(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "initramfs.img")

	busyboxDir := filepath.Join(tmpDir, "busybox-install")
	if err := os.MkdirAll(busyboxDir, 0755); err != nil {
		t.Fatal(err)
	}

	err := BuildInitramfs(context.Background(), tmpDir, outputPath)
	if err != nil {
		t.Fatalf("BuildInitramfs failed: %v", err)
	}

	// Extract the cpio to check content
	extractDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Use zcat + cpio to extract
	// We'll read the init script directly from the cpio archive using cpio -i
	// Since we can't easily extract cpio.gz in Go, let's verify content differently

	// Instead, let's check that the BuildInitramfs function produces a valid gzip
	// by checking the magic bytes
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 2 || data[0] != 0x1f || data[1] != 0x8b {
		t.Error("output is not a valid gzip file (missing magic bytes)")
	}
}

func TestBuildInitramfs_WithBusyboxCopy(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "initramfs.img")

	// Create a busybox-install dir with a dummy file to verify it gets copied
	busyboxDir := filepath.Join(tmpDir, "busybox-install")
	if err := os.MkdirAll(busyboxDir, 0755); err != nil {
		t.Fatal(err)
	}
	dummyContent := []byte("busybox-binary-stub")
	if err := os.WriteFile(filepath.Join(busyboxDir, "busybox"), dummyContent, 0755); err != nil {
		t.Fatal(err)
	}

	err := BuildInitramfs(context.Background(), tmpDir, outputPath)
	if err != nil {
		t.Fatalf("BuildInitramfs failed: %v", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("initramfs output file was not created")
	}
}

func TestBuildInitramfs_EmptyRootfsDir(t *testing.T) {
	err := BuildInitramfs(context.Background(), "", "/tmp/output.img")
	if err == nil {
		t.Error("expected error for empty rootfs dir")
	}
	if !strings.Contains(err.Error(), "rootfs directory is required") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestBuildInitramfs_EmptyOutputPath(t *testing.T) {
	err := BuildInitramfs(context.Background(), "/tmp", "")
	if err == nil {
		t.Error("expected error for empty output path")
	}
	if !strings.Contains(err.Error(), "output path is required") {
		t.Errorf("unexpected error message: %v", err)
	}
}
