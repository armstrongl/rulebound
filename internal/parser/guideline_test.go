package parser

import (
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
