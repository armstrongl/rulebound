package generator

import (
	"testing"

	"github.com/larah/rulebound/internal/config"
	"github.com/larah/rulebound/internal/parser"
)

func makeGuidelineInternal(name, title string, weight int) *parser.Guideline {
	return &parser.Guideline{
		Name:        name,
		Title:       title,
		Description: "A test guideline",
		Weight:      weight,
		Body:        "Guideline prose content.",
	}
}

// ── applyGuidelinesConfig ────────────────────────────────────────────────────

func TestApplyGuidelinesConfig_ExcludeRemovesGuidelines(t *testing.T) {
	guidelines := []*parser.Guideline{
		makeGuidelineInternal("alpha", "Alpha", 0),
		makeGuidelineInternal("beta", "Beta", 0),
		makeGuidelineInternal("gamma", "Gamma", 0),
	}
	cfg := config.GuidelinesConfig{
		Exclude: []string{"beta"},
	}

	result, warnings := applyGuidelinesConfig(guidelines, cfg)

	if len(result) != 2 {
		t.Fatalf("expected 2 guidelines after exclude, got %d", len(result))
	}
	for _, g := range result {
		if g.Name == "beta" {
			t.Error("beta should have been excluded")
		}
	}
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(warnings))
	}
}

func TestApplyGuidelinesConfig_OrderSetsWeight(t *testing.T) {
	guidelines := []*parser.Guideline{
		makeGuidelineInternal("gamma", "Gamma", 0),
		makeGuidelineInternal("alpha", "Alpha", 0),
		makeGuidelineInternal("beta", "Beta", 0),
	}
	cfg := config.GuidelinesConfig{
		Order: []string{"beta", "alpha"},
	}

	result, _ := applyGuidelinesConfig(guidelines, cfg)

	// beta should have weight -10000, alpha -9999, gamma keeps 0
	find := func(name string) *parser.Guideline {
		for _, g := range result {
			if g.Name == name {
				return g
			}
		}
		return nil
	}

	beta := find("beta")
	if beta == nil || beta.Weight != -10000 {
		t.Errorf("beta weight = %d, want -10000", beta.Weight)
	}
	alpha := find("alpha")
	if alpha == nil || alpha.Weight != -9999 {
		t.Errorf("alpha weight = %d, want -9999", alpha.Weight)
	}
	gamma := find("gamma")
	if gamma == nil || gamma.Weight != 0 {
		t.Errorf("gamma weight = %d, want 0 (unchanged)", gamma.Weight)
	}
}

func TestApplyGuidelinesConfig_ExcludeWinsOverOrder(t *testing.T) {
	guidelines := []*parser.Guideline{
		makeGuidelineInternal("alpha", "Alpha", 0),
		makeGuidelineInternal("beta", "Beta", 0),
	}
	cfg := config.GuidelinesConfig{
		Order:   []string{"alpha", "beta"},
		Exclude: []string{"alpha"},
	}

	result, _ := applyGuidelinesConfig(guidelines, cfg)

	if len(result) != 1 {
		t.Fatalf("expected 1 guideline, got %d", len(result))
	}
	if result[0].Name != "beta" {
		t.Errorf("expected beta, got %s", result[0].Name)
	}
}

func TestApplyGuidelinesConfig_UnmatchedOrderProducesWarning(t *testing.T) {
	guidelines := []*parser.Guideline{
		makeGuidelineInternal("alpha", "Alpha", 0),
	}
	cfg := config.GuidelinesConfig{
		Order: []string{"alpha", "nonexistent"},
	}

	_, warnings := applyGuidelinesConfig(guidelines, cfg)

	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning for unmatched order stem, got %d", len(warnings))
	}
}

func TestApplyGuidelinesConfig_DeterministicTieBreaking(t *testing.T) {
	// Two unlisted guidelines with same weight should sort alphabetically
	guidelines := []*parser.Guideline{
		makeGuidelineInternal("zebra", "Zebra", 0),
		makeGuidelineInternal("apple", "Apple", 0),
	}
	cfg := config.GuidelinesConfig{}

	result, _ := applyGuidelinesConfig(guidelines, cfg)

	if len(result) != 2 {
		t.Fatalf("expected 2 guidelines, got %d", len(result))
	}
	// Same weight, so alphabetical: apple before zebra
	if result[0].Name != "apple" || result[1].Name != "zebra" {
		t.Errorf("expected [apple, zebra], got [%s, %s]", result[0].Name, result[1].Name)
	}
}
