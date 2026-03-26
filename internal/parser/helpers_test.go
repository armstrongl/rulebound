package parser

import (
	"os"
	"path/filepath"
	"testing"
)

// ── stripFrontmatter ─────────────────────────────────────────────────────────

func TestStripFrontmatter_ValidFrontmatter(t *testing.T) {
	input := "---\ntitle: Hello\n---\nBody content here."
	got := stripFrontmatter(input)
	want := "Body content here."
	if got != want {
		t.Errorf("stripFrontmatter with valid frontmatter:\ngot  %q\nwant %q", got, want)
	}
}

func TestStripFrontmatter_NoFrontmatter(t *testing.T) {
	input := "Just some plain text.\nNo frontmatter at all."
	got := stripFrontmatter(input)
	if got != input {
		t.Errorf("stripFrontmatter with no frontmatter:\ngot  %q\nwant %q", got, input)
	}
}

func TestStripFrontmatter_OnlyOpeningFence(t *testing.T) {
	input := "---\ntitle: Hello\nNo closing fence here."
	got := stripFrontmatter(input)
	if got != input {
		t.Errorf("stripFrontmatter with only opening fence:\ngot  %q\nwant %q", got, input)
	}
}

func TestStripFrontmatter_EmptyString(t *testing.T) {
	got := stripFrontmatter("")
	if got != "" {
		t.Errorf("stripFrontmatter with empty string: got %q, want empty", got)
	}
}

func TestStripFrontmatter_FrontmatterWithEmptyBody(t *testing.T) {
	input := "---\ntitle: Hello\n---\n"
	got := stripFrontmatter(input)
	if got != "" {
		t.Errorf("stripFrontmatter with empty body:\ngot  %q\nwant %q", got, "")
	}
}

func TestStripFrontmatter_TripleDashInBody(t *testing.T) {
	// Content that does NOT start with --- should be returned unchanged,
	// even if --- appears later in the text.
	input := "Some intro text.\n---\nMore text after a horizontal rule."
	got := stripFrontmatter(input)
	if got != input {
		t.Errorf("stripFrontmatter with --- in body (not at start):\ngot  %q\nwant %q", got, input)
	}
}

func TestStripFrontmatter_TripleDashInBodyAfterFrontmatter(t *testing.T) {
	// Valid frontmatter followed by body that itself contains ---
	input := "---\ntitle: Test\n---\nBody line 1.\n---\nBody line 2."
	got := stripFrontmatter(input)
	want := "Body line 1.\n---\nBody line 2."
	if got != want {
		t.Errorf("stripFrontmatter with --- in body after valid frontmatter:\ngot  %q\nwant %q", got, want)
	}
}

func TestStripFrontmatter_WindowsLineEndings(t *testing.T) {
	input := "---\r\ntitle: Hello\r\n---\r\nBody with CRLF."
	got := stripFrontmatter(input)
	want := "Body with CRLF."
	if got != want {
		t.Errorf("stripFrontmatter with Windows line endings:\ngot  %q\nwant %q", got, want)
	}
}

func TestStripFrontmatter_FenceNotFollowedByNewline(t *testing.T) {
	// "---something" at the start should not be treated as frontmatter.
	input := "---not a fence\ntitle: Hello\n---\nBody."
	got := stripFrontmatter(input)
	if got != input {
		t.Errorf("stripFrontmatter with fence not followed by newline:\ngot  %q\nwant %q", got, input)
	}
}

func TestStripFrontmatter_MultipleFrontmatterFields(t *testing.T) {
	input := "---\ntitle: Hello\nauthor: World\ntags:\n  - go\n  - test\n---\nActual body."
	got := stripFrontmatter(input)
	want := "Actual body."
	if got != want {
		t.Errorf("stripFrontmatter with multiple fields:\ngot  %q\nwant %q", got, want)
	}
}

// ── readCompanion ────────────────────────────────────────────────────────────

func TestReadCompanion_WithMatchingMD(t *testing.T) {
	dir := t.TempDir()
	ymlPath := filepath.Join(dir, "Rule.yml")
	mdPath := filepath.Join(dir, "Rule.md")

	if err := os.WriteFile(ymlPath, []byte("extends: existence\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mdPath, []byte("Companion body content."), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := readCompanion(ymlPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "Companion body content."
	if got != want {
		t.Errorf("readCompanion with matching .md:\ngot  %q\nwant %q", got, want)
	}
}

func TestReadCompanion_NoMDFile(t *testing.T) {
	dir := t.TempDir()
	ymlPath := filepath.Join(dir, "Rule.yml")

	if err := os.WriteFile(ymlPath, []byte("extends: existence\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := readCompanion(ymlPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("readCompanion with no .md file: got %q, want empty", got)
	}
}

func TestReadCompanion_MDWithFrontmatter(t *testing.T) {
	dir := t.TempDir()
	ymlPath := filepath.Join(dir, "Rule.yml")
	mdPath := filepath.Join(dir, "Rule.md")

	if err := os.WriteFile(ymlPath, []byte("extends: existence\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mdContent := "---\ntitle: Rule Docs\nauthor: Test\n---\nStripped body here."
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := readCompanion(ymlPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "Stripped body here."
	if got != want {
		t.Errorf("readCompanion with frontmatter in .md:\ngot  %q\nwant %q", got, want)
	}
}

func TestReadCompanion_MDWithOnlyFrontmatter(t *testing.T) {
	dir := t.TempDir()
	ymlPath := filepath.Join(dir, "Rule.yml")
	mdPath := filepath.Join(dir, "Rule.md")

	if err := os.WriteFile(ymlPath, []byte("extends: existence\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mdContent := "---\ntitle: Empty Body\n---\n"
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := readCompanion(ymlPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("readCompanion with frontmatter-only .md: got %q, want empty", got)
	}
}

// ── nameFromPath ─────────────────────────────────────────────────────────────

func TestNameFromPath_StandardPath(t *testing.T) {
	got := nameFromPath("Microsoft/Avoid.yml")
	want := "Avoid"
	if got != want {
		t.Errorf("nameFromPath(%q): got %q, want %q", "Microsoft/Avoid.yml", got, want)
	}
}

func TestNameFromPath_MultipleDots(t *testing.T) {
	got := nameFromPath("styles/My.Custom.Rule.yml")
	want := "My.Custom.Rule"
	if got != want {
		t.Errorf("nameFromPath(%q): got %q, want %q", "styles/My.Custom.Rule.yml", got, want)
	}
}

func TestNameFromPath_JustFilename(t *testing.T) {
	got := nameFromPath("Avoid.yml")
	want := "Avoid"
	if got != want {
		t.Errorf("nameFromPath(%q): got %q, want %q", "Avoid.yml", got, want)
	}
}

func TestNameFromPath_YamlExtension(t *testing.T) {
	got := nameFromPath("/some/path/Rule.yaml")
	want := "Rule"
	if got != want {
		t.Errorf("nameFromPath(%q): got %q, want %q", "/some/path/Rule.yaml", got, want)
	}
}

func TestNameFromPath_DeepPath(t *testing.T) {
	got := nameFromPath("/usr/local/styles/Microsoft/Headings.yml")
	want := "Headings"
	if got != want {
		t.Errorf("nameFromPath(%q): got %q, want %q", "/usr/local/styles/Microsoft/Headings.yml", got, want)
	}
}
