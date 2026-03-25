// Package generator transforms parsed Vale rules into Hugo content files and
// project structure (hugo.toml, content pages, data files, indexes).
package generator

import (
	"fmt"
	"strings"
	"unicode"

	"go.yaml.in/yaml/v3"

	"github.com/larah/rulebound/internal/parser"
)

// DisplayName converts a CamelCase rule name into a human-readable display
// name by inserting spaces between word boundaries.
//
// Rules:
//   - A run of consecutive uppercase letters is treated as an acronym and kept
//     together, unless followed by a lowercase letter that starts a new word.
//   - A single uppercase letter that starts a new titlecase word gets a space
//     prepended (except at position 0).
//
// Examples:
//
//	HeadingPunctuation → "Heading Punctuation"
//	OxfordComma        → "Oxford Comma"
//	AMPM               → "AMPM"
//	URLFormat          → "URL Format"
//	GeneralURL         → "General URL"
func DisplayName(name string) string {
	runes := []rune(name)
	n := len(runes)
	if n == 0 {
		return name
	}

	var b strings.Builder
	for i := 0; i < n; i++ {
		cur := runes[i]
		if i == 0 {
			b.WriteRune(cur)
			continue
		}
		prev := runes[i-1]
		// Determine if we insert a space before runes[i].
		if unicode.IsUpper(cur) {
			if unicode.IsLower(prev) {
				// e.g. "g" → "P" in "HeadingPunctuation"
				b.WriteRune(' ')
			} else if unicode.IsUpper(prev) {
				// Consecutive uppercase: only split if the *next* char is lowercase,
				// meaning cur starts a new titlecase word (e.g. "RL" in "URLFormat"
				// → split before "F").
				if i+1 < n && unicode.IsLower(runes[i+1]) {
					b.WriteRune(' ')
				}
			}
		}
		b.WriteRune(cur)
	}
	return b.String()
}

// AutoDescription generates a human-readable description for a rule when no
// companion Markdown file is present.
//
// Format:
//
//	{DisplayName} is a {level} {extends} rule. {message without %s}.
//	[See the [{link_domain}]({link}) for details.]
//	[This rule flags the following patterns: {first 10 tokens}.]
//	[This rule suggests replacements for {count} terms.]
func AutoDescription(rule *parser.ValeRule) string {
	var parts []string

	displayName := DisplayName(rule.Name)
	msg := strings.ReplaceAll(rule.Message, "'%s'", "")
	msg = strings.ReplaceAll(msg, "%s", "")
	msg = strings.TrimSpace(msg)
	// Remove double spaces that may result from stripping %s
	for strings.Contains(msg, "  ") {
		msg = strings.ReplaceAll(msg, "  ", " ")
	}

	base := fmt.Sprintf("%s is a %s %s rule.", displayName, rule.Level, rule.Extends)
	if msg != "" && msg != "." {
		// Strip trailing period from base and append message.
		base = strings.TrimSuffix(base, ".") + " " + msg
		if !strings.HasSuffix(base, ".") {
			base += "."
		}
	}
	parts = append(parts, base)

	if rule.Link != "" {
		domain := linkDomain(rule.Link)
		parts = append(parts, fmt.Sprintf("See the [%s](%s) for details.", domain, rule.Link))
	}

	if len(rule.Tokens) > 0 {
		tokens := rule.Tokens
		suffix := ""
		if len(tokens) > 10 {
			tokens = tokens[:10]
			suffix = "..."
		}
		quoted := make([]string, len(tokens))
		for i, tok := range tokens {
			quoted[i] = fmt.Sprintf("`%s`", tok)
		}
		parts = append(parts, fmt.Sprintf("This rule flags the following patterns: %s%s.", strings.Join(quoted, ", "), suffix))
	}

	if len(rule.Swap) > 0 {
		parts = append(parts, fmt.Sprintf("This rule suggests replacements for %d terms.", len(rule.Swap)))
	}

	return strings.Join(parts, " ")
}

