// Package mdgen converts structured Markdown files into Vale-compatible YAML
// rule files. It handles the .md → YAML direction; the parser package handles
// YAML → ValeRule for downstream content generation.
package mdgen

import (
	"fmt"
	"strings"

	"github.com/armstrongl/rulebound/internal/parser"
	"go.yaml.in/yaml/v3"
)

// Supported extends types for v1 generation.
var supportedTypes = map[string]bool{
	parser.ExtendsSubstitution:   true,
	parser.ExtendsExistence:      true,
	parser.ExtendsOccurrence:     true,
	parser.ExtendsCapitalization: true,
}

// SwapPair holds a single key-value entry from a vale-swap block,
// preserving the original file order.
type SwapPair struct {
	Key   string
	Value string
}

// Warning records a non-fatal issue encountered during parsing.
type Warning struct {
	Message string
}

// RuleSource is the intermediate representation of a structured Markdown file,
// ready for YAML emission.
type RuleSource struct {
	// Required fields.
	Extends string
	Message string
	Level   string

	// Pass-through scalar fields (scope, ignorecase, nonword, link, etc.).
	Fields map[string]interface{}

	// Type-specific data.
	Swap       []SwapPair // substitution
	Tokens     []string   // existence
	Exceptions []string   // capitalization

	// Occurrence scalars are in Fields (max, min, token).

	// Meta holds rulebound-only metadata stripped before emission.
	Meta map[string]interface{}
}

// ParseMarkdown parses a structured Markdown file into a RuleSource.
// It extracts YAML frontmatter and vale-* fenced code blocks.
func ParseMarkdown(data []byte) (*RuleSource, []Warning, error) {
	fmBytes, body, err := parser.ExtractFrontmatter(data)
	if err != nil {
		return nil, nil, fmt.Errorf("frontmatter: %w", err)
	}

	// Parse frontmatter into generic map.
	var fm map[string]interface{}
	if err := yaml.Unmarshal(fmBytes, &fm); err != nil {
		return nil, nil, fmt.Errorf("parsing frontmatter YAML: %w", err)
	}

	var warnings []Warning

	// Extract and validate required fields.
	extends, err := requireString(fm, "extends")
	if err != nil {
		return nil, nil, err
	}
	if !supportedTypes[extends] {
		supported := []string{
			parser.ExtendsSubstitution,
			parser.ExtendsExistence,
			parser.ExtendsOccurrence,
			parser.ExtendsCapitalization,
		}
		return nil, nil, fmt.Errorf("unsupported extends type %q (supported: %s)", extends, strings.Join(supported, ", "))
	}

	message, err := requireString(fm, "message")
	if err != nil {
		return nil, nil, err
	}

	// Level defaults to "warning" with advisory.
	level, _ := fm["level"].(string)
	if level == "" {
		level = "warning"
		warnings = append(warnings, Warning{Message: "level not specified, defaulting to 'warning'"})
	}

	// Strip meta block silently.
	var meta map[string]interface{}
	if m, ok := fm["meta"]; ok {
		if mmap, ok := m.(map[string]interface{}); ok {
			meta = mmap
		}
	}

	// Build pass-through fields (everything except extends, message, level, meta).
	fields := make(map[string]interface{})
	skipKeys := map[string]bool{"extends": true, "message": true, "level": true, "meta": true}
	for k, v := range fm {
		if skipKeys[k] {
			continue
		}
		if !parser.IsKnownField(k) {
			warnings = append(warnings, Warning{
				Message: fmt.Sprintf("unknown frontmatter field %q (not a known Vale field); passing through to YAML", k),
			})
		}
		fields[k] = v
	}

	// Parse fenced blocks from body.
	blocks, blockWarnings := parseFencedBlocks(body)
	warnings = append(warnings, blockWarnings...)

	src := &RuleSource{
		Extends: extends,
		Message: message,
		Level:   level,
		Fields:  fields,
		Meta:    meta,
	}

	// Type-specific extraction and validation.
	switch extends {
	case parser.ExtendsSubstitution:
		swapBlock, ok := blocks["vale-swap"]
		if !ok {
			return nil, nil, fmt.Errorf("substitution rule requires a ```vale-swap``` fenced block")
		}
		pairs, swapWarnings, err := parseSwapBlock(swapBlock)
		if err != nil {
			return nil, nil, fmt.Errorf("parsing vale-swap block: %w", err)
		}
		warnings = append(warnings, swapWarnings...)
		src.Swap = pairs

	case parser.ExtendsExistence:
		tokenBlock, ok := blocks["vale-tokens"]
		if !ok {
			return nil, nil, fmt.Errorf("existence rule requires a ```vale-tokens``` fenced block")
		}
		src.Tokens = parseLineBlock(tokenBlock)

	case parser.ExtendsOccurrence:
		// max, min, token come from frontmatter (already in Fields).
		hasMax := hasField(fm, "max")
		hasMin := hasField(fm, "min")
		hasToken := hasField(fm, "token")
		if !hasMax && !hasMin {
			return nil, nil, fmt.Errorf("occurrence rule requires at least one of 'max' or 'min' in frontmatter")
		}
		if !hasToken {
			return nil, nil, fmt.Errorf("occurrence rule requires 'token' in frontmatter")
		}

	case parser.ExtendsCapitalization:
		if !hasField(fm, "match") {
			return nil, nil, fmt.Errorf("capitalization rule requires 'match' in frontmatter")
		}
		if exBlock, ok := blocks["vale-exceptions"]; ok {
			src.Exceptions = parseLineBlock(exBlock)
		}
	}

	return src, warnings, nil
}

