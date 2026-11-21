// Copyright 2023-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package specpath

import (
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestUnderOpenAPIExamplePath(t *testing.T) {
	tests := []struct {
		name string
		path []string
		want bool
	}{
		{"empty", nil, false},
		{"no_example_segments", []string{"paths", "get", "responses", "200", "content", "application/json", "schema"}, false},
		{"under_example", []string{"paths", "get", "responses", "200", "content", "application/json", "schema", "example"}, true},
		{"under_examples", []string{"content", "application/json", "schema", "examples", "sample", "value"}, true},
		{"example_not_whole_segment", []string{"paths", "exampled"}, false},
		{"example_as_property_name", []string{"components", "schemas", "Foo", "properties", "example"}, false},
		{"examples_as_property_name", []string{"components", "schemas", "Foo", "properties", "examples"}, false},
		{"nested_under_property_example", []string{"components", "schemas", "Foo", "properties", "example", "properties", "id"}, false},
		{"patternProperties_example", []string{"components", "schemas", "Foo", "patternProperties", "example"}, false},
		{"real_example_after_property_example", []string{"components", "schemas", "Foo", "properties", "example", "example"}, true},
		{"path_item_fragment_parameter_example", []string{"parameters", "example"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, UnderOpenAPIExamplePath(tt.path))
		})
	}
}

func TestUnderOpenAPIExamplePayloadPath(t *testing.T) {
	tests := []struct {
		name string
		path []string
		want bool
	}{
		{"empty", nil, false},
		{"example_root", []string{"paths", "get", "responses", "200", "content", "application/json", "schema", "example"}, false},
		{"nested_under_example_payload", []string{"components", "schemas", "Foo", "example", "nested"}, true},
		{"examples_collection", []string{"components", "examples"}, false},
		{"example_object_entry", []string{"components", "examples", "ReusableExample"}, false},
		{"examples_value_payload", []string{"content", "application/json", "examples", "sample", "value"}, true},
		{"examples_value_nested_payload", []string{"content", "application/json", "examples", "sample", "value", "nested"}, true},
		{"examples_data_value_payload", []string{"components", "examples", "sample", "dataValue"}, true},
		{"property_named_example", []string{"components", "schemas", "Foo", "properties", "example"}, false},
		{"property_named_examples_value", []string{"components", "schemas", "Foo", "properties", "examples", "value"}, false},
		{"real_example_after_property_example", []string{"components", "schemas", "Foo", "properties", "example", "example"}, false},
		{"path_item_fragment_parameter_example_payload", []string{"parameters", "example", "owner"}, true},
		{"path_item_fragment_parameter_examples_value", []string{"parameters", "examples", "sample", "value"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, UnderOpenAPIExamplePayloadPath(tt.path))
		})
	}
}

func TestIsOpenAPIExampleKeywordSegment(t *testing.T) {
	path := []string{"components", "examples", "ReusableExample"}

	tests := []struct {
		name string
		idx  int
		want bool
	}{
		{"negative index", -1, false},
		{"index too large", len(path), false},
		{"examples keyword", 1, true},
		{"non keyword segment", 2, false},
		{"property named example", 2, false},
	}

	propertyPath := []string{"components", "schemas", "Foo", "properties", "example"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetPath := path
			if tt.name == "property named example" {
				targetPath = propertyPath
			}
			assert.Equal(t, tt.want, isOpenAPIExampleKeywordSegment(targetPath, tt.idx))
		})
	}
}

