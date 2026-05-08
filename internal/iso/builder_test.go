package iso

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// mockExecutor implementa Executor para testes.
// Se createOutput for true, ele cria o arquivo passado após o flag -o.
type mockExecutor struct {
	Output       []byte
	Err          error
	createOutput bool
}

func (m *mockExecutor) RunContext(ctx context.Context, name string, args ...string) ([]byte, error) {
	if m.createOutput {
		for i, arg := range args {
			if arg == "-o" && i+1 < len(args) {
				// Cria o arquivo de saída para que o checksum funcione
				_ = os.WriteFile(args[i+1], m.Output, 0644)
				break
			}
		}
	}
	return m.Output, m.Err
}

// setupFakeTool cria um diretório com um script xorrisofs fake e o adiciona ao PATH.
// O script fake grava o conteúdo em outputPath se fornecida como "-o <output>".
// Retorna uma função de cleanup.
func setupFakeTool(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	// Criar script fake xorrisofs que cria o arquivo de saída
	fakeTool := filepath.Join(tmpDir, "xorrisofs")
	content := `#!/bin/sh
# fake xorrisofs: look for -o flag and create output file
for i in "$@"; do
  case "$i" in
    -o) found=1 ;;
    *)  if [ -n "$found" ]; then
          touch "$i"
          echo "fake ISO created" > "$i"
          exit 0
        fi ;;
  esac
done
echo "fake ISO created"
`
	if err := os.WriteFile(fakeTool, []byte(content), 0755); err != nil {
		t.Fatalf("failed to create fake xorrisofs: %v", err)
	}

	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+oldPath)

	return tmpDir
}

func TestNewISOBuilder(t *testing.T) {
	b := NewISOBuilder("/tmp/rootfs", "/tmp/output.iso")
	if b == nil {
		t.Fatal("expected non-nil builder")
	}
	if b.rootfsPath != "/tmp/rootfs" {
		t.Errorf("expected rootfsPath /tmp/rootfs, got %s", b.rootfsPath)
	}
	if b.outputPath != "/tmp/output.iso" {
		t.Errorf("expected outputPath /tmp/output.iso, got %s", b.outputPath)
	}
}

func TestDetectTool_NoTool(t *testing.T) {
	b := &ISOBuilder{executor: &mockExecutor{}}
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", "")
	defer func() { _ = os.Setenv("PATH", oldPath) }()

	tool, err := b.detectTool()
	if err == nil {
		t.Error("expected error when no tool is in PATH")
	}
	if tool != "" {
		t.Errorf("expected empty tool, got %s", tool)
	}
}

func TestDetectTool_FindsXorrisofs(t *testing.T) {
	setupFakeTool(t)
	b := &ISOBuilder{executor: &mockExecutor{}}
	tool, err := b.detectTool()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if tool == "" {
		t.Fatal("expected tool path, got empty")
	}
	if filepath.Base(tool) != "xorrisofs" {
		t.Errorf("expected xorrisofs, got %s", filepath.Base(tool))
	}
}

func TestISOBuilder_Build_EmptyRootFS(t *testing.T) {
	b := &ISOBuilder{executor: &mockExecutor{}}
	err := b.Build(context.Background(), "", "/tmp/output.iso")
	if err == nil {
		t.Error("expected error for empty rootfs path")
	}
}

func TestISOBuilder_Build_EmptyOutput(t *testing.T) {
	b := &ISOBuilder{executor: &mockExecutor{}}
	err := b.Build(context.Background(), "/tmp/rootfs", "")
	if err == nil {
		t.Error("expected error for empty output path")
	}
}

func TestISOBuilder_Build_ToolNotFound(t *testing.T) {
	b := &ISOBuilder{executor: &mockExecutor{}}
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", "")
	defer func() { _ = os.Setenv("PATH", oldPath) }()

	err := b.Build(context.Background(), "/tmp/rootfs", "/tmp/output.iso")
	if err == nil {
		t.Error("expected error when ISO tool is not found")
	}
}

