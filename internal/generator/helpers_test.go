package generator

import (
	"testing"

	"github.com/larah/rulebound/internal/parser"
)

// ── ruleVerb ──────────────────────────────────────────────────────────────────

func TestRuleVerb(t *testing.T) {
	cases := []struct {
		extends string
		want    string
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
		{"unknown", "checks"},
		{"", "checks"},
	}
	for _, tc := range cases {
		got := ruleVerb(tc.extends)
		if got != tc.want {
			t.Errorf("ruleVerb(%q) = %q, want %q", tc.extends, got, tc.want)
		}
	}
}

// ── salvageMessage ────────────────────────────────────────────────────────────

func TestSalvageMessage_CleanMessage(t *testing.T) {
	got := salvageMessage("Try to keep sentences short (< 30 words).")
	if got != "Try to keep sentences short (< 30 words)." {
		t.Errorf("clean message should be kept verbatim: %q", got)
	}
}

func TestSalvageMessage_AllFormatVerbs(t *testing.T) {
	got := salvageMessage("Don't use '%s'.")
	if got != "" {
		t.Errorf("all-format-verb message should return empty: %q", got)
	}
}

func TestSalvageMessage_MixedSentences(t *testing.T) {
	got := salvageMessage("Don't use '%s'. See the A-Z word list for details.")
	if got != "See the A-Z word list for details." {
		t.Errorf("mixed message should keep clean sentence: %q", got)
	}
}

func TestSalvageMessage_PositionalFormatVerb(t *testing.T) {
	got := salvageMessage("Avoid passive voice: '%[1]s %[2]s'.")
	if got != "" {
		t.Errorf("positional format verb should be detected: %q", got)
	}
}

func TestSalvageMessage_Empty(t *testing.T) {
	got := salvageMessage("")
	if got != "" {
		t.Errorf("empty message should return empty: %q", got)
	}
}

func TestSalvageMessage_WhitespaceOnly(t *testing.T) {
	got := salvageMessage("   ")
	if got != "" {
		t.Errorf("whitespace-only message should return empty: %q", got)
	}
}

// ── swapSampler ───────────────────────────────────────────────────────────────

func TestSwapSampler_TwoPairs(t *testing.T) {
	swap := map[string]string{
		"adaptor":    "adapter",
		"afterwards": "afterward",
	}
	got := swapSampler(swap)
	if got == "" {
		t.Fatal("swapSampler should return non-empty for 2-pair map")
	}
	// Should show examples with total count
	if want := "2 substitutions total"; !contains(got, want) {
		t.Errorf("swapSampler = %q, missing %q", got, want)
	}
	if !contains(got, "adapter") {
		t.Errorf("swapSampler = %q, missing example value", got)
	}
}

func TestSwapSampler_SinglePair(t *testing.T) {
	swap := map[string]string{"adaptor": "adapter"}
	got := swapSampler(swap)
	if !contains(got, "Suggests using") {
		t.Errorf("single-pair swap should use compact format: %q", got)
	}
	if contains(got, "total") {
		t.Errorf("single-pair swap should not show total: %q", got)
	}
}

func TestSwapSampler_AllRegex(t *testing.T) {
	swap := map[string]string{
		"(?:agent|virtual assistant)": "personal digital assistant",
		"(?:drive C:|drive C>)":       "drive C",
	}
	got := swapSampler(swap)
	if !contains(got, "replacements for 2 terms") {
		t.Errorf("all-regex swap should fall back to count: %q", got)
	}
}

func TestSwapSampler_FilterRegexShowPlain(t *testing.T) {
	swap := map[string]string{
		"adaptor":                     "adapter",
		"(?:agent|virtual assistant)": "personal digital assistant",
		"afterwards":                  "afterward",
	}
	got := swapSampler(swap)
	// Should show plain-key examples
	if !contains(got, "adapter") {
		t.Errorf("should show plain-key example: %q", got)
	}
	// Should not show regex key content
	if contains(got, "agent") {
		t.Errorf("should filter regex key: %q", got)
	}
	if !contains(got, "3 substitutions total") {
		t.Errorf("should show total including regex entries: %q", got)
	}
}

