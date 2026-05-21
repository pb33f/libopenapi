// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package bundler

import (
	"strings"

	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"go.yaml.in/yaml/v4"
)

// inferComponentTypeFromSourcePath returns the component bucket implied by the
// OpenAPI slot that contains a $ref. It is deliberately context-based: sparse
// but valid targets, such as description-only responses or empty schemas, cannot
// always be classified from their own shape.
func inferComponentTypeFromSourcePath(sourcePath []string) (string, bool) {
	if len(sourcePath) == 0 {
		return "", false
	}

	for i := len(sourcePath) - 1; i >= 0; i-- {
		segment := sourcePath[i]
		previous := ""
		if i > 0 {
			previous = sourcePath[i-1]
		}

		if isSingularExampleSourceSegment(sourcePath, i) {
			return v3.ExamplesLabel, true
		}

		if isSchemaSourceSegment(sourcePath, i) {
			return v3.SchemasLabel, true
		}

		switch previous {
		case v3.ResponsesLabel:
			return v3.ResponsesLabel, true
		case v3.ParametersLabel:
			return v3.ParametersLabel, true
		case v3.HeadersLabel:
			return v3.HeadersLabel, true
		case v3.ExamplesLabel:
			return v3.ExamplesLabel, true
		case v3.LinksLabel:
			return v3.LinksLabel, true
		case v3.CallbacksLabel:
			if i == len(sourcePath)-1 {
				return v3.CallbacksLabel, true
			}
		case v3.PathItemsLabel:
			return v3.PathItemsLabel, true
		case v3.MediaTypesLabel:
			return v3.MediaTypesLabel, true
		case v3.ContentLabel:
			return v3.MediaTypesLabel, true
		}

		if segment == v3.RequestBodyLabel {
			return v3.RequestBodiesLabel, true
		}
	}

	if pathContains(sourcePath, v3.CallbacksLabel) {
		return v3.PathItemsLabel, true
	}
	if len(sourcePath) == 2 && (sourcePath[0] == v3.PathsLabel || sourcePath[0] == v3.WebhooksLabel) {
		return v3.PathItemsLabel, true
	}
	if len(sourcePath) > 1 && sourcePath[0] == v3.ComponentsLabel && sourcePath[1] == v3.SchemasLabel {
		return v3.SchemasLabel, true
	}
	return "", false
}

// canComposeContextualReference reports whether a source-slot inference is safe
// for the referenced node. JSON Pointer refs already identify a specific node,
// so source context can classify sparse but valid targets. Bare-file refs need a
// stronger guard because the file may be a wrapper map or full OpenAPI document.
func canComposeContextualReference(componentType string, node *yaml.Node, bareFile bool) bool {
	node = unwrapDocumentNode(node)
	if node == nil || isOpenAPIDocumentNode(node) {
		return false
	}
	if !bareFile {
		return true
	}

	if detectedType, ok := DetectOpenAPIComponentType(node); ok {
		if detectedType == componentType {
			return true
		}
		// Media Type and Header objects both use schema/content-shaped fields.
		// In a media type slot, the source path breaks that tie.
		if componentType != v3.MediaTypesLabel {
			return false
		}
	}

	keys := getNodeKeys(node)
	if len(keys) == 0 {
		return componentType == v3.SchemasLabel || componentType == v3.MediaTypesLabel
	}

	switch componentType {
	case v3.ResponsesLabel:
		return containsKey(keys, v3.DescriptionLabel)
	case v3.SchemasLabel:
		return containsKey(keys, v3.DescriptionLabel) ||
			containsKey(keys, v3.TitleLabel) ||
			containsKey(keys, v3.JSONSchemaLabel) ||
			containsKey(keys, v3.SchemaDialectLabel)
	case v3.ExamplesLabel:
		return containsKey(keys, v3.SummaryLabel) || containsKey(keys, v3.DescriptionLabel)
	case v3.HeadersLabel, v3.LinksLabel, v3.PathItemsLabel:
		return containsKey(keys, v3.SummaryLabel) || containsKey(keys, v3.DescriptionLabel)
	case v3.MediaTypesLabel:
		return containsKey(keys, v3.SchemaLabel) ||
			containsKey(keys, v3.ExampleLabel) ||
			containsKey(keys, v3.ExamplesLabel) ||
			containsKey(keys, v3.EncodingLabel)
	default:
		return false
	}
}

func unwrapDocumentNode(node *yaml.Node) *yaml.Node {
	if node != nil && node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return node.Content[0]
	}
	return node
}

func isOpenAPIDocumentNode(node *yaml.Node) bool {
	keys := getNodeKeys(node)
	return containsKey(keys, v3.OpenAPILabel) ||
		containsKey(keys, v3.SwaggerLabel) ||
		(containsKey(keys, v3.InfoLabel) && containsKey(keys, v3.PathsLabel))
}

// isSingularExampleSourceSegment reports whether sourcePath[index] is the
// OpenAPI example keyword, excluding schema properties named "example".
func isSingularExampleSourceSegment(sourcePath []string, index int) bool {
	if index < 0 || index >= len(sourcePath) || sourcePath[index] != v3.ExampleLabel {
		return false
	}
	if index == 0 {
		return true
	}
	switch sourcePath[index-1] {
	case "properties", "patternProperties":
		return false
	default:
		return true
	}
}

func isSchemaSourceSegment(sourcePath []string, index int) bool {
	segment := sourcePath[index]
	previous := ""
	if index > 0 {
		previous = sourcePath[index-1]
	}

	switch segment {
	case "schema", "items", "additionalProperties", "unevaluatedItems", "unevaluatedProperties",
		"contains", "not", "if", "then", "else", "propertyNames":
		return true
	}

	switch previous {
	case v3.SchemasLabel, "properties", "patternProperties", "$defs", "definitions", "dependentSchemas",
		"allOf", "anyOf", "oneOf", "prefixItems":
		return true
	}

	return false
}

func pathContains(path []string, needle string) bool {
	for _, segment := range path {
		if segment == needle {
			return true
		}
	}
	return false
}

func decodeSingleSegmentPointer(segment string) string {
	if strings.Contains(segment, "~") {
		segment = strings.ReplaceAll(segment, "~1", "/")
		segment = strings.ReplaceAll(segment, "~0", "~")
	}
	return segment
}
