// Copyright 2025 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestNewSiblingRefTransformer(t *testing.T) {
	config := index.CreateOpenAPIIndexConfig()
	config.TransformSiblingRefs = true
	var rootNode yaml.Node
	idx := index.NewSpecIndexWithConfig(&rootNode, config)

	transformer := NewSiblingRefTransformer(idx)
	assert.NotNil(t, transformer)
	assert.Equal(t, idx, transformer.index)
}

func TestSiblingRefTransformer_ExtractSiblingProperties(t *testing.T) {
	config := index.CreateOpenAPIIndexConfig()
	config.TransformSiblingRefs = true
	var rootNode yaml.Node
	idx := index.NewSpecIndexWithConfig(&rootNode, config)
	transformer := NewSiblingRefTransformer(idx)

	t.Run("simple sibling properties", func(t *testing.T) {
		yml := `title: "Custom Title"
description: "Custom Description"
$ref: "#/components/schemas/Base"`

		var node yaml.Node
		_ = yaml.Unmarshal([]byte(yml), &node)
		// get the actual content node (document node contains the content)
		actualNode := node.Content[0]

		siblings, refValue := transformer.ExtractSiblingProperties(actualNode)

		assert.Equal(t, "#/components/schemas/Base", refValue)
		assert.Len(t, siblings, 2)
		assert.Contains(t, siblings, "title")
		assert.Contains(t, siblings, "description")
		if siblings["title"] != nil {
			assert.Equal(t, "Custom Title", siblings["title"].Value)
		}
		if siblings["description"] != nil {
			assert.Equal(t, "Custom Description", siblings["description"].Value)
		}
	})

	t.Run("only ref no siblings", func(t *testing.T) {
		yml := `$ref: "#/components/schemas/Base"`

		var node yaml.Node
		_ = yaml.Unmarshal([]byte(yml), &node)
		// get the actual content node (document node contains the content)
		actualNode := node.Content[0]

		siblings, refValue := transformer.ExtractSiblingProperties(actualNode)

		assert.Empty(t, refValue)
		assert.Empty(t, siblings)
	})

	t.Run("no ref only properties", func(t *testing.T) {
		yml := `title: "Custom Title"
description: "Custom Description"`

		var node yaml.Node
		_ = yaml.Unmarshal([]byte(yml), &node)
		// get the actual content node (document node contains the content)
		actualNode := node.Content[0]

		siblings, refValue := transformer.ExtractSiblingProperties(actualNode)

		assert.Empty(t, refValue)
		assert.Empty(t, siblings)
	})

	t.Run("various property types", func(t *testing.T) {
		yml := `title: "String Value"
nullable: true
example: {"key": "value"}
enum: ["one", "two"]
$ref: "#/components/schemas/Base"`

		var node yaml.Node
		_ = yaml.Unmarshal([]byte(yml), &node)
		// get the actual content node (document node contains the content)
		actualNode := node.Content[0]

		siblings, refValue := transformer.ExtractSiblingProperties(actualNode)

		assert.Equal(t, "#/components/schemas/Base", refValue)
		assert.Len(t, siblings, 4)
		assert.Contains(t, siblings, "title")
		assert.Contains(t, siblings, "nullable")
		assert.Contains(t, siblings, "example")
		assert.Contains(t, siblings, "enum")
	})
}

