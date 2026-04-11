package parser_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/armstrongl/rulebound/internal/parser"
)

// validateTestdata returns the absolute path to the validate testdata directory.
func validateTestdata(parts ...string) string {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(file), "testdata", "validate")
	return filepath.Join(append([]string{dir}, parts...)...)
}

func TestValidateRule(t *testing.T) {
	tests := []struct {
		name       string
		file       string
		wantCount  int    // expected number of validation errors
		wantField  string // if non-empty, at least one error must reference this field
		wantSubstr string // if non-empty, at least one error message must contain this
	}{
		{
			name:      "valid substitution",
			file:      "valid-substitution.yml",
			wantCount: 0,
		},
		{
			name:      "valid existence",
			file:      "valid-existence.yml",
			wantCount: 0,
		},
		{
			name:      "valid occurrence",
			file:      "valid-occurrence.yml",
			wantCount: 0,
		},
		{
			name:      "valid capitalization",
			file:      "valid-capitalization.yml",
			wantCount: 0,
		},
		{
			name:       "missing message",
			file:       "missing-message.yml",
			wantCount:  1,
			wantField:  "message",
			wantSubstr: "message",
		},
		{
			name:       "invalid extends (script)",
			file:       "invalid-extends.yml",
			wantCount:  1,
			wantField:  "extends",
			wantSubstr: "supported types",
		},
		{
			name:       "substitution missing swap",
			file:       "missing-swap.yml",
			wantCount:  1,
			wantField:  "swap",
			wantSubstr: "swap",
		},
		{
			name:       "invalid level",
			file:       "invalid-level.yml",
			wantCount:  1,
			wantField:  "level",
			wantSubstr: "critical",
		},
		{
			name:      "occurrence with min zero is valid",
			file:      "occurrence-min-zero.yml",
			wantCount: 0,
		},
		{
			name:      "occurrence with only max is valid",
			file:      "occurrence-max-only.yml",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs, err := parser.ValidateRule(validateTestdata(tt.file))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(errs) != tt.wantCount {
				t.Errorf("got %d validation errors, want %d: %+v", len(errs), tt.wantCount, errs)
			}

			if tt.wantField != "" {
				found := false
				for _, ve := range errs {
					if ve.Field == tt.wantField {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error referencing field %q, got: %+v", tt.wantField, errs)
				}
			}

			if tt.wantSubstr != "" {
				found := false
				for _, ve := range errs {
					if strings.Contains(ve.Message, tt.wantSubstr) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error message containing %q, got: %+v", tt.wantSubstr, errs)
				}
			}
		})
	}
}

func TestValidateRule_MultipleErrors(t *testing.T) {
	errs, err := parser.ValidateRule(validateTestdata("multiple-errors.yml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// multiple-errors.yml is missing both message and swap
	if len(errs) < 2 {
		t.Errorf("expected at least 2 validation errors, got %d: %+v", len(errs), errs)
	}

	fields := make(map[string]bool)
	for _, ve := range errs {
		fields[ve.Field] = true
	}
	if !fields["message"] {
		t.Errorf("expected error for field 'message', got fields: %v", fields)
	}
	if !fields["swap"] {
		t.Errorf("expected error for field 'swap', got fields: %v", fields)
	}
}

func TestValidateRule_MissingExtends(t *testing.T) {
	// Create a temp file with no extends field
	dir := t.TempDir()
	path := filepath.Join(dir, "no-extends.yml")
	if err := writeTestFile(t, path, "message: test\nlevel: warning\n"); err != nil {
		t.Fatalf("setup: %v", err)
	}

	errs, err := parser.ValidateRule(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %+v", len(errs), errs)
	}
	if errs[0].Field != "extends" {
		t.Errorf("expected error for 'extends', got %q", errs[0].Field)
	}
}

func TestValidateRule_ExistenceEmptyTokens(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty-tokens.yml")
	content := "extends: existence\nmessage: test\nlevel: warning\ntokens: []\n"
	if err := writeTestFile(t, path, content); err != nil {
		t.Fatalf("setup: %v", err)
	}

	errs, err := parser.ValidateRule(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, ve := range errs {
		if ve.Field == "tokens" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error for empty tokens, got: %+v", errs)
	}
}

func TestValidateRule_OccurrenceMissingBothMaxMin(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "no-max-min.yml")
	content := "extends: occurrence\nmessage: test\nlevel: warning\ntoken: '\\S+'\n"
	if err := writeTestFile(t, path, content); err != nil {
		t.Fatalf("setup: %v", err)
	}

	errs, err := parser.ValidateRule(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, ve := range errs {
		if ve.Field == "max/min" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error for missing max/min, got: %+v", errs)
	}
}

func TestValidateRule_CapitalizationMissingMatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "no-match.yml")
	content := "extends: capitalization\nmessage: test\nlevel: warning\n"
	if err := writeTestFile(t, path, content); err != nil {
		t.Fatalf("setup: %v", err)
	}

	errs, err := parser.ValidateRule(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, ve := range errs {
		if ve.Field == "match" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error for missing match, got: %+v", errs)
	}
}

func TestValidateRule_ExtraFieldsNoError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "extras.yml")
	content := "extends: existence\nmessage: test\nlevel: warning\ntokens:\n  - foo\ncustom_field: bar\n"
	if err := writeTestFile(t, path, content); err != nil {
		t.Fatalf("setup: %v", err)
	}

	errs, err := parser.ValidateRule(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(errs) != 0 {
		t.Errorf("extra unknown fields should not cause errors, got: %+v", errs)
	}
}

func TestValidateRule_ValidLevelError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "level-error.yml")
	content := "extends: existence\nmessage: test\nlevel: error\ntokens:\n  - foo\n"
	if err := writeTestFile(t, path, content); err != nil {
		t.Fatalf("setup: %v", err)
	}

	errs, err := parser.ValidateRule(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(errs) != 0 {
		t.Errorf("level 'error' should be valid, got: %+v", errs)
	}
}

