// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package bundler

import (
	"strings"

	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"go.yaml.in/yaml/v4"
)

// DetectOpenAPIComponentType attempts to determine what type of OpenAPI component a node represents.
//
// This conservative default ignores quoted mapping keys so YAML inputs can escape
// reserved OpenAPI keywords without being misclassified as components. For nodes
// parsed from JSON, where object keys are inherently quoted, use
// DetectOpenAPIComponentTypeForJSON instead.
func DetectOpenAPIComponentType(node *yaml.Node) (string, bool) {
	return detectOpenAPIComponentType(node, false)
}

// DetectOpenAPIComponentTypeForJSON attempts to determine what type of OpenAPI
// component a node represents when the source document uses JSON object syntax.
//
// JSON parsers preserve object keys as quoted scalars, so this variant treats
// quoted keys as OpenAPI keywords. It does not change the default YAML-oriented
// behavior of DetectOpenAPIComponentType.
func DetectOpenAPIComponentTypeForJSON(node *yaml.Node) (string, bool) {
	return detectOpenAPIComponentType(node, true)
}

func detectOpenAPIComponentType(node *yaml.Node, includeQuotedKeys bool) (string, bool) {
	if node == nil {
		return "", false
	}

	// Try to build different component types and see which one succeeds
	// Order matters - try more specific component types first
	if hasParameterPropertiesWithOptions(node, includeQuotedKeys) {
		return v3.ParametersLabel, true
	}

	if hasResponsePropertiesWithOptions(node, includeQuotedKeys) {
		return v3.ResponsesLabel, true
	}

	if hasExamplePropertiesWithOptions(node, includeQuotedKeys) {
		return v3.ExamplesLabel, true
	}

	if hasLinkPropertiesWithOptions(node, includeQuotedKeys) {
		return v3.LinksLabel, true
	}

	if hasCallbackProperties(node) {
		return v3.CallbacksLabel, true
	}

	if hasPathItemPropertiesWithOptions(node, includeQuotedKeys) {
		return v3.PathItemsLabel, true
	}

	if hasRequestBodyPropertiesWithOptions(node, includeQuotedKeys) {
		return v3.RequestBodiesLabel, true
	}

	if hasHeaderPropertiesWithOptions(node, includeQuotedKeys) {
		return v3.HeadersLabel, true
	}

	if hasSchemaPropertiesWithOptions(node, includeQuotedKeys) {
		return v3.SchemasLabel, true
	}

	return "", false
}

func hasSchemaProperties(node *yaml.Node) bool {
	return hasSchemaPropertiesWithOptions(node, false)
}

func hasSchemaPropertiesWithOptions(node *yaml.Node, includeQuotedKeys bool) bool {
	// Schema typically has properties like "type", "properties", "items", "allOf", etc.
	keys := getNodeKeysWithOptions(node, includeQuotedKeys)
	schemaIndicators := []string{
		v3.TypeLabel, v3.PropertiesLabel,
		v3.ItemsLabel, v3.AllOfLabel, v3.AnyOfLabel, v3.OneOfLabel, v3.EnumLabel,
	}

	for _, indicator := range schemaIndicators {
		if containsKey(keys, indicator) {
			return true
		}
	}
	return false
}

func hasResponseProperties(node *yaml.Node) bool {
	return hasResponsePropertiesWithOptions(node, false)
}

func hasResponsePropertiesWithOptions(node *yaml.Node, includeQuotedKeys bool) bool {
	// Response typically has "description" and "content" or "headers"
	keys := getNodeKeysWithOptions(node, includeQuotedKeys)

	// And typically has content or headers
	return (containsKey(keys, v3.ContentLabel) || containsKey(keys, v3.HeadersLabel) ||
		containsKey(keys, v3.LinksLabel)) && !containsKey(keys, v3.RequiredLabel)
}

func hasParameterProperties(node *yaml.Node) bool {
	return hasParameterPropertiesWithOptions(node, false)
}

func hasParameterPropertiesWithOptions(node *yaml.Node, includeQuotedKeys bool) bool {
	// Parameter must have "name" or "in"
	keys := getNodeKeysWithOptions(node, includeQuotedKeys)
	return containsKey(keys, v3.NameLabel) || containsKey(keys, v3.InLabel)
}

func hasRequestBodyProperties(node *yaml.Node) bool {
	return hasRequestBodyPropertiesWithOptions(node, false)
}