// requireString extracts a non-empty string field from the frontmatter map.
func requireString(fm map[string]interface{}, key string) (string, error) {
	v, ok := fm[key]
	if !ok {
		return "", fmt.Errorf("missing required field '%s' in frontmatter", key)
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return "", fmt.Errorf("missing required field '%s' in frontmatter (empty or non-string value)", key)
	}
	return s, nil
}

// hasField reports whether key exists in the frontmatter map.
func hasField(fm map[string]interface{}, key string) bool {
	_, ok := fm[key]
	return ok
}

// fencedBlock holds the raw content of a vale-* fenced block.
type fencedBlock struct {
	lang    string
	content string
}

// parseFencedBlocks extracts vale-* fenced code blocks from the Markdown body.
// It returns a map from language tag to block content, and warnings for
// duplicate blocks.
func parseFencedBlocks(body []byte) (map[string]fencedBlock, []Warning) {
	blocks := make(map[string]fencedBlock)
	var warnings []Warning

	lines := strings.Split(string(body), "\n")
	var inBlock bool
	var currentLang string
	var currentLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if inBlock {
			if trimmed == "```" {
				// Closing fence. Only store non-discarded blocks.
				if currentLang != "" {
					blocks[currentLang] = fencedBlock{
						lang:    currentLang,
						content: strings.Join(currentLines, "\n"),
					}
				}
				inBlock = false
				currentLang = ""
				currentLines = nil
				continue
			}
			currentLines = append(currentLines, line)
			continue
		}

		// Check for opening fence with vale-* language.
		if strings.HasPrefix(trimmed, "```vale-") {
			lang := strings.TrimPrefix(trimmed, "```")
			// Strip any trailing content after the language tag.
			if idx := strings.IndexByte(lang, ' '); idx != -1 {
				lang = lang[:idx]
			}

			if _, exists := blocks[lang]; exists {
				warnings = append(warnings, Warning{
					Message: fmt.Sprintf("duplicate %s block; only the first is used", lang),
				})
				// Skip this duplicate by consuming until closing fence.
				inBlock = true
				currentLang = "" // empty lang means we discard
				currentLines = nil
				continue
			}

			inBlock = true
			currentLang = lang
			currentLines = nil
		}
	}

	return blocks, warnings
}

// parseSwapBlock parses a vale-swap block's YAML content into ordered SwapPairs.
// It uses yaml.Node to preserve key insertion order.
func parseSwapBlock(block fencedBlock) ([]SwapPair, []Warning, error) {
	if strings.TrimSpace(block.content) == "" {
		return nil, nil, nil
	}

	var node yaml.Node
	if err := yaml.Unmarshal([]byte(block.content), &node); err != nil {
		return nil, nil, fmt.Errorf("invalid YAML in vale-swap block: %w", err)
	}

	// The document node wraps the actual mapping.
	if node.Kind != yaml.DocumentNode || len(node.Content) == 0 {
		return nil, nil, nil
	}

	mapping := node.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return nil, []Warning{{Message: "vale-swap block is not a YAML mapping; skipping"}}, nil
	}

	var pairs []SwapPair
	var warnings []Warning

	for i := 0; i+1 < len(mapping.Content); i += 2 {
		keyNode := mapping.Content[i]
		valNode := mapping.Content[i+1]

		if keyNode.Kind != yaml.ScalarNode || valNode.Kind != yaml.ScalarNode {
			warnings = append(warnings, Warning{
				Message: fmt.Sprintf("vale-swap: skipping non-scalar entry at line %d", keyNode.Line),
			})
			continue
		}

		pairs = append(pairs, SwapPair{
			Key:   keyNode.Value,
			Value: valNode.Value,
		})
	}

	return pairs, warnings, nil
}

// parseLineBlock splits a fenced block's content into non-empty lines.
func parseLineBlock(block fencedBlock) []string {
	var lines []string
	for _, line := range strings.Split(block.content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines
}
