// Copyright 2024-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package renderer

import (
	"context"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func createSchemaFromYAML(t *testing.T, yamlStr string) *base.Schema {
	t.Helper()
	var root yaml.Node
	err := yaml.Unmarshal([]byte(yamlStr), &root)
	require.NoError(t, err)
	var lowProxy lowbase.SchemaProxy
	err = lowProxy.Build(context.Background(), &root, root.Content[0], nil)
	require.NoError(t, err)
	lowRef := low.NodeReference[*lowbase.SchemaProxy]{
		Value: &lowProxy,
	}
	highSchema := base.NewSchemaProxy(&lowRef)
	return highSchema.Schema()
}

func TestRenderXML_NilValue(t *testing.T) {
	mg := NewMockGenerator(XML)
	result := mg.RenderXML(nil, nil)
	assert.Nil(t, result)
}

func TestRenderXML_InvalidYAMLNodeDecode(t *testing.T) {
	mg := NewMockGenerator(XML)

	// Unknown node kinds cannot be decoded into native Go values.
	result := mg.RenderXML(&yaml.Node{Kind: 255}, nil)
	assert.Nil(t, result)
}

func TestRenderXML_BasicMap_NoSchema(t *testing.T) {
	mg := NewMockGenerator(XML)
	mg.SetPretty()

	value := map[string]any{
		"name": "test",
		"age":  42,
	}

	result := mg.RenderXML(value, nil)
	require.NotNil(t, result)
	str := string(result)
	assert.Contains(t, str, `<?xml version="1.0" encoding="UTF-8"?>`)
	assert.Contains(t, str, `<root>`)
	assert.Contains(t, str, `</root>`)
	assert.Contains(t, str, `<name>test</name>`)
	assert.Contains(t, str, `<age>42</age>`)

	// Verify it's valid XML
	assertValidXML(t, result)
}

func TestRenderXML_ScalarValues(t *testing.T) {
	mg := NewMockGenerator(XML)

	tests := []struct {
		name     string
		value    any
		expected string
	}{
		{"string", "hello", "<root>hello</root>"},
		{"int", 42, "<root>42</root>"},
		{"float", 3.14, "<root>3.14</root>"},
		{"bool", true, "<root>true</root>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mg.RenderXML(tt.value, nil)
			require.NotNil(t, result)
			assert.Contains(t, string(result), tt.expected)
			assertValidXML(t, result)
		})
	}
}

func TestRenderXML_XMLNameOverride(t *testing.T) {
	mg := NewMockGenerator(XML)

	schema := createSchemaFromYAML(t, `
type: object
xml:
  name: person
properties:
  firstName:
    type: string
    xml:
      name: first-name
  lastName:
    type: string
`)

	value := map[string]any{
		"firstName": "John",
		"lastName":  "Doe",
	}

	result := mg.RenderXML(value, schema)
	require.NotNil(t, result)
	str := string(result)
	assert.Contains(t, str, `<person>`)
	assert.Contains(t, str, `<first-name>John</first-name>`)
	assert.Contains(t, str, `<lastName>Doe</lastName>`)
	assertValidXML(t, result)
}

func TestRenderXML_NodeTypeAttribute(t *testing.T) {
	mg := NewMockGenerator(XML)

	schema := createSchemaFromYAML(t, `
type: object
xml:
  name: item
properties:
  id:
    type: integer
    xml:
      nodeType: attribute
  name:
    type: string
`)

	value := map[string]any{
		"id":   100,
		"name": "Widget",
	}

	result := mg.RenderXML(value, schema)
	require.NotNil(t, result)
	str := string(result)
	assert.Contains(t, str, `id="100"`)
	assert.Contains(t, str, `<name>Widget</name>`)
	assertValidXML(t, result)
}

func TestRenderXML_LegacyAttribute(t *testing.T) {
	mg := NewMockGenerator(XML)

	schema := createSchemaFromYAML(t, `
type: object
xml:
  name: item
properties:
  currency:
    type: string
    xml:
      attribute: true
  amount:
    type: number
`)

	value := map[string]any{
		"currency": "USD",
		"amount":   120.50,
	}

	result := mg.RenderXML(value, schema)
	require.NotNil(t, result)
	str := string(result)
	assert.Contains(t, str, `currency="USD"`)
	assert.Contains(t, str, `<amount>120.5</amount>`)
	assertValidXML(t, result)
}

