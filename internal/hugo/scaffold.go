package hugo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/armstrongl/rulebound/internal/config"
	"github.com/armstrongl/rulebound/internal/generator"
	"github.com/armstrongl/rulebound/internal/parser"
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
//	├── static/            (copied from package, if present)
//	└── themes/
//	    └── rulebound/     (extracted from embedded theme)
//
// If packageDir contains a static/ subdirectory, its contents are copied into
// the Hugo project's static/ directory after theme extraction, so package
// assets take precedence over theme defaults.
//
// The caller is responsible for cleaning up tempDir (see CleanupFunc).
func Scaffold(parseResult *parser.ParseResult, cfg *config.Config, packageDir string) (*ScaffoldResult, error) {
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

	// 2. Copy package static/ assets (if present) into the Hugo project.
	//    This runs after theme extraction so package assets take precedence.
	if err := copyPackageStatic(packageDir, tempDir); err != nil {
		return result, fmt.Errorf("copying package static assets: %w", err)
	}

	// 3. Generate content/, data/, and hugo.toml.
	//    GenerateSite writes hugo.toml, content/_index.md, content/rules/*.md,
	//    and data/site.json into the output directory.
	if err := generator.GenerateSite(parseResult, cfg, tempDir); err != nil {
		return result, fmt.Errorf("generating site content: %w", err)
	}

	// 4. Patch hugo.toml to add theme = "rulebound" as a top-level key.
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

// copyPackageStatic copies the contents of packageDir/static/ into
// destDir/static/, preserving directory structure. If the source directory
// does not exist, it returns nil (no-op).
func copyPackageStatic(packageDir, destDir string) error {
	if packageDir == "" {
		return nil
	}
	srcStatic := filepath.Join(packageDir, "static")
	if _, err := os.Stat(srcStatic); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("checking static directory: %w", err)
	}

	return filepath.WalkDir(srcStatic, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcStatic, path)
		if err != nil {
			return fmt.Errorf("computing relative path: %w", err)
		}
		dest := filepath.Join(destDir, "static", rel)

		if d.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", rel, err)
		}
		return os.WriteFile(dest, data, 0o644)
	})
}
