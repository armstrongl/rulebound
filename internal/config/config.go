// Package config handles loading and parsing of rulebound.yml configuration files.
// The config file is optional; when absent, sensible defaults are applied.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// Config holds the parsed contents of a rulebound.yml file.
type Config struct {
	// Title is the human-readable name of the style guide.
	// Defaults to the package directory name when rulebound.yml is absent.
	Title string `yaml:"title"`

	// Description is an optional short description of the style guide.
	Description string `yaml:"description"`

	// BaseURL is the base URL for the generated Hugo site.
	// Defaults to "/" when rulebound.yml is absent.
	BaseURL string `yaml:"baseURL"`

	// Categories maps category names to slices of rule identifiers.
	// A rule may appear in multiple categories; the config layer does not deduplicate.
	Categories map[string][]string `yaml:"categories"`
}

// Load reads and parses the rulebound.yml file located in packageDir.
// If the file does not exist, Load returns a Config populated with defaults:
//   - Title: base name of packageDir
//   - BaseURL: "/"
//   - Categories: nil
//
// If the file exists but is malformed YAML, Load returns an error.
func Load(packageDir string) (*Config, error) {
	cfgPath := filepath.Join(packageDir, "rulebound.yml")

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaults(packageDir), nil
		}
		return nil, fmt.Errorf("reading rulebound.yml: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing rulebound.yml: %w", err)
	}

	// Fill in any omitted fields with defaults.
	if cfg.Title == "" {
		cfg.Title = filepath.Base(packageDir)
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "/"
	}

	return &cfg, nil
}

// defaults returns a Config built entirely from defaults for packageDir.
func defaults(packageDir string) *Config {
	return &Config{
		Title:   filepath.Base(packageDir),
		BaseURL: "/",
	}
}