func TestSwapSampler_AlphabeticalOrder(t *testing.T) {
	swap := map[string]string{
		"zebra":    "z-replacement",
		"adaptor":  "adapter",
		"backbone": "b-replacement",
	}
	got := swapSampler(swap)
	// "adaptor" should come before "backbone" alphabetically
	idxA := indexOf(got, "adaptor")
	idxB := indexOf(got, "backbone")
	if idxA < 0 || idxB < 0 {
		t.Fatalf("both examples should appear: %q", got)
	}
	if idxA > idxB {
		t.Errorf("examples should be alphabetical (adaptor before backbone): %q", got)
	}
}

func TestSwapSampler_Empty(t *testing.T) {
	got := swapSampler(map[string]string{})
	if got != "" {
		t.Errorf("empty swap should return empty: %q", got)
	}
}

// ── categoriesFromRule ────────────────────────────────────────────────────────

func TestCategoriesFromRule_WithCategory(t *testing.T) {
	rule := &parser.ValeRule{
		Name:     "Avoid",
		Extends:  "existence",
		Level:    "error",
		Category: "Formatting",
	}
	got := categoriesFromRule(rule)
	if len(got) != 1 || got[0] != "Formatting" {
		t.Errorf("categoriesFromRule(single category) = %v, want [Formatting]", got)
	}
}

func TestCategoriesFromRule_EmptyFallsBackToExtends(t *testing.T) {
	rule := &parser.ValeRule{
		Name:     "Avoid",
		Extends:  "existence",
		Level:    "error",
		Category: "",
	}
	got := categoriesFromRule(rule)
	if len(got) != 1 || got[0] != "existence" {
		t.Errorf("categoriesFromRule(empty) = %v, want [existence]", got)
	}
}

func TestCategoriesFromRule_CommaSeparated(t *testing.T) {
	rule := &parser.ValeRule{
		Name:     "Terms",
		Extends:  "substitution",
		Level:    "warning",
		Category: "Formatting,Terminology",
	}
	got := categoriesFromRule(rule)
	if len(got) != 2 {
		t.Fatalf("categoriesFromRule(comma-separated) returned %d items, want 2: %v", len(got), got)
	}
	if got[0] != "Formatting" {
		t.Errorf("first category = %q, want %q", got[0], "Formatting")
	}
	if got[1] != "Terminology" {
		t.Errorf("second category = %q, want %q", got[1], "Terminology")
	}
}

func TestCategoriesFromRule_CommaSeparatedWithSpaces(t *testing.T) {
	rule := &parser.ValeRule{
		Name:     "Terms",
		Extends:  "substitution",
		Level:    "warning",
		Category: "Formatting , Terminology , Style",
	}
	got := categoriesFromRule(rule)
	if len(got) != 3 {
		t.Fatalf("categoriesFromRule(with spaces) returned %d items, want 3: %v", len(got), got)
	}
	if got[0] != "Formatting" {
		t.Errorf("first category = %q, want %q", got[0], "Formatting")
	}
	if got[1] != "Terminology" {
		t.Errorf("second category = %q, want %q", got[1], "Terminology")
	}
	if got[2] != "Style" {
		t.Errorf("third category = %q, want %q", got[2], "Style")
	}
}

func TestCategoriesFromRule_OnlyCommasFallsBackToExtends(t *testing.T) {
	rule := &parser.ValeRule{
		Name:     "Terms",
		Extends:  "substitution",
		Level:    "warning",
		Category: ", , ,",
	}
	got := categoriesFromRule(rule)
	if len(got) != 1 || got[0] != "substitution" {
		t.Errorf("categoriesFromRule(only commas) = %v, want [substitution]", got)
	}
}

// ── aggregateCounts ───────────────────────────────────────────────────────────

