package generator_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/larah/rulebound/internal/generator"
	"github.com/larah/rulebound/internal/parser"
)

// ── sectionTreeIsEmpty ────────────────────────────────────────────────────────

func TestSectionTreeIsEmpty_NilTree(t *testing.T) {
	if !generator.SectionTreeIsEmpty(nil) {
		t.Error("nil tree should be empty")
	}
}

func TestSectionTreeIsEmpty_EmptyTree(t *testing.T) {
	tree := &parser.SectionTree{Name: "pages", Title: "Pages", Path: "/pages/"}
	if !generator.SectionTreeIsEmpty(tree) {
		t.Error("tree with no pages and no children should be empty")
	}
}

func TestSectionTreeIsEmpty_TreeWithPages(t *testing.T) {
	tree := &parser.SectionTree{
		Name:  "pages",
		Title: "Pages",
		Path:  "/pages/",
		Pages: []*parser.Page{
			{Title: "A Page", Path: "/pages/a-page/"},
		},
	}
	if generator.SectionTreeIsEmpty(tree) {
		t.Error("tree with pages should not be empty")
	}
}

func TestSectionTreeIsEmpty_TreeWithIndexPage(t *testing.T) {
	tree := &parser.SectionTree{
		Name:      "pages",
		Title:     "Pages",
		Path:      "/pages/",
		IndexPage: &parser.Page{Title: "Hub", Path: "/pages/"},
	}
	if generator.SectionTreeIsEmpty(tree) {
		t.Error("tree with IndexPage should not be empty")
	}
}

func TestSectionTreeIsEmpty_TreeWithNestedPages(t *testing.T) {
	tree := &parser.SectionTree{
		Name:  "pages",
		Title: "Pages",
		Path:  "/pages/",
		Children: []*parser.SectionTree{
			{
				Name:  "language",
				Title: "Language",
				Path:  "/pages/language/",
				Pages: []*parser.Page{
					{Title: "Active Voice", Path: "/pages/language/active-voice/"},
				},
			},
		},
	}
	if generator.SectionTreeIsEmpty(tree) {
		t.Error("tree with nested pages should not be empty")
	}
}

func TestSectionTreeIsEmpty_NestedEmptyChildren(t *testing.T) {
	tree := &parser.SectionTree{
		Name:  "pages",
		Title: "Pages",
		Path:  "/pages/",
		Children: []*parser.SectionTree{
			{
				Name:  "language",
				Title: "Language",
				Path:  "/pages/language/",
				// No pages, no children, no IndexPage
			},
		},
	}
	if !generator.SectionTreeIsEmpty(tree) {
		t.Error("tree with only empty children should be empty")
	}
}

// ── generateNavigationJSON ────────────────────────────────────────────────────

func TestGenerateNavigationJSON_SimpleTwoSections(t *testing.T) {
	dataDir := t.TempDir()

	tree := &parser.SectionTree{
		Name:  "pages",
		Title: "Pages",
		Path:  "/pages/",
		Children: []*parser.SectionTree{
			{
				Name:  "language",
				Title: "Language",
				Path:  "/pages/language/",
				Pages: []*parser.Page{
					{Title: "Active Voice", Path: "/pages/language/active-voice/"},
					{Title: "Pronouns", Path: "/pages/language/pronouns/"},
				},
			},
			{
				Name:  "formatting",
				Title: "Formatting",
				Path:  "/pages/formatting/",
				Pages: []*parser.Page{
					{Title: "Headings", Path: "/pages/formatting/headings/"},
				},
			},
		},
	}

	rules := []*parser.ValeRule{
		{Name: "Avoid", Extends: "existence", Level: "error", Category: "Style"},
	}
	categories := map[string][]string{"Style": {"Avoid"}}

	if err := generator.GenerateNavigationJSON(tree, rules, categories, dataDir); err != nil {
		t.Fatalf("GenerateNavigationJSON: %v", err)
	}

	data := readFile(t, filepath.Join(dataDir, "navigation.json"))
	var nav map[string]interface{}
	if err := json.Unmarshal([]byte(data), &nav); err != nil {
		t.Fatalf("navigation.json is not valid JSON: %v\n%s", err, data)
	}

	sections, ok := nav["sections"].([]interface{})
	if !ok {
		t.Fatalf("expected sections array in navigation.json")
	}
	if len(sections) != 2 {
		t.Errorf("expected 2 sections, got %d", len(sections))
	}

	first := sections[0].(map[string]interface{})
	if first["name"] != "language" {
		t.Errorf("first section name = %v, want language", first["name"])
	}
	if first["title"] != "Language" {
		t.Errorf("first section title = %v, want Language", first["title"])
	}

	pages := first["pages"].([]interface{})
	if len(pages) != 2 {
		t.Errorf("language section should have 2 pages, got %d", len(pages))
	}
}