func hasRequestBodyPropertiesWithOptions(node *yaml.Node, includeQuotedKeys bool) bool {
	// RequestBody typically has "content" and optionally "required" and "description"
	keys := getNodeKeysWithOptions(node, includeQuotedKeys)
	return containsKey(keys, v3.ContentLabel)
}

func hasHeaderProperties(node *yaml.Node) bool {
	return hasHeaderPropertiesWithOptions(node, false)
}

func hasHeaderPropertiesWithOptions(node *yaml.Node, includeQuotedKeys bool) bool {
	// Headers are similar to parameters but without "in" and "name"
	keys := getNodeKeysWithOptions(node, includeQuotedKeys)

	// Headers can have schema or content but not both
	return (containsKey(keys, v3.SchemaLabel) || containsKey(keys, v3.ContentLabel)) &&
		!containsKey(keys, v3.InLabel) && !containsKey(keys, v3.NameLabel)
}

func hasExampleProperties(node *yaml.Node) bool {
	return hasExamplePropertiesWithOptions(node, false)
}

func hasExamplePropertiesWithOptions(node *yaml.Node, includeQuotedKeys bool) bool {
	// Example typically has "value" or "externalValue" or both
	keys := getNodeKeysWithOptions(node, includeQuotedKeys)
	return containsKey(keys, v3.ValueLabel) || containsKey(keys, v3.ExternalValue)
}

func hasLinkProperties(node *yaml.Node) bool {
	return hasLinkPropertiesWithOptions(node, false)
}

func hasLinkPropertiesWithOptions(node *yaml.Node, includeQuotedKeys bool) bool {
	// Link typically has "operationRef" or "operationId"
	keys := getNodeKeysWithOptions(node, includeQuotedKeys)
	return containsKey(keys, v3.OperationRefLabel) || containsKey(keys, v3.OperationIdLabel)
}

func hasCallbackProperties(node *yaml.Node) bool {
	// Callback is a map where keys are expressions and values are PathItems
	// This is harder to detect, but we can check if it's a map with path-like keys
	if node.Kind != yaml.MappingNode || len(node.Content) < 2 {
		return false
	}

	// Check if at least one key contains a path-like pattern (with {})
	for i := 0; i < len(node.Content); i += 2 {
		if strings.Contains(node.Content[i].Value, "{$") {
			return true
		}
	}
	return false
}

func hasPathItemProperties(node *yaml.Node) bool {
	return hasPathItemPropertiesWithOptions(node, false)
}

func hasPathItemPropertiesWithOptions(node *yaml.Node, includeQuotedKeys bool) bool {
	// PathItem typically has HTTP methods as keys
	keys := getNodeKeysWithOptions(node, includeQuotedKeys)
	httpMethods := []string{"get", "post", "put", "delete", "options", "head", "patch", "trace"}

	for _, method := range httpMethods {
		if containsKey(keys, method) {
			return true
		}
	}

	// It might also have "parameters" or "$ref"
	return containsKey(keys, v3.ParametersLabel)
}

// Helper function to get all keys from a mapping node.
//
// By default quoted keys are skipped so YAML inputs can quote reserved words
// like "type" or "items" without affecting component-type detection. Callers
// handling JSON syntax should opt in to quoted keys.
func getNodeKeys(node *yaml.Node) []string {
	return getNodeKeysWithOptions(node, false)
}

func getNodeKeysWithOptions(node *yaml.Node, includeQuotedKeys bool) []string {
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}
	if node.Kind != yaml.MappingNode {
		return nil
	}

	var keys []string
	for i := 0; i < len(node.Content); i += 2 {
		if i < len(node.Content) {
			keyNode := node.Content[i]
			// Skip quoted keys for the conservative YAML-oriented default.
			if !includeQuotedKeys &&
				(keyNode.Style == yaml.SingleQuotedStyle || keyNode.Style == yaml.DoubleQuotedStyle) {
				continue
			}
			keys = append(keys, keyNode.Value)
		}
	}
	return keys
}

// Helper function to check if a slice contains a string
func containsKey(keys []string, key string) bool {
	for _, k := range keys {
		if k == key {
			return true
		}
	}
	return false
}

// Helper function to get a value for a specific key in a mapping node
func getNodeValueForKey(node *yaml.Node, key string) string {
	if node.Kind != yaml.MappingNode {
		return ""
	}

	for i := 0; i < len(node.Content); i += 2 {
		if i+1 < len(node.Content) && node.Content[i].Value == key {
			return node.Content[i+1].Value
		}
	}
	return ""
}
