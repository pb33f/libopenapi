// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"context"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/assert"
)

func TestCompareMediaTypes(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()

	left := `schema:
  type: string
example: tasty herbs in the morning
examples:
  exampleOne:
    value: yummy coffee
encoding:
  contentType: application/json`

	right := `schema:
  type: string
example: tasty herbs in the morning
examples:
  exampleOne:
    value: yummy coffee
encoding:
  contentType: application/json`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.MediaType
	var rDoc v3.MediaType
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareMediaTypes(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareMediaTypes_Modify(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `schema:
  type: string
example: tasty herbs in the morning
examples:
  exampleOne:
    value: yummy coffee
encoding:
  contentType: application/json
itemSchema:
  type: string
itemEncoding:
  contentType: application/json`

	right := `schema:
  type: string
example: smoke and a pancake?
examples:
  exampleOne:
    value: yummy coffee
encoding:
  contentType: application/json
itemSchema:
  type: int
itemEncoding:
  contentType: fish/paste`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.MediaType
	var rDoc v3.MediaType
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareMediaTypes(&lDoc, &rDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 3, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 3)
	// itemSchema type and itemEncoding contentType are both breaking modifications
	assert.Equal(t, 2, extChanges.TotalBreakingChanges())
	assert.Equal(t, Modified, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.ExampleLabel, extChanges.Changes[0].Property)
}

func TestCompareMediaTypes_Modify_Examples(t *testing.T) {
	left := `schema:
  type: string
example:
  smoke: and a pancake?`

	right := `schema:
  type: string
example:
  smoke: and a pancake?
  pipe: pipe and a crepe?`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.MediaType
	var rDoc v3.MediaType
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareMediaTypes(&lDoc, &rDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, Modified, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.ExampleLabel, extChanges.Changes[0].Property)
}

func TestCompareMediaTypes_ExamplesRemoved(t *testing.T) {
	left := `schema:
  type: string
examples:
  exampleOne:
    value: yummy coffee
  exampleTwo:
    value: yummy tea`

	right := `schema:
  type: string
examples:
  exampleOne:
    value: yummy coffee`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	var lDoc v3.MediaType
	var rDoc v3.MediaType
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	extChanges := CompareMediaTypes(&lDoc, &rDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
}

func TestCompareMediaTypes_ExamplesModified_WithCustomBreakingRule(t *testing.T) {
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	defer ResetDefaultBreakingRules()
	defer ResetActiveBreakingRulesConfig()

	config := GenerateDefaultBreakingRules()
	config.MediaType.Examples = rule(false, true, false)
	SetActiveBreakingRulesConfig(config)

	left := `schema:
  type: string
examples:
  exampleOne:
    value: left coffee`

	right := `schema:
  type: string
examples:
  exampleOne:
    value: right coffee`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	var lDoc v3.MediaType
	var rDoc v3.MediaType
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	extChanges := CompareMediaTypes(&lDoc, &rDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Len(t, extChanges.ExampleChanges, 1)
	assert.True(t, extChanges.ExampleChanges["exampleOne"].GetAllChanges()[0].Breaking)
}

func TestCompareMediaTypes_ExampleChangedToMap(t *testing.T) {
	left := `schema:
  type: string`

	right := `schema:
  type: string
example:
  smoke: and a pancake?
  pipe: pipe and a crepe?`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.MediaType
	var rDoc v3.MediaType
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareMediaTypes(&lDoc, &rDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.ExampleLabel, extChanges.Changes[0].Property)
}

func TestCompareMediaTypes_ExampleMapRemoved(t *testing.T) {
	left := `schema:
  type: string`

	right := `schema:
  type: string
example:
  smoke: and a pancake?
  pipe: pipe and a crepe?`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.MediaType
	var rDoc v3.MediaType
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareMediaTypes(&rDoc, &lDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.ExampleLabel, extChanges.Changes[0].Property)
}

func TestCompareMediaTypes_AddSchema(t *testing.T) {
	left := `example: tasty herbs in the morning
examples:
  exampleOne:
    value: yummy coffee
encoding:
  contentType: application/json`

	right := `schema:
  type: string
example: tasty herbs in the morning
examples:
  exampleOne:
    value: yummy coffee
encoding:
  contentType: application/json`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.MediaType
	var rDoc v3.MediaType
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareMediaTypes(&lDoc, &rDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.SchemaLabel, extChanges.Changes[0].Property)
}

func TestCompareMediaTypes_RemoveSchema(t *testing.T) {
	left := `example: tasty herbs in the morning
examples:
  exampleOne:
    value: yummy coffee
encoding:
  contentType: application/json`

	right := `schema:
  type: string
example: tasty herbs in the morning
examples:
  exampleOne:
    value: yummy coffee
encoding:
  contentType: application/json`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.MediaType
	var rDoc v3.MediaType
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareMediaTypes(&rDoc, &lDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.SchemaLabel, extChanges.Changes[0].Property)
}

func TestCompareMediaTypes_ModifyObjects(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()

	left := `schema:
  type: string
example: tasty herbs in the morning
examples:
  exampleOne:
    value: yummy coffee
itemEncoding:
  contentType: milk/hotwater
encoding:
  something: 
    contentType: application/json
x-tea: cake`

	right := `schema:
  type: int
example: smoke and a pancake?
examples:
  exampleOne:
    value: yummy coffee is great!
itemEncoding:
  contentType: milk/hotwater/sugar
encoding:
  something:
    contentType: application/xml
x-tea: cup`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.MediaType
	var rDoc v3.MediaType
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareMediaTypes(&lDoc, &rDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 6, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 6)
	assert.Equal(t, 3, extChanges.TotalBreakingChanges())
}

func TestCompareMediaTypes_ItemSchemaAdded(t *testing.T) {
	// Clear hash cache for deterministic testing
	low.ClearHashCache()

	left := `schema:
  type: array`

	right := `schema:
  type: array
itemSchema:
  type: object
  properties:
    id:
      type: string`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	lIdx := index.NewSpecIndex(&lNode)
	rIdx := index.NewSpecIndex(&rNode)

	var lMt, rMt v3.MediaType
	_ = low.BuildModel(&lNode, &lMt)
	_ = low.BuildModel(&rNode, &rMt)
	_ = lMt.Build(context.Background(), nil, lNode.Content[0], lIdx)
	_ = rMt.Build(context.Background(), nil, rNode.Content[0], rIdx)

	changes := CompareMediaTypes(&lMt, &rMt)

	assert.NotNil(t, changes)
	assert.Nil(t, changes.ItemSchemaChanges) // No changes, just addition
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Equal(t, 1, changes.TotalBreakingChanges()) // Adding itemSchema is breaking

	// Check that the change is ObjectAdded for itemSchema
	allChanges := changes.GetAllChanges()
	assert.Len(t, allChanges, 1)
	assert.Equal(t, ObjectAdded, allChanges[0].ChangeType)
	assert.Equal(t, v3.ItemSchemaLabel, allChanges[0].Property)
}

func TestCompareMediaTypes_ItemSchemaRemoved(t *testing.T) {
	// Clear hash cache for deterministic testing
	low.ClearHashCache()

	left := `schema:
  type: array
itemSchema:
  type: object
  properties:
    id:
      type: string`

	right := `schema:
  type: array`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	lIdx := index.NewSpecIndex(&lNode)
	rIdx := index.NewSpecIndex(&rNode)

	var lMt, rMt v3.MediaType
	_ = low.BuildModel(&lNode, &lMt)
	_ = low.BuildModel(&rNode, &rMt)
	_ = lMt.Build(context.Background(), nil, lNode.Content[0], lIdx)
	_ = rMt.Build(context.Background(), nil, rNode.Content[0], rIdx)

	changes := CompareMediaTypes(&lMt, &rMt)

	assert.NotNil(t, changes)
	assert.Nil(t, changes.ItemSchemaChanges) // No changes, just removal
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Equal(t, 1, changes.TotalBreakingChanges()) // Removing itemSchema is breaking

	// Check that the change is ObjectRemoved for itemSchema
	allChanges := changes.GetAllChanges()
	assert.Len(t, allChanges, 1)
	assert.Equal(t, ObjectRemoved, allChanges[0].ChangeType)
	assert.Equal(t, v3.ItemSchemaLabel, allChanges[0].Property)
}

func TestCompareMediaTypes_ItemSchemaModified(t *testing.T) {
	// Clear hash cache for deterministic testing
	low.ClearHashCache()

	left := `schema:
  type: array
itemSchema:
  type: object
  properties:
    id:
      type: string
    name:
      type: string`

	right := `schema:
  type: array
itemSchema:
  type: object
  properties:
    id:
      type: integer
    name:
      type: string
    age:
      type: integer`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	lIdx := index.NewSpecIndex(&lNode)
	rIdx := index.NewSpecIndex(&rNode)

	var lMt, rMt v3.MediaType
	_ = low.BuildModel(&lNode, &lMt)
	_ = low.BuildModel(&rNode, &rMt)
	_ = lMt.Build(context.Background(), nil, lNode.Content[0], lIdx)
	_ = rMt.Build(context.Background(), nil, rNode.Content[0], rIdx)

	changes := CompareMediaTypes(&lMt, &rMt)

	assert.NotNil(t, changes)
	assert.NotNil(t, changes.ItemSchemaChanges)
	assert.Greater(t, changes.TotalChanges(), 0)
	assert.Greater(t, changes.TotalBreakingChanges(), 0) // Schema changes are breaking
}

func TestCompareMediaTypes_ItemEncodingAdded(t *testing.T) {
	// Clear hash cache for deterministic testing
	low.ClearHashCache()

	left := `schema:
  type: array`

	right := `schema:
  type: array
itemEncoding:
  contentType: image/jpeg
  allowReserved: true`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	lIdx := index.NewSpecIndex(&lNode)
	rIdx := index.NewSpecIndex(&rNode)

	var lMt, rMt v3.MediaType
	_ = low.BuildModel(&lNode, &lMt)
	_ = low.BuildModel(&rNode, &rMt)
	_ = lMt.Build(context.Background(), nil, lNode.Content[0], lIdx)
	_ = rMt.Build(context.Background(), nil, rNode.Content[0], rIdx)

	changes := CompareMediaTypes(&lMt, &rMt)

	assert.NotNil(t, changes)
	assert.Greater(t, changes.TotalChanges(), 0)

	// When a whole encoding block is added, it's tracked in the changes array, not ItemEncodingChanges
	allChanges := changes.GetAllChanges()
	assert.Len(t, allChanges, 1)
	assert.Equal(t, ObjectAdded, allChanges[0].ChangeType)
	assert.Equal(t, v3.ItemEncodingLabel, allChanges[0].Property)
}

func TestCompareMediaTypes_ItemEncodingModified(t *testing.T) {
	// Clear hash cache for deterministic testing
	low.ClearHashCache()

	left := `schema:
  type: array
itemEncoding:
  contentType: image/jpeg
  allowReserved: true`

	right := `schema:
  type: array
itemEncoding:
  contentType: image/png
  allowReserved: false`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	lIdx := index.NewSpecIndex(&lNode)
	rIdx := index.NewSpecIndex(&rNode)

	var lMt, rMt v3.MediaType
	_ = low.BuildModel(&lNode, &lMt)
	_ = low.BuildModel(&rNode, &rMt)
	_ = lMt.Build(context.Background(), nil, lNode.Content[0], lIdx)
	_ = rMt.Build(context.Background(), nil, rNode.Content[0], rIdx)

	changes := CompareMediaTypes(&lMt, &rMt)

	assert.NotNil(t, changes)
	assert.NotNil(t, changes.ItemEncodingChanges)
	assert.Equal(t, 2, changes.ItemEncodingChanges.TotalChanges())
	assert.Greater(t, changes.TotalChanges(), 0)
}

func TestCompareMediaTypes_BothEncodingTypes(t *testing.T) {
	// Clear hash cache for deterministic testing
	low.ClearHashCache()

	left := `schema:
  type: object
encoding:
  field1:
    contentType: text/plain
itemEncoding:
  contentType: application/json`

	right := `schema:
  type: object
encoding:
  field1:
    contentType: text/html
itemEncoding:
  contentType: application/xml`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	lIdx := index.NewSpecIndex(&lNode)
	rIdx := index.NewSpecIndex(&rNode)

	var lMt, rMt v3.MediaType
	_ = low.BuildModel(&lNode, &lMt)
	_ = low.BuildModel(&rNode, &rMt)
	_ = lMt.Build(context.Background(), nil, lNode.Content[0], lIdx)
	_ = rMt.Build(context.Background(), nil, rNode.Content[0], rIdx)

	changes := CompareMediaTypes(&lMt, &rMt)

	assert.NotNil(t, changes)
	assert.NotNil(t, changes.EncodingChanges)
	assert.NotNil(t, changes.ItemEncodingChanges)
	assert.Equal(t, 1, len(changes.EncodingChanges))
	assert.Equal(t, 1, changes.ItemEncodingChanges.TotalChanges())
	assert.Greater(t, changes.TotalChanges(), 1) // At least 2 changes
}

func TestCompareMediaTypes_NoChangesWithItemFields(t *testing.T) {
	// Clear hash cache for deterministic testing
	low.ClearHashCache()

	yml := `schema:
  type: array
itemSchema:
  type: object
  properties:
    id:
      type: string
itemEncoding:
  contentType: text/plain`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &lNode)
	_ = yaml.Unmarshal([]byte(yml), &rNode)

	lIdx := index.NewSpecIndex(&lNode)
	rIdx := index.NewSpecIndex(&rNode)

	var lMt, rMt v3.MediaType
	_ = low.BuildModel(&lNode, &lMt)
	_ = low.BuildModel(&rNode, &rMt)
	_ = lMt.Build(context.Background(), nil, lNode.Content[0], lIdx)
	_ = rMt.Build(context.Background(), nil, rNode.Content[0], rIdx)

	changes := CompareMediaTypes(&lMt, &rMt)

	assert.Nil(t, changes) // No changes
}

func TestMediaTypeChanges_GetAllChanges_WithItemFields(t *testing.T) {
	// Clear hash cache for deterministic testing
	low.ClearHashCache()

	left := `schema:
  type: array`

	right := `schema:
  type: array
itemSchema:
  type: object
itemEncoding:
  contentType: application/json`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	lIdx := index.NewSpecIndex(&lNode)
	rIdx := index.NewSpecIndex(&rNode)

	var lMt, rMt v3.MediaType
	_ = low.BuildModel(&lNode, &lMt)
	_ = low.BuildModel(&rNode, &rMt)
	_ = lMt.Build(context.Background(), nil, lNode.Content[0], lIdx)
	_ = rMt.Build(context.Background(), nil, rNode.Content[0], rIdx)

	changes := CompareMediaTypes(&lMt, &rMt)

	assert.NotNil(t, changes)
	allChanges := changes.GetAllChanges()
	assert.GreaterOrEqual(t, len(allChanges), 2) // At least itemSchema and itemEncoding changes

	// Verify we can find both types of changes
	var foundItemSchema, foundItemEncoding bool
	for _, change := range allChanges {
		if change.Property == v3.ItemSchemaLabel {
			foundItemSchema = true
		}
		if change.Property == v3.ItemEncodingLabel {
			foundItemEncoding = true
		}
	}
	assert.True(t, foundItemSchema, "Should find itemSchema change")
	assert.True(t, foundItemEncoding, "Should find itemEncoding change")
}

func TestMediaTypeChanges_TotalBreakingChanges_WithItemSchema(t *testing.T) {
	// Clear hash cache for deterministic testing
	low.ClearHashCache()

	left := `schema:
  type: array
itemSchema:
  type: object
  required:
    - id
  properties:
    id:
      type: string`

	right := `schema:
  type: array
itemSchema:
  type: object
  required:
    - id
    - name
  properties:
    id:
      type: string
    name:
      type: string`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	lIdx := index.NewSpecIndex(&lNode)
	rIdx := index.NewSpecIndex(&rNode)

	var lMt, rMt v3.MediaType
	_ = low.BuildModel(&lNode, &lMt)
	_ = low.BuildModel(&rNode, &rMt)
	_ = lMt.Build(context.Background(), nil, lNode.Content[0], lIdx)
	_ = rMt.Build(context.Background(), nil, rNode.Content[0], rIdx)

	changes := CompareMediaTypes(&lMt, &rMt)

	assert.NotNil(t, changes)
	assert.NotNil(t, changes.ItemSchemaChanges)
	assert.Greater(t, changes.TotalBreakingChanges(), 0) // Adding required field is breaking
}

func TestCompareMediaTypes_ExampleAdded_AppearsInMap(t *testing.T) {
	// Test that added examples appear in ExampleChanges map for proper tree rendering
	// (instead of just being flat ObjectAdded changes)
	low.ClearHashCache()

	left := `schema:
  type: string`

	right := `schema:
  type: string
examples:
  chewy:
    summary: A chewy example
    value: chewy value`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	lIdx := index.NewSpecIndex(&lNode)
	rIdx := index.NewSpecIndex(&rNode)

	var lMt, rMt v3.MediaType
	_ = low.BuildModel(&lNode, &lMt)
	_ = low.BuildModel(&rNode, &rMt)
	_ = lMt.Build(context.Background(), nil, lNode.Content[0], lIdx)
	_ = rMt.Build(context.Background(), nil, rNode.Content[0], rIdx)

	changes := CompareMediaTypes(&lMt, &rMt)

	assert.NotNil(t, changes)
	// The added example should appear in ExampleChanges map, not just as flat change
	assert.NotNil(t, changes.ExampleChanges)
	assert.NotNil(t, changes.ExampleChanges["chewy"], "Added example 'chewy' should appear in ExampleChanges map")
	assert.Equal(t, 1, changes.ExampleChanges["chewy"].TotalChanges())
	assert.Equal(t, ObjectAdded, changes.ExampleChanges["chewy"].Changes[0].ChangeType)
	// Verify line numbers are set (not 0:0)
	ctx := changes.ExampleChanges["chewy"].Changes[0].Context
	assert.NotNil(t, ctx.NewLine, "Line number should be set for added example")
	assert.Greater(t, *ctx.NewLine, 0, "Line number should be > 0")
}

func TestCompareMediaTypes_ExampleRemoved_AppearsInMap(t *testing.T) {
	// Test that removed examples appear in ExampleChanges map for proper tree rendering
	low.ClearHashCache()

	left := `schema:
  type: string
examples:
  oldExample:
    summary: An old example
    value: old value`

	right := `schema:
  type: string`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	lIdx := index.NewSpecIndex(&lNode)
	rIdx := index.NewSpecIndex(&rNode)

	var lMt, rMt v3.MediaType
	_ = low.BuildModel(&lNode, &lMt)
	_ = low.BuildModel(&rNode, &rMt)
	_ = lMt.Build(context.Background(), nil, lNode.Content[0], lIdx)
	_ = rMt.Build(context.Background(), nil, rNode.Content[0], rIdx)

	changes := CompareMediaTypes(&lMt, &rMt)

	assert.NotNil(t, changes)
	// The removed example should appear in ExampleChanges map
	assert.NotNil(t, changes.ExampleChanges)
	assert.NotNil(t, changes.ExampleChanges["oldExample"], "Removed example 'oldExample' should appear in ExampleChanges map")
	assert.Equal(t, 1, changes.ExampleChanges["oldExample"].TotalChanges())
	assert.Equal(t, ObjectRemoved, changes.ExampleChanges["oldExample"].Changes[0].ChangeType)
}

func compareMediaTypeYAML(t *testing.T, left, right string) *MediaTypeChanges {
	t.Helper()
	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)
	var lMt, rMt v3.MediaType
	_ = low.BuildModel(&lNode, &lMt)
	_ = low.BuildModel(&rNode, &rMt)
	assert.NoError(t, lMt.Build(context.Background(), nil, lNode.Content[0], index.NewSpecIndex(&lNode)))
	assert.NoError(t, rMt.Build(context.Background(), nil, rNode.Content[0], index.NewSpecIndex(&rNode)))
	return CompareMediaTypes(&lMt, &rMt)
}

func TestCompareMediaTypes_PrefixEncoding(t *testing.T) {
	low.ClearHashCache()
	ResetActiveBreakingRulesConfig()

	left := `prefixEncoding:
  - contentType: application/json
  - contentType: image/png`

	// modified by position
	changes := compareMediaTypeYAML(t, left, `prefixEncoding:
  - contentType: application/json
  - contentType: image/gif`)
	assert.NotNil(t, changes)
	assert.Len(t, changes.PrefixEncodingChanges, 1)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Len(t, changes.GetAllChanges(), 1)

	// removing a positional entry is breaking
	changes = compareMediaTypeYAML(t, left, `prefixEncoding:
  - contentType: application/json`)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.PrefixEncodingLabel, changes.GetAllChanges()[0].Property)

	// adding one is not
	changes = compareMediaTypeYAML(t, left, left+`
  - contentType: text/plain`)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Equal(t, 0, changes.TotalBreakingChanges())

	// reordering changes the entry at both positions
	changes = compareMediaTypeYAML(t, left, `prefixEncoding:
  - contentType: image/png
  - contentType: application/json`)
	assert.NotNil(t, changes)
	assert.Len(t, changes.PrefixEncodingChanges, 2)
}

func TestCompareMediaTypes_ItemEncodingRemovedIsBreaking(t *testing.T) {
	low.ClearHashCache()
	ResetActiveBreakingRulesConfig()

	changes := compareMediaTypeYAML(t, `schema:
  type: array
itemEncoding:
  contentType: image/png`, `schema:
  type: array`)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
}

func TestCompareMediaTypes_NestedEncoding(t *testing.T) {
	low.ClearHashCache()
	ResetActiveBreakingRulesConfig()

	changes := compareMediaTypeYAML(t, `encoding:
  batch:
    contentType: multipart/mixed
    prefixEncoding:
      - contentType: application/json
    itemEncoding:
      contentType: image/png
    encoding:
      inner:
        contentType: text/plain`, `encoding:
  batch:
    contentType: multipart/mixed
    prefixEncoding:
      - contentType: application/xml
    itemEncoding:
      contentType: image/gif`)
	assert.NotNil(t, changes)
	batch := changes.EncodingChanges["batch"]
	assert.NotNil(t, batch)
	assert.Len(t, batch.PrefixEncodingChanges, 1)
	assert.NotNil(t, batch.ItemEncodingChanges)
	// prefixEncoding[0].contentType, itemEncoding.contentType, encoding.inner removed
	assert.Equal(t, 3, changes.TotalChanges())
	assert.Equal(t, 3, changes.TotalBreakingChanges())
}

func TestCompareMediaTypes_NestedEncoding_Modified(t *testing.T) {
	low.ClearHashCache()
	ResetActiveBreakingRulesConfig()

	changes := compareMediaTypeYAML(t, `encoding:
  batch:
    encoding:
      inner:
        contentType: text/plain
    prefixEncoding:
      - itemEncoding:
          contentType: image/png`, `encoding:
  batch:
    encoding:
      inner:
        contentType: text/csv
    prefixEncoding:
      - itemEncoding:
          contentType: image/gif`)
	assert.NotNil(t, changes)
	batch := changes.EncodingChanges["batch"]
	assert.NotNil(t, batch)
	assert.NotNil(t, batch.EncodingChanges["inner"])
	assert.NotNil(t, batch.PrefixEncodingChanges[0].ItemEncodingChanges)
	// encoding.inner.contentType and prefixEncoding[0].itemEncoding.contentType
	assert.Equal(t, 2, changes.TotalChanges())
	assert.Equal(t, 2, changes.TotalBreakingChanges())
	assert.Len(t, changes.GetAllChanges(), 2)
}

func TestCompareMediaTypes_EncodingRulesConfigurable(t *testing.T) {
	low.ClearHashCache()
	defer ResetActiveBreakingRulesConfig()

	config := GenerateDefaultBreakingRules()
	config.Merge(&BreakingRulesConfig{
		MediaType: &MediaTypeRules{PrefixEncoding: rule(false, false, false)},
		Encoding:  &EncodingRules{ItemEncoding: rule(false, false, false)},
	})
	SetActiveBreakingRulesConfig(config)

	changes := compareMediaTypeYAML(t, `prefixEncoding:
  - contentType: application/json
encoding:
  batch:
    itemEncoding:
      contentType: image/png`, `encoding:
  batch: {}`)
	assert.NotNil(t, changes)
	assert.Equal(t, 2, changes.TotalChanges())
	assert.Equal(t, 0, changes.TotalBreakingChanges())
}
