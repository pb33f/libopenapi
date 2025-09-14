// Copyright 2025 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestNewPropertyMerger(t *testing.T) {
	merger := NewPropertyMerger(datamodel.PreserveLocal)
	assert.NotNil(t, merger)
	assert.Equal(t, datamodel.PreserveLocal, merger.strategy)
}

func TestPropertyMerger_extractProperties(t *testing.T) {
	merger := NewPropertyMerger(datamodel.PreserveLocal)

	t.Run("extract from mapping node", func(t *testing.T) {
		yml := `title: "Test Title"
description: "Test Description"
example: {"key": "value"}`

		var node yaml.Node
		_ = yaml.Unmarshal([]byte(yml), &node)
		actualNode := node.Content[0]

		props := merger.extractProperties(actualNode)

		assert.Len(t, props, 3)
		assert.Contains(t, props, "title")
		assert.Contains(t, props, "description")
		assert.Contains(t, props, "example")
		assert.Equal(t, "Test Title", props["title"].Value)
		assert.Equal(t, "Test Description", props["description"].Value)
	})

	t.Run("extract from non-mapping node", func(t *testing.T) {
		node := &yaml.Node{Kind: yaml.ScalarNode, Value: "not a map"}

		props := merger.extractProperties(node)

		assert.Empty(t, props)
	})

	t.Run("extract from nil node", func(t *testing.T) {
		props := merger.extractProperties(nil)

		assert.Empty(t, props)
	})
}

func TestPropertyMerger_MergeProperties(t *testing.T) {
	t.Run("preserve local strategy", func(t *testing.T) {
		merger := NewPropertyMerger(datamodel.PreserveLocal)

		localYml := `title: "Local Title"
example: {"local": "example"}
customProp: "local value"`

		referencedYml := `type: object
title: "Referenced Title"
description: "Referenced Description"
properties:
  id:
    type: string`

		var localNode, referencedNode yaml.Node
		_ = yaml.Unmarshal([]byte(localYml), &localNode)
		_ = yaml.Unmarshal([]byte(referencedYml), &referencedNode)

		merged, err := merger.MergeProperties(localNode.Content[0], referencedNode.Content[0])

		assert.NoError(t, err)
		assert.NotNil(t, merged)

		mergedProps := merger.extractProperties(merged)

		// local properties should be preserved
		assert.Equal(t, "Local Title", mergedProps["title"].Value)
		assert.Contains(t, mergedProps, "example")
		assert.Contains(t, mergedProps, "customProp")

		// referenced properties should be included where no conflict
		assert.Equal(t, "object", mergedProps["type"].Value)
		assert.Equal(t, "Referenced Description", mergedProps["description"].Value)
		assert.Contains(t, mergedProps, "properties")
	})

	t.Run("overwrite with remote strategy", func(t *testing.T) {
		merger := NewPropertyMerger(datamodel.OverwriteWithRemote)

		localYml := `title: "Local Title"
example: {"local": "example"}`

		referencedYml := `title: "Referenced Title"
description: "Referenced Description"`

		var localNode, referencedNode yaml.Node
		_ = yaml.Unmarshal([]byte(localYml), &localNode)
		_ = yaml.Unmarshal([]byte(referencedYml), &referencedNode)

		merged, err := merger.MergeProperties(localNode.Content[0], referencedNode.Content[0])

		assert.NoError(t, err)
		assert.NotNil(t, merged)

		mergedProps := merger.extractProperties(merged)

		// referenced properties should take precedence
		assert.Equal(t, "Referenced Title", mergedProps["title"].Value)
		assert.Equal(t, "Referenced Description", mergedProps["description"].Value)

		// local-only properties should still be included
		assert.Contains(t, mergedProps, "example")
	})

	t.Run("reject conflicts strategy", func(t *testing.T) {
		merger := NewPropertyMerger(datamodel.RejectConflicts)

		localYml := `title: "Local Title"`
		referencedYml := `title: "Referenced Title"`

		var localNode, referencedNode yaml.Node
		_ = yaml.Unmarshal([]byte(localYml), &localNode)
		_ = yaml.Unmarshal([]byte(referencedYml), &referencedNode)

		merged, err := merger.MergeProperties(localNode.Content[0], referencedNode.Content[0])

		assert.Error(t, err)
		assert.Nil(t, merged)
		assert.Contains(t, err.Error(), "property conflict")
		assert.Contains(t, err.Error(), "title")
	})

	t.Run("handle nil nodes", func(t *testing.T) {
		merger := NewPropertyMerger(datamodel.PreserveLocal)

		t.Run("both nil", func(t *testing.T) {
			merged, err := merger.MergeProperties(nil, nil)
			assert.NoError(t, err)
			assert.Nil(t, merged)
		})

		t.Run("local nil", func(t *testing.T) {
			yml := `type: object`
			var node yaml.Node
			_ = yaml.Unmarshal([]byte(yml), &node)

			merged, err := merger.MergeProperties(nil, node.Content[0])
			assert.NoError(t, err)
			assert.NotNil(t, merged)
			assert.Equal(t, "object", merger.extractProperties(merged)["type"].Value)
		})

		t.Run("referenced nil", func(t *testing.T) {
			yml := `example: "test"`
			var node yaml.Node
			_ = yaml.Unmarshal([]byte(yml), &node)

			merged, err := merger.MergeProperties(node.Content[0], nil)
			assert.NoError(t, err)
			assert.NotNil(t, merged)
			assert.Equal(t, "test", merger.extractProperties(merged)["example"].Value)
		})
	})
}