func TestGenerateNavigationJSON_RulesPositionInOrder(t *testing.T) {
	dataDir := t.TempDir()

	tree := &parser.SectionTree{
		Name:  "pages",
		Title: "Pages",
		Path:  "/pages/",
		Meta: &parser.SectionMeta{
			Order: []string{"language", "rules", "formatting"},
		},
		Children: []*parser.SectionTree{
			{Name: "language", Title: "Language", Path: "/pages/language/",
				Pages: []*parser.Page{{Title: "Voice", Path: "/pages/language/voice/"}}},
			{Name: "formatting", Title: "Formatting", Path: "/pages/formatting/",
				Pages: []*parser.Page{{Title: "Headings", Path: "/pages/formatting/headings/"}}},
		},
	}

	rules := []*parser.ValeRule{
		{Name: "Avoid", Extends: "existence", Level: "error", Category: "Style"},
	}
	categories := map[string][]string{"Style": {"Avoid"}}

	if err := generator.GenerateNavigationJSON(tree, rules, categories, dataDir); err != nil {
		t.Fatalf("GenerateNavigationJSON: %v", err)
	}

	data := readFile(t, filepath.Join(dataDir, "navigation.json"))
	var nav map[string]interface{}
	if err := json.Unmarshal([]byte(data), &nav); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	rulesSection := nav["rules_section"].(map[string]interface{})
	pos := int(rulesSection["position"].(float64))
	if pos != 1 {
		t.Errorf("rules position = %d, want 1 (after language, before formatting)", pos)
	}
}

func TestGenerateNavigationJSON_RulesPositionZero(t *testing.T) {
	dataDir := t.TempDir()

	tree := &parser.SectionTree{
		Name:  "pages",
		Title: "Pages",
		Path:  "/pages/",
		Meta: &parser.SectionMeta{
			Order: []string{"rules", "language"},
		},
		Children: []*parser.SectionTree{
			{Name: "language", Title: "Language", Path: "/pages/language/",
				Pages: []*parser.Page{{Title: "Voice", Path: "/pages/language/voice/"}}},
		},
	}

	rules := []*parser.ValeRule{
		{Name: "Avoid", Extends: "existence", Level: "error", Category: "Style"},
	}
	categories := map[string][]string{"Style": {"Avoid"}}

	if err := generator.GenerateNavigationJSON(tree, rules, categories, dataDir); err != nil {
		t.Fatalf("GenerateNavigationJSON: %v", err)
	}

	data := readFile(t, filepath.Join(dataDir, "navigation.json"))
	var nav map[string]interface{}
	if err := json.Unmarshal([]byte(data), &nav); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	rulesSection := nav["rules_section"].(map[string]interface{})
	pos := int(rulesSection["position"].(float64))
	if pos != 0 {
		t.Errorf("rules position = %d, want 0", pos)
	}
}

