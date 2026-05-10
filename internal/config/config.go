// Package config handles loading and validation of AgnosticOS configuration.
package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// SupportedBackends lista os backends válidos
var SupportedBackends = map[string]bool{
	"pacman":  true,
	"nix":     true,
	"flatpak": true,
}

// SupportedProfiles lista os perfis válidos
var SupportedProfiles = map[string]bool{
	"minimal": true,
	"desktop": true,
	"server":  true,
	"dev":     true,
}

// localeRegex valida formato <idioma>_<PAÍS>.<codificação> (ex: pt_BR.UTF-8)
var localeRegex = regexp.MustCompile(`^[a-z]{2}_[A-Z]{2}\.[a-zA-Z0-9._-]+$`)

// timezoneRegex valida formato <Região>/<Cidade> (ex: America/Sao_Paulo)
// ou nomes simples como "UTC", "CET", "EST5EDT"
var timezoneRegex = regexp.MustCompile(`^[A-Za-z_]+(/[A-Za-z_]+)?$`)

// CacheConfig holds configuration for the package metadata cache.
type CacheConfig struct {
	// Dir is the directory for disk cache storage.
	// If empty, defaults to ~/.cache/agnostikos.
	Dir string `yaml:"cache_dir,omitempty"`
	// StableTTL is the TTL for "stable" version policy cache entries.
	// Default is 24 hours.
	StableTTL time.Duration `yaml:"stable_cache_ttl,omitempty"`
	// LatestTTL is the TTL for "latest" version policy cache entries.
	// Default is 1 hour.
	LatestTTL time.Duration `yaml:"latest_cache_ttl,omitempty"`
}

// Config represents the agnostic.yaml configuration structure.
type Config struct {
	Version  string `yaml:"version"`
	Profile  string `yaml:"profile"`
	Locale   string `yaml:"locale"`
	Timezone string `yaml:"timezone"`
	Packages struct {
		Base    []string `yaml:"base"`
		Extra   []string `yaml:"extra"`
		Dev     []string `yaml:"dev"`
		Desktop []string `yaml:"desktop"`
	} `yaml:"packages"`
	Backends struct {
		Default         string   `yaml:"default"`
		Fallback        string   `yaml:"fallback"`
		Priority        []string `yaml:"priority"`
		Version         string   `yaml:"version"`
		FallbackEnabled bool     `yaml:"fallback_enabled"`
	} `yaml:"backends"`
	User struct {
		Name   string   `yaml:"name"`
		Shell  string   `yaml:"shell"`
		Groups []string `yaml:"groups"`
	} `yaml:"user"`
	Dotfiles        *DotfilesConfig `yaml:"dotfiles,omitempty"`
	PackagePolicies []PackagePolicy `yaml:"package_policies,omitempty"`
	Cache           CacheConfig     `yaml:"cache,omitempty"`
}

// PackagePolicy defines a per-package version policy override.
type PackagePolicy struct {
	// Name is the package name this policy applies to.
	Name string `yaml:"name"`
	// Version is the policy: "latest", "stable", "pinned", or empty (inherit global).
	Version string `yaml:"version"`
	// Pin is the exact version to pin when Version is "pinned" (e.g. "1.2.3").
	Pin string `yaml:"pin"`
}

// DotfilesConfig configura o gerenciamento de dotfiles.
type DotfilesConfig struct {
	// Source é o caminho local ou URL git para obter os dotfiles.
	// Se vazio, usa os dotfiles embutidos (configs/).
	Source string `yaml:"source,omitempty"`
	// Apply indica se os dotfiles devem ser aplicados automaticamente
	// durante o bootstrap.
	Apply bool `yaml:"apply"`
}