func TestAggregateCounts_MultipleRules(t *testing.T) {
	rules := []*parser.ValeRule{
		{Name: "Avoid", Extends: "existence", Level: "error", Category: "Formatting"},
		{Name: "Terms", Extends: "substitution", Level: "warning", Category: "Terminology"},
		{Name: "Headings", Extends: "existence", Level: "warning", Category: "Formatting"},
		{Name: "Acronyms", Extends: "conditional", Level: "suggestion", Category: "Style"},
	}

	byType, bySeverity, byCategory := aggregateCounts(rules)

	// byType
	if byType["existence"] != 2 {
		t.Errorf("byType[existence] = %d, want 2", byType["existence"])
	}
	if byType["substitution"] != 1 {
		t.Errorf("byType[substitution] = %d, want 1", byType["substitution"])
	}
	if byType["conditional"] != 1 {
		t.Errorf("byType[conditional] = %d, want 1", byType["conditional"])
	}

	// bySeverity
	if bySeverity["error"] != 1 {
		t.Errorf("bySeverity[error] = %d, want 1", bySeverity["error"])
	}
	if bySeverity["warning"] != 2 {
		t.Errorf("bySeverity[warning] = %d, want 2", bySeverity["warning"])
	}
	if bySeverity["suggestion"] != 1 {
		t.Errorf("bySeverity[suggestion] = %d, want 1", bySeverity["suggestion"])
	}

	// byCategory
	if byCategory["Formatting"] != 2 {
		t.Errorf("byCategory[Formatting] = %d, want 2", byCategory["Formatting"])
	}
	if byCategory["Terminology"] != 1 {
		t.Errorf("byCategory[Terminology] = %d, want 1", byCategory["Terminology"])
	}
	if byCategory["Style"] != 1 {
		t.Errorf("byCategory[Style] = %d, want 1", byCategory["Style"])
	}
}

func TestAggregateCounts_EmptySlice(t *testing.T) {
	byType, bySeverity, byCategory := aggregateCounts(nil)

	if byType == nil {
		t.Error("byType should not be nil for empty input")
	}
	if bySeverity == nil {
		t.Error("bySeverity should not be nil for empty input")
	}
	if byCategory == nil {
		t.Error("byCategory should not be nil for empty input")
	}
	if len(byType) != 0 {
		t.Errorf("byType should be empty, got %v", byType)
	}
	if len(bySeverity) != 0 {
		t.Errorf("bySeverity should be empty, got %v", bySeverity)
	}
	if len(byCategory) != 0 {
		t.Errorf("byCategory should be empty, got %v", byCategory)
	}
}

func TestAggregateCounts_MultipleCategories(t *testing.T) {
	rules := []*parser.ValeRule{
		{Name: "Terms", Extends: "substitution", Level: "warning", Category: "Formatting,Terminology"},
		{Name: "Avoid", Extends: "existence", Level: "error", Category: "Formatting"},
	}

	_, _, byCategory := aggregateCounts(rules)

	// "Terms" contributes to both Formatting and Terminology
	// "Avoid" contributes to Formatting
	if byCategory["Formatting"] != 2 {
		t.Errorf("byCategory[Formatting] = %d, want 2", byCategory["Formatting"])
	}
	if byCategory["Terminology"] != 1 {
		t.Errorf("byCategory[Terminology] = %d, want 1", byCategory["Terminology"])
	}
}

func TestAggregateCounts_NoCategoryFallsBackToExtends(t *testing.T) {
	rules := []*parser.ValeRule{
		{Name: "Avoid", Extends: "existence", Level: "error", Category: ""},
		{Name: "Terms", Extends: "substitution", Level: "warning", Category: ""},
	}

	_, _, byCategory := aggregateCounts(rules)

	if byCategory["existence"] != 1 {
		t.Errorf("byCategory[existence] = %d, want 1", byCategory["existence"])
	}
	if byCategory["substitution"] != 1 {
		t.Errorf("byCategory[substitution] = %d, want 1", byCategory["substitution"])
	}
}

// ── test helpers ──────────────────────────────────────────────────────────────

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
