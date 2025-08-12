// Copyright 2024 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"testing"

	"github.com/pkg-base/yaml"
	"github.com/stretchr/testify/assert"
)

func TestSpecIndex_TagCircularReferences_SimpleCircle(t *testing.T) {
	yml := `openapi: 3.2.0
info:
  title: Test API
  version: 1.0.0
tags:
  - name: tagA
    summary: Tag A
    parent: tagB
  - name: tagB
    summary: Tag B
    parent: tagA
paths: {}`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	idx := NewSpecIndex(&idxNode)

	// Trigger the tag counting which will check for circular references
	count := idx.GetGlobalTagsCount()
	assert.Equal(t, 2, count)

	// Check that circular references were detected
	circRefs := idx.GetTagCircularReferences()
	assert.Len(t, circRefs, 1)

	circRef := circRefs[0]
	assert.True(t, circRef.IsInfiniteLoop)
	assert.False(t, circRef.IsArrayResult)
	assert.False(t, circRef.IsPolymorphicResult)

	// Check the journey path - should be 3 items forming a circle
	assert.Len(t, circRef.Journey, 3)

	// Extract journey names for easier checking
	journeyNames := make([]string, len(circRef.Journey))
	for i, ref := range circRef.Journey {
		journeyNames[i] = ref.Name
	}

	// Should contain both tags and form a circle
	assert.Contains(t, journeyNames, "tagA")
	assert.Contains(t, journeyNames, "tagB")

	// First and last elements should be the same (forming a circle)
	assert.Equal(t, journeyNames[0], journeyNames[2])

	// Loop point and start should be the same
	assert.Equal(t, circRef.LoopPoint.Name, circRef.Start.Name)
}

func TestSpecIndex_TagCircularReferences_ThreeTagCircle(t *testing.T) {
	yml := `openapi: 3.2.0
info:
  title: Test API
  version: 1.0.0
tags:
  - name: tagA
    summary: Tag A
    parent: tagB
  - name: tagB
    summary: Tag B
    parent: tagC
  - name: tagC
    summary: Tag C
    parent: tagA
paths: {}`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	idx := NewSpecIndex(&idxNode)

	count := idx.GetGlobalTagsCount()
	assert.Equal(t, 3, count)

	circRefs := idx.GetTagCircularReferences()
	assert.Len(t, circRefs, 1)

	circRef := circRefs[0]
	assert.True(t, circRef.IsInfiniteLoop)

	// Check the journey path - should be 4 items: tagA -> tagB -> tagC -> tagA
	assert.Len(t, circRef.Journey, 4)
	journeyNames := make([]string, len(circRef.Journey))
	for i, ref := range circRef.Journey {
		journeyNames[i] = ref.Name
	}

	// Should contain the cycle
	assert.Contains(t, journeyNames, "tagA")
	assert.Contains(t, journeyNames, "tagB")
	assert.Contains(t, journeyNames, "tagC")
}

func TestSpecIndex_TagCircularReferences_NoCircle(t *testing.T) {
	yml := `openapi: 3.2.0
info:
  title: Test API
  version: 1.0.0
tags:
  - name: external
    summary: External
    description: Operations available to external consumers
    kind: audience
  - name: partner
    summary: Partner
    description: Operations available to the partners network
    parent: external
    kind: audience
  - name: account-updates
    summary: Account Updates
    description: Account update operations
    kind: nav
paths: {}`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	idx := NewSpecIndex(&idxNode)

	count := idx.GetGlobalTagsCount()
	assert.Equal(t, 3, count)

	// Should have no circular references
	circRefs := idx.GetTagCircularReferences()
	assert.Len(t, circRefs, 0)
}

func TestSpecIndex_TagCircularReferences_NonExistentParent(t *testing.T) {
	yml := `openapi: 3.2.0
info:
  title: Test API
  version: 1.0.0
tags:
  - name: tagA
    summary: Tag A
    parent: nonExistentTag
  - name: tagB
    summary: Tag B
    parent: tagA
paths: {}`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	idx := NewSpecIndex(&idxNode)

	count := idx.GetGlobalTagsCount()
	assert.Equal(t, 2, count)

	// Should have no circular references (nonExistentTag is not defined)
	circRefs := idx.GetTagCircularReferences()
	assert.Len(t, circRefs, 0)
}

