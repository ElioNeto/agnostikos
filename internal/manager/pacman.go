package manager

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// PacmanBackend implementa PackageService usando pacman
type PacmanBackend struct {
	exec Executor
}

func NewPacmanBackend() *PacmanBackend {
	return &PacmanBackend{exec: &RealExecutor{}}
}

func (p *PacmanBackend) Install(pkgName string) error {
	if strings.TrimSpace(pkgName) == "" {
		return fmt.Errorf("package name cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	out, err := p.exec.RunContext(ctx, "pacman", "-S", "--noconfirm", pkgName)
	if err != nil {
		return fmt.Errorf("pacman install: %s — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (p *PacmanBackend) Remove(pkgName string) error {
	if strings.TrimSpace(pkgName) == "" {
		return fmt.Errorf("package name cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	out, err := p.exec.RunContext(ctx, "pacman", "-R", "--noconfirm", pkgName)
	if err != nil {
		return fmt.Errorf("pacman remove: %s — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (p *PacmanBackend) Update() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	out, err := p.exec.RunContext(ctx, "pacman", "-Syu", "--noconfirm")
	if err != nil {
		return fmt.Errorf("pacman update: %s — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (p *PacmanBackend) Search(query string) ([]string, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := p.exec.RunContext(ctx, "pacman", "-Ss", query)
	if err != nil && len(out) == 0 {
		return nil, fmt.Errorf("pacman search: %s", err)
	}
	var results []string
	for _, line := range strings.Split(string(out), "\n") {
		if line != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			results = append(results, line)
		}
	}
	return results, nil
}
