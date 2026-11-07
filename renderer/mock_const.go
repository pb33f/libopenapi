// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package renderer

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/pb33f/go-yaml"
)

// mockConstString prevents YAML from interpreting scalar constants as numbers or null.
type mockConstString string

func (s mockConstString) MarshalYAML() (any, error) {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: string(s)}, nil
}

// mockConstNumber preserves a constant's exact decimal text in both formats.
type mockConstNumber string

func (n mockConstNumber) MarshalJSON() ([]byte, error) { return []byte(n), nil }
func (n mockConstNumber) MarshalYAML() (any, error) {
	tag := "!!int"
	if strings.ContainsAny(string(n), ".eE") {
		tag = "!!float"
	}
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: string(n), Style: yaml.TaggedStyle}, nil
}

// renderConst uses the existing work budgets before allocating containers. Alias
// expansion remains bounded and never changes the source YAML nodes.
func (ctx *mockRenderContext) renderConst(node *yaml.Node, depth int, active map[*yaml.Node]bool) (any, bool) {
	if node == nil {
		ctx.err = fmt.Errorf("mock const contains a missing value")
		return nil, false
	}
	if depth > ctx.options.MaxMockDepth {
		ctx.err = &MockGenerationBudgetError{Budget: "depth", Limit: ctx.options.MaxMockDepth, Actual: depth}
		return nil, false
	}
	if !ctx.noteNode() {
		return nil, false
	}
	if active[node] {
		ctx.err = fmt.Errorf("mock const contains an alias cycle")
		return nil, false
	}
	active[node] = true
	defer delete(active, node)
	fail := func() (any, bool) {
		ctx.err = fmt.Errorf("mock const must contain JSON-compatible values")
		return nil, false
	}
	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) != 1 {
			return fail()
		}
		return ctx.renderConst(node.Content[0], depth+1, active)
	case yaml.AliasNode:
		return ctx.renderConst(node.Alias, depth+1, active)
	case yaml.MappingNode:
		if len(node.Content)%2 != 0 {
			return fail()
		}
		output := make(map[string]any)
		if !ctx.noteValue(output) {
			return nil, false
		}
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			if key == nil || key.Kind != yaml.ScalarNode || key.ShortTag() != "!!str" {
				return fail()
			}
			if _, exists := output[key.Value]; exists {
				return fail()
			}
			if !ctx.noteProperty(key.Value) {
				return nil, false
			}
			value, ok := ctx.renderConst(node.Content[i+1], depth+1, active)
			if !ok {
				return nil, false
			}
			output[key.Value] = value
		}
		return output, true
	case yaml.SequenceNode:
		output := make([]any, 0)
		if !ctx.noteValue(output) {
			return nil, false
		}
		for _, child := range node.Content {
			value, ok := ctx.renderConst(child, depth+1, active)
			if !ok {
				return nil, false
			}
			output = append(output, value)
		}
		return output, true
	case yaml.ScalarNode:
		var value any
		switch node.ShortTag() {
		case "!!null":
			value = nil
		case "!!str":
			value = mockConstString(node.Value)
		case "!!bool":
			if strings.EqualFold(node.Value, "true") {
				value = true
			} else if strings.EqualFold(node.Value, "false") {
				value = false
			} else {
				return fail()
			}
		case "!!int", "!!float":
			if !ctx.checkBudget("bytes", ctx.options.MaxMockBytes, ctx.bytes+len(node.Value)) {
				return nil, false
			}
			number := strings.ReplaceAll(node.Value, "_", "")
			number = strings.TrimPrefix(number, "+")
			if node.ShortTag() == "!!int" && !jsonNumberText(number) {
				integer, ok := new(big.Int).SetString(number, 0)
				if !ok {
					return fail()
				}
				number = integer.String()
			}
			if node.ShortTag() == "!!float" {
				// normalization must not invent a mantissa for malformed explicit tags.
				mantissaEnd := strings.IndexAny(number, "eE")
				if mantissaEnd < 0 {
					mantissaEnd = len(number)
				}
				if strings.IndexAny(number[:mantissaEnd], "0123456789") < 0 {
					return fail()
				}
				// YAML permits a missing zero on either side of the decimal point.
				sign := ""
				if strings.HasPrefix(number, "-") {
					sign, number = "-", number[1:]
				}
				if strings.HasPrefix(number, ".") {
					number = "0" + number
				}
				parts := strings.SplitN(strings.ToLower(number), "e", 2)
				// YAML allows leading zeroes in a decimal mantissa; JSON does not.
				mantissa := strings.SplitN(parts[0], ".", 2)
				mantissa[0] = strings.TrimLeft(mantissa[0], "0")
				if mantissa[0] == "" {
					mantissa[0] = "0"
				}
				parts[0] = strings.Join(mantissa, ".")
				if strings.HasSuffix(parts[0], ".") {
					parts[0] += "0"
				}
				number = sign + strings.Join(parts, "e")
			}
			if !jsonNumberText(number) {
				return fail()
			}
			value = mockConstNumber(number)
		default:
			return fail()
		}
		return value, ctx.noteValue(value)
	default:
		return fail()
	}
}

func jsonNumberText(value string) bool {
	return len(value) > 0 && (value[0] == '-' || (value[0] >= '0' && value[0] <= '9')) && json.Valid([]byte(value))
}
