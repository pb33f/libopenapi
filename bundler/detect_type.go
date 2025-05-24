// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package bundler

import (
	"strings"

	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"gopkg.in/yaml.v3"
)

// DetectOpenAPIComponentType attempts to determine what type of OpenAPI component a node represents.
// It returns the component type as a string (schema, response, parameter, etc.) and a boolean indicating
// whether the type was successfully detected.
func DetectOpenAPIComponentType(node *yaml.Node) (string, bool) {
	if node == nil {
		return "", false
	}

	// Try to build different component types and see which one succeeds
	// Order matters - try more specific component types first
	if hasParameterProperties(node) {
		return v3.ParametersLabel, true
	}

	if hasResponseProperties(node) {
		return v3.ResponsesLabel, true
	}

	if hasExampleProperties(node) {
		return v3.ExamplesLabel, true
	}

	if hasLinkProperties(node) {
		return v3.LinksLabel, true
	}

	if hasCallbackProperties(node) {
		return v3.CallbacksLabel, true
	}

	if hasPathItemProperties(node) {
		return v3.PathItemsLabel, true
	}

	if hasRequestBodyProperties(node) {
		return v3.RequestBodiesLabel, true
	}

	if hasHeaderProperties(node) {
		return v3.HeadersLabel, true
	}

	if hasSchemaProperties(node) {
		return v3.SchemasLabel, true
	}

	return "", false
}

func hasSchemaProperties(node *yaml.Node) bool {
	// Schema typically has properties like "type", "properties", "items", "allOf", etc.
	keys := getNodeKeys(node)
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
	// Response typically has "description" and "content" or "headers"
	keys := getNodeKeys(node)

	// And typically has content or headers
	return (containsKey(keys, v3.ContentLabel) || containsKey(keys, v3.HeadersLabel) ||
		containsKey(keys, v3.LinksLabel)) && !containsKey(keys, v3.RequiredLabel)
}

func hasParameterProperties(node *yaml.Node) bool {
	// Parameter must have "name" or "in"
	keys := getNodeKeys(node)
	return containsKey(keys, v3.NameLabel) || containsKey(keys, v3.InLabel)
}

func hasRequestBodyProperties(node *yaml.Node) bool {
	// RequestBody typically has "content" and optionally "required" and "description"
	keys := getNodeKeys(node)
	return containsKey(keys, v3.ContentLabel)
}

func hasHeaderProperties(node *yaml.Node) bool {
	// Headers are similar to parameters but without "in" and "name"
	keys := getNodeKeys(node)

	// Headers can have schema or content but not both
	return (containsKey(keys, v3.SchemaLabel) || containsKey(keys, v3.ContentLabel)) &&
		!containsKey(keys, v3.InLabel) && !containsKey(keys, v3.NameLabel)
}

func hasExampleProperties(node *yaml.Node) bool {
	// Example typically has "value" or "externalValue" or both
	keys := getNodeKeys(node)
	return containsKey(keys, v3.ValueLabel) || containsKey(keys, v3.ExternalValue)
}

func hasLinkProperties(node *yaml.Node) bool {
	// Link typically has "operationRef" or "operationId"
	keys := getNodeKeys(node)
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
	// PathItem typically has HTTP methods as keys
	keys := getNodeKeys(node)
	httpMethods := []string{"get", "post", "put", "delete", "options", "head", "patch", "trace"}

	for _, method := range httpMethods {
		if containsKey(keys, method) {
			return true
		}
	}

	// It might also have "parameters" or "$ref"
	return containsKey(keys, v3.ParametersLabel)
}

// Helper function to get all keys from a mapping node
func getNodeKeys(node *yaml.Node) []string {
	if node.Kind != yaml.MappingNode {
		return nil
	}

	var keys []string
	for i := 0; i < len(node.Content); i += 2 {
		if i < len(node.Content) {
			keys = append(keys, node.Content[i].Value)
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
