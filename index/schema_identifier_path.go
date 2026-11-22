// Copyright 2023-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"strings"
)

// schemaPointer joins path segments into a local JSON pointer.
func schemaPointer(path []string) string {
	if len(path) == 0 {
		return "#"
	}
	return "#/" + strings.Join(path, "/")
}

// inSchemaData reports whether a keyword at pointer, the pointer segments to the mapping holding
// it, sits in data rather than schema. Inside an $id resource, whose root is a schema, the path from
// that root decides, so resourceDepth is the length of the pointer to the innermost enclosing
// resource, or 0 outside any. extensions tells whether the content of a specification extension
// is data: an $anchor there declares nothing, while an $id there starts a resource of its own.
func inSchemaData(pointer []string, resourceDepth int, extensions bool) bool {
	if resourceDepth > 0 && resourceDepth <= len(pointer) {
		return underSchemaDataPath(pointer[resourceDepth:], true, extensions)
	}
	return underSchemaDataPath(pointer, false, extensions)
}

// underSchemaDataPath reports whether path, the pointer segments to a schema, runs through a value
// that is data rather than schema, where an $anchor declares nothing: an example, the value of
// enum, const or default, a Link Object's parameter values or request body, or a specification
// extension when extensions is set. A segment is data only where it is a field name; a property,
// component or response named "default" or "x-foo" is a name. When inSchema is set, path starts at
// a schema, the root of an $id resource, and only the schema's own keywords apply.
func underSchemaDataPath(path []string, inSchema, extensions bool) bool {
	// slotField is the field whose map the next segment names an entry of, and link is set when the
	// previous segment named a Link Object.
	slotField, link := "", false
	for i, segment := range path {
		if slotField != "" {
			field := slotField
			slotField = ""
			// an operation's Responses Object may carry extensions beside its status codes. Neither
			// components/responses nor Swagger 2.0's root responses (here i is 1) may.
			if extensions && field == "responses" && i > 1 && strings.HasPrefix(segment, "x-") {
				return true
			}
			link = field == "links"
			continue
		}
		if link {
			link = false
			if segment == "parameters" || segment == "requestBody" {
				return true
			}
		}
		switch segment {
		case "example", "examples", "enum", "const", "default":
			return true
		}
		if extensions && strings.HasPrefix(segment, "x-") {
			return true
		}
		if !inSchema && i == 1 && path[0] == "components" {
			// a component name. components/links holds Link Objects, as a response's links does.
			slotField = "components/" + segment
			if segment == "links" {
				slotField = segment
			}
			continue
		}
		switch segment {
		case "properties", "patternProperties", "definitions", "$defs", "dependentSchemas", "dependencies":
			slotField = segment
		case "headers", "links", "callbacks", "content", "encoding", "webhooks", "responses":
			if !inSchema {
				slotField = segment
			}
		case "parameters":
			// Swagger 2.0 keeps reusable parameters in a root map. Elsewhere it is an array.
			if !inSchema && i == 0 {
				slotField = segment
			}
		}
	}
	return false
}
