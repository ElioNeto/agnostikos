package manager

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ZypperBackend implementa PackageService usando zypper
type ZypperBackend struct {
	exec Executor
}

// NewZypperBackend cria um backend Zypper com o executor fornecido
func NewZypperBackend(exec Executor) *ZypperBackend {
	return &ZypperBackend{exec: exec}
}

// Install instala um pacote via zypper install -y
func (z *ZypperBackend) Install(pkgName string) error {
	if strings.TrimSpace(pkgName) == "" {
		return errors.New("package name cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	out, err := z.exec.RunContext(ctx, "zypper", "install", "-y", pkgName)
	if err != nil {
		return fmt.Errorf("zypper install: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Remove remove um pacote via zypper remove -y
func (z *ZypperBackend) Remove(pkgName string) error {
	if strings.TrimSpace(pkgName) == "" {
		return errors.New("package name cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	out, err := z.exec.RunContext(ctx, "zypper", "remove", "-y", pkgName)
	if err != nil {
		return fmt.Errorf("zypper remove: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Update atualiza um pacote específico via zypper update -y <pkg>
func (z *ZypperBackend) Update(pkg string) error {
	if strings.TrimSpace(pkg) == "" {
		return errors.New("package name cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	out, err := z.exec.RunContext(ctx, "zypper", "update", "-y", pkg)
	if err != nil {
		return fmt.Errorf("zypper update: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// UpdateAll atualiza todos os pacotes via zypper update -y
func (z *ZypperBackend) UpdateAll() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	out, err := z.exec.RunContext(ctx, "zypper", "update", "-y")
	if err != nil {
		return fmt.Errorf("zypper update all: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// List lista pacotes instalados via zypper packages --installed-only
func (z *ZypperBackend) List() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := z.exec.RunContext(ctx, "zypper", "packages", "--installed-only")
	if err != nil {
		return nil, fmt.Errorf("zypper list: %w — %s", err, strings.TrimSpace(string(out)))
	}
	var results []string
	started := false
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Pula o cabeçalho e a linha separadora
		if strings.HasPrefix(line, "S | Name") || strings.HasPrefix(line, "--+--") {
			started = true
			continue
		}
		if started {
			// Linhas: "i | pkgname | version | arch | ..."
			results = append(results, line)
		}
	}
	return results, nil
}

// Search pesquisa pacotes via zypper search
func (z *ZypperBackend) Search(query string) ([]string, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("search query cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := z.exec.RunContext(ctx, "zypper", "search", query)
	if err != nil && len(out) == 0 {
		return nil, fmt.Errorf("zypper search: %w", err)
	}
	var results []string
	started := false
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Pula cabeçalho e separador
		if strings.HasPrefix(line, "S | Name") || strings.HasPrefix(line, "--+--") {
			started = true
			continue
		}
		if started {
			results = append(results, line)
		}
	}
	return results, nil
}

// Name retorna o nome do backend
func (z *ZypperBackend) Name() string {
	return "zypper"
}
