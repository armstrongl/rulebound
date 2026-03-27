package hugo

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestThemeFS_CriticalFiles verifies that the embedded filesystem includes
// files under _default/ — the all: prefix is required because Go's embed
// silently excludes underscore-prefixed directories by default.
func TestThemeFS_CriticalFiles(t *testing.T) {
	criticalFiles := []string{
		"theme/layouts/_default/baseof.html",
		"theme/layouts/_default/single.html",
		"theme/layouts/_default/list.html",
	}

	for _, path := range criticalFiles {
		t.Run(path, func(t *testing.T) {
			data, err := themeFS.ReadFile(path)
			if err != nil {
				t.Fatalf("expected embedded file %s to exist, got error: %v", path, err)
			}
			if len(data) == 0 {
				t.Fatalf("embedded file %s is empty", path)
			}
		})
	}
}

// TestThemeFS_ContainsThemeToml verifies the theme metadata file is embedded.
func TestThemeFS_ContainsThemeToml(t *testing.T) {
	data, err := themeFS.ReadFile("theme/theme.toml")
	if err != nil {
		t.Fatalf("expected theme/theme.toml in embedded FS: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("theme/theme.toml is empty")
	}
}

// TestThemeFS_ContainsPartials verifies partials are embedded.
func TestThemeFS_ContainsPartials(t *testing.T) {
	partials := []string{
		"theme/layouts/partials/head.html",
		"theme/layouts/partials/sidebar.html",
		"theme/layouts/partials/search.html",
		"theme/layouts/partials/severity-badge.html",
		"theme/layouts/partials/rule-details.html",
	}

	for _, path := range partials {
		t.Run(path, func(t *testing.T) {
			_, err := themeFS.ReadFile(path)
			if err != nil {
				t.Fatalf("expected embedded file %s: %v", path, err)
			}
		})
	}
}

// TestThemeFS_ContainsStylesheet verifies CSS is embedded.
func TestThemeFS_ContainsStylesheet(t *testing.T) {
	_, err := themeFS.ReadFile("theme/static/css/style.css")
	if err != nil {
		t.Fatalf("expected theme/static/css/style.css: %v", err)
	}
}

// TestExtractTheme verifies that ExtractTheme correctly extracts the embedded
// theme to a temporary directory, preserving the full directory tree.
func TestExtractTheme(t *testing.T) {
	destDir := t.TempDir()

	if err := ExtractTheme(destDir); err != nil {
		t.Fatalf("ExtractTheme failed: %v", err)
	}

	// Verify critical files exist on disk.
	expectedFiles := []string{
		"theme.toml",
		"layouts/_default/baseof.html",
		"layouts/_default/single.html",
		"layouts/_default/list.html",
		"layouts/index.html",
		"layouts/partials/head.html",
		"layouts/partials/sidebar.html",
		"static/css/style.css",
	}

	for _, rel := range expectedFiles {
		path := filepath.Join(destDir, rel)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("expected extracted file %s: %v", rel, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("extracted file %s is empty", rel)
		}
	}
}

// TestExtractTheme_PreservesUnderscore specifically tests that the _default
// directory is correctly extracted — the primary risk of missing all: prefix.
func TestExtractTheme_PreservesUnderscore(t *testing.T) {
	destDir := t.TempDir()

	if err := ExtractTheme(destDir); err != nil {
		t.Fatalf("ExtractTheme failed: %v", err)
	}

	defaultDir := filepath.Join(destDir, "layouts", "_default")
	entries, err := os.ReadDir(defaultDir)
	if err != nil {
		t.Fatalf("_default directory not found after extraction: %v", err)
	}

	if len(entries) < 3 {
		t.Fatalf("expected at least 3 files in _default/, got %d", len(entries))
	}
}

// TestSearchHTML_UsesModularUI verifies the search partial references the
// correct Pagefind Modular UI assets (not Component UI) and uses the JS API
// with a custom resultTemplate (not the <pagefind-results> web component).
func TestSearchHTML_UsesModularUI(t *testing.T) {
	data, err := themeFS.ReadFile("theme/layouts/partials/search.html")
	if err != nil {
		t.Fatalf("reading search.html: %v", err)
	}
	content := string(data)

	mustContain := []struct {
		substr string
		reason string
	}{
		{"pagefind-modular-ui.js", "must reference Modular UI JS asset"},
		{"pagefind-modular-ui.css", "must reference Modular UI CSS asset"},
		{"PagefindModularUI.Instance", "must initialize Modular UI instance"},
		{"resultTemplate", "must use custom result template function"},
		{"meta.severity", "must render severity badge from Pagefind meta"},
		{"meta.type", "must render type badge from Pagefind meta"},
		{"FilterPills", "must include filter pills component"},
	}
	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("search.html %s — missing %q", tc.reason, tc.substr)
		}
	}

	mustNotContain := []struct {
		substr string
		reason string
	}{
		{"pagefind-component-ui", "must not reference old Component UI filename"},
		{"<pagefind-results>", "must not use web component (conflicts with Modular UI)"},
		{"pagefind-ui.js", "must not reference standard PagefindUI"},
	}
	for _, tc := range mustNotContain {
		if strings.Contains(content, tc.substr) {
			t.Errorf("search.html %s — found %q", tc.reason, tc.substr)
		}
	}
}

