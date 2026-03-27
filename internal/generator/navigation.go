package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/larah/rulebound/internal/parser"
)

// ── JSON serialization types ─────────────────────────────────────────────────

// navigationData is the top-level structure written to data/navigation.json.
type navigationData struct {
	Sections     []navSection     `json:"sections"`
	RulesSection navRulesSection  `json:"rules_section"`
}

// navSection represents a content section in the sidebar.
type navSection struct {
	Name      string       `json:"name"`
	Title     string       `json:"title"`
	Path      string       `json:"path"`
	Collapsed bool         `json:"collapsed"`
	Pages     []navPage    `json:"pages"`
	Children  []navSection `json:"children"`
}

// navPage represents a single content page in the sidebar.
type navPage struct {
	Title string `json:"title"`
	Path  string `json:"path"`
}

// navRulesSection represents the auto-generated rules section in the sidebar.
type navRulesSection struct {
	Title      string        `json:"title"`
	Position   int           `json:"position"`
	Categories []navCategory `json:"categories"`
}

// navCategory represents a category grouping of rules.
type navCategory struct {
	Name  string    `json:"name"`
	Title string    `json:"title"`
	Rules []navRule `json:"rules"`
}

// navRule represents a single rule in the sidebar.
type navRule struct {
	Title string `json:"title"`
	Path  string `json:"path"`
	Level string `json:"level,omitempty"`
}

// ── Public functions ─────────────────────────────────────────────────────────

// SectionTreeIsEmpty returns true if the tree has no content at any level:
// no pages, no children with pages, and no IndexPage.
func SectionTreeIsEmpty(tree *parser.SectionTree) bool {
	if tree == nil {
		return true
	}
	if tree.IndexPage != nil {
		return false
	}
	if len(tree.Pages) > 0 {
		return false
	}
	for _, child := range tree.Children {
		if !SectionTreeIsEmpty(child) {
			return false
		}
	}
	return true
}

// CountPages returns the total number of content pages in a SectionTree,
// counting pages at every level plus index pages. Returns 0 for nil trees.
func CountPages(tree *parser.SectionTree) int {
	if tree == nil {
		return 0
	}
	count := len(tree.Pages)
	if tree.IndexPage != nil {
		count++
	}
	for _, child := range tree.Children {
		count += CountPages(child)
	}
	return count
}

// GenerateNavigationJSON builds and writes data/navigation.json from the
// SectionTree and rules. When pages is nil or empty, no file is written.
func GenerateNavigationJSON(pages *parser.SectionTree, rules []*parser.ValeRule, categories map[string][]string, dataDir string) error {
	if SectionTreeIsEmpty(pages) {
		return nil
	}

	// Build sections from the tree's children (each child is a top-level section).
	sections := make([]navSection, 0, len(pages.Children))
	for _, child := range pages.Children {
		sections = append(sections, buildNavSection(child))
	}

	// Build rules section.
	rulesSection := buildRulesSection(pages, rules, categories)

	data := navigationData{
		Sections:     sections,
		RulesSection: rulesSection,
	}

	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling navigation.json: %w", err)
	}

	path := filepath.Join(dataDir, "navigation.json")
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("writing navigation.json: %w", err)
	}
	return nil
}

// ── Internal helpers ─────────────────────────────────────────────────────────

// buildNavSection recursively converts a SectionTree node into a navSection,
// filtering out hidden pages.
func buildNavSection(tree *parser.SectionTree) navSection {
	section := navSection{
		Name:  tree.Name,
		Title: tree.Title,
		Path:  tree.Path,
	}

	// Check collapsed from Meta.
	if tree.Meta != nil {
		section.Collapsed = tree.Meta.Collapsed
	}

	// Collect visible pages.
	pages := make([]navPage, 0, len(tree.Pages))
	for _, p := range tree.Pages {
		if p.Hidden {
			continue
		}
		pages = append(pages, navPage{
			Title: p.Title,
			Path:  p.Path,
		})
	}
	section.Pages = pages

	// Recurse into children.
	children := make([]navSection, 0, len(tree.Children))
	for _, child := range tree.Children {
		children = append(children, buildNavSection(child))
	}
	section.Children = children

	return section
}

// buildRulesSection constructs the navRulesSection from rules and categories.
func buildRulesSection(pages *parser.SectionTree, rules []*parser.ValeRule, categories map[string][]string) navRulesSection {
	// Determine title.
	title := "Rules"
	if pages.Meta != nil && pages.Meta.RulesTitle != "" {
		title = pages.Meta.RulesTitle
	}

	// Determine position from the order list.
	position := -1
	if pages.Meta != nil {
		for i, item := range pages.Meta.Order {
			if item == "rules" {
				position = i
				break
			}
		}
	}

	// Clamp position if it exceeds actual section count.
	if position > len(pages.Children) {
		position = -1
	}

	// Build rule lookup by name for fast access.
	ruleByName := make(map[string]*parser.ValeRule, len(rules))
	for _, r := range rules {
		ruleByName[r.Name] = r
	}

	// Build categories list, sorted alphabetically by category name.
	catNames := make([]string, 0, len(categories))
	for catName := range categories {
		catNames = append(catNames, catName)
	}
	sort.Strings(catNames)

	navCats := make([]navCategory, 0, len(catNames))
	for _, catName := range catNames {
		ruleNames := categories[catName]

		// Sort rules within category alphabetically.
		sorted := make([]string, len(ruleNames))
		copy(sorted, ruleNames)
		sort.Strings(sorted)

		navRules := make([]navRule, 0, len(sorted))
		for _, ruleName := range sorted {
			r, ok := ruleByName[ruleName]
			if !ok {
				continue
			}
			navRules = append(navRules, navRule{
				Title: DisplayName(r.Name),
				Path:  "/rules/" + strings.ToLower(r.Name) + "/",
				Level: r.Level,
			})
		}

		navCats = append(navCats, navCategory{
			Name:  catName,
			Title: catName,
			Rules: navRules,
		})
	}

	return navRulesSection{
		Title:      title,
		Position:   position,
		Categories: navCats,
	}
}
