// Package generator transforms parsed Vale rules into Hugo content files and
// project structure (hugo.toml, content pages, data files, indexes).
package generator

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"go.yaml.in/yaml/v3"

	"github.com/armstrongl/rulebound/internal/parser"
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
		// Determine whether to insert a space before runes[i].
		if unicode.IsUpper(cur) {
			if unicode.IsLower(prev) {
				// for example, "g" → "P" in "HeadingPunctuation"
				b.WriteRune(' ')
			} else if unicode.IsUpper(prev) {
				// Consecutive uppercase: only split if the *next* char is lowercase,
				// meaning cur starts a new titlecase word (for example, "RL" in "URLFormat"
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
// The description consists of up to three parts:
//  1. A behavioral opening sentence using a verb derived from the rule type.
//  2. Salvaged static sentences from the rule message (sentences containing
//     format verbs like %s are dropped since Hugo renders those separately).
//  3. A swap-map sampler showing up to 2 concrete examples for substitution rules.
//
// Token-list and link sentences are omitted because Hugo's rule-details.html
// partial and single.html header already render that data.
func AutoDescription(rule *parser.ValeRule) string {
	var parts []string

	// R1: Type-aware behavioral verb opening.
	displayName := DisplayName(rule.Name)
	verb := ruleVerb(rule.Extends)
	scope := rule.Scope
	if scope == "" {
		scope = "text"
	}
	parts = append(parts, fmt.Sprintf("%s %s %s.", displayName, verb, scope))

	// R2: Conditional message inclusion (salvage clean sentences).
	if rule.Message != "" {
		if salvaged := salvageMessage(rule.Message); salvaged != "" {
			parts = append(parts, salvaged)
		}
	}

	// R3: Swap-map sampler (replaces old count-only sentence).
	if len(rule.Swap) > 0 {
		if sample := swapSampler(rule.Swap); sample != "" {
			parts = append(parts, sample)
		}
	}

	// R4: Token list and link sentences are removed entirely.
	// Tokens are rendered by rule-details.html.
	// Links are rendered as "Style guide reference" in single.html header.

	return strings.Join(parts, " ")
}

// ruleVerb returns the behavioral verb phrase for a given extends type.
func ruleVerb(extends string) string {
	switch extends {
	case "existence":
		return "flags"
	case "substitution":
		return "suggests preferred alternatives for"
	case "occurrence":
		return "limits"
	case "repetition":
		return "limits repetition of"
	case "consistency":
		return "enforces consistent usage of"
	case "conditional":
		return "checks that"
	case "capitalization":
		return "enforces capitalization of"
	case "metric":
		return "evaluates readability of"
	case "script":
		return "applies a custom check to"
	case "spelling":
		return "checks spelling of"
	case "sequence":
		return "detects patterns in"
	default:
		return "checks"
	}
}

// salvageMessage splits a message on sentence boundaries and returns only
// sentences that do not contain format verbs (%s, %[N]s). Returns empty
// string if all sentences contain format verbs or if the input is empty.
func salvageMessage(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return ""
	}

	// Split on sentence boundaries.
	sentences := strings.Split(msg, ". ")

	var kept []string
	for _, s := range sentences {
		if !strings.Contains(s, "%s") && !strings.Contains(s, "%[") {
			kept = append(kept, s)
		}
	}

	if len(kept) == 0 {
		return ""
	}

	result := strings.Join(kept, ". ")
	// Ensure terminal period.
	result = strings.TrimRight(result, ".")
	if result != "" {
		result += "."
	}
	return result
}

// swapSampler returns a sentence with up to 2 concrete swap examples plus
// total count. Filters out regex-heavy keys (containing metacharacters).
// Returns empty string for empty maps.
func swapSampler(swap map[string]string) string {
	total := len(swap)
	if total == 0 {
		return ""
	}

	// Sort keys alphabetically for deterministic output.
	keys := make([]string, 0, total)
	for k := range swap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Collect up to 2 non-regex examples.
	type example struct {
		from string
		to   string
	}
	var examples []example
	for _, k := range keys {
		if len(examples) >= 2 {
			break
		}
		if strings.ContainsAny(k, "()[]{}+*?\\|") {
			continue
		}
		examples = append(examples, example{from: k, to: swap[k]})
	}

	switch {
	case len(examples) == 0:
		// All keys are regex patterns; fall back to count-only.
		return fmt.Sprintf("This rule suggests replacements for %d terms.", total)
	case total == 1:
		// Single-entry swap map: compact format without total count.
		return fmt.Sprintf("Suggests using '%s' instead of '%s'.", examples[0].to, examples[0].from)
	default:
		// 1-2 examples with total count.
		var pairs []string
		for _, ex := range examples {
			pairs = append(pairs, fmt.Sprintf("'%s' instead of '%s'", ex.to, ex.from))
		}
		return fmt.Sprintf("For example, use %s (%d substitutions total).", strings.Join(pairs, ", "), total)
	}
}

// frontmatterData is the ordered representation written to Hugo frontmatter.
// yaml.Marshal handles special characters (backslashes, colons, brackets)
// correctly, avoiding ad-hoc string templates.
type frontmatterData struct {
	Title      string            `yaml:"title"`
	Extends    string            `yaml:"extends"`
	Level      string            `yaml:"level"`
	Message    string            `yaml:"message"`
	Link       string            `yaml:"link,omitempty"`
	Scope      string            `yaml:"scope,omitempty"`
	Ignorecase bool              `yaml:"ignorecase,omitempty"`
	Nonword    bool              `yaml:"nonword,omitempty"`
	Raw        []string          `yaml:"raw,omitempty"`
	Tokens     []string          `yaml:"tokens,omitempty"`
	Swap       map[string]string `yaml:"swap,omitempty"`
	// Action fields (flattened from Action struct)
	ActionName   string   `yaml:"action_name,omitempty"`
	ActionParams []string `yaml:"action_params,omitempty"`
	// Type-specific fields
	First        string            `yaml:"first,omitempty"`
	Second       string            `yaml:"second,omitempty"`
	Exceptions   []string          `yaml:"exceptions,omitempty"`
	Match        string            `yaml:"match,omitempty"`
	Indicators   []string          `yaml:"indicators,omitempty"`
	Max          int               `yaml:"max,omitempty"`
	Min          int               `yaml:"min,omitempty"`
	Token        string            `yaml:"token,omitempty"`
	Alpha        bool              `yaml:"alpha,omitempty"`
	Formula      string            `yaml:"formula,omitempty"`
	Condition    string            `yaml:"condition,omitempty"`
	Pattern      string            `yaml:"pattern,omitempty"`
	Script       string            `yaml:"script,omitempty"`
	Either       map[string]string `yaml:"either,omitempty"`
	Vocab        bool              `yaml:"vocab,omitempty"`
	Dictionaries []string          `yaml:"dictionaries,omitempty"`
	Custom       bool              `yaml:"custom,omitempty"`
	Filters      []string          `yaml:"filters,omitempty"`
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
		Title:        title,
		Extends:      rule.Extends,
		Level:        rule.Level,
		Message:      rule.Message,
		Link:         rule.Link,
		Scope:        rule.Scope,
		Ignorecase:   rule.Ignorecase,
		Nonword:      rule.Nonword,
		Raw:          rule.Raw,
		Tokens:       rule.Tokens,
		Swap:         rule.Swap,
		First:        rule.First,
		Second:       rule.Second,
		Exceptions:   rule.Exceptions,
		Match:        rule.Match,
		Indicators:   rule.Indicators,
		Max:          rule.Max,
		Min:          rule.Min,
		Token:        rule.Token,
		Alpha:        rule.Alpha,
		Formula:      rule.Formula,
		Condition:    rule.Condition,
		Pattern:      rule.Pattern,
		Script:       rule.Script,
		Either:       rule.Either,
		Vocab:        rule.Vocab,
		Dictionaries: rule.Dictionaries,
		Custom:       rule.Custom,
		Filters:      rule.Filters,
		Categories:   categories,
		Ruletypes:    []string{rule.Extends},
		Severities:   []string{rule.Level},
	}

	if rule.Action != nil {
		data.ActionName = rule.Action.Name
		data.ActionParams = rule.Action.Params
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