func TestGenerateNavigationJSON_NoRulesKeyword_PositionNegativeOne(t *testing.T) {
	dataDir := t.TempDir()

	tree := &parser.SectionTree{
		Name:  "pages",
		Title: "Pages",
		Path:  "/pages/",
		// No Meta, so no order list
		Children: []*parser.SectionTree{
			{Name: "language", Title: "Language", Path: "/pages/language/",
				Pages: []*parser.Page{{Title: "Voice", Path: "/pages/language/voice/"}}},
		},
	}

	rules := []*parser.ValeRule{
		{Name: "Avoid", Extends: "existence", Level: "error", Category: "Style"},
	}
	categories := map[string][]string{"Style": {"Avoid"}}

	if err := generator.GenerateNavigationJSON(tree, rules, categories, dataDir); err != nil {
		t.Fatalf("GenerateNavigationJSON: %v", err)
	}

	data := readFile(t, filepath.Join(dataDir, "navigation.json"))
	var nav map[string]interface{}
	if err := json.Unmarshal([]byte(data), &nav); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	rulesSection := nav["rules_section"].(map[string]interface{})
	pos := int(rulesSection["position"].(float64))
	if pos != -1 {
		t.Errorf("rules position = %d, want -1 (no rules keyword)", pos)
	}
}

func TestGenerateNavigationJSON_PositionExceedsSectionCount_ClampedToNegativeOne(t *testing.T) {
	dataDir := t.TempDir()

	tree := &parser.SectionTree{
		Name:  "pages",
		Title: "Pages",
		Path:  "/pages/",
		Meta: &parser.SectionMeta{
			// "rules" at index 5, but only 1 section exists
			Order: []string{"a", "b", "c", "d", "e", "rules"},
		},
		Children: []*parser.SectionTree{
			{Name: "language", Title: "Language", Path: "/pages/language/",
				Pages: []*parser.Page{{Title: "Voice", Path: "/pages/language/voice/"}}},
		},
	}

	rules := []*parser.ValeRule{
		{Name: "Avoid", Extends: "existence", Level: "error", Category: "Style"},
	}
	categories := map[string][]string{"Style": {"Avoid"}}

	if err := generator.GenerateNavigationJSON(tree, rules, categories, dataDir); err != nil {
		t.Fatalf("GenerateNavigationJSON: %v", err)
	}

	data := readFile(t, filepath.Join(dataDir, "navigation.json"))
	var nav map[string]interface{}
	if err := json.Unmarshal([]byte(data), &nav); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	rulesSection := nav["rules_section"].(map[string]interface{})
	pos := int(rulesSection["position"].(float64))
	if pos != -1 {
		t.Errorf("rules position = %d, want -1 (clamped because exceeds section count)", pos)
	}
}

func TestGenerateNavigationJSON_RulesTitleOverride(t *testing.T) {
	dataDir := t.TempDir()

	tree := &parser.SectionTree{
		Name:  "pages",
		Title: "Pages",
		Path:  "/pages/",
		Meta: &parser.SectionMeta{
			RulesTitle: "Style Rules",
		},
		Children: []*parser.SectionTree{
			{Name: "language", Title: "Language", Path: "/pages/language/",
				Pages: []*parser.Page{{Title: "Voice", Path: "/pages/language/voice/"}}},
		},
	}

	rules := []*parser.ValeRule{
		{Name: "Avoid", Extends: "existence", Level: "error", Category: "Style"},
	}
	categories := map[string][]string{"Style": {"Avoid"}}

	if err := generator.GenerateNavigationJSON(tree, rules, categories, dataDir); err != nil {
		t.Fatalf("GenerateNavigationJSON: %v", err)
	}

	data := readFile(t, filepath.Join(dataDir, "navigation.json"))
	var nav map[string]interface{}
	if err := json.Unmarshal([]byte(data), &nav); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	rulesSection := nav["rules_section"].(map[string]interface{})
	if rulesSection["title"] != "Style Rules" {
		t.Errorf("rules title = %v, want 'Style Rules'", rulesSection["title"])
	}
}

