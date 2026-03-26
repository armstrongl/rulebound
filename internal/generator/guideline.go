package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"go.yaml.in/yaml/v3"

	"github.com/larah/rulebound/internal/config"
	"github.com/larah/rulebound/internal/parser"
)

// guidelineFrontmatter is the Hugo frontmatter for a guideline page.
type guidelineFrontmatter struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description,omitempty"`
	Weight      int    `yaml:"weight"`
	Type        string `yaml:"type"`
}

// GenerateGuideline writes a single Hugo content page for a guideline.
func GenerateGuideline(g *parser.Guideline, outDir string) error {
	fm := guidelineFrontmatter{
		Title:       g.Title,
		Description: g.Description,
		Weight:      g.Weight,
		Type:        "guideline",
	}

	out, err := yaml.Marshal(fm)
	if err != nil {
		return fmt.Errorf("marshaling guideline frontmatter for %s: %w", g.Name, err)
	}

	content := "---\n" + string(out) + "---\n"
	if g.Body != "" {
		content += "\n" + g.Body + "\n"
	}

	path := filepath.Join(outDir, g.Name+".md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing guideline page %s: %w", path, err)
	}
	return nil
}

// generateGuidelinesIndex writes content/guidelines/_index.md.
// The type: guideline frontmatter tells Hugo to use layouts/guideline/list.html.
func generateGuidelinesIndex(sectionTitle string, guidelinesDir string) error {
	if sectionTitle == "" {
		sectionTitle = "Guidelines"
	}

	type guidelinesIndexData struct {
		Title string `yaml:"title"`
		Type  string `yaml:"type"`
	}

	data := guidelinesIndexData{Title: sectionTitle, Type: "guideline"}
	out, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling guidelines index: %w", err)
	}

	content := "---\n" + string(out) + "---\n"
	path := filepath.Join(guidelinesDir, "_index.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing guidelines/_index.md: %w", err)
	}
	return nil
}

// applyGuidelinesConfig filters and reweights guidelines based on config.
// It returns the processed guidelines and any warnings (for example, unmatched order stems).
func applyGuidelinesConfig(guidelines []*parser.Guideline, cfg config.GuidelinesConfig) ([]*parser.Guideline, []parser.ParseWarning) {
	var warnings []parser.ParseWarning

	// Build exclude set.
	excludeSet := make(map[string]bool, len(cfg.Exclude))
	for _, stem := range cfg.Exclude {
		excludeSet[stem] = true
	}

	// Filter out excluded guidelines.
	var filtered []*parser.Guideline
	for _, g := range guidelines {
		if !excludeSet[g.Name] {
			filtered = append(filtered, g)
		}
	}

	// Build name set for order validation.
	nameSet := make(map[string]bool, len(filtered))
	for _, g := range filtered {
		nameSet[g.Name] = true
	}

	// Apply synthetic weights from order list.
	for i, stem := range cfg.Order {
		if excludeSet[stem] {
			continue // excluded stems are silently skipped in order
		}
		if !nameSet[stem] {
			warnings = append(warnings, parser.ParseWarning{
				File:    stem,
				Message: "stem in guidelines.order does not match any guideline file",
			})
			continue
		}
		for _, g := range filtered {
			if g.Name == stem {
				g.Weight = -10000 + i
				break
			}
		}
	}

	// Sort: by weight first, then alphabetically by stem name for tie-breaking.
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Weight != filtered[j].Weight {
			return filtered[i].Weight < filtered[j].Weight
		}
		return filtered[i].Name < filtered[j].Name
	})

	return filtered, warnings
}
