package hugo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larah/rulebound/internal/config"
	"github.com/larah/rulebound/internal/parser"
)

// ── Scaffold ──────────────────────────────────────────────────────────────────

func TestScaffold_CreatesExpectedStructure(t *testing.T) {
	rules := []*parser.ValeRule{
		{Name: "Avoid", Extends: "existence", Level: "error", Message: "Don't use '%s'."},
		{Name: "Terms", Extends: "substitution", Level: "warning", Message: "Use '%s' instead."},
	}
	cfg := &config.Config{
		Title:   "Test Style Guide",
		BaseURL: "https://example.com/",
	}

	result, err := Scaffold(&parser.ParseResult{Rules: rules}, cfg)
	if err != nil {
		t.Fatalf("Scaffold: %v", err)
	}
	defer os.RemoveAll(result.TempDir)

	// Verify top-level directories exist
	dirs := []string{
		"content",
		"themes/rulebound",
		"data",
	}
	for _, dir := range dirs {
		path := filepath.Join(result.TempDir, dir)
		info, statErr := os.Stat(path)
		if statErr != nil {
			t.Errorf("expected directory %s to exist: %v", dir, statErr)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %s to be a directory", dir)
		}
	}
}

func TestScaffold_HugoTomlContainsTheme(t *testing.T) {
	rules := []*parser.ValeRule{
		{Name: "Avoid", Extends: "existence", Level: "error", Message: "Don't use '%s'."},
	}
	cfg := &config.Config{
		Title:   "Test Style Guide",
		BaseURL: "/",
	}

	result, err := Scaffold(&parser.ParseResult{Rules: rules}, cfg)
	if err != nil {
		t.Fatalf("Scaffold: %v", err)
	}
	defer os.RemoveAll(result.TempDir)

	hugoToml, readErr := os.ReadFile(filepath.Join(result.TempDir, "hugo.toml"))
	if readErr != nil {
		t.Fatalf("reading hugo.toml: %v", readErr)
	}
	content := string(hugoToml)

	if !strings.Contains(content, `theme = "rulebound"`) {
		t.Errorf("hugo.toml missing theme directive:\n%s", content)
	}

	// theme = "rulebound" should be at the top (first line)
	lines := strings.SplitN(content, "\n", 2)
	if !strings.Contains(lines[0], `theme = "rulebound"`) {
		t.Errorf("theme directive should be at the top of hugo.toml, first line is: %q", lines[0])
	}
}

func TestScaffold_ThemeExtracted(t *testing.T) {
	cfg := &config.Config{
		Title:   "Test Style Guide",
		BaseURL: "/",
	}

	result, err := Scaffold(&parser.ParseResult{}, cfg)
	if err != nil {
		t.Fatalf("Scaffold: %v", err)
	}
	defer os.RemoveAll(result.TempDir)

	// theme.toml should exist in the extracted theme
	themeToml := filepath.Join(result.ThemeDir, "theme.toml")
	info, statErr := os.Stat(themeToml)
	if statErr != nil {
		t.Fatalf("expected theme.toml at %s: %v", themeToml, statErr)
	}
	if info.Size() == 0 {
		t.Error("theme.toml should not be empty")
	}
}

func TestScaffold_ContentFilesGenerated(t *testing.T) {
	rules := []*parser.ValeRule{
		{Name: "Avoid", Extends: "existence", Level: "error", Message: "Don't use '%s'."},
		{Name: "OxfordComma", Extends: "existence", Level: "warning", Message: "Use the Oxford comma."},
	}
	cfg := &config.Config{
		Title:   "Test Style Guide",
		BaseURL: "/",
	}

	result, err := Scaffold(&parser.ParseResult{Rules: rules}, cfg)
	if err != nil {
		t.Fatalf("Scaffold: %v", err)
	}
	defer os.RemoveAll(result.TempDir)

	// content/rules/ should contain .md files for each rule
	rulesDir := filepath.Join(result.ContentDir, "rules")
	entries, readErr := os.ReadDir(rulesDir)
	if readErr != nil {
		t.Fatalf("reading content/rules/: %v", readErr)
	}

	// Expect at least _index.md + avoid.md + oxfordcomma.md = 3 files
	mdCount := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			mdCount++
		}
	}
	if mdCount < 3 {
		t.Errorf("expected at least 3 .md files in content/rules/, got %d", mdCount)
	}

	// Specifically check the rule files
	for _, name := range []string{"avoid.md", "oxfordcomma.md"} {
		path := filepath.Join(rulesDir, name)
		if _, statErr := os.Stat(path); statErr != nil {
			t.Errorf("expected %s in content/rules/: %v", name, statErr)
		}
	}
}

func TestScaffold_ReturnsTempDir(t *testing.T) {
	cfg := &config.Config{
		Title:   "Test Style Guide",
		BaseURL: "/",
	}

	result, err := Scaffold(&parser.ParseResult{}, cfg)
	if err != nil {
		t.Fatalf("Scaffold: %v", err)
	}
	defer os.RemoveAll(result.TempDir)

	if result.TempDir == "" {
		t.Error("ScaffoldResult.TempDir should not be empty")
	}
	if result.ThemeDir == "" {
		t.Error("ScaffoldResult.ThemeDir should not be empty")
	}
	if result.ContentDir == "" {
		t.Error("ScaffoldResult.ContentDir should not be empty")
	}
	if result.DataDir == "" {
		t.Error("ScaffoldResult.DataDir should not be empty")
	}

	// TempDir should exist on disk
	if _, statErr := os.Stat(result.TempDir); statErr != nil {
		t.Errorf("TempDir %s should exist: %v", result.TempDir, statErr)
	}
}

func TestScaffold_DataDirContainsSiteJSON(t *testing.T) {
	rules := []*parser.ValeRule{
		{Name: "Avoid", Extends: "existence", Level: "error", Message: "Don't use '%s'."},
	}
	cfg := &config.Config{
		Title:   "Test Style Guide",
		BaseURL: "/",
	}

	result, err := Scaffold(&parser.ParseResult{Rules: rules}, cfg)
	if err != nil {
		t.Fatalf("Scaffold: %v", err)
	}
	defer os.RemoveAll(result.TempDir)

	siteJSON := filepath.Join(result.DataDir, "site.json")
	info, statErr := os.Stat(siteJSON)
	if statErr != nil {
		t.Fatalf("expected data/site.json: %v", statErr)
	}
	if info.Size() == 0 {
		t.Error("data/site.json should not be empty")
	}
}

// ── RunPagefind ───────────────────────────────────────────────────────────────

func TestRunPagefind_NotOnPath(t *testing.T) {
	// Override PATH to ensure pagefind is not found
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", t.TempDir())
	defer os.Setenv("PATH", origPath)

	found, err := RunPagefind(t.TempDir())
	if err != nil {
		t.Fatalf("RunPagefind should return nil error when pagefind not found, got: %v", err)
	}
	if found {
		t.Error("RunPagefind should return found=false when pagefind is not on PATH")
	}
}