func TestISOBuilder_Build_Success(t *testing.T) {
	setupFakeTool(t)

	tmpDir := t.TempDir()
	isolinuxDir := filepath.Join(tmpDir, "isolinux")
	if err := os.MkdirAll(isolinuxDir, 0755); err != nil {
		t.Fatalf("failed to create isolinux dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(isolinuxDir, "isolinux.bin"), []byte("dummy"), 0644); err != nil {
		t.Fatalf("failed to create isolinux.bin: %v", err)
	}

	outputPath := filepath.Join(tmpDir, "output.iso")

	b := &ISOBuilder{
		executor: &mockExecutor{Output: []byte("ISO build successful"), createOutput: true},
	}
	err := b.Build(context.Background(), tmpDir, outputPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verificar que o checksum foi gerado
	checksumFile := outputPath + ".sha256"
	if _, err := os.Stat(checksumFile); os.IsNotExist(err) {
		t.Error("expected checksum file to exist")
	}

	data, err := os.ReadFile(checksumFile)
	if err != nil {
		t.Fatalf("failed to read checksum file: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty checksum file")
	}
}

func TestISOBuilder_Build_WithUEFI(t *testing.T) {
	setupFakeTool(t)

	tmpDir := t.TempDir()

	isolinuxDir := filepath.Join(tmpDir, "isolinux")
	if err := os.MkdirAll(isolinuxDir, 0755); err != nil {
		t.Fatalf("failed to create isolinux dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(isolinuxDir, "isolinux.bin"), []byte("dummy"), 0644); err != nil {
		t.Fatalf("failed to create isolinux.bin: %v", err)
	}

	efiDir := filepath.Join(tmpDir, "boot", "grub")
	if err := os.MkdirAll(efiDir, 0755); err != nil {
		t.Fatalf("failed to create efi dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(efiDir, "efi.img"), []byte("dummy efi"), 0644); err != nil {
		t.Fatalf("failed to create efi.img: %v", err)
	}

	outputPath := filepath.Join(tmpDir, "output.iso")

	b := &ISOBuilder{
		executor: &mockExecutor{Output: []byte("ISO with UEFI built"), createOutput: true},
	}
	err := b.Build(context.Background(), tmpDir, outputPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	checksumFile := outputPath + ".sha256"
	if _, err := os.Stat(checksumFile); os.IsNotExist(err) {
		t.Error("expected checksum file to exist")
	}
}

func TestISOBuilder_Build_ExecError(t *testing.T) {
	setupFakeTool(t)

	tmpDir := t.TempDir()
	isolinuxDir := filepath.Join(tmpDir, "isolinux")
	if err := os.MkdirAll(isolinuxDir, 0755); err != nil {
		t.Fatalf("failed to create isolinux dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(isolinuxDir, "isolinux.bin"), []byte("dummy"), 0644); err != nil {
		t.Fatalf("failed to create isolinux.bin: %v", err)
	}

	b := &ISOBuilder{
		executor: &mockExecutor{Err: errors.New("xorriso: command failed")},
	}
	err := b.Build(context.Background(), tmpDir, filepath.Join(tmpDir, "output.iso"))
	if err == nil {
		t.Error("expected error from executor, got nil")
	}
}

func TestSHA256Checksum(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "hello world"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cs, err := sha256Checksum(testFile)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cs == "" {
		t.Error("expected non-empty checksum")
	}
}

func TestSHA256Checksum_FileNotFound(t *testing.T) {
	_, err := sha256Checksum("/nonexistent/file.iso")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestFileExists(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "exists.txt")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	if !fileExists(tmpFile) {
		t.Error("expected fileExists to return true for existing file")
	}
	if fileExists("/nonexistent/file") {
		t.Error("expected fileExists to return false for nonexistent file")
	}
}
