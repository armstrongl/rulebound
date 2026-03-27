package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── kebabToTitle ──────────────────────────────────────────────────────────────

func TestKebabToTitle_BasicConversion(t *testing.T) {
	got := kebabToTitle("active-voice")
	want := "Active Voice"
	if got != want {
		t.Errorf("kebabToTitle(%q) = %q, want %q", "active-voice", got, want)
	}
}

func TestKebabToTitle_SingleWord(t *testing.T) {
	got := kebabToTitle("grammar")
	want := "Grammar"
	if got != want {
		t.Errorf("kebabToTitle(%q) = %q, want %q", "grammar", got, want)
	}
}

func TestKebabToTitle_MultipleSegments(t *testing.T) {
	got := kebabToTitle("language-and-grammar")
	want := "Language And Grammar"
	if got != want {
		t.Errorf("kebabToTitle(%q) = %q, want %q", "language-and-grammar", got, want)
	}
}

func TestKebabToTitle_EmptyString(t *testing.T) {
	got := kebabToTitle("")
	if got != "" {
		t.Errorf("kebabToTitle(%q) = %q, want %q", "", got, "")
	}
}

func TestKebabToTitle_AlreadyCapitalized(t *testing.T) {
	got := kebabToTitle("Already-Caps")
	want := "Already Caps"
	if got != want {
		t.Errorf("kebabToTitle(%q) = %q, want %q", "Already-Caps", got, want)
	}
}

// ── parsePageFrontmatter ──────────────────────────────────────────────────────

