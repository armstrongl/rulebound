package parser

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"go.yaml.in/yaml/v3"
)

// maxSectionDepth is the maximum nesting depth for page sections.
// Directories deeper than this are flattened to the cap level.
const maxSectionDepth = 6

// pageFrontmatter holds the YAML structure that page .md files contain.
type pageFrontmatter struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
}

// parsePageFrontmatter extracts YAML frontmatter and body from a page Markdown
// string. It returns the title, description, and body text. It returns an error
// if no frontmatter is found or the YAML is malformed.
func parsePageFrontmatter(content string) (title, description, body string, err error) {
	const fence = "---"

	content = strings.ReplaceAll(content, "\r\n", "\n")

	if !strings.HasPrefix(content, fence) {
		return "", "", "", fmt.Errorf("no frontmatter found")
	}

	rest := content[len(fence):]
	if len(rest) == 0 || rest[0] != '\n' {
		return "", "", "", fmt.Errorf("no frontmatter found")
	}
	rest = rest[1:]

	idx := strings.Index(rest, "\n"+fence)
	if idx == -1 {
		return "", "", "", fmt.Errorf("no closing frontmatter fence")
	}

	fmRaw := rest[:idx]
	bodyRaw := rest[idx+1+len(fence):]
	if len(bodyRaw) > 0 && bodyRaw[0] == '\n' {
		bodyRaw = bodyRaw[1:]
	}
	bodyRaw = strings.TrimSpace(bodyRaw)

	var fm pageFrontmatter
	if err := yaml.Unmarshal([]byte(fmRaw), &fm); err != nil {
		return "", "", "", fmt.Errorf("parsing frontmatter YAML: %w", err)
	}

	return fm.Title, fm.Description, bodyRaw, nil
}

// parsePages is the entry point for parsing the pages/ directory within a Vale
// package directory. It returns nil if pages/ does not exist or is empty.
func parsePages(packageDir string) (*SectionTree, []ParseWarning, error) {
	pagesDir := filepath.Join(packageDir, "pages")

	info, err := os.Stat(pagesDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("checking pages directory: %w", err)
	}
	if !info.IsDir() {
		return nil, nil, nil
	}

	tree, warnings, err := parsePagesDir(pagesDir, "pages", "/pages/", 0)
	if err != nil {
		return nil, nil, err
	}

	// An empty pages/ directory (no pages, no children, no index) is treated as absent.
	if tree != nil && len(tree.Pages) == 0 && len(tree.Children) == 0 && tree.IndexPage == nil {
		return nil, warnings, nil
	}

	return tree, warnings, nil
}

