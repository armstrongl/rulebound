package mdgen_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/armstrongl/rulebound/internal/mdgen"
	"github.com/armstrongl/rulebound/internal/parser"
)

func testdata(parts ...string) string {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(file), "testdata")
	return filepath.Join(append([]string{dir}, parts...)...)
}

func readTestdata(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(testdata(name))
	if err != nil {
		t.Fatalf("reading testdata %s: %v", name, err)
	}
	return data
}

// ── Happy paths ────────────────────────────────────────────────────────────

func TestParseMarkdown_Substitution(t *testing.T) {
	src, warnings, err := mdgen.ParseMarkdown(readTestdata(t, "substitution.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if src.Extends != parser.ExtendsSubstitution {
		t.Errorf("Extends: got %q, want %q", src.Extends, parser.ExtendsSubstitution)
	}
	if src.Message != "Prefer '%s' over '%s'." {
		t.Errorf("Message: got %q", src.Message)
	}
	if src.Level != "warning" {
		t.Errorf("Level: got %q, want %q", src.Level, "warning")
	}

	// Check ignorecase passed through.
	if v, ok := src.Fields["ignorecase"]; !ok || v != true {
		t.Errorf("Fields[ignorecase]: got %v", src.Fields["ignorecase"])
	}

	// Verify swap pairs in file order.
	expectedSwap := []mdgen.SwapPair{
		{Key: "leverage", Value: "use"},
		{Key: "utilize", Value: "use"},
		{Key: "functionality", Value: "feature"},
		{Key: "in order to", Value: "to"},
		{Key: "at this point in time", Value: "now"},
	}
	if len(src.Swap) != len(expectedSwap) {
		t.Fatalf("Swap: got %d pairs, want %d", len(src.Swap), len(expectedSwap))
	}
	for i, want := range expectedSwap {
		got := src.Swap[i]
		if got.Key != want.Key || got.Value != want.Value {
			t.Errorf("Swap[%d]: got %q:%q, want %q:%q", i, got.Key, got.Value, want.Key, want.Value)
		}
	}

	// No unexpected warnings.
	for _, w := range warnings {
		t.Logf("warning: %s", w.Message)
	}
}

func TestParseMarkdown_Existence(t *testing.T) {
	src, _, err := mdgen.ParseMarkdown(readTestdata(t, "existence.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if src.Extends != parser.ExtendsExistence {
		t.Errorf("Extends: got %q", src.Extends)
	}

	expectedTokens := []string{
		"was created", "was deleted", "was found",
		"was made", "is expected", "are required",
	}
	if len(src.Tokens) != len(expectedTokens) {
		t.Fatalf("Tokens: got %d, want %d", len(src.Tokens), len(expectedTokens))
	}
	for i, want := range expectedTokens {
		if src.Tokens[i] != want {
			t.Errorf("Tokens[%d]: got %q, want %q", i, src.Tokens[i], want)
		}
	}
}

func TestParseMarkdown_Occurrence(t *testing.T) {
	src, _, err := mdgen.ParseMarkdown(readTestdata(t, "occurrence.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if src.Extends != parser.ExtendsOccurrence {
		t.Errorf("Extends: got %q", src.Extends)
	}

	if v, ok := src.Fields["max"]; !ok || v != 30 {
		t.Errorf("Fields[max]: got %v", src.Fields["max"])
	}
	if v, ok := src.Fields["token"]; !ok || v != "[^\\s]+" {
		t.Errorf("Fields[token]: got %v", src.Fields["token"])
	}
	if v, ok := src.Fields["scope"]; !ok || v != "sentence" {
		t.Errorf("Fields[scope]: got %v", src.Fields["scope"])
	}
}

func TestParseMarkdown_Capitalization(t *testing.T) {
	src, _, err := mdgen.ParseMarkdown(readTestdata(t, "capitalization.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if src.Extends != parser.ExtendsCapitalization {
		t.Errorf("Extends: got %q", src.Extends)
	}
	if v, ok := src.Fields["match"]; !ok || v != "$sentence" {
		t.Errorf("Fields[match]: got %v", src.Fields["match"])
	}

	expectedExceptions := []string{"iOS", "macOS", "API", "UI", "GraphQL"}
	if len(src.Exceptions) != len(expectedExceptions) {
		t.Fatalf("Exceptions: got %d, want %d", len(src.Exceptions), len(expectedExceptions))
	}
	for i, want := range expectedExceptions {
		if src.Exceptions[i] != want {
			t.Errorf("Exceptions[%d]: got %q, want %q", i, src.Exceptions[i], want)
		}
	}
}

func TestParseMarkdown_MetaStripped(t *testing.T) {
	src, _, err := mdgen.ParseMarkdown(readTestdata(t, "meta-block.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if src.Meta == nil {
		t.Fatal("Meta: expected non-nil")
	}
	if src.Meta["author"] != "Jane Smith" {
		t.Errorf("Meta[author]: got %v", src.Meta["author"])
	}

	// Meta should not appear in Fields.
	if _, ok := src.Fields["meta"]; ok {
		t.Error("Fields should not contain 'meta' key")
	}
}

func TestParseMarkdown_UnknownFieldWarning(t *testing.T) {
	src, warnings, err := mdgen.ParseMarkdown(readTestdata(t, "unknown-field.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Unknown field should be preserved in Fields.
	if v, ok := src.Fields["reviewed_by"]; !ok || v != "Jane Smith" {
		t.Errorf("Fields[reviewed_by]: got %v", src.Fields["reviewed_by"])
	}

	// Should have a warning about the unknown field.
	found := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "reviewed_by") && strings.Contains(w.Message, "unknown") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning about unknown field 'reviewed_by'")
	}
}

func TestParseMarkdown_DefaultLevel(t *testing.T) {
	src, warnings, err := mdgen.ParseMarkdown(readTestdata(t, "no-level.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if src.Level != "warning" {
		t.Errorf("Level: got %q, want %q", src.Level, "warning")
	}

	found := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "level") && strings.Contains(w.Message, "warning") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected advisory warning about default level")
	}
}

// ── CRLF ───────────────────────────────────────────────────────────────────

func TestParseMarkdown_CRLF(t *testing.T) {
	src, _, err := mdgen.ParseMarkdown(readTestdata(t, "crlf.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if src.Extends != parser.ExtendsExistence {
		t.Errorf("Extends: got %q", src.Extends)
	}
	if len(src.Tokens) != 2 {
		t.Errorf("Tokens: got %d, want 2", len(src.Tokens))
	}
}

// ── Edge cases ─────────────────────────────────────────────────────────────

func TestParseMarkdown_EmptySwapBlock(t *testing.T) {
	src, _, err := mdgen.ParseMarkdown(readTestdata(t, "empty-swap.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(src.Swap) != 0 {
		t.Errorf("Swap: got %d pairs, want 0", len(src.Swap))
	}
}

func TestParseMarkdown_QuotedSwapKeys(t *testing.T) {
	src, _, err := mdgen.ParseMarkdown(readTestdata(t, "quoted-swap-keys.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(src.Swap) != 2 {
		t.Fatalf("Swap: got %d pairs, want 2", len(src.Swap))
	}
	if src.Swap[0].Key != "e.g." || src.Swap[0].Value != "for example" {
		t.Errorf("Swap[0]: got %q:%q", src.Swap[0].Key, src.Swap[0].Value)
	}
	if src.Swap[1].Key != "i.e." || src.Swap[1].Value != "that is" {
		t.Errorf("Swap[1]: got %q:%q", src.Swap[1].Key, src.Swap[1].Value)
	}
}

func TestParseMarkdown_DuplicateSwapBlocks(t *testing.T) {
	_, warnings, err := mdgen.ParseMarkdown(readTestdata(t, "duplicate-swap-blocks.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "duplicate") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning about duplicate vale-swap block")
	}
}

// ── Error paths ────────────────────────────────────────────────────────────

func TestParseMarkdown_MissingExtends(t *testing.T) {
	_, _, err := mdgen.ParseMarkdown(readTestdata(t, "missing-extends.md"))
	if err == nil {
		t.Fatal("expected error for missing extends")
	}
	if !strings.Contains(err.Error(), "extends") {
		t.Errorf("error should name 'extends' field: %v", err)
	}
}

func TestParseMarkdown_MissingMessage(t *testing.T) {
	_, _, err := mdgen.ParseMarkdown(readTestdata(t, "missing-message.md"))
	if err == nil {
		t.Fatal("expected error for missing message")
	}
	if !strings.Contains(err.Error(), "message") {
		t.Errorf("error should name 'message' field: %v", err)
	}
}

func TestParseMarkdown_EmptyMessage(t *testing.T) {
	_, _, err := mdgen.ParseMarkdown(readTestdata(t, "empty-message.md"))
	if err == nil {
		t.Fatal("expected error for empty message")
	}
	if !strings.Contains(err.Error(), "message") {
		t.Errorf("error should name 'message' field: %v", err)
	}
}

func TestParseMarkdown_UnsupportedExtends(t *testing.T) {
	_, _, err := mdgen.ParseMarkdown(readTestdata(t, "unsupported-extends.md"))
	if err == nil {
		t.Fatal("expected error for unsupported extends")
	}
	if !strings.Contains(err.Error(), "script") {
		t.Errorf("error should name the unsupported type: %v", err)
	}
	if !strings.Contains(err.Error(), "substitution") {
		t.Errorf("error should list supported types: %v", err)
	}
}

func TestParseMarkdown_MissingSwapBlock(t *testing.T) {
	_, _, err := mdgen.ParseMarkdown(readTestdata(t, "missing-swap.md"))
	if err == nil {
		t.Fatal("expected error for missing swap block")
	}
	if !strings.Contains(err.Error(), "vale-swap") {
		t.Errorf("error should name expected block: %v", err)
	}
}

func TestParseMarkdown_MissingTokensBlock(t *testing.T) {
	_, _, err := mdgen.ParseMarkdown(readTestdata(t, "missing-tokens.md"))
	if err == nil {
		t.Fatal("expected error for missing tokens block")
	}
	if !strings.Contains(err.Error(), "vale-tokens") {
		t.Errorf("error should name expected block: %v", err)
	}
}

func TestParseMarkdown_OccurrenceMissingToken(t *testing.T) {
	_, _, err := mdgen.ParseMarkdown(readTestdata(t, "occurrence-missing-token.md"))
	if err == nil {
		t.Fatal("expected error for missing token field")
	}
	if !strings.Contains(err.Error(), "token") {
		t.Errorf("error should name 'token' field: %v", err)
	}
}

func TestParseMarkdown_CapitalizationMissingMatch(t *testing.T) {
	_, _, err := mdgen.ParseMarkdown(readTestdata(t, "capitalization-missing-match.md"))
	if err == nil {
		t.Fatal("expected error for missing match field")
	}
	if !strings.Contains(err.Error(), "match") {
		t.Errorf("error should name 'match' field: %v", err)
	}
}

func TestParseMarkdown_NoFrontmatter(t *testing.T) {
	_, _, err := mdgen.ParseMarkdown(readTestdata(t, "no-frontmatter.md"))
	if err == nil {
		t.Fatal("expected error for no frontmatter")
	}
}