func TestSpecIndex_TagCircularReferences_SelfReference(t *testing.T) {
	yml := `openapi: 3.2.0
info:
  title: Test API
  version: 1.0.0
tags:
  - name: selfRef
    summary: Self Reference
    parent: selfRef
paths: {}`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	idx := NewSpecIndex(&idxNode)

	count := idx.GetGlobalTagsCount()
	assert.Equal(t, 1, count)

	// Should detect self-reference as circular
	circRefs := idx.GetTagCircularReferences()
	assert.Len(t, circRefs, 1)

	circRef := circRefs[0]
	assert.True(t, circRef.IsInfiniteLoop)
	assert.Equal(t, "selfRef", circRef.Start.Name)
	assert.Equal(t, "selfRef", circRef.LoopPoint.Name)

	// Journey should be [selfRef, selfRef]
	assert.Len(t, circRef.Journey, 2)
	assert.Equal(t, "selfRef", circRef.Journey[0].Name)
	assert.Equal(t, "selfRef", circRef.Journey[1].Name)
}

func TestSpecIndex_TagCircularReferences_ComplexHierarchy(t *testing.T) {
	yml := `openapi: 3.2.0
info:
  title: Test API
  version: 1.0.0
tags:
  - name: root
    summary: Root tag
  - name: childA
    summary: Child A
    parent: root
  - name: childB
    summary: Child B  
    parent: root
  - name: grandchildA1
    summary: Grandchild A1
    parent: childA
  - name: grandchildA2
    summary: Grandchild A2
    parent: childA
  - name: circularChild
    summary: Circular Child
    parent: circularParent
  - name: circularParent
    summary: Circular Parent
    parent: circularChild
paths: {}`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	idx := NewSpecIndex(&idxNode)

	count := idx.GetGlobalTagsCount()
	assert.Equal(t, 7, count)

	// Should detect the one circular reference between circularChild and circularParent
	circRefs := idx.GetTagCircularReferences()
	assert.Len(t, circRefs, 1)

	circRef := circRefs[0]
	assert.True(t, circRef.IsInfiniteLoop)

	// Check that the circular reference involves the expected tags
	journeyNames := make([]string, len(circRef.Journey))
	for i, ref := range circRef.Journey {
		journeyNames[i] = ref.Name
	}

	assert.Contains(t, journeyNames, "circularChild")
	assert.Contains(t, journeyNames, "circularParent")
}

func TestSpecIndex_TagCircularReferences_NoTags(t *testing.T) {
	yml := `openapi: 3.2.0
info:
  title: Test API
  version: 1.0.0
paths: {}`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	idx := NewSpecIndex(&idxNode)

	count := idx.GetGlobalTagsCount()
	assert.Equal(t, 0, count)

	// Should have no circular references
	circRefs := idx.GetTagCircularReferences()
	assert.Len(t, circRefs, 0)
}

func TestSpecIndex_TagCircularReferences_EmptyTags(t *testing.T) {
	yml := `openapi: 3.2.0
info:
  title: Test API
  version: 1.0.0
tags: []
paths: {}`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	idx := NewSpecIndex(&idxNode)

	count := idx.GetGlobalTagsCount()
	assert.Equal(t, 0, count)

	// Should have no circular references
	circRefs := idx.GetTagCircularReferences()
	assert.Len(t, circRefs, 0)
}

func TestSpecIndex_TagCircularReferences_NilTagsNode(t *testing.T) {
	yml := `openapi: 3.2.0
info:
  title: Test API
  version: 1.0.0
paths: {}`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	idx := NewSpecIndex(&idxNode)

	// Explicitly set tagsNode to nil to test the early return
	idx.tagsNode = nil

	// This should trigger checkTagCircularReferences() which should return early due to nil tagsNode
	count := idx.GetGlobalTagsCount()
	assert.Equal(t, 0, count)

	// Should have no circular references due to early return
	circRefs := idx.GetTagCircularReferences()
	assert.Len(t, circRefs, 0)
}

func TestSpecIndex_detectTagCircularHelper_NonExistentTag(t *testing.T) {
	yml := `openapi: 3.2.0
info:
  title: Test API
  version: 1.0.0
tags:
  - name: existingTag
    summary: Existing Tag
paths: {}`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	idx := NewSpecIndex(&idxNode)

	// Create the maps that would be passed to detectTagCircularHelper
	parentMap := map[string]string{}
	tagRefs := map[string]*Reference{
		"existingTag": {
			Name: "existingTag",
			Node: &yaml.Node{Value: "existingTag"},
			Path: "$.tags[0]",
		},
	}
	visited := map[string]bool{}
	recStack := map[string]bool{}

	// Test calling detectTagCircularHelper with a non-existent tag name
	// This should trigger the early return on lines 756-757
	path := idx.detectTagCircularHelper("nonExistentTag", parentMap, tagRefs, visited, recStack, []string{})

	// Should return empty slice since the tag doesn't exist
	assert.Len(t, path, 0)

	// Verify that visited and recStack remain untouched
	assert.Len(t, visited, 0)
	assert.Len(t, recStack, 0)
}
