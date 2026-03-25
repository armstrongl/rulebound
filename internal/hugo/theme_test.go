package hugo

import (
	"io/fs"
	"os"
	"path/filepath"
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
