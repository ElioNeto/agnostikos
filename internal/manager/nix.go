package manager

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// NixBackend implementa PackageService usando nix
type NixBackend struct{}

func (n *NixBackend) Install(pkgName string) error {
	if strings.TrimSpace(pkgName) == "" {
		return fmt.Errorf("package name cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	// nix profile install aceita atributos no formato nixpkgs#pkg
	pkg := pkgName
	if !strings.Contains(pkgName, "#") {
		pkg = "nixpkgs#" + pkgName
	}
	out, err := exec.CommandContext(ctx, "nix", "profile", "install", pkg).CombinedOutput()
	if err != nil {
		return fmt.Errorf("nix install: %s — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (n *NixBackend) Remove(pkgName string) error {
	if strings.TrimSpace(pkgName) == "" {
		return fmt.Errorf("package name cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	out, err := exec.CommandContext(ctx, "nix", "profile", "remove", pkgName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("nix remove: %s — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (n *NixBackend) Update() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	out, err := exec.CommandContext(ctx, "nix", "profile", "upgrade", ".*").CombinedOutput()
	if err != nil {
		return fmt.Errorf("nix update: %s — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (n *NixBackend) Search(query string) ([]string, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// --json retorna saída estruturada; sem ele retorna texto legível
	out, err := exec.CommandContext(ctx, "nix", "search", "nixpkgs", query).CombinedOutput()
	if err != nil && len(out) == 0 {
		return nil, fmt.Errorf("nix search: %s", err)
	}
	var results []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		// linhas de resultado começam com "* legacyPackages." ou "* nixpkgs."
		if strings.HasPrefix(line, "* ") {
			results = append(results, strings.TrimPrefix(line, "* "))
		}
	}
	return results, nil
}
