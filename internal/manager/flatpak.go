package manager

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// FlatpakBackend implementa PackageService usando flatpak
type FlatpakBackend struct {
	exec Executor
}

func NewFlatpakBackend() *FlatpakBackend {
	return &FlatpakBackend{exec: &RealExecutor{}}
}

func (f *FlatpakBackend) Install(pkgName string) error {
	if strings.TrimSpace(pkgName) == "" {
		return fmt.Errorf("package name cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	out, err := f.exec.RunContext(ctx, "flatpak", "install", "--noninteractive", "-y", pkgName)
	if err != nil {
		return fmt.Errorf("flatpak install: %s — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (f *FlatpakBackend) Remove(pkgName string) error {
	if strings.TrimSpace(pkgName) == "" {
		return fmt.Errorf("package name cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	out, err := f.exec.RunContext(ctx, "flatpak", "uninstall", "--noninteractive", "-y", pkgName)
	if err != nil {
		return fmt.Errorf("flatpak remove: %s — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (f *FlatpakBackend) Update() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	out, err := f.exec.RunContext(ctx, "flatpak", "update", "--noninteractive", "-y")
	if err != nil {
		return fmt.Errorf("flatpak update: %s — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (f *FlatpakBackend) List() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := f.exec.RunContext(ctx, "flatpak", "list", "--columns=application,name,description")
	if err != nil {
		return nil, fmt.Errorf("flatpak list: %s — %s", err, strings.TrimSpace(string(out)))
	}
	var results []string
	for i, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if i == 0 || line == "" {
			continue // skip header
		}
		results = append(results, line)
	}
	return results, nil
}

func (f *FlatpakBackend) Search(query string) ([]string, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := f.exec.RunContext(ctx, "flatpak", "search", "--columns=application,name,description", query)
	if err != nil && len(out) == 0 {
		return nil, fmt.Errorf("flatpak search: %s", err)
	}
	var results []string
	for i, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		// pula o cabeçalho (primeira linha) e linhas vazias
		if i == 0 || line == "" {
			continue
		}
		results = append(results, line)
	}
	return results, nil
}
