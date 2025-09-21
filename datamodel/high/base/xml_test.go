// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func ExampleNewXML() {
	// create an example schema object
	// this can be either JSON or YAML.
	yml := `
namespace: https://pb33f.io/schema
name: something
attribute: true
prefix: sample
wrapped: true`

	// unmarshal raw bytes
	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	// build out the low-level model
	var lowXML lowbase.XML
	_ = lowmodel.BuildModel(node.Content[0], &lowXML)
	_ = lowXML.Build(node.Content[0], nil)

	// build the high level tag
	highXML := NewXML(&lowXML)

	// print out the XML namespace
	fmt.Print(highXML.Namespace)
	// Output: https://pb33f.io/schema
}

func TestNewXML_DeprecatedAttributeWarningOpenAPI32(t *testing.T) {
	// Test for lines 50-57: Deprecated attribute field warning in OpenAPI 3.2+
	yml := `openapi: 3.2.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: object
      xml:
        namespace: https://pb33f.io/schema
        name: something
        attribute: true
        prefix: sample
        wrapped: true`

	// unmarshal raw bytes
	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	// Create config with logger
	logger := &testLogHandler{}
	config := &datamodel.DocumentConfiguration{
		Logger: slog.New(logger),
	}

	// Build spec info
	specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

	// Create index with config
	idxConfig := &index.SpecIndexConfig{
		SpecInfo: specInfo,
		Logger:   config.Logger,
	}
	idx := index.NewSpecIndexWithConfig(&node, idxConfig)

	// Navigate to the XML node: openapi->components->schemas->Test->xml
	xmlNode := node.Content[0].Content[5].Content[1].Content[1].Content[3]

	// build out the low-level model
	var lowXML lowbase.XML
	_ = lowmodel.BuildModel(xmlNode, &lowXML)
	_ = lowXML.Build(xmlNode, idx)

	// build the high level XML
	highXML := NewXML(&lowXML)

	// Check that the values are correct
	assert.Equal(t, "https://pb33f.io/schema", highXML.Namespace)
	assert.Equal(t, "something", highXML.Name)
	assert.True(t, highXML.Attribute)
	assert.Equal(t, "", highXML.NodeType) // NodeType should be empty

	// Check that warning was logged
	assert.Len(t, logger.messages, 1)
	assert.Contains(t, logger.messages[0], "XML 'attribute' field is deprecated in OpenAPI 3.2+")
}

func TestNewXML_DeprecatedAttributeNoWarningWithNodeType(t *testing.T) {
	// Test that no warning is logged if nodeType is present
	yml := `openapi: 3.2.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: object
      xml:
        namespace: https://pb33f.io/schema
        name: something
        attribute: true
        nodeType: attribute
        prefix: sample
        wrapped: true`

	// unmarshal raw bytes
	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	// Create config with logger
	logger := &testLogHandler{}
	config := &datamodel.DocumentConfiguration{
		Logger: slog.New(logger),
	}

	// Build spec info
	specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

	// Create index with config
	idxConfig := &index.SpecIndexConfig{
		SpecInfo: specInfo,
		Logger:   config.Logger,
	}
	idx := index.NewSpecIndexWithConfig(&node, idxConfig)

	// Navigate to the XML node: openapi->components->schemas->Test->xml
	xmlNode := node.Content[0].Content[5].Content[1].Content[1].Content[3]

	// build out the low-level model
	var lowXML lowbase.XML
	_ = lowmodel.BuildModel(xmlNode, &lowXML)
	_ = lowXML.Build(xmlNode, idx)

	// build the high level XML
	highXML := NewXML(&lowXML)

	// Check that the values are correct
	assert.Equal(t, "https://pb33f.io/schema", highXML.Namespace)
	assert.Equal(t, "something", highXML.Name)
	assert.True(t, highXML.Attribute)
	assert.Equal(t, "attribute", highXML.NodeType)

	// Check that NO warning was logged since nodeType is present
	assert.Len(t, logger.messages, 0)
}

