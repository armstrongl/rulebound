package mdgen_test

import (
	"os"
	"strings"
	"testing"

	"github.com/armstrongl/rulebound/internal/mdgen"
	"github.com/armstrongl/rulebound/internal/parser"
	"go.yaml.in/yaml/v3"
)

// ── Semantic tests ─────────────────────────────────────────────────────────

func TestEmitYAML_Substitution_Semantic(t *testing.T) {
	src := &mdgen.RuleSource{
		Extends: parser.ExtendsSubstitution,
		Message: "Prefer '%s' over '%s'.",
		Level:   "warning",
		Fields:  map[string]interface{}{"ignorecase": true},
		Swap: []mdgen.SwapPair{
			{Key: "leverage", Value: "use"},
			{Key: "utilize", Value: "use"},
		},
	}

	out, _, err := mdgen.EmitYAML(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]interface{}
	if err := yaml.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}

	if m["extends"] != "substitution" {
		t.Errorf("extends: got %v", m["extends"])
	}
	if m["message"] != "Prefer '%s' over '%s'." {
		t.Errorf("message: got %v", m["message"])
	}
	if m["level"] != "warning" {
		t.Errorf("level: got %v", m["level"])
	}
	if m["ignorecase"] != true {
		t.Errorf("ignorecase: got %v", m["ignorecase"])
	}

	// Verify swap is present and has correct values.
	swapRaw, ok := m["swap"].(map[string]interface{})
	if !ok {
		t.Fatalf("swap: expected map, got %T", m["swap"])
	}
	if swapRaw["leverage"] != "use" {
		t.Errorf("swap[leverage]: got %v", swapRaw["leverage"])
	}
}

func TestEmitYAML_Existence_Semantic(t *testing.T) {
	src := &mdgen.RuleSource{
		Extends: parser.ExtendsExistence,
		Message: "test",
		Level:   "warning",
		Fields:  map[string]interface{}{},
		Tokens:  []string{"foo", "bar"},
	}

	out, _, err := mdgen.EmitYAML(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]interface{}
	if err := yaml.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	tokens, ok := m["tokens"].([]interface{})
	if !ok {
		t.Fatalf("tokens: expected slice, got %T", m["tokens"])
	}
	if len(tokens) != 2 || tokens[0] != "foo" || tokens[1] != "bar" {
		t.Errorf("tokens: got %v", tokens)
	}
}

func TestEmitYAML_Occurrence_Semantic(t *testing.T) {
	src := &mdgen.RuleSource{
		Extends: parser.ExtendsOccurrence,
		Message: "test",
		Level:   "suggestion",
		Fields: map[string]interface{}{
			"max":   30,
			"token": `[^\s]+`,
			"scope": "sentence",
		},
	}

	out, _, err := mdgen.EmitYAML(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]interface{}
	if err := yaml.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if m["max"] != 30 {
		t.Errorf("max: got %v", m["max"])
	}
	if m["token"] != `[^\s]+` {
		t.Errorf("token: got %v", m["token"])
	}
	if m["scope"] != "sentence" {
		t.Errorf("scope: got %v", m["scope"])
	}
}

func TestEmitYAML_Capitalization_Semantic(t *testing.T) {
	src := &mdgen.RuleSource{
		Extends:    parser.ExtendsCapitalization,
		Message:    "test",
		Level:      "warning",
		Fields:     map[string]interface{}{"match": "$sentence"},
		Exceptions: []string{"iOS", "API"},
	}

	out, _, err := mdgen.EmitYAML(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]interface{}
	if err := yaml.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if m["match"] != "$sentence" {
		t.Errorf("match: got %v", m["match"])
	}

	exceptions, ok := m["exceptions"].([]interface{})
	if !ok {
		t.Fatalf("exceptions: expected slice, got %T", m["exceptions"])
	}
	if len(exceptions) != 2 || exceptions[0] != "iOS" || exceptions[1] != "API" {
		t.Errorf("exceptions: got %v", exceptions)
	}
}

func TestEmitYAML_UnknownField_Passthrough(t *testing.T) {
	src := &mdgen.RuleSource{
		Extends: parser.ExtendsExistence,
		Message: "test",
		Level:   "warning",
		Fields:  map[string]interface{}{"reviewed_by": "Jane Smith"},
		Tokens:  []string{"foo"},
	}

	out, _, err := mdgen.EmitYAML(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]interface{}
	if err := yaml.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if m["reviewed_by"] != "Jane Smith" {
		t.Errorf("reviewed_by: got %v", m["reviewed_by"])
	}
}

