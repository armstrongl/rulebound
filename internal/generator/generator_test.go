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
	// Should use behavioral verb, not "is a ... rule"
	if strings.Contains(desc, "is a") {
		t.Errorf("description should not use 'is a' phrasing: %q", desc)
	}
	if !strings.Contains(desc, "flags") {
		t.Errorf("description missing verb 'flags' for existence type: %q", desc)
	}
	// %s-containing sentences should be dropped
	if strings.Contains(desc, "%s") {
		t.Errorf("description should not contain %%s: %q", desc)
	}
	if strings.Contains(desc, "Don't use") {
		t.Errorf("description should not contain salvaged %%s sentence: %q", desc)
	}
}

func TestAutoDescription_WithLink(t *testing.T) {
	rule := makeRule("Terms", "substitution", "warning")
	rule.Link = "https://docs.microsoft.com/en-us/style-guide"
	desc := generator.AutoDescription(rule)
	// Link sentence should be absent (rendered by Hugo theme).
	if strings.Contains(desc, "docs.microsoft.com") {
		t.Errorf("description should not contain link domain: %q", desc)
	}
	if strings.Contains(desc, "See the") {
		t.Errorf("description should not contain link sentence: %q", desc)
	}
}

func TestAutoDescription_WithTokens(t *testing.T) {
	rule := makeRule("Avoid", "existence", "error")
	rule.Tokens = []string{"foo", "bar", "baz"}
	desc := generator.AutoDescription(rule)
	// Token sentence should be absent (rendered by rule-details.html).
	if strings.Contains(desc, "foo") {
		t.Errorf("description should not contain tokens: %q", desc)
	}
	if strings.Contains(desc, "flags the following") {
		t.Errorf("description should not contain token sentence: %q", desc)
	}
}

func TestAutoDescription_WithSwap(t *testing.T) {
	rule := makeRule("Terms", "substitution", "warning")
	rule.Swap = map[string]string{"adaptor": "adapter", "afterwards": "afterward"}
	desc := generator.AutoDescription(rule)
	// Should show concrete examples, not just count
	if !strings.Contains(desc, "adapter") {
		t.Errorf("description missing swap example: %q", desc)
	}
	if !strings.Contains(desc, "adaptor") {
		t.Errorf("description missing swap key: %q", desc)
	}
	if !strings.Contains(desc, "2 substitutions total") {
		t.Errorf("description missing swap total: %q", desc)
	}
}

func TestAutoDescription_VerbByType(t *testing.T) {
	cases := []struct {
		extends string
		verb    string
	}{
		{"existence", "flags"},
		{"substitution", "suggests preferred alternatives for"},
		{"occurrence", "limits"},
		{"repetition", "limits repetition of"},
		{"consistency", "enforces consistent usage of"},
		{"conditional", "checks that"},
		{"capitalization", "enforces capitalization of"},
		{"metric", "evaluates readability of"},
		{"script", "applies a custom check to"},
		{"spelling", "checks spelling of"},
		{"sequence", "detects patterns in"},
	}
	for _, tc := range cases {
		rule := makeRule("TestRule", tc.extends, "warning")
		rule.Message = "" // avoid message interference
		desc := generator.AutoDescription(rule)
		if !strings.Contains(desc, tc.verb) {
			t.Errorf("AutoDescription(extends=%q) = %q, missing verb %q", tc.extends, desc, tc.verb)
		}
	}
}

func TestAutoDescription_UnrecognizedExtends(t *testing.T) {
	rule := makeRule("TestRule", "futuristic", "warning")
	rule.Message = ""
	desc := generator.AutoDescription(rule)
	if !strings.Contains(desc, "checks") {
		t.Errorf("unrecognized extends should use 'checks' fallback: %q", desc)
	}
}

func TestAutoDescription_ScopeInOpening(t *testing.T) {
	rule := makeRule("Headings", "capitalization", "suggestion")
	rule.Message = ""
	rule.Scope = "heading"
	desc := generator.AutoDescription(rule)
	if !strings.Contains(desc, "heading") {
		t.Errorf("description should contain scope 'heading': %q", desc)
	}
}

