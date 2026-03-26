package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// knownFields lists top-level YAML keys that map to ValeRule struct fields.
// The parser stores any key NOT in this set in ValeRule.Extra.
var knownFields = map[string]bool{
	"extends":      true,
	"message":      true,
	"level":        true,
	"link":         true,
	"scope":        true,
	"ignorecase":   true,
	"nonword":      true,
	"raw":          true,
	"action":       true,
	"tokens":       true,
	"swap":         true,
	"first":        true,
	"second":       true,
	"exceptions":   true,
	"match":        true,
	"indicators":   true,
	"max":          true,
	"min":          true,
	"token":        true,
	"alpha":        true,
	"formula":      true,
	"condition":    true,
	"pattern":      true,
	"script":       true,
	"either":       true,
	"vocab":        true,
	"dictionaries": true,
	"custom":       true,
	"filters":      true,
}

// rawRule is the intermediate YAML representation used during parsing.
// Several fields use custom unmarshalers to handle Vale's polymorphic YAML
// syntax (action, swap, scope, tokens).
type rawRule struct {
	Extends      string            `yaml:"extends"`
	Message      string            `yaml:"message"`
	Level        string            `yaml:"level"`
	Link         string            `yaml:"link"`
	Scope        flexibleScope     `yaml:"scope"`
	Ignorecase   bool              `yaml:"ignorecase"`
	Nonword      bool              `yaml:"nonword"`
	Raw          []string          `yaml:"raw"`
	Action       *rawAction        `yaml:"action"`
	Tokens       flexibleTokens    `yaml:"tokens"`
	Swap         flexibleSwap      `yaml:"swap"`
	First        string            `yaml:"first"`
	Second       string            `yaml:"second"`
	Exceptions   []string          `yaml:"exceptions"`
	Match        string            `yaml:"match"`
	Indicators   []string          `yaml:"indicators"`
	Max          int               `yaml:"max"`
	Min          int               `yaml:"min"`
	Token        string            `yaml:"token"`
	Alpha        bool              `yaml:"alpha"`
	Formula      string            `yaml:"formula"`
	Condition    string            `yaml:"condition"`
	Pattern      string            `yaml:"pattern"`
	Script       string            `yaml:"script"`
	Either       map[string]string `yaml:"either"`
	Vocab        bool              `yaml:"vocab"`
	Dictionaries []string          `yaml:"dictionaries"`
	Custom       bool              `yaml:"custom"`
	Filters      []string          `yaml:"filters"`
}

// flexibleSwap handles swap as either a map or a list of single-key maps:
//
//	swap: {old: new}           (mapping)
//	swap:
//	  - old: new               (sequence of mappings)
type flexibleSwap map[string]string

func (s *flexibleSwap) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.MappingNode:
		var m map[string]string
		if err := value.Decode(&m); err != nil {
			return err
		}
		*s = m
		return nil
	case yaml.SequenceNode:
		result := make(map[string]string)
		for _, item := range value.Content {
			if item.Kind != yaml.MappingNode {
				return fmt.Errorf("swap: expected mapping in sequence, got %v", item.Kind)
			}
			var m map[string]string
			if err := item.Decode(&m); err != nil {
				return err
			}
			for k, v := range m {
				result[k] = v
			}
		}
		*s = result
		return nil
	default:
		return fmt.Errorf("swap: unexpected YAML node kind %v", value.Kind)
	}
}

// flexibleScope handles scope as either a string or a list of strings:
//
//	scope: heading             (scalar)
//	scope:
//	  - list
//	  - heading                (sequence)
type flexibleScope string

func (s *flexibleScope) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		*s = flexibleScope(value.Value)
		return nil
	case yaml.SequenceNode:
		var items []string
		if err := value.Decode(&items); err != nil {
			return err
		}
		*s = flexibleScope(strings.Join(items, ", "))
		return nil
	default:
		return fmt.Errorf("scope: unexpected YAML node kind %v", value.Kind)
	}
}

// flexibleTokens handles tokens as either a list of strings or a list of
// objects (used by sequence-type rules with tag/pattern fields):
//
//	tokens:
//	  - 'pattern'              (sequence of scalars)
//	tokens:
//	  - tag: "VBN"
//	    pattern: ".+"          (sequence of mappings)
type flexibleTokens []string

func (t *flexibleTokens) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.SequenceNode {
		return fmt.Errorf("tokens: expected sequence, got %v", value.Kind)
	}

	var result []string
	for _, item := range value.Content {
		switch item.Kind {
		case yaml.ScalarNode:
			result = append(result, item.Value)
		case yaml.MappingNode:
			var obj map[string]string
			if err := item.Decode(&obj); err != nil {
				return err
			}
			tag := obj["tag"]
			pattern := obj["pattern"]
			if tag != "" && pattern != "" {
				result = append(result, tag+": "+pattern)
			} else if pattern != "" {
				result = append(result, pattern)
			} else if tag != "" {
				result = append(result, tag)
			}
		default:
			return fmt.Errorf("tokens: unexpected item kind %v", item.Kind)
		}
	}
	*t = result
	return nil
}

// rawAction handles the dual form of the action field:
//
//	action: replace            (string shorthand)
//	action: {name: replace, params: [URL, address]}  (object form)
type rawAction struct {
	Name   string
	Params []string
}