func TestRenderXML_NodeTypeText(t *testing.T) {
	mg := NewMockGenerator(XML)

	schema := createSchemaFromYAML(t, `
type: object
xml:
  name: amount
properties:
  currency:
    type: string
    xml:
      nodeType: attribute
  value:
    type: number
    xml:
      nodeType: text
`)

	value := map[string]any{
		"currency": "USD",
		"value":    120.50,
	}

	result := mg.RenderXML(value, schema)
	require.NotNil(t, result)
	str := string(result)
	// Should produce <amount currency="USD">120.5</amount>
	assert.Contains(t, str, `currency="USD"`)
	assert.Contains(t, str, `120.5`)
	// Should NOT have <value> wrapper
	assert.NotContains(t, str, `<value>`)
	assertValidXML(t, result)
}

func TestRenderXML_NodeTypeCdata_FallsBackToText(t *testing.T) {
	mg := NewMockGenerator(XML)

	schema := createSchemaFromYAML(t, `
type: object
xml:
  name: note
properties:
  content:
    type: string
    xml:
      nodeType: cdata
`)

	value := map[string]any{
		"content": "Some text content",
	}

	result := mg.RenderXML(value, schema)
	require.NotNil(t, result)
	str := string(result)
	assert.Contains(t, str, `Some text content`)
	assertValidXML(t, result)
}

func TestRenderXML_InvalidXMLNames(t *testing.T) {
	mg := NewMockGenerator(XML)

	value := map[string]any{
		"my key":   "value1",
		"123start": "value2",
		"normal":   "value3",
	}

	result := mg.RenderXML(value, nil)
	require.NotNil(t, result)
	str := string(result)
	assert.Contains(t, str, `<my_key>value1</my_key>`)
	assert.Contains(t, str, `<_123start>value2</_123start>`)
	assert.Contains(t, str, `<normal>value3</normal>`)
	assertValidXML(t, result)
}

func TestRenderXML_NamespaceAndPrefix(t *testing.T) {
	mg := NewMockGenerator(XML)

	schema := createSchemaFromYAML(t, `
type: object
xml:
  name: order
  namespace: http://example.com/schema
  prefix: ex
properties:
  id:
    type: integer
`)

	value := map[string]any{
		"id": 42,
	}

	result := mg.RenderXML(value, schema)
	require.NotNil(t, result)
	str := string(result)
	assert.Contains(t, str, `ex:order`)
	assert.Contains(t, str, `xmlns:ex="http://example.com/schema"`)
	assertValidXML(t, result)
}

func TestRenderXML_AttributePrefixDeclaresNamespace(t *testing.T) {
	mg := NewMockGenerator(XML)

	schema := createSchemaFromYAML(t, `
type: object
xml:
  name: order
properties:
  id:
    type: integer
    xml:
      nodeType: attribute
      prefix: ex
      namespace: http://example.com/attr
  name:
    type: string
`)

	value := map[string]any{
		"id":   42,
		"name": "Widget",
	}

	result := mg.RenderXML(value, schema)
	require.NotNil(t, result)
	str := string(result)
	assert.Contains(t, str, `ex:id="42"`)
	assert.Contains(t, str, `xmlns:ex="http://example.com/attr"`)
	assert.Contains(t, str, `<name>Widget</name>`)
	assertValidXML(t, result)
}

func TestRenderXML_ArrayUnwrapped(t *testing.T) {
	mg := NewMockGenerator(XML)
	mg.SetPretty()

	schema := createSchemaFromYAML(t, `
type: object
xml:
  name: order
properties:
  tags:
    type: array
    items:
      type: string
      xml:
        name: tag
`)

	value := map[string]any{
		"tags": []any{"food", "drink"},
	}

	result := mg.RenderXML(value, schema)
	require.NotNil(t, result)
	str := string(result)
	assert.Contains(t, str, `<tag>food</tag>`)
	assert.Contains(t, str, `<tag>drink</tag>`)
	// Should NOT have a <tags> wrapper
	assert.NotContains(t, str, `<tags>`)
	assertValidXML(t, result)
}

