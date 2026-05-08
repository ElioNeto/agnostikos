package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agnostic.yaml")
	content := []byte(`
version: "1.0"
locale: pt_BR.UTF-8
timezone: America/Sao_Paulo
packages:
  base:
    - vim
    - git
  extra:
    - docker
    - neovim
backends:
  default: pacman
  fallback: nix
user:
  name: dev
  shell: /bin/zsh
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", cfg.Version)
	}
	if cfg.Locale != "pt_BR.UTF-8" {
		t.Errorf("expected locale pt_BR.UTF-8, got %s", cfg.Locale)
	}
	if len(cfg.Packages.Base) != 2 || cfg.Packages.Base[0] != "vim" {
		t.Errorf("unexpected base packages: %v", cfg.Packages.Base)
	}
	if cfg.Backends.Default != "pacman" {
		t.Errorf("expected default backend pacman, got %s", cfg.Backends.Default)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/agnostic.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestValidate_EmptyVersion(t *testing.T) {
	cfg := &Config{
		Version: "",
		Locale:  "en_US.UTF-8",
		Backends: struct {
			Default  string `yaml:"default"`
			Fallback string `yaml:"fallback"`
		}{Default: "pacman"},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for empty version, got nil")
	}
}

func TestValidate_EmptyLocale(t *testing.T) {
	cfg := &Config{
		Version: "1.0",
		Locale:  "",
		Backends: struct {
			Default  string `yaml:"default"`
			Fallback string `yaml:"fallback"`
		}{Default: "pacman"},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for empty locale, got nil")
	}
}

func TestValidate_EmptyDefaultBackend(t *testing.T) {
	cfg := &Config{
		Version: "1.0",
		Locale:  "en_US.UTF-8",
		Backends: struct {
			Default  string `yaml:"default"`
			Fallback string `yaml:"fallback"`
		}{Default: ""},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for empty default backend, got nil")
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Version:  "1.0",
		Locale:   "en_US.UTF-8",
		Timezone: "UTC",
		Packages: struct {
			Base    []string `yaml:"base"`
			Extra   []string `yaml:"extra"`
			Dev     []string `yaml:"dev"`
			Desktop []string `yaml:"desktop"`
		}{
			Base:  []string{"vim"},
			Extra: []string{"docker"},
		},
		Backends: struct {
			Default  string `yaml:"default"`
			Fallback string `yaml:"fallback"`
		}{Default: "pacman", Fallback: "nix"},
		User: struct {
			Name   string   `yaml:"name"`
			Shell  string   `yaml:"shell"`
			Groups []string `yaml:"groups"`
		}{Name: "dev", Shell: "/bin/bash"},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
