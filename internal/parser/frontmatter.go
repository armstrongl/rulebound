package parser

import (
	"fmt"
	"strings"
)

// ExtractFrontmatter splits a Markdown document into its YAML frontmatter and
// body. It normalises CRLF to LF before processing. The returned frontmatter
// bytes do not include the --- delimiters. If no valid frontmatter is found,
// ExtractFrontmatter returns an error.
func ExtractFrontmatter(data []byte) (frontmatter []byte, body []byte, err error) {
	const fence = "---"

	content := strings.ReplaceAll(string(data), "\r\n", "\n")

	if !strings.HasPrefix(content, fence) {
		return nil, nil, fmt.Errorf("no frontmatter found")
	}

	rest := content[len(fence):]
	if len(rest) == 0 || rest[0] != '\n' {
		return nil, nil, fmt.Errorf("no frontmatter found")
	}
	rest = rest[1:] // consume newline after opening ---

	idx := strings.Index(rest, "\n"+fence)
	if idx == -1 {
		return nil, nil, fmt.Errorf("no closing frontmatter fence")
	}

	fmRaw := rest[:idx]
	bodyStr := rest[idx+1+len(fence):]

	// Consume optional trailing newline after the closing fence.
	if len(bodyStr) > 0 && bodyStr[0] == '\n' {
		bodyStr = bodyStr[1:]
	}

	return []byte(fmRaw), []byte(bodyStr), nil
}