func TestRenderXML_ArrayWrapped(t *testing.T) {
	mg := NewMockGenerator(XML)

	schema := createSchemaFromYAML(t, `
type: object
xml:
  name: order
properties:
  tags:
    type: array
    xml:
      wrapped: true
    items:
      type: string
      xml:
        name: tag
`)

	value := map[string]any{
		"tags": []any{"food", "drink"},
	}

	result := mg.RenderXML(value, schema)
	require.NotNil(t, result)
	str := string(result)
	assert.Contains(t, str, `<tags>`)
	assert.Contains(t, str, `<tag>food</tag>`)
	assert.Contains(t, str, `<tag>drink</tag>`)
	assert.Contains(t, str, `</tags>`)
	assertValidXML(t, result)
}

func TestRenderXML_ArrayWrappedByNodeTypeElement(t *testing.T) {
	mg := NewMockGenerator(XML)

	schema := createSchemaFromYAML(t, `
type: object
xml:
  name: order
properties:
  tags:
    type: array
    xml:
      nodeType: element
    items:
      type: string
      xml:
        name: tag
`)

	value := map[string]any{
		"tags": []any{"food", "drink"},
	}

	result := mg.RenderXML(value, schema)
	require.NotNil(t, result)
	str := string(result)
	assert.Contains(t, str, `<tags>`)
	assert.Contains(t, str, `<tag>food</tag>`)
	assert.Contains(t, str, `<tag>drink</tag>`)
	assert.Contains(t, str, `</tags>`)
	assertValidXML(t, result)
}

func TestRenderXML_ArrayXMLNameDoesNotImplyWrapping(t *testing.T) {
	mg := NewMockGenerator(XML)

	schema := createSchemaFromYAML(t, `
type: object
xml:
  name: order
properties:
  tags:
    type: array
    xml:
      name: collection
    items:
      type: string
      xml:
        name: tag
`)

	value := map[string]any{
		"tags": []any{"food", "drink"},
	}

	result := mg.RenderXML(value, schema)
	require.NotNil(t, result)
	str := string(result)
	assert.NotContains(t, str, `<collection>`)
	assert.Contains(t, str, `<tag>food</tag>`)
	assert.Contains(t, str, `<tag>drink</tag>`)
	assertValidXML(t, result)
}

func TestRenderXML_ArrayItemsDefaultName(t *testing.T) {
	mg := NewMockGenerator(XML)

	schema := createSchemaFromYAML(t, `
type: object
xml:
  name: data
properties:
  items:
    type: array
    xml:
      wrapped: true
    items:
      type: string
`)

	value := map[string]any{
		"items": []any{"a", "b"},
	}

	result := mg.RenderXML(value, schema)
	require.NotNil(t, result)
	str := string(result)
	// Without xml.Name on items, should default to "item"
	assert.Contains(t, str, `<item>a</item>`)
	assert.Contains(t, str, `<item>b</item>`)
	assertValidXML(t, result)
}

func TestRenderXML_NestedObjects(t *testing.T) {
	mg := NewMockGenerator(XML)
	mg.SetPretty()

	value := map[string]any{
		"person": map[string]any{
			"name": "Alice",
			"address": map[string]any{
				"city":    "London",
				"country": "UK",
			},
		},
	}

	result := mg.RenderXML(value, nil)
	require.NotNil(t, result)
	str := string(result)
	assert.Contains(t, str, `<person>`)
	assert.Contains(t, str, `<name>Alice</name>`)
	assert.Contains(t, str, `<address>`)
	assert.Contains(t, str, `<city>London</city>`)
	assert.Contains(t, str, `<country>UK</country>`)
	assertValidXML(t, result)
}

func TestRenderXML_NodeTypeNoneFlattensChildren(t *testing.T) {
	mg := NewMockGenerator(XML)

	schema := createSchemaFromYAML(t, `
type: object
xml:
  name: person
properties:
  profile:
    type: object
    xml:
      nodeType: none
    properties:
      firstName:
        type: string
      city:
        type: string
`)

	value := map[string]any{
		"profile": map[string]any{
			"firstName": "Alice",
			"city":      "London",
			"nickname":  "Al",
		},
	}

	result := mg.RenderXML(value, schema)
	require.NotNil(t, result)
	str := string(result)
	assert.NotContains(t, str, `<profile>`)
	assert.Contains(t, str, `<firstName>Alice</firstName>`)
	assert.Contains(t, str, `<city>London</city>`)
	assert.Contains(t, str, `<nickname>Al</nickname>`)
	assertValidXML(t, result)
}

