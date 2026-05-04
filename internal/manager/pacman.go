package manager

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// PacmanBackend implementa PackageService usando pacman
type PacmanBackend struct{}

func (p *PacmanBackend) Install(pkgName string) error {
	if strings.TrimSpace(pkgName) == "" {
		return fmt.Errorf("package name cannot be empty")
	}
	out, err := exec.Command("pacman", "-S", "--noconfirm", pkgName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("pacman install: %s — %s", err, string(out))
	}
	return nil
}

func (p *PacmanBackend) Remove(pkgName string) error {
	if strings.TrimSpace(pkgName) == "" {
		return fmt.Errorf("package name cannot be empty")
	}
	out, err := exec.Command("pacman", "-R", "--noconfirm", pkgName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("pacman remove: %s — %s", err, string(out))
	}
	return nil
}

func (p *PacmanBackend) Update() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	out, err := exec.CommandContext(ctx, "pacman", "Syu", "--noconfirm").CombinedOutput()
	if err != nil {
		return fmt.Errorf("pacman update: %s - %s", err, string(out))
	}
	return nil
}

func (p *PacmanBackend) Search(query string) ([]string, error) {
	out, err := exec.Command("pacman", "-Ss", query).CombinedOutput()
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