// Load reads a YAML file at the given path, unmarshals it into a Config,
// and validates all fields. Returns all validation errors if any.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Validate checks all fields in the config for correctness.
// Returns a list of all validation errors (not just the first one).
func (c *Config) Validate() error {
	var errs []string

	// Version
	if c.Version == "" {
		errs = append(errs, "version is required")
	}

	// Profile (optional, validate if set)
	if c.Profile != "" && !SupportedProfiles[c.Profile] {
		errs = append(errs, fmt.Sprintf("profile %q is not supported — must be one of: minimal, desktop, server, dev", c.Profile))
	}

	// Locale
	if c.Locale == "" {
		errs = append(errs, "locale is required")
	} else if !localeRegex.MatchString(c.Locale) {
		errs = append(errs, fmt.Sprintf("locale %q has invalid format — expected <lang>_<REGION>.<encoding> (e.g. pt_BR.UTF-8)", c.Locale))
	}

	// Timezone (optional, validate if set)
	if c.Timezone != "" && !timezoneRegex.MatchString(c.Timezone) {
		errs = append(errs, fmt.Sprintf("timezone %q has invalid format — expected <Region>/<City> (e.g. America/Sao_Paulo)", c.Timezone))
	}

	// Backends.Default
	if c.Backends.Default == "" {
		errs = append(errs, "backends.default is required")
	} else if !SupportedBackends[c.Backends.Default] {
		errs = append(errs, fmt.Sprintf("backends.default %q is not supported — must be one of: pacman, nix, flatpak", c.Backends.Default))
	}

	// Backends.Fallback (optional, validate if set)
	if c.Backends.Fallback != "" && !SupportedBackends[c.Backends.Fallback] {
		errs = append(errs, fmt.Sprintf("backends.fallback %q is not supported — must be one of: pacman, nix, flatpak", c.Backends.Fallback))
	}

	// Backends.Priority (optional, validate entries if set)
	for i, name := range c.Backends.Priority {
		if !SupportedBackends[name] {
			errs = append(errs, fmt.Sprintf("backends.priority[%d] %q is not supported — must be one of: pacman, nix, flatpak", i, name))
		}
	}

	// Backends.Version (optional, validate if set)
	if c.Backends.Version != "" {
		switch c.Backends.Version {
		case "latest", "stable":
			// valid
		default:
			// Assume it's a semver pin, just warn about format
			if !strings.Contains(c.Backends.Version, ".") {
				errs = append(errs, fmt.Sprintf("backends.version %q is not valid — must be 'latest', 'stable', or a semver string (e.g. '1.2.3')", c.Backends.Version))
			}
		}
	}

	// Packages — check for empty entries
	for i, pkg := range c.Packages.Base {
		if strings.TrimSpace(pkg) == "" {
			errs = append(errs, fmt.Sprintf("packages.base[%d] is empty", i))
		}
	}
	for i, pkg := range c.Packages.Extra {
		if strings.TrimSpace(pkg) == "" {
			errs = append(errs, fmt.Sprintf("packages.extra[%d] is empty", i))
		}
	}
	for i, pkg := range c.Packages.Dev {
		if strings.TrimSpace(pkg) == "" {
			errs = append(errs, fmt.Sprintf("packages.dev[%d] is empty", i))
		}
	}
	for i, pkg := range c.Packages.Desktop {
		if strings.TrimSpace(pkg) == "" {
			errs = append(errs, fmt.Sprintf("packages.desktop[%d] is empty", i))
		}
	}

	// User (optional, validate if set)
	if c.User.Shell != "" && !strings.HasPrefix(c.User.Shell, "/") {
		errs = append(errs, fmt.Sprintf("user.shell %q must be an absolute path (starting with /)", c.User.Shell))
	}
	for i, g := range c.User.Groups {
		if strings.TrimSpace(g) == "" {
			errs = append(errs, fmt.Sprintf("user.groups[%d] is empty", i))
		}
	}

	// Dotfiles (optional, validate source if set)
	if c.Dotfiles != nil && c.Dotfiles.Source != "" {
		if !strings.HasPrefix(c.Dotfiles.Source, "http://") &&
			!strings.HasPrefix(c.Dotfiles.Source, "https://") &&
			!strings.HasPrefix(c.Dotfiles.Source, "git@") &&
			!strings.HasPrefix(c.Dotfiles.Source, "/") {
			errs = append(errs, fmt.Sprintf("dotfiles.source %q must be a git URL (http/https/git@) or an absolute local path", c.Dotfiles.Source))
		}
	}

	// PackagePolicies (optional, validate entries)
	for i, pp := range c.PackagePolicies {
		if pp.Name == "" {
			errs = append(errs, fmt.Sprintf("package_policies[%d].name is required", i))
		}
		switch pp.Version {
		case "", "latest", "stable":
			if pp.Pin != "" {
				errs = append(errs, fmt.Sprintf("package_policies[%d].pin must be empty when version is %q", i, pp.Version))
			}
		case "pinned":
			if pp.Pin == "" {
				errs = append(errs, fmt.Sprintf("package_policies[%d].pin is required when version is 'pinned'", i))
			}
		default:
			errs = append(errs, fmt.Sprintf("package_policies[%d].version %q is not valid — must be 'latest', 'stable', 'pinned', or empty", i, pp.Version))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// DefaultConfig returns a Config populated with sensible defaults.
// Cache defaults: StableTTL = 24h, LatestTTL = 1h.
func DefaultConfig() *Config {
	return &Config{
		Version: "1.0",
		Profile: "minimal",
		Locale:  "en_US.UTF-8",
		Backends: struct {
			Default         string   `yaml:"default"`
			Fallback        string   `yaml:"fallback"`
			Priority        []string `yaml:"priority"`
			Version         string   `yaml:"version"`
			FallbackEnabled bool     `yaml:"fallback_enabled"`
		}{
			Default:  "pacman",
			Priority: []string{"pacman", "nix", "flatpak"},
		},
		Cache: CacheConfig{
			StableTTL: 24 * time.Hour,
			LatestTTL: 1 * time.Hour,
		},
	}
}