func TestNewXML_NoWarningForOlderVersions(t *testing.T) {
	// Test that no warning is logged for OpenAPI versions < 3.2
	yml := `openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: object
      xml:
        namespace: https://pb33f.io/schema
        name: something
        attribute: true
        prefix: sample
        wrapped: true`

	// unmarshal raw bytes
	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	// Create config with logger
	logger := &testLogHandler{}
	config := &datamodel.DocumentConfiguration{
		Logger: slog.New(logger),
	}

	// Build spec info
	specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

	// Create index with config
	idxConfig := &index.SpecIndexConfig{
		SpecInfo: specInfo,
		Logger:   config.Logger,
	}
	idx := index.NewSpecIndexWithConfig(&node, idxConfig)

	// Navigate to the XML node: openapi->components->schemas->Test->xml
	xmlNode := node.Content[0].Content[5].Content[1].Content[1].Content[3]

	// build out the low-level model
	var lowXML lowbase.XML
	_ = lowmodel.BuildModel(xmlNode, &lowXML)
	_ = lowXML.Build(xmlNode, idx)

	// build the high level XML
	highXML := NewXML(&lowXML)

	// Check that the values are correct
	assert.Equal(t, "https://pb33f.io/schema", highXML.Namespace)
	assert.Equal(t, "something", highXML.Name)
	assert.True(t, highXML.Attribute)

	// Check that NO warning was logged for older version
	assert.Len(t, logger.messages, 0)
}

// testLogHandler is a simple handler for testing log output
type testLogHandler struct {
	messages []string
}

func (h *testLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (h *testLogHandler) Handle(ctx context.Context, r slog.Record) error {
	h.messages = append(h.messages, r.Message)
	return nil
}

func (h *testLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *testLogHandler) WithGroup(name string) slog.Handler {
	return h
}

func TestNewXML_WithNodeType(t *testing.T) {
	// test OpenAPI 3.2+ nodeType field
	yml := `namespace: https://pb33f.io/schema
name: something
nodeType: element
prefix: sample
wrapped: true`

	// unmarshal raw bytes
	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	// build out the low-level model
	var lowXML lowbase.XML
	_ = lowmodel.BuildModel(node.Content[0], &lowXML)
	_ = lowXML.Build(node.Content[0], nil)

	// build the high level XML
	highXML := NewXML(&lowXML)

	assert.Equal(t, "https://pb33f.io/schema", highXML.Namespace)
	assert.Equal(t, "something", highXML.Name)
	assert.Equal(t, "element", highXML.NodeType)
	assert.Equal(t, "sample", highXML.Prefix)
	assert.True(t, highXML.Wrapped)
}

func TestNewXML_NodeTypeValues(t *testing.T) {
	// test different nodeType values
	testCases := []struct {
		nodeType string
		expected string
	}{
		{"attribute", "attribute"},
		{"element", "element"},
		{"text", "text"},
	}

	for _, tc := range testCases {
		yml := fmt.Sprintf(`name: test
nodeType: %s`, tc.nodeType)

		var node yaml.Node
		_ = yaml.Unmarshal([]byte(yml), &node)

		var lowXML lowbase.XML
		_ = lowmodel.BuildModel(node.Content[0], &lowXML)
		_ = lowXML.Build(node.Content[0], nil)

		highXML := NewXML(&lowXML)
		assert.Equal(t, tc.expected, highXML.NodeType)
	}
}

func TestContact_Render(t *testing.T) {
	// create an example schema object
	// this can be either JSON or YAML.
	yml := `namespace: https://pb33f.io/schema
name: something
attribute: true
prefix: sample
wrapped: true`

	// unmarshal raw bytes
	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	// build out the low-level model
	var lowXML lowbase.XML
	_ = lowmodel.BuildModel(node.Content[0], &lowXML)
	_ = lowXML.Build(node.Content[0], nil)

	// build the high level tag
	highXML := NewXML(&lowXML)

	// print out the XML doc
	highXMLBytes, _ := highXML.Render()
	assert.Equal(t, yml, strings.TrimSpace(string(highXMLBytes)))

	highXML.Attribute = false
	highXMLBytes, _ = highXML.Render()
	assert.NotEqual(t, yml, strings.TrimSpace(string(highXMLBytes)))
}
