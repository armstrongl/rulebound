package generator_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/armstrongl/rulebound/internal/config"
	"github.com/armstrongl/rulebound/internal/generator"
	"github.com/armstrongl/rulebound/internal/parser"
)

// ── Integration: realistic pages structure ───────────────────────────────────

func TestIntegration_RealisticPagesStructure(t *testing.T) {
	outDir := t.TempDir()

	result := &parser.ParseResult{
		Rules: []*parser.ValeRule{
			{Name: "Avoid", Extends: "existence", Level: "error", Message: "Don't use '%s'."},
			{Name: "Terms", Extends: "substitution", Level: "warning", Message: "Use '%s' instead."},
			{Name: "Headings", Extends: "capitalization", Level: "suggestion", Message: "Use sentence case."},
		},
		Pages: &parser.SectionTree{
			Name:  "pages",
			Title: "Pages",
			Path:  "/pages/",
			Children: []*parser.SectionTree{
				{
					Name:  "language",
					Title: "Language",
					Path:  "/pages/language/",
					Meta: &parser.SectionMeta{
						Order: []string{"active-voice", "pronouns"},
					},
					Pages: []*parser.Page{
						{Title: "Active Voice", Body: "Use active voice whenever possible.", Path: "/pages/language/active-voice/"},
						{Title: "Pronouns", Body: "Prefer they/them as a singular pronoun.", Path: "/pages/language/pronouns/"},
					},
				},
				{
					Name:  "formatting",
					Title: "Formatting",
					Path:  "/pages/formatting/",
					Meta: &parser.SectionMeta{
						Collapsed: true,
					},
					Pages: []*parser.Page{
						{Title: "Headings", Body: "Use sentence case for headings.", Path: "/pages/formatting/headings/"},
						{Title: "Lists", Body: "Use parallel construction in lists.", Path: "/pages/formatting/lists/"},
					},
				},
				{
					Name:  "resources",
					Title: "Resources",
					Path:  "/pages/resources/",
					Pages: []*parser.Page{
						{Title: "Glossary", Body: "Key terms and definitions.", Path: "/pages/resources/glossary/"},
					},
					Children: []*parser.SectionTree{
						{
							Name:  "templates",
							Title: "Templates",
							Path:  "/pages/resources/templates/",
							Pages: []*parser.Page{
								{Title: "Email Template", Body: "Standard email format.", Path: "/pages/resources/templates/email-template/"},
							},
						},
					},
				},
			},
		},
	}
	cfg := &config.Config{
		Title:   "Test Style Guide",
		BaseURL: "/",
		Categories: map[string][]string{
			"Style":      {"Avoid", "Headings"},
			"Vocabulary": {"Terms"},
		},
	}

	if err := generator.GenerateSite(result, cfg, outDir); err != nil {
		t.Fatalf("GenerateSite: %v", err)
	}

	// All page .md files exist at correct paths under content/pages/.
	expectedPages := []string{
		"content/pages/language/active-voice.md",
		"content/pages/language/pronouns.md",
		"content/pages/formatting/headings.md",
		"content/pages/formatting/lists.md",
		"content/pages/resources/glossary.md",
		"content/pages/resources/templates/email-template.md",
	}
	for _, rel := range expectedPages {
		fullPath := filepath.Join(outDir, rel)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("expected page file %s to exist", rel)
		}
	}

	// All _index.md files exist for each section.
	expectedIndexes := []string{
		"content/pages/_index.md",
		"content/pages/language/_index.md",
		"content/pages/formatting/_index.md",
		"content/pages/resources/_index.md",
		"content/pages/resources/templates/_index.md",
	}
	for _, rel := range expectedIndexes {
		fullPath := filepath.Join(outDir, rel)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("expected index file %s to exist", rel)
		}
	}

	// Page content has type: page in frontmatter.
	pageContent := readFile(t, filepath.Join(outDir, "content", "pages", "language", "active-voice.md"))
	if !strings.Contains(pageContent, "type: page") {
		t.Errorf("page file missing type: page in frontmatter:\n%s", pageContent)
	}

	// navigation.json exists and contains all 3 top-level sections.
	navPath := filepath.Join(outDir, "data", "navigation.json")
	if _, err := os.Stat(navPath); os.IsNotExist(err) {
		t.Fatal("expected data/navigation.json to exist")
	}

	navData := readFile(t, navPath)
	var nav map[string]interface{}
	if err := json.Unmarshal([]byte(navData), &nav); err != nil {
		t.Fatalf("navigation.json is not valid JSON: %v\n%s", err, navData)
	}

	sections, ok := nav["sections"].([]interface{})
	if !ok {
		t.Fatal("expected sections array in navigation.json")
	}
	if len(sections) != 3 {
		t.Errorf("expected 3 top-level sections, got %d", len(sections))
	}

	// navigation.json has rules_section with categories.
	rulesSection, ok := nav["rules_section"].(map[string]interface{})
	if !ok {
		t.Fatal("expected rules_section in navigation.json")
	}
	cats, ok := rulesSection["categories"].([]interface{})
	if !ok {
		t.Fatal("expected categories in rules_section")
	}
	if len(cats) != 2 {
		t.Errorf("expected 2 categories (Style, Vocabulary), got %d", len(cats))
	}

	// Rule .md files still exist under content/rules/.
	ruleFiles := []string{
		"content/rules/avoid.md",
		"content/rules/terms.md",
		"content/rules/headings.md",
	}
	for _, rel := range ruleFiles {
		fullPath := filepath.Join(outDir, rel)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("expected rule file %s to exist", rel)
		}
	}

	// Rules _index.md exists.
	if _, err := os.Stat(filepath.Join(outDir, "content", "rules", "_index.md")); os.IsNotExist(err) {
		t.Error("expected content/rules/_index.md to exist")
	}

	// data/site.json exists.
	if _, err := os.Stat(filepath.Join(outDir, "data", "site.json")); os.IsNotExist(err) {
		t.Error("expected data/site.json to exist")
	}

	// hugo.toml exists.
	if _, err := os.Stat(filepath.Join(outDir, "hugo.toml")); os.IsNotExist(err) {
		t.Error("expected hugo.toml to exist")
	}
}

