// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/pb33f/go-yaml"
	validator "github.com/pb33f/jsonschema/v6"
)

// The validator ships the draft and its vocabularies. Compiling it needs no network.
var draftMetaSchema = sync.OnceValue(func() *validator.Schema {
	return validator.NewCompiler().MustCompile(Dialect)
})

func (d *Document) validateSource(root *yaml.Node) error {
	if root.Kind == yaml.ScalarNode {
		return nil // indexSchema has already checked that this is a boolean.
	}
	validate := func(skipped *bool) error {
		value, err := d.metaValue(root, false, skipped)
		if err != nil {
			return err
		}
		return draftMetaSchema().Validate(value)
	}
	var skipped bool
	err := validate(&skipped)
	if err != nil && skipped {
		// Include the authored path of every invalid member in the diagnostic.
		return validate(nil)
	}
	return err
}

// Use authored number strings. A float64 round trip can turn a fractional schema
// limit into an integer or round a small, positive multipleOf to zero.
func (d *Document) metaValue(node *yaml.Node, repeatedSchemas bool, skipped *bool) (any, error) {
	switch node.Kind {
	case yaml.MappingNode:
		value := make(map[string]any, len(node.Content)/2)
		_, schema := d.schemas[node]
		var previous *yaml.Node
		for i := 0; i < len(node.Content); i += 2 {
			name, child := node.Content[i].Value, node.Content[i+1]
			// These meta-schema maps apply exactly the same schema to every
			// value and impose no constraints on member names or cardinality.
			if skipped != nil && repeatedSchemas && previous != nil && sameNode(previous, child) {
				*skipped = true
				continue
			}
			previous = child
			repeated := schema && (name == "properties" || name == "$defs" || name == "dependentSchemas")
			v, err := d.metaValue(child, repeated, skipped)
			if err != nil {
				return nil, err
			}
			value[name] = v
		}
		return value, nil
	case yaml.SequenceNode:
		value := make([]any, len(node.Content))
		for i, child := range node.Content {
			v, err := d.metaValue(child, false, skipped)
			if err != nil {
				return nil, err
			}
			value[i] = v
		}
		return value, nil
	}
	switch node.Tag {
	case "!!null":
		return nil, nil
	case "!!bool":
		return strings.EqualFold(node.Value, "true"), nil
	case "!!int", "!!float":
		n := node.Value
		if !json.Valid([]byte(n)) {
			if node.Tag == "!!int" {
				if v, ok := new(big.Int).SetString(n, 0); ok {
					return json.Number(v.String()), nil
				}
			}
			n = strings.TrimPrefix(strings.ReplaceAll(n, "_", ""), "+")
			if strings.HasPrefix(n, ".") {
				n = "0" + n
			} else if strings.HasPrefix(n, "-.") {
				n = "-0" + n[1:]
			}
			n = strings.ReplaceAll(strings.ReplaceAll(n, ".e", ".0e"), ".E", ".0E")
			if strings.HasSuffix(n, ".") {
				n += "0"
			}
			if !json.Valid([]byte(n)) {
				return nil, fmt.Errorf("line %d: invalid JSON number %q", node.Line, node.Value)
			}
		}
		return json.Number(n), nil
	}
	return node.Value, nil
}
