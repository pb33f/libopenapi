// Copyright 2023-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

// Package specpath classifies a position in an OpenAPI or JSON Schema document by its path, the
// mapping keys from the document root: an example payload, a schema instance value, a
// specification extension. The index and the bundler share these rules, so the bundler does not
// rewrite a $ref the index treats as data. They are heuristics, kept internal so they can change.
package specpath

import (
	"strings"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/utils"
)

// UnderOpenAPIExamplePath reports whether seenPath is under an OpenAPI example or examples
// keyword (sample data, not schema). Entries in maps of named schemas are names,
// even when the name is "example" or "examples".
func UnderOpenAPIExamplePath(seenPath []string) bool {
	for i := range seenPath {
		if isOpenAPIExampleKeywordSegment(seenPath, i) {
			return true
		}
	}
	return false
}

func isOpenAPIExampleKeywordSegment(seenPath []string, idx int) bool {
	if idx < 0 || idx >= len(seenPath) {
		return false
	}
	if seenPath[idx] != "example" && seenPath[idx] != "examples" {
		return false
	}
	return isOpenAPIKeywordSegment(seenPath, idx)
}

// isOpenAPIKeywordSegment reports whether seenPath[idx] is a field name rather than a name the
// document chose, such as a component, property, header, response code, callback, webhook,
// operation or Example Object name. A property named "default" or a response keyed "default" is
// a name; the "default" field of a schema is not.
func isOpenAPIKeywordSegment(seenPath []string, idx int) bool {
	if idx < 0 || idx >= len(seenPath) {
		return false
	}
	// A component or schema name cannot itself introduce another named map.
	// For example, properties/definitions/example ends at a payload keyword.
	nameSlot := false
	for i, segment := range seenPath[:idx] {
		if nameSlot {
			nameSlot = false
			continue
		}
		if i == 1 && seenPath[0] == "components" {
			nameSlot = true
			continue
		}
		// Not parameters: the Swagger 2.0 root map shares its path with a path item fragment's
		// array, whose elements add no segment. Those parameter names rely on the schema check.
		switch segment {
		case "properties", "patternProperties", "definitions", "$defs", "dependentSchemas",
			"headers", "links", "callbacks", "encoding", "webhooks", "responses", "examples",
			"additionalOperations":
			nameSlot = true
		}
	}
	return !nameSlot
}

// IsOpenAPIExtensionSegment reports whether seenPath[idx] is a specification extension rather than
// a name the document chose. A schema property named x-... is not an extension. The Responses
// Object allows extensions alongside response codes, so an x- key there is an extension. The root
// responses map of Swagger 2.0 and components/responses hold response names.
func IsOpenAPIExtensionSegment(seenPath []string, idx int) bool {
	if idx < 0 || idx >= len(seenPath) || !strings.HasPrefix(seenPath[idx], "x-") {
		return false
	}
	if idx > 1 && seenPath[idx-1] == "responses" && (idx != 2 || seenPath[0] != "components") {
		return true
	}
	return isOpenAPIKeywordSegment(seenPath, idx)
}

// isSchemaLocation reports whether path ends at a Schema Object: the value of a keyword that holds
// a schema or an array of schemas, an element of a schema array, or an entry of properties,
// patternProperties, definitions, $defs, dependentSchemas or components/schemas. These path rules
// do not look at node. Only when none matches, as at the root of a referenced file or below a name
// the document chose, is node checked for schema keywords; a nil node then gives false.
//
// isSchemaSourceSegment in the bundler classifies a $ref's source path with a similar list; the
// two differ on purpose, see the comment there.
func isSchemaLocation(path []string, node *yaml.Node) bool {
	if n := len(path); n > 0 {
		switch path[n-1] {
		case "schema", "itemSchema", "items", "additionalItems", "additionalProperties", "contains",
			"not", "if", "then", "else", "propertyNames", "contentSchema", "unevaluatedItems",
			"unevaluatedProperties",
			// The index walk adds no segment for array elements.
			"allOf", "anyOf", "oneOf", "prefixItems":
			return true
		}
		if n > 1 {
			switch path[n-2] {
			case "properties", "patternProperties", "definitions", "$defs", "dependentSchemas",
				// An element of a schema array, by resolver marker or JSON pointer index.
				"allOf", "anyOf", "oneOf", "prefixItems":
				return true
			case "schemas":
				if n == 3 && path[0] == "components" {
					return true
				}
			}
		}
	}
	return hasSchemaKeyword(node)
}

