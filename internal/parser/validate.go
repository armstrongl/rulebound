package parser

import (
	"fmt"
	"os"
	"strings"

	"go.yaml.in/yaml/v3"
)

// ValidationError records a single structural issue found in a Vale rule file.
type ValidationError struct {
	Field    string // The YAML field that caused the error (e.g. "extends", "message").
	Message  string // Human-readable description of the problem.
	Severity string // "error" or "warning".
}

// supportedExtendsV1 lists the extends types that ValidateRule performs
// type-specific checks for.
var supportedExtendsV1 = map[string]bool{
	ExtendsSubstitution:   true,
	ExtendsExistence:      true,
	ExtendsOccurrence:     true,
	ExtendsCapitalization: true,
}

// validLevels lists the accepted values for the level field.
var validLevels = map[string]bool{
	"suggestion": true,
	"warning":    true,
	"error":      true,
}

// ValidateRule reads a Vale YAML rule file and performs structural validation.
// It returns a slice of all validation errors found (empty when the rule is
// valid) and an error only for I/O or YAML-parse failures.
//
// Validation is performed against a map[string]interface{} rather than a typed
// struct so that key presence can be distinguished from Go zero values.
func ValidateRule(path string) ([]ValidationError, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return ValidateRuleBytes(data)
}

// ValidateRuleBytes validates raw YAML bytes as a Vale rule. It performs the
// same structural checks as ValidateRule but without reading from disk.
func ValidateRuleBytes(data []byte) ([]ValidationError, error) {
	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	var errs []ValidationError

	// ── extends ─────────────────────────────────────────────────────────────
	extendsVal, hasExtends := m["extends"]
	if !hasExtends {
		errs = append(errs, ValidationError{
			Field:    "extends",
			Message:  "missing required field 'extends'",
			Severity: "error",
		})
		return errs, nil
	}

	extendsStr, ok := extendsVal.(string)
	if !ok || strings.TrimSpace(extendsStr) == "" {
		errs = append(errs, ValidationError{
			Field:    "extends",
			Message:  "field 'extends' must be a non-empty string",
			Severity: "error",
		})
		return errs, nil
	}

	// ── message ─────────────────────────────────────────────────────────────
	if msgVal, hasMsg := m["message"]; !hasMsg {
		errs = append(errs, ValidationError{
			Field:    "message",
			Message:  "missing required field 'message'",
			Severity: "error",
		})
	} else if msgStr, ok := msgVal.(string); !ok || strings.TrimSpace(msgStr) == "" {
		errs = append(errs, ValidationError{
			Field:    "message",
			Message:  "field 'message' must be a non-empty string",
			Severity: "error",
		})
	}

	// ── level ───────────────────────────────────────────────────────────────
	if levelVal, hasLevel := m["level"]; hasLevel {
		levelStr, ok := levelVal.(string)
		if !ok || !validLevels[levelStr] {
			errs = append(errs, ValidationError{
				Field:   "level",
				Message: fmt.Sprintf("invalid level %q; must be one of: suggestion, warning, error", levelVal),
				Severity: "error",
			})
		}
	}

	// ── unsupported extends ─────────────────────────────────────────────────
	if !supportedExtendsV1[extendsStr] {
		supported := []string{
			ExtendsSubstitution,
			ExtendsExistence,
			ExtendsOccurrence,
			ExtendsCapitalization,
		}
		errs = append(errs, ValidationError{
			Field:   "extends",
			Message: fmt.Sprintf("unsupported extends type %q; supported types: %s", extendsStr, strings.Join(supported, ", ")),
			Severity: "error",
		})
		return errs, nil
	}

	// ── type-specific checks ────────────────────────────────────────────────
	switch extendsStr {
	case ExtendsSubstitution:
		errs = append(errs, validateSubstitution(m)...)
	case ExtendsExistence:
		errs = append(errs, validateExistence(m)...)
	case ExtendsOccurrence:
		errs = append(errs, validateOccurrence(m)...)
	case ExtendsCapitalization:
		errs = append(errs, validateCapitalization(m)...)
	}

	return errs, nil
}

