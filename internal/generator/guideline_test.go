package generator_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larah/rulebound/internal/generator"
	"github.com/larah/rulebound/internal/parser"
)

func makeGuideline(name, title string, weight int) *parser.Guideline {
	return &parser.Guideline{
		Name:        name,
		Title:       title,
		Description: "A test guideline",
		Weight:      weight,
		Body:        "Guideline prose content.",
	}
}

// ── GenerateGuideline ────────────────────────────────────────────────────────

func TestGenerateGuideline_CreatesFile(t *testing.T) {
	outDir := t.TempDir()
	g := makeGuideline("voice-and-tone", "Voice and Tone", 10)

	err := generator.GenerateGuideline(g, outDir)
	if err != nil {
		t.Fatalf("GenerateGuideline: %v", err)
	}

	path := filepath.Join(outDir, "voice-and-tone.md")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist", path)
	}
}

func TestGenerateGuideline_HasCorrectFrontmatter(t *testing.T) {
	outDir := t.TempDir()
	g := makeGuideline("voice-and-tone", "Voice and Tone", 10)
	g.Description = "How to write in our voice"

	if err := generator.GenerateGuideline(g, outDir); err != nil {
		t.Fatalf("GenerateGuideline: %v", err)
	}

	content := readFile(t, filepath.Join(outDir, "voice-and-tone.md"))

	if !strings.HasPrefix(content, "---\n") {
		t.Error("expected frontmatter delimiter at start")
	}
	if !strings.Contains(content, "title:") {
		t.Errorf("missing title in frontmatter: %s", content)
	}
	if !strings.Contains(content, "type: guideline") {
		t.Errorf("missing type: guideline in frontmatter: %s", content)
	}
	if !strings.Contains(content, "weight: 10") {
		t.Errorf("missing weight in frontmatter: %s", content)
	}
	if !strings.Contains(content, "description:") {
		t.Errorf("missing description in frontmatter: %s", content)
	}
}

func TestGenerateGuideline_BodyContent(t *testing.T) {
	outDir := t.TempDir()
	g := makeGuideline("inclusive-language", "Inclusive Language", 20)
	g.Body = "Use gender-neutral language."

	if err := generator.GenerateGuideline(g, outDir); err != nil {
		t.Fatalf("GenerateGuideline: %v", err)
	}

	content := readFile(t, filepath.Join(outDir, "inclusive-language.md"))
	if !strings.Contains(content, "Use gender-neutral language.") {
		t.Errorf("missing body content: %s", content)
	}
}

func TestGenerateGuideline_EmptyBody(t *testing.T) {
	outDir := t.TempDir()
	g := makeGuideline("stub", "Stub Guideline", 0)
	g.Body = ""

	if err := generator.GenerateGuideline(g, outDir); err != nil {
		t.Fatalf("GenerateGuideline: %v", err)
	}

	path := filepath.Join(outDir, "stub.md")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist even with empty body", path)
	}
}
