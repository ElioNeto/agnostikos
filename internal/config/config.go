package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the agnostic.yaml configuration structure.
type Config struct {
	Version  string `yaml:"version"`
	Locale   string `yaml:"locale"`
	Timezone string `yaml:"timezone"`
	Packages struct {
		Base  []string `yaml:"base"`
		Extra []string `yaml:"extra"`
	} `yaml:"packages"`
	Backends struct {
		Default  string `yaml:"default"`
		Fallback string `yaml:"fallback"`
	} `yaml:"backends"`
	User struct {
		Name  string `yaml:"name"`
		Shell string `yaml:"shell"`
	} `yaml:"user"`
}

// Load reads a YAML file at the given path and unmarshals it into a Config.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}
	return &cfg, nil
}

// Validate checks that all required fields are present.
func (c *Config) Validate() error {
	if c.Version == "" {
		return fmt.Errorf("version is required")
	}
	if c.Locale == "" {
		return fmt.Errorf("locale is required")
	}
	if c.Backends.Default == "" {
		return fmt.Errorf("backends.default is required")
	}
	return nil
}