// validateSubstitution checks substitution-specific fields.
func validateSubstitution(m map[string]interface{}) []ValidationError {
	var errs []ValidationError

	swapVal, hasSwap := m["swap"]
	if !hasSwap {
		errs = append(errs, ValidationError{
			Field:    "swap",
			Message:  "substitution rule requires a non-empty 'swap' map",
			Severity: "error",
		})
		return errs
	}

	switch v := swapVal.(type) {
	case map[string]interface{}:
		if len(v) == 0 {
			errs = append(errs, ValidationError{
				Field:    "swap",
				Message:  "substitution rule requires a non-empty 'swap' map",
				Severity: "error",
			})
		} else {
			for key, val := range v {
				if _, ok := val.(string); !ok {
					errs = append(errs, ValidationError{
						Field:    "swap",
						Message:  fmt.Sprintf("swap value for key %q must be a string, got %T", key, val),
						Severity: "error",
					})
				}
			}
		}
	case []interface{}:
		if len(v) == 0 {
			errs = append(errs, ValidationError{
				Field:    "swap",
				Message:  "substitution rule requires a non-empty 'swap' map",
				Severity: "error",
			})
		} else {
			for i, item := range v {
				m, ok := item.(map[string]interface{})
				if !ok {
					errs = append(errs, ValidationError{
						Field:    "swap",
						Message:  fmt.Sprintf("swap item [%d] must be a mapping, got %T", i, item),
						Severity: "error",
					})
					continue
				}
				for key, val := range m {
					if _, ok := val.(string); !ok {
						errs = append(errs, ValidationError{
							Field:    "swap",
							Message:  fmt.Sprintf("swap value for key %q must be a string, got %T", key, val),
							Severity: "error",
						})
					}
				}
			}
		}
	default:
		errs = append(errs, ValidationError{
			Field:    "swap",
			Message:  fmt.Sprintf("'swap' must be a mapping, got %T", swapVal),
			Severity: "error",
		})
	}

	return errs
}

// validateExistence checks existence-specific fields.
func validateExistence(m map[string]interface{}) []ValidationError {
	var errs []ValidationError

	tokensVal, hasTokens := m["tokens"]
	if !hasTokens {
		errs = append(errs, ValidationError{
			Field:    "tokens",
			Message:  "existence rule requires a non-empty 'tokens' list",
			Severity: "error",
		})
		return errs
	}

	switch v := tokensVal.(type) {
	case []interface{}:
		if len(v) == 0 {
			errs = append(errs, ValidationError{
				Field:    "tokens",
				Message:  "existence rule requires a non-empty 'tokens' list",
				Severity: "error",
			})
		}
	default:
		errs = append(errs, ValidationError{
			Field:    "tokens",
			Message:  fmt.Sprintf("'tokens' must be a list, got %T", tokensVal),
			Severity: "error",
		})
	}

	return errs
}

// validateOccurrence checks occurrence-specific fields.
func validateOccurrence(m map[string]interface{}) []ValidationError {
	var errs []ValidationError

	maxVal, hasMax := m["max"]
	minVal, hasMin := m["min"]
	if !hasMax && !hasMin {
		errs = append(errs, ValidationError{
			Field:    "max/min",
			Message:  "occurrence rule requires at least one of 'max' or 'min'",
			Severity: "error",
		})
	}
	if hasMax {
		if _, ok := maxVal.(int); !ok {
			errs = append(errs, ValidationError{
				Field:    "max",
				Message:  fmt.Sprintf("'max' must be an integer, got %T", maxVal),
				Severity: "error",
			})
		}
	}
	if hasMin {
		if _, ok := minVal.(int); !ok {
			errs = append(errs, ValidationError{
				Field:    "min",
				Message:  fmt.Sprintf("'min' must be an integer, got %T", minVal),
				Severity: "error",
			})
		}
	}

	tokenVal, hasToken := m["token"]
	if !hasToken {
		errs = append(errs, ValidationError{
			Field:    "token",
			Message:  "occurrence rule requires a 'token' field",
			Severity: "error",
		})
	} else if tokenStr, ok := tokenVal.(string); !ok || strings.TrimSpace(tokenStr) == "" {
		errs = append(errs, ValidationError{
			Field:    "token",
			Message:  "field 'token' must be a non-empty string",
			Severity: "error",
		})
	}

	return errs
}

// validateCapitalization checks capitalization-specific fields.
func validateCapitalization(m map[string]interface{}) []ValidationError {
	var errs []ValidationError

	if _, hasMatch := m["match"]; !hasMatch {
		errs = append(errs, ValidationError{
			Field:    "match",
			Message:  "capitalization rule requires a 'match' field",
			Severity: "error",
		})
	}

	return errs
}