// ── Integration: rules-only backward compat ──────────────────────────────────

func TestIntegration_RulesOnlyBackwardCompat(t *testing.T) {
	outDir := t.TempDir()

	result := &parser.ParseResult{
		Rules: []*parser.ValeRule{
			{Name: "Avoid", Extends: "existence", Level: "error", Message: "Don't use '%s'."},
			{Name: "Terms", Extends: "substitution", Level: "warning", Message: "Use '%s' instead."},
		},
	}
	cfg := &config.Config{
		Title:   "Rules Only Guide",
		BaseURL: "/",
	}

	if err := generator.GenerateSite(result, cfg, outDir); err != nil {
		t.Fatalf("GenerateSite: %v", err)
	}

	// content/rules/ directory and rule files exist.
	for _, name := range []string{"avoid.md", "terms.md"} {
		path := filepath.Join(outDir, "content", "rules", name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected rule file %s to exist", name)
		}
	}

	// content/pages/ does NOT exist.
	pagesDir := filepath.Join(outDir, "content", "pages")
	if _, err := os.Stat(pagesDir); !os.IsNotExist(err) {
		t.Error("content/pages/ should not exist in rules-only mode")
	}

	// data/navigation.json does NOT exist.
	navPath := filepath.Join(outDir, "data", "navigation.json")
	if _, err := os.Stat(navPath); !os.IsNotExist(err) {
		t.Error("data/navigation.json should not exist in rules-only mode")
	}

	// data/site.json exists.
	if _, err := os.Stat(filepath.Join(outDir, "data", "site.json")); os.IsNotExist(err) {
		t.Error("expected data/site.json to exist")
	}

	// content/guidelines/ does NOT exist.
	guidelinesDir := filepath.Join(outDir, "content", "guidelines")
	if _, err := os.Stat(guidelinesDir); !os.IsNotExist(err) {
		t.Error("content/guidelines/ should not exist in rules-only mode")
	}
}

// ── Integration: guidelines backward compat ──────────────────────────────────

