package manager

import (
	"fmt"
	"os/exec"
	"strings"
)

// PacmanBackend implementa PackageService usando pacman
type PacmanBackend struct{}

func (p *PacmanBackend) Install(pkgName string) error {
	out, err := exec.Command("pacman", "-S", "--noconfirm", pkgName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("pacman install: %s — %s", err, string(out))
	}
	return nil
}

func (p *PacmanBackend) Remove(pkgName string) error {
	out, err := exec.Command("pacman", "-R", "--noconfirm", pkgName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("pacman remove: %s — %s", err, string(out))
	}
	return nil
}

func (p *PacmanBackend) Update() error {
	out, err := exec.Command("pacman", "-Syu", "--noconfirm").CombinedOutput()
	if err != nil {
		return fmt.Errorf("pacman update: %s — %s", err, string(out))
	}
	return nil
}

func (p *PacmanBackend) Search(query string) ([]string, error) {
	out, err := exec.Command("pacman", "-Ss", query).CombinedOutput()
	if err != nil && !strings.Contains(string(out), "no results") {
		return nil, fmt.Errorf("pacman search: %s", err)
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n"), nil
}
