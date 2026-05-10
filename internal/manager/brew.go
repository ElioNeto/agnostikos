package manager

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// BrewBackend implementa PackageService usando brew (Homebrew)
type BrewBackend struct {
	exec Executor
}

// NewBrewBackend cria um backend Homebrew com o executor fornecido
func NewBrewBackend(exec Executor) *BrewBackend {
	return &BrewBackend{exec: exec}
}

// Install instala um pacote via brew install
func (b *BrewBackend) Install(pkgName string) error {
	if strings.TrimSpace(pkgName) == "" {
		return errors.New("package name cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	out, err := b.exec.RunContext(ctx, "brew", "install", pkgName)
	if err != nil {
		return fmt.Errorf("brew install: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Remove remove um pacote via brew uninstall
func (b *BrewBackend) Remove(pkgName string) error {
	if strings.TrimSpace(pkgName) == "" {
		return errors.New("package name cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	out, err := b.exec.RunContext(ctx, "brew", "uninstall", pkgName)
	if err != nil {
		return fmt.Errorf("brew uninstall: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Update atualiza um pacote específico via brew upgrade <pkg>
func (b *BrewBackend) Update(pkg string) error {
	if strings.TrimSpace(pkg) == "" {
		return errors.New("package name cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	out, err := b.exec.RunContext(ctx, "brew", "upgrade", pkg)
	if err != nil {
		return fmt.Errorf("brew upgrade: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// UpdateAll atualiza todos os pacotes via brew upgrade
func (b *BrewBackend) UpdateAll() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	out, err := b.exec.RunContext(ctx, "brew", "upgrade")
	if err != nil {
		return fmt.Errorf("brew upgrade all: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// List lista pacotes instalados via brew list
func (b *BrewBackend) List() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := b.exec.RunContext(ctx, "brew", "list")
	if err != nil {
		return nil, fmt.Errorf("brew list: %w — %s", err, strings.TrimSpace(string(out)))
	}
	var results []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			results = append(results, line)
		}
	}
	return results, nil
}

// Search pesquisa pacotes via brew search
func (b *BrewBackend) Search(query string) ([]string, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("search query cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := b.exec.RunContext(ctx, "brew", "search", query)
	if err != nil && len(out) == 0 {
		return nil, fmt.Errorf("brew search: %w", err)
	}
	var results []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Ignora cabeçalhos de seção do tipo "==> Formulae" ou "==> Casks"
		if strings.HasPrefix(line, "==>") {
			continue
		}
		results = append(results, line)
	}
	return results, nil
}

// Name retorna o nome do backend
func (b *BrewBackend) Name() string {
	return "brew"
}
