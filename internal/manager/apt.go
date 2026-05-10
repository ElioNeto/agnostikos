package manager

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

// AptBackend implementa PackageService usando apt-get e dpkg
type AptBackend struct {
	exec Executor
}

// NewAptBackend cria um backend APT com o executor fornecido
func NewAptBackend(exec Executor) *AptBackend {
	return &AptBackend{exec: exec}
}

// Install instala um pacote via apt-get install -y com DEBIAN_FRONTEND=noninteractive
func (a *AptBackend) Install(pkgName string) error {
	if strings.TrimSpace(pkgName) == "" {
		return errors.New("package name cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	out, err := a.exec.RunContext(ctx, "env",
		"DEBIAN_FRONTEND=noninteractive", "apt-get", "install", "-y", pkgName)
	if err != nil {
		return fmt.Errorf("apt install: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Remove remove um pacote via apt-get remove -y
func (a *AptBackend) Remove(pkgName string) error {
	if strings.TrimSpace(pkgName) == "" {
		return errors.New("package name cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	out, err := a.exec.RunContext(ctx, "apt-get", "remove", "-y", pkgName)
	if err != nil {
		return fmt.Errorf("apt remove: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Update atualiza um pacote via apt-get install --only-upgrade -y com DEBIAN_FRONTEND=noninteractive
func (a *AptBackend) Update(pkg string) error {
	if strings.TrimSpace(pkg) == "" {
		return errors.New("package name cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	out, err := a.exec.RunContext(ctx, "env",
		"DEBIAN_FRONTEND=noninteractive", "apt-get", "install", "--only-upgrade", "-y", pkg)
	if err != nil {
		return fmt.Errorf("apt update: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// UpdateAll atualiza todos os pacotes via apt-get upgrade -y com DEBIAN_FRONTEND=noninteractive
func (a *AptBackend) UpdateAll() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	out, err := a.exec.RunContext(ctx, "env",
		"DEBIAN_FRONTEND=noninteractive", "apt-get", "upgrade", "-y")
	if err != nil {
		return fmt.Errorf("apt update all: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// List lista pacotes instalados via dpkg -l, retornando apenas os com status "ii" (installed)
func (a *AptBackend) List() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := a.exec.RunContext(ctx, "dpkg", "-l")
	if err != nil {
		return nil, fmt.Errorf("apt list: %w — %s", err, strings.TrimSpace(string(out)))
	}
	var results []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		// dpkg -l output: lines starting with "ii" are installed packages
		if strings.HasPrefix(line, "ii ") {
			results = append(results, line)
		}
	}
	return results, nil
}

// Search pesquisa pacotes via apt-cache search, retornando os nomes dos pacotes
func (a *AptBackend) Search(query string) ([]string, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("search query cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := a.exec.RunContext(ctx, "apt-cache", "search", query)
	if err != nil && len(out) == 0 {
		return nil, fmt.Errorf("apt search: %w", err)
	}
	if err != nil && len(out) > 0 {
		log.Printf("warning: apt-cache search returned error but partial output: %v", err)
	}
	var results []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			// apt-cache search returns "pkgname - description"
			// Extract just the package name before " - "
			if idx := strings.Index(line, " - "); idx != -1 {
				results = append(results, line[:idx])
			} else {
				results = append(results, line)
			}
		}
	}
	return results, nil
}

// Name retorna o nome do backend
func (a *AptBackend) Name() string {
	return "apt"
}
