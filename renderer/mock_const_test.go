// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package renderer

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/testify/require"
)

func TestMockConstValues(t *testing.T) {
	for _, tc := range []struct{ name, constant, json, tag string }{
		{"false", "false", "false", "!!bool"},
		{"zero", "0", "0", "!!int"},
		{"null", "null", "null", "!!null"},
		{"empty string", `""`, `""`, "!!str"},
		{"string looks boolean", `"false"`, `"false"`, "!!str"},
		{"decimal precision", "12345678901234567890.123456789", "12345678901234567890.123456789", "!!float"},
		{"integer precision", "123456789012345678901234567890", "123456789012345678901234567890", "!!int"},
		{"hex", "0x10", "16", "!!int"},
		{"octal", "012", "10", "!!int"},
		{"leading decimal", "-.5", "-0.5", "!!float"},
		{"trailing decimal", "1.", "1.0", "!!float"},
		{"leading zero decimal", "01.0", "1.0", "!!float"},
		{"leading zero exponent", "01e2", "1e2", "!!float"},
		{"negative leading zero decimal", "-00.50", "-0.50", "!!float"},
		{"zero exponent", "00.0e+2", "0.0e+2", "!!float"},
		{"leading zero precision", "0012345678901234567890.123456789", "12345678901234567890.123456789", "!!float"},
		{"array", `[false, 0, null, ""]`, `[false,0,null,""]`, "!!seq"},
		{"object", `{accepted: false, count: 0, empty: "", missing: null}`, `{"accepted":false,"count":0,"empty":"","missing":null}`, "!!map"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mockable := createFakeMock("const: "+tc.constant, nil, nil)
			before, err := yaml.Marshal(mockable.Schema.Schema().Const)
			require.NoError(t, err)
			output, err := NewMockGenerator(JSON).GenerateMock(mockable, "")
			require.NoError(t, err)
			require.Equal(t, tc.json, string(output))
			output, err = NewMockGenerator(YAML).GenerateMock(mockable, "")
			require.NoError(t, err)
			var root yaml.Node
			require.NoError(t, yaml.Unmarshal(output, &root))
			require.Len(t, root.Content, 1)
			require.Equal(t, tc.tag, root.Content[0].ShortTag())
			after, err := yaml.Marshal(mockable.Schema.Schema().Const)
			require.NoError(t, err)
			require.Equal(t, string(before), string(after))
			values := make(map[string]any)
			require.True(t, CreateRendererUsingDefaultDictionary().DiveIntoSchema(mockable.Schema.Schema(), "result", values, make(map[string]bool), 0))
			encoded, err := json.Marshal(values["result"])
			require.NoError(t, err)
			require.Equal(t, tc.json, string(encoded))
		})
	}
}

func TestMockConstNestedAndExamplePrecedence(t *testing.T) {
	mock := createFakeMock(`type: object
properties:
  accepted:
    type: boolean
    const: false
  omitted:
    const: null
required: [accepted, omitted]`, nil, nil)
	output, err := NewMockGenerator(JSON).GenerateMock(mock, "")
	require.NoError(t, err)
	require.JSONEq(t, `{"accepted":false,"omitted":null}`, string(output))
	output, err = NewMockGenerator(JSON).GenerateMock(createFakeMock("const: false\nexample: true", nil, nil), "")
	require.NoError(t, err)
	require.Equal(t, "true", string(output))
	_, err = NewMockGenerator(JSON).GenerateMock(createFakeMock("{}", nil, nil), "")
	require.Error(t, err)
}

func TestMockConstBudgetsAndMalformedValues(t *testing.T) {
	for _, tc := range []struct {
		name, value, budget string
		options             MockGenerationOptions
	}{
		{"depth", "[[[0]]]", "depth", MockGenerationOptions{MaxMockDepth: 2}},
		{"nodes", "[0,0,0]", "nodes", MockGenerationOptions{MaxMockNodes: 2}},
		{"properties", "{a: 0, b: 0}", "properties", MockGenerationOptions{MaxMockProperties: 1}},
		{"string bytes", `"abcdefghijk"`, "bytes", MockGenerationOptions{MaxMockBytes: 8}},
		{"number bytes", strings.Repeat("9", 10000), "bytes", MockGenerationOptions{MaxMockBytes: 16}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			generator := NewMockGenerator(JSON)
			generator.SetMockGenerationOptions(tc.options)
			output, err := generator.GenerateMock(createFakeMock("const: "+tc.value, nil, nil), "")
			require.Nil(t, output)
			var budget *MockGenerationBudgetError
			require.ErrorAs(t, err, &budget)
			require.Equal(t, tc.budget, budget.Budget)
		})
	}
	cycle := &yaml.Node{Kind: yaml.AliasNode}
	cycle.Alias = cycle
	for name, node := range map[string]*yaml.Node{
		"cycle":          cycle,
		"non-string key": {Kind: yaml.MappingNode, Content: []*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!int", Value: "1"}, {Kind: yaml.ScalarNode, Tag: "!!str", Value: "x"}}},
		"odd mapping":    {Kind: yaml.MappingNode, Content: []*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!str", Value: "x"}}},
		"infinite float": {Kind: yaml.ScalarNode, Tag: "!!float", Value: ".inf"},
		"nil alias":      {Kind: yaml.AliasNode},
		"duplicate key":  {Kind: yaml.MappingNode, Content: []*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!str", Value: "x"}, {Kind: yaml.ScalarNode, Tag: "!!null"}, {Kind: yaml.ScalarNode, Tag: "!!str", Value: "x"}, {Kind: yaml.ScalarNode, Tag: "!!null"}}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := CreateRendererUsingDefaultDictionary().RenderSchemaWithError(&base.Schema{Const: node})
			require.Error(t, err)
		})
	}
}