func TestRenderXML_NodeTypeNoneScalarFallsBackToElement(t *testing.T) {
	mg := NewMockGenerator(XML)

	schema := createSchemaFromYAML(t, `
type: object
xml:
  name: person
properties:
  nickname:
    type: string
    xml:
      nodeType: none
`)

	value := map[string]any{
		"nickname": "Al",
	}

	result := mg.RenderXML(value, schema)
	require.NotNil(t, result)
	str := string(result)
	assert.Contains(t, str, `<nickname>Al</nickname>`)
	assertValidXML(t, result)
}

func TestRenderXML_Escaping(t *testing.T) {
	mg := NewMockGenerator(XML)

	value := map[string]any{
		"text": `<script>alert("xss")</script> & more`,
	}

	result := mg.RenderXML(value, nil)
	require.NotNil(t, result)
	str := string(result)
	// xml.Encoder should escape all special characters
	assert.NotContains(t, str, `<script>`)
	assert.Contains(t, str, `&lt;script&gt;`)
	assertValidXML(t, result)
}

func TestRenderXML_EscapingInAttributes(t *testing.T) {
	mg := NewMockGenerator(XML)

	schema := createSchemaFromYAML(t, `
type: object
xml:
  name: item
properties:
  desc:
    type: string
    xml:
      nodeType: attribute
`)

	value := map[string]any{
		"desc": `value with "quotes" & <angle>`,
	}

	result := mg.RenderXML(value, schema)
	require.NotNil(t, result)
	assertValidXML(t, result)
	str := string(result)
	assert.Contains(t, str, `desc=`)
	// Verify the attribute value is properly escaped (round-trip test via assertValidXML)
}

func TestRenderXML_YamlNodeInput(t *testing.T) {
	mg := NewMockGenerator(XML)

	yamlNode := utils.CreateYamlNode(map[string]any{
		"name": "test",
	})

	result := mg.RenderXML(yamlNode, nil)
	require.NotNil(t, result)
	str := string(result)
	assert.Contains(t, str, `<name>test</name>`)
	assertValidXML(t, result)
}

func TestRenderXML_NilMapValue_IsOmitted(t *testing.T) {
	mg := NewMockGenerator(XML)

	value := map[string]any{
		"name":  nil,
		"other": "value",
	}

	result := mg.RenderXML(value, nil)
	require.NotNil(t, result)
	str := string(result)
	assert.NotContains(t, str, `<name>`)
	assert.Contains(t, str, `<other>value</other>`)
	assertValidXML(t, result)
}

func TestRenderXML_PrettyVsCompact(t *testing.T) {
	value := map[string]any{"name": "test"}

	mgPretty := NewMockGenerator(XML)
	mgPretty.SetPretty()
	prettyResult := mgPretty.RenderXML(value, nil)

	mgCompact := NewMockGenerator(XML)
	compactResult := mgCompact.RenderXML(value, nil)

	// Pretty should have newlines and indentation
	assert.Contains(t, string(prettyResult), "\n")
	// Both should be valid XML
	assertValidXML(t, prettyResult)
	assertValidXML(t, compactResult)
	// Compact should NOT have indentation
	assert.NotContains(t, string(compactResult), "\n  <name>")
}

func TestRenderXML_RootElementFromSchema(t *testing.T) {
	mg := NewMockGenerator(XML)

	schema := createSchemaFromYAML(t, `
type: object
xml:
  name: myRoot
properties:
  id:
    type: integer
`)

	value := map[string]any{"id": 1}

	result := mg.RenderXML(value, schema)
	require.NotNil(t, result)
	assert.Contains(t, string(result), `<myRoot>`)
	assert.Contains(t, string(result), `</myRoot>`)
	assertValidXML(t, result)
}

func TestRenderXML_NilSchema_BasicFallback(t *testing.T) {
	mg := NewMockGenerator(XML)

	value := map[string]any{
		"key": "value",
	}

	result := mg.RenderXML(value, nil)
	require.NotNil(t, result)
	assert.Contains(t, string(result), `<root>`)
	assert.Contains(t, string(result), `<key>value</key>`)
	assertValidXML(t, result)
}