func TestGenerateNavigationJSON_DefaultRulesTitle(t *testing.T) {
	dataDir := t.TempDir()

	tree := &parser.SectionTree{
		Name:  "pages",
		Title: "Pages",
		Path:  "/pages/",
		Children: []*parser.SectionTree{
			{Name: "language", Title: "Language", Path: "/pages/language/",
				Pages: []*parser.Page{{Title: "Voice", Path: "/pages/language/voice/"}}},
		},
	}

	rules := []*parser.ValeRule{
		{Name: "Avoid", Extends: "existence", Level: "error", Category: "Style"},
	}
	categories := map[string][]string{"Style": {"Avoid"}}

	if err := generator.GenerateNavigationJSON(tree, rules, categories, dataDir); err != nil {
		t.Fatalf("GenerateNavigationJSON: %v", err)
	}

	data := readFile(t, filepath.Join(dataDir, "navigation.json"))
	var nav map[string]interface{}
	if err := json.Unmarshal([]byte(data), &nav); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	rulesSection := nav["rules_section"].(map[string]interface{})
	if rulesSection["title"] != "Rules" {
		t.Errorf("rules title = %v, want 'Rules' (default)", rulesSection["title"])
	}
}

func TestGenerateNavigationJSON_NestedSections(t *testing.T) {
	dataDir := t.TempDir()

	tree := &parser.SectionTree{
		Name:  "pages",
		Title: "Pages",
		Path:  "/pages/",
		Children: []*parser.SectionTree{
			{
				Name:  "language",
				Title: "Language",
				Path:  "/pages/language/",
				Pages: []*parser.Page{
					{Title: "Active Voice", Path: "/pages/language/active-voice/"},
				},
				Children: []*parser.SectionTree{
					{
						Name:  "advanced",
						Title: "Advanced",
						Path:  "/pages/language/advanced/",
						Pages: []*parser.Page{
							{Title: "Subjunctive", Path: "/pages/language/advanced/subjunctive/"},
						},
					},
				},
			},
		},
	}

	rules := []*parser.ValeRule{}
	categories := map[string][]string{}

	if err := generator.GenerateNavigationJSON(tree, rules, categories, dataDir); err != nil {
		t.Fatalf("GenerateNavigationJSON: %v", err)
	}

	data := readFile(t, filepath.Join(dataDir, "navigation.json"))
	var nav map[string]interface{}
	if err := json.Unmarshal([]byte(data), &nav); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	sections := nav["sections"].([]interface{})
	langSection := sections[0].(map[string]interface{})
	children := langSection["children"].([]interface{})
	if len(children) != 1 {
		t.Fatalf("language section should have 1 child, got %d", len(children))
	}
	advSection := children[0].(map[string]interface{})
	if advSection["name"] != "advanced" {
		t.Errorf("child name = %v, want advanced", advSection["name"])
	}
	advPages := advSection["pages"].([]interface{})
	if len(advPages) != 1 {
		t.Errorf("advanced section should have 1 page, got %d", len(advPages))
	}
}

func TestGenerateNavigationJSON_HiddenPagesFilteredOut(t *testing.T) {
	dataDir := t.TempDir()

	tree := &parser.SectionTree{
		Name:  "pages",
		Title: "Pages",
		Path:  "/pages/",
		Children: []*parser.SectionTree{
			{
				Name:  "language",
				Title: "Language",
				Path:  "/pages/language/",
				Pages: []*parser.Page{
					{Title: "Active Voice", Path: "/pages/language/active-voice/", Hidden: false},
					{Title: "Hidden Notes", Path: "/pages/language/hidden-notes/", Hidden: true},
				},
			},
		},
	}

	rules := []*parser.ValeRule{}
	categories := map[string][]string{}

	if err := generator.GenerateNavigationJSON(tree, rules, categories, dataDir); err != nil {
		t.Fatalf("GenerateNavigationJSON: %v", err)
	}

	data := readFile(t, filepath.Join(dataDir, "navigation.json"))
	var nav map[string]interface{}
	if err := json.Unmarshal([]byte(data), &nav); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	sections := nav["sections"].([]interface{})
	langSection := sections[0].(map[string]interface{})
	pages := langSection["pages"].([]interface{})
	if len(pages) != 1 {
		t.Errorf("expected 1 visible page (hidden filtered out), got %d", len(pages))
	}
	firstPage := pages[0].(map[string]interface{})
	if firstPage["title"] != "Active Voice" {
		t.Errorf("visible page title = %v, want Active Voice", firstPage["title"])
	}
}

