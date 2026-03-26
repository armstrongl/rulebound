package generator_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larah/rulebound/internal/config"
	"github.com/larah/rulebound/internal/generator"
	"github.com/larah/rulebound/internal/parser"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func makeRule(name, extends, level string) *parser.ValeRule {
	return &parser.ValeRule{
		Name:    name,
		Extends: extends,
		Level:   level,
		Message: "Use '%s' instead.",
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}

// ── DisplayName ───────────────────────────────────────────────────────────────

func TestDisplayName_Simple(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"HeadingPunctuation", "Heading Punctuation"},
		{"OxfordComma", "Oxford Comma"},
		{"AMPM", "AMPM"},
		{"URLFormat", "URL Format"},
		{"Avoid", "Avoid"},
		{"SentenceLength", "Sentence Length"},
		{"GeneralURL", "General URL"},
		{"Terms", "Terms"},
		{"ABCDef", "ABC Def"},
		{"ABCdef", "AB Cdef"},
	}
	for _, tc := range cases {
		got := generator.DisplayName(tc.input)
		if got != tc.want {
			t.Errorf("DisplayName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ── AutoDescription ───────────────────────────────────────────────────────────

func TestAutoDescription_Base(t *testing.T) {
	rule := makeRule("Avoid", "existence", "error")
	rule.Message = "Don't use '%s'."
	desc := generator.AutoDescription(rule)
	if !strings.Contains(desc, "Avoid") {
		t.Errorf("description missing rule name: %q", desc)
	}
	if !strings.Contains(desc, "error") {
		t.Errorf("description missing level: %q", desc)
	}
	if !strings.Contains(desc, "existence") {
		t.Errorf("description missing extends: %q", desc)
	}
	// %s should be stripped
	if strings.Contains(desc, "%s") {
		t.Errorf("description should not contain %%s: %q", desc)
	}
}

func TestAutoDescription_WithLink(t *testing.T) {
	rule := makeRule("Terms", "substitution", "warning")
	rule.Link = "https://docs.microsoft.com/en-us/style-guide"
	desc := generator.AutoDescription(rule)
	if !strings.Contains(desc, "docs.microsoft.com") {
		t.Errorf("description missing link domain: %q", desc)
	}
	if !strings.Contains(desc, rule.Link) {
		t.Errorf("description missing full link: %q", desc)
	}
}

func TestAutoDescription_WithTokens(t *testing.T) {
	rule := makeRule("Avoid", "existence", "error")
	rule.Tokens = []string{"foo", "bar", "baz"}
	desc := generator.AutoDescription(rule)
	if !strings.Contains(desc, "foo") {
		t.Errorf("description missing token: %q", desc)
	}
}

func TestAutoDescription_TokensTruncateAt10(t *testing.T) {
	rule := makeRule("Avoid", "existence", "error")
	for i := 0; i < 15; i++ {
		rule.Tokens = append(rule.Tokens, fmt.Sprintf("tok%d", i))
	}
	desc := generator.AutoDescription(rule)
	if !strings.Contains(desc, "tok9") {
		t.Errorf("should include 10th token tok9: %q", desc)
	}
	if strings.Contains(desc, "tok10") {
		t.Errorf("should not include 11th token tok10: %q", desc)
	}
	if !strings.Contains(desc, "...") {
		t.Errorf("should include truncation marker: %q", desc)
	}
}

func TestAutoDescription_WithSwap(t *testing.T) {
	rule := makeRule("Terms", "substitution", "warning")
	rule.Swap = map[string]string{"foo": "bar", "baz": "qux"}
	desc := generator.AutoDescription(rule)
	if !strings.Contains(desc, "2") {
		t.Errorf("description missing swap count: %q", desc)
	}
}

// ── BuildFrontmatter ──────────────────────────────────────────────────────────

func TestBuildFrontmatter_BasicFields(t *testing.T) {
	rule := makeRule("Avoid", "existence", "error")
	rule.Message = "Don't use '%s'."
	fm, err := generator.BuildFrontmatter(rule)
	if err != nil {
		t.Fatalf("BuildFrontmatter: %v", err)
	}
	if !strings.Contains(fm, "title:") {
		t.Errorf("frontmatter missing title: %s", fm)
	}
	if !strings.Contains(fm, "extends:") {
		t.Errorf("frontmatter missing extends: %s", fm)
	}
	if !strings.Contains(fm, "level:") {
		t.Errorf("frontmatter missing level: %s", fm)
	}
	if !strings.Contains(fm, "message:") {
		t.Errorf("frontmatter missing message: %s", fm)
	}
}

func TestBuildFrontmatter_TaxonomyTerms(t *testing.T) {
	rule := makeRule("Avoid", "existence", "error")
	rule.Category = "Formatting"
	fm, err := generator.BuildFrontmatter(rule)
	if err != nil {
		t.Fatalf("BuildFrontmatter: %v", err)
	}
	if !strings.Contains(fm, "categories:") {
		t.Errorf("frontmatter missing categories: %s", fm)
	}
	if !strings.Contains(fm, "Formatting") {
		t.Errorf("frontmatter missing category value: %s", fm)
	}
	if !strings.Contains(fm, "ruletypes:") {
		t.Errorf("frontmatter missing ruletypes: %s", fm)
	}
	if !strings.Contains(fm, "severities:") {
		t.Errorf("frontmatter missing severities: %s", fm)
	}
}

func TestBuildFrontmatter_RegexEscapingInSwap(t *testing.T) {
	rule := makeRule("Terms", "substitution", "warning")
	rule.Swap = map[string]string{
		`(?:agent|virtual assistant)`: "personal digital assistant",
		`24/7`:                        "every day",
	}
	fm, err := generator.BuildFrontmatter(rule)
	if err != nil {
		t.Fatalf("BuildFrontmatter: %v", err)
	}
	// The regex key must appear without corrupting the YAML
	if !strings.Contains(fm, "swap:") {
		t.Errorf("frontmatter missing swap: %s", fm)
	}
	// yaml.Marshal must have quoted the key properly — check the key is present
	if !strings.Contains(fm, "24/7") {
		t.Errorf("frontmatter missing swap key '24/7': %s", fm)
	}
}

func TestBuildFrontmatter_RegexEscapingInTokens(t *testing.T) {
	rule := makeRule("Avoid", "existence", "error")
	rule.Tokens = []string{`\.`, `;`, `we'(?:ve|re)`}
	fm, err := generator.BuildFrontmatter(rule)
	if err != nil {
		t.Fatalf("BuildFrontmatter: %v", err)
	}
	if !strings.Contains(fm, "tokens:") {
		t.Errorf("frontmatter missing tokens: %s", fm)
	}
	// Backslash must be preserved in the output
	if !strings.Contains(fm, `\.`) {
		t.Errorf("frontmatter tokens missing regex: %s", fm)
	}
}

func TestBuildFrontmatter_CompanionTitleOverride(t *testing.T) {
	rule := makeRule("Avoid", "existence", "error")
	// Simulate companion .md providing a title override
	rule.CompanionMD = "Some prose content."
	// Title override is set on the rule directly
	// (the parser reads the companion frontmatter title)
	// In generator, we accept an optional title override via a separate field.
	// For this test, we verify DisplayName is used when no override is given.
	fm, err := generator.BuildFrontmatter(rule)
	if err != nil {
		t.Fatalf("BuildFrontmatter: %v", err)
	}
	if !strings.Contains(fm, "title: Avoid") {
		t.Errorf("frontmatter title should be 'Avoid' (display name): %s", fm)
	}
}

func TestBuildFrontmatter_OmitsZeroValueFields(t *testing.T) {
	rule := makeRule("Avoid", "existence", "error")
	// No tokens, no swap, no link, etc.
	fm, err := generator.BuildFrontmatter(rule)
	if err != nil {
		t.Fatalf("BuildFrontmatter: %v", err)
	}
	// Fields that are empty/zero should not appear
	if strings.Contains(fm, "tokens:") {
		t.Errorf("frontmatter should omit empty tokens: %s", fm)
	}
	if strings.Contains(fm, "swap:") {
		t.Errorf("frontmatter should omit nil swap: %s", fm)
	}
	if strings.Contains(fm, "link:") {
		t.Errorf("frontmatter should omit empty link: %s", fm)
	}
}

func TestBuildFrontmatter_IgnorecaseIncluded(t *testing.T) {
	rule := makeRule("Avoid", "existence", "error")
	rule.Ignorecase = true
	fm, err := generator.BuildFrontmatter(rule)
	if err != nil {
		t.Fatalf("BuildFrontmatter: %v", err)
	}
	if !strings.Contains(fm, "ignorecase: true") {
		t.Errorf("frontmatter missing ignorecase: %s", fm)
	}
}

func TestBuildFrontmatter_MultipleCategories(t *testing.T) {
	rule := makeRule("Avoid", "existence", "error")
	rule.Category = "Formatting,Style"
	fm, err := generator.BuildFrontmatter(rule)
	if err != nil {
		t.Fatalf("BuildFrontmatter: %v", err)
	}
	if !strings.Contains(fm, "Formatting") {
		t.Errorf("frontmatter missing Formatting category: %s", fm)
	}
	if !strings.Contains(fm, "Style") {
		t.Errorf("frontmatter missing Style category: %s", fm)
	}
}

// ── GenerateRule ──────────────────────────────────────────────────────────────

func TestGenerateRule_CreatesFile(t *testing.T) {
	outDir := t.TempDir()
	rule := makeRule("Avoid", "existence", "error")
	err := generator.GenerateRule(rule, outDir)
	if err != nil {
		t.Fatalf("GenerateRule: %v", err)
	}
	path := filepath.Join(outDir, "avoid.md")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist", path)
	}
}

func TestGenerateRule_ContentHasFrontmatter(t *testing.T) {
	outDir := t.TempDir()
	rule := makeRule("OxfordComma", "existence", "warning")
	rule.Tokens = []string{"and", "or"}
	if err := generator.GenerateRule(rule, outDir); err != nil {
		t.Fatalf("GenerateRule: %v", err)
	}
	content := readFile(t, filepath.Join(outDir, "oxfordcomma.md"))
	if !strings.HasPrefix(content, "---\n") {
		t.Errorf("expected frontmatter delimiter at start, got: %q", content[:min(50, len(content))])
	}
	if !strings.Contains(content, "Oxford Comma") {
		t.Errorf("expected display name in frontmatter: %s", content)
	}
}

func TestGenerateRule_BodyIsCompanionProse(t *testing.T) {
	outDir := t.TempDir()
	rule := makeRule("Avoid", "existence", "error")
	rule.CompanionMD = "Use this rule to flag terms that should be avoided."
	if err := generator.GenerateRule(rule, outDir); err != nil {
		t.Fatalf("GenerateRule: %v", err)
	}
	content := readFile(t, filepath.Join(outDir, "avoid.md"))
	if !strings.Contains(content, "Use this rule to flag terms") {
		t.Errorf("body should contain companion prose: %s", content)
	}
}

func TestGenerateRule_BodyIsAutoGeneratedWhenNoCompanion(t *testing.T) {
	outDir := t.TempDir()
	rule := makeRule("Terms", "substitution", "warning")
	rule.CompanionMD = "" // no companion
	if err := generator.GenerateRule(rule, outDir); err != nil {
		t.Fatalf("GenerateRule: %v", err)
	}
	content := readFile(t, filepath.Join(outDir, "terms.md"))
	// Should have auto-generated description
	if !strings.Contains(content, "Terms") {
		t.Errorf("auto-generated body should mention rule name: %s", content)
	}
}

func TestGenerateRule_FilenameIsLowercase(t *testing.T) {
	outDir := t.TempDir()
	rule := makeRule("HeadingPunctuation", "existence", "warning")
	if err := generator.GenerateRule(rule, outDir); err != nil {
		t.Fatalf("GenerateRule: %v", err)
	}
	// Should be lowercase
	path := filepath.Join(outDir, "headingpunctuation.md")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected lowercase filename %s", path)
	}
}

// ── GenerateSite ──────────────────────────────────────────────────────────────

func TestGenerateSite_CreatesExpectedStructure(t *testing.T) {
	outDir := t.TempDir()
	rules := []*parser.ValeRule{
		makeRule("Avoid", "existence", "error"),
		makeRule("Terms", "substitution", "warning"),
	}
	rules[0].Category = "Formatting"
	rules[1].Category = "Terminology"

	cfg := &config.Config{
		Title:       "Microsoft Style Guide",
		Description: "Rules from the Microsoft Writing Style Guide.",
		BaseURL:     "https://example.com/",
	}

	if err := generator.GenerateSite(&parser.ParseResult{Rules: rules}, cfg, outDir); err != nil {
		t.Fatalf("GenerateSite: %v", err)
	}

	// hugo.toml
	if _, err := os.Stat(filepath.Join(outDir, "hugo.toml")); os.IsNotExist(err) {
		t.Error("expected hugo.toml")
	}
	// content/_index.md
	if _, err := os.Stat(filepath.Join(outDir, "content", "_index.md")); os.IsNotExist(err) {
		t.Error("expected content/_index.md")
	}
	// content/rules/_index.md
	if _, err := os.Stat(filepath.Join(outDir, "content", "rules", "_index.md")); os.IsNotExist(err) {
		t.Error("expected content/rules/_index.md")
	}
	// content/rules/avoid.md
	if _, err := os.Stat(filepath.Join(outDir, "content", "rules", "avoid.md")); os.IsNotExist(err) {
		t.Error("expected content/rules/avoid.md")
	}
	// data/site.json
	if _, err := os.Stat(filepath.Join(outDir, "data", "site.json")); os.IsNotExist(err) {
		t.Error("expected data/site.json")
	}
}

func TestGenerateSite_HugoTOML(t *testing.T) {
	outDir := t.TempDir()
	cfg := &config.Config{
		Title:       "My Style Guide",
		Description: "A style guide.",
		BaseURL:     "https://example.com/",
	}
	if err := generator.GenerateSite(&parser.ParseResult{}, cfg, outDir); err != nil {
		t.Fatalf("GenerateSite: %v", err)
	}
	toml := readFile(t, filepath.Join(outDir, "hugo.toml"))
	if !strings.Contains(toml, `title = "My Style Guide"`) {
		t.Errorf("hugo.toml missing title: %s", toml)
	}
	if !strings.Contains(toml, `baseURL = "https://example.com/"`) {
		t.Errorf("hugo.toml missing baseURL: %s", toml)
	}
	// Taxonomies
	if !strings.Contains(toml, "[taxonomies]") {
		t.Errorf("hugo.toml missing [taxonomies]: %s", toml)
	}
	if !strings.Contains(toml, `category = "categories"`) {
		t.Errorf("hugo.toml missing category taxonomy: %s", toml)
	}
	if !strings.Contains(toml, `ruletype = "ruletypes"`) {
		t.Errorf("hugo.toml missing ruletype taxonomy: %s", toml)
	}
	if !strings.Contains(toml, `severity = "severities"`) {
		t.Errorf("hugo.toml missing severity taxonomy: %s", toml)
	}
}

// ── Index ─────────────────────────────────────────────────────────────────────

func TestGenerateIndex_CountsByType(t *testing.T) {
	outDir := t.TempDir()
	rules := []*parser.ValeRule{
		makeRule("Avoid", "existence", "error"),
		makeRule("Terms", "substitution", "warning"),
		makeRule("Acronyms", "conditional", "warning"),
		makeRule("Headings", "existence", "warning"),
	}
	cfg := &config.Config{Title: "Test Guide", BaseURL: "/"}
	if err := generator.GenerateSite(&parser.ParseResult{Rules: rules}, cfg, outDir); err != nil {
		t.Fatalf("GenerateSite: %v", err)
	}

	indexContent := readFile(t, filepath.Join(outDir, "content", "rules", "_index.md"))
	// Should contain total count in frontmatter
	if !strings.Contains(indexContent, "4") {
		t.Errorf("rules _index.md should contain total rule count (4): %s", indexContent)
	}
}

func TestGenerateIndex_SiteJSON(t *testing.T) {
	outDir := t.TempDir()
	rules := []*parser.ValeRule{
		makeRule("Avoid", "existence", "error"),
		makeRule("Terms", "substitution", "warning"),
		makeRule("Acronyms", "conditional", "warning"),
	}
	rules[0].Category = "Formatting"
	rules[1].Category = "Terminology"
	rules[2].Category = "Formatting"

	cfg := &config.Config{Title: "Test Guide", BaseURL: "/"}
	if err := generator.GenerateSite(&parser.ParseResult{Rules: rules}, cfg, outDir); err != nil {
		t.Fatalf("GenerateSite: %v", err)
	}

	data := readFile(t, filepath.Join(outDir, "data", "site.json"))
	var stats map[string]interface{}
	if err := json.Unmarshal([]byte(data), &stats); err != nil {
		t.Fatalf("site.json is not valid JSON: %v\n%s", err, data)
	}
	total, ok := stats["total_rules"]
	if !ok {
		t.Errorf("site.json missing total_rules: %s", data)
	}
	if int(total.(float64)) != 3 {
		t.Errorf("site.json total_rules: got %v, want 3", total)
	}
}

func TestGenerateIndex_HomepageIndex(t *testing.T) {
	outDir := t.TempDir()
	cfg := &config.Config{
		Title:       "My Style Guide",
		Description: "A style guide.",
		BaseURL:     "/",
	}
	if err := generator.GenerateSite(&parser.ParseResult{}, cfg, outDir); err != nil {
		t.Fatalf("GenerateSite: %v", err)
	}
	index := readFile(t, filepath.Join(outDir, "content", "_index.md"))
	if !strings.Contains(index, "My Style Guide") {
		t.Errorf("homepage _index.md missing title: %s", index)
	}
}

// ── CategoryAssignment ────────────────────────────────────────────────────────

func TestCategoryAssignment_FallsBackToExtends(t *testing.T) {
	outDir := t.TempDir()
	rule := makeRule("Avoid", "existence", "error")
	rule.Category = "" // no category set

	cfg := &config.Config{Title: "Test", BaseURL: "/"}
	if err := generator.GenerateSite(&parser.ParseResult{Rules: []*parser.ValeRule{rule}}, cfg, outDir); err != nil {
		t.Fatalf("GenerateSite: %v", err)
	}
	content := readFile(t, filepath.Join(outDir, "content", "rules", "avoid.md"))
	// Category should fall back to extends type
	if !strings.Contains(content, "existence") {
		t.Errorf("category should fall back to extends type: %s", content)
	}
}

// ── AssignCategories ─────────────────────────────────────────────────────────

func TestAssignCategories_FromConfig(t *testing.T) {
	rules := []*parser.ValeRule{
		makeRule("Avoid", "existence", "error"),
		makeRule("Terms", "substitution", "warning"),
		makeRule("Acronyms", "conditional", "warning"),
	}
	cfg := &config.Config{
		Title:   "Test",
		BaseURL: "/",
		Categories: map[string][]string{
			"Formatting":  {"Avoid", "Terms"},
			"Terminology": {"Terms", "Acronyms"},
		},
	}
	generator.AssignCategories(rules, cfg)

	find := func(name string) *parser.ValeRule {
		for _, r := range rules {
			if r.Name == name {
				return r
			}
		}
		return nil
	}

	avoid := find("Avoid")
	if avoid == nil {
		t.Fatal("Avoid not found")
	}
	if !strings.Contains(avoid.Category, "Formatting") {
		t.Errorf("Avoid should be in Formatting, got: %q", avoid.Category)
	}

	terms := find("Terms")
	if terms == nil {
		t.Fatal("Terms not found")
	}
	// Terms is in both Formatting and Terminology
	if !strings.Contains(terms.Category, "Formatting") || !strings.Contains(terms.Category, "Terminology") {
		t.Errorf("Terms should be in Formatting,Terminology, got: %q", terms.Category)
	}

	acronyms := find("Acronyms")
	if acronyms == nil {
		t.Fatal("Acronyms not found")
	}
	if !strings.Contains(acronyms.Category, "Terminology") {
		t.Errorf("Acronyms should be in Terminology, got: %q", acronyms.Category)
	}
}

func TestAssignCategories_UnassignedFallsBackToExtends(t *testing.T) {
	rules := []*parser.ValeRule{
		makeRule("Orphan", "spelling", "warning"),
	}
	cfg := &config.Config{
		Title:      "Test",
		BaseURL:    "/",
		Categories: map[string][]string{"Other": {"Avoid"}},
	}
	generator.AssignCategories(rules, cfg)
	if rules[0].Category != "spelling" {
		t.Errorf("unassigned rule should fall back to extends %q, got: %q", "spelling", rules[0].Category)
	}
}

// ── GenerateSite with guidelines ──────────────────────────────────────────────

func TestGenerateSite_WithGuidelines(t *testing.T) {
	outDir := t.TempDir()
	result := &parser.ParseResult{
		Rules: []*parser.ValeRule{
			makeRule("Avoid", "existence", "error"),
		},
		Guidelines: []*parser.Guideline{
			{Name: "voice-and-tone", Title: "Voice and Tone", Weight: 10, Body: "Write clearly."},
			{Name: "inclusive-language", Title: "Inclusive Language", Weight: 20, Body: "Be inclusive."},
		},
	}
	cfg := &config.Config{Title: "Test Guide", BaseURL: "/"}

	if err := generator.GenerateSite(result, cfg, outDir); err != nil {
		t.Fatalf("GenerateSite: %v", err)
	}

	// content/guidelines/_index.md
	indexPath := filepath.Join(outDir, "content", "guidelines", "_index.md")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Error("expected content/guidelines/_index.md")
	}

	// content/guidelines/voice-and-tone.md
	vtPath := filepath.Join(outDir, "content", "guidelines", "voice-and-tone.md")
	if _, err := os.Stat(vtPath); os.IsNotExist(err) {
		t.Error("expected content/guidelines/voice-and-tone.md")
	}

	vtContent := readFile(t, vtPath)
	if !strings.Contains(vtContent, "type: guideline") {
		t.Errorf("guideline page missing type: guideline: %s", vtContent)
	}
}