// linkDomain extracts the hostname from a URL string.
// Falls back to the full URL if parsing fails.
func linkDomain(link string) string {
	// Strip scheme
	s := link
	if idx := strings.Index(s, "://"); idx != -1 {
		s = s[idx+3:]
	}
	// Strip path
	if idx := strings.Index(s, "/"); idx != -1 {
		s = s[:idx]
	}
	return s
}

// frontmatterData is the ordered representation written to Hugo frontmatter.
// We use yaml.Marshal so special characters (backslashes, colons, brackets)
// are handled correctly by the YAML library rather than ad-hoc string templates.
type frontmatterData struct {
	Title      string            `yaml:"title"`
	Extends    string            `yaml:"extends"`
	Level      string            `yaml:"level"`
	Message    string            `yaml:"message"`
	Link       string            `yaml:"link,omitempty"`
	Scope      string            `yaml:"scope,omitempty"`
	Ignorecase bool              `yaml:"ignorecase,omitempty"`
	Nonword    bool              `yaml:"nonword,omitempty"`
	Tokens     []string          `yaml:"tokens,omitempty"`
	Swap       map[string]string `yaml:"swap,omitempty"`
	// Type-specific fields
	First      string            `yaml:"first,omitempty"`
	Second     string            `yaml:"second,omitempty"`
	Exceptions []string          `yaml:"exceptions,omitempty"`
	Match      string            `yaml:"match,omitempty"`
	Indicators []string          `yaml:"indicators,omitempty"`
	Max        int               `yaml:"max,omitempty"`
	Min        int               `yaml:"min,omitempty"`
	Token      string            `yaml:"token,omitempty"`
	Alpha      bool              `yaml:"alpha,omitempty"`
	Formula    string            `yaml:"formula,omitempty"`
	Condition  string            `yaml:"condition,omitempty"`
	Pattern    string            `yaml:"pattern,omitempty"`
	Script     string            `yaml:"script,omitempty"`
	Either     map[string]string `yaml:"either,omitempty"`
	Vocab      bool              `yaml:"vocab,omitempty"`
	// Taxonomy terms (Hugo lowercases all param keys automatically)
	Categories []string `yaml:"categories"`
	Ruletypes  []string `yaml:"ruletypes"`
	Severities []string `yaml:"severities"`
}

// BuildFrontmatter converts a ValeRule into a YAML frontmatter string (without
// the --- delimiters). yaml.Marshal handles all escaping of special characters.
func BuildFrontmatter(rule *parser.ValeRule) (string, error) {
	title := DisplayName(rule.Name)

	categories := categoriesFromRule(rule)

	data := frontmatterData{
		Title:      title,
		Extends:    rule.Extends,
		Level:      rule.Level,
		Message:    rule.Message,
		Link:       rule.Link,
		Scope:      rule.Scope,
		Ignorecase: rule.Ignorecase,
		Nonword:    rule.Nonword,
		Tokens:     rule.Tokens,
		Swap:       rule.Swap,
		First:      rule.First,
		Second:     rule.Second,
		Exceptions: rule.Exceptions,
		Match:      rule.Match,
		Indicators: rule.Indicators,
		Max:        rule.Max,
		Min:        rule.Min,
		Token:      rule.Token,
		Alpha:      rule.Alpha,
		Formula:    rule.Formula,
		Condition:  rule.Condition,
		Pattern:    rule.Pattern,
		Script:     rule.Script,
		Either:     rule.Either,
		Vocab:      rule.Vocab,
		Categories: categories,
		Ruletypes:  []string{rule.Extends},
		Severities: []string{rule.Level},
	}

	out, err := yaml.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshaling frontmatter for %s: %w", rule.Name, err)
	}
	return string(out), nil
}

// categoriesFromRule returns the category list for the rule.
// Rule.Category may be a comma-separated list of categories (when assigned from
// config), or a single value. If empty, falls back to rule.Extends.
func categoriesFromRule(rule *parser.ValeRule) []string {
	if rule.Category == "" {
		return []string{rule.Extends}
	}
	parts := strings.Split(rule.Category, ",")
	cats := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			cats = append(cats, p)
		}
	}
	if len(cats) == 0 {
		return []string{rule.Extends}
	}
	return cats
}