func TestGenerateNavigationJSON_NilPages_NoFile(t *testing.T) {
	dataDir := t.TempDir()

	rules := []*parser.ValeRule{
		{Name: "Avoid", Extends: "existence", Level: "error", Category: "Style"},
	}
	categories := map[string][]string{"Style": {"Avoid"}}

	if err := generator.GenerateNavigationJSON(nil, rules, categories, dataDir); err != nil {
		t.Fatalf("GenerateNavigationJSON: %v", err)
	}

	navPath := filepath.Join(dataDir, "navigation.json")
	if _, err := os.Stat(navPath); !os.IsNotExist(err) {
		t.Error("navigation.json should not exist when pages is nil")
	}
}

func TestGenerateNavigationJSON_EmptySectionTree_NoFile(t *testing.T) {
	dataDir := t.TempDir()

	tree := &parser.SectionTree{
		Name:  "pages",
		Title: "Pages",
		Path:  "/pages/",
		// No pages, no children, no IndexPage
	}

	rules := []*parser.ValeRule{
		{Name: "Avoid", Extends: "existence", Level: "error", Category: "Style"},
	}
	categories := map[string][]string{"Style": {"Avoid"}}

	if err := generator.GenerateNavigationJSON(tree, rules, categories, dataDir); err != nil {
		t.Fatalf("GenerateNavigationJSON: %v", err)
	}

	navPath := filepath.Join(dataDir, "navigation.json")
	if _, err := os.Stat(navPath); !os.IsNotExist(err) {
		t.Error("navigation.json should not exist when tree is empty")
	}
}

func TestGenerateNavigationJSON_RulesCategoriesGrouped(t *testing.T) {
	dataDir := t.TempDir()

	tree := &parser.SectionTree{
		Name:  "pages",
		Title: "Pages",
		Path:  "/pages/",
		Children: []*parser.SectionTree{
			{Name: "language", Title: "Language", Path: "/pages/language/",
				Pages: []*parser.Page{{Title: "Voice", Path: "/pages/language/voice/"}}},
		},
	}

	rules := []*parser.ValeRule{
		{Name: "Avoid", Extends: "existence", Level: "error", Category: "Style"},
		{Name: "Terms", Extends: "substitution", Level: "warning", Category: "Terminology"},
		{Name: "Passive", Extends: "existence", Level: "warning", Category: "Style"},
	}
	categories := map[string][]string{
		"Style":       {"Avoid", "Passive"},
		"Terminology": {"Terms"},
	}

	if err := generator.GenerateNavigationJSON(tree, rules, categories, dataDir); err != nil {
		t.Fatalf("GenerateNavigationJSON: %v", err)
	}

	data := readFile(t, filepath.Join(dataDir, "navigation.json"))
	var nav map[string]interface{}
	if err := json.Unmarshal([]byte(data), &nav); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	rulesSection := nav["rules_section"].(map[string]interface{})
	cats := rulesSection["categories"].([]interface{})
	if len(cats) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(cats))
	}

	// Categories should be sorted alphabetically.
	first := cats[0].(map[string]interface{})
	if first["name"] != "Style" {
		t.Errorf("first category = %v, want Style", first["name"])
	}
	firstRules := first["rules"].([]interface{})
	if len(firstRules) != 2 {
		t.Errorf("Style category should have 2 rules, got %d", len(firstRules))
	}

	second := cats[1].(map[string]interface{})
	if second["name"] != "Terminology" {
		t.Errorf("second category = %v, want Terminology", second["name"])
	}
}

