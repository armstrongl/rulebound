package generator

import (
	"testing"

	"github.com/larah/rulebound/internal/parser"
)

// ── linkDomain ────────────────────────────────────────────────────────────────

func TestLinkDomain_FullURL(t *testing.T) {
	got := linkDomain("https://example.com/path/to/page")
	if got != "example.com" {
		t.Errorf("linkDomain(full URL) = %q, want %q", got, "example.com")
	}
}

func TestLinkDomain_HTTPScheme(t *testing.T) {
	got := linkDomain("http://docs.microsoft.com/en-us/style-guide")
	if got != "docs.microsoft.com" {
		t.Errorf("linkDomain(http URL) = %q, want %q", got, "docs.microsoft.com")
	}
}

func TestLinkDomain_NoScheme(t *testing.T) {
	got := linkDomain("example.com/path")
	if got != "example.com" {
		t.Errorf("linkDomain(no scheme) = %q, want %q", got, "example.com")
	}
}

func TestLinkDomain_Empty(t *testing.T) {
	got := linkDomain("")
	if got != "" {
		t.Errorf("linkDomain(empty) = %q, want %q", got, "")
	}
}

func TestLinkDomain_WithPort(t *testing.T) {
	got := linkDomain("https://example.com:8080/path")
	// The implementation strips scheme and path, leaving host:port
	if got != "example.com:8080" {
		t.Errorf("linkDomain(with port) = %q, want %q", got, "example.com:8080")
	}
}

func TestLinkDomain_MalformedURL(t *testing.T) {
	// No path separator after host, so entire string after scheme strip is returned
	got := linkDomain("https://example.com")
	if got != "example.com" {
		t.Errorf("linkDomain(no path) = %q, want %q", got, "example.com")
	}
}

func TestLinkDomain_SchemeOnly(t *testing.T) {
	// Edge case: just a scheme with nothing after it
	got := linkDomain("https://")
	if got != "" {
		t.Errorf("linkDomain(scheme only) = %q, want %q", got, "")
	}
}

func TestLinkDomain_NoPath(t *testing.T) {
	got := linkDomain("example.com")
	if got != "example.com" {
		t.Errorf("linkDomain(bare domain) = %q, want %q", got, "example.com")
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
