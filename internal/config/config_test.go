package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/larah/rulebound/internal/config"
)

// TestLoadValidConfig verifies that a well-formed rulebound.yml is parsed correctly.
func TestLoadValidConfig(t *testing.T) {
	dir := t.TempDir()
	yml := `title: My Style Guide
description: A test style guide
baseURL: https://example.com/
categories:
  punctuation:
    - Punctuation.Comma
    - Punctuation.Period
  casing:
    - Casing.HeadingTitle
`
	if err := os.WriteFile(filepath.Join(dir, "rulebound.yml"), []byte(yml), 0o644); err != nil {
		t.Fatalf("setup: write config file: %v", err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	if cfg.Title != "My Style Guide" {
		t.Errorf("Title = %q, want %q", cfg.Title, "My Style Guide")
	}
	if cfg.Description != "A test style guide" {
		t.Errorf("Description = %q, want %q", cfg.Description, "A test style guide")
	}
	if cfg.BaseURL != "https://example.com/" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "https://example.com/")
	}
	if len(cfg.Categories) != 2 {
		t.Fatalf("Categories length = %d, want 2", len(cfg.Categories))
	}
	if len(cfg.Categories["punctuation"]) != 2 {
		t.Errorf("punctuation category length = %d, want 2", len(cfg.Categories["punctuation"]))
	}
	if len(cfg.Categories["casing"]) != 1 {
		t.Errorf("casing category length = %d, want 1", len(cfg.Categories["casing"]))
	}
}

// TestLoadMissingConfigUsesDefaults verifies that when rulebound.yml is absent,
// sensible defaults are applied (title from dir name, baseURL "/", no categories).
func TestLoadMissingConfigUsesDefaults(t *testing.T) {
	dir := t.TempDir()

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load() returned unexpected error when config is absent: %v", err)
	}

	expectedTitle := filepath.Base(dir)
	if cfg.Title != expectedTitle {
		t.Errorf("Title = %q, want directory base name %q", cfg.Title, expectedTitle)
	}
	if cfg.BaseURL != "/" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "/")
	}
	if cfg.Categories != nil && len(cfg.Categories) != 0 {
		t.Errorf("Categories = %v, want nil/empty map", cfg.Categories)
	}
}

// TestLoadMalformedConfigReturnsError verifies that invalid YAML returns an error.
func TestLoadMalformedConfigReturnsError(t *testing.T) {
	dir := t.TempDir()
	malformed := `title: [unclosed bracket
description: oops
`
	if err := os.WriteFile(filepath.Join(dir, "rulebound.yml"), []byte(malformed), 0o644); err != nil {
		t.Fatalf("setup: write malformed config: %v", err)
	}

	_, err := config.Load(dir)
	if err == nil {
		t.Fatal("Load() expected an error for malformed YAML, got nil")
	}
}