func TestMockConstReferencedWithSibling(t *testing.T) {
	for _, constant := range []string{"false", "null"} {
		t.Run(constant, func(t *testing.T) {
			schema := componentSchemaForMockTest(t, "Root", `openapi: 3.1.0
info: {title: constant, version: '1'}
paths: {}
components:
  schemas:
    Root:
      $ref: '#/components/schemas/Value'
      description: A retained constant.
    Value:
      const: `+constant)
			output, err := NewMockGenerator(JSON).GenerateMock(schema, "")
			require.NoError(t, err)
			require.Equal(t, constant, string(output))
		})
	}
}

func TestMockConstReferenceSiblingOverridesTypeGeneration(t *testing.T) {
	schema := componentSchemaForMockTest(t, "Root", `openapi: 3.1.0
info: {title: constant, version: '1'}
paths: {}
components:
  schemas:
    Root:
      $ref: '#/components/schemas/Value'
      const: false
    Value:
      type: boolean`)
	output, err := NewMockGenerator(JSON).GenerateMock(schema, "")
	require.NoError(t, err)
	require.Equal(t, "false", string(output))
}

func TestMockConstRejectsMalformedTreesWithoutPartialOutput(t *testing.T) {
	key := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "field"}
	invalid := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "not-an-integer"}
	for name, node := range map[string]*yaml.Node{
		"empty document":       {Kind: yaml.DocumentNode},
		"multiple documents":   {Kind: yaml.DocumentNode, Content: []*yaml.Node{key, key}},
		"invalid boolean":      {Kind: yaml.ScalarNode, Tag: "!!bool", Value: "yes"},
		"invalid integer":      invalid,
		"unsupported scalar":   {Kind: yaml.ScalarNode, Tag: "!!timestamp", Value: "2026-11-07"},
		"unsupported node":     {Kind: 99},
		"invalid object child": {Kind: yaml.MappingNode, Content: []*yaml.Node{key, invalid}},
		"invalid array child":  {Kind: yaml.SequenceNode, Content: []*yaml.Node{invalid}},
	} {
		t.Run(name, func(t *testing.T) {
			schema := &base.Schema{Const: node}
			value, err := CreateRendererUsingDefaultDictionary().RenderSchemaWithError(schema)
			require.Error(t, err)
			require.Nil(t, value)
			values := make(map[string]any)
			require.False(t, CreateRendererUsingDefaultDictionary().DiveIntoSchema(schema, "result", values, make(map[string]bool), 0))
			require.Empty(t, values)
		})
	}
}

func TestMockConstDocumentAndAliasValues(t *testing.T) {
	constant := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "TRUE"}
	for _, node := range []*yaml.Node{constant, {Kind: yaml.DocumentNode, Content: []*yaml.Node{constant}}, {Kind: yaml.AliasNode, Alias: constant}} {
		value, err := CreateRendererUsingDefaultDictionary().RenderSchemaWithError(&base.Schema{Const: node})
		require.NoError(t, err)
		require.Equal(t, true, value)
	}
}

func TestMockConstContainerOverheadUsesByteBudget(t *testing.T) {
	for _, kind := range []yaml.Kind{yaml.MappingNode, yaml.SequenceNode} {
		generator := NewMockGenerator(JSON)
		generator.SetMockGenerationOptions(MockGenerationOptions{MaxMockBytes: 1})
		output, err := generator.GenerateMock(&base.Schema{Const: &yaml.Node{Kind: kind}}, "")
		require.Nil(t, output)
		var budget *MockGenerationBudgetError
		require.ErrorAs(t, err, &budget)
		require.Equal(t, "bytes", budget.Budget)
	}
}

func TestMockConstRejectsLargeNumericTextBeforeNormalization(t *testing.T) {
	generator := NewMockGenerator(JSON)
	generator.SetMockGenerationOptions(MockGenerationOptions{MaxMockBytes: 16})
	output, err := generator.GenerateMock(&base.Schema{Const: &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "0x" + strings.Repeat("f", 10000)}}, "")
	require.Nil(t, output)
	var budget *MockGenerationBudgetError
	require.ErrorAs(t, err, &budget)
	require.Equal(t, "bytes", budget.Budget)
}
