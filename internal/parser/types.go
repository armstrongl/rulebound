// Package parser reads Vale rule YAML files and their companion Markdown files,
// producing a structured [ValeRule] for downstream content generation.
package parser

// Extends constants for Vale rule types.
const (
	ExtendsExistence      = "existence"
	ExtendsSubstitution   = "substitution"
	ExtendsOccurrence     = "occurrence"
	ExtendsRepetition     = "repetition"
	ExtendsConsistency    = "consistency"
	ExtendsConditional    = "conditional"
	ExtendsCapitalization = "capitalization"
	ExtendsMetric         = "metric"
	ExtendsScript         = "script"
	ExtendsSpelling       = "spelling"
	ExtendsSequence       = "sequence"
)

// ValeRule is the unified in-memory representation of a parsed Vale rule file.
// ParseRule populates fields according to the rule's extends type; unused
// fields remain at their zero values.
type ValeRule struct {
	// Name is the rule name derived from the filename without extension (for example, "Avoid").
	Name string

	// Extends identifies the rule type: one of the Extends* constants or an unrecognized string.
	Extends string

	// Message is the human-readable message template shown to writers.
	Message string

	// Level is the severity: "error", "warning", or "suggestion".
	Level string

	// Link is an optional URL to the source style guide.
	Link string

	// Scope limits the rule to a document scope (sentence, heading, paragraph, and so on).
	Scope string

	// Ignorecase controls case-insensitive token matching.
	Ignorecase bool

	// Nonword changes token matching to non-word boundaries.
	Nonword bool

	// Raw holds regex patterns used alongside tokens (existence rules).
	Raw []string

	// Action is an optional action attached to the rule (for example, replace, remove).
	Action *Action

	// ── Type-specific fields ────────────────────────────────────────────────

	// Tokens are the word/pattern list for existence rules.
	Tokens []string

	// Swap maps incorrect patterns to their preferred replacements (substitution).
	Swap map[string]string

	// First is the antecedent regex pattern (conditional).
	First string

	// Second is the consequent regex pattern (conditional).
	Second string

	// Exceptions lists allowed exclusions (conditional, capitalization).
	Exceptions []string

	// Match is the capitalization style token (for example, "$sentence").
	Match string

	// Indicators lists punctuation that resets capitalization tracking.
	Indicators []string

	// Max is the maximum allowed occurrences/words (occurrence, repetition).
	Max int

	// Min is the minimum required occurrences (occurrence).
	Min int

	// Token is the regex used to count tokens (occurrence).
	Token string

	// Alpha restricts repetition checks to alphabetic tokens.
	Alpha bool

	// Formula is an arithmetic expression to evaluate (metric).
	Formula string

	// Condition is the comparison expression for metric rules (for example, "> 10").
	Condition string

	// Pattern is the structural pattern for sequence rules.
	Pattern string

	// Script is an embedded Tengo/Lua script body.
	Script string

	// Either maps key/value alternatives for consistency rules.
	Either map[string]string

	// Vocab enables project vocabulary for spelling rules.
	Vocab bool

	// Dictionaries lists custom dictionary paths for spelling rules.
	Dictionaries []string

	// Custom flags a custom dictionary for spelling rules.
	Custom bool

	// Filters lists regex patterns to exclude from spelling checks.
	Filters []string

	// ── Rulebound metadata ─────────────────────────────────────────────────

	// CompanionMD holds the body content of the companion .md file, with
	// frontmatter stripped. Empty if no companion file exists.
	CompanionMD string

	// Category comes from the rulebound.yml categories configuration
	// or defaults to the package directory name.
	Category string

	// SourceFile is the absolute path of the original .yml source file.
	SourceFile string

	// Extra captures any unrecognised YAML fields for lossless passthrough.
	Extra map[string]interface{}
}

// Guideline represents a parsed editorial guideline Markdown file.
type Guideline struct {
	// Name is the stem name derived from the filename (for example, "voice-and-tone").
	Name string

	// Title holds the title from YAML frontmatter. Required.
	Title string

	// Description holds the description from YAML frontmatter. Optional.
	Description string

	// Weight controls sort order. Lower values sort first. Default: 0.
	Weight int

	// Body holds the Markdown content after frontmatter extraction.
	Body string

	// SourceFile is the absolute path of the original .md file.
	SourceFile string
}

// ParseResult holds all parsed content from a Vale package directory.
type ParseResult struct {
	Rules      []*ValeRule
	Guidelines []*Guideline
	Warnings   []ParseWarning
}

// Action represents an optional inline action attached to a Vale rule.
type Action struct {
	// Name is the action type: "replace", "remove", "edit", or "convert".
	Name string

	// Params holds additional parameters for the action.
	Params []string
}