// ── Validate real Microsoft rules ───────────────────────────────────────────

func TestValidateRule_RealRules(t *testing.T) {
	tests := []struct {
		name string
		file string
	}{
		{"Avoid (existence)", "Avoid.yml"},
		{"Headings (capitalization)", "Headings.yml"},
		{"SentenceLength (occurrence)", "SentenceLength.yml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := testdata(tt.file)
			errs, err := parser.ValidateRule(path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(errs) != 0 {
				t.Errorf("expected 0 validation errors for %s, got %d: %+v", tt.file, len(errs), errs)
			}
		})
	}
}

func TestValidateRuleBytes_SwapNonStringValues(t *testing.T) {
	content := "extends: substitution\nmessage: test\nlevel: warning\nswap:\n  foo: 123\n"
	errs, err := parser.ValidateRuleBytes([]byte(content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, ve := range errs {
		if ve.Field == "swap" && strings.Contains(ve.Message, "string") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error for non-string swap value, got: %+v", errs)
	}
}

func TestValidateRuleBytes_SwapSequenceNonMapping(t *testing.T) {
	content := "extends: substitution\nmessage: test\nlevel: warning\nswap:\n  - notamap\n"
	errs, err := parser.ValidateRuleBytes([]byte(content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, ve := range errs {
		if ve.Field == "swap" && strings.Contains(ve.Message, "mapping") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error for non-mapping swap item, got: %+v", errs)
	}
}

func TestValidateRuleBytes_OccurrenceNonIntMax(t *testing.T) {
	content := "extends: occurrence\nmessage: test\nlevel: warning\nmax: foo\ntoken: '\\S+'\n"
	errs, err := parser.ValidateRuleBytes([]byte(content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, ve := range errs {
		if ve.Field == "max" && strings.Contains(ve.Message, "integer") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error for non-integer max, got: %+v", errs)
	}
}

func TestValidateRuleBytes_OccurrenceNonStringToken(t *testing.T) {
	content := "extends: occurrence\nmessage: test\nlevel: warning\nmax: 10\ntoken: 123\n"
	errs, err := parser.ValidateRuleBytes([]byte(content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, ve := range errs {
		if ve.Field == "token" && strings.Contains(ve.Message, "non-empty string") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error for non-string token, got: %+v", errs)
	}
}

func TestExtractFrontmatter_EmptyFrontmatter(t *testing.T) {
	data := []byte("---\n---\nBody here.\n")
	fm, body, err := parser.ExtractFrontmatter(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fm) != 0 {
		t.Errorf("expected empty frontmatter, got %q", fm)
	}
	if string(body) != "Body here.\n" {
		t.Errorf("body: got %q", body)
	}
}

// writeTestFile is a helper that writes content to a file, returning any error.
func writeTestFile(t *testing.T, path, content string) error {
	t.Helper()
	return os.WriteFile(path, []byte(content), 0o644)
}