// parsePagesDir recursively walks a directory under pages/, building a
// SectionTree from .md files, _meta.yml metadata, and subdirectories.
func parsePagesDir(dir string, name string, pathPrefix string, depth int) (*SectionTree, []ParseWarning, error) {
	var warnings []ParseWarning

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("reading pages directory %s: %w", dir, err)
	}

	tree := &SectionTree{
		Name: name,
		Path: pathPrefix,
	}

	// ── Parse _meta.yml if present ────────────────────────────────────────
	metaPath := filepath.Join(dir, "_meta.yml")
	if data, err := os.ReadFile(metaPath); err == nil {
		var meta SectionMeta
		if err := yaml.Unmarshal(data, &meta); err != nil {
			warnings = append(warnings, ParseWarning{
				File:    "_meta.yml",
				Message: fmt.Sprintf("malformed _meta.yml in %s: %v", dir, err),
			})
			// Use defaults — meta stays nil
		} else {
			tree.Meta = &meta
			if meta.Title != "" {
				tree.Title = meta.Title
			}
		}
	}

	// If title not set from _meta.yml, derive from directory name.
	if tree.Title == "" {
		tree.Title = kebabToTitle(name)
	}

	// ── Collect hidden set from meta ──────────────────────────────────────
	hiddenSet := make(map[string]bool)
	if tree.Meta != nil {
		for _, h := range tree.Meta.Hidden {
			hiddenSet[h] = true
		}
	}

	// ── Check for rules/ collision ────────────────────────────────────────
	hasRulesDir := false
	rulesInOrder := false
	if tree.Meta != nil {
		for _, o := range tree.Meta.Order {
			if o == "rules" {
				rulesInOrder = true
				break
			}
		}
	}

	// ── Scan entries ──────────────────────────────────────────────────────
	var pages []*Page
	var subdirs []os.DirEntry

	for _, entry := range entries {
		entryName := entry.Name()

		if entry.IsDir() {
			if entryName == "rules" {
				hasRulesDir = true
			}
			subdirs = append(subdirs, entry)
			continue
		}

		// Skip non-.md files silently
		if filepath.Ext(entryName) != ".md" {
			continue
		}

		// Skip _meta.yml (already handled above, but just in case of .md check)
		// Skip _index.md — handle separately
		if entryName == "_index.md" {
			filePath := filepath.Join(dir, entryName)
			data, err := os.ReadFile(filePath)
			if err != nil {
				warnings = append(warnings, ParseWarning{
					File:    entryName,
					Message: fmt.Sprintf("reading _index.md: %v", err),
				})
				continue
			}

			title, desc, body, err := parsePageFrontmatter(string(data))
			if err != nil {
				warnings = append(warnings, ParseWarning{
					File:    entryName,
					Message: fmt.Sprintf("parsing _index.md frontmatter: %v", err),
				})
				continue
			}

			if title == "" {
				title = kebabToTitle(name)
			}

			tree.IndexPage = &Page{
				Title:       title,
				Description: desc,
				Body:        body,
				SourceFile:  filePath,
				Path:        pathPrefix,
			}
			continue
		}

		// Regular .md file
		filePath := filepath.Join(dir, entryName)
		data, err := os.ReadFile(filePath)
		if err != nil {
			warnings = append(warnings, ParseWarning{
				File:    entryName,
				Message: fmt.Sprintf("reading page: %v", err),
			})
			continue
		}

		title, desc, body, err := parsePageFrontmatter(string(data))
		if err != nil {
			warnings = append(warnings, ParseWarning{
				File:    entryName,
				Message: fmt.Sprintf("parsing page frontmatter: %v", err),
			})
			continue
		}

		stem := strings.TrimSuffix(entryName, ".md")

		if title == "" {
			title = kebabToTitle(stem)
			warnings = append(warnings, ParseWarning{
				File:    entryName,
				Message: "page missing 'title' in frontmatter; derived from filename",
			})
		}

		page := &Page{
			Title:       title,
			Description: desc,
			Body:        body,
			SourceFile:  filePath,
			Path:        pathPrefix + stem + "/",
			Hidden:      hiddenSet[stem],
		}
		pages = append(pages, page)
	}

	// ── Emit rules/ collision warning ─────────────────────────────────────
	if rulesInOrder && hasRulesDir {
		warnings = append(warnings, ParseWarning{
			File:    "_meta.yml",
			Message: "rules/ directory collision: 'rules' in order list and pages/rules/ directory both exist; directory takes precedence",
		})
		// Remove the reserved "rules" keyword from the order list so navigation
		// generation doesn't treat it as the auto-generated rules section position.
		filtered := make([]string, 0, len(tree.Meta.Order))
		for _, item := range tree.Meta.Order {
			if item != "rules" {
				filtered = append(filtered, item)
			}
		}
		tree.Meta.Order = filtered
	}

	// ── Order pages ───────────────────────────────────────────────────────
	pages = orderPages(pages, tree.Meta)
	tree.Pages = pages

	// ── Recurse into subdirectories ───────────────────────────────────────
	// Sort subdirectories for deterministic order.
	sort.Slice(subdirs, func(i, j int) bool {
		return subdirs[i].Name() < subdirs[j].Name()
	})

	for _, sub := range subdirs {
		subName := sub.Name()
		subPath := pathPrefix + subName + "/"

		if depth+1 > maxSectionDepth {
			warnings = append(warnings, ParseWarning{
				File:    subName,
				Message: fmt.Sprintf("nesting depth exceeds %d levels; flattening to level %d", maxSectionDepth, maxSectionDepth),
			})
			// Flatten: still parse, but keep logical depth capped at maxSectionDepth.
			child, childWarnings, err := parsePagesDir(filepath.Join(dir, subName), subName, subPath, maxSectionDepth)
			if err != nil {
				return nil, nil, err
			}
			warnings = append(warnings, childWarnings...)
			if child != nil {
				tree.Children = append(tree.Children, child)
			}
			continue
		}

		child, childWarnings, err := parsePagesDir(filepath.Join(dir, subName), subName, subPath, depth+1)
		if err != nil {
			return nil, nil, err
		}
		warnings = append(warnings, childWarnings...)
		if child != nil {
			tree.Children = append(tree.Children, child)
		}
	}

	return tree, warnings, nil
}

// orderPages sorts pages according to the _meta.yml order list. Pages listed
// in order appear first (in that order); unlisted pages sort alphabetically
// after.
func orderPages(pages []*Page, meta *SectionMeta) []*Page {
	if meta == nil || len(meta.Order) == 0 {
		// Default: alphabetical by source file stem.
		sort.Slice(pages, func(i, j int) bool {
			return filepath.Base(pages[i].SourceFile) < filepath.Base(pages[j].SourceFile)
		})
		return pages
	}

	// Build position map from order list.
	pos := make(map[string]int)
	for i, name := range meta.Order {
		pos[name] = i
	}

	sort.SliceStable(pages, func(i, j int) bool {
		stemI := strings.TrimSuffix(filepath.Base(pages[i].SourceFile), ".md")
		stemJ := strings.TrimSuffix(filepath.Base(pages[j].SourceFile), ".md")

		posI, inI := pos[stemI]
		posJ, inJ := pos[stemJ]

		switch {
		case inI && inJ:
			return posI < posJ
		case inI:
			return true
		case inJ:
			return false
		default:
			return stemI < stemJ
		}
	})

	return pages
}

// kebabToTitle converts a kebab-case string to Title Case.
// For example, "active-voice" becomes "Active Voice".
func kebabToTitle(s string) string {
	if s == "" {
		return ""
	}

	parts := strings.Split(s, "-")
	for i, part := range parts {
		if len(part) == 0 {
			continue
		}
		runes := []rune(part)
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, " ")
}
