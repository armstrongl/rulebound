// Package hugo embeds the Hugo theme, scaffolds temporary projects, and runs Hugo builds.
package hugo

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// themeFS embeds the entire theme/ directory from the project root.
// The all: prefix is required because Go's embed silently excludes files and
// directories starting with underscore (for example, _default/) without it.
//
//go:embed all:theme
var themeFS embed.FS

// ThemeFS returns the embedded filesystem for use in tests.
func ThemeFS() embed.FS {
	return themeFS
}

// ExtractTheme copies the embedded theme into destDir (typically
// tempDir/themes/rulebound/). It walks the embedded FS and recreates
// the directory tree with all files.
func ExtractTheme(destDir string) error {
	return fs.WalkDir(themeFS, "theme", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Strip the leading "theme/" prefix so that the extracted tree starts
		// at destDir directly (for example, destDir/hugo.toml, destDir/layouts/...).
		rel, err := filepath.Rel("theme", path)
		if err != nil {
			return fmt.Errorf("computing relative path for %s: %w", path, err)
		}
		if rel == "." {
			return nil
		}

		target := filepath.Join(destDir, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		data, err := themeFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading embedded file %s: %w", path, err)
		}

		if err := os.WriteFile(target, data, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", target, err)
		}
		return nil
	})
}