func TestSiblingRefTransformer_CreateAllOfStructure(t *testing.T) {
	config := index.CreateOpenAPIIndexConfig()
	config.TransformSiblingRefs = true
	var rootNode yaml.Node
	idx := index.NewSpecIndexWithConfig(&rootNode, config)
	transformer := NewSiblingRefTransformer(idx)

	t.Run("create allOf with title and description", func(t *testing.T) {
		siblings := map[string]*yaml.Node{
			"title":       {Kind: yaml.ScalarNode, Value: "Custom Title"},
			"description": {Kind: yaml.ScalarNode, Value: "Custom Description"},
		}
		refValue := "#/components/schemas/Base"

		result := transformer.CreateAllOfStructure(refValue, siblings)

		assert.NotNil(t, result)
		assert.Equal(t, yaml.MappingNode, result.Kind)
		assert.Len(t, result.Content, 2)

		// check allOf key
		assert.Equal(t, "allOf", result.Content[0].Value)

		// check allOf array
		allOfArray := result.Content[1]
		assert.Equal(t, yaml.SequenceNode, allOfArray.Kind)
		assert.Len(t, allOfArray.Content, 2)

		// check first element (sibling properties)
		siblingSchema := allOfArray.Content[0]
		assert.Equal(t, yaml.MappingNode, siblingSchema.Kind)
		assert.Len(t, siblingSchema.Content, 4) // 2 properties * 2 (key+value)

		// check second element (ref)
		refSchema := allOfArray.Content[1]
		assert.Equal(t, yaml.MappingNode, refSchema.Kind)
		assert.Len(t, refSchema.Content, 2)
		assert.Equal(t, "$ref", refSchema.Content[0].Value)
		assert.Equal(t, "#/components/schemas/Base", refSchema.Content[1].Value)
	})

	t.Run("create allOf with empty siblings", func(t *testing.T) {
		siblings := map[string]*yaml.Node{}
		refValue := "#/components/schemas/Base"

		result := transformer.CreateAllOfStructure(refValue, siblings)

		assert.NotNil(t, result)
		// should still create structure but with only ref element
		allOfArray := result.Content[1]
		assert.Len(t, allOfArray.Content, 1) // only ref schema
		assert.Equal(t, "$ref", allOfArray.Content[0].Content[0].Value)
	})
}