func TestUnderSchemaInstanceValuePath(t *testing.T) {
	var seq, mapping, schema, catalog, swaggerParameters yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("[{$ref: '#/a'}]"), &seq))
	require.NoError(t, yaml.Unmarshal([]byte("{named: {$ref: '#/a'}}"), &mapping))
	require.NoError(t, yaml.Unmarshal([]byte("{type: object, default: {$ref: '#/a'}}"), &schema))
	require.NoError(t, yaml.Unmarshal([]byte("{default: {type: object}, other: {type: string}}"), &catalog))
	require.NoError(t, yaml.Unmarshal([]byte("{default: {name: default, in: body, schema: {$ref: '#/a'}}, type: {name: type, in: query, type: string}}"), &swaggerParameters))
	tests := []struct {
		name   string
		path   []string
		node   *yaml.Node
		parent *yaml.Node
		want   bool
	}{
		{"empty", nil, nil, nil, false},
		{"enum", []string{"components", "schemas", "S", "enum"}, nil, nil, true},
		{"const", []string{"components", "schemas", "S", "const"}, nil, nil, true},
		{"default", []string{"components", "schemas", "S", "default"}, nil, nil, true},
		{"below_default", []string{"components", "schemas", "S", "default", "nested"}, nil, nil, true},
		{"items_enum", []string{"components", "schemas", "S", "items", "enum"}, nil, nil, true},
		{"all_of_element_default", []string{"components", "schemas", "S", "allOf", "default"}, nil, nil, true},
		{"all_of_marker_default", []string{"components", "schemas", "S", "allOf", "[]", "default"}, nil, nil, true},
		{"all_of_index_default", []string{"components", "schemas", "S", "allOf", "0", "default"}, nil, nil, true},
		{"examples_array", []string{"components", "schemas", "S", "examples"}, seq.Content[0], nil, true},
		{"examples_map", []string{"content", "application/json", "examples"}, mapping.Content[0], nil, false},
		{"examples_unknown_node", []string{"components", "schemas", "S", "examples"}, nil, nil, false},
		{"example_object_named_brackets", []string{"content", "application/json", "examples", "[]"}, nil, mapping.Content[0], false},
		{"example_object_named_default", []string{"content", "application/json", "examples", "default"}, nil, nil, false},
		{"schema_named_default", []string{"components", "schemas", "default"}, nil, nil, false},
		{"property_named_default", []string{"components", "schemas", "S", "properties", "default"}, nil, nil, false},
		{"property_named_enum", []string{"components", "schemas", "S", "properties", "enum"}, nil, nil, false},
		{"property_named_const", []string{"components", "schemas", "S", "properties", "const"}, nil, nil, false},
		{"default_response", []string{"paths", "~1pets", "get", "responses", "default"}, nil, nil, false},
		{"default_of_property_named_default", []string{"components", "schemas", "S", "properties", "default", "default"}, nil, nil, true},
		{"swagger_parameter_named_default", []string{"parameters", "default"}, nil, nil, false},
		{"swagger_parameter_named_default_in_map", []string{"parameters", "default"}, nil, swaggerParameters.Content[0], false},
		{"operation_parameter_schema_default", []string{"paths", "~1pets", "get", "parameters", "schema", "default"}, nil, nil, true},
		{"default_response_schema_default", []string{"paths", "~1pets", "get", "responses", "default", "content", "application/json", "schema", "default"}, nil, nil, true},
		{"item_schema_default", []string{"paths", "~1pets", "get", "responses", "200", "content", "application/jsonl", "itemSchema", "default"}, nil, nil, true},
		{"property_named_item_schema", []string{"components", "schemas", "S", "properties", "itemSchema"}, nil, nil, false},
		{"additional_operation_named_default", []string{"paths", "~1pets", "additionalOperations", "default"}, nil, nil, false},
		{"root_default_unknown_parent", []string{"default"}, nil, nil, false},
		{"root_default_of_schema_file", []string{"default"}, nil, schema.Content[0], true},
		{"root_schema_named_default", []string{"default"}, nil, catalog.Content[0], false},
		{"below_root_schema_named_default", []string{"default", "properties", "pet"}, nil, nil, false},
		{"default_of_root_schema_named_default", []string{"default", "default"}, nil, schema.Content[0], true},
		{"sequence_element_is_not_a_key", []string{"components", "schemas", "S", "default"}, nil, seq.Content[0], false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, UnderSchemaInstanceValuePath(tt.path, tt.node, tt.parent))
		})
	}
}

func TestIsOpenAPIExtensionSegment(t *testing.T) {
	tests := []struct {
		name string
		path []string
		want bool
	}{
		{"not_extension", []string{"components", "schemas", "S", "properties"}, false},
		{"schema_extension", []string{"components", "schemas", "S", "x-hello"}, true},
		{"property_named_extension", []string{"components", "schemas", "S", "properties", "x-name"}, false},
		{"schema_named_extension", []string{"components", "schemas", "x-name"}, false},
		{"responses_extension", []string{"paths", "~1pets", "get", "responses", "x-notes"}, true},
		{"path_item_fragment_responses_extension", []string{"get", "responses", "x-notes"}, true},
		{"swagger_response_definition_named_extension", []string{"responses", "x-shared"}, false},
		{"below_swagger_response_definition_named_extension", []string{"responses", "x-shared", "schema"}, false},
		{"component_response_named_extension", []string{"components", "responses", "x-name"}, false},
		{"example_named_extension", []string{"content", "application/json", "examples", "x-name"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsOpenAPIExtensionSegment(tt.path, len(tt.path)-1))
		})
	}
	assert.False(t, IsOpenAPIExtensionSegment([]string{"x-a"}, 1))
	assert.False(t, IsOpenAPIExtensionSegment([]string{"x-a"}, -1))
}

