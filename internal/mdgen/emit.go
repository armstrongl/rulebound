package mdgen

import (
	"fmt"
	"sort"

	"go.yaml.in/yaml/v3"
)

// EmitYAML converts a RuleSource into Vale-compatible YAML bytes.
// Field ordering: extends, message, level, then remaining scalars alphabetically,
// then type-specific compound fields (swap, tokens, exceptions).
func EmitYAML(src *RuleSource) ([]byte, []Warning, error) {
	var warnings []Warning

	root := &yaml.Node{Kind: yaml.MappingNode}

	// Required fields in fixed order.
	addScalar(root, "extends", src.Extends)
	addScalar(root, "message", src.Message)
	addScalar(root, "level", src.Level)

	// Remaining scalar fields, sorted alphabetically.
	// Separate known compound/type-specific keys to emit last.
	compoundKeys := map[string]bool{
		"swap": true, "tokens": true, "exceptions": true,
	}
	var scalarKeys []string
	for k := range src.Fields {
		if !compoundKeys[k] {
			scalarKeys = append(scalarKeys, k)
		}
	}
	sort.Strings(scalarKeys)

	for _, k := range scalarKeys {
		v := src.Fields[k]
		node, err := valueToNode(v)
		if err != nil {
			warnings = append(warnings, Warning{
				Message: fmt.Sprintf("skipping field %q: %v", k, err),
			})
			continue
		}
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: k},
			node,
		)
	}

	// Type-specific compound fields.
	switch src.Extends {
	case "substitution":
		if len(src.Swap) > 0 {
			swapNode := &yaml.Node{Kind: yaml.MappingNode}
			for _, pair := range src.Swap {
				swapNode.Content = append(swapNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: pair.Key},
					&yaml.Node{Kind: yaml.ScalarNode, Value: pair.Value},
				)
			}
			root.Content = append(root.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "swap"},
				swapNode,
			)
		}

	case "existence":
		if len(src.Tokens) > 0 {
			root.Content = append(root.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "tokens"},
				stringsToSeqNode(src.Tokens),
			)
		}

	case "capitalization":
		if len(src.Exceptions) > 0 {
			root.Content = append(root.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "exceptions"},
				stringsToSeqNode(src.Exceptions),
			)
		}
	}
	// occurrence: all fields are scalars, already emitted above.

	doc := &yaml.Node{
		Kind:    yaml.DocumentNode,
		Content: []*yaml.Node{root},
	}

	out, err := yaml.Marshal(doc)
	if err != nil {
		return nil, nil, fmt.Errorf("marshaling YAML: %w", err)
	}

	return out, warnings, nil
}

// addScalar appends a key-value pair with string values to a mapping node.
func addScalar(mapping *yaml.Node, key, value string) {
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Value: value},
	)
}

// stringsToSeqNode builds a YAML sequence node from a string slice.
func stringsToSeqNode(items []string) *yaml.Node {
	seq := &yaml.Node{Kind: yaml.SequenceNode}
	for _, item := range items {
		seq.Content = append(seq.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: item},
		)
	}
	return seq
}

// valueToNode converts a Go value to an appropriate yaml.Node.
func valueToNode(v interface{}) (*yaml.Node, error) {
	var node yaml.Node
	if err := node.Encode(v); err != nil {
		return nil, err
	}
	return &node, nil
}