func TestGenerateSite_NoGuidelines_NoGuidelinesDir(t *testing.T) {
	outDir := t.TempDir()
	result := &parser.ParseResult{
		Rules: []*parser.ValeRule{makeRule("Avoid", "existence", "error")},
	}
	cfg := &config.Config{Title: "Test Guide", BaseURL: "/"}

	if err := generator.GenerateSite(result, cfg, outDir); err != nil {
		t.Fatalf("GenerateSite: %v", err)
	}

	guidelinesDir := filepath.Join(outDir, "content", "guidelines")
	if _, err := os.Stat(guidelinesDir); !os.IsNotExist(err) {
		t.Error("content/guidelines/ should not exist when there are no guidelines")
	}
}

func TestGenerateSite_GuidelinesDisabled(t *testing.T) {
	outDir := t.TempDir()
	disabled := false
	result := &parser.ParseResult{
		Rules: []*parser.ValeRule{makeRule("Avoid", "existence", "error")},
		Guidelines: []*parser.Guideline{
			{Name: "voice-and-tone", Title: "Voice and Tone", Body: "Content."},
		},
	}
	cfg := &config.Config{
		Title:      "Test Guide",
		BaseURL:    "/",
		Guidelines: config.GuidelinesConfig{Enabled: &disabled},
	}

	if err := generator.GenerateSite(result, cfg, outDir); err != nil {
		t.Fatalf("GenerateSite: %v", err)
	}

	guidelinesDir := filepath.Join(outDir, "content", "guidelines")
	if _, err := os.Stat(guidelinesDir); !os.IsNotExist(err) {
		t.Error("content/guidelines/ should not exist when guidelines are disabled")
	}
}
