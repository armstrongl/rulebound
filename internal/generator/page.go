package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/armstrongl/rulebound/internal/parser"
)

// pageFrontmatter is the YAML frontmatter for a content page.
type pageFrontmatter struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description,omitempty"`
	Type        string `yaml:"type"`
	Pagefind    *bool  `yaml:"pagefind,omitempty"`
}

// GeneratePage writes a single Hugo content .md file for the given page.
// The file path is derived from page.Path relative to contentDir.
// For a page with Path "/pages/language/active-voice/", the file is written to
// contentDir/pages/language/active-voice.md.
func GeneratePage(page *parser.Page, contentDir string) error {
	fm := pageFrontmatter{
		Title:       page.Title,
		Description: page.Description,
		Type:        "page",
	}
	if page.Hidden {
		f := false
		fm.Pagefind = &f
	}

	out, err := yaml.Marshal(fm)
	if err != nil {
		return fmt.Errorf("marshaling page frontmatter for %s: %w", page.Path, err)
	}

	content := "---\n" + string(out) + "---\n"
	if page.Body != "" {
		content += "\n" + page.Body + "\n"
	}

	// Derive filesystem path from the page's URL path.
	// "/pages/language/active-voice/" → "pages/language/active-voice.md"
	relPath := strings.TrimPrefix(page.Path, "/")
	relPath = strings.TrimSuffix(relPath, "/")
	filePath := filepath.Join(contentDir, relPath+".md")

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing page %s: %w", filePath, err)
	}
	return nil
}

// sectionIndexFrontmatter is the minimal frontmatter for auto-generated _index.md files.
type sectionIndexFrontmatter struct {
	Title string `yaml:"title"`
	Type  string `yaml:"type"`
}

// GeneratePageTree recursively generates all page content files and _index.md
// files for a SectionTree. It creates directories as needed.
func GeneratePageTree(tree *parser.SectionTree, contentDir string) error {
	return generatePageTreeRecursive(tree, contentDir)
}

func generatePageTreeRecursive(tree *parser.SectionTree, contentDir string) error {
	// Compute directory path from the tree's URL path.
	// "/pages/" → "pages", "/pages/language/" → "pages/language"
	relPath := strings.TrimPrefix(tree.Path, "/")
	relPath = strings.TrimSuffix(relPath, "/")
	dirPath := filepath.Join(contentDir, relPath)

	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dirPath, err)
	}

	// Generate _index.md for this section.
	if err := generateSectionIndex(tree, dirPath); err != nil {
		return err
	}

	// Generate each page in this directory.
	for _, page := range tree.Pages {
		if err := GeneratePage(page, contentDir); err != nil {
			return err
		}
	}

	// Recurse into children.
	for _, child := range tree.Children {
		if err := generatePageTreeRecursive(child, contentDir); err != nil {
			return err
		}
	}

	return nil
}

// generateSectionIndex writes a _index.md file for the given section tree node.
// If the tree has an author-provided IndexPage, the author's content is used
// with type: page injected. Otherwise, a minimal _index.md is auto-generated.
func generateSectionIndex(tree *parser.SectionTree, dirPath string) error {
	indexPath := filepath.Join(dirPath, "_index.md")

	if tree.IndexPage != nil {
		// Author-provided _index.md: preserve their content, inject type: page.
		fm := pageFrontmatter{
			Title:       tree.IndexPage.Title,
			Description: tree.IndexPage.Description,
			Type:        "page",
		}
		if tree.IndexPage.Hidden {
			f := false
			fm.Pagefind = &f
		}

		out, err := yaml.Marshal(fm)
		if err != nil {
			return fmt.Errorf("marshaling index page frontmatter for %s: %w", tree.Path, err)
		}

		content := "---\n" + string(out) + "---\n"
		if tree.IndexPage.Body != "" {
			content += "\n" + tree.IndexPage.Body + "\n"
		}

		if err := os.WriteFile(indexPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("writing _index.md for %s: %w", tree.Path, err)
		}
		return nil
	}

	// Auto-generate minimal _index.md.
	title := tree.Title
	if tree.Meta != nil && tree.Meta.Title != "" {
		title = tree.Meta.Title
	}

	data := sectionIndexFrontmatter{
		Title: title,
		Type:  "page",
	}

	out, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling auto _index.md for %s: %w", tree.Path, err)
	}

	content := "---\n" + string(out) + "---\n"
	if err := os.WriteFile(indexPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing _index.md for %s: %w", tree.Path, err)
	}
	return nil
}