func TestParsePageFrontmatter_ValidFull(t *testing.T) {
	input := "---\ntitle: Active Voice\ndescription: Use active voice for clarity\n---\nBody content here."
	title, desc, body, err := parsePageFrontmatter(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if title != "Active Voice" {
		t.Errorf("title = %q, want %q", title, "Active Voice")
	}
	if desc != "Use active voice for clarity" {
		t.Errorf("description = %q, want %q", desc, "Use active voice for clarity")
	}
	if body != "Body content here." {
		t.Errorf("body = %q, want %q", body, "Body content here.")
	}
}

func TestParsePageFrontmatter_TitleOnly(t *testing.T) {
	input := "---\ntitle: Minimal Page\n---\nSome content."
	title, desc, body, err := parsePageFrontmatter(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if title != "Minimal Page" {
		t.Errorf("title = %q, want %q", title, "Minimal Page")
	}
	if desc != "" {
		t.Errorf("description = %q, want empty", desc)
	}
	if body != "Some content." {
		t.Errorf("body = %q, want %q", body, "Some content.")
	}
}

func TestParsePageFrontmatter_NoFrontmatter(t *testing.T) {
	input := "Just plain text."
	_, _, _, err := parsePageFrontmatter(input)
	if err == nil {
		t.Error("expected error for content without frontmatter")
	}
}

func TestParsePageFrontmatter_InvalidYAML(t *testing.T) {
	input := "---\ntitle: [unclosed\n---\nBody."
	_, _, _, err := parsePageFrontmatter(input)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestParsePageFrontmatter_EmptyBody(t *testing.T) {
	input := "---\ntitle: Frontmatter Only\n---\n"
	title, _, body, err := parsePageFrontmatter(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if title != "Frontmatter Only" {
		t.Errorf("title = %q, want %q", title, "Frontmatter Only")
	}
	if body != "" {
		t.Errorf("body = %q, want empty", body)
	}
}

func TestParsePageFrontmatter_WindowsLineEndings(t *testing.T) {
	input := "---\r\ntitle: CRLF Page\r\n---\r\nWindows body."
	title, _, body, err := parsePageFrontmatter(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if title != "CRLF Page" {
		t.Errorf("title = %q, want %q", title, "CRLF Page")
	}
	if body != "Windows body." {
		t.Errorf("body = %q, want %q", body, "Windows body.")
	}
}

// ── parsePages: no pages/ directory ───────────────────────────────────────────

func TestParsePages_NoPagesDir(t *testing.T) {
	dir := t.TempDir()
	tree, warnings, err := parsePages(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tree != nil {
		t.Error("expected nil SectionTree when pages/ absent")
	}
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(warnings))
	}
}

// ── parsePages: empty pages/ directory ────────────────────────────────────────

func TestParsePages_EmptyPagesDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "pages"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	tree, warnings, err := parsePages(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tree != nil {
		t.Error("expected nil SectionTree for empty pages/ directory")
	}
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(warnings))
	}
}

// ── parsePages: single level with 3 .md files ────────────────────────────────

func TestParsePages_SingleLevel(t *testing.T) {
	dir := t.TempDir()
	pagesDir := filepath.Join(dir, "pages")
	if err := os.MkdirAll(pagesDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	writeFile(t, filepath.Join(pagesDir, "active-voice.md"),
		"---\ntitle: Active Voice\ndescription: Write actively\n---\nUse active voice.")
	writeFile(t, filepath.Join(pagesDir, "passive-voice.md"),
		"---\ntitle: Passive Voice\n---\nAvoid passive voice.")
	writeFile(t, filepath.Join(pagesDir, "contractions.md"),
		"---\ntitle: Contractions\n---\nUse contractions.")

	tree, warnings, err := parsePages(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tree == nil {
		t.Fatal("expected non-nil SectionTree")
	}
	if len(tree.Pages) != 3 {
		t.Fatalf("expected 3 pages, got %d", len(tree.Pages))
	}
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(warnings))
	}

	// Check pages are sorted alphabetically by default
	if tree.Pages[0].Title != "Active Voice" {
		t.Errorf("Pages[0].Title = %q, want %q", tree.Pages[0].Title, "Active Voice")
	}
	if tree.Pages[1].Title != "Contractions" {
		t.Errorf("Pages[1].Title = %q, want %q", tree.Pages[1].Title, "Contractions")
	}
	if tree.Pages[2].Title != "Passive Voice" {
		t.Errorf("Pages[2].Title = %q, want %q", tree.Pages[2].Title, "Passive Voice")
	}

	// Check paths
	if tree.Pages[0].Path != "/pages/active-voice/" {
		t.Errorf("Pages[0].Path = %q, want %q", tree.Pages[0].Path, "/pages/active-voice/")
	}
	if tree.Pages[1].Path != "/pages/contractions/" {
		t.Errorf("Pages[1].Path = %q, want %q", tree.Pages[1].Path, "/pages/contractions/")
	}

	// Check description
	if tree.Pages[0].Description != "Write actively" {
		t.Errorf("Pages[0].Description = %q, want %q", tree.Pages[0].Description, "Write actively")
	}

	// Check body
	if tree.Pages[0].Body != "Use active voice." {
		t.Errorf("Pages[0].Body = %q, want %q", tree.Pages[0].Body, "Use active voice.")
	}
}

// ── parsePages: nested 3 levels deep ──────────────────────────────────────────

func TestParsePages_Nested3Levels(t *testing.T) {
	dir := t.TempDir()
	pagesDir := filepath.Join(dir, "pages")

	// Level 1: pages/
	writeFile(t, filepath.Join(pagesDir, "top.md"), "---\ntitle: Top Page\n---\nTop.")

	// Level 2: pages/language/
	langDir := filepath.Join(pagesDir, "language")
	writeFile(t, filepath.Join(langDir, "intro.md"), "---\ntitle: Language Intro\n---\nIntro.")

	// Level 3: pages/language/grammar/
	gramDir := filepath.Join(langDir, "grammar")
	writeFile(t, filepath.Join(gramDir, "nouns.md"), "---\ntitle: Nouns\n---\nNouns.")

	tree, _, err := parsePages(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tree == nil {
		t.Fatal("expected non-nil SectionTree")
	}

	// Level 1 has 1 page
	if len(tree.Pages) != 1 {
		t.Fatalf("root pages: expected 1, got %d", len(tree.Pages))
	}
	if tree.Pages[0].Title != "Top Page" {
		t.Errorf("root page title = %q, want %q", tree.Pages[0].Title, "Top Page")
	}

	// Level 2: 1 child
	if len(tree.Children) != 1 {
		t.Fatalf("root children: expected 1, got %d", len(tree.Children))
	}
	langTree := tree.Children[0]
	if langTree.Name != "language" {
		t.Errorf("child name = %q, want %q", langTree.Name, "language")
	}
	if langTree.Title != "Language" {
		t.Errorf("child title = %q, want %q", langTree.Title, "Language")
	}
	if len(langTree.Pages) != 1 {
		t.Fatalf("language pages: expected 1, got %d", len(langTree.Pages))
	}
	if langTree.Pages[0].Path != "/pages/language/intro/" {
		t.Errorf("language page path = %q, want %q", langTree.Pages[0].Path, "/pages/language/intro/")
	}

	// Level 3: 1 grandchild
	if len(langTree.Children) != 1 {
		t.Fatalf("language children: expected 1, got %d", len(langTree.Children))
	}
	gramTree := langTree.Children[0]
	if gramTree.Name != "grammar" {
		t.Errorf("grandchild name = %q, want %q", gramTree.Name, "grammar")
	}
	if len(gramTree.Pages) != 1 {
		t.Fatalf("grammar pages: expected 1, got %d", len(gramTree.Pages))
	}
	if gramTree.Pages[0].Path != "/pages/language/grammar/nouns/" {
		t.Errorf("grammar page path = %q, want %q", gramTree.Pages[0].Path, "/pages/language/grammar/nouns/")
	}
}

// ── parsePages: depth cap at 7+ levels ────────────────────────────────────────

func TestParsePages_DepthCap(t *testing.T) {
	dir := t.TempDir()

	// Build 8 levels deep: pages/a/b/c/d/e/f/g/h/
	current := filepath.Join(dir, "pages")
	levels := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	for _, lvl := range levels {
		current = filepath.Join(current, lvl)
	}
	writeFile(t, filepath.Join(current, "deep.md"), "---\ntitle: Deep Page\n---\nDeep.")

	tree, warnings, err := parsePages(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tree == nil {
		t.Fatal("expected non-nil SectionTree")
	}

	// Should have at least one depth cap warning
	found := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "depth") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected depth cap warning for 8-level nesting")
	}
}

// ── _meta.yml parsing ─────────────────────────────────────────────────────────

func TestParsePages_MetaYMLAllFields(t *testing.T) {
	dir := t.TempDir()
	pagesDir := filepath.Join(dir, "pages")

	writeFile(t, filepath.Join(pagesDir, "_meta.yml"),
		"title: Content Pages\norder:\n  - contractions\n  - active-voice\ncollapsed: true\nhidden:\n  - contractions\nrules_title: Custom Rules\n")
	writeFile(t, filepath.Join(pagesDir, "active-voice.md"), "---\ntitle: Active Voice\n---\nBody.")
	writeFile(t, filepath.Join(pagesDir, "contractions.md"), "---\ntitle: Contractions\n---\nBody.")

	tree, warnings, err := parsePages(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tree == nil {
		t.Fatal("expected non-nil SectionTree")
	}
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d: %v", len(warnings), warnings)
	}

	if tree.Meta == nil {
		t.Fatal("expected non-nil Meta")
	}
	if tree.Title != "Content Pages" {
		t.Errorf("Title = %q, want %q", tree.Title, "Content Pages")
	}
	if !tree.Meta.Collapsed {
		t.Error("Meta.Collapsed = false, want true")
	}
	if tree.Meta.RulesTitle != "Custom Rules" {
		t.Errorf("Meta.RulesTitle = %q, want %q", tree.Meta.RulesTitle, "Custom Rules")
	}
}

func TestParsePages_NoMetaYML(t *testing.T) {
	dir := t.TempDir()
	pagesDir := filepath.Join(dir, "pages")

	writeFile(t, filepath.Join(pagesDir, "alpha.md"), "---\ntitle: Alpha\n---\nA.")
	writeFile(t, filepath.Join(pagesDir, "beta.md"), "---\ntitle: Beta\n---\nB.")

	tree, _, err := parsePages(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tree == nil {
		t.Fatal("expected non-nil SectionTree")
	}

	if tree.Meta != nil {
		t.Error("expected nil Meta when _meta.yml absent")
	}

	// Pages should be alphabetical
	if tree.Pages[0].Title != "Alpha" {
		t.Errorf("Pages[0].Title = %q, want %q", tree.Pages[0].Title, "Alpha")
	}
	if tree.Pages[1].Title != "Beta" {
		t.Errorf("Pages[1].Title = %q, want %q", tree.Pages[1].Title, "Beta")
	}
}

func TestParsePages_MetaYMLOrder(t *testing.T) {
	dir := t.TempDir()
	pagesDir := filepath.Join(dir, "pages")

	writeFile(t, filepath.Join(pagesDir, "_meta.yml"), "order:\n  - zebra\n  - alpha\n")
	writeFile(t, filepath.Join(pagesDir, "alpha.md"), "---\ntitle: Alpha\n---\nA.")
	writeFile(t, filepath.Join(pagesDir, "beta.md"), "---\ntitle: Beta\n---\nB.")
	writeFile(t, filepath.Join(pagesDir, "zebra.md"), "---\ntitle: Zebra\n---\nZ.")

	tree, _, err := parsePages(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tree == nil {
		t.Fatal("expected non-nil SectionTree")
	}

	// Order: zebra, alpha (from _meta.yml), then beta (unlisted, alphabetical)
	if len(tree.Pages) != 3 {
		t.Fatalf("expected 3 pages, got %d", len(tree.Pages))
	}
	if tree.Pages[0].Title != "Zebra" {
		t.Errorf("Pages[0].Title = %q, want %q", tree.Pages[0].Title, "Zebra")
	}
	if tree.Pages[1].Title != "Alpha" {
		t.Errorf("Pages[1].Title = %q, want %q", tree.Pages[1].Title, "Alpha")
	}
	if tree.Pages[2].Title != "Beta" {
		t.Errorf("Pages[2].Title = %q, want %q", tree.Pages[2].Title, "Beta")
	}
}

func TestParsePages_MetaYMLMalformed(t *testing.T) {
	dir := t.TempDir()
	pagesDir := filepath.Join(dir, "pages")

	writeFile(t, filepath.Join(pagesDir, "_meta.yml"), "title: [unclosed\n")
	writeFile(t, filepath.Join(pagesDir, "page.md"), "---\ntitle: Page\n---\nBody.")

	tree, warnings, err := parsePages(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tree == nil {
		t.Fatal("expected non-nil SectionTree")
	}

	// Should have a warning about malformed _meta.yml
	found := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "_meta.yml") || strings.Contains(w.File, "_meta.yml") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about malformed _meta.yml, got %v", warnings)
	}

	// Should still parse with defaults (alphabetical order, nil Meta)
	if len(tree.Pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(tree.Pages))
	}
}

// ── Missing title in frontmatter ──────────────────────────────────────────────

func TestParsePages_MissingTitle(t *testing.T) {
	dir := t.TempDir()
	pagesDir := filepath.Join(dir, "pages")

	writeFile(t, filepath.Join(pagesDir, "active-voice.md"),
		"---\ndescription: No title here\n---\nBody content.")

	tree, warnings, err := parsePages(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tree == nil {
		t.Fatal("expected non-nil SectionTree")
	}
	if len(tree.Pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(tree.Pages))
	}

	// Title derived from filename: "active-voice" -> "Active Voice"
	if tree.Pages[0].Title != "Active Voice" {
		t.Errorf("Title = %q, want %q (derived from filename)", tree.Pages[0].Title, "Active Voice")
	}

	// Should have a warning about missing title
	found := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "title") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning about missing title in frontmatter")
	}
}

