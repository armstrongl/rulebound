// Package config loads and parses rulebound.yml configuration files.
// The config file is optional; when absent, the loader applies defaults.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// Config represents the parsed contents of a rulebound.yml file.
type Config struct {
	// Title is the human-readable name of the style guide.
	// Defaults to the package directory name when rulebound.yml is absent or omits it.
	Title string `yaml:"title"`

	// Description is an optional short description of the style guide.
	Description string `yaml:"description"`

	// BaseURL is the base URL for the generated Hugo site.
	// Defaults to "/" when rulebound.yml is absent or omits it.
	BaseURL string `yaml:"baseURL"`

	// Categories maps category names to slices of rule identifiers.
	// A rule may appear in multiple categories; the config layer does not deduplicate.
	Categories map[string][]string `yaml:"categories"`

	// Guidelines controls how editorial guidelines are processed.
	Guidelines GuidelinesConfig `yaml:"guidelines"`

	// Pages controls how content pages are processed.
	Pages PagesConfig `yaml:"pages"`
}

// GuidelinesConfig controls how the build processes editorial guidelines.
type GuidelinesConfig struct {
	// SectionTitle overrides the sidebar heading. Default: "Guidelines".
	SectionTitle string `yaml:"section_title"`
	// Order defines explicit page ordering by stem name.
	Order []string `yaml:"order"`
	// Exclude skips specific files by stem name (takes precedence over order).
	Exclude []string `yaml:"exclude"`
	// Enabled controls auto-detection. Default: true (nil means true).
	Enabled *bool `yaml:"enabled"`
}

// PagesConfig controls how the build processes content pages.
type PagesConfig struct {
	// Enabled controls auto-detection of the pages/ directory. Default: true (nil means true).
	Enabled *bool `yaml:"enabled"`
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
	return loadFromPath(cfgPath, packageDir)
}

// LoadFile reads and parses a rulebound.yml from an explicit file path.
// If the file does not exist, it returns an error (unlike Load, which defaults).
func LoadFile(cfgPath string) (*Config, error) {
	packageDir := filepath.Dir(cfgPath)
	return loadFromPath(cfgPath, packageDir)
}

func loadFromPath(cfgPath, packageDir string) (*Config, error) {
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
