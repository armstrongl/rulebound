package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/larah/rulebound/internal/config"
	"github.com/larah/rulebound/internal/parser"
)

// GenerateRule writes a single Hugo content page for the given rule to outDir.
// The filename is the lowercase rule name with a .md extension.
// Content format: YAML frontmatter (---) followed by companion prose or an
// auto-generated description.
func GenerateRule(rule *parser.ValeRule, outDir string) error {
	fm, err := BuildFrontmatter(rule)
	if err != nil {
		return err
	}

	body := rule.CompanionMD
	if body == "" {
		body = AutoDescription(rule)
	}

	content := "---\n" + fm + "---\n\n" + body + "\n"

	filename := strings.ToLower(rule.Name) + ".md"
	path := filepath.Join(outDir, filename)

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing rule page %s: %w", path, err)
	}
	return nil
}

// GenerateSite orchestrates full site generation into outputDir.
// It creates the Hugo project structure:
//
//	outputDir/
//	├── hugo.toml
//	├── content/
//	│   ├── _index.md
//	│   └── rules/
//	│       ├── _index.md
//	│       └── <rule>.md  (one per rule)
//	└── data/
//	    └── site.json
func GenerateSite(result *parser.ParseResult, cfg *config.Config, outputDir string) error {
	rules := result.Rules

	// Assign categories from config before generating content.
	AssignCategories(rules, cfg)

	// Create directory structure.
	rulesDir := filepath.Join(outputDir, "content", "rules")
	dataDir := filepath.Join(outputDir, "data")
	for _, dir := range []string{
		filepath.Join(outputDir, "content"),
		rulesDir,
		dataDir,
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	// hugo.toml
	if err := generateHugoTOML(cfg, outputDir); err != nil {
		return err
	}

	// ── Guidelines ───────────────────────────────────────────────────────
	// Check if guidelines are enabled (default: true when Enabled is nil).
	guidelinesEnabled := cfg.Guidelines.Enabled == nil || *cfg.Guidelines.Enabled
	guidelines := result.Guidelines
	guidelinesCount := 0

	if guidelinesEnabled && len(guidelines) > 0 {
		// Apply ordering and exclusion from config.
		var guidelineWarnings []parser.ParseWarning
		guidelines, guidelineWarnings = applyGuidelinesConfig(guidelines, cfg.Guidelines)
		// Log config validation warnings (e.g., unmatched order stems) to stderr.
		for _, w := range guidelineWarnings {
			fmt.Fprintf(os.Stderr, "Warning: guidelines config: %s: %s\n", w.File, w.Message)
		}
		guidelinesCount = len(guidelines)

		if len(guidelines) > 0 {
			guidelinesDir := filepath.Join(outputDir, "content", "guidelines")
			if err := os.MkdirAll(guidelinesDir, 0o755); err != nil {
				return fmt.Errorf("creating guidelines directory: %w", err)
			}

			sectionTitle := cfg.Guidelines.SectionTitle
			if err := generateGuidelinesIndex(sectionTitle, guidelinesDir); err != nil {
				return err
			}

			for _, g := range guidelines {
				if err := GenerateGuideline(g, guidelinesDir); err != nil {
					return err
				}
			}
		}
	}

	sectionTitle := cfg.Guidelines.SectionTitle

	// content/_index.md (homepage)
	if err := generateHomepageIndex(cfg, rules, guidelinesCount, outputDir); err != nil {
		return err
	}

	// content/rules/_index.md
	if err := generateRulesIndex(rules, rulesDir); err != nil {
		return err
	}

	// data/site.json
	if err := generateSiteJSON(rules, guidelinesCount, sectionTitle, dataDir); err != nil {
		return err
	}

	// One content page per rule.
	for _, rule := range rules {
		if err := GenerateRule(rule, rulesDir); err != nil {
			return err
		}
	}

	return nil
}

// AssignCategories applies category assignments from config to the rule slice.
// A rule listed under multiple categories receives a comma-separated Category
// string. Rules not referenced in any config category fall back to their
// Extends type.
func AssignCategories(rules []*parser.ValeRule, cfg *config.Config) {
	// Build rule-name → []category mapping.
	catsByRule := make(map[string][]string)
	for catName, ruleNames := range cfg.Categories {
		for _, ruleName := range ruleNames {
			catsByRule[ruleName] = append(catsByRule[ruleName], catName)
		}
	}

	for _, rule := range rules {
		if cats, ok := catsByRule[rule.Name]; ok {
			sort.Strings(cats)
			rule.Category = strings.Join(cats, ",")
		} else {
			rule.Category = rule.Extends
		}
	}
}

// generateHugoTOML writes hugo.toml from config values.
func generateHugoTOML(cfg *config.Config, outputDir string) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("baseURL = %q\n", cfg.BaseURL))
	sb.WriteString(fmt.Sprintf("title = %q\n", cfg.Title))
	if cfg.Description != "" {
		sb.WriteString(fmt.Sprintf("description = %q\n", cfg.Description))
	}
	sb.WriteString("\n[taxonomies]\n")
	sb.WriteString("  category = \"categories\"\n")
	sb.WriteString("  ruletype = \"ruletypes\"\n")
	sb.WriteString("  severity = \"severities\"\n")

	path := filepath.Join(outputDir, "hugo.toml")
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		return fmt.Errorf("writing hugo.toml: %w", err)
	}
	return nil
}