func TestIntegration_GuidelinesBackwardCompat(t *testing.T) {
	outDir := t.TempDir()

	result := &parser.ParseResult{
		Rules: []*parser.ValeRule{
			{Name: "Avoid", Extends: "existence", Level: "error", Message: "Don't use '%s'."},
			{Name: "Terms", Extends: "substitution", Level: "warning", Message: "Use '%s' instead."},
		},
		Guidelines: []*parser.Guideline{
			{Name: "voice-and-tone", Title: "Voice and Tone", Weight: 10, Body: "Write clearly."},
			{Name: "inclusive-language", Title: "Inclusive Language", Weight: 20, Body: "Be inclusive."},
		},
	}
	cfg := &config.Config{
		Title:   "Guidelines Guide",
		BaseURL: "/",
	}

	if err := generator.GenerateSite(result, cfg, outDir); err != nil {
		t.Fatalf("GenerateSite: %v", err)
	}

	// content/guidelines/ exists with guideline .md files.
	for _, name := range []string{"voice-and-tone.md", "inclusive-language.md"} {
		path := filepath.Join(outDir, "content", "guidelines", name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected guideline file %s to exist", name)
		}
	}

	// content/guidelines/_index.md exists.
	indexPath := filepath.Join(outDir, "content", "guidelines", "_index.md")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Error("expected content/guidelines/_index.md to exist")
	}

	// content/pages/ does NOT exist.
	pagesDir := filepath.Join(outDir, "content", "pages")
	if _, err := os.Stat(pagesDir); !os.IsNotExist(err) {
		t.Error("content/pages/ should not exist when only guidelines are present")
	}

	// data/navigation.json does NOT exist.
	navPath := filepath.Join(outDir, "data", "navigation.json")
	if _, err := os.Stat(navPath); !os.IsNotExist(err) {
		t.Error("data/navigation.json should not exist when only guidelines are present")
	}

	// content/rules/ and rule files exist.
	for _, name := range []string{"avoid.md", "terms.md"} {
		path := filepath.Join(outDir, "content", "rules", name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected rule file %s to exist alongside guidelines", name)
		}
	}
}

// ── Integration: hidden pages excluded from navigation ───────────────────────

func TestIntegration_HiddenPagesExcludedFromNavigation(t *testing.T) {
	outDir := t.TempDir()

	result := &parser.ParseResult{
		Rules: []*parser.ValeRule{
			{Name: "Avoid", Extends: "existence", Level: "error", Message: "Don't use '%s'."},
		},
		Pages: &parser.SectionTree{
			Name:  "pages",
			Title: "Pages",
			Path:  "/pages/",
			Children: []*parser.SectionTree{
				{
					Name:  "language",
					Title: "Language",
					Path:  "/pages/language/",
					Pages: []*parser.Page{
						{Title: "Active Voice", Body: "Use active voice.", Path: "/pages/language/active-voice/"},
						{Title: "Pronouns", Body: "Use they/them.", Path: "/pages/language/pronouns/"},
						{Title: "Internal Notes", Body: "Hidden content.", Path: "/pages/language/internal-notes/", Hidden: true},
					},
				},
			},
		},
	}
	cfg := &config.Config{
		Title:   "Test Guide",
		BaseURL: "/",
		Categories: map[string][]string{
			"Style": {"Avoid"},
		},
	}

	if err := generator.GenerateSite(result, cfg, outDir); err != nil {
		t.Fatalf("GenerateSite: %v", err)
	}

	// All 3 page .md files exist (hidden pages still get content files).
	for _, slug := range []string{"active-voice", "pronouns", "internal-notes"} {
		path := filepath.Join(outDir, "content", "pages", "language", slug+".md")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected page file %s.md to exist", slug)
		}
	}

	// Hidden page has pagefind: false in its frontmatter.
	hiddenContent := readFile(t, filepath.Join(outDir, "content", "pages", "language", "internal-notes.md"))
	if !strings.Contains(hiddenContent, "pagefind: false") {
		t.Errorf("hidden page missing pagefind: false:\n%s", hiddenContent)
	}

	// navigation.json does NOT contain the hidden page in its sections.
	navData := readFile(t, filepath.Join(outDir, "data", "navigation.json"))
	var nav map[string]interface{}
	if err := json.Unmarshal([]byte(navData), &nav); err != nil {
		t.Fatalf("navigation.json is not valid JSON: %v", err)
	}

	sections := nav["sections"].([]interface{})
	langSection := sections[0].(map[string]interface{})
	pages := langSection["pages"].([]interface{})
	if len(pages) != 2 {
		t.Errorf("expected 2 visible pages in navigation (hidden filtered), got %d", len(pages))
	}
	for _, p := range pages {
		page := p.(map[string]interface{})
		if page["title"] == "Internal Notes" {
			t.Error("hidden page 'Internal Notes' should not appear in navigation.json")
		}
	}
}

// ── Integration: rules position in navigation ────────────────────────────────