func TestPropertyMerger_ShouldMergeProperties(t *testing.T) {
	merger := NewPropertyMerger(datamodel.PreserveLocal)

	t.Run("should merge when enabled and both have properties", func(t *testing.T) {
		config := &datamodel.DocumentConfiguration{
			MergeReferencedProperties: true,
		}

		localYml := `example: "test"`
		referencedYml := `type: object`

		var localNode, referencedNode yaml.Node
		_ = yaml.Unmarshal([]byte(localYml), &localNode)
		_ = yaml.Unmarshal([]byte(referencedYml), &referencedNode)

		should := merger.ShouldMergeProperties(localNode.Content[0], referencedNode.Content[0], config)
		assert.True(t, should)
	})

	t.Run("should not merge when disabled", func(t *testing.T) {
		config := &datamodel.DocumentConfiguration{
			MergeReferencedProperties: false,
		}

		localYml := `example: "test"`
		referencedYml := `type: object`

		var localNode, referencedNode yaml.Node
		_ = yaml.Unmarshal([]byte(localYml), &localNode)
		_ = yaml.Unmarshal([]byte(referencedYml), &referencedNode)

		should := merger.ShouldMergeProperties(localNode.Content[0], referencedNode.Content[0], config)
		assert.False(t, should)
	})

	t.Run("should not merge when local has no properties", func(t *testing.T) {
		config := &datamodel.DocumentConfiguration{
			MergeReferencedProperties: true,
		}

		referencedYml := `type: object`

		var referencedNode yaml.Node
		_ = yaml.Unmarshal([]byte(referencedYml), &referencedNode)

		// create empty local node
		localNode := &yaml.Node{Kind: yaml.MappingNode}

		should := merger.ShouldMergeProperties(localNode, referencedNode.Content[0], config)
		assert.False(t, should)
	})

	t.Run("handle nil config", func(t *testing.T) {
		localYml := `example: "test"`
		referencedYml := `type: object`

		var localNode, referencedNode yaml.Node
		_ = yaml.Unmarshal([]byte(localYml), &localNode)
		_ = yaml.Unmarshal([]byte(referencedYml), &referencedNode)

		should := merger.ShouldMergeProperties(localNode.Content[0], referencedNode.Content[0], nil)
		assert.False(t, should)
	})
}

func TestPropertyMerger_copyNode_Nil(t *testing.T) {
	// test that copyNode handles nil input correctly (lines 113-114)
	merger := NewPropertyMerger(datamodel.PreserveLocal)

	result := merger.copyNode(nil)
	assert.Nil(t, result)
}
