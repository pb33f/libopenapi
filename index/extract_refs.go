// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"context"

	"go.yaml.in/yaml/v4"
)

func isSchemaContainingNode(v string) bool {
	switch v {
	case "schema", "items", "additionalProperties", "contains", "not",
		"unevaluatedItems", "unevaluatedProperties":
		return true
	}
	return false
}

func isMapOfSchemaContainingNode(v string) bool {
	switch v {
	case "properties", "patternProperties":
		return true
	}
	return false
}

func isArrayOfSchemaContainingNode(v string) bool {
	switch v {
	case "allOf", "anyOf", "oneOf", "prefixItems":
		return true
	}
	return false
}

// underOpenAPIExamplePath reports whether seenPath is under an OpenAPI example or examples
// keyword (sample data, not schema). A segment named "example" or "examples" that is preceded
// by "properties" or "patternProperties" is a schema property name, not an OpenAPI keyword.
func underOpenAPIExamplePath(seenPath []string) bool {
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
	switch seenPath[idx] {
	case "example", "examples":
		return idx == 0 || (seenPath[idx-1] != "properties" && seenPath[idx-1] != "patternProperties")
	default:
		return false
	}
}

// underOpenAPIExamplePayloadPath reports whether seenPath points to raw example payload content,
// not the Example Object itself. This lets the walker keep traversing legitimate Example Objects
// for $ref values while still skipping sample data under example/value/dataValue.
func underOpenAPIExamplePayloadPath(seenPath []string) bool {
	for i := range seenPath {
		if !isOpenAPIExampleKeywordSegment(seenPath, i) {
			continue
		}
		switch seenPath[i] {
		case "example":
			return true
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

// ExtractRefs will return a deduplicated slice of references for every unique ref found in the document.
// The total number of refs, will generally be much higher, you can extract those from GetRawReferenceCount()
func (index *SpecIndex) ExtractRefs(ctx context.Context, node, parent *yaml.Node, seenPath []string, level int, poly bool, pName string) []*Reference {
	if node == nil {
		return nil
	}

	state := index.initializeExtractRefsState(ctx, node, seenPath, level, poly, pName)
	found := index.walkExtractRefs(node, parent, &state)
	index.refCount = len(index.allRefs)
	return found
}
