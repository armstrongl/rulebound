package parser

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// ── parseFrontmatter ─────────────────────────────────────────────────────────

func TestParseFrontmatter_ValidFrontmatter(t *testing.T) {
	input := "---\ntitle: Voice and Tone\ndescription: How to write\nweight: 10\n---\nBody here."
	fm, body, err := parseFrontmatter(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.Title != "Voice and Tone" {
		t.Errorf("Title = %q, want %q", fm.Title, "Voice and Tone")
	}
	if fm.Description != "How to write" {
		t.Errorf("Description = %q, want %q", fm.Description, "How to write")
	}
	if fm.Weight != 10 {
		t.Errorf("Weight = %d, want 10", fm.Weight)
	}
	if body != "Body here." {
		t.Errorf("Body = %q, want %q", body, "Body here.")
	}
}

func TestParseFrontmatter_TitleOnly(t *testing.T) {
	input := "---\ntitle: Minimal\n---\nSome content."
	fm, body, err := parseFrontmatter(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.Title != "Minimal" {
		t.Errorf("Title = %q, want %q", fm.Title, "Minimal")
	}
	if fm.Weight != 0 {
		t.Errorf("Weight = %d, want 0 (default)", fm.Weight)
	}
	if body != "Some content." {
		t.Errorf("Body = %q, want %q", body, "Some content.")
	}
}

func TestParseFrontmatter_NoFrontmatter(t *testing.T) {
	input := "Just plain text, no fences."
	_, _, err := parseFrontmatter(input)
	if err == nil {
		t.Error("expected error for content without frontmatter")
	}
}

func TestParseFrontmatter_InvalidYAML(t *testing.T) {
	input := "---\ntitle: [unclosed\n---\nBody."
	_, _, err := parseFrontmatter(input)
	if err == nil {
		t.Error("expected error for invalid YAML frontmatter")
	}
}

func TestParseFrontmatter_EmptyBody(t *testing.T) {
	input := "---\ntitle: Frontmatter Only\n---\n"
	fm, body, err := parseFrontmatter(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.Title != "Frontmatter Only" {
		t.Errorf("Title = %q, want %q", fm.Title, "Frontmatter Only")
	}
	if body != "" {
		t.Errorf("Body = %q, want empty", body)
	}
}

func TestParseFrontmatter_WindowsLineEndings(t *testing.T) {
	input := "---\r\ntitle: CRLF Test\r\n---\r\nWindows body."
	fm, body, err := parseFrontmatter(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.Title != "CRLF Test" {
		t.Errorf("Title = %q, want %q", fm.Title, "CRLF Test")
	}
	if body != "Windows body." {
		t.Errorf("Body = %q, want %q", body, "Windows body.")
	}
}

// ── parseGuidelines ─────────────────────────────────────────────────────────

func guidelinesTestdata() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata", "Microsoft")
}

func TestParseGuidelines_ValidFiles(t *testing.T) {
	guidelines, warnings, err := parseGuidelines(guidelinesTestdata())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 3 valid guidelines: voice-and-tone, inclusive-language, frontmatter-only
	if len(guidelines) != 3 {
		t.Fatalf("expected 3 guidelines, got %d", len(guidelines))
	}

	// Should have 2 warnings: no-title.md and bad-yaml.md
	if len(warnings) < 2 {
		t.Errorf("expected at least 2 warnings, got %d", len(warnings))
	}
}

func TestParseGuidelines_StemNames(t *testing.T) {
	guidelines, _, err := parseGuidelines(guidelinesTestdata())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	find := func(name string) *Guideline {
		for _, g := range guidelines {
			if g.Name == name {
				return g
			}
		}
		return nil
	}

	vt := find("voice-and-tone")
	if vt == nil {
		t.Fatal("expected guideline with stem name 'voice-and-tone'")
	}
	if vt.Title != "Voice and Tone" {
		t.Errorf("Title = %q, want %q", vt.Title, "Voice and Tone")
	}
	if vt.Weight != 10 {
		t.Errorf("Weight = %d, want 10", vt.Weight)
	}
	if vt.Body == "" {
		t.Error("Body should not be empty")
	}

	il := find("inclusive-language")
	if il == nil {
		t.Fatal("expected guideline with stem name 'inclusive-language'")
	}
	if il.Weight != 20 {
		t.Errorf("Weight = %d, want 20", il.Weight)
	}
}

func TestParseGuidelines_FrontmatterOnlyIsValid(t *testing.T) {
	guidelines, _, err := parseGuidelines(guidelinesTestdata())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	find := func(name string) *Guideline {
		for _, g := range guidelines {
			if g.Name == name {
				return g
			}
		}
		return nil
	}

	fo := find("frontmatter-only")
	if fo == nil {
		t.Fatal("expected frontmatter-only guideline to be valid")
	}
	if fo.Body != "" {
		t.Errorf("Body = %q, want empty", fo.Body)
	}
}

func TestParseGuidelines_SkipsNonMDFiles(t *testing.T) {
	guidelines, warnings, err := parseGuidelines(guidelinesTestdata())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, g := range guidelines {
		if g.Name == "ignore-me" {
			t.Error("non-.md file should not be parsed as a guideline")
		}
	}
	for _, w := range warnings {
		if w.File == "ignore-me.txt" {
			t.Error("non-.md file should not produce a warning")
		}
	}
}

func TestParseGuidelines_NoGuidelinesDir(t *testing.T) {
	guidelines, warnings, err := parseGuidelines(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(guidelines) != 0 {
		t.Errorf("expected 0 guidelines, got %d", len(guidelines))
	}
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(warnings))
	}
}

func TestParseGuidelines_IgnoresSubdirectories(t *testing.T) {
	guidelines, _, err := parseGuidelines(guidelinesTestdata())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, g := range guidelines {
		if g.Name == "nested" {
			t.Error("guidelines in subdirectories should be ignored")
		}
	}
}

func TestParseGuidelines_EmptyGuidelinesDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "guidelines"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	guidelines, warnings, err := parseGuidelines(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(guidelines) != 0 {
		t.Errorf("expected 0 guidelines, got %d", len(guidelines))
	}
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(warnings))
	}
}
