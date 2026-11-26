// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package datamodel

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/internal/jsonnode"
)

// DecodeSchemaNode reads one JSON or YAML schema and preserves authored positions.
func DecodeSchemaNode(data []byte) (*yaml.Node, error) {
	var doc *yaml.Node
	if json.Valid(data) {
		var ok bool
		doc, ok = jsonnode.ParseJSON(data)
		if !ok {
			return nil, fmt.Errorf("invalid JSON schema document")
		}
	} else {
		decoder := yaml.NewDecoder(bytes.NewReader(data))
		doc = new(yaml.Node)
		if err := decoder.Decode(doc); err != nil {
			return nil, fmt.Errorf("invalid schema document: %w", err)
		}
		var extra yaml.Node
		if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
			if err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("expected one schema document")
		}
	}
	if err := checkSchemaKeys(doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// Check keys in linear time on wide property maps. Every YAML mapping in a JSON
// document, including objects inside examples, has unique string keys.
func checkSchemaKeys(n *yaml.Node) error {
	switch n.Kind {
	case yaml.MappingNode:
		seen := make(map[string]int, len(n.Content)/2)
		for i := 0; i < len(n.Content); i += 2 {
			key := n.Content[i]
			if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
				return fmt.Errorf("line %d: JSON object keys must be strings", key.Line)
			}
			if line, ok := seen[key.Value]; ok {
				return fmt.Errorf("line %d: mapping key %q already defined at line %d", key.Line, key.Value, line)
			}
			seen[key.Value] = key.Line
			if err := checkSchemaKeys(n.Content[i+1]); err != nil {
				return err
			}
		}
	case yaml.DocumentNode, yaml.SequenceNode:
		for _, child := range n.Content {
			if err := checkSchemaKeys(child); err != nil {
				return err
			}
		}
	case yaml.ScalarNode:
		// YAML's resolver tags a number outside float64 range as a plain string.
		// Keep the JSON numeric type when the authored token is a JSON number.
		if n.Tag == "!!str" && n.Style == 0 && n.Value != "" &&
			(n.Value[0] == '-' || n.Value[0] >= '0' && n.Value[0] <= '9') && json.Valid([]byte(n.Value)) {
			n.Tag = "!!int"
			if strings.ContainsAny(n.Value, ".eE") {
				n.Tag = "!!float"
			}
		}
		switch n.Tag {
		case "!!str", "!!bool", "!!int", "!!float", "!!null":
		default:
			return fmt.Errorf("line %d: unsupported JSON scalar %s", n.Line, n.Tag)
		}
	default:
		return fmt.Errorf("line %d: expected JSON-compatible YAML", n.Line)
	}
	return nil
}
