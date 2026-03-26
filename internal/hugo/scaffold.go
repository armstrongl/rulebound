package hugo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/larah/rulebound/internal/config"
	"github.com/larah/rulebound/internal/generator"
	"github.com/larah/rulebound/internal/parser"
)

// ScaffoldResult holds the paths that Scaffold creates.
type ScaffoldResult struct {
	// TempDir is the root of the scaffolded Hugo project.
	TempDir string
	// ThemeDir is tempDir/themes/rulebound/.
	ThemeDir string
	// ContentDir is tempDir/content/.
	ContentDir string
	// DataDir is tempDir/data/.
	DataDir string
}

// Scaffold creates a temporary Hugo project directory with this structure:
//
//	tempDir/
//	├── hugo.toml          (generated with theme = "rulebound")
//	├── content/
//	│   ├── _index.md
//	│   └── rules/
//	│       └── *.md
//	├── data/
//	│   └── site.json
//	└── themes/
//	    └── rulebound/     (extracted from embedded theme)
//
// The caller is responsible for cleaning up tempDir (see CleanupFunc).
func Scaffold(parseResult *parser.ParseResult, cfg *config.Config) (*ScaffoldResult, error) {
	tempDir, err := os.MkdirTemp("", "rulebound-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp directory: %w", err)
	}

	result := &ScaffoldResult{
		TempDir:    tempDir,
		ThemeDir:   filepath.Join(tempDir, "themes", "rulebound"),
		ContentDir: filepath.Join(tempDir, "content"),
		DataDir:    filepath.Join(tempDir, "data"),
	}

	// 1. Extract embedded theme to themes/rulebound/.
	if err := os.MkdirAll(result.ThemeDir, 0o755); err != nil {
		return result, fmt.Errorf("creating theme directory: %w", err)
	}
	if err := ExtractTheme(result.ThemeDir); err != nil {
		return result, fmt.Errorf("extracting theme: %w", err)
	}

	// 2. Generate content/, data/, and hugo.toml.
	//    GenerateSite writes hugo.toml, content/_index.md, content/rules/*.md,
	//    and data/site.json into the output directory.
	if err := generator.GenerateSite(parseResult, cfg, tempDir); err != nil {
		return result, fmt.Errorf("generating site content: %w", err)
	}

	// 3. Patch hugo.toml to add theme = "rulebound" as a top-level key.
	//    The generator writes hugo.toml without a theme directive, so we
	//    prepend it before any TOML table headers (for example, [taxonomies]).
	//    Appending would place it inside the last table, which is invalid.
	hugoTomlPath := filepath.Join(tempDir, "hugo.toml")
	existing, err := os.ReadFile(hugoTomlPath)
	if err != nil {
		return result, fmt.Errorf("reading hugo.toml for patching: %w", err)
	}
	// Prepend the theme line so it appears before any [section] header.
	content := string(existing)
	patched := "theme = \"rulebound\"\n" + content
	if err := os.WriteFile(hugoTomlPath, []byte(patched), 0o644); err != nil {
		return result, fmt.Errorf("writing patched hugo.toml: %w", err)
	}

	return result, nil
}
