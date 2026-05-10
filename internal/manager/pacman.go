package manager

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// PacmanBackend implementa PackageService usando pacman
//
// Segurança: pacman verifica assinaturas GPG por padrão (via SigLevel no
// pacman.conf). A flag --noconfirm apenas pula a confirmação interativa e
// NÃO desativa a verificação de assinaturas. --needed (não usado aqui)
// também não afeta verificação — apenas evita reinstalar pacotes já atuais.
//
// Nix e Flatpak também verificam integridade por padrão (Nix usa hashes
// no nixpkgs, Flatpak usa GPG + checksums do repositório OSTree).
type PacmanBackend struct {
	exec Executor
}

func NewPacmanBackend(exec Executor) *PacmanBackend {
	return &PacmanBackend{exec: exec}
}

func (p *PacmanBackend) Install(pkgName string) error {
	if strings.TrimSpace(pkgName) == "" {
		return errors.New("package name cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	// --noconfirm não desativa verificação de assinatura (ver doc do tipo)
	out, err := p.exec.RunContext(ctx, "pacman", "-S", "--noconfirm", pkgName)
	if err != nil {
		return fmt.Errorf("pacman install: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (p *PacmanBackend) Remove(pkgName string) error {
	if strings.TrimSpace(pkgName) == "" {
		return errors.New("package name cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	out, err := p.exec.RunContext(ctx, "pacman", "-R", "--noconfirm", pkgName)
	if err != nil {
		return fmt.Errorf("pacman remove: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (p *PacmanBackend) Update(pkg string) error {
	if strings.TrimSpace(pkg) == "" {
		return errors.New("package name cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	// --noconfirm não desativa verificação de assinatura (ver doc do tipo)
	out, err := p.exec.RunContext(ctx, "pacman", "-S", "--noconfirm", pkg)
	if err != nil {
		return fmt.Errorf("pacman update: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (p *PacmanBackend) UpdateAll() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	// --noconfirm não desativa verificação de assinatura (ver doc do tipo)
	out, err := p.exec.RunContext(ctx, "pacman", "-Syu", "--noconfirm")
	if err != nil {
		return fmt.Errorf("pacman update all: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (p *PacmanBackend) List() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := p.exec.RunContext(ctx, "pacman", "-Q")
	if err != nil {
		return nil, fmt.Errorf("pacman list: %w — %s", err, strings.TrimSpace(string(out)))
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

func (p *PacmanBackend) Search(query string) ([]string, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("search query cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := p.exec.RunContext(ctx, "pacman", "-Ss", query)
	if err != nil && len(out) == 0 {
		return nil, fmt.Errorf("pacman search: %w", err)
	}
	var results []string
	for _, line := range strings.Split(string(out), "\n") {
		if line != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			results = append(results, line)
		}
	}
	return results, nil
}
