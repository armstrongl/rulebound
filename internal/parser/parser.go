package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// knownFields is the set of top-level YAML keys that map to ValeRule struct
// fields. Any key NOT in this set is stored in ValeRule.Extra.
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
// The Action field uses a custom unmarshaller to handle both string and
// object forms.
type rawRule struct {
	Extends      string            `yaml:"extends"`
	Message      string            `yaml:"message"`
	Level        string            `yaml:"level"`
	Link         string            `yaml:"link"`
	Scope        string            `yaml:"scope"`
	Ignorecase   bool              `yaml:"ignorecase"`
	Nonword      bool              `yaml:"nonword"`
	Raw          []string          `yaml:"raw"`
	Action       *rawAction        `yaml:"action"`
	Tokens       []string          `yaml:"tokens"`
	Swap         map[string]string `yaml:"swap"`
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
// populated *ValeRule. The companion .md file (same basename) is read
// automatically when present.
//
// Errors are returned for:
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
		// Should not happen if the first pass succeeded, but guard anyway.
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
		Scope:        rr.Scope,
		Ignorecase:   rr.Ignorecase,
		Nonword:      rr.Nonword,
		Raw:          rr.Raw,
		Tokens:       rr.Tokens,
		Swap:         rr.Swap,
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

// ParsePackage scans dir for .yml and .yaml files, parses each as a Vale rule,
// and returns the successfully parsed rules sorted by name. Files that fail to
// parse (malformed YAML, missing extends, etc.) are silently skipped.
// Non-YAML files (e.g., meta.json) are ignored.
//
// Returns an error only if dir cannot be read.
func ParsePackage(dir string) ([]*ValeRule, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}

	packageName := filepath.Base(dir)

	var rules []*ValeRule
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
			// Skip files that fail to parse.
			continue
		}

		rule.Category = packageName
		rules = append(rules, rule)
	}

	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Name < rules[j].Name
	})

	return rules, nil
}

// nameFromPath derives the rule name from a file path by stripping the
// directory and extension (e.g., "/styles/Microsoft/Avoid.yml" → "Avoid").
func nameFromPath(filePath string) string {
	base := filepath.Base(filePath)
	ext := filepath.Ext(base)
	return base[:len(base)-len(ext)]
}
