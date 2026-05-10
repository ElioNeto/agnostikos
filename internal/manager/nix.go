package manager

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// NixBackend implementa PackageService usando nix
type NixBackend struct {
	exec Executor
}

func NewNixBackend(exec Executor) *NixBackend {
	return &NixBackend{exec: exec}
}

func (n *NixBackend) Install(pkgName string) error {
	if strings.TrimSpace(pkgName) == "" {
		return errors.New("package name cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	pkg := pkgName
	if !strings.Contains(pkgName, "#") {
		pkg = "nixpkgs#" + pkgName
	}
	out, err := n.exec.RunContext(ctx, "nix", "profile", "install", pkg)
	if err != nil {
		return fmt.Errorf("nix install: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (n *NixBackend) Remove(pkgName string) error {
	if strings.TrimSpace(pkgName) == "" {
		return errors.New("package name cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	out, err := n.exec.RunContext(ctx, "nix", "profile", "remove", pkgName)
	if err != nil {
		return fmt.Errorf("nix remove: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (n *NixBackend) Update(pkg string) error {
	if strings.TrimSpace(pkg) == "" {
		return errors.New("package name cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	out, err := n.exec.RunContext(ctx, "nix", "profile", "upgrade", pkg)
	if err != nil {
		return fmt.Errorf("nix update: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (n *NixBackend) UpdateAll() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	out, err := n.exec.RunContext(ctx, "nix", "profile", "upgrade")
	if err != nil {
		return fmt.Errorf("nix update all: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (n *NixBackend) List() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := n.exec.RunContext(ctx, "nix", "profile", "list", "--json")
	if err != nil {
		return nil, fmt.Errorf("nix list: %w — %s", err, strings.TrimSpace(string(out)))
	}
	// nix profile list --json retorna JSON array — simplificamos para texto puro
	var results []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			results = append(results, line)
		}
	}
	return results, nil
}

func (n *NixBackend) Search(query string) ([]string, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("search query cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := n.exec.RunContext(ctx, "nix", "search", "nixpkgs", query)
	if err != nil && len(out) == 0 {
		return nil, fmt.Errorf("nix search: %w", err)
	}
	var results []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "* ") {
			results = append(results, strings.TrimPrefix(line, "* "))
		}
	}
	return results, nil
}
