package generator_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/armstrongl/rulebound/internal/generator"
	"github.com/armstrongl/rulebound/internal/parser"
)

// ── GeneratePage ──────────────────────────────────────────────────────────────

func TestGeneratePage_SinglePage_TypeInFrontmatter(t *testing.T) {
	contentDir := t.TempDir()
	pagesDir := filepath.Join(contentDir, "pages", "language")
	if err := os.MkdirAll(pagesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	page := &parser.Page{
		Title:       "Active Voice",
		Description: "Guidelines for using active voice",
		Body:        "Use active voice whenever possible.",
		Path:        "/pages/language/active-voice/",
	}

	if err := generator.GeneratePage(page, contentDir); err != nil {
		t.Fatalf("GeneratePage: %v", err)
	}

	path := filepath.Join(contentDir, "pages", "language", "active-voice.md")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("expected file %s to exist", path)
	}

	content := readFile(t, path)
	if !strings.Contains(content, "type: page") {
		t.Errorf("missing type: page in frontmatter:\n%s", content)
	}
}

func TestGeneratePage_TitleAndDescription(t *testing.T) {
	contentDir := t.TempDir()
	pagesDir := filepath.Join(contentDir, "pages", "language")
	if err := os.MkdirAll(pagesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	page := &parser.Page{
		Title:       "Active Voice",
		Description: "Guidelines for using active voice",
		Body:        "",
		Path:        "/pages/language/active-voice/",
	}

	if err := generator.GeneratePage(page, contentDir); err != nil {
		t.Fatalf("GeneratePage: %v", err)
	}

	content := readFile(t, filepath.Join(contentDir, "pages", "language", "active-voice.md"))
	if !strings.Contains(content, "title:") {
		t.Errorf("missing title in frontmatter:\n%s", content)
	}
	if !strings.Contains(content, "Active Voice") {
		t.Errorf("missing title value in frontmatter:\n%s", content)
	}
	if !strings.Contains(content, "description:") {
		t.Errorf("missing description in frontmatter:\n%s", content)
	}
	if !strings.Contains(content, "Guidelines for using active voice") {
		t.Errorf("missing description value in frontmatter:\n%s", content)
	}
}

func TestGeneratePage_BodyPreserved(t *testing.T) {
	contentDir := t.TempDir()
	pagesDir := filepath.Join(contentDir, "pages", "formatting")
	if err := os.MkdirAll(pagesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	page := &parser.Page{
		Title: "Headings",
		Body:  "## Best practices\n\nAlways use sentence case for headings.",
		Path:  "/pages/formatting/headings/",
	}

	if err := generator.GeneratePage(page, contentDir); err != nil {
		t.Fatalf("GeneratePage: %v", err)
	}

	content := readFile(t, filepath.Join(contentDir, "pages", "formatting", "headings.md"))
	if !strings.Contains(content, "## Best practices") {
		t.Errorf("body content not preserved:\n%s", content)
	}
	if !strings.Contains(content, "Always use sentence case for headings.") {
		t.Errorf("body content not preserved:\n%s", content)
	}
}

func TestGeneratePage_HiddenPage_PagefindFalse(t *testing.T) {
	contentDir := t.TempDir()
	pagesDir := filepath.Join(contentDir, "pages", "internal")
	if err := os.MkdirAll(pagesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	page := &parser.Page{
		Title:  "Internal Notes",
		Body:   "This is hidden content.",
		Path:   "/pages/internal/notes/",
		Hidden: true,
	}

	if err := generator.GeneratePage(page, contentDir); err != nil {
		t.Fatalf("GeneratePage: %v", err)
	}

	content := readFile(t, filepath.Join(contentDir, "pages", "internal", "notes.md"))
	if !strings.Contains(content, "pagefind: false") {
		t.Errorf("hidden page missing pagefind: false in frontmatter:\n%s", content)
	}
}

func TestGeneratePage_NotHidden_NoPagefindField(t *testing.T) {
	contentDir := t.TempDir()
	pagesDir := filepath.Join(contentDir, "pages", "language")
	if err := os.MkdirAll(pagesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	page := &parser.Page{
		Title:  "Active Voice",
		Body:   "Use active voice.",
		Path:   "/pages/language/active-voice/",
		Hidden: false,
	}

	if err := generator.GeneratePage(page, contentDir); err != nil {
		t.Fatalf("GeneratePage: %v", err)
	}

	content := readFile(t, filepath.Join(contentDir, "pages", "language", "active-voice.md"))
	if strings.Contains(content, "pagefind:") {
		t.Errorf("non-hidden page should not have pagefind field:\n%s", content)
	}
}

// ── generatePageTree ──────────────────────────────────────────────────────────

func TestGeneratePageTree_RootContainerCreated(t *testing.T) {
	contentDir := t.TempDir()

	tree := &parser.SectionTree{
		Name:  "pages",
		Title: "Pages",
		Path:  "/pages/",
	}

	if err := generator.GeneratePageTree(tree, contentDir); err != nil {
		t.Fatalf("GeneratePageTree: %v", err)
	}

	indexPath := filepath.Join(contentDir, "pages", "_index.md")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Fatalf("expected content/pages/_index.md to exist")
	}

	content := readFile(t, indexPath)
	if !strings.Contains(content, "type: page") {
		t.Errorf("root _index.md missing type: page:\n%s", content)
	}
}

func TestGeneratePageTree_SectionWithoutIndexPage_AutoGenerated(t *testing.T) {
	contentDir := t.TempDir()

	tree := &parser.SectionTree{
		Name:  "pages",
		Title: "Pages",
		Path:  "/pages/",
		Children: []*parser.SectionTree{
			{
				Name:  "language",
				Title: "Language & Grammar",
				Path:  "/pages/language/",
				Pages: []*parser.Page{
					{
						Title: "Active Voice",
						Body:  "Use active voice.",
						Path:  "/pages/language/active-voice/",
					},
				},
			},
		},
	}

	if err := generator.GeneratePageTree(tree, contentDir); err != nil {
		t.Fatalf("GeneratePageTree: %v", err)
	}

	indexPath := filepath.Join(contentDir, "pages", "language", "_index.md")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Fatalf("expected auto-generated _index.md at %s", indexPath)
	}

	content := readFile(t, indexPath)
	if !strings.Contains(content, "Language & Grammar") {
		t.Errorf("auto-generated _index.md should have section title:\n%s", content)
	}
	if !strings.Contains(content, "type: page") {
		t.Errorf("auto-generated _index.md missing type: page:\n%s", content)
	}
}

func TestGeneratePageTree_SectionWithIndexPage_AuthorContentPreserved(t *testing.T) {
	contentDir := t.TempDir()

	tree := &parser.SectionTree{
		Name:  "pages",
		Title: "Pages",
		Path:  "/pages/",
		Children: []*parser.SectionTree{
			{
				Name:  "language",
				Title: "Language & Grammar",
				Path:  "/pages/language/",
				IndexPage: &parser.Page{
					Title:       "Language & Grammar Hub",
					Description: "Everything about language",
					Body:        "Welcome to the language section.",
					Path:        "/pages/language/",
				},
			},
		},
	}

	if err := generator.GeneratePageTree(tree, contentDir); err != nil {
		t.Fatalf("GeneratePageTree: %v", err)
	}

	indexPath := filepath.Join(contentDir, "pages", "language", "_index.md")
	content := readFile(t, indexPath)

	if !strings.Contains(content, "Language & Grammar Hub") {
		t.Errorf("author title not preserved:\n%s", content)
	}
	if !strings.Contains(content, "Welcome to the language section.") {
		t.Errorf("author body not preserved:\n%s", content)
	}
	if !strings.Contains(content, "type: page") {
		t.Errorf("type: page not injected:\n%s", content)
	}
}

func TestGeneratePageTree_SectionTitleFromMeta(t *testing.T) {
	contentDir := t.TempDir()

	tree := &parser.SectionTree{
		Name:  "pages",
		Title: "Pages",
		Path:  "/pages/",
		Children: []*parser.SectionTree{
			{
				Name:  "language",
				Title: "Language & Grammar",
				Path:  "/pages/language/",
				Meta: &parser.SectionMeta{
					Title: "Language & Grammar",
				},
			},
		},
	}

	if err := generator.GeneratePageTree(tree, contentDir); err != nil {
		t.Fatalf("GeneratePageTree: %v", err)
	}

	indexPath := filepath.Join(contentDir, "pages", "language", "_index.md")
	content := readFile(t, indexPath)

	if !strings.Contains(content, "Language & Grammar") {
		t.Errorf("title from Meta not used in _index.md:\n%s", content)
	}
}

func TestGeneratePageTree_NestedTree_CorrectDirectoryStructure(t *testing.T) {
	contentDir := t.TempDir()

	tree := &parser.SectionTree{
		Name:  "pages",
		Title: "Pages",
		Path:  "/pages/",
		Children: []*parser.SectionTree{
			{
				Name:  "language",
				Title: "Language",
				Path:  "/pages/language/",
				Pages: []*parser.Page{
					{Title: "Active Voice", Body: "Use active voice.", Path: "/pages/language/active-voice/"},
					{Title: "Pronouns", Body: "Use they/them.", Path: "/pages/language/pronouns/"},
				},
				Children: []*parser.SectionTree{
					{
						Name:  "advanced",
						Title: "Advanced",
						Path:  "/pages/language/advanced/",
						Pages: []*parser.Page{
							{Title: "Subjunctive", Body: "Rare usage.", Path: "/pages/language/advanced/subjunctive/"},
						},
					},
				},
			},
			{
				Name:  "formatting",
				Title: "Formatting",
				Path:  "/pages/formatting/",
				Pages: []*parser.Page{
					{Title: "Headings", Body: "Use sentence case.", Path: "/pages/formatting/headings/"},
				},
			},
		},
	}

	if err := generator.GeneratePageTree(tree, contentDir); err != nil {
		t.Fatalf("GeneratePageTree: %v", err)
	}

	// Verify expected files exist.
	expectedFiles := []string{
		"pages/_index.md",
		"pages/language/_index.md",
		"pages/language/active-voice.md",
		"pages/language/pronouns.md",
		"pages/language/advanced/_index.md",
		"pages/language/advanced/subjunctive.md",
		"pages/formatting/_index.md",
		"pages/formatting/headings.md",
	}
	for _, rel := range expectedFiles {
		fullPath := filepath.Join(contentDir, rel)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", rel)
		}
	}
}

func TestGeneratePageTree_EmptyBody_ValidFile(t *testing.T) {
	contentDir := t.TempDir()

	tree := &parser.SectionTree{
		Name:  "pages",
		Title: "Pages",
		Path:  "/pages/",
		Pages: []*parser.Page{
			{Title: "Empty Page", Body: "", Path: "/pages/empty/"},
		},
	}

	if err := generator.GeneratePageTree(tree, contentDir); err != nil {
		t.Fatalf("GeneratePageTree: %v", err)
	}

	path := filepath.Join(contentDir, "pages", "empty.md")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("expected file %s to exist", path)
	}

	content := readFile(t, path)
	if !strings.HasPrefix(content, "---\n") {
		t.Errorf("page should start with frontmatter:\n%s", content)
	}
	if !strings.Contains(content, "type: page") {
		t.Errorf("page missing type: page:\n%s", content)
	}
}

func TestGeneratePage_NoDescription_OmittedFromFrontmatter(t *testing.T) {
	contentDir := t.TempDir()
	pagesDir := filepath.Join(contentDir, "pages")
	if err := os.MkdirAll(pagesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	page := &parser.Page{
		Title: "Simple Page",
		Body:  "Just content.",
		Path:  "/pages/simple/",
	}

	if err := generator.GeneratePage(page, contentDir); err != nil {
		t.Fatalf("GeneratePage: %v", err)
	}

	content := readFile(t, filepath.Join(contentDir, "pages", "simple.md"))
	if strings.Contains(content, "description:") {
		t.Errorf("empty description should be omitted from frontmatter:\n%s", content)
	}
}
