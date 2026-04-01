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
	for i, p := range seenPath {
		if p == "example" || p == "examples" {
			if i == 0 || (seenPath[i-1] != "properties" && seenPath[i-1] != "patternProperties") {
				return true
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