func TestIntegration_RulesPositionInNavigation(t *testing.T) {
	outDir := t.TempDir()

	result := &parser.ParseResult{
		Rules: []*parser.ValeRule{
			{Name: "Avoid", Extends: "existence", Level: "error", Message: "Don't use '%s'."},
		},
		Pages: &parser.SectionTree{
			Name:  "pages",
			Title: "Pages",
			Path:  "/pages/",
			Meta: &parser.SectionMeta{
				Order: []string{"language", "rules", "formatting"},
			},
			Children: []*parser.SectionTree{
				{
					Name:  "language",
					Title: "Language",
					Path:  "/pages/language/",
					Pages: []*parser.Page{
						{Title: "Active Voice", Body: "Use active voice.", Path: "/pages/language/active-voice/"},
					},
				},
				{
					Name:  "formatting",
					Title: "Formatting",
					Path:  "/pages/formatting/",
					Pages: []*parser.Page{
						{Title: "Headings", Body: "Use sentence case.", Path: "/pages/formatting/headings/"},
					},
				},
			},
		},
	}
	cfg := &config.Config{
		Title:   "Test Guide",
		BaseURL: "/",
		Categories: map[string][]string{
			"Style": {"Avoid"},
		},
	}

	if err := generator.GenerateSite(result, cfg, outDir); err != nil {
		t.Fatalf("GenerateSite: %v", err)
	}

	navData := readFile(t, filepath.Join(outDir, "data", "navigation.json"))
	var nav map[string]interface{}
	if err := json.Unmarshal([]byte(navData), &nav); err != nil {
		t.Fatalf("navigation.json is not valid JSON: %v", err)
	}

	rulesSection := nav["rules_section"].(map[string]interface{})
	pos := int(rulesSection["position"].(float64))
	if pos != 1 {
		t.Errorf("rules_section.position = %d, want 1 (between language and formatting)", pos)
	}
}

// ── Integration: rules custom title ──────────────────────────────────────────

func TestIntegration_RulesCustomTitle(t *testing.T) {
	outDir := t.TempDir()

	result := &parser.ParseResult{
		Rules: []*parser.ValeRule{
			{Name: "Avoid", Extends: "existence", Level: "error", Message: "Don't use '%s'."},
		},
		Pages: &parser.SectionTree{
			Name:  "pages",
			Title: "Pages",
			Path:  "/pages/",
			Meta: &parser.SectionMeta{
				RulesTitle: "Style Rules",
			},
			Children: []*parser.SectionTree{
				{
					Name:  "language",
					Title: "Language",
					Path:  "/pages/language/",
					Pages: []*parser.Page{
						{Title: "Voice", Body: "Use active voice.", Path: "/pages/language/voice/"},
					},
				},
			},
		},
	}
	cfg := &config.Config{
		Title:   "Test Guide",
		BaseURL: "/",
		Categories: map[string][]string{
			"Style": {"Avoid"},
		},
	}

	if err := generator.GenerateSite(result, cfg, outDir); err != nil {
		t.Fatalf("GenerateSite: %v", err)
	}

	navData := readFile(t, filepath.Join(outDir, "data", "navigation.json"))
	var nav map[string]interface{}
	if err := json.Unmarshal([]byte(navData), &nav); err != nil {
		t.Fatalf("navigation.json is not valid JSON: %v", err)
	}

	rulesSection := nav["rules_section"].(map[string]interface{})
	if rulesSection["title"] != "Style Rules" {
		t.Errorf("rules_section.title = %v, want 'Style Rules'", rulesSection["title"])
	}
}

// ── Integration: collapsed section ───────────────────────────────────────────

func TestIntegration_CollapsedSection(t *testing.T) {
	outDir := t.TempDir()

	result := &parser.ParseResult{
		Rules: []*parser.ValeRule{
			{Name: "Avoid", Extends: "existence", Level: "error", Message: "Don't use '%s'."},
		},
		Pages: &parser.SectionTree{
			Name:  "pages",
			Title: "Pages",
			Path:  "/pages/",
			Children: []*parser.SectionTree{
				{
					Name:  "reference",
					Title: "Reference",
					Path:  "/pages/reference/",
					Meta: &parser.SectionMeta{
						Collapsed: true,
					},
					Pages: []*parser.Page{
						{Title: "Glossary", Body: "Key terms.", Path: "/pages/reference/glossary/"},
						{Title: "Acronyms", Body: "Common acronyms.", Path: "/pages/reference/acronyms/"},
					},
				},
			},
		},
	}
	cfg := &config.Config{
		Title:   "Test Guide",
		BaseURL: "/",
		Categories: map[string][]string{
			"Style": {"Avoid"},
		},
	}

	if err := generator.GenerateSite(result, cfg, outDir); err != nil {
		t.Fatalf("GenerateSite: %v", err)
	}

	navData := readFile(t, filepath.Join(outDir, "data", "navigation.json"))
	var nav map[string]interface{}
	if err := json.Unmarshal([]byte(navData), &nav); err != nil {
		t.Fatalf("navigation.json is not valid JSON: %v", err)
	}

	sections := nav["sections"].([]interface{})
	sec := sections[0].(map[string]interface{})
	if sec["collapsed"] != true {
		t.Errorf("collapsed = %v, want true", sec["collapsed"])
	}
}