func TestGenerateMock_XML_FromExample(t *testing.T) {
	mg := NewMockGenerator(XML)
	mg.SetPretty()

	mock := createFakeMock(objectFakeMockSchema, nil, map[string]any{
		"coffee": "espresso",
		"herbs":  375,
	})

	result, err := mg.GenerateMock(mock, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	str := string(result)
	assert.Contains(t, str, `<?xml`)
	assert.Contains(t, str, `<coffee>espresso</coffee>`)
	assert.Contains(t, str, `<herbs>375</herbs>`)
	assertValidXML(t, result)
}

func TestGenerateMock_XML_FromNamedExamples(t *testing.T) {
	mg := NewMockGenerator(XML)

	mock := createFakeMock(objectFakeMockSchema, map[string]any{
		"myExample": map[string]any{
			"coffee": "latte",
			"herbs":  400,
		},
	}, nil)

	result, err := mg.GenerateMock(mock, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	str := string(result)
	assert.Contains(t, str, `<coffee>latte</coffee>`)
	assert.Contains(t, str, `<herbs>400</herbs>`)
	assertValidXML(t, result)
}

func TestGenerateMock_XML_FromSchemaFallback(t *testing.T) {
	mg := NewMockGenerator(XML)
	mg.DisableRequiredCheck()

	mock := createFakeMock(objectFakeMockSchema, nil, nil)

	result, err := mg.GenerateMock(mock, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	str := string(result)
	assert.Contains(t, str, `<?xml`)
	assert.Contains(t, str, `<coffee>`)
	assert.Contains(t, str, `<herbs>`)
	assertValidXML(t, result)
}

func TestSanitizeXMLName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"name", "name"},
		{"my key", "my_key"},
		{"123start", "_123start"},
		{"-dash", "_-dash"},
		{"valid.name", "valid.name"},
		{"", "_"},
		{"a b c", "a_b_c"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, sanitizeXMLName(tt.input))
		})
	}
}

func TestRenderXML_NamespaceWithoutPrefix(t *testing.T) {
	mg := NewMockGenerator(XML)

	schema := createSchemaFromYAML(t, `
type: object
xml:
  name: order
  namespace: http://example.com/schema
properties:
  id:
    type: integer
`)

	value := map[string]any{"id": 1}

	result := mg.RenderXML(value, schema)
	require.NotNil(t, result)
	str := string(result)
	assert.Contains(t, str, `xmlns="http://example.com/schema"`)
	assertValidXML(t, result)
}

func TestRenderXML_TopLevelArray(t *testing.T) {
	mg := NewMockGenerator(XML)

	value := []any{"a", "b", "c"}

	result := mg.RenderXML(value, nil)
	require.NotNil(t, result)
	str := string(result)
	assert.Contains(t, str, `<root>`)
	assert.Contains(t, str, `<item>a</item>`)
	assert.Contains(t, str, `<item>b</item>`)
	assert.Contains(t, str, `<item>c</item>`)
	assertValidXML(t, result)
}

func TestRenderXML_TopLevelArray_ItemNameFromSchema(t *testing.T) {
	mg := NewMockGenerator(XML)

	schema := createSchemaFromYAML(t, `
type: array
xml:
  name: tags
items:
  type: string
  xml:
    name: tag
`)

	value := []any{"a", "b", "c"}

	result := mg.RenderXML(value, schema)
	require.NotNil(t, result)
	str := string(result)
	assert.Contains(t, str, `<tags>`)
	assert.Contains(t, str, `<tag>a</tag>`)
	assert.Contains(t, str, `<tag>b</tag>`)
	assert.Contains(t, str, `<tag>c</tag>`)
	assertValidXML(t, result)
}

func TestAppendNamespaceAttr_SkipsEmptyAndDeduplicates(t *testing.T) {
	attrs := appendNamespaceAttr(nil, "ex", "")
	assert.Nil(t, attrs)

	attrs = appendNamespaceAttr(attrs, "ex", "http://example.com/schema")
	require.Len(t, attrs, 1)
	assert.Equal(t, "xmlns:ex", attrs[0].Name.Local)

	attrs = appendNamespaceAttr(attrs, "ex", "http://example.com/schema")
	require.Len(t, attrs, 1)
}

// assertValidXML verifies that the byte slice is valid XML by round-tripping through xml.Decoder.
func assertValidXML(t *testing.T, data []byte) {
	t.Helper()
	d := xml.NewDecoder(strings.NewReader(string(data)))
	for {
		_, err := d.Token()
		if err != nil {
			if err.Error() == "EOF" {
				return
			}
			t.Fatalf("invalid XML: %v\nXML content:\n%s", err, string(data))
		}
	}
}