func TestAutoDescription_MessageWithFormatVerbs(t *testing.T) {
	rule := makeRule("Avoid", "existence", "error")
	rule.Message = "Don't use '%s'."
	desc := generator.AutoDescription(rule)
	// The entire message has %s so nothing should be salvaged
	if strings.Contains(desc, "Don't use") {
		t.Errorf("message with %%s should be dropped: %q", desc)
	}
}

func TestAutoDescription_MessageClean(t *testing.T) {
	rule := makeRule("SentenceLength", "occurrence", "suggestion")
	rule.Message = "Try to keep sentences short (< 30 words)."
	rule.Scope = "sentence"
	desc := generator.AutoDescription(rule)
	if !strings.Contains(desc, "Try to keep sentences short") {
		t.Errorf("clean message should be kept: %q", desc)
	}
}

func TestAutoDescription_MessageMixed(t *testing.T) {
	rule := makeRule("Avoid", "existence", "error")
	rule.Message = "Don't use '%s'. See the A-Z word list for details."
	desc := generator.AutoDescription(rule)
	// The first sentence with %s should be dropped
	if strings.Contains(desc, "Don't use") {
		t.Errorf("sentence with %%s should be dropped: %q", desc)
	}
	// The second clean sentence should be kept
	if !strings.Contains(desc, "See the A-Z word list for details") {
		t.Errorf("clean sentence should be salvaged: %q", desc)
	}
}

func TestAutoDescription_SwapSamplerExamples(t *testing.T) {
	rule := makeRule("Terms", "substitution", "warning")
	rule.Message = ""
	rule.Swap = map[string]string{
		"adaptor":                     "adapter",
		"afterwards":                  "afterward",
		"(?:agent|virtual assistant)": "personal digital assistant",
	}
	desc := generator.AutoDescription(rule)
	// Should show non-regex examples
	if !strings.Contains(desc, "adapter") {
		t.Errorf("description missing swap example: %q", desc)
	}
	// Regex key should be filtered out
	if strings.Contains(desc, "agent") {
		t.Errorf("description should not show regex key: %q", desc)
	}
	if !strings.Contains(desc, "3 substitutions total") {
		t.Errorf("description missing total count: %q", desc)
	}
}

func TestAutoDescription_SwapSamplerOneEntry(t *testing.T) {
	rule := makeRule("Terms", "substitution", "warning")
	rule.Message = ""
	rule.Swap = map[string]string{"adaptor": "adapter"}
	desc := generator.AutoDescription(rule)
	if !strings.Contains(desc, "Suggests using") {
		t.Errorf("single-entry swap should use compact format: %q", desc)
	}
	if strings.Contains(desc, "total") {
		t.Errorf("single-entry swap should not show total count: %q", desc)
	}
}

func TestAutoDescription_SwapSamplerAllRegex(t *testing.T) {
	rule := makeRule("Terms", "substitution", "warning")
	rule.Message = ""
	rule.Swap = map[string]string{
		"(?:agent|virtual assistant)": "personal digital assistant",
		"(?:drive C:|drive C>)":       "drive C",
	}
	desc := generator.AutoDescription(rule)
	// All keys are regex, should fall back to count-only
	if !strings.Contains(desc, "replacements for 2 terms") {
		t.Errorf("all-regex swap should use count-only format: %q", desc)
	}
}

func TestAutoDescription_NoTokenSentence(t *testing.T) {
	rule := makeRule("Avoid", "existence", "error")
	rule.Message = ""
	rule.Tokens = []string{"foo", "bar", "baz"}
	desc := generator.AutoDescription(rule)
	if strings.Contains(desc, "flags the following patterns") {
		t.Errorf("token sentence should be absent: %q", desc)
	}
}

