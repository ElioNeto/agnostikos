package manager

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// FlatpakBackend implementa PackageService usando flatpak
type FlatpakBackend struct {
	exec Executor
}

func NewFlatpakBackend(exec Executor) *FlatpakBackend {
	return &FlatpakBackend{exec: exec}
}

func (f *FlatpakBackend) Install(pkgName string) error {
	if strings.TrimSpace(pkgName) == "" {
		return errors.New("package name cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	out, err := f.exec.RunContext(ctx, "flatpak", "install", "--noninteractive", "-y", pkgName)
	if err != nil {
		return fmt.Errorf("flatpak install: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (f *FlatpakBackend) Remove(pkgName string) error {
	if strings.TrimSpace(pkgName) == "" {
		return errors.New("package name cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	out, err := f.exec.RunContext(ctx, "flatpak", "uninstall", "--noninteractive", "-y", pkgName)
	if err != nil {
		return fmt.Errorf("flatpak remove: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (f *FlatpakBackend) Update(pkg string) error {
	if strings.TrimSpace(pkg) == "" {
		return errors.New("package name cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	out, err := f.exec.RunContext(ctx, "flatpak", "update", "--noninteractive", "-y", pkg)
	if err != nil {
		return fmt.Errorf("flatpak update: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (f *FlatpakBackend) UpdateAll() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	out, err := f.exec.RunContext(ctx, "flatpak", "update", "--noninteractive", "-y")
	if err != nil {
		return fmt.Errorf("flatpak update all: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (f *FlatpakBackend) List() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := f.exec.RunContext(ctx, "flatpak", "list", "--columns=application,name,description")
	if err != nil {
		return nil, fmt.Errorf("flatpak list: %w — %s", err, strings.TrimSpace(string(out)))
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
		return nil, errors.New("search query cannot be empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := f.exec.RunContext(ctx, "flatpak", "search", "--columns=application,name,description", query)
	if err != nil && len(out) == 0 {
		return nil, fmt.Errorf("flatpak search: %w", err)
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