// ── Page with only frontmatter, no body ───────────────────────────────────────

func TestParsePages_FrontmatterOnly(t *testing.T) {
	dir := t.TempDir()
	pagesDir := filepath.Join(dir, "pages")

	writeFile(t, filepath.Join(pagesDir, "empty-body.md"),
		"---\ntitle: Empty Body Page\n---\n")

	tree, _, err := parsePages(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tree == nil {
		t.Fatal("expected non-nil SectionTree")
	}
	if len(tree.Pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(tree.Pages))
	}
	if tree.Pages[0].Title != "Empty Body Page" {
		t.Errorf("Title = %q, want %q", tree.Pages[0].Title, "Empty Body Page")
	}
	if tree.Pages[0].Body != "" {
		t.Errorf("Body = %q, want empty", tree.Pages[0].Body)
	}
}

// ── _index.md present ─────────────────────────────────────────────────────────

func TestParsePages_IndexMD(t *testing.T) {
	dir := t.TempDir()
	pagesDir := filepath.Join(dir, "pages")

	writeFile(t, filepath.Join(pagesDir, "_index.md"),
		"---\ntitle: Pages Hub\ndescription: Overview of all pages\n---\nThis is the hub page.")
	writeFile(t, filepath.Join(pagesDir, "page1.md"),
		"---\ntitle: Page One\n---\nContent.")

	tree, _, err := parsePages(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tree == nil {
		t.Fatal("expected non-nil SectionTree")
	}

	// _index.md should be attached to IndexPage, not in Pages slice
	if tree.IndexPage == nil {
		t.Fatal("expected non-nil IndexPage")
	}
	if tree.IndexPage.Title != "Pages Hub" {
		t.Errorf("IndexPage.Title = %q, want %q", tree.IndexPage.Title, "Pages Hub")
	}
	if tree.IndexPage.Description != "Overview of all pages" {
		t.Errorf("IndexPage.Description = %q, want %q", tree.IndexPage.Description, "Overview of all pages")
	}
	if tree.IndexPage.Body != "This is the hub page." {
		t.Errorf("IndexPage.Body = %q, want %q", tree.IndexPage.Body, "This is the hub page.")
	}

	// Pages slice should only contain page1, not _index.md
	if len(tree.Pages) != 1 {
		t.Fatalf("expected 1 page (excluding _index.md), got %d", len(tree.Pages))
	}
	if tree.Pages[0].Title != "Page One" {
		t.Errorf("Pages[0].Title = %q, want %q", tree.Pages[0].Title, "Page One")
	}
}

func TestParsePages_NoIndexMD(t *testing.T) {
	dir := t.TempDir()
	pagesDir := filepath.Join(dir, "pages")

	writeFile(t, filepath.Join(pagesDir, "page1.md"), "---\ntitle: Page One\n---\nContent.")

	tree, _, err := parsePages(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tree == nil {
		t.Fatal("expected non-nil SectionTree")
	}
	if tree.IndexPage != nil {
		t.Errorf("expected nil IndexPage when _index.md absent, got %+v", tree.IndexPage)
	}
}

// ── Non-.md files silently ignored ────────────────────────────────────────────

func TestParsePages_IgnoresNonMDFiles(t *testing.T) {
	dir := t.TempDir()
	pagesDir := filepath.Join(dir, "pages")

	writeFile(t, filepath.Join(pagesDir, "page.md"), "---\ntitle: Page\n---\nBody.")
	writeFile(t, filepath.Join(pagesDir, "notes.txt"), "Some notes")
	writeFile(t, filepath.Join(pagesDir, "image.png"), "binary data")
	writeFile(t, filepath.Join(pagesDir, "data.json"), `{"key":"value"}`)

	tree, warnings, err := parsePages(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tree == nil {
		t.Fatal("expected non-nil SectionTree")
	}
	if len(tree.Pages) != 1 {
		t.Errorf("expected 1 page, got %d", len(tree.Pages))
	}
	// Non-md files should not produce warnings
	for _, w := range warnings {
		if strings.Contains(w.File, ".txt") || strings.Contains(w.File, ".png") || strings.Contains(w.File, ".json") {
			t.Errorf("non-.md file produced warning: %v", w)
		}
	}
}

// ── Hidden pages ──────────────────────────────────────────────────────────────

func TestParsePages_HiddenPages(t *testing.T) {
	dir := t.TempDir()
	pagesDir := filepath.Join(dir, "pages")

	writeFile(t, filepath.Join(pagesDir, "_meta.yml"), "hidden:\n  - draft-page\n")
	writeFile(t, filepath.Join(pagesDir, "draft-page.md"), "---\ntitle: Draft Page\n---\nDraft content.")
	writeFile(t, filepath.Join(pagesDir, "published.md"), "---\ntitle: Published\n---\nPublished content.")

	tree, _, err := parsePages(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tree == nil {
		t.Fatal("expected non-nil SectionTree")
	}
	if len(tree.Pages) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(tree.Pages))
	}

	// Find draft-page
	var draft, published *Page
	for _, p := range tree.Pages {
		switch p.Title {
		case "Draft Page":
			draft = p
		case "Published":
			published = p
		}
	}
	if draft == nil {
		t.Fatal("expected to find draft page")
	}
	if !draft.Hidden {
		t.Error("draft page Hidden = false, want true")
	}
	if published == nil {
		t.Fatal("expected to find published page")
	}
	if published.Hidden {
		t.Error("published page Hidden = true, want false")
	}
}

// ── SectionTree path and title computation ────────────────────────────────────

func TestParsePages_SectionTreePathAndTitle(t *testing.T) {
	dir := t.TempDir()
	pagesDir := filepath.Join(dir, "pages")
	subDir := filepath.Join(pagesDir, "language-and-grammar")

	writeFile(t, filepath.Join(subDir, "intro.md"), "---\ntitle: Intro\n---\nIntro.")

	tree, _, err := parsePages(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tree == nil {
		t.Fatal("expected non-nil SectionTree")
	}

	// Root section
	if tree.Name != "pages" {
		t.Errorf("root Name = %q, want %q", tree.Name, "pages")
	}
	if tree.Path != "/pages/" {
		t.Errorf("root Path = %q, want %q", tree.Path, "/pages/")
	}

	// Child section: title derived from directory name
	if len(tree.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(tree.Children))
	}
	child := tree.Children[0]
	if child.Name != "language-and-grammar" {
		t.Errorf("child Name = %q, want %q", child.Name, "language-and-grammar")
	}
	if child.Title != "Language And Grammar" {
		t.Errorf("child Title = %q, want %q", child.Title, "Language And Grammar")
	}
	if child.Path != "/pages/language-and-grammar/" {
		t.Errorf("child Path = %q, want %q", child.Path, "/pages/language-and-grammar/")
	}
}

func TestParsePages_SectionTitleFromMeta(t *testing.T) {
	dir := t.TempDir()
	pagesDir := filepath.Join(dir, "pages")
	subDir := filepath.Join(pagesDir, "lang")

	writeFile(t, filepath.Join(subDir, "_meta.yml"), "title: Language & Grammar\n")
	writeFile(t, filepath.Join(subDir, "intro.md"), "---\ntitle: Intro\n---\nIntro.")

	tree, _, err := parsePages(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tree == nil {
		t.Fatal("expected non-nil SectionTree")
	}
	if len(tree.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(tree.Children))
	}
	if tree.Children[0].Title != "Language & Grammar" {
		t.Errorf("child Title = %q, want %q", tree.Children[0].Title, "Language & Grammar")
	}
}

// ── SourceFile field ──────────────────────────────────────────────────────────

func TestParsePages_SourceFile(t *testing.T) {
	dir := t.TempDir()
	pagesDir := filepath.Join(dir, "pages")
	pagePath := filepath.Join(pagesDir, "test.md")

	writeFile(t, pagePath, "---\ntitle: Test\n---\nBody.")

	tree, _, err := parsePages(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tree == nil {
		t.Fatal("expected non-nil SectionTree")
	}
	if len(tree.Pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(tree.Pages))
	}
	if tree.Pages[0].SourceFile != pagePath {
		t.Errorf("SourceFile = %q, want %q", tree.Pages[0].SourceFile, pagePath)
	}
}

// ── rules/ directory collision ────────────────────────────────────────────────

func TestParsePages_RulesCollision(t *testing.T) {
	dir := t.TempDir()
	pagesDir := filepath.Join(dir, "pages")
	rulesDir := filepath.Join(pagesDir, "rules")

	writeFile(t, filepath.Join(pagesDir, "_meta.yml"), "order:\n  - rules\n")
	writeFile(t, filepath.Join(rulesDir, "custom-rule.md"), "---\ntitle: Custom Rule\n---\nBody.")

	tree, warnings, err := parsePages(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tree == nil {
		t.Fatal("expected non-nil SectionTree")
	}

	// rules/ directory should take precedence
	if len(tree.Children) != 1 {
		t.Fatalf("expected 1 child (rules dir), got %d", len(tree.Children))
	}
	if tree.Children[0].Name != "rules" {
		t.Errorf("child Name = %q, want %q", tree.Children[0].Name, "rules")
	}

	// Should have a warning about collision
	found := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "rules") && strings.Contains(w.Message, "collision") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning about rules/ directory collision")
	}
}

// ── Test helper ───────────────────────────────────────────────────────────────

// writeFile creates the parent directories and writes content to a file.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
}