func TestHasSchemaKeyword(t *testing.T) {
	var schema, alias, seq yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("{title: T, properties: {}}"), &schema))
	require.NoError(t, yaml.Unmarshal([]byte("a: &s {type: string}\nb: *s"), &alias))
	require.NoError(t, yaml.Unmarshal([]byte("[type]"), &seq))
	assert.True(t, hasSchemaKeyword(schema.Content[0]))
	assert.True(t, hasSchemaKeyword(alias.Content[0].Content[3]))
	assert.False(t, hasSchemaKeyword(seq.Content[0]))
	assert.False(t, hasSchemaKeyword(nil))
	assert.False(t, hasSchemaKeyword(&yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{{Value: "type"}, nil}}))

	tests := []struct {
		name string
		yaml string
		want bool
	}{
		{"type_scalar", "{type: string}", true},
		{"format_scalar", "{format: date}", true},
		{"type_list", "{type: [string, 'null']}", true},
		{"required_list", "{required: [a]}", true},
		{"swagger_parameter_required_flag", "{name: q, in: query, type: string, required: true}", true},
		{"required_flag_alone", "{name: q, in: query, required: true}", false},
		{"all_of_list", "{allOf: [{type: string}]}", true},
		{"items_boolean", "{items: false}", true},
		{"catalog_entry_named_type", "{default: {type: object}, type: {name: type, in: query}}", false},
		{"catalog_entry_named_format", "{default: {type: object}, format: {type: string}}", false},
		{"catalog_entry_named_required", "{default: {type: object}, required: {type: string}}", false},
		{"catalog_entry_named_all_of", "{default: {type: object}, allOf: {type: string}}", false},
		{"catalog_entry_named_ref", "{default: {type: object}, $ref: {type: string}}", false},
		{"catalog_entry_named_items_and_type", "{items: {type: object}, type: {type: string}}", false},
		{"merged_type", "{<<: {type: object}, default: {$ref: '#/a'}}", true},
		{"merged_catalog_entry_named_type", "{<<: {type: {name: type, in: query}}, default: {type: object}}", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var node yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(tt.yaml), &node))
			assert.Equal(t, tt.want, hasSchemaKeyword(node.Content[0]))
		})
	}

	var anchored yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("base: &b {type: object}\nthing: {<<: *b, default: {$ref: '#/a'}}"), &anchored))
	assert.True(t, hasSchemaKeyword(anchored.Content[0].Content[3]), "keywords merged from an anchor count")
}

func TestHasSchemaKeyword_NoAllocationsWithoutMergeKeys(t *testing.T) {
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("{description: d, default: {$ref: '#/a'}, title: t, type: object}"), &node))
	mapping := node.Content[0]
	assert.Zero(t, testing.AllocsPerRun(100, func() { hasSchemaKeyword(mapping) }))
}

func TestIsOpenAPIKeywordSegment_OutOfRange(t *testing.T) {
	path := []string{"components", "schemas", "S", "default"}
	assert.False(t, isOpenAPIKeywordSegment(path, -1))
	assert.False(t, isOpenAPIKeywordSegment(path, len(path)))
	assert.True(t, isOpenAPIKeywordSegment(path, 3))
}

func TestIsLiteralValue(t *testing.T) {
	var refOnly, refWithSibling, payload, schema yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("{$ref: '#/components/examples/E'}"), &refOnly))
	require.NoError(t, yaml.Unmarshal([]byte("{$ref: '#/a', note: n}"), &refWithSibling))
	require.NoError(t, yaml.Unmarshal([]byte("{id: 1}"), &payload))
	require.NoError(t, yaml.Unmarshal([]byte("{type: object}"), &schema))
	tests := []struct {
		name   string
		path   []string
		node   *yaml.Node
		parent *yaml.Node
		want   bool
	}{
		{"empty", nil, nil, nil, false},
		{"structure", []string{"components", "schemas", "S", "properties"}, payload.Content[0], nil, false},
		{"example_payload", []string{"components", "schemas", "S", "example", "nested"}, payload.Content[0], nil, true},
		{"example_object_value", []string{"components", "examples", "E", "value"}, payload.Content[0], nil, true},
		// OAS types example as Any, so a $ref there is data by the spec. libopenapi deliberately
		// keeps a $ref-only wrapper as a reference so bundling can share examples (see
		// UnderOpenAPIExamplePayloadPath); this case pins that extension, not spec behaviour.
		{"direct_example_ref_wrapper", []string{"components", "schemas", "S", "example"}, refOnly.Content[0], nil, false},
		{"direct_example_ref_with_sibling", []string{"components", "schemas", "S", "example"}, refWithSibling.Content[0], nil, true},
		{"direct_example_payload", []string{"components", "schemas", "S", "example"}, payload.Content[0], nil, true},
		{"property_named_example", []string{"components", "schemas", "S", "properties", "example"}, payload.Content[0], nil, false},
		{"schema_default", []string{"components", "schemas", "S", "default"}, refOnly.Content[0], schema.Content[0], true},
		{"property_named_default", []string{"components", "schemas", "S", "properties", "default"}, refOnly.Content[0], nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsLiteralValue(tt.path, tt.node, tt.parent))
		})
	}
}

func TestIsDirectOpenAPIExampleRefNode(t *testing.T) {
	var merged yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("base: &b {$ref: '#/a'}\nwrapper: {<<: *b}"), &merged))
	assert.True(t, isDirectOpenAPIExampleRefNode(merged.Content[0].Content[1]))
	assert.True(t, isDirectOpenAPIExampleRefNode(merged.Content[0].Content[3]), "a $ref merged with << counts")
	assert.False(t, isDirectOpenAPIExampleRefNode(nil))
	assert.False(t, isDirectOpenAPIExampleRefNode(&yaml.Node{Kind: yaml.ScalarNode, Value: "x"}))
}
