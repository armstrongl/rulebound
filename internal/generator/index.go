package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"

	"github.com/larah/rulebound/internal/config"
	"github.com/larah/rulebound/internal/parser"
)

// homepageIndexData is the frontmatter for content/_index.md.
type homepageIndexData struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description,omitempty"`
	TotalRules  int    `yaml:"total_rules"`
}

// rulesIndexData is the frontmatter for content/rules/_index.md.
type rulesIndexData struct {
	Title      string         `yaml:"title"`
	TotalRules int            `yaml:"total_rules"`
	ByType     map[string]int `yaml:"by_type"`
	BySeverity map[string]int `yaml:"by_severity"`
	ByCategory map[string]int `yaml:"by_category"`
}

// siteStats is the data written to data/site.json.
type siteStats struct {
	TotalRules int            `json:"total_rules"`
	ByType     map[string]int `json:"by_type"`
	BySeverity map[string]int `json:"by_severity"`
	ByCategory map[string]int `json:"by_category"`
}

// generateHomepageIndex writes content/_index.md.
func generateHomepageIndex(cfg *config.Config, rules []*parser.ValeRule, outputDir string) error {
	data := homepageIndexData{
		Title:       cfg.Title,
		Description: cfg.Description,
		TotalRules:  len(rules),
	}

	out, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling homepage index: %w", err)
	}

	content := "---\n" + string(out) + "---\n"
	path := filepath.Join(outputDir, "content", "_index.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing content/_index.md: %w", err)
	}
	return nil
}

// generateRulesIndex writes content/rules/_index.md with aggregated counts.
func generateRulesIndex(rules []*parser.ValeRule, rulesDir string) error {
	byType, bySeverity, byCategory := aggregateCounts(rules)

	data := rulesIndexData{
		Title:      "Rules",
		TotalRules: len(rules),
		ByType:     byType,
		BySeverity: bySeverity,
		ByCategory: byCategory,
	}

	out, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling rules index: %w", err)
	}

	content := "---\n" + string(out) + "---\n"
	path := filepath.Join(rulesDir, "_index.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing rules/_index.md: %w", err)
	}
	return nil
}

// generateSiteJSON writes data/site.json with aggregated statistics.
func generateSiteJSON(rules []*parser.ValeRule, dataDir string) error {
	byType, bySeverity, byCategory := aggregateCounts(rules)

	stats := siteStats{
		TotalRules: len(rules),
		ByType:     byType,
		BySeverity: bySeverity,
		ByCategory: byCategory,
	}

	out, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling site.json: %w", err)
	}

	path := filepath.Join(dataDir, "site.json")
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("writing data/site.json: %w", err)
	}
	return nil
}

// aggregateCounts returns counts by rule type, severity, and category.
func aggregateCounts(rules []*parser.ValeRule) (byType, bySeverity, byCategory map[string]int) {
	byType = make(map[string]int)
	bySeverity = make(map[string]int)
	byCategory = make(map[string]int)

	for _, rule := range rules {
		byType[rule.Extends]++
		bySeverity[rule.Level]++
		for _, cat := range categoriesFromRule(rule) {
			byCategory[cat]++
		}
	}
	return
}
