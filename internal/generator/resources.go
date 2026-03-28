package generator

import (
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"

	"github.com/armstrongl/rulebound/internal/config"
)

// resourceLink is the JSON-serializable form of a resource link for site.json.
type resourceLink struct {
	Label       string `json:"label"`
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
	Footer      bool   `json:"footer"`
}

// defaultResourceLinks returns the hardcoded Vale ecosystem links.
func defaultResourceLinks() []resourceLink {
	return []resourceLink{
		{
			Label:       "Vale",
			URL:         "https://vale.sh",
			Description: "A linter for prose — write with style.",
			Footer:      true,
		},
		{
			Label:       "Vale Studio",
			URL:         "https://studio.vale.sh",
			Description: "Test Vale rules in the browser.",
			Footer:      true,
		},
		{
			Label:       "Vale on GitHub",
			URL:         "https://github.com/vale-cli/vale",
			Description: "Source code, issues, and releases.",
			Footer:      false,
		},
	}
}

// buildResourceLinks merges defaults with extra links from config.
func buildResourceLinks(cfg *config.Config) []resourceLink {
	links := defaultResourceLinks()
	for _, extra := range cfg.Resources.ExtraLinks {
		links = append(links, resourceLink{
			Label:       extra.Label,
			URL:         extra.URL,
			Description: extra.Description,
			Footer:      false,
		})
	}
	return links
}

// resourcesIndexData is the frontmatter for content/resources/_index.md.
type resourcesIndexData struct {
	Title string `yaml:"title"`
	Type  string `yaml:"type"`
}

// generateResourcesPage writes content/resources/_index.md.
func generateResourcesPage(outputDir string) error {
	resourcesDir := filepath.Join(outputDir, "content", "resources")
	if err := os.MkdirAll(resourcesDir, 0o755); err != nil {
		return fmt.Errorf("creating resources directory: %w", err)
	}

	data := resourcesIndexData{Title: "Resources", Type: "resources"}
	out, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling resources index: %w", err)
	}

	content := "---\n" + string(out) + "---\n"
	path := filepath.Join(resourcesDir, "_index.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing resources/_index.md: %w", err)
	}
	return nil
}
