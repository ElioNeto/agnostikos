// Package iso provides ISO image building utilities for AgnosticOS.
package iso

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// Executor abstrai a execução de comandos externos
type Executor interface {
	RunContext(ctx context.Context, name string, args ...string) ([]byte, error)
}

// RealExecutor chama exec.CommandContext de verdade
type RealExecutor struct{}

func (r *RealExecutor) RunContext(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

// ISOBuilder constrói uma ISO bootável a partir de um RootFS
type ISOBuilder struct {
	executor   Executor
	rootfsPath string
	outputPath string
}

// NewISOBuilder cria um novo ISOBuilder
func NewISOBuilder(rootfsPath, outputPath string) *ISOBuilder {
	return &ISOBuilder{
		executor:   &RealExecutor{},
		rootfsPath: rootfsPath,
		outputPath: outputPath,
	}
}

// detectTool procura xorriso ou mkisofs no PATH
func (b *ISOBuilder) detectTool() (string, error) {
	tools := []string{"xorrisofs", "mkisofs", "genisoimage"}
	for _, tool := range tools {
		if path, err := exec.LookPath(tool); err == nil {
			return path, nil
		}
	}
	return "", errors.New("no ISO creation tool found (tried xorrisofs, mkisofs, genisoimage); install libisoburn or genisoimage")
}

// fileExists verifica se um arquivo existe
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// sha256Checksum calcula o checksum SHA256 de um arquivo
func sha256Checksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file for checksum: %w", err)
	}
	defer func() {
		// Erro no Close não compromete o checksum pois o arquivo já foi lido
		_ = f.Close()
	}()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("compute checksum: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Build gera a ISO bootável a partir do RootFS
func (b *ISOBuilder) Build(ctx context.Context, rootfsPath, outputPath string) error {
	if rootfsPath == "" {
		rootfsPath = b.rootfsPath
	}
	if outputPath == "" {
		outputPath = b.outputPath
	}

	if rootfsPath == "" {
		return errors.New("rootfs path is required")
	}
	if outputPath == "" {
		return errors.New("output path is required")
	}

	tool, err := b.detectTool()
	if err != nil {
		return err
	}

	fmt.Printf("[iso] creating ISO from %s -> %s\n", rootfsPath, outputPath)

	// Construir argumentos básicos para BIOS (isolinux)
	isolinuxBin := filepath.Join(rootfsPath, "isolinux", "isolinux.bin")

	args := []string{
		"-o", outputPath,
	}

	if fileExists(isolinuxBin) {
		args = append(args,
			"-b", "isolinux/isolinux.bin",
			"-c", "isolinux/boot.cat",
			"-no-emul-boot",
			"-boot-load-size", "4",
			"-boot-info-table",
		)
	}

	// Adicionar suporte UEFI se o arquivo efi.img existir
	efiImg := filepath.Join(rootfsPath, "boot", "grub", "efi.img")
	if fileExists(efiImg) {
		fmt.Println("[iso] adding UEFI boot support")
		args = append(args,
			"-eltorito-alt-boot",
			"-e", "boot/grub/efi.img",
			"-no-emul-boot",
		)
	}

	args = append(args, rootfsPath)

	output, err := b.executor.RunContext(ctx, tool, args...)
	if err != nil {
		return fmt.Errorf("ISO build failed: %w\nOutput: %s", err, string(output))
	}
	fmt.Printf("[iso] build output: %s\n", string(output))

	// Gerar checksum SHA256
	checksum, err := sha256Checksum(outputPath)
	if err != nil {
		return fmt.Errorf("checksum generation failed: %w", err)
	}

	// Escrever arquivo de checksum
	checksumFile := outputPath + ".sha256"
	csContent := fmt.Sprintf("%s  %s\n", checksum, filepath.Base(outputPath))
	if err := os.WriteFile(checksumFile, []byte(csContent), 0644); err != nil {
		return fmt.Errorf("write checksum file: %w", err)
	}

	fmt.Printf("[iso] ISO created: %s (%s)\n", outputPath, checksum)
	return nil
}
