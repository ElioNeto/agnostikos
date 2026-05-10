package manager

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// DNFBackend implementa PackageService usando dnf (ou yum como fallback)
type DNFBackend struct {
	exec Executor
	bin  string // "dnf" ou "yum"
}

// NewDNFBackend cria um backend DNF/YUM, detectando automaticamente qual binário está disponível
func NewDNFBackend(executor Executor) *DNFBackend {
	bin := "dnf"
	if _, err := exec.LookPath("dnf"); err != nil {
		bin = "yum"
	}
	return &DNFBackend{exec: executor, bin: bin}
}

// Install instala um pacote via dnf install -y
func (d *DNFBackend) Install(pkgName string) error {
	if strings.TrimSpace(pkgName) == "" {
		return errors.New("package name cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	out, err := d.exec.RunContext(ctx, d.bin, "install", "-y", pkgName)
	if err != nil {
		return fmt.Errorf("%s install: %w — %s", d.bin, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Remove remove um pacote via dnf remove -y
func (d *DNFBackend) Remove(pkgName string) error {
	if strings.TrimSpace(pkgName) == "" {
		return errors.New("package name cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	out, err := d.exec.RunContext(ctx, d.bin, "remove", "-y", pkgName)
	if err != nil {
		return fmt.Errorf("%s remove: %w — %s", d.bin, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Update atualiza um pacote específico via dnf upgrade -y <pkg>
func (d *DNFBackend) Update(pkg string) error {
	if strings.TrimSpace(pkg) == "" {
		return errors.New("package name cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	out, err := d.exec.RunContext(ctx, d.bin, "upgrade", "-y", pkg)
	if err != nil {
		return fmt.Errorf("%s update: %w — %s", d.bin, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// UpdateAll atualiza todos os pacotes via dnf upgrade -y
func (d *DNFBackend) UpdateAll() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	out, err := d.exec.RunContext(ctx, d.bin, "upgrade", "-y")
	if err != nil {
		return fmt.Errorf("%s update all: %w — %s", d.bin, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// List lista pacotes instalados via dnf list installed
func (d *DNFBackend) List() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := d.exec.RunContext(ctx, d.bin, "list", "installed")
	if err != nil {
		return nil, fmt.Errorf("%s list: %w — %s", d.bin, err, strings.TrimSpace(string(out)))
	}
	var results []string
	started := false
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Pula o cabeçalho "Installed Packages"
		if !started && strings.HasPrefix(line, "Installed Packages") {
			started = true
			continue
		}
		if started {
			// Linhas do dnf list installed: "pkgname.arch  version  repo"
			results = append(results, line)
		}
	}
	return results, nil
}

// Search pesquisa pacotes via dnf search
func (d *DNFBackend) Search(query string) ([]string, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("search query cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := d.exec.RunContext(ctx, d.bin, "search", query)
	if err != nil && len(out) == 0 {
		return nil, fmt.Errorf("%s search: %w", d.bin, err)
	}
	var results []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Linhas de resultado: "pkgname.arch : description"
		// Ignora seções como "=== Name Exactly Matched ==="
		if strings.HasPrefix(line, "===") || strings.HasPrefix(line, "---") {
			continue
		}
		if strings.Contains(line, " : ") {
			results = append(results, line)
		}
	}
	return results, nil
}

// Name retorna o nome do backend
func (d *DNFBackend) Name() string {
	return "dnf"
}