func TestSiblingRefTransformer_TransformSiblingRef(t *testing.T) {
	config := index.CreateOpenAPIIndexConfig()
	config.TransformSiblingRefs = true
	var rootNode yaml.Node
	idx := index.NewSpecIndexWithConfig(&rootNode, config)
	transformer := NewSiblingRefTransformer(idx)

	t.Run("transform valid sibling ref", func(t *testing.T) {
		yml := `title: "Custom Title"
description: "Custom Description"
$ref: "#/components/schemas/Base"`

		var node yaml.Node
		_ = yaml.Unmarshal([]byte(yml), &node)
		actualNode := node.Content[0]

		result, err := transformer.TransformSiblingRef(actualNode)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, yaml.MappingNode, result.Kind)

		// verify allOf structure was created
		assert.Equal(t, "allOf", result.Content[0].Value)
		allOfArray := result.Content[1]
		assert.Len(t, allOfArray.Content, 2)
	})

	t.Run("no transformation for ref only", func(t *testing.T) {
		yml := `$ref: "#/components/schemas/Base"`

		var node yaml.Node
		_ = yaml.Unmarshal([]byte(yml), &node)
		actualNode := node.Content[0]
		original := actualNode

		result, err := transformer.TransformSiblingRef(actualNode)

		assert.NoError(t, err)
		assert.Equal(t, original, result) // should return original node unchanged
	})

	t.Run("no transformation for non-ref schema", func(t *testing.T) {
		yml := `type: object
properties:
  id:
    type: string`

		var node yaml.Node
		_ = yaml.Unmarshal([]byte(yml), &node)
		actualNode := node.Content[0]
		original := actualNode

		result, err := transformer.TransformSiblingRef(actualNode)

		assert.NoError(t, err)
		assert.Equal(t, original, result) // should return original node unchanged
	})

	t.Run("handle nil node", func(t *testing.T) {
		result, err := transformer.TransformSiblingRef(nil)

		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestSiblingRefTransformer_ShouldTransform(t *testing.T) {
	t.Run("should transform when enabled and has siblings", func(t *testing.T) {
		config := index.CreateOpenAPIIndexConfig()
		config.TransformSiblingRefs = true
		var rootNode yaml.Node
		idx := index.NewSpecIndexWithConfig(&rootNode, config)
		transformer := NewSiblingRefTransformer(idx)

		yml := `title: "Custom Title"
$ref: "#/components/schemas/Base"`

		var node yaml.Node
		_ = yaml.Unmarshal([]byte(yml), &node)
		actualNode := node.Content[0]

		should := transformer.ShouldTransform(actualNode)
		assert.True(t, should)
	})

	t.Run("should not transform when disabled", func(t *testing.T) {
		config := index.CreateOpenAPIIndexConfig()
		config.TransformSiblingRefs = false
		var rootNode yaml.Node
		idx := index.NewSpecIndexWithConfig(&rootNode, config)
		transformer := NewSiblingRefTransformer(idx)

		yml := `title: "Custom Title"
$ref: "#/components/schemas/Base"`

		var node yaml.Node
		_ = yaml.Unmarshal([]byte(yml), &node)
		actualNode := node.Content[0]

		should := transformer.ShouldTransform(actualNode)
		assert.False(t, should)
	})

	t.Run("should not transform when no siblings", func(t *testing.T) {
		config := index.CreateOpenAPIIndexConfig()
		config.TransformSiblingRefs = true
		var rootNode yaml.Node
		idx := index.NewSpecIndexWithConfig(&rootNode, config)
		transformer := NewSiblingRefTransformer(idx)

		yml := `$ref: "#/components/schemas/Base"`

		var node yaml.Node
		_ = yaml.Unmarshal([]byte(yml), &node)
		actualNode := node.Content[0]

		should := transformer.ShouldTransform(actualNode)
		assert.False(t, should)
	})

	t.Run("should handle nil index", func(t *testing.T) {
		transformer := NewSiblingRefTransformer(nil)

		yml := `title: "Custom Title"
$ref: "#/components/schemas/Base"`

		var node yaml.Node
		_ = yaml.Unmarshal([]byte(yml), &node)
		actualNode := node.Content[0]

		should := transformer.ShouldTransform(actualNode)
		assert.False(t, should)
	})
}

func TestSiblingRefTransformer_IntegrationWorks(t *testing.T) {
	t.Run("transformation creates valid allOf structure", func(t *testing.T) {
		yml := `title: "Custom Title"
description: "Custom Description"
$ref: "#/components/schemas/Base"`

		var node yaml.Node
		_ = yaml.Unmarshal([]byte(yml), &node)
		actualNode := node.Content[0]

		config := index.CreateOpenAPIIndexConfig()
		config.TransformSiblingRefs = true
		var testRootNode yaml.Node
		idx := index.NewSpecIndexWithConfig(&testRootNode, config)

		transformer := NewSiblingRefTransformer(idx)

		// verify transformation creates correct structure
		transformed, err := transformer.TransformSiblingRef(actualNode)
		assert.NoError(t, err)
		assert.Equal(t, "allOf", transformed.Content[0].Value)

		// verify allOf array structure
		allOfArray := transformed.Content[1]
		assert.Equal(t, yaml.SequenceNode, allOfArray.Kind)
		assert.Len(t, allOfArray.Content, 2)

		// verify first element has sibling properties
		firstElement := allOfArray.Content[0]
		assert.Equal(t, yaml.MappingNode, firstElement.Kind)
		hasTitle := false
		hasDescription := false
		for i := 0; i < len(firstElement.Content); i += 2 {
			if firstElement.Content[i].Value == "title" {
				hasTitle = true
				assert.Equal(t, "Custom Title", firstElement.Content[i+1].Value)
			}
			if firstElement.Content[i].Value == "description" {
				hasDescription = true
				assert.Equal(t, "Custom Description", firstElement.Content[i+1].Value)
			}
		}
		assert.True(t, hasTitle)
		assert.True(t, hasDescription)

		// verify second element is the reference
		secondElement := allOfArray.Content[1]
		assert.Equal(t, yaml.MappingNode, secondElement.Kind)
		assert.Len(t, secondElement.Content, 2)
		assert.Equal(t, "$ref", secondElement.Content[0].Value)
		assert.Equal(t, "#/components/schemas/Base", secondElement.Content[1].Value)
	})

	t.Run("verify transformation integration point works", func(t *testing.T) {
		yml := `title: "Integration Test"
$ref: "#/components/schemas/Base"`

		var node yaml.Node
		_ = yaml.Unmarshal([]byte(yml), &node)
		actualNode := node.Content[0]

		config := index.CreateOpenAPIIndexConfig()
		config.TransformSiblingRefs = true
		var testRootNode yaml.Node
		idx := index.NewSpecIndexWithConfig(&testRootNode, config)

		// verify transformation occurs at SchemaProxy.Build level
		schemaProxy := &SchemaProxy{}
		err := schemaProxy.Build(context.Background(), nil, actualNode, idx)
		assert.NoError(t, err)

		// verify the transformation occurred in the value node
		assert.NotNil(t, schemaProxy.vn)
		if len(schemaProxy.vn.Content) > 0 {
			assert.Equal(t, "allOf", schemaProxy.vn.Content[0].Value, "value node should be transformed to allOf structure")
		}

		// verify transformation flag is set
		assert.NotNil(t, schemaProxy.TransformedRef, "TransformedRef should be set for transformed schemas")
	})

	t.Run("verify no transformation when disabled", func(t *testing.T) {
		yml := `title: "Custom Title"
$ref: "#/components/schemas/Base"`

		var node yaml.Node
		_ = yaml.Unmarshal([]byte(yml), &node)
		actualNode := node.Content[0] // get content node from document

		config := index.CreateOpenAPIIndexConfig()
		config.TransformSiblingRefs = false
		var testRootNode yaml.Node
		idx := index.NewSpecIndexWithConfig(&testRootNode, config)

		// verify transformer correctly detects disabled state
		transformer := NewSiblingRefTransformer(idx)
		shouldTransform := transformer.ShouldTransform(actualNode)
		assert.False(t, shouldTransform, "should not transform when disabled")

		// when disabled, ShouldTransform should return false, so TransformSiblingRef should return original
		result, err := transformer.TransformSiblingRef(actualNode)
		assert.NoError(t, err)
		assert.Equal(t, actualNode, result, "should return original node when transformation disabled")
	})
}

func TestSiblingRefTransformer_copyNode(t *testing.T) {
	config := index.CreateOpenAPIIndexConfig()
	var rootNode yaml.Node
	idx := index.NewSpecIndexWithConfig(&rootNode, config)
	transformer := NewSiblingRefTransformer(idx)

	t.Run("copy simple scalar node", func(t *testing.T) {
		original := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: "test value",
			Line:  10,
			Column: 5,
		}

		copied := transformer.copyNode(original)

		assert.NotSame(t, original, copied)
		assert.Equal(t, original.Kind, copied.Kind)
		assert.Equal(t, original.Value, copied.Value)
		assert.Equal(t, original.Line, copied.Line)
		assert.Equal(t, original.Column, copied.Column)
	})

	t.Run("copy mapping node with content", func(t *testing.T) {
		original := &yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "key1"},
				{Kind: yaml.ScalarNode, Value: "value1"},
				{Kind: yaml.ScalarNode, Value: "key2"},
				{Kind: yaml.ScalarNode, Value: "value2"},
			},
		}

		copied := transformer.copyNode(original)

		assert.NotSame(t, original, copied)
		assert.Equal(t, original.Kind, copied.Kind)
		assert.Len(t, copied.Content, 4)

		// verify content is copied but not same references
		for i, child := range copied.Content {
			assert.NotSame(t, original.Content[i], child)
			assert.Equal(t, original.Content[i].Value, child.Value)
		}
	})

	t.Run("copy nil node", func(t *testing.T) {
		copied := transformer.copyNode(nil)
		assert.Nil(t, copied)
	})
}