package parser_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/larah/rulebound/internal/parser"
)

// testdata returns the absolute path to the testdata directory.
func testdata(parts ...string) string {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(file), "testdata", "Microsoft")
	return filepath.Join(append([]string{dir}, parts...)...)
}

// ── ParseRule: real Vale rule samples ────────────────────────────────────────

func TestParseRule_Existence_Avoid(t *testing.T) {
	rule, err := parser.ParseRule(testdata("Avoid.yml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rule.Name != "Avoid" {
		t.Errorf("Name: got %q, want %q", rule.Name, "Avoid")
	}
	if rule.Extends != parser.ExtendsExistence {
		t.Errorf("Extends: got %q, want %q", rule.Extends, parser.ExtendsExistence)
	}
	if rule.Level != "error" {
		t.Errorf("Level: got %q, want %q", rule.Level, "error")
	}
	if !rule.Ignorecase {
		t.Error("Ignorecase: got false, want true")
	}
	if len(rule.Tokens) == 0 {
		t.Error("Tokens: expected non-empty slice")
	}
	// Check one token with regex chars
	found := false
	for _, tok := range rule.Tokens {
		if tok == `app(?:lication)?s? (?:developer|program)` {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Tokens: expected to find regex token, got: %v", rule.Tokens)
	}
	if rule.SourceFile != testdata("Avoid.yml") {
		t.Errorf("SourceFile: got %q, want %q", rule.SourceFile, testdata("Avoid.yml"))
	}
}

func TestParseRule_Existence_Avoid_CompanionMD(t *testing.T) {
	rule, err := parser.ParseRule(testdata("Avoid.yml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rule.CompanionMD == "" {
		t.Fatal("CompanionMD: expected non-empty string")
	}
	// Should not start with "---" (frontmatter should be stripped)
	if len(rule.CompanionMD) >= 3 && rule.CompanionMD[:3] == "---" {
		t.Error("CompanionMD should not contain frontmatter")
	}
}

func TestParseRule_Existence_Plurals_RawOnly(t *testing.T) {
	rule, err := parser.ParseRule(testdata("Plurals.yml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rule.Extends != parser.ExtendsExistence {
		t.Errorf("Extends: got %q, want %q", rule.Extends, parser.ExtendsExistence)
	}
	if len(rule.Tokens) != 0 {
		t.Errorf("Tokens: expected empty, got %v", rule.Tokens)
	}
	if len(rule.Raw) == 0 {
		t.Error("Raw: expected non-empty (Plurals.yml has raw but no tokens)")
	}
	if rule.Raw[0] != `\(s\)|\(es\)` {
		t.Errorf("Raw[0]: got %q, want %q", rule.Raw[0], `\(s\)|\(es\)`)
	}
}

func TestParseRule_Existence_We_Tokens(t *testing.T) {
	rule, err := parser.ParseRule(testdata("We.yml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rule.Extends != parser.ExtendsExistence {
		t.Errorf("Extends: got %q", rule.Extends)
	}
	// "we'(?:ve|re)" must be preserved with regex chars
	found := false
	for _, tok := range rule.Tokens {
		if tok == `we'(?:ve|re)` {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Tokens: expected we'(?:ve|re) in %v", rule.Tokens)
	}
}

func TestParseRule_Existence_GeneralURL_ActionWithParams(t *testing.T) {
	rule, err := parser.ParseRule(testdata("GeneralURL.yml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rule.Extends != parser.ExtendsExistence {
		t.Errorf("Extends: got %q", rule.Extends)
	}
	if rule.Action == nil {
		t.Fatal("Action: expected non-nil")
	}
	if rule.Action.Name != "replace" {
		t.Errorf("Action.Name: got %q, want %q", rule.Action.Name, "replace")
	}
	if len(rule.Action.Params) != 2 {
		t.Errorf("Action.Params: got %v, want [URL address]", rule.Action.Params)
	}
	if rule.Action.Params[0] != "URL" || rule.Action.Params[1] != "address" {
		t.Errorf("Action.Params: got %v", rule.Action.Params)
	}
}

func TestParseRule_Existence_ActionStringForm(t *testing.T) {
	rule, err := parser.ParseRule(testdata("ActionString.yml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule.Action == nil {
		t.Fatal("Action: expected non-nil for string-form action")
	}
	if rule.Action.Name != "suggest" {
		t.Errorf("Action.Name: got %q, want %q", rule.Action.Name, "suggest")
	}
	if rule.Action.Params != nil {
		t.Errorf("Action.Params: expected nil for string-form, got %v", rule.Action.Params)
	}
}

func TestParseRule_Substitution_Terms(t *testing.T) {
	rule, err := parser.ParseRule(testdata("Terms.yml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rule.Extends != parser.ExtendsSubstitution {
		t.Errorf("Extends: got %q, want %q", rule.Extends, parser.ExtendsSubstitution)
	}
	if rule.Level != "warning" {
		t.Errorf("Level: got %q, want %q", rule.Level, "warning")
	}
	if len(rule.Swap) == 0 {
		t.Error("Swap: expected non-empty map")
	}
	// Check a specific swap entry with regex key
	val, ok := rule.Swap[`(?:agent|virtual assistant|intelligent personal assistant)`]
	if !ok {
		t.Errorf("Swap: missing regex key, keys: %v", rule.Swap)
	}
	if val != "personal digital assistant" {
		t.Errorf("Swap value: got %q, want %q", val, "personal digital assistant")
	}
	// action: {name: replace} with no params
	if rule.Action == nil {
		t.Fatal("Action: expected non-nil for Terms.yml")
	}
	if rule.Action.Name != "replace" {
		t.Errorf("Action.Name: got %q, want %q", rule.Action.Name, "replace")
	}
}

func TestParseRule_Conditional_Acronyms(t *testing.T) {
	rule, err := parser.ParseRule(testdata("Acronyms.yml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rule.Extends != parser.ExtendsConditional {
		t.Errorf("Extends: got %q, want %q", rule.Extends, parser.ExtendsConditional)
	}
	if rule.First != `\b([A-Z]{3,5})\b` {
		t.Errorf("First: got %q", rule.First)
	}
	if rule.Second != `(?:\b[A-Z][a-z]+ )+\(([A-Z]{3,5})\)` {
		t.Errorf("Second: got %q", rule.Second)
	}
	if len(rule.Exceptions) == 0 {
		t.Error("Exceptions: expected non-empty")
	}
	// Spot-check a few exceptions
	found := false
	for _, ex := range rule.Exceptions {
		if ex == "API" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Exceptions: expected 'API' in list")
	}
}

func TestParseRule_Capitalization_Headings(t *testing.T) {
	rule, err := parser.ParseRule(testdata("Headings.yml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rule.Extends != parser.ExtendsCapitalization {
		t.Errorf("Extends: got %q, want %q", rule.Extends, parser.ExtendsCapitalization)
	}
	if rule.Scope != "heading" {
		t.Errorf("Scope: got %q, want %q", rule.Scope, "heading")
	}
	if rule.Match != "$sentence" {
		t.Errorf("Match: got %q, want %q", rule.Match, "$sentence")
	}
	if len(rule.Indicators) == 0 {
		t.Error("Indicators: expected non-empty")
	}
	if rule.Indicators[0] != ":" {
		t.Errorf("Indicators[0]: got %q, want ':'", rule.Indicators[0])
	}
	if len(rule.Exceptions) == 0 {
		t.Error("Exceptions: expected non-empty")
	}
}

func TestParseRule_Occurrence_SentenceLength(t *testing.T) {
	rule, err := parser.ParseRule(testdata("SentenceLength.yml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rule.Extends != parser.ExtendsOccurrence {
		t.Errorf("Extends: got %q, want %q", rule.Extends, parser.ExtendsOccurrence)
	}
	if rule.Max != 30 {
		t.Errorf("Max: got %d, want 30", rule.Max)
	}
	if rule.Token != `\b(\w+)\b` {
		t.Errorf("Token: got %q, want %q", rule.Token, `\b(\w+)\b`)
	}
	if rule.Scope != "sentence" {
		t.Errorf("Scope: got %q, want %q", rule.Scope, "sentence")
	}
}

// ── ParseRule: companion .md absent ──────────────────────────────────────────

func TestParseRule_NoCompanionMD(t *testing.T) {
	rule, err := parser.ParseRule(testdata("We.yml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// We.yml has no companion .md file in testdata
	if rule.CompanionMD != "" {
		t.Errorf("CompanionMD: expected empty when .md absent, got %q", rule.CompanionMD)
	}
}

// ── ParseRule: edge cases ─────────────────────────────────────────────────────

func TestParseRule_MalformedYAML(t *testing.T) {
	_, err := parser.ParseRule(testdata("malformed.yml"))
	if err == nil {
		t.Error("expected error for malformed YAML, got nil")
	}
}

func TestParseRule_EmptyFile(t *testing.T) {
	_, err := parser.ParseRule(testdata("empty.yml"))
	if err == nil {
		t.Error("expected error for empty YAML file, got nil")
	}
}

func TestParseRule_UnknownExtendsType(t *testing.T) {
	rule, err := parser.ParseRule(testdata("UnknownExtends.yml"))
	if err != nil {
		t.Fatalf("unexpected error for unknown extends type: %v", err)
	}
	if rule.Extends != "futuristic" {
		t.Errorf("Extends: got %q, want %q", rule.Extends, "futuristic")
	}
}

func TestParseRule_MissingExtendsField(t *testing.T) {
	_, err := parser.ParseRule(testdata("NoExtends.yml"))
	if err == nil {
		t.Error("expected error when extends field missing, got nil")
	}
}

func TestParseRule_ExtraFieldsPassthrough(t *testing.T) {
	rule, err := parser.ParseRule(testdata("ExtraFields.yml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule.Extra == nil {
		t.Fatal("Extra: expected non-nil map for unknown fields")
	}
	if _, ok := rule.Extra["custom_field"]; !ok {
		t.Error("Extra: expected custom_field key")
	}
	if _, ok := rule.Extra["another_extra"]; !ok {
		t.Error("Extra: expected another_extra key")
	}
	// Known fields must NOT appear in Extra
	if _, ok := rule.Extra["extends"]; ok {
		t.Error("Extra: 'extends' is a known field and must not appear in Extra")
	}
	if _, ok := rule.Extra["tokens"]; ok {
		t.Error("Extra: 'tokens' is a known field and must not appear in Extra")
	}
}

func TestParseRule_NonExistentFile(t *testing.T) {
	_, err := parser.ParseRule(testdata("doesnotexist.yml"))
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

// ── ParsePackage ─────────────────────────────────────────────────────────────

func TestParsePackage_Microsoft(t *testing.T) {
	result, err := parser.ParsePackage(testdata())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Expect warnings for malformed.yml, empty.yml, NoExtends.yml
	if len(result.Warnings) == 0 {
		t.Error("ParsePackage: expected warnings for unparseable files")
	}

	if len(result.Rules) == 0 {
		t.Fatal("ParsePackage: expected rules, got empty slice")
	}

	// Find rule by name
	find := func(name string) *parser.ValeRule {
		for _, r := range result.Rules {
			if r.Name == name {
				return r
			}
		}
		return nil
	}

	if find("Avoid") == nil {
		t.Error("ParsePackage: expected 'Avoid' rule")
	}
	if find("Terms") == nil {
		t.Error("ParsePackage: expected 'Terms' rule")
	}
	if find("Acronyms") == nil {
		t.Error("ParsePackage: expected 'Acronyms' rule")
	}

	// meta.json must not appear
	if find("meta") != nil {
		t.Error("ParsePackage: 'meta' should not be present (meta.json skipped)")
	}
}

func TestParsePackage_SortedByName(t *testing.T) {
	result, err := parser.ParsePackage(testdata())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i := 1; i < len(result.Rules); i++ {
		if result.Rules[i].Name < result.Rules[i-1].Name {
			t.Errorf("ParsePackage: rules not sorted at index %d: %q < %q",
				i, result.Rules[i].Name, result.Rules[i-1].Name)
		}
	}
}

func TestParsePackage_PackageName(t *testing.T) {
	result, err := parser.ParsePackage(testdata())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Rules) == 0 {
		t.Fatal("no rules returned")
	}
	for _, r := range result.Rules {
		if r.Category != "Microsoft" {
			t.Errorf("rule %q Category (package name): got %q, want %q", r.Name, r.Category, "Microsoft")
		}
	}
}

func TestParsePackage_EmptyDir(t *testing.T) {
	result, err := parser.ParsePackage(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error for empty dir: %v", err)
	}
	if len(result.Rules) != 0 {
		t.Errorf("ParsePackage on empty dir: expected 0 rules, got %d", len(result.Rules))
	}
}

func TestParsePackage_NonExistentDir(t *testing.T) {
	_, err := parser.ParsePackage("/does/not/exist/xyz")
	if err == nil {
		t.Error("expected error for non-existent directory, got nil")
	}
}

func TestParsePackage_IncludesGuidelines(t *testing.T) {
	result, err := parser.ParsePackage(testdata())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// testdata/Microsoft/guidelines/ has 3 valid guidelines
	if len(result.Guidelines) != 3 {
		t.Errorf("expected 3 guidelines, got %d", len(result.Guidelines))
	}
}

func TestParsePackage_IncludesPages(t *testing.T) {
	dir := t.TempDir()
	pagesDir := filepath.Join(dir, "pages")
	if err := os.MkdirAll(pagesDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pagesDir, "test-page.md"),
		[]byte("---\ntitle: Test Page\n---\nPage body."), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	result, err := parser.ParsePackage(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Pages == nil {
		t.Fatal("expected non-nil Pages when pages/ directory exists")
	}
	if len(result.Pages.Pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(result.Pages.Pages))
	}
	if result.Pages.Pages[0].Title != "Test Page" {
		t.Errorf("page title = %q, want %q", result.Pages.Pages[0].Title, "Test Page")
	}
}

func TestParsePackage_NoPagesDir(t *testing.T) {
	result, err := parser.ParsePackage(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Pages != nil {
		t.Errorf("expected nil Pages when pages/ absent, got %+v", result.Pages)
	}
}

// ── Polymorphic YAML field tests ──────────────────────────────────────────────

// TestParseRule_SwapAsList verifies that swap as a YAML sequence of single-key
// maps (- key: value) is parsed into the same map[string]string as the standard
// map form.
func TestParseRule_SwapAsList(t *testing.T) {
	rule, err := parser.ParseRule(testdata("SwapList.yml"))
	if err != nil {
		t.Fatalf("ParseRule returned unexpected error: %v", err)
	}
	if rule.Swap == nil {
		t.Fatal("Swap is nil, expected map with 2 entries")
	}
	if len(rule.Swap) != 2 {
		t.Fatalf("Swap length = %d, want 2", len(rule.Swap))
	}
	if rule.Swap["kb"] != "documentation" {
		t.Errorf("Swap[kb] = %q, want %q", rule.Swap["kb"], "documentation")
	}
	if rule.Swap["knowledge base"] != "docs" {
		t.Errorf("Swap[knowledge base] = %q, want %q", rule.Swap["knowledge base"], "docs")
	}
}

// TestParseRule_ScopeAsList verifies that scope as a YAML sequence
// (- list\n- heading) is joined into a comma-separated string.
func TestParseRule_ScopeAsList(t *testing.T) {
	rule, err := parser.ParseRule(testdata("ScopeList.yml"))
	if err != nil {
		t.Fatalf("ParseRule returned unexpected error: %v", err)
	}
	// Scope should contain both values.
	if rule.Scope == "" {
		t.Fatal("Scope is empty, expected list and heading")
	}
	for _, want := range []string{"list", "heading"} {
		if !contains(rule.Scope, want) {
			t.Errorf("Scope %q does not contain %q", rule.Scope, want)
		}
	}
}

// TestParseRule_SequenceTokens verifies that tokens as a list of {tag, pattern}
// objects (used by sequence-type rules) are parsed into Tokens strings.
func TestParseRule_SequenceTokens(t *testing.T) {
	rule, err := parser.ParseRule(testdata("SequenceTokens.yml"))
	if err != nil {
		t.Fatalf("ParseRule returned unexpected error: %v", err)
	}
	if rule.Extends != "sequence" {
		t.Errorf("Extends = %q, want %q", rule.Extends, "sequence")
	}
	if len(rule.Tokens) != 2 {
		t.Fatalf("Tokens length = %d, want 2", len(rule.Tokens))
	}
	// Each token should encode the tag and pattern information.
	for i, tok := range rule.Tokens {
		if tok == "" {
			t.Errorf("Tokens[%d] is empty", i)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