// hasSchemaKeyword reports whether node is a mapping that uses a schema keyword, such as type,
// format, properties, allOf or $ref. Swagger 2.0 parameters and headers that are not in the body
// use type and format as well, and their default is instance data too. A keyword whose value has a
// shape no schema allows, such as a type that is a mapping, means node is a map of names, such as
// a parameter catalog with an entry named type, and not a schema. Keys merged with << count, as
// they do for the index walk.
func hasSchemaKeyword(node *yaml.Node) bool {
	node = utils.ResolveMergeNodes(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return false
	}
	found := false
	for i := 0; i+1 < len(node.Content); i += 2 {
		value := utils.NodeAlias(node.Content[i+1])
		if value == nil {
			continue
		}
		switch node.Content[i].Value {
		case "$schema", "$id", "$ref", "format":
			if value.Kind != yaml.ScalarNode {
				return false
			}
			found = true
		case "type":
			if value.Kind == yaml.MappingNode {
				return false
			}
			found = true
		case "required":
			// Swagger 2.0 parameters and headers use a boolean required.
			if value.Kind == yaml.MappingNode {
				return false
			}
			found = found || value.Kind == yaml.SequenceNode
		case "prefixItems", "allOf", "anyOf", "oneOf":
			if value.Kind == yaml.MappingNode {
				return false
			}
			found = true
		case "$defs", "properties", "patternProperties", "additionalProperties", "items", "not",
			"if", "then", "else", "discriminator":
			found = true
		}
	}
	return found
}

// UnderSchemaInstanceValuePath reports whether path, the location of node, is inside a value
// JSON Schema treats as instance data: the value of enum, const or default, or the examples array,
// in a schema. A $ref key there is part of the data, not a reference. An OpenAPI examples field is
// a map of Example Objects, so examples counts only when node is an array. parent, when not nil,
// is the node holding node. When the keyword is the last segment of path, parent must be a mapping,
// as a sequence element has no key, and it is checked for schema keywords when the path cannot tell
// whether it is a schema.
func UnderSchemaInstanceValuePath(path []string, node, parent *yaml.Node) bool {
	last := len(path) - 1
	for i, segment := range path {
		switch segment {
		case "enum", "const", "default":
		case "examples":
			// Deeper examples segments were already checked when the walk descended.
			if i != last || !utils.IsNodeArray(node) {
				continue
			}
		default:
			continue
		}
		var holder *yaml.Node
		if i == last && parent != nil {
			// A segment taken from a sequence element is not a mapping key.
			if !utils.IsNodeMap(parent) {
				continue
			}
			holder = parent
		}
		if isOpenAPIKeywordSegment(path, i) && isSchemaLocation(path[:i], holder) {
			return true
		}
	}
	return false
}

// IsLiteralValue reports whether node, the value at path within parent, is literal data rather
// than document structure: an OpenAPI example payload, or a schema instance value such as the
// value of default, enum or const. A $ref key inside literal data is not a reference. A sequence
// element may add no segment to path or a "[]" segment. parent may be nil when it is not known.
func IsLiteralValue(path []string, node, parent *yaml.Node) bool {
	if UnderOpenAPIExamplePayloadPath(path) {
		return true
	}
	if isDirectOpenAPIExampleValuePath(path) && !isDirectOpenAPIExampleRefNode(node) {
		return true
	}
	return UnderSchemaInstanceValuePath(path, node, parent)
}

// UnderOpenAPIExamplePayloadPath reports whether seenPath points to raw example payload content.
// A bare `example` path is not payload by itself because libopenapi still supports a direct
// `$ref` wrapper there for bundling. Once traversal moves below `example`, or into an Example
// Object's `value`/`dataValue`, the path is payload and nested `$ref` keys should be ignored.
func UnderOpenAPIExamplePayloadPath(seenPath []string) bool {
	for i := range seenPath {
		if !isOpenAPIExampleKeywordSegment(seenPath, i) {
			continue
		}
		switch seenPath[i] {
		case "example":
			if len(seenPath) > i+1 {
				return true
			}
		case "examples":
			if len(seenPath) > i+2 {
				switch seenPath[i+2] {
				case "value", "dataValue":
					return true
				}
			}
		}
	}
	return false
}

// isDirectOpenAPIExampleValuePath reports whether seenPath points at the value of an OpenAPI
// `example` field itself. This is used to allow a top-level `$ref` wrapper while still skipping
// traversal into arbitrary example payload objects.
func isDirectOpenAPIExampleValuePath(seenPath []string) bool {
	for i := range seenPath {
		if seenPath[i] == "example" && isOpenAPIExampleKeywordSegment(seenPath, i) && len(seenPath) == i+1 {
			return true
		}
	}
	return false
}

// isDirectOpenAPIExampleRefNode reports whether node is a mapping holding only a $ref, the wrapper
// an example field may use.
func isDirectOpenAPIExampleRefNode(node *yaml.Node) bool {
	node = utils.ResolveMergeNodes(node)
	return utils.IsNodeMap(node) && utils.GetRefValueNode(node) != nil && len(node.Content) == 2
}