func TestEmitYAML_DefaultLevel_Emitted(t *testing.T) {
	src := &mdgen.RuleSource{
		Extends: parser.ExtendsExistence,
		Message: "test",
		Level:   "warning",
		Fields:  map[string]interface{}{},
		Tokens:  []string{"foo"},
	}

	out, _, err := mdgen.EmitYAML(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(string(out), "level: warning") {
		t.Error("default level 'warning' should still be emitted in output")
	}
}

func TestEmitYAML_EmptyExceptions_Omitted(t *testing.T) {
	src := &mdgen.RuleSource{
		Extends:    parser.ExtendsCapitalization,
		Message:    "test",
		Level:      "warning",
		Fields:     map[string]interface{}{"match": "$sentence"},
		Exceptions: nil,
	}

	out, _, err := mdgen.EmitYAML(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(string(out), "exceptions") {
		t.Error("empty exceptions should be omitted from output")
	}
}

func TestEmitYAML_ScopeField(t *testing.T) {
	src := &mdgen.RuleSource{
		Extends: parser.ExtendsExistence,
		Message: "test",
		Level:   "warning",
		Fields:  map[string]interface{}{"scope": "heading"},
		Tokens:  []string{"foo"},
	}

	out, _, err := mdgen.EmitYAML(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(string(out), "scope: heading") {
		t.Error("scope field should appear in output")
	}
}

func TestEmitYAML_NoMetaInOutput(t *testing.T) {
	src := &mdgen.RuleSource{
		Extends: parser.ExtendsExistence,
		Message: "test",
		Level:   "warning",
		Fields:  map[string]interface{}{},
		Tokens:  []string{"foo"},
		Meta:    map[string]interface{}{"author": "Jane"},
	}

	out, _, err := mdgen.EmitYAML(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(string(out), "meta") || strings.Contains(string(out), "author") {
		t.Error("meta fields should not appear in generated YAML")
	}
}

// ── Golden-file tests ──────────────────────────────────────────────────────

func TestEmitYAML_GoldenFile_Substitution(t *testing.T) {
	goldenTest(t, "substitution.md", "expected/substitution.yml")
}

func TestEmitYAML_GoldenFile_Existence(t *testing.T) {
	goldenTest(t, "existence.md", "expected/existence.yml")
}

func TestEmitYAML_GoldenFile_Occurrence(t *testing.T) {
	goldenTest(t, "occurrence.md", "expected/occurrence.yml")
}

func TestEmitYAML_GoldenFile_Capitalization(t *testing.T) {
	goldenTest(t, "capitalization.md", "expected/capitalization.yml")
}

// ── Round-trip integration ─────────────────────────────────────────────────

func TestEmitYAML_RoundTrip_Substitution(t *testing.T) {
	data := readTestdata(t, "substitution.md")
	src, _, err := mdgen.ParseMarkdown(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	out, _, err := mdgen.EmitYAML(src)
	if err != nil {
		t.Fatalf("emit: %v", err)
	}

	// Unmarshal and verify key fields.
	var m map[string]interface{}
	if err := yaml.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if m["extends"] != "substitution" {
		t.Errorf("extends: got %v", m["extends"])
	}
	swapRaw, ok := m["swap"].(map[string]interface{})
	if !ok {
		t.Fatalf("swap: expected map, got %T", m["swap"])
	}
	if len(swapRaw) != 5 {
		t.Errorf("swap entries: got %d, want 5", len(swapRaw))
	}
}

// ── Swap order verification ────────────────────────────────────────────────

func TestEmitYAML_SwapOrder(t *testing.T) {
	src := &mdgen.RuleSource{
		Extends: parser.ExtendsSubstitution,
		Message: "test",
		Level:   "warning",
		Fields:  map[string]interface{}{},
		Swap: []mdgen.SwapPair{
			{Key: "zebra", Value: "z"},
			{Key: "apple", Value: "a"},
			{Key: "mango", Value: "m"},
		},
	}

	out, _, err := mdgen.EmitYAML(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify insertion order preserved (zebra before apple before mango).
	outStr := string(out)
	zebraIdx := strings.Index(outStr, "zebra")
	appleIdx := strings.Index(outStr, "apple")
	mangoIdx := strings.Index(outStr, "mango")

	if zebraIdx == -1 || appleIdx == -1 || mangoIdx == -1 {
		t.Fatalf("missing swap keys in output:\n%s", outStr)
	}
	if zebraIdx >= appleIdx || appleIdx >= mangoIdx {
		t.Errorf("swap order not preserved: zebra@%d, apple@%d, mango@%d", zebraIdx, appleIdx, mangoIdx)
	}
}

// ── Helpers ────────────────────────────────────────────────────────────────

func goldenTest(t *testing.T, inputFile, goldenFile string) {
	t.Helper()

	data := readTestdata(t, inputFile)
	src, _, err := mdgen.ParseMarkdown(data)
	if err != nil {
		t.Fatalf("parse %s: %v", inputFile, err)
	}

	out, _, err := mdgen.EmitYAML(src)
	if err != nil {
		t.Fatalf("emit: %v", err)
	}

	expected, err := os.ReadFile(testdata(goldenFile))
	if err != nil {
		t.Fatalf("reading golden file %s: %v", goldenFile, err)
	}

	if string(out) != string(expected) {
		t.Errorf("output differs from golden file %s:\n--- got ---\n%s\n--- want ---\n%s", goldenFile, out, expected)
	}
}
