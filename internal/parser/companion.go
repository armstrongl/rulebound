package parser

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// readCompanion looks for a .md file with the same basename as ymlPath.
// If found, it reads the file, strips any YAML frontmatter (--- delimited),
// and returns the body content trimmed of leading/trailing whitespace.
// If no companion file exists, it returns ("", nil).
func readCompanion(ymlPath string) (string, error) {
	ext := filepath.Ext(ymlPath)
	mdPath := ymlPath[:len(ymlPath)-len(ext)] + ".md"

	data, err := os.ReadFile(mdPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}

	body := stripFrontmatter(string(data))
	return strings.TrimSpace(body), nil
}

// stripFrontmatter removes a leading YAML frontmatter block (--- ... ---) from
// Markdown content. Hugo shortcodes are not interpreted; the raw Markdown body
// is returned as-is. If no frontmatter is present, the original content is
// returned unchanged.
func stripFrontmatter(content string) string {
	const fence = "---"

	// Normalise line endings for reliable splitting.
	content = strings.ReplaceAll(content, "\r\n", "\n")

	if !strings.HasPrefix(content, fence) {
		return content
	}

	// The fence must be followed immediately by a newline (or EOF).
	rest := content[len(fence):]
	if len(rest) == 0 || rest[0] != '\n' {
		return content
	}
	rest = rest[1:] // consume the newline after the opening ---

	// Find the closing ---
	idx := strings.Index(rest, "\n"+fence)
	if idx == -1 {
		// No closing fence — treat the whole content as body.
		return content
	}

	// Body starts after the closing fence line.
	body := rest[idx+1+len(fence):]
	// Consume optional trailing newline after the closing fence.
	if len(body) > 0 && body[0] == '\n' {
		body = body[1:]
	}
	return body
}