// UnmarshalYAML implements yaml.Unmarshaler for rawAction.
func (a *rawAction) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		// String shorthand: action: replace
		a.Name = value.Value
		return nil
	case yaml.MappingNode:
		// Object form: {name: replace, params: [...]}
		type actionObj struct {
			Name   string   `yaml:"name"`
			Params []string `yaml:"params"`
		}
		var obj actionObj
		if err := value.Decode(&obj); err != nil {
			return err
		}
		a.Name = obj.Name
		a.Params = obj.Params
		return nil
	default:
		return fmt.Errorf("action: unexpected YAML node kind %v", value.Kind)
	}
}

// ParseRule reads a single Vale rule .yml (or .yaml) file and returns a
// populated *ValeRule. It reads the companion .md file (same basename)
// automatically when present.
//
// ParseRule returns an error for:
//   - file read errors
//   - malformed YAML
//   - empty YAML (nil document)
//   - missing extends field
func ParseRule(filePath string) (*ValeRule, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filePath, err)
	}

	// ── First pass: decode into typed struct ─────────────────────────────
	var rr rawRule
	if err := yaml.Unmarshal(data, &rr); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", filePath, err)
	}

	// Treat empty/nil documents as an error.
	if rr.Extends == "" && rr.Message == "" && rr.Level == "" {
		// Confirm the file is genuinely empty or whitespace-only.
		if len(strings.TrimSpace(string(data))) == 0 {
			return nil, fmt.Errorf("parsing %s: empty file", filePath)
		}
		// Non-empty YAML but no extends — handled below.
	}

	if rr.Extends == "" {
		return nil, fmt.Errorf("parsing %s: missing required field 'extends'", filePath)
	}

	// ── Second pass: collect unknown fields into Extra ────────────────────
	var rawMap map[string]interface{}
	if err := yaml.Unmarshal(data, &rawMap); err != nil {
		// This cannot happen if the first pass succeeded, but guard anyway.
		return nil, fmt.Errorf("parsing extras in %s: %w", filePath, err)
	}

	var extra map[string]interface{}
	for k, v := range rawMap {
		if !knownFields[k] {
			if extra == nil {
				extra = make(map[string]interface{})
			}
			extra[k] = v
		}
	}

	// ── Assemble ValeRule ─────────────────────────────────────────────────
	rule := &ValeRule{
		Name:         nameFromPath(filePath),
		Extends:      rr.Extends,
		Message:      rr.Message,
		Level:        rr.Level,
		Link:         rr.Link,
		Scope:        string(rr.Scope),
		Ignorecase:   rr.Ignorecase,
		Nonword:      rr.Nonword,
		Raw:          rr.Raw,
		Tokens:       []string(rr.Tokens),
		Swap:         map[string]string(rr.Swap),
		First:        rr.First,
		Second:       rr.Second,
		Exceptions:   rr.Exceptions,
		Match:        rr.Match,
		Indicators:   rr.Indicators,
		Max:          rr.Max,
		Min:          rr.Min,
		Token:        rr.Token,
		Alpha:        rr.Alpha,
		Formula:      rr.Formula,
		Condition:    rr.Condition,
		Pattern:      rr.Pattern,
		Script:       rr.Script,
		Either:       rr.Either,
		Vocab:        rr.Vocab,
		Dictionaries: rr.Dictionaries,
		Custom:       rr.Custom,
		Filters:      rr.Filters,
		SourceFile:   filePath,
		Extra:        extra,
	}

	if rr.Action != nil {
		rule.Action = &Action{
			Name:   rr.Action.Name,
			Params: rr.Action.Params,
		}
	}

	// ── Companion Markdown ────────────────────────────────────────────────
	md, err := readCompanion(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading companion for %s: %w", filePath, err)
	}
	rule.CompanionMD = md

	return rule, nil
}

// ParseWarning records a non-fatal issue that ParsePackage encounters during parsing.
type ParseWarning struct {
	File    string // The file that caused the warning
	Message string // Human-readable description
}

// ParsePackage scans dir for .yml and .yaml files, parses each as a Vale rule,
// and returns a *ParseResult with rules sorted by name, any parsed guidelines,
// and warnings. It skips files that fail to parse (malformed YAML, missing
// extends, and so on) and reports them as warnings. It ignores non-YAML files
// (for example, meta.json).
//
// ParsePackage returns an error only if dir cannot be read.
func ParsePackage(dir string) (*ParseResult, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}

	packageName := filepath.Base(dir)

	var rules []*ValeRule
	var warnings []ParseWarning
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".yml" && ext != ".yaml" {
			continue
		}

		filePath := filepath.Join(dir, name)
		rule, err := ParseRule(filePath)
		if err != nil {
			warnings = append(warnings, ParseWarning{
				File:    name,
				Message: err.Error(),
			})
			continue
		}

		rule.Category = packageName
		rules = append(rules, rule)
	}

	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Name < rules[j].Name
	})

	// Parse guidelines from guidelines/ subdirectory.
	guidelines, guidelineWarnings, err := parseGuidelines(dir)
	if err != nil {
		return nil, fmt.Errorf("parsing guidelines: %w", err)
	}
	warnings = append(warnings, guidelineWarnings...)

	return &ParseResult{
		Rules:      rules,
		Guidelines: guidelines,
		Warnings:   warnings,
	}, nil
}

// nameFromPath derives the rule name from a file path by stripping the
// directory and extension (for example, "/styles/Microsoft/Avoid.yml" becomes "Avoid").
func nameFromPath(filePath string) string {
	base := filepath.Base(filePath)
	ext := filepath.Ext(base)
	return base[:len(base)-len(ext)]
}