// TestSearchHTML_NoModuleType verifies the Pagefind script is loaded as a
// regular script, not type="module". The Modular UI JS sets a global
// (window.PagefindModularUI) which module scope would prevent.
func TestSearchHTML_NoModuleType(t *testing.T) {
	data, err := themeFS.ReadFile("theme/layouts/partials/search.html")
	if err != nil {
		t.Fatalf("reading search.html: %v", err)
	}
	content := string(data)

	// Script tags must NOT have type="module" — check actual tags, not comments
	forbidden := []string{
		`<script type="module">`,
		`<script src="/pagefind/pagefind-modular-ui.js" type="module">`,
		`.js" type="module"`,
	}
	for _, s := range forbidden {
		if strings.Contains(content, s) {
			t.Errorf("search.html must not use type=\"module\" on script tags — "+
				"pagefind-modular-ui.js is a regular script that sets a global; found %q", s)
		}
	}
}

// TestSingleHTML_PagefindAttributes verifies the rule page template includes
// Pagefind weight, meta, and filter attributes for search relevance.
func TestSingleHTML_PagefindAttributes(t *testing.T) {
	data, err := themeFS.ReadFile("theme/layouts/_default/single.html")
	if err != nil {
		t.Fatalf("reading single.html: %v", err)
	}
	content := string(data)

	mustContain := []struct {
		substr string
		reason string
	}{
		{`data-pagefind-body`, "article must be indexed by Pagefind"},
		{`data-pagefind-meta="title"`, "title must be captured as Pagefind meta"},
		{`data-pagefind-meta="severity:`, "severity must be captured as Pagefind meta"},
		{`data-pagefind-weight="2"`, "message must have boosted weight"},
		{`data-pagefind-weight="0.5"`, "technical details must have reduced weight"},
		{`data-pagefind-filter="severity"`, "severity must be a Pagefind filter"},
		{`data-pagefind-filter="type"`, "type must be a Pagefind filter"},
		{`data-pagefind-filter="category"`, "category must be a Pagefind filter"},
		{`data-pagefind-filter="content_type:rule"`, "content_type filter for rule/guideline distinction"},
	}
	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("single.html %s — missing %q", tc.reason, tc.substr)
		}
	}
}

// TestThemeFS_FileCount verifies we have a reasonable number of files embedded.
func TestThemeFS_FileCount(t *testing.T) {
	count := 0
	err := fs.WalkDir(themeFS, "theme", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			count++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking embedded FS: %v", err)
	}

	// We know there are 11 theme files from Phase 4.
	if count < 10 {
		t.Fatalf("expected at least 10 embedded theme files, got %d", count)
	}
}