func TestGenerateNavigationJSON_RulePathDerivedFromName(t *testing.T) {
	dataDir := t.TempDir()

	tree := &parser.SectionTree{
		Name:  "pages",
		Title: "Pages",
		Path:  "/pages/",
		Children: []*parser.SectionTree{
			{Name: "language", Title: "Language", Path: "/pages/language/",
				Pages: []*parser.Page{{Title: "Voice", Path: "/pages/language/voice/"}}},
		},
	}

	rules := []*parser.ValeRule{
		{Name: "HeadingPunctuation", Extends: "existence", Level: "warning", Category: "Style"},
	}
	categories := map[string][]string{"Style": {"HeadingPunctuation"}}

	if err := generator.GenerateNavigationJSON(tree, rules, categories, dataDir); err != nil {
		t.Fatalf("GenerateNavigationJSON: %v", err)
	}

	data := readFile(t, filepath.Join(dataDir, "navigation.json"))
	var nav map[string]interface{}
	if err := json.Unmarshal([]byte(data), &nav); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	rulesSection := nav["rules_section"].(map[string]interface{})
	cats := rulesSection["categories"].([]interface{})
	cat := cats[0].(map[string]interface{})
	navRules := cat["rules"].([]interface{})
	rule := navRules[0].(map[string]interface{})
	if rule["path"] != "/rules/headingpunctuation/" {
		t.Errorf("rule path = %v, want /rules/headingpunctuation/", rule["path"])
	}
	if rule["level"] != "warning" {
		t.Errorf("rule level = %v, want warning", rule["level"])
	}
}

func TestGenerateNavigationJSON_RuleLevelIncluded(t *testing.T) {
	dataDir := t.TempDir()

	tree := &parser.SectionTree{
		Name:  "pages",
		Title: "Pages",
		Path:  "/pages/",
		Children: []*parser.SectionTree{
			{Name: "language", Title: "Language", Path: "/pages/language/",
				Pages: []*parser.Page{{Title: "Voice", Path: "/pages/language/voice/"}}},
		},
	}

	rules := []*parser.ValeRule{
		{Name: "Avoid", Extends: "existence", Level: "error", Category: "Style"},
		{Name: "Terms", Extends: "substitution", Level: "suggestion", Category: "Style"},
	}
	categories := map[string][]string{"Style": {"Avoid", "Terms"}}

	if err := generator.GenerateNavigationJSON(tree, rules, categories, dataDir); err != nil {
		t.Fatalf("GenerateNavigationJSON: %v", err)
	}

	data := readFile(t, filepath.Join(dataDir, "navigation.json"))
	var nav map[string]interface{}
	if err := json.Unmarshal([]byte(data), &nav); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	rulesSection := nav["rules_section"].(map[string]interface{})
	cats := rulesSection["categories"].([]interface{})
	cat := cats[0].(map[string]interface{})
	navRules := cat["rules"].([]interface{})

	// Rules are sorted alphabetically: Avoid, Terms
	avoidRule := navRules[0].(map[string]interface{})
	if avoidRule["level"] != "error" {
		t.Errorf("Avoid level = %v, want error", avoidRule["level"])
	}

	termsRule := navRules[1].(map[string]interface{})
	if termsRule["level"] != "suggestion" {
		t.Errorf("Terms level = %v, want suggestion", termsRule["level"])
	}
}

func TestGenerateNavigationJSON_CollapsedSection(t *testing.T) {
	dataDir := t.TempDir()

	tree := &parser.SectionTree{
		Name:  "pages",
		Title: "Pages",
		Path:  "/pages/",
		Children: []*parser.SectionTree{
			{
				Name:  "language",
				Title: "Language",
				Path:  "/pages/language/",
				Meta: &parser.SectionMeta{
					Collapsed: true,
				},
				Pages: []*parser.Page{
					{Title: "Voice", Path: "/pages/language/voice/"},
				},
			},
		},
	}

	rules := []*parser.ValeRule{}
	categories := map[string][]string{}

	if err := generator.GenerateNavigationJSON(tree, rules, categories, dataDir); err != nil {
		t.Fatalf("GenerateNavigationJSON: %v", err)
	}

	data := readFile(t, filepath.Join(dataDir, "navigation.json"))
	var nav map[string]interface{}
	if err := json.Unmarshal([]byte(data), &nav); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	sections := nav["sections"].([]interface{})
	sec := sections[0].(map[string]interface{})
	if sec["collapsed"] != true {
		t.Errorf("collapsed = %v, want true", sec["collapsed"])
	}
}
