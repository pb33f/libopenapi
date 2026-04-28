// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import "go.yaml.in/yaml/v4"

type enumShape struct {
	goType        string
	constants     bool
	mixed         bool
	nullable      bool
	nonNullValues int
}

func enumShapeFor(nodes []*yaml.Node) enumShape {
	var shape enumShape
	var family string
	for _, node := range nodes {
		if node == nil || nodeIsNull(node) {
			shape.nullable = true
			continue
		}
		shape.nonNullValues++
		next := enumFamily(node)
		if family == "" {
			family = next
			continue
		}
		if family == "number" && next == "integer" {
			continue
		}
		if family == "integer" && next == "number" {
			family = "number"
			continue
		}
		if family != next {
			shape.mixed = true
		}
	}
	switch {
	case shape.nonNullValues == 0:
		shape.goType = "any"
	case shape.mixed:
		shape.goType = "any"
	case family == "integer":
		shape.goType = "int"
		shape.constants = true
	case family == "number":
		shape.goType = "float64"
		shape.constants = true
	case family == "boolean":
		shape.goType = "bool"
		shape.constants = true
	default:
		shape.goType = "string"
		shape.constants = true
	}
	return shape
}

func enumFamily(node *yaml.Node) string {
	switch node.Tag {
	case "!!int":
		return "integer"
	case "!!float":
		return "number"
	case "!!bool":
		return "boolean"
	case "!!str":
		return "string"
	default:
		return "unknown"
	}
}

func enumHasNull(nodes []*yaml.Node) bool {
	return enumShapeFor(nodes).nullable
}

func enumIsMixed(nodes []*yaml.Node) bool {
	return enumShapeFor(nodes).mixed
}

func enumLiteral(node *yaml.Node, goType string) string {
	if node == nil || nodeIsNull(node) {
		return ""
	}
	switch goType {
	case "string":
		return strconvQuote(node.Value)
	case "int", "float64", "bool":
		return node.Value
	default:
		return ""
	}
}

func nodeIsNull(node *yaml.Node) bool {
	return node != nil && node.Tag == "!!null"
}