// TestLoadEmptyCategories verifies that an explicit empty categories map is valid.
func TestLoadEmptyCategories(t *testing.T) {
	dir := t.TempDir()
	yml := `title: Empty Cats
baseURL: /
categories: {}
`
	if err := os.WriteFile(filepath.Join(dir, "rulebound.yml"), []byte(yml), 0o644); err != nil {
		t.Fatalf("setup: write config: %v", err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}
	if len(cfg.Categories) != 0 {
		t.Errorf("Categories length = %d, want 0", len(cfg.Categories))
	}
}

// TestLoadOverlappingCategories verifies that the same rule in multiple categories
// is accepted without error — deduplication is not the config layer's job.
func TestLoadOverlappingCategories(t *testing.T) {
	dir := t.TempDir()
	yml := `title: Overlap Test
baseURL: /
categories:
  alpha:
    - Rules.Shared
    - Rules.AlphaOnly
  beta:
    - Rules.Shared
    - Rules.BetaOnly
`
	if err := os.WriteFile(filepath.Join(dir, "rulebound.yml"), []byte(yml), 0o644); err != nil {
		t.Fatalf("setup: write config: %v", err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}
	if len(cfg.Categories) != 2 {
		t.Fatalf("Categories length = %d, want 2", len(cfg.Categories))
	}

	alphaRules := cfg.Categories["alpha"]
	betaRules := cfg.Categories["beta"]

	sharedInAlpha := false
	for _, r := range alphaRules {
		if r == "Rules.Shared" {
			sharedInAlpha = true
		}
	}
	sharedInBeta := false
	for _, r := range betaRules {
		if r == "Rules.Shared" {
			sharedInBeta = true
		}
	}

	if !sharedInAlpha {
		t.Error("Rules.Shared not found in alpha category")
	}
	if !sharedInBeta {
		t.Error("Rules.Shared not found in beta category")
	}
}

// ── LoadFile tests ──────────────────────────────────────────────────────────

func TestLoadFileValidConfig(t *testing.T) {
	dir := t.TempDir()
	yml := `title: File Test
baseURL: https://example.com/
`
	cfgPath := filepath.Join(dir, "rulebound.yml")
	if err := os.WriteFile(cfgPath, []byte(yml), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	cfg, err := config.LoadFile(cfgPath)
	if err != nil {
		t.Fatalf("LoadFile() returned unexpected error: %v", err)
	}
	if cfg.Title != "File Test" {
		t.Errorf("Title = %q, want %q", cfg.Title, "File Test")
	}
}

func TestLoadFileNonExistentReturnsDefaults(t *testing.T) {
	cfg, err := config.LoadFile("/does/not/exist/rulebound.yml")
	if err != nil {
		t.Fatalf("LoadFile() returned error for nonexistent file: %v", err)
	}
	// LoadFile falls through to loadFromPath which returns defaults when file not found
	if cfg.BaseURL != "/" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "/")
	}
}

// ── Guidelines config tests ──────────────────────────────────────────────────

func TestLoadGuidelinesConfig(t *testing.T) {
	dir := t.TempDir()
	yml := `title: My Style Guide
baseURL: /
guidelines:
  section_title: Editorial Guidelines
  order:
    - voice-and-tone
    - inclusive-language
  exclude:
    - draft-notes
  enabled: true
`
	if err := os.WriteFile(filepath.Join(dir, "rulebound.yml"), []byte(yml), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	if cfg.Guidelines.SectionTitle != "Editorial Guidelines" {
		t.Errorf("SectionTitle = %q, want %q", cfg.Guidelines.SectionTitle, "Editorial Guidelines")
	}
	if len(cfg.Guidelines.Order) != 2 {
		t.Fatalf("Order length = %d, want 2", len(cfg.Guidelines.Order))
	}
	if cfg.Guidelines.Order[0] != "voice-and-tone" {
		t.Errorf("Order[0] = %q, want %q", cfg.Guidelines.Order[0], "voice-and-tone")
	}
	if len(cfg.Guidelines.Exclude) != 1 || cfg.Guidelines.Exclude[0] != "draft-notes" {
		t.Errorf("Exclude = %v, want [draft-notes]", cfg.Guidelines.Exclude)
	}
	if cfg.Guidelines.Enabled == nil || !*cfg.Guidelines.Enabled {
		t.Error("Enabled should be true")
	}
}

func TestLoadGuidelinesConfig_DefaultsWhenAbsent(t *testing.T) {
	dir := t.TempDir()
	yml := `title: No Guidelines Config
baseURL: /
`
	if err := os.WriteFile(filepath.Join(dir, "rulebound.yml"), []byte(yml), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	if cfg.Guidelines.Enabled != nil {
		t.Errorf("Enabled should be nil (unset), got %v", *cfg.Guidelines.Enabled)
	}
	if cfg.Guidelines.SectionTitle != "" {
		t.Errorf("SectionTitle should be empty, got %q", cfg.Guidelines.SectionTitle)
	}
	if len(cfg.Guidelines.Order) != 0 {
		t.Errorf("Order should be empty, got %v", cfg.Guidelines.Order)
	}
}

func TestLoadGuidelinesConfig_ExplicitlyDisabled(t *testing.T) {
	dir := t.TempDir()
	yml := `title: Disabled Guidelines
baseURL: /
guidelines:
  enabled: false
`
	if err := os.WriteFile(filepath.Join(dir, "rulebound.yml"), []byte(yml), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	if cfg.Guidelines.Enabled == nil {
		t.Fatal("Enabled should not be nil")
	}
	if *cfg.Guidelines.Enabled {
		t.Error("Enabled should be false")
	}
}

// ── LoadFile tests ──────────────────────────────────────────────────────────

func TestLoadFileMalformedReturnsError(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "bad.yml")
	if err := os.WriteFile(cfgPath, []byte("title: [unclosed"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err := config.LoadFile(cfgPath)
	if err == nil {
		t.Fatal("LoadFile() expected error for malformed YAML")
	}
}