func TestAutoDescription_NoLinkSentence(t *testing.T) {
	rule := makeRule("Avoid", "existence", "error")
	rule.Message = ""
	rule.Link = "https://example.com/style-guide"
	desc := generator.AutoDescription(rule)
	if strings.Contains(desc, "See the") {
		t.Errorf("link sentence should be absent: %q", desc)
	}
	if strings.Contains(desc, "example.com") {
		t.Errorf("link domain should be absent: %q", desc)
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

func TestGenerateIndex_SiteJSON_WithGuidelines(t *testing.T) {
	outDir := t.TempDir()
	result := &parser.ParseResult{
		Rules: []*parser.ValeRule{
			makeRule("Avoid", "existence", "error"),
		},
		Guidelines: []*parser.Guideline{
			{Name: "voice-and-tone", Title: "Voice and Tone", Body: "Content."},
			{Name: "inclusive", Title: "Inclusive Language", Body: "Content."},
		},
	}
	cfg := &config.Config{
		Title:   "Test Guide",
		BaseURL: "/",
		Guidelines: config.GuidelinesConfig{
			SectionTitle: "Editorial Guidelines",
		},
	}

	if err := generator.GenerateSite(result, cfg, outDir); err != nil {
		t.Fatalf("GenerateSite: %v", err)
	}

	data := readFile(t, filepath.Join(outDir, "data", "site.json"))
	var stats map[string]interface{}
	if err := json.Unmarshal([]byte(data), &stats); err != nil {
		t.Fatalf("site.json is not valid JSON: %v", err)
	}

	gc, ok := stats["guidelines_count"]
	if !ok {
		t.Fatal("site.json missing guidelines_count")
	}
	if int(gc.(float64)) != 2 {
		t.Errorf("guidelines_count = %v, want 2", gc)
	}

	gst, ok := stats["guidelines_section_title"]
	if !ok {
		t.Fatal("site.json missing guidelines_section_title")
	}
	if gst != "Editorial Guidelines" {
		t.Errorf("guidelines_section_title = %v, want 'Editorial Guidelines'", gst)
	}
}

func TestGenerateIndex_SiteJSON_NoGuidelines_OmitsFields(t *testing.T) {
	outDir := t.TempDir()
	result := &parser.ParseResult{
		Rules: []*parser.ValeRule{makeRule("Avoid", "existence", "error")},
	}
	cfg := &config.Config{Title: "Test Guide", BaseURL: "/"}

	if err := generator.GenerateSite(result, cfg, outDir); err != nil {
		t.Fatalf("GenerateSite: %v", err)
	}

	data := readFile(t, filepath.Join(outDir, "data", "site.json"))
	var stats map[string]interface{}
	if err := json.Unmarshal([]byte(data), &stats); err != nil {
		t.Fatalf("site.json is not valid JSON: %v", err)
	}

	if _, ok := stats["guidelines_count"]; ok {
		t.Error("site.json should omit guidelines_count when 0")
	}
}

func TestGenerateIndex_HomepageWithGuidelines(t *testing.T) {
	outDir := t.TempDir()
	result := &parser.ParseResult{
		Rules: []*parser.ValeRule{makeRule("Avoid", "existence", "error")},
		Guidelines: []*parser.Guideline{
			{Name: "voice-and-tone", Title: "Voice and Tone", Body: "Content."},
		},
	}
	cfg := &config.Config{Title: "Test Guide", BaseURL: "/"}

	if err := generator.GenerateSite(result, cfg, outDir); err != nil {
		t.Fatalf("GenerateSite: %v", err)
	}

	index := readFile(t, filepath.Join(outDir, "content", "_index.md"))
	if !strings.Contains(index, "guidelines_count: 1") {
		t.Errorf("homepage _index.md should contain guidelines_count: %s", index)
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

// ── GenerateSite with pages ───────────────────────────────────────────────────

func TestGenerateSite_WithPages_CreatesPageFiles(t *testing.T) {
	outDir := t.TempDir()
	result := &parser.ParseResult{
		Rules: []*parser.ValeRule{
			makeRule("Avoid", "existence", "error"),
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

	// content/pages/ directory should exist.
	pagesDir := filepath.Join(outDir, "content", "pages")
	if _, err := os.Stat(pagesDir); os.IsNotExist(err) {
		t.Error("expected content/pages/ directory to exist")
	}

	// content/pages/language/active-voice.md should exist.
	pagePath := filepath.Join(outDir, "content", "pages", "language", "active-voice.md")
	if _, err := os.Stat(pagePath); os.IsNotExist(err) {
		t.Error("expected content/pages/language/active-voice.md to exist")
	}

	// data/navigation.json should exist.
	navPath := filepath.Join(outDir, "data", "navigation.json")
	if _, err := os.Stat(navPath); os.IsNotExist(err) {
		t.Error("expected data/navigation.json to exist")
	}

	// Verify navigation.json has correct structure.
	navData := readFile(t, navPath)
	var nav map[string]interface{}
	if err := json.Unmarshal([]byte(navData), &nav); err != nil {
		t.Fatalf("navigation.json is not valid JSON: %v\n%s", err, navData)
	}

	sections, ok := nav["sections"].([]interface{})
	if !ok {
		t.Fatal("expected sections array in navigation.json")
	}
	if len(sections) != 1 {
		t.Errorf("expected 1 section, got %d", len(sections))
	}

	rulesSection, ok := nav["rules_section"].(map[string]interface{})
	if !ok {
		t.Fatal("expected rules_section in navigation.json")
	}
	cats, ok := rulesSection["categories"].([]interface{})
	if !ok {
		t.Fatal("expected categories in rules_section")
	}
	if len(cats) != 1 {
		t.Errorf("expected 1 category, got %d", len(cats))
	}
}

func TestGenerateSite_WithoutPages_GuidelinesRun(t *testing.T) {
	outDir := t.TempDir()
	result := &parser.ParseResult{
		Rules: []*parser.ValeRule{
			makeRule("Avoid", "existence", "error"),
		},
		Guidelines: []*parser.Guideline{
			{Name: "voice-and-tone", Title: "Voice and Tone", Weight: 10, Body: "Write clearly."},
		},
		// Pages is nil — no pages/ directory
	}
	cfg := &config.Config{Title: "Test Guide", BaseURL: "/"}

	if err := generator.GenerateSite(result, cfg, outDir); err != nil {
		t.Fatalf("GenerateSite: %v", err)
	}

	// Guidelines should be generated.
	guidelinesDir := filepath.Join(outDir, "content", "guidelines")
	if _, err := os.Stat(guidelinesDir); os.IsNotExist(err) {
		t.Error("expected content/guidelines/ to exist when no pages and guidelines present")
	}

	// Pages directory should NOT exist.
	pagesDir := filepath.Join(outDir, "content", "pages")
	if _, err := os.Stat(pagesDir); !os.IsNotExist(err) {
		t.Error("content/pages/ should not exist when no pages tree provided")
	}

	// navigation.json should NOT exist.
	navPath := filepath.Join(outDir, "data", "navigation.json")
	if _, err := os.Stat(navPath); !os.IsNotExist(err) {
		t.Error("navigation.json should not exist when no pages tree provided")
	}
}

func TestGenerateSite_WithPages_AND_Guidelines_PagesWins(t *testing.T) {
	outDir := t.TempDir()
	result := &parser.ParseResult{
		Rules: []*parser.ValeRule{
			makeRule("Avoid", "existence", "error"),
		},
		Guidelines: []*parser.Guideline{
			{Name: "voice-and-tone", Title: "Voice and Tone", Weight: 10, Body: "Write clearly."},
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
	cfg := &config.Config{Title: "Test Guide", BaseURL: "/"}

	if err := generator.GenerateSite(result, cfg, outDir); err != nil {
		t.Fatalf("GenerateSite: %v", err)
	}

	// Pages should be generated.
	pagePath := filepath.Join(outDir, "content", "pages", "language", "active-voice.md")
	if _, err := os.Stat(pagePath); os.IsNotExist(err) {
		t.Error("expected page file to exist when pages supersede guidelines")
	}

	// Guidelines should NOT be generated (pages supersedes).
	guidelinesDir := filepath.Join(outDir, "content", "guidelines")
	if _, err := os.Stat(guidelinesDir); !os.IsNotExist(err) {
		t.Error("content/guidelines/ should not exist when pages are present (pages supersedes guidelines)")
	}

	// navigation.json should exist.
	navPath := filepath.Join(outDir, "data", "navigation.json")
	if _, err := os.Stat(navPath); os.IsNotExist(err) {
		t.Error("expected data/navigation.json to exist when pages present")
	}
}

func TestGenerateSite_WithEmptyPages_GuidelinesRun(t *testing.T) {
	outDir := t.TempDir()
	result := &parser.ParseResult{
		Rules: []*parser.ValeRule{
			makeRule("Avoid", "existence", "error"),
		},
		Guidelines: []*parser.Guideline{
			{Name: "voice-and-tone", Title: "Voice and Tone", Weight: 10, Body: "Write clearly."},
		},
		Pages: &parser.SectionTree{
			Name:  "pages",
			Title: "Pages",
			Path:  "/pages/",
			// Empty tree — no pages, no children, no IndexPage
		},
	}
	cfg := &config.Config{Title: "Test Guide", BaseURL: "/"}

	if err := generator.GenerateSite(result, cfg, outDir); err != nil {
		t.Fatalf("GenerateSite: %v", err)
	}

	// Empty pages tree should be treated as "no pages" → guidelines run.
	guidelinesDir := filepath.Join(outDir, "content", "guidelines")
	if _, err := os.Stat(guidelinesDir); os.IsNotExist(err) {
		t.Error("expected content/guidelines/ when pages tree is empty")
	}
}

// ── Resources ──────────────────────────────────────────────────────────────

func TestGenerateSite_SiteJSON_ContainsDefaultResourceLinks(t *testing.T) {
	outDir := t.TempDir()
	result := &parser.ParseResult{
		Rules: []*parser.ValeRule{makeRule("Avoid", "existence", "error")},
	}
	cfg := &config.Config{Title: "Test Guide", BaseURL: "/"}

	if err := generator.GenerateSite(result, cfg, outDir); err != nil {
		t.Fatalf("GenerateSite: %v", err)
	}

	data := readFile(t, filepath.Join(outDir, "data", "site.json"))
	var stats map[string]interface{}
	if err := json.Unmarshal([]byte(data), &stats); err != nil {
		t.Fatalf("site.json is not valid JSON: %v", err)
	}

	links, ok := stats["resource_links"].([]interface{})
	if !ok {
		t.Fatal("site.json missing resource_links array")
	}
	if len(links) != 3 {
		t.Fatalf("resource_links length = %d, want 3 defaults", len(links))
	}

	// Verify first default is Vale
	first := links[0].(map[string]interface{})
	if first["label"] != "Vale" {
		t.Errorf("first default label = %v, want 'Vale'", first["label"])
	}
	if first["url"] != "https://vale.sh" {
		t.Errorf("first default url = %v, want 'https://vale.sh'", first["url"])
	}
}

func TestGenerateSite_SiteJSON_ExtraLinksAppended(t *testing.T) {
	outDir := t.TempDir()
	result := &parser.ParseResult{
		Rules: []*parser.ValeRule{makeRule("Avoid", "existence", "error")},
	}
	cfg := &config.Config{
		Title:   "Test Guide",
		BaseURL: "/",
		Resources: config.ResourcesConfig{
			ExtraLinks: []config.ResourceLink{
				{Label: "Custom", URL: "https://example.com", Description: "A custom link"},
			},
		},
	}

	if err := generator.GenerateSite(result, cfg, outDir); err != nil {
		t.Fatalf("GenerateSite: %v", err)
	}

	data := readFile(t, filepath.Join(outDir, "data", "site.json"))
	var stats map[string]interface{}
	if err := json.Unmarshal([]byte(data), &stats); err != nil {
		t.Fatalf("site.json is not valid JSON: %v", err)
	}

	links := stats["resource_links"].([]interface{})
	if len(links) != 4 {
		t.Fatalf("resource_links length = %d, want 4 (3 defaults + 1 custom)", len(links))
	}

	last := links[3].(map[string]interface{})
	if last["label"] != "Custom" {
		t.Errorf("last link label = %v, want 'Custom'", last["label"])
	}
}

func TestGenerateSite_ResourcesPage_Created(t *testing.T) {
	outDir := t.TempDir()
	result := &parser.ParseResult{
		Rules: []*parser.ValeRule{makeRule("Avoid", "existence", "error")},
	}
	cfg := &config.Config{Title: "Test Guide", BaseURL: "/"}

	if err := generator.GenerateSite(result, cfg, outDir); err != nil {
		t.Fatalf("GenerateSite: %v", err)
	}

	indexPath := filepath.Join(outDir, "content", "resources", "_index.md")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Error("expected content/resources/_index.md to exist")
	}

	content := readFile(t, indexPath)
	if !strings.Contains(content, "type: resources") {
		t.Errorf("resources _index.md missing type: resources: %s", content)
	}
	if !strings.Contains(content, "title: Resources") {
		t.Errorf("resources _index.md missing title: %s", content)
	}
}

func TestGenerateSite_ResourcesPage_Suppressed(t *testing.T) {
	outDir := t.TempDir()
	disabled := false
	result := &parser.ParseResult{
		Rules: []*parser.ValeRule{makeRule("Avoid", "existence", "error")},
	}
	cfg := &config.Config{
		Title:     "Test Guide",
		BaseURL:   "/",
		Resources: config.ResourcesConfig{Enabled: &disabled},
	}

	if err := generator.GenerateSite(result, cfg, outDir); err != nil {
		t.Fatalf("GenerateSite: %v", err)
	}

	// Page should not exist
	indexPath := filepath.Join(outDir, "content", "resources", "_index.md")
	if _, err := os.Stat(indexPath); !os.IsNotExist(err) {
		t.Error("content/resources/_index.md should not exist when resources.enabled is false")
	}

	// But site.json should still have resource_links (footer needs them)
	data := readFile(t, filepath.Join(outDir, "data", "site.json"))
	var stats map[string]interface{}
	if err := json.Unmarshal([]byte(data), &stats); err != nil {
		t.Fatalf("site.json is not valid JSON: %v", err)
	}
	if _, ok := stats["resource_links"]; !ok {
		t.Error("site.json should still contain resource_links even when page is disabled")
	}
}

// ── CountPages ────────────────────────────────────────────────────────────────

func TestCountPages_NilTree(t *testing.T) {
	if got := generator.CountPages(nil); got != 0 {
		t.Errorf("CountPages(nil) = %d, want 0", got)
	}
}

func TestCountPages_EmptyTree(t *testing.T) {
	tree := &parser.SectionTree{Name: "pages", Title: "Pages", Path: "/pages/"}
	if got := generator.CountPages(tree); got != 0 {
		t.Errorf("CountPages(empty) = %d, want 0", got)
	}
}

func TestCountPages_FlatTree(t *testing.T) {
	tree := &parser.SectionTree{
		Name:  "pages",
		Title: "Pages",
		Path:  "/pages/",
		Pages: []*parser.Page{
			{Title: "A", Path: "/pages/a/"},
			{Title: "B", Path: "/pages/b/"},
		},
	}
	if got := generator.CountPages(tree); got != 2 {
		t.Errorf("CountPages(flat) = %d, want 2", got)
	}
}

func TestCountPages_NestedTree(t *testing.T) {
	tree := &parser.SectionTree{
		Name:  "pages",
		Title: "Pages",
		Path:  "/pages/",
		Pages: []*parser.Page{
			{Title: "Root Page", Path: "/pages/root/"},
		},
		Children: []*parser.SectionTree{
			{
				Name:  "language",
				Title: "Language",
				Path:  "/pages/language/",
				Pages: []*parser.Page{
					{Title: "Active Voice", Path: "/pages/language/active-voice/"},
					{Title: "Pronouns", Path: "/pages/language/pronouns/"},
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
	// 1 root + 2 language + 1 advanced + 1 formatting = 5
	if got := generator.CountPages(tree); got != 5 {
		t.Errorf("CountPages(nested) = %d, want 5", got)
	}
}

func TestCountPages_TreeWithIndexPages(t *testing.T) {
	tree := &parser.SectionTree{
		Name:      "pages",
		Title:     "Pages",
		Path:      "/pages/",
		IndexPage: &parser.Page{Title: "Hub", Path: "/pages/"},
		Pages: []*parser.Page{
			{Title: "A", Path: "/pages/a/"},
		},
		Children: []*parser.SectionTree{
			{
				Name:      "language",
				Title:     "Language",
				Path:      "/pages/language/",
				IndexPage: &parser.Page{Title: "Language Hub", Path: "/pages/language/"},
				Pages: []*parser.Page{
					{Title: "Voice", Path: "/pages/language/voice/"},
				},
			},
		},
	}
	// IndexPages count: 2 (root hub + language hub)
	// Pages count: 2 (a + voice)
	// Total: 4
	if got := generator.CountPages(tree); got != 4 {
		t.Errorf("CountPages(with index pages) = %d, want 4", got)
	}
}