// ── Integration: author-provided index page ──────────────────────────────────

func TestIntegration_AuthorProvidedIndexPage(t *testing.T) {
	outDir := t.TempDir()

	result := &parser.ParseResult{
		Rules: []*parser.ValeRule{
			{Name: "Avoid", Extends: "existence", Level: "error", Message: "Don't use '%s'."},
		},
		Pages: &parser.SectionTree{
			Name:  "pages",
			Title: "Pages",
			Path:  "/pages/",
			Children: []*parser.SectionTree{
				{
					Name:  "language",
					Title: "Language",
					Path:  "/pages/language/",
					IndexPage: &parser.Page{
						Title:       "Language Hub",
						Description: "Everything about language and grammar",
						Body:        "Welcome to the language section. Here you will find all language guidance.",
						Path:        "/pages/language/",
					},
					Pages: []*parser.Page{
						{Title: "Active Voice", Body: "Use active voice.", Path: "/pages/language/active-voice/"},
					},
				},
			},
		},
	}
	cfg := &config.Config{
		Title:   "Test Guide",
		BaseURL: "/",
		Categories: map[string][]string{
			"Style": {"Avoid"},
		},
	}

	if err := generator.GenerateSite(result, cfg, outDir); err != nil {
		t.Fatalf("GenerateSite: %v", err)
	}

	// _index.md for that section contains the author's body content.
	indexContent := readFile(t, filepath.Join(outDir, "content", "pages", "language", "_index.md"))
	if !strings.Contains(indexContent, "Welcome to the language section.") {
		t.Errorf("_index.md should contain author body content:\n%s", indexContent)
	}

	// _index.md has the author's title in frontmatter.
	if !strings.Contains(indexContent, "Language Hub") {
		t.Errorf("_index.md should contain author title 'Language Hub':\n%s", indexContent)
	}

	// _index.md has type: page.
	if !strings.Contains(indexContent, "type: page") {
		t.Errorf("_index.md should have type: page:\n%s", indexContent)
	}
}

// ── Integration: pages supersede guidelines ──────────────────────────────────

func TestIntegration_PagesSupersededGuidelines(t *testing.T) {
	outDir := t.TempDir()

	result := &parser.ParseResult{
		Rules: []*parser.ValeRule{
			{Name: "Avoid", Extends: "existence", Level: "error", Message: "Don't use '%s'."},
		},
		Guidelines: []*parser.Guideline{
			{Name: "voice-and-tone", Title: "Voice and Tone", Weight: 10, Body: "Write clearly."},
			{Name: "inclusive-language", Title: "Inclusive Language", Weight: 20, Body: "Be inclusive."},
		},
		Pages: &parser.SectionTree{
			Name:  "pages",
			Title: "Pages",
			Path:  "/pages/",
			Children: []*parser.SectionTree{
				{
					Name:  "language",
					Title: "Language",
					Path:  "/pages/language/",
					Pages: []*parser.Page{
						{Title: "Active Voice", Body: "Use active voice.", Path: "/pages/language/active-voice/"},
					},
				},
			},
		},
	}
	cfg := &config.Config{
		Title:   "Test Guide",
		BaseURL: "/",
		Categories: map[string][]string{
			"Style": {"Avoid"},
		},
	}

	if err := generator.GenerateSite(result, cfg, outDir); err != nil {
		t.Fatalf("GenerateSite: %v", err)
	}

	// Pages content files exist.
	pagePath := filepath.Join(outDir, "content", "pages", "language", "active-voice.md")
	if _, err := os.Stat(pagePath); os.IsNotExist(err) {
		t.Error("expected page file to exist when pages supersede guidelines")
	}

	// Guidelines directory does NOT exist.
	guidelinesDir := filepath.Join(outDir, "content", "guidelines")
	if _, err := os.Stat(guidelinesDir); !os.IsNotExist(err) {
		t.Error("content/guidelines/ should not exist when pages are present (pages supersede guidelines)")
	}

	// navigation.json exists (confirming pages path ran).
	navPath := filepath.Join(outDir, "data", "navigation.json")
	if _, err := os.Stat(navPath); os.IsNotExist(err) {
		t.Error("expected data/navigation.json to exist when pages are present")
	}
}
