package parser

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// guidelineFrontmatter is the YAML structure expected in guideline .md files.
type guidelineFrontmatter struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Weight      int    `yaml:"weight"`
}

// parseFrontmatter extracts YAML frontmatter and body from a Markdown string.
// Returns an error if no frontmatter is found or YAML is malformed.
func parseFrontmatter(content string) (*guidelineFrontmatter, string, error) {
	const fence = "---"

	content = strings.ReplaceAll(content, "\r\n", "\n")

	if !strings.HasPrefix(content, fence) {
		return nil, "", fmt.Errorf("no frontmatter found")
	}

	rest := content[len(fence):]
	if len(rest) == 0 || rest[0] != '\n' {
		return nil, "", fmt.Errorf("no frontmatter found")
	}
	rest = rest[1:]

	idx := strings.Index(rest, "\n"+fence)
	if idx == -1 {
		return nil, "", fmt.Errorf("no closing frontmatter fence")
	}

	fmRaw := rest[:idx]
	body := rest[idx+1+len(fence):]
	if len(body) > 0 && body[0] == '\n' {
		body = body[1:]
	}
	body = strings.TrimSpace(body)

	var fm guidelineFrontmatter
	if err := yaml.Unmarshal([]byte(fmRaw), &fm); err != nil {
		return nil, "", fmt.Errorf("parsing frontmatter YAML: %w", err)
	}

	return &fm, body, nil
}

// parseGuidelines reads all .md files from the guidelines/ subdirectory of
// packageDir. Returns parsed guidelines and any non-fatal warnings.
// If guidelines/ does not exist, returns empty results without error.
func parseGuidelines(packageDir string) ([]*Guideline, []ParseWarning, error) {
	guidelinesDir := filepath.Join(packageDir, "guidelines")

	info, err := os.Stat(guidelinesDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("checking guidelines directory: %w", err)
	}
	if !info.IsDir() {
		return nil, nil, nil
	}

	entries, err := os.ReadDir(guidelinesDir)
	if err != nil {
		return nil, nil, fmt.Errorf("reading guidelines directory: %w", err)
	}

	var guidelines []*Guideline
	var warnings []ParseWarning

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".md" {
			continue
		}

		filePath := filepath.Join(guidelinesDir, name)
		data, err := os.ReadFile(filePath)
		if err != nil {
			warnings = append(warnings, ParseWarning{
				File:    name,
				Message: fmt.Sprintf("reading guideline: %v", err),
			})
			continue
		}

		fm, body, err := parseFrontmatter(string(data))
		if err != nil {
			warnings = append(warnings, ParseWarning{
				File:    name,
				Message: fmt.Sprintf("parsing guideline frontmatter: %v", err),
			})
			continue
		}

		if fm.Title == "" {
			warnings = append(warnings, ParseWarning{
				File:    name,
				Message: "guideline missing required 'title' in frontmatter",
			})
			continue
		}

		stem := strings.TrimSuffix(name, ".md")
		guidelines = append(guidelines, &Guideline{
			Name:        stem,
			Title:       fm.Title,
			Description: fm.Description,
			Weight:      fm.Weight,
			Body:        body,
			SourceFile:  filePath,
		})
	}

	sort.Slice(guidelines, func(i, j int) bool {
		return guidelines[i].Name < guidelines[j].Name
	})

	return guidelines, warnings, nil
}
