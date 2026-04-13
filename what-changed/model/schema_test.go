// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"

	"github.com/pb33f/libopenapi/utils"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	v2 "github.com/pb33f/libopenapi/datamodel/low/v2"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

// These tests require full documents to be tested properly. schemas are perhaps the most complex
// of all the things in OpenAPI, to ensure correctness, we must test the whole document structure.
//
// To test this correctly, we need a simulated document with inline schemas for recursive
// checking, as well as a couple of references, so we can avoid that disaster.
// in our model, components/definitions will be checked independently for changes
// and references will be checked only for value changes (points to a different reference)
func TestCompareSchemas_PropertyModification(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      title: an OK message`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      title: an OK message Changed`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Equal(t, Modified, changes.Changes[0].ChangeType)
	assert.Equal(t, "an OK message Changed", changes.Changes[0].New)
}

func TestCompareSchemas_PropertyAdd(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      title: an OK message`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      title: an OK message
      description: a thing`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.Changes, 1)
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, PropertyAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, "a thing", changes.Changes[0].New)
	assert.Equal(t, v3.DescriptionLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_PropertyRemove(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      title: an OK message
      description: a thing`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      title: an OK message`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.Changes, 1)
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, PropertyRemoved, changes.Changes[0].ChangeType)
	assert.Equal(t, "a thing", changes.Changes[0].Original)
	assert.Equal(t, v3.DescriptionLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_Removed(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      title: an OK message
      description: a thing`

	right := `openapi: 3.0
components:
  schemas:`

	leftDoc, _ := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	var rSchemaProxy *base.SchemaProxy

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.Changes, 1)
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, ObjectRemoved, changes.Changes[0].ChangeType)
}

func TestCompareSchemas_Added(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      title: an OK message
      description: a thing`

	right := `openapi: 3.0
components:
  schemas:`

	leftDoc, _ := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	var rSchemaProxy *base.SchemaProxy

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.Changes, 1)
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, ObjectAdded, changes.Changes[0].ChangeType)
}

func test_BuildDoc(l, r string) (*v3.Document, *v3.Document) {
	leftInfo, _ := datamodel.ExtractSpecInfo([]byte(l))
	rightInfo, _ := datamodel.ExtractSpecInfo([]byte(r))

	leftDoc, _ := v3.CreateDocumentFromConfig(leftInfo, datamodel.NewDocumentConfiguration())
	rightDoc, _ := v3.CreateDocumentFromConfig(rightInfo, datamodel.NewDocumentConfiguration())
	return leftDoc, rightDoc
}

func test_BuildDocv2(l, r string) (*v2.Swagger, *v2.Swagger) {
	leftInfo, _ := datamodel.ExtractSpecInfo([]byte(l))
	rightInfo, _ := datamodel.ExtractSpecInfo([]byte(r))

	var err error
	var leftDoc, rightDoc *v2.Swagger
	leftDoc, _ = v2.CreateDocumentFromConfig(leftInfo, datamodel.NewDocumentConfiguration())
	rightDoc, err = v2.CreateDocumentFromConfig(rightInfo, datamodel.NewDocumentConfiguration())

	uErr := utils.UnwrapErrors(err)
	if len(uErr) > 0 {
		for i := range uErr {
			fmt.Printf("error: %v\n", uErr[i])
		}
		panic("failed to create doc")
	}
	return leftDoc, rightDoc
}

func TestSchemaCompositionHelpers(t *testing.T) {
	low.ClearHashCache()

	spec := `openapi: 3.0
components:
  schemas:
    A:
      type: string
    RefBranch:
      $ref: '#/components/schemas/A'
    InlineBranch:
      type: integer`

	doc, _ := test_BuildDoc(spec, spec)
	refProxy := doc.Components.Value.FindSchema("RefBranch").Value
	inlineProxy := doc.Components.Value.FindSchema("InlineBranch").Value

	assert.True(t, isOrderInsensitiveSchemaCompositionLabel(v3.OneOfLabel))
	assert.True(t, isOrderInsensitiveSchemaCompositionLabel(v3.AnyOfLabel))
	assert.True(t, isOrderInsensitiveSchemaCompositionLabel(v3.AllOfLabel))
	assert.False(t, isOrderInsensitiveSchemaCompositionLabel(v3.PrefixItemsLabel))
	assert.False(t, isOrderInsensitiveSchemaCompositionLabel("unknown"))

	assert.Equal(t, "nil", schemaCompositionEntryIdentity(nil))
	assert.Equal(t, "ref:#/components/schemas/A", schemaCompositionEntryIdentity(refProxy))
	assert.Contains(t, schemaCompositionEntryIdentity(inlineProxy), "hash:")

	entries := buildSchemaCompositionEntries([]low.ValueReference[*base.SchemaProxy]{
		{Value: refProxy},
		{Value: inlineProxy},
	})
	assert.Len(t, entries, 2)
	assert.Equal(t, "ref:#/components/schemas/A", entries[0].identity)
	assert.Equal(t, refProxy, entries[0].proxy)
	assert.Equal(t, inlineProxy, entries[1].proxy)

	leftEntries := []schemaCompositionEntry{
		{identity: "a"},
		{identity: "a"},
		{identity: "b"},
	}
	rightEntries := []schemaCompositionEntry{
		{identity: "a"},
		{identity: "c"},
		{identity: "a"},
	}
	exactPairs, leftUnmatched, rightUnmatched := pairExactSchemaCompositionEntries(leftEntries, rightEntries)
	assert.Len(t, exactPairs, 2)
	assert.Equal(t,
		[]schemaCompositionEntry{{identity: "b"}},
		leftUnmatched,
	)
	assert.Equal(t,
		[]schemaCompositionEntry{{identity: "c"}},
		rightUnmatched,
	)

	assert.True(t, schemaCompositionChangeBreaking(v3.AllOfLabel, ObjectRemoved))
	assert.False(t, schemaCompositionChangeBreaking(v3.AllOfLabel, ObjectAdded))
	assert.True(t, schemaCompositionChangeBreaking(v3.AnyOfLabel, ObjectRemoved))
	assert.False(t, schemaCompositionChangeBreaking(v3.AnyOfLabel, ObjectAdded))
	assert.True(t, schemaCompositionChangeBreaking(v3.OneOfLabel, ObjectRemoved))
	assert.False(t, schemaCompositionChangeBreaking(v3.OneOfLabel, ObjectAdded))
	assert.True(t, schemaCompositionChangeBreaking(v3.PrefixItemsLabel, ObjectRemoved))
	assert.False(t, schemaCompositionChangeBreaking(v3.PrefixItemsLabel, ObjectAdded))
	assert.True(t, schemaCompositionChangeBreaking("unknown", ObjectRemoved))
	assert.False(t, schemaCompositionChangeBreaking("unknown", ObjectAdded))
}

func TestCompareSchemas_RefIgnore(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    Yo:
      type: int
    OK:
      $ref: '#/components/schemas/Yo'`

	right := `openapi: 3.0
components:
  schemas:
    Yo:
      type: int
    OK:
      $ref: '#/components/schemas/Yo'`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.Nil(t, changes)
}

func TestCompareSchemas_RefChanged(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    Woah:
      type: int
    Yo:
      type: string
    OK:
      $ref: '#/components/schemas/Woah'`

	right := `openapi: 3.0
components:
  schemas:
    Woah:
      type: int
    Yo:
      type: string
    OK:
      $ref: '#/components/schemas/Yo'`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.Changes, 1)
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, Modified, changes.Changes[0].ChangeType)
	assert.Equal(t, "#/components/schemas/Yo", changes.Changes[0].New)
}

func TestCompareSchemas_RefToInline(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    Yo:
      type: int
    OK:
      $ref: '#/components/schemas/Yo'`

	right := `openapi: 3.0
components:
  schemas:
    Yo:
      type: int
    OK:
      type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.Changes, 1)
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, Modified, changes.Changes[0].ChangeType)
	assert.Equal(t, v3.RefLabel, changes.Changes[0].Property)
	assert.Equal(t, "#/components/schemas/Yo", changes.Changes[0].Original)
}

func TestCompareSchemas_InlineToRef(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    Yo:
      type: int
    OK:
      $ref: '#/components/schemas/Yo'`

	right := `openapi: 3.0
components:
  schemas:
    Yo:
      type: int
    OK:
      type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.Changes, 1)
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, Modified, changes.Changes[0].ChangeType)
	assert.Equal(t, v3.RefLabel, changes.Changes[0].Property)
	assert.Equal(t, "#/components/schemas/Yo", changes.Changes[0].New)
}

func TestCompareSchemas_Identical(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    Yo:
      type: int
    OK:
      type: string`

	right := `openapi: 3.0
components:
  schemas:
    Yo:
      type: int
    OK:
      type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.Nil(t, changes)
}

func TestCompareSchemas_Identical_Ref(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    Yo:
      type: int
    OK:
      $ref: '#/components/schemas/Yo'`

	right := `openapi: 3.0
components:
  schemas:
    Yo:
      type: int
    OK:
       $ref: '#/components/schemas/Yo'`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.Nil(t, changes)
}

func TestCompareSchemas_RequiredAdded(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      title: an OK message
      description: a thing
      required:
        - one`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      title: an OK message
      description: a thing
      required:
        - one
        - two`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.Changes, 1)
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, PropertyAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, "two", changes.Changes[0].New)
	assert.Equal(t, v3.RequiredLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_RequiredRemoved(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      required:
        - one`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      required:
        - one
        - two`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.Changes, 1)
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, PropertyRemoved, changes.Changes[0].ChangeType)
	assert.Equal(t, "two", changes.Changes[0].Original)
	assert.Equal(t, v3.RequiredLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_EnumSimilar(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      enum: [a]`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      enum: ["a"]`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.Nil(t, changes)
}

func TestCompareSchemas_EnumAdded(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      enum: [a,b,c]`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      enum: [a,b,c,d]`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.Changes, 1)
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, PropertyAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, "d", changes.Changes[0].New)
	assert.Equal(t, v3.EnumLabel, changes.Changes[0].Property)
	assert.Equal(t, 5, *changes.GetAllChanges()[0].Context.NewLine)
	assert.Equal(t, 20, *changes.GetAllChanges()[0].Context.NewColumn)
}

func TestCompareSchemas_EnumRemoved(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      enum: [a,b,c]`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      enum: [a,b,c,d]`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.Changes, 1)
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, PropertyRemoved, changes.Changes[0].ChangeType)
	assert.Equal(t, "d", changes.Changes[0].Original)
	assert.Equal(t, v3.EnumLabel, changes.Changes[0].Property)
	assert.Equal(t, 5, *changes.GetAllChanges()[0].Context.OriginalLine)
	assert.Equal(t, 20, *changes.GetAllChanges()[0].Context.OriginalColumn)
}

func TestCompareSchemas_PropertyAdded(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      properties:
        propA:
          type: int`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      properties:
        propB:
          type: string
        propA:
          type: int`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.Changes, 1)
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, ObjectAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, "propB", changes.Changes[0].New)
	assert.Equal(t, v3.PropertiesLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_PropertyRemoved(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      properties:
        propA:
          type: int`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      properties:
        propB:
          type: string
        propA:
          type: int`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.Changes, 1)
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, ObjectRemoved, changes.Changes[0].ChangeType)
	assert.Equal(t, "propB", changes.Changes[0].Original)
	assert.Equal(t, v3.PropertiesLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_If(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      if:
        type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      if:
        type: int`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, 1, changes.IfChanges.PropertyChanges.TotalChanges())
}

func TestCompareSchemas_If_Added(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      type: string
      if:
        type: int`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.IfLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_If_Removed(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      type: string
      if:
        type: int`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.IfLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_Else(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      else:
        type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      else:
        type: int`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, 1, changes.ElseChanges.PropertyChanges.TotalChanges())
}

func TestCompareSchemas_Else_Added(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      type: string
      else:
        type: int`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ElseLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_Else_Removed(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      type: string
      else:
        type: int`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ElseLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_Then(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      then:
        type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      then:
        type: int`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, 1, changes.ThenChanges.PropertyChanges.TotalChanges())
}

func TestCompareSchemas_Then_Added(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      type: string
      then:
        type: int`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ThenLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_Then_Removed(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      type: string
      then:
        type: int`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ThenLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_DependentSchemas(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      dependentSchemas:
        schemaOne:
          type: int`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      dependentSchemas:
        schemaOne:
          type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, 1, changes.DependentSchemasChanges["schemaOne"].PropertyChanges.TotalChanges())
}

func TestCompareSchemas_PatternProperties(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      patternProperties:
        schemaOne:
          type: int`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      patternProperties:
        schemaOne:
          type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, 1, changes.PatternPropertiesChanges["schemaOne"].PropertyChanges.TotalChanges())
}

func TestCompareSchemas_PropertyNames(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      propertyNames:
        type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      propertyNames:
        type: int`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, 1, changes.PropertyNamesChanges.PropertyChanges.TotalChanges())
}

func TestCompareSchemas_PropertyNames_Added(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      type: string
      propertyNames:
        type: int`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.PropertyNamesLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_PropertyNames_Removed(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      type: string
      propertyNames:
        type: int`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.PropertyNamesLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_Contains(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      contains:
        type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      contains:
        type: int`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, 1, changes.ContainsChanges.PropertyChanges.TotalChanges())
}

func TestCompareSchemas_Contains_Added(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      type: string
      contains:
        type: int`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ContainsLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_Contains_Removed(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      type: string
      contains:
        type: int`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ContainsLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_UnevaluatedProperties_Bool(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      unevaluatedProperties: false`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      unevaluatedProperties: true`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
}

func TestCompareSchemas_UnevaluatedProperties_Bool_Schema(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      unevaluatedProperties: false`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      unevaluatedProperties:
        type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
}

func TestCompareSchemas_UnevaluatedProperties_Schema_Bool(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      unevaluatedProperties: false`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      unevaluatedProperties:
        type: string`

	leftDoc, rightDoc := test_BuildDoc(right, left)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
}

func TestCompareSchemas_UnevaluatedProperties(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      unevaluatedProperties:
        type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      unevaluatedProperties:
        type: int`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, 1, changes.UnevaluatedPropertiesChanges.PropertyChanges.TotalChanges())
}

func TestCompareSchemas_UnevaluatedProperties_Added(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      type: string
      unevaluatedProperties:
        type: int`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.UnevaluatedPropertiesLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_UnevaluatedProperties_Removed(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      type: string
      unevaluatedProperties:
        type: int`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.UnevaluatedPropertiesLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_AdditionalProperties(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      additionalProperties:
        type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      additionalProperties:
        type: int`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, 1, changes.AdditionalPropertiesChanges.PropertyChanges.TotalChanges())
}

func TestCompareSchemas_AdditionalProperties_Boolean(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      additionalProperties: true`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      additionalProperties: false`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, 1, changes.PropertyChanges.TotalChanges())
}

func TestCompareSchemas_AdditionalProperties_Boolean_To_Schema(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      additionalProperties: true`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      additionalProperties:
        type: int`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, 1, changes.PropertyChanges.TotalChanges())
}

func TestCompareSchemas_AdditionalProperties_Added(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      type: string
      additionalProperties:
        type: int`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.AdditionalPropertiesLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_AdditionalProperties_Removed(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      type: string
      additionalProperties:
        type: int`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.AdditionalPropertiesLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_UnevaluatedItems(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      unevaluatedItems:
        type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      unevaluatedItems:
        type: int`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, 1, changes.UnevaluatedItemsChanges.PropertyChanges.TotalChanges())
}

func TestCompareSchemas_UnevaluatedItems_Added(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      type: string
      unevaluatedItems:
        type: int`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.UnevaluatedItemsLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_UnevaluatedItems_Removed(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      type: string
      unevaluatedItems:
        type: int`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.UnevaluatedItemsLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_ItemsBoolean(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      items: true`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      items: false`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Equal(t, 1, changes.TotalBreakingChanges())
}

func TestCompareSchemas_ItemsAdded(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      type: string
      items: true`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Equal(t, 1, changes.TotalBreakingChanges())
}

func TestCompareSchemas_ItemsRemoved(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      type: string
      items: true`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Equal(t, 1, changes.TotalBreakingChanges())
}

func TestCompareSchemas_NotAdded(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      type: string
      not:
        type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Equal(t, 1, changes.TotalBreakingChanges())
}

func TestCompareSchemas_NotRemoved(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      type: string
      not:
        type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
}

func TestCompareSchemas_PropertyChanged(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      properties:
        propA:
          type: int`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      properties:
        propA:
          type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, Modified, changes.SchemaPropertyChanges["propA"].Changes[0].ChangeType)
	assert.Equal(t, "string", changes.SchemaPropertyChanges["propA"].Changes[0].New)
	assert.Equal(t, "int", changes.SchemaPropertyChanges["propA"].Changes[0].Original)
}

func TestCompareSchemas_PropertySwap(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      properties:
        propA:
          type: int`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      properties:
        propN:
          type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 2, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 2)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, "propN", changes.Changes[0].New)
	assert.Equal(t, v3.PropertiesLabel, changes.Changes[0].Property)
	assert.Equal(t, ObjectRemoved, changes.Changes[1].ChangeType)
	assert.Equal(t, "propA", changes.Changes[1].Original)
	assert.Equal(t, v3.PropertiesLabel, changes.Changes[1].Property)
}

func TestCompareSchemas_AnyOfModifyAndAddItem(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      anyOf:
        - type: bool`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      anyOf:
        - type: string
        - type: int"`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 2, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 2)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, v3.AnyOfLabel, changes.Changes[0].Property)
	assert.Equal(t, Modified, changes.AnyOfChanges[0].Changes[0].ChangeType)
	assert.Equal(t, "string", changes.AnyOfChanges[0].Changes[0].New)
	assert.Equal(t, "bool", changes.AnyOfChanges[0].Changes[0].Original)
}

func TestCompareSchemas_AnyOfModifyAndRemoveItem(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      anyOf:
        - type: bool`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      anyOf:
        - type: string
        - type: int"`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 2, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 2)
	assert.Equal(t, 2, changes.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, changes.Changes[0].ChangeType)
	assert.Equal(t, v3.AnyOfLabel, changes.Changes[0].Property)
	assert.Equal(t, Modified, changes.AnyOfChanges[0].Changes[0].ChangeType)
	assert.Equal(t, "bool", changes.AnyOfChanges[0].Changes[0].New)
	assert.Equal(t, "string", changes.AnyOfChanges[0].Changes[0].Original)
}

func TestCompareSchemas_AnyOfModified(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      anyOf:
        - type: bool`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      anyOf:
        - type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, Modified, changes.AnyOfChanges[0].Changes[0].ChangeType)
	assert.Equal(t, "string", changes.AnyOfChanges[0].Changes[0].New)
	assert.Equal(t, "bool", changes.AnyOfChanges[0].Changes[0].Original)
}

// https://github.com/pb33f/openapi-changes/issues/105
func TestCompareSchemas_AnyOfSimpleScalarUnionMatchesTypeArray(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1.0
components:
  schemas:
    operator:
      anyOf:
        - type: string
        - type: number
        - type: null`

	right := `openapi: 3.1.0
components:
  schemas:
    operator:
      type:
        - null
        - string
        - number`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("operator").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("operator").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.Nil(t, changes)
	assert.Equal(t, 0, changes.TotalChanges())

	changes = CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.Nil(t, changes)
	assert.Equal(t, 0, changes.TotalChanges())
}

func TestCompareSchemas_AnyOfSimpleScalarUnionDoesNotIgnoreBranchConstraints(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1.0
components:
  schemas:
    operator:
      anyOf:
        - type: string
          format: uuid
        - type: number
        - type: null`

	right := `openapi: 3.1.0
components:
  schemas:
    operator:
      type:
        - null
        - string
        - number`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("operator").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("operator").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Greater(t, changes.TotalChanges(), 0)
}

func TestCompareSchemas_OneOfModifyAndAddItem(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      oneOf:
        - type: bool`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      oneOf:
        - type: string
        - type: int"`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 2, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 2)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, v3.OneOfLabel, changes.Changes[0].Property)
	assert.Equal(t, Modified, changes.OneOfChanges[0].Changes[0].ChangeType)
	assert.Equal(t, "string", changes.OneOfChanges[0].Changes[0].New)
	assert.Equal(t, "bool", changes.OneOfChanges[0].Changes[0].Original)
}

func TestCompareSchemas_AllOfModifyAndAddItem(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      allOf:
        - type: bool`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      allOf:
        - type: string
        - type: int"`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 2, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 2)
	assert.Equal(t, 2, changes.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, v3.AllOfLabel, changes.Changes[0].Property)
	assert.Equal(t, Modified, changes.AllOfChanges[0].Changes[0].ChangeType)
	assert.Equal(t, "string", changes.AllOfChanges[0].Changes[0].New)
	assert.Equal(t, "bool", changes.AllOfChanges[0].Changes[0].Original)
}

func TestCompareSchemas_AllOfEquivalentObjectRewrite_NonBreaking(t *testing.T) {
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      type: object
      required:
        - a
        - b
      properties:
        a:
          type: string
        b:
          type: string`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      allOf:
        - type: object
          required:
            - a
          properties:
            a:
              type: string
        - type: object
          required:
            - b
          properties:
            b:
              type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 2, changes.TotalChanges())
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Len(t, changes.GetAllChanges(), 2)
	for _, change := range changes.GetAllChanges() {
		assert.Equal(t, ObjectAdded, change.ChangeType)
		assert.Equal(t, v3.AllOfLabel, change.Property)
		assert.False(t, change.Breaking)
	}
}

func TestCompareSchemas_AllOfEquivalentObjectRewrite_MovedTypeAndDescription(t *testing.T) {
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      type: object
      description: payload thing
      required:
        - a
        - b
      properties:
        a:
          type: string
        b:
          type: string`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      allOf:
        - type: object
          description: payload thing
          required:
            - a
          properties:
            a:
              type: string
        - type: object
          required:
            - b
          properties:
            b:
              type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 2, changes.TotalChanges())
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	for _, change := range changes.GetAllChanges() {
		assert.Equal(t, v3.AllOfLabel, change.Property)
	}
}

func TestCompareSchemas_AllOfObjectRewrite_StillDetectsSemanticRequiredRemoval(t *testing.T) {
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      type: object
      required:
        - a
        - b
      properties:
        a:
          type: string
        b:
          type: string`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      allOf:
        - type: object
          required:
            - a
          properties:
            a:
              type: string
        - type: object
          properties:
            b:
              type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalBreakingChanges())

	foundRequiredRemoval := false
	for _, change := range changes.GetAllChanges() {
		if change.Property == v3.RequiredLabel && change.ChangeType == PropertyRemoved {
			foundRequiredRemoval = true
			assert.Equal(t, "b", change.Original)
			assert.True(t, change.Breaking)
		}
	}
	assert.True(t, foundRequiredRemoval)
}

func TestCompareSchemas_ItemsModifyAndAddItem(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      type: string
      items:
        type: bool`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      type: string
      items:
        type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.TypeLabel, changes.ItemsChanges.Changes[0].Property)
	assert.Equal(t, Modified, changes.ItemsChanges.Changes[0].ChangeType)
	assert.Equal(t, "string", changes.ItemsChanges.Changes[0].New)
	assert.Equal(t, "bool", changes.ItemsChanges.Changes[0].Original)
}

func TestCompareSchemas_ItemsModifyAndAddItemArray(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      type: string
      items:
        type: bool`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      type: string
      items:
        type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.TypeLabel, changes.ItemsChanges.Changes[0].Property)
	assert.Equal(t, Modified, changes.ItemsChanges.Changes[0].ChangeType)
	assert.Equal(t, "string", changes.ItemsChanges.Changes[0].New)
	assert.Equal(t, "bool", changes.ItemsChanges.Changes[0].Original)
}

func TestCompareSchemas_NotModifyAndAddItem(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      type: string
      not:
        type: bool`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      type: string
      not:
        type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.TypeLabel, changes.NotChanges.Changes[0].Property)
	assert.Equal(t, Modified, changes.NotChanges.Changes[0].ChangeType)
	assert.Equal(t, "string", changes.NotChanges.Changes[0].New)
	assert.Equal(t, "bool", changes.NotChanges.Changes[0].Original)
}

func TestCompareSchemas_DiscriminatorChange(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      type: string
      discriminator:
        propertyName: melody`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      type: string
      discriminator:
        propertyName: maddox`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.PropertyNameLabel, changes.DiscriminatorChanges.Changes[0].Property)
	assert.Equal(t, Modified, changes.DiscriminatorChanges.Changes[0].ChangeType)
	assert.Equal(t, "maddox", changes.DiscriminatorChanges.Changes[0].New)
	assert.Equal(t, "melody", changes.DiscriminatorChanges.Changes[0].Original)
}

func TestCompareSchemas_DiscriminatorAdd(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      type: string`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      type: string
      discriminator:
        propertyName: maddox`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.DiscriminatorLabel, changes.Changes[0].Property)
	assert.Equal(t, ObjectAdded, changes.Changes[0].ChangeType)
	// maphash uses random seed per process, just verify non-zero
	assert.NotEqual(t, uint64(0), changes.Changes[0].NewObject.(*base.Discriminator).Hash())
}

func TestCompareSchemas_DiscriminatorRemove(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      type: string`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      type: string
      discriminator:
        propertyName: maddox`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.DiscriminatorLabel, changes.Changes[0].Property)
	assert.Equal(t, ObjectRemoved, changes.Changes[0].ChangeType)
	// maphash uses random seed per process, just verify non-zero
	assert.NotEqual(t, uint64(0), changes.Changes[0].OriginalObject.(*base.Discriminator).Hash())
}

func TestCompareSchemas_ExternalDocsChange(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      type: string
      externalDocs:
        url: https://pb33f.io`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      type: string
      externalDocs:
        url: https://pb33f.io/new`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Equal(t, v3.URLLabel, changes.ExternalDocChanges.Changes[0].Property)
	assert.Equal(t, Modified, changes.ExternalDocChanges.Changes[0].ChangeType)
	assert.Equal(t, "https://pb33f.io/new", changes.ExternalDocChanges.Changes[0].New)
	assert.Equal(t, "https://pb33f.io", changes.ExternalDocChanges.Changes[0].Original)
}

func TestCompareSchemas_ExternalDocsAdd(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      type: string`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      type: string
      externalDocs:
        url: https://pb33f.io`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ExternalDocsLabel, changes.Changes[0].Property)
	assert.Equal(t, ObjectAdded, changes.Changes[0].ChangeType)
	// maphash uses random seed per process, just verify non-zero
	assert.NotEqual(t, uint64(0), changes.Changes[0].NewObject.(*base.ExternalDoc).Hash())
}

func TestCompareSchemas_ExternalDocsRemove(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      type: string`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      type: string
      externalDocs:
        url: https://pb33f.io`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ExternalDocsLabel, changes.Changes[0].Property)
	assert.Equal(t, ObjectRemoved, changes.Changes[0].ChangeType)
	// maphash uses random seed per process, just verify non-zero
	assert.NotEqual(t, uint64(0), changes.Changes[0].OriginalObject.(*base.ExternalDoc).Hash())
}

func TestCompareSchemas_AddExtension(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      type: string`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      type: string
      x-melody: song`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Equal(t, "x-melody", changes.ExtensionChanges.Changes[0].Property)
	assert.Equal(t, ObjectAdded, changes.ExtensionChanges.Changes[0].ChangeType)
	assert.Equal(t, "song", changes.ExtensionChanges.Changes[0].New)
}

func TestCompareSchemas_ExampleChange(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      example: sausages`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      example: yellow boat`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ExampleLabel, changes.Changes[0].Property)
	assert.Equal(t, Modified, changes.Changes[0].ChangeType)
	assert.Equal(t, "yellow boat", changes.Changes[0].New)
	assert.Equal(t, "sausages", changes.Changes[0].Original)
}

func TestCompareSchemas_ExampleAdd(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      title: nice`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      title: nice
      example: yellow boat`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ExampleLabel, changes.Changes[0].Property)
	assert.Equal(t, PropertyAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, "yellow boat", changes.Changes[0].New)
}

func TestCompareSchemas_ExampleRemove(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      title: nice`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      title: nice
      example: yellow boat`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ExampleLabel, changes.Changes[0].Property)
	assert.Equal(t, PropertyRemoved, changes.Changes[0].ChangeType)
	assert.Equal(t, "yellow boat", changes.Changes[0].Original)
}

func TestCompareSchemas_ExamplesChange(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      title: nice
      examples:
        - sausages`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      title: nice
      examples:
       - yellow boat`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ExamplesLabel, changes.Changes[0].Property)
	assert.Equal(t, Modified, changes.Changes[0].ChangeType)
	assert.Equal(t, "yellow boat", changes.Changes[0].New)
	assert.Equal(t, "sausages", changes.Changes[0].Original)
}

func TestCompareSchemas_ExamplesAdd(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      title: nice`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      title: nice
      examples:
       - yellow boat`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ExamplesLabel, changes.Changes[0].Property)
	assert.Equal(t, ObjectAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, "yellow boat", changes.Changes[0].New)
}

func TestCompareSchemas_ExamplesAddAndModify(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      title: nice
      examples:
        - sausages`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      title: nice
      examples:
       - yellow boat
       - seal pup`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 2, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 2)
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ExamplesLabel, changes.Changes[0].Property)
	assert.Equal(t, Modified, changes.Changes[0].ChangeType)
	assert.Equal(t, "yellow boat", changes.Changes[0].New)
	assert.Equal(t, "sausages", changes.Changes[0].Original)
	assert.Equal(t, ObjectAdded, changes.Changes[1].ChangeType)
	assert.Equal(t, "seal pup", changes.Changes[1].New)
}

func TestCompareSchemas_ExamplesRemove(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      title: nice`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      title: nice
      examples:
       - yellow boat`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ExamplesLabel, changes.Changes[0].Property)
	assert.Equal(t, ObjectRemoved, changes.Changes[0].ChangeType)
	assert.Equal(t, "yellow boat", changes.Changes[0].Original)
}

func TestCompareSchemas_ExamplesRemoveAndModify(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      title: nice
      examples:
        - sausages`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      title: nice
      examples:
       - yellow boat
       - seal pup`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 2, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 2)
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ExamplesLabel, changes.Changes[0].Property)
	assert.Equal(t, Modified, changes.Changes[0].ChangeType)
	assert.Equal(t, "yellow boat", changes.Changes[0].Original)
	assert.Equal(t, "sausages", changes.Changes[0].New)
	assert.Equal(t, ObjectRemoved, changes.Changes[1].ChangeType)
	assert.Equal(t, "seal pup", changes.Changes[1].Original)
}

func TestCompareSchemas_XMLChange(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
 schemas:
   OK:
     xml:
       name: baby xml`

	right := `openapi: 3.0
components:
 schemas:
   OK:
     xml:
       name: big xml`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.NameLabel, changes.XMLChanges.Changes[0].Property)
	assert.Equal(t, Modified, changes.XMLChanges.Changes[0].ChangeType)
	assert.Equal(t, "big xml", changes.XMLChanges.Changes[0].New)
	assert.Equal(t, "baby xml", changes.XMLChanges.Changes[0].Original)
}

func TestCompareSchemas_XMLAdd(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      description: OK`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      description: OK
      xml:
        name: big xml`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Equal(t, v3.XMLLabel, changes.Changes[0].Property)
	assert.Equal(t, ObjectAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, "big xml", changes.Changes[0].NewObject.(*base.XML).Name.Value)
}

func TestCompareSchemas_XMLRemove(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
 schemas:
   OK:`

	right := `openapi: 3.0
components:
 schemas:
   OK:
     xml:
       name: big xml`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.XMLLabel, changes.Changes[0].Property)
	assert.Equal(t, ObjectRemoved, changes.Changes[0].ChangeType)
	assert.Equal(t, "big xml", changes.Changes[0].OriginalObject.(*base.XML).Name.Value)
}

func TestCompareSchemas_SchemaRefChecks(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    Burger:
      type: object
      properties:
        fries:
          $ref: '#/components/schemas/Fries'
    Fries:
      type: object
      required:
        - potatoShape
        - favoriteDrink
        - seasoning`

	right := `openapi: 3.0
components:
  schemas:
    Burger:
      type: object
      properties:
        fries:
          $ref: '#/components/schemas/Fries'
    Fries:
      type: object
      required:
        - potatoShape
        - favoriteDrink`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	changes := CompareDocuments(leftDoc, rightDoc)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
}

func TestCompareSchemas_SchemaAdditionalPropertiesCheck(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    Dressing:
      type: object
      additionalProperties:
        type: object
        description: something in here. please`

	right := `openapi: 3.0
components:
  schemas:
    Dressing:
      type: object
      additionalProperties:
        type: object
        description: something in here. please, but changed`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	changes := CompareDocuments(leftDoc, rightDoc)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 0, changes.TotalBreakingChanges())
}

func TestCompareSchemas_Schema_DeletePoly(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    Drink:
      type: int
    SomePayload:
      type: string
      anyOf:
        - $ref: '#/components/schemas/Drink'
      allOf:
        - $ref: '#/components/schemas/Drink'`

	right := `openapi: 3.0
components:
  schemas:
    Drink:
      type: int
    SomePayload:
      type: string
      anyOf:
        - $ref: '#/components/schemas/Drink'`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	changes := CompareDocuments(leftDoc, rightDoc)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
}

func TestCompareSchemas_Schema_AddExamplesArray_AllOf(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    SomePayload:
      type: object
      allOf:
        - type: array
          items:
            type: string
          example: [ "a", "b", "c" ]`
	right := `openapi: 3.0
components:
  schemas:
    SomePayload:
      type: object
      allOf:
        - type: array
          items:
            type: string
          example: [ "a", "b", "c","d","e"]`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	changes := CompareDocuments(leftDoc, rightDoc)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 0, changes.TotalBreakingChanges())
}

func TestCompareSchemas_Schema_AddExampleMap_AllOf(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    SomePayload:
      type: object
      description: payload thing
      allOf:
        - type: object
          description: allOf thing
          example:
            - name: chicken`
	right := `openapi: 3.0
components:
  schemas:
    SomePayload:
      type: object
      description: payload thing
      allOf:
        - type: object
          description: allOf thing
          example:
            - name: nuggets`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	changes := CompareDocuments(leftDoc, rightDoc)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 0, changes.TotalBreakingChanges())
}

func TestCompareSchemas_Schema_AddExamplesArray(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    SomePayload:
      type: object
      oneOf:
        - type: array
          items:
            type: string
          example: [ "a", "b", "c" ]`
	right := `openapi: 3.0
components:
  schemas:
    SomePayload:
      type: object
      oneOf:
        - type: array
          items:
            type: string
          example: [ "a", "b", "c","d","e"]`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	changes := CompareDocuments(leftDoc, rightDoc)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 0, changes.TotalBreakingChanges())
}

func TestCompareSchemas_Schema_AddExamplesMap(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    SomePayload:
      type: object
      oneOf:
        - type: array
          items:
            type: string
          example:
            oh: my`
	right := `openapi: 3.0
components:
  schemas:
    SomePayload:
      type: object
      oneOf:
        - type: array
          items:
            type: string
          example:
            oh: why`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	changes := CompareDocuments(leftDoc, rightDoc)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 0, changes.TotalBreakingChanges())
}

func TestCompareSchemas_Schema_AddExamples(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    containerShared:
     description: Shared properties by request payload and response
     type: object
     properties:
       close_time:
         example: '2020-07-09T00:17:55Z'
       container_type:
         type: string
         enum:
           - default
           - case
         example: default
       custom_fields:
         type:
           - array
           - 'null'
         items:
           type: object`
	right := `openapi: 3.0
components:
  schemas:
    containerShared:
     description: Shared properties by request payload and response
     type: object
     properties:
       close_time:
         example: '2020-07-09T00:17:55Z'
       container_type:
         type: string
         enum:
           - default
           - case
         example: default
       custom_fields:
         type:
           - array
           - 'null'
         items:
           type: object
         example:
           - name: auditedAt
             source: global
             dataType: text
             requiredToResolve: false`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	changes := CompareDocuments(leftDoc, rightDoc)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 0, changes.TotalBreakingChanges())
}

func TestCompareSchemas_CheckIssue_170(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
 schemas:
   OK:
     type: object
     properties:
       id:
         type: integer
         format: int64
       name:
         type: string
       tag:
         type: string`

	right := `openapi: 3.0
components:
 schemas:
   OK:
     type: object
     properties:
       id:
         type: integer
         format: int64
       name:
         type: integer
         format: int64
       foo:
         type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 4, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 4)
	// Breaking changes: name.type modified (string->integer), tag removed
	// Non-breaking: name.format added (format added is not breaking by default), foo added
	assert.Equal(t, 2, changes.TotalBreakingChanges())
	assert.Len(t, changes.SchemaPropertyChanges["name"].PropertyChanges.Changes, 2)
	assert.Len(t, changes.PropertyChanges.Changes, 2)
}

func TestSchemaChanges_TotalChanges_NoNilPanic(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	var changes *SchemaChanges
	assert.Equal(t, 0, changes.TotalChanges())
}

func TestSchemaChanges_TotalBreakingChanges_NoNilPanic(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	var changes *SchemaChanges
	assert.Equal(t, 0, changes.TotalBreakingChanges())
}

func TestCompareSchemas_Nil(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	assert.Nil(t, CompareSchemas(nil, nil))
}

// Test for issue https://github.com/pb33f/libopenapi/issues/218
func TestCompareSchemas_PropertyRefChange_Identical(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    Yo:
      type: int 
    OK:
      $ref: '#/components/schemas/Yo'`

	right := `openapi: 3.0
components:
  schemas:
    Yo:
      type: int 
    OK:
      type: int`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.Nil(t, changes)
}

func TestCompareSchemas_PropertyRefChange_IdenticalReverse(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    Yo:
      type: int 
    OK:
      $ref: '#/components/schemas/Yo'`

	right := `openapi: 3.0
components:
  schemas:
    Yo:
      type: int 
    OK:
      type: int`

	leftDoc, rightDoc := test_BuildDoc(right, left)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.Nil(t, changes)
}

func TestCompareSchemas_PropertyRefChange_Fail(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    Yo:
      type: int 
    OK:
      $ref: '#/components/schemas/Yo'`

	right := `openapi: 3.0
components:
  schemas:
    Yo:
      type: int 
    OK:
      type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
}

// https://github.com/pb33f/openapi-changes/issues/104
func TestCompareSchemas_CheckOrderingDoesNotCreateChanges(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1.0
components:
  schemas:
    operatorArray:
      type: bool
    operator:
      type: bool
    Ordering:
      properties:
        three: 
          $ref: '#/components/schemas/operator'
        one: 
          $ref: '#/components/schemas/operator'
        two: 
          $ref: '#/components/schemas/operator'
        four: 
          $ref: '#/components/schemas/operatorArray'`

	right := `openapi: 3.1.0
components:
  schemas:
    operator:
      type: bool
    operatorArray:
      type: bool
    Ordering:
      properties:
        four: 
          $ref: '#/components/schemas/operatorArray'
        one: 
          $ref: '#/components/schemas/operator'
        two: 
          $ref: '#/components/schemas/operator'
        three: 
          $ref: '#/components/schemas/operator'`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("Ordering").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Ordering").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.Nil(t, changes)
	assert.Equal(t, 0, changes.TotalChanges())
}

// https://github.com/pb33f/openapi-changes/issues/108
func TestCompareSchemas_CheckRequiredOrdering(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1.0
components:
  schemas:
    Ordering:
      required:
        - one
        - three
      properties:
        three: 
          type: string
        one: 
           type: string`

	right := `openapi: 3.1.0
components:
  schemas:
    Ordering:
      required:
        - three
        - one
      properties:
        three: 
          type: string
        one: 
           type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("Ordering").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Ordering").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.Nil(t, changes)
	assert.Equal(t, 0, changes.TotalChanges())
}

func TestCompareSchemas_CheckEnums(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1.0
components:
  schemas:
    Pet:
      type: object
      properties:
        id:
          readOnly: true
          type: string
        name:
          type: string
          enum: 
            - Bob`

	right := `openapi: 3.1.0
components:
  schemas:
    Pet:
      type: object
      properties:
        id:
          readOnly: true
          type: string
        name:
          type: string
          enum: 
            - "Bob"`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("Pet").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Pet").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.Nil(t, changes)
	assert.Equal(t, 0, changes.TotalChanges())
}

// https://github.com/pb33f/openapi-changes/issues/160
func TestCompareSchemas_CheckAddProp(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1.0
components:
  schemas:
    TestResult:
      properties:
        result:
          type: string
        resultType:
          type: string`

	right := `openapi: 3.1.0
components:
  schemas:
    TestResult:
      properties:
        result:
          type: string
        testResultType:
          type: string
        test:
          type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("TestResult").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("TestResult").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 3, changes.TotalChanges())

	changes = CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 3, changes.TotalChanges())
}

// https://github.com/pb33f/openapi-changes/issues/177
func TestCompareSchemas_CheckOneOfIdenticalChange(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1.0
components:
  schemas:
    TestResult:
      type: string
      oneOf:
        - title: CUT
          const: CUT
          description: cat
        - title: HEAD
          const: HEAD
          description: sat
        - title: SCENARIO
          const: SCENARIO
          description: mat`

	right := `openapi: 3.1.0
components:
  schemas:
    TestResult:
      type: string
      oneOf:
        - title: CUT
          const: CUT
          description: bat
        - title: HEAD
          const: HEAD
          description: clap
        - title: SCENARIO
          const: SCENARIO
          description: chap`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("TestResult").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("TestResult").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 3, changes.TotalChanges())
	for _, c := range changes.OneOfChanges {
		assert.Equal(t, "description", c.PropertyChanges.Changes[0].Property)
	}
}

func TestCompareSchemas_TestSchemaLockIssue(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1.0
components:
  schemas:
    CompanyInformation:
      description: |
        Provides information about Company
      type: object
      properties:
        companyName:
          type: string
          maxLength: 100
        website:
          type: string
          maxLength: 1024
`

	right := `openapi: 3.1.0
components:
  schemas:
    CompanyInformation:
      description: |
        Provides information about Company
      type: object
      properties:
        companyName:
          type: string
          maxLength: 100
        website:
          type: string
          maxLength: 1024
        extraField:
          type: string
          maxLength: 1024`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("CompanyInformation").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("CompanyInformation").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
}

func TestCompareSchemas_TestGetPropertiesChanges(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      patternProperties:
        ketchup:
          type: number
          const: 2
      dependentSchemas:
        monkey:
          type: string
          const: fluff
      properties:
        mick:
          description: hey`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      patternProperties:
        ketchup:
          type: number
          const: 1
      dependentSchemas:
        monkey:
          type: string
          const: flaff
      properties:
        mick:
          description: hey ho`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.GetPropertyChanges(), 3)
}

func TestCompareSchemas_PrefixItems(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      prefixItems:
        - type: number
          const: 1`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      prefixItems:
        - type: number
          const: 2`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, changes.TotalChanges(), 1)
	assert.Equal(t, changes.TotalBreakingChanges(), 1)
}

func TestCompareSchemas_OneOfRemoveItem(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    OK:
      oneOf:
        - type: string
        - type: int`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      oneOf:
        - type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, changes.Changes[0].ChangeType)
	assert.Equal(t, v3.OneOfLabel, changes.Changes[0].Property)
	assert.True(t, changes.Changes[0].Breaking)
}

func TestCompareSchemas_OneOfAddReferencedBranchAppendIsNonBreaking(t *testing.T) {
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    A:
      type: string
    B:
      type: integer
    C:
      type: boolean
    Input:
      oneOf:
        - $ref: '#/components/schemas/A'
        - $ref: '#/components/schemas/B'`

	right := `openapi: 3.0
components:
  schemas:
    A:
      type: string
    B:
      type: integer
    C:
      type: boolean
    Input:
      oneOf:
        - $ref: '#/components/schemas/A'
        - $ref: '#/components/schemas/B'
        - $ref: '#/components/schemas/C'`

	leftDoc, rightDoc := test_BuildDoc(left, right)
	lSchemaProxy := leftDoc.Components.Value.FindSchema("Input").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Input").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Len(t, changes.Changes, 1)
	assert.Empty(t, changes.OneOfChanges)
	assert.Equal(t, ObjectAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, v3.OneOfLabel, changes.Changes[0].Property)
	assert.False(t, changes.Changes[0].Breaking)
}

func TestCompareSchemas_OneOfAddReferencedBranchPrependIsNonBreaking(t *testing.T) {
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    A:
      type: string
    B:
      type: integer
    C:
      type: boolean
    Input:
      oneOf:
        - $ref: '#/components/schemas/A'
        - $ref: '#/components/schemas/B'`

	right := `openapi: 3.0
components:
  schemas:
    A:
      type: string
    B:
      type: integer
    C:
      type: boolean
    Input:
      oneOf:
        - $ref: '#/components/schemas/C'
        - $ref: '#/components/schemas/A'
        - $ref: '#/components/schemas/B'`

	leftDoc, rightDoc := test_BuildDoc(left, right)
	lSchemaProxy := leftDoc.Components.Value.FindSchema("Input").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Input").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Len(t, changes.Changes, 1)
	assert.Empty(t, changes.OneOfChanges)
	assert.Equal(t, ObjectAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, v3.OneOfLabel, changes.Changes[0].Property)
	assert.False(t, changes.Changes[0].Breaking)
}

func TestCompareSchemas_AnyOfAddReferencedBranchMiddleIsNonBreaking(t *testing.T) {
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    A:
      type: string
    B:
      type: integer
    C:
      type: boolean
    Input:
      anyOf:
        - $ref: '#/components/schemas/A'
        - $ref: '#/components/schemas/B'`

	right := `openapi: 3.0
components:
  schemas:
    A:
      type: string
    B:
      type: integer
    C:
      type: boolean
    Input:
      anyOf:
        - $ref: '#/components/schemas/A'
        - $ref: '#/components/schemas/C'
        - $ref: '#/components/schemas/B'`

	leftDoc, rightDoc := test_BuildDoc(left, right)
	lSchemaProxy := leftDoc.Components.Value.FindSchema("Input").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Input").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Len(t, changes.Changes, 1)
	assert.Empty(t, changes.AnyOfChanges)
	assert.Equal(t, ObjectAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, v3.AnyOfLabel, changes.Changes[0].Property)
	assert.False(t, changes.Changes[0].Breaking)
}

func TestCompareSchemas_AllOfReferencedReorderIsNoop(t *testing.T) {
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    A:
      type: object
      properties:
        a:
          type: string
    B:
      type: object
      properties:
        b:
          type: integer
    Input:
      allOf:
        - $ref: '#/components/schemas/A'
        - $ref: '#/components/schemas/B'`

	right := `openapi: 3.0
components:
  schemas:
    A:
      type: object
      properties:
        a:
          type: string
    B:
      type: object
      properties:
        b:
          type: integer
    Input:
      allOf:
        - $ref: '#/components/schemas/B'
        - $ref: '#/components/schemas/A'`

	leftDoc, rightDoc := test_BuildDoc(left, right)
	lSchemaProxy := leftDoc.Components.Value.FindSchema("Input").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Input").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.Nil(t, changes)
}

func TestCompareSchemas_OneOfReferencedReorderIsNoop(t *testing.T) {
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    A:
      type: string
    B:
      type: integer
    C:
      type: boolean
    Input:
      oneOf:
        - $ref: '#/components/schemas/A'
        - $ref: '#/components/schemas/B'
        - $ref: '#/components/schemas/C'`

	right := `openapi: 3.0
components:
  schemas:
    A:
      type: string
    B:
      type: integer
    C:
      type: boolean
    Input:
      oneOf:
        - $ref: '#/components/schemas/C'
        - $ref: '#/components/schemas/A'
        - $ref: '#/components/schemas/B'`

	leftDoc, rightDoc := test_BuildDoc(left, right)
	lSchemaProxy := leftDoc.Components.Value.FindSchema("Input").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Input").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.Nil(t, changes)
}

func TestCompareSchemas_OneOfDuplicateReferencedReorderIsNoop(t *testing.T) {
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    A:
      type: string
    B:
      type: integer
    Input:
      oneOf:
        - $ref: '#/components/schemas/A'
        - $ref: '#/components/schemas/A'
        - $ref: '#/components/schemas/B'`

	right := `openapi: 3.0
components:
  schemas:
    A:
      type: string
    B:
      type: integer
    Input:
      oneOf:
        - $ref: '#/components/schemas/A'
        - $ref: '#/components/schemas/B'
        - $ref: '#/components/schemas/A'`

	leftDoc, rightDoc := test_BuildDoc(left, right)
	lSchemaProxy := leftDoc.Components.Value.FindSchema("Input").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Input").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.Nil(t, changes)
}

func TestCompareSchemas_OneOfReorderAndModifyStillProducesModification(t *testing.T) {
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    B:
      type: integer
    Input:
      oneOf:
        - type: string
          description: old
        - $ref: '#/components/schemas/B'`

	right := `openapi: 3.0
components:
  schemas:
    B:
      type: integer
    Input:
      oneOf:
        - $ref: '#/components/schemas/B'
        - type: string
          description: new`

	leftDoc, rightDoc := test_BuildDoc(left, right)
	lSchemaProxy := leftDoc.Components.Value.FindSchema("Input").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Input").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.Changes, 0)
	assert.Len(t, changes.OneOfChanges, 1)
	assert.Len(t, changes.OneOfChanges[0].Changes, 1)
	assert.Equal(t, Modified, changes.OneOfChanges[0].Changes[0].ChangeType)
	assert.Equal(t, v3.DescriptionLabel, changes.OneOfChanges[0].Changes[0].Property)
	assert.Equal(t, "new", changes.OneOfChanges[0].Changes[0].New)
	assert.Equal(t, "old", changes.OneOfChanges[0].Changes[0].Original)
	assert.Equal(t, 0, changes.TotalBreakingChanges())
}

func TestCompareSchemas_OneOfReorderAndTwoBranchModificationsPairLocally(t *testing.T) {
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    Input:
      oneOf:
        - type: string
          title: StringBranch
          description: left string
        - type: integer
          title: IntegerBranch
          description: left integer`

	right := `openapi: 3.0
components:
  schemas:
    Input:
      oneOf:
        - type: integer
          title: IntegerBranch
          description: right integer
        - type: string
          title: StringBranch
          description: right string`

	leftDoc, rightDoc := test_BuildDoc(left, right)
	lSchemaProxy := leftDoc.Components.Value.FindSchema("Input").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Input").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.Changes, 0)
	assert.Len(t, changes.OneOfChanges, 2)
	assert.Equal(t, 2, changes.TotalChanges())
	assert.Equal(t, 0, changes.TotalBreakingChanges())

	first := changes.OneOfChanges[0]
	second := changes.OneOfChanges[1]
	assert.Equal(t, "left string", first.Changes[0].Original)
	assert.Equal(t, "right string", first.Changes[0].New)
	assert.Equal(t, "left integer", second.Changes[0].Original)
	assert.Equal(t, "right integer", second.Changes[0].New)
}

func TestExtractOrderInsensitiveSchemaChanges_ExactRefPairsCanProduceChanges(t *testing.T) {
	low.ClearHashCache()
	leftProxy := buildPreloadedRefSchemaProxy(t, "#/components/schemas/Branch", "type: string\ndescription: left")
	rightProxy := buildPreloadedRefSchemaProxy(t, "#/components/schemas/Branch", "type: string\ndescription: right")

	var schemaChanges []*SchemaChanges
	var changes []*Change
	extractOrderInsensitiveSchemaChanges(
		[]low.ValueReference[*base.SchemaProxy]{{Value: leftProxy}},
		[]low.ValueReference[*base.SchemaProxy]{{Value: rightProxy}},
		v3.OneOfLabel,
		&schemaChanges,
		&changes,
	)

	assert.Len(t, schemaChanges, 1)
	assert.Empty(t, changes)
	assert.Equal(t, "left", schemaChanges[0].Changes[0].Original)
	assert.Equal(t, "right", schemaChanges[0].Changes[0].New)
}

func setSpecIndexAbsolutePath(idx *index.SpecIndex, absolutePath string) {
	field := reflect.ValueOf(idx).Elem().FieldByName("specAbsolutePath")
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().SetString(absolutePath)
}

func buildPreloadedRefSchemaProxy(t *testing.T, ref, rawSchema string) *base.SchemaProxy {
	t.Helper()

	var node yaml.Node
	assert.NoError(t, yaml.Unmarshal([]byte(rawSchema), &node))

	var schema base.Schema
	assert.NoError(t, low.BuildModel(node.Content[0], &schema))
	assert.NoError(t, schema.Build(context.Background(), node.Content[0], nil))

	idx := index.NewSpecIndex(&yaml.Node{Content: []*yaml.Node{{Content: []*yaml.Node{}}}})
	setSpecIndexAbsolutePath(idx, "/tmp/schema.yaml")

	proxy := &base.SchemaProxy{}
	assert.NoError(t, proxy.Build(context.Background(), nil, utils.CreateRefNode(ref), idx))
	setSchemaProxyRendered(proxy, &schema)
	markSchemaProxyBuilt(proxy)
	return proxy
}

func buildSchemaProxyFromYAML(t *testing.T, raw string) *base.SchemaProxy {
	t.Helper()

	var node yaml.Node
	assert.NoError(t, yaml.Unmarshal([]byte(raw), &node))

	proxy := &base.SchemaProxy{}
	assert.NoError(t, proxy.Build(context.Background(), nil, node.Content[0], nil))
	return proxy
}

func buildSchemaFromYAML(t *testing.T, raw string) (*base.SchemaProxy, *base.Schema) {
	t.Helper()

	proxy := buildSchemaProxyFromYAML(t, raw)
	schema := proxy.Schema()
	assert.NotNil(t, schema)
	return proxy, schema
}

func buildPropertiesMap(entries map[string]*base.SchemaProxy) *orderedmap.Map[low.KeyReference[string], low.ValueReference[*base.SchemaProxy]] {
	props := orderedmap.New[low.KeyReference[string], low.ValueReference[*base.SchemaProxy]]()
	for key, value := range entries {
		props.Set(low.KeyReference[string]{Value: key}, low.ValueReference[*base.SchemaProxy]{Value: value})
	}
	return props
}

func buildRenderedSchemaProxy(t *testing.T, schema *base.Schema) *base.SchemaProxy {
	t.Helper()

	proxy := &base.SchemaProxy{}
	assert.NoError(t, proxy.Build(context.Background(), nil, utils.CreateEmptyMapNode(), nil))
	setSchemaProxyRendered(proxy, schema)
	markSchemaProxyBuilt(proxy)
	return proxy
}

func setSchemaProxyRendered(proxy *base.SchemaProxy, schema *base.Schema) {
	field := reflect.ValueOf(proxy).Elem().FieldByName("rendered")
	rendered := reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
	rendered.Addr().Interface().(*atomic.Pointer[base.Schema]).Store(schema)
}

func markSchemaProxyBuilt(proxy *base.SchemaProxy) {
	field := reflect.ValueOf(proxy).Elem().FieldByName("schemaOnce")
	done := sync.Once{}
	done.Do(func() {})
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(done))
}

func TestSchemaCompositionEntryStableKey(t *testing.T) {
	assert.Equal(t, "nil", schemaCompositionEntryStableKey(nil))

	refProxy := &base.SchemaProxy{}
	refNode := utils.CreateRefNode("#/components/schemas/Test")
	assert.NoError(t, refProxy.Build(context.Background(), nil, refNode, nil))
	assert.Equal(t, "ref:#/components/schemas/Test", schemaCompositionEntryStableKey(refProxy))

	inlineNode := utils.CreateEmptyMapNode()
	inlineNode.Content = append(inlineNode.Content,
		utils.CreateStringNode("type"),
		utils.CreateStringNode("string"),
	)
	inlineProxy := &base.SchemaProxy{}
	assert.NoError(t, inlineProxy.Build(context.Background(), nil, inlineNode, nil))
	assert.Contains(t, schemaCompositionEntryStableKey(inlineProxy), "type:string")

	complexNode := utils.CreateEmptyMapNode()
	complexNode.Content = append(complexNode.Content,
		utils.CreateStringNode("title"),
		utils.CreateStringNode("ComplexBranch"),
		utils.CreateStringNode("type"),
		&yaml.Node{
			Kind: yaml.SequenceNode,
			Tag:  "!!seq",
			Content: []*yaml.Node{
				utils.CreateStringNode("integer"),
				utils.CreateStringNode("null"),
			},
		},
		utils.CreateStringNode("properties"),
		&yaml.Node{
			Kind: yaml.MappingNode,
			Tag:  "!!map",
			Content: []*yaml.Node{
				utils.CreateStringNode("id"),
				utils.CreateEmptyMapNode(),
			},
		},
	)
	complexProxy := &base.SchemaProxy{}
	assert.NoError(t, complexProxy.Build(context.Background(), nil, complexNode, nil))
	complexKey := schemaCompositionEntryStableKey(complexProxy)
	assert.Contains(t, complexKey, "title:ComplexBranch")
	assert.Contains(t, complexKey, "types:integer,null")
	assert.Contains(t, complexKey, "props:id")

	emptyProxy := &base.SchemaProxy{}
	assert.NoError(t, emptyProxy.Build(context.Background(), nil, utils.CreateEmptyMapNode(), nil))
	assert.Contains(t, schemaCompositionEntryStableKey(emptyProxy), "hash:")

	invalidProxy := &base.SchemaProxy{}
	assert.NoError(t, invalidProxy.Build(context.Background(), nil, utils.CreateStringNode("not-a-schema"), nil))
	assert.Contains(t, schemaCompositionEntryStableKey(invalidProxy), "hash:")

	unbuiltProxy := &base.SchemaProxy{}
	assert.Contains(t, schemaCompositionEntryStableKey(unbuiltProxy), "hash:")
}

func TestSelectBestSchemaCompositionPairCandidates(t *testing.T) {
	best := selectBestSchemaCompositionPairCandidates(nil)
	assert.Nil(t, best)

	matrix := [][]schemaCompositionPairCandidate{
		{
			{left: schemaCompositionEntry{position: 0}, right: schemaCompositionEntry{position: 0}, totalChanges: 4, stableKeyMatch: false},
			{left: schemaCompositionEntry{position: 0}, right: schemaCompositionEntry{position: 1}, totalChanges: 1, stableKeyMatch: true},
		},
		{
			{left: schemaCompositionEntry{position: 1}, right: schemaCompositionEntry{position: 0}, totalChanges: 1, stableKeyMatch: true},
			{left: schemaCompositionEntry{position: 1}, right: schemaCompositionEntry{position: 1}, totalChanges: 4, stableKeyMatch: false},
		},
	}

	best = selectBestSchemaCompositionPairCandidates(matrix)
	assert.Len(t, best, 2)
	assert.Equal(t, 1, best[0].right.position)
	assert.Equal(t, 0, best[1].right.position)

	tieMatrix := [][]schemaCompositionPairCandidate{
		{
			{left: schemaCompositionEntry{position: 0}, right: schemaCompositionEntry{position: 0}, totalChanges: 1, stableKeyMatch: false},
			{left: schemaCompositionEntry{position: 0}, right: schemaCompositionEntry{position: 1}, totalChanges: 1, stableKeyMatch: true},
		},
		{
			{left: schemaCompositionEntry{position: 1}, right: schemaCompositionEntry{position: 0}, totalChanges: 1, stableKeyMatch: true},
			{left: schemaCompositionEntry{position: 1}, right: schemaCompositionEntry{position: 1}, totalChanges: 1, stableKeyMatch: false},
		},
	}

	best = selectBestSchemaCompositionPairCandidates(tieMatrix)
	assert.Len(t, best, 2)
	assert.True(t, best[0].stableKeyMatch)
	assert.True(t, best[1].stableKeyMatch)
}

func TestSchemaCompositionPairingHelpers(t *testing.T) {
	signature := schemaCompositionPairingSignature([]schemaCompositionPairCandidate{
		{left: schemaCompositionEntry{position: 1}, right: schemaCompositionEntry{position: 2}},
		{left: schemaCompositionEntry{position: 0}, right: schemaCompositionEntry{position: 1}},
	})
	assert.Equal(t, []int{0, 1, 1, 2}, signature)

	signature = schemaCompositionPairingSignature([]schemaCompositionPairCandidate{
		{left: schemaCompositionEntry{position: 0}, right: schemaCompositionEntry{position: 2}},
		{left: schemaCompositionEntry{position: 0}, right: schemaCompositionEntry{position: 1}},
	})
	assert.Equal(t, []int{0, 1, 0, 2}, signature)

	assert.True(t, schemaCompositionPairingIsBetter(
		[]schemaCompositionPairCandidate{{left: schemaCompositionEntry{position: 0}, right: schemaCompositionEntry{position: 0}, totalChanges: 1, stableKeyMatch: true, breakingChanges: 0}},
		[]schemaCompositionPairCandidate{{left: schemaCompositionEntry{position: 0}, right: schemaCompositionEntry{position: 1}, totalChanges: 1, stableKeyMatch: true, breakingChanges: 1}},
	))
	assert.False(t, schemaCompositionPairingIsBetter(
		[]schemaCompositionPairCandidate{{left: schemaCompositionEntry{position: 0}, right: schemaCompositionEntry{position: 1}, totalChanges: 1, stableKeyMatch: true, breakingChanges: 1}},
		[]schemaCompositionPairCandidate{{left: schemaCompositionEntry{position: 0}, right: schemaCompositionEntry{position: 0}, totalChanges: 1, stableKeyMatch: true, breakingChanges: 0}},
	))
	assert.False(t, schemaCompositionPairingIsBetter(
		[]schemaCompositionPairCandidate{{left: schemaCompositionEntry{position: 0}, right: schemaCompositionEntry{position: 0}, totalChanges: 1, stableKeyMatch: true, breakingChanges: 0}},
		[]schemaCompositionPairCandidate{{left: schemaCompositionEntry{position: 0}, right: schemaCompositionEntry{position: 0}, totalChanges: 1, stableKeyMatch: true, breakingChanges: 0}},
	))

	assert.Nil(t, pairUnmatchedSchemaCompositionEntries(nil, []schemaCompositionEntry{{position: 0}}))
	assert.Nil(t, pairUnmatchedSchemaCompositionEntries([]schemaCompositionEntry{{position: 0}}, nil))
	pairs := pairUnmatchedSchemaCompositionEntries(
		[]schemaCompositionEntry{{position: 0}, {position: 0}},
		[]schemaCompositionEntry{{position: 1}, {position: 0}},
	)
	assert.Len(t, pairs, 2)
	assert.Equal(t, 0, pairs[0].right.position)
	assert.Equal(t, 1, pairs[1].right.position)
}

func TestSimpleAllOfObjectHelpers(t *testing.T) {
	low.ClearHashCache()

	t.Run("isSimpleAllOfObjectSchema", func(t *testing.T) {
		assert.False(t, isSimpleAllOfObjectSchema(nil, nil))

		refProxy := &base.SchemaProxy{}
		assert.NoError(t, refProxy.Build(context.Background(), nil, utils.CreateRefNode("#/components/schemas/Test"), nil))
		assert.False(t, isSimpleAllOfObjectSchema(refProxy, nil))

		proxy, schema := buildSchemaFromYAML(t, `type: object`)
		assert.False(t, isSimpleAllOfObjectSchema(proxy, schema))

		proxy, schema = buildSchemaFromYAML(t, `allOf:
  - type: object
anyOf:
  - type: string`)
		assert.False(t, isSimpleAllOfObjectSchema(proxy, schema))

		proxy = buildSchemaProxyFromYAML(t, `allOf: []`)
		schema = &base.Schema{
			AllOf: low.NodeReference[[]low.ValueReference[*base.SchemaProxy]]{
				Value: []low.ValueReference[*base.SchemaProxy]{
					{Value: refProxy},
				},
			},
		}
		assert.False(t, isSimpleAllOfObjectSchema(proxy, schema))

		proxy, schema = buildSchemaFromYAML(t, `allOf:
  - type: object
    additionalProperties: true`)
		assert.False(t, isSimpleAllOfObjectSchema(proxy, schema))

		nilBranchProxy := buildSchemaProxyFromYAML(t, `type: object`)
		markSchemaProxyBuilt(nilBranchProxy)
		setSchemaProxyRendered(nilBranchProxy, nil)
		proxy = buildSchemaProxyFromYAML(t, `allOf: []`)
		schema = &base.Schema{
			AllOf: low.NodeReference[[]low.ValueReference[*base.SchemaProxy]]{
				Value: []low.ValueReference[*base.SchemaProxy]{
					{Value: nilBranchProxy},
				},
			},
		}
		assert.False(t, isSimpleAllOfObjectSchema(proxy, schema))

		proxy, schema = buildSchemaFromYAML(t, `allOf:
  - type: object
    not:
      type: string`)
		assert.False(t, isSimpleAllOfObjectSchema(proxy, schema))

		proxy, schema = buildSchemaFromYAML(t, `allOf:
  - type: object
    properties:
      id:
        type: string`)
		assert.True(t, isSimpleAllOfObjectSchema(proxy, schema))
	})

	t.Run("schemaComparisonViewForSimpleAllOfObject", func(t *testing.T) {
		proxy, schema := buildSchemaFromYAML(t, `type: object
description: same
title: merged
properties:
  id:
    type: string
required:
  - id
allOf:
  - type: object
    description: same
    title: merged
    properties:
      name:
        type: string
    required:
      - name`)

		merged := schemaComparisonViewForSimpleAllOfObject(proxy, schema)
		assert.NotNil(t, merged)
		assert.NotSame(t, schema, merged)
		assert.Equal(t, "same", merged.Description.Value)
		assert.Equal(t, "merged", merged.Title.Value)
		assert.Equal(t, 2, merged.Properties.Value.Len())
		assert.Len(t, merged.Required.Value, 2)

		_, conflictSchema := buildSchemaFromYAML(t, `type: object
allOf:
  - type: string`)
		assert.Same(t, conflictSchema, schemaComparisonViewForSimpleAllOfObject(proxy, conflictSchema))

		merged, ok := mergeSimpleAllOfObjectSchemaView(nil)
		assert.False(t, ok)
		assert.Nil(t, merged)

		_, descriptionConflict := buildSchemaFromYAML(t, `type: object
description: alpha
allOf:
  - type: object
    description: beta`)
		merged, ok = mergeSimpleAllOfObjectSchemaView(descriptionConflict)
		assert.False(t, ok)
		assert.Nil(t, merged)

		_, titleConflict := buildSchemaFromYAML(t, `type: object
title: alpha
allOf:
  - type: object
    title: beta`)
		merged, ok = mergeSimpleAllOfObjectSchemaView(titleConflict)
		assert.False(t, ok)
		assert.Nil(t, merged)

		_, propertiesConflict := buildSchemaFromYAML(t, `type: object
properties:
  id:
    type: string
allOf:
  - type: object
    properties:
      id:
        type: integer`)
		merged, ok = mergeSimpleAllOfObjectSchemaView(propertiesConflict)
		assert.False(t, ok)
		assert.Nil(t, merged)
	})

	t.Run("mergeSimpleAllOfObjectType", func(t *testing.T) {
		selected, ok := mergeSimpleAllOfObjectType(&base.Schema{
			Type: low.NodeReference[base.SchemaDynamicValue[string, []low.ValueReference[string]]]{
				KeyNode:   utils.CreateStringNode("type"),
				ValueNode: utils.CreateEmptySequenceNode(),
				Value: base.SchemaDynamicValue[string, []low.ValueReference[string]]{
					N: 1,
					B: []low.ValueReference[string]{{Value: "string"}},
				},
			},
		})
		assert.False(t, ok)
		assert.True(t, selected.Value.IsB())

		selected, ok = mergeSimpleAllOfObjectType(&base.Schema{
			Type: low.NodeReference[base.SchemaDynamicValue[string, []low.ValueReference[string]]]{
				KeyNode:   utils.CreateStringNode("type"),
				ValueNode: utils.CreateStringNode("string"),
				Value: base.SchemaDynamicValue[string, []low.ValueReference[string]]{
					A: "string",
				},
			},
		})
		assert.False(t, ok)
		assert.Equal(t, "string", selected.Value.A)

		selected, ok = mergeSimpleAllOfObjectType(&base.Schema{
			AllOf: low.NodeReference[[]low.ValueReference[*base.SchemaProxy]]{
				Value: []low.ValueReference[*base.SchemaProxy]{
					{Value: buildSchemaProxyFromYAML(t, `description: branch only`)},
					{Value: buildSchemaProxyFromYAML(t, `type:
  - string
  - 'null'`)},
				},
			},
		})
		assert.False(t, ok)
		assert.Equal(t, "", selected.Value.A)

		selected, ok = mergeSimpleAllOfObjectType(&base.Schema{
			AllOf: low.NodeReference[[]low.ValueReference[*base.SchemaProxy]]{
				Value: []low.ValueReference[*base.SchemaProxy]{
					{Value: buildSchemaProxyFromYAML(t, `type: string`)},
				},
			},
		})
		assert.False(t, ok)
		assert.Equal(t, "", selected.Value.A)

		selected, ok = mergeSimpleAllOfObjectType(&base.Schema{
			Type: low.NodeReference[base.SchemaDynamicValue[string, []low.ValueReference[string]]]{
				KeyNode:   utils.CreateStringNode("type"),
				ValueNode: utils.CreateStringNode(""),
			},
			AllOf: low.NodeReference[[]low.ValueReference[*base.SchemaProxy]]{
				Value: []low.ValueReference[*base.SchemaProxy]{
					{Value: buildSchemaProxyFromYAML(t, `type: object`)},
				},
			},
		})
		assert.False(t, ok)

		selected, ok = mergeSimpleAllOfObjectType(&base.Schema{
			Type: low.NodeReference[base.SchemaDynamicValue[string, []low.ValueReference[string]]]{
				KeyNode:   utils.CreateStringNode("type"),
				ValueNode: utils.CreateStringNode("object"),
				Value: base.SchemaDynamicValue[string, []low.ValueReference[string]]{
					A: "object",
				},
			},
			AllOf: low.NodeReference[[]low.ValueReference[*base.SchemaProxy]]{
				Value: []low.ValueReference[*base.SchemaProxy]{
					{Value: buildSchemaProxyFromYAML(t, `type: object`)},
				},
			},
		})
		assert.True(t, ok)
		assert.Equal(t, "object", selected.Value.A)
	})

	t.Run("mergeCompatibleStringNodeReference", func(t *testing.T) {
		selected, ok := mergeCompatibleStringNodeReference(
			low.NodeReference[string]{},
			[]low.ValueReference[*base.SchemaProxy]{
				{Value: &base.SchemaProxy{}},
				{Value: buildSchemaProxyFromYAML(t, `type: object`)},
				{Value: buildSchemaProxyFromYAML(t, `description: alpha`)},
				{Value: buildSchemaProxyFromYAML(t, `description: alpha`)},
			},
			func(branch *base.Schema) low.NodeReference[string] {
				return branch.Description
			},
		)
		assert.True(t, ok)
		assert.Equal(t, "alpha", selected.Value)

		selected, ok = mergeCompatibleStringNodeReference(
			low.NodeReference[string]{Value: "alpha"},
			[]low.ValueReference[*base.SchemaProxy]{
				{Value: buildSchemaProxyFromYAML(t, `description: beta`)},
			},
			func(branch *base.Schema) low.NodeReference[string] {
				return branch.Description
			},
		)
		assert.False(t, ok)
		assert.Equal(t, "alpha", selected.Value)
	})

	t.Run("mergeSimpleAllOfObjectProperties", func(t *testing.T) {
		emptyRef, ok := mergeSimpleAllOfObjectProperties(&base.Schema{
			AllOf: low.NodeReference[[]low.ValueReference[*base.SchemaProxy]]{
				Value: []low.ValueReference[*base.SchemaProxy]{
					{Value: buildSchemaProxyFromYAML(t, `type: object`)},
				},
			},
		})
		assert.True(t, ok)
		assert.Nil(t, emptyRef.Value)

		stringProxy := buildSchemaProxyFromYAML(t, `type: string`)
		stringProxySame := buildSchemaProxyFromYAML(t, `type: string`)
		intProxy := buildSchemaProxyFromYAML(t, `type: integer`)
		mergedRef, ok := mergeSimpleAllOfObjectProperties(&base.Schema{
			Properties: low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*base.SchemaProxy]]]{
				Value: buildPropertiesMap(map[string]*base.SchemaProxy{"id": stringProxy}),
			},
			AllOf: low.NodeReference[[]low.ValueReference[*base.SchemaProxy]]{
				Value: []low.ValueReference[*base.SchemaProxy]{
					{Value: buildSchemaProxyFromYAML(t, `properties:
  id:
    type: string
  name:
    type: string`)},
				},
			},
		})
		assert.True(t, ok)
		assert.Equal(t, 2, mergedRef.Value.Len())
		assert.Equal(t, stringProxy.Hash(), mergedRef.Value.GetOrZero(low.KeyReference[string]{Value: "id"}).Value.Hash())

		nilMergedRef, ok := mergeSimpleAllOfObjectProperties(&base.Schema{
			Properties: low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*base.SchemaProxy]]]{
				Value: buildPropertiesMap(map[string]*base.SchemaProxy{"id": nil}),
			},
			AllOf: low.NodeReference[[]low.ValueReference[*base.SchemaProxy]]{
				Value: []low.ValueReference[*base.SchemaProxy]{
					{Value: buildRenderedSchemaProxy(t, &base.Schema{
						Properties: low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*base.SchemaProxy]]]{
							Value: buildPropertiesMap(map[string]*base.SchemaProxy{"id": nil}),
						},
					})},
				},
			},
		})
		assert.True(t, ok)
		assert.Equal(t, 1, nilMergedRef.Value.Len())
		assert.Nil(t, nilMergedRef.Value.GetOrZero(low.KeyReference[string]{Value: "id"}).Value)

		_, ok = mergeSimpleAllOfObjectProperties(&base.Schema{
			Properties: low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*base.SchemaProxy]]]{
				Value: buildPropertiesMap(map[string]*base.SchemaProxy{"id": nil}),
			},
			AllOf: low.NodeReference[[]low.ValueReference[*base.SchemaProxy]]{
				Value: []low.ValueReference[*base.SchemaProxy]{
					{Value: buildSchemaProxyFromYAML(t, `properties:
  id:
    type: string`)},
				},
			},
		})
		assert.False(t, ok)

		_, ok = mergeSimpleAllOfObjectProperties(&base.Schema{
			Properties: low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*base.SchemaProxy]]]{
				Value: buildPropertiesMap(map[string]*base.SchemaProxy{"id": stringProxySame}),
			},
			AllOf: low.NodeReference[[]low.ValueReference[*base.SchemaProxy]]{
				Value: []low.ValueReference[*base.SchemaProxy]{
					{Value: buildSchemaProxyFromYAML(t, `properties:
  id:
    type: integer`)},
				},
			},
		})
		assert.False(t, ok)

		duplicateBaseProps := orderedmap.New[low.KeyReference[string], low.ValueReference[*base.SchemaProxy]]()
		duplicateBaseProps.Set(low.KeyReference[string]{Value: "id", KeyNode: utils.CreateStringNode("id")}, low.ValueReference[*base.SchemaProxy]{Value: stringProxySame})
		duplicateBaseProps.Set(low.KeyReference[string]{Value: "id", KeyNode: utils.CreateStringNode("id")}, low.ValueReference[*base.SchemaProxy]{Value: intProxy})
		_, ok = mergeSimpleAllOfObjectProperties(&base.Schema{
			Properties: low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*base.SchemaProxy]]]{
				Value: duplicateBaseProps,
			},
		})
		assert.False(t, ok)

		_, ok = mergeSimpleAllOfObjectProperties(&base.Schema{
			AllOf: low.NodeReference[[]low.ValueReference[*base.SchemaProxy]]{
				Value: []low.ValueReference[*base.SchemaProxy]{
					{Value: buildSchemaProxyFromYAML(t, `properties:
  id:
    type: string`)},
				},
			},
		})
		assert.True(t, ok)
	})

	t.Run("mergeSimpleAllOfRequired", func(t *testing.T) {
		required := mergeSimpleAllOfRequired(&base.Schema{
			Required: low.NodeReference[[]low.ValueReference[string]]{
				Value: []low.ValueReference[string]{{Value: "id"}},
			},
			AllOf: low.NodeReference[[]low.ValueReference[*base.SchemaProxy]]{
				Value: []low.ValueReference[*base.SchemaProxy]{
					{Value: buildSchemaProxyFromYAML(t, `required:
  - id
  - name`)},
					{Value: &base.SchemaProxy{}},
				},
			},
		})
		assert.Len(t, required.Value, 2)
		assert.Equal(t, "id", required.Value[0].Value)
		assert.Equal(t, "name", required.Value[1].Value)

		required = mergeSimpleAllOfRequired(&base.Schema{})
		assert.Nil(t, required.Value)
	})
}

func TestSimpleScalarUnionHelpers(t *testing.T) {
	low.ClearHashCache()

	t.Run("schemaNodeHasOnlyAllowedKeys", func(t *testing.T) {
		assert.False(t, schemaNodeHasOnlyAllowedKeys(nil, simpleAllOfObjectBranchKeys))
		assert.False(t, schemaNodeHasOnlyAllowedKeys(utils.CreateStringNode("type"), simpleAllOfObjectBranchKeys))
		assert.True(t, schemaNodeHasOnlyAllowedKeys(utils.CreateEmptyMapNode(), simpleAllOfObjectBranchKeys))

		node := utils.CreateEmptyMapNode()
		node.Content = append(node.Content, utils.CreateStringNode("description"), utils.CreateStringNode("ok"))
		assert.True(t, schemaNodeHasOnlyAllowedKeys(node, simpleAllOfObjectBranchKeys))
		node.Content = append(node.Content, utils.CreateStringNode("additionalProperties"), utils.CreateStringNode("true"))
		assert.False(t, schemaNodeHasOnlyAllowedKeys(node, simpleAllOfObjectBranchKeys))
	})

	t.Run("extractScalarTypeName", func(t *testing.T) {
		value, ok := extractScalarTypeName(nil)
		assert.False(t, ok)
		assert.Equal(t, "", value)

		value, ok = extractScalarTypeName(utils.CreateEmptyMapNode())
		assert.False(t, ok)
		assert.Equal(t, "", value)

		nullNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null"}
		value, ok = extractScalarTypeName(nullNode)
		assert.True(t, ok)
		assert.Equal(t, "null", value)

		value, ok = extractScalarTypeName(utils.CreateStringNode(""))
		assert.False(t, ok)
		assert.Equal(t, "", value)

		value, ok = extractScalarTypeName(utils.CreateStringNode("string"))
		assert.True(t, ok)
		assert.Equal(t, "string", value)
	})

	t.Run("extractTypeArraySet", func(t *testing.T) {
		typeSet, ok := extractTypeArraySet(nil)
		assert.False(t, ok)
		assert.Nil(t, typeSet)

		refProxy := &base.SchemaProxy{}
		assert.NoError(t, refProxy.Build(context.Background(), nil, utils.CreateRefNode("#/components/schemas/Test"), nil))
		typeSet, ok = extractTypeArraySet(refProxy)
		assert.False(t, ok)
		assert.Nil(t, typeSet)

		typeSet, ok = extractTypeArraySet(buildSchemaProxyFromYAML(t, `type: string`))
		assert.False(t, ok)
		assert.Nil(t, typeSet)

		typeSet, ok = extractTypeArraySet(buildSchemaProxyFromYAML(t, `type: []`))
		assert.False(t, ok)
		assert.Nil(t, typeSet)

		typeSet, ok = extractTypeArraySet(buildSchemaProxyFromYAML(t, `type:
  - {}`))
		assert.False(t, ok)
		assert.Nil(t, typeSet)

		typeSet, ok = extractTypeArraySet(buildSchemaProxyFromYAML(t, `type:
  - string
  - 'null'
  - string`))
		assert.True(t, ok)
		assert.Len(t, typeSet, 2)
	})

	t.Run("pureAnyOfAndTypeSetHelpers", func(t *testing.T) {
		assert.False(t, isPureAnyOfUnionSchema(nil))

		refProxy := &base.SchemaProxy{}
		assert.NoError(t, refProxy.Build(context.Background(), nil, utils.CreateRefNode("#/components/schemas/Test"), nil))
		assert.False(t, isPureAnyOfUnionSchema(refProxy))

		assert.False(t, isPureAnyOfUnionSchema(buildSchemaProxyFromYAML(t, `anyOf: []`)))
		assert.False(t, isPureAnyOfUnionSchema(buildSchemaProxyFromYAML(t, `anyOf:
  - type: string
description: extra`)))
		assert.True(t, isPureAnyOfUnionSchema(buildSchemaProxyFromYAML(t, `anyOf:
  - type: string`)))

		typeSet, ok := extractSimpleAnyOfTypeSet(nil)
		assert.False(t, ok)
		assert.Nil(t, typeSet)

		typeSet, ok = extractSimpleAnyOfTypeSet([]low.ValueReference[*base.SchemaProxy]{{Value: nil}})
		assert.False(t, ok)
		assert.Nil(t, typeSet)

		typeSet, ok = extractSimpleAnyOfTypeSet([]low.ValueReference[*base.SchemaProxy]{{Value: refProxy}})
		assert.False(t, ok)
		assert.Nil(t, typeSet)

		typeSet, ok = extractSimpleAnyOfTypeSet([]low.ValueReference[*base.SchemaProxy]{
			{Value: buildSchemaProxyFromYAML(t, `type: string
description: extra`)},
		})
		assert.False(t, ok)
		assert.Nil(t, typeSet)

		typeSet, ok = extractSimpleAnyOfTypeSet([]low.ValueReference[*base.SchemaProxy]{
			{Value: buildSchemaProxyFromYAML(t, `type: ""`)},
		})
		assert.False(t, ok)
		assert.Nil(t, typeSet)

		typeSet, ok = extractSimpleAnyOfTypeSet([]low.ValueReference[*base.SchemaProxy]{
			{Value: buildSchemaProxyFromYAML(t, `type: string`)},
			{Value: buildSchemaProxyFromYAML(t, `type: 'null'`)},
		})
		assert.True(t, ok)
		assert.Len(t, typeSet, 2)
	})

	t.Run("schemaPairUsesEquivalentSimpleScalarUnion", func(t *testing.T) {
		assert.False(t, schemaPairUsesEquivalentSimpleScalarUnion(
			buildSchemaProxyFromYAML(t, `type: string`),
			buildSchemaProxyFromYAML(t, `anyOf:
  - type: string`),
		))

		assert.False(t, schemaPairUsesEquivalentSimpleScalarUnion(
			buildSchemaProxyFromYAML(t, `type:
  - {}
  - 'null'`),
			buildSchemaProxyFromYAML(t, `anyOf:
  - type: string
  - type: 'null'`),
		))

		assert.False(t, schemaPairUsesEquivalentSimpleScalarUnion(
			buildSchemaProxyFromYAML(t, `type:
  - string
  - 'null'`),
			buildSchemaProxyFromYAML(t, `anyOf:
  - type: string`),
		))

		assert.False(t, schemaPairUsesEquivalentSimpleScalarUnion(
			buildSchemaProxyFromYAML(t, `type:
  - string
  - 'null'`),
			buildSchemaProxyFromYAML(t, `anyOf:
  - type: string
  - type: integer`),
		))

		assert.True(t, schemaPairUsesEquivalentSimpleScalarUnion(
			buildSchemaProxyFromYAML(t, `type:
  - string
  - 'null'`),
			buildSchemaProxyFromYAML(t, `anyOf:
  - type: string
  - type: 'null'`),
		))
	})
}

func TestExtractSchemaChanges_PrefixItemsRemovedAndModified(t *testing.T) {
	low.ClearHashCache()

	left := []low.ValueReference[*base.SchemaProxy]{
		{Value: buildSchemaProxyFromYAML(t, `type: string`)},
		{Value: buildSchemaProxyFromYAML(t, `type: number`)},
	}
	right := []low.ValueReference[*base.SchemaProxy]{
		{Value: buildSchemaProxyFromYAML(t, `type: integer`)},
	}

	var schemaChanges []*SchemaChanges
	var changes []*Change
	extractSchemaChanges(left, right, v3.PrefixItemsLabel, &schemaChanges, &changes)

	assert.Len(t, schemaChanges, 1)
	assert.Len(t, changes, 1)
	assert.Equal(t, Modified, schemaChanges[0].Changes[0].ChangeType)
	assert.Equal(t, ObjectRemoved, changes[0].ChangeType)
	assert.True(t, changes[0].Breaking)
}

func TestExtractSchemaChanges_PrefixItemsAddedAndModified(t *testing.T) {
	low.ClearHashCache()

	left := []low.ValueReference[*base.SchemaProxy]{
		{Value: buildSchemaProxyFromYAML(t, `type: string`)},
	}
	right := []low.ValueReference[*base.SchemaProxy]{
		{Value: buildSchemaProxyFromYAML(t, `type: integer`)},
		{Value: buildSchemaProxyFromYAML(t, `type: boolean`)},
	}

	var schemaChanges []*SchemaChanges
	var changes []*Change
	extractSchemaChanges(left, right, v3.PrefixItemsLabel, &schemaChanges, &changes)

	assert.Len(t, schemaChanges, 1)
	assert.Len(t, changes, 1)
	assert.Equal(t, Modified, schemaChanges[0].Changes[0].ChangeType)
	assert.Equal(t, ObjectAdded, changes[0].ChangeType)
	assert.False(t, changes[0].Breaking)
}

func TestCompareSchemas_PrefixItemsRemoveItem(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      prefixItems:
        - type: number
        - type: string`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      prefixItems:
        - type: number`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, changes.Changes[0].ChangeType)
	assert.Equal(t, v3.PrefixItemsLabel, changes.Changes[0].Property)
	assert.True(t, changes.Changes[0].Breaking)
}

func TestCompareSchemas_PrefixItemsAddItem(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      prefixItems:
        - type: number`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      prefixItems:
        - type: number
        - type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	// prefixItems addition is non-breaking by default
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, v3.PrefixItemsLabel, changes.Changes[0].Property)
	assert.False(t, changes.Changes[0].Breaking)
}

func TestCompareSchemas_fireNilCheck(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	checkSchemaXML(nil, nil, nil, nil)
	checkSchemaPropertyChanges(nil, nil, nil, nil, nil, nil, false)
	checkExamples(nil, nil, nil)
}

func TestCompareSchemas_TestProps(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      $schema: moo
      exclusiveMaximum: 1
      exclusiveMinimum: 1
      multipleOf: 1
      minimum: 1
      maximum: 1
      maxLength: 1
      minLength: 1
      pattern: a
      format: b
      maxItems: 1
      minItems: 1
      maxProperties: 1
      minProperties: 1
      uniqueItems: true
      contentMediaType: a
      contentEncoding: b
      default: a
      nullable: true
      deprecated: true
      readOnly: true
      writeOnly: true
`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      $schema: cows
      exclusiveMaximum: 2
      exclusiveMinimum: 2
      multipleOf: 2
      minimum: 2
      maximum: 2
      maxLength: 2
      minLength: 2
      pattern: b
      format: a
      maxItems: 2
      minItems: 2
      maxProperties: 2
      minProperties: 2
      uniqueItems: false
      contentMediaType: b
      contentEncoding: a
      default: b
      nullable: false
      deprecated: false
      readOnly: false
      writeOnly: false
`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.GetAllChanges(), 22)
	assert.Equal(t, changes.TotalChanges(), 22)
	assert.Equal(t, changes.TotalBreakingChanges(), 21)
	assert.Len(t, changes.GetPropertyChanges(), 22)
	changes.PropertiesOnly() // this does nothing in this lib.
}

// https://github.com/pb33f/openapi-changes/issues/207
func TestCompareSchemas_DefaultNumericFormattingIsSemanticallyEqual(t *testing.T) {
	low.ClearHashCache()
	left := `openapi: 3.0.0
components:
  schemas:
    Config:
      type: object
      properties:
        angle_threshold:
          type: number
          exclusiveMinimum: 0.0
          title: Angle Threshold
          default: 1e-08
`

	right := `openapi: 3.0.0
components:
  schemas:
    Config:
      type: object
      properties:
        angle_threshold:
          type: number
          exclusiveMinimum: 0.0
          title: Angle Threshold
          default: 1e-8
`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Config").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Config").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.Nil(t, changes)
}

func TestCompareSchemas_DefaultNumericKeyFormattingIsDetected(t *testing.T) {
	low.ClearHashCache()
	left := `openapi: 3.0.0
components:
  schemas:
    Config:
      type: object
      default:
        1: coffee
`

	right := `openapi: 3.0.0
components:
  schemas:
    Config:
      type: object
      default:
        1.0: coffee
`

	leftDoc, rightDoc := test_BuildDoc(left, right)
	lSchemaProxy := leftDoc.Components.Value.FindSchema("Config").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Config").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Equal(t, 1, len(changes.Changes))
	assert.Equal(t, Modified, changes.Changes[0].ChangeType)
	assert.Equal(t, v3.DefaultLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_ExclusiveMaximumNodeSwap(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    TestSchema:
      type: number
      exclusiveMaximum: 100`

	right := `openapi: 3.1
components:
  schemas:
    TestSchema:
      type: number
      exclusiveMaximum: 200`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("TestSchema").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("TestSchema").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, changes.TotalChanges(), 1)
	assert.Equal(t, changes.TotalBreakingChanges(), 1)

	// Find the exclusiveMaximum change
	exclusiveMaxChange := changes.GetPropertyChanges()[0]
	assert.Equal(t, "exclusiveMaximum", exclusiveMaxChange.Property)
	assert.Equal(t, Modified, exclusiveMaxChange.ChangeType)

	// Test the values are correct
	assert.Equal(t, "100", exclusiveMaxChange.Original)
	assert.Equal(t, "200", exclusiveMaxChange.New)
}

func TestCompareSchemas_CheckXML(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1
components:
  schemas:
    OK:
      xml:
        name: ding
        namespace: dong`

	right := `openapi: 3.1
components:
  schemas:
    OK:
      xml:
        name: bing
        namespace: bong`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.GetAllChanges(), 2)
	assert.Equal(t, changes.TotalChanges(), 2)
	assert.Equal(t, changes.TotalBreakingChanges(), 2)
	assert.Len(t, changes.GetPropertyChanges(), 2)
}

// https://github.com/pb33f/openapi-changes/issues/203
func TestCompareSchemas_CheckRogueDescription(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0.3
paths:
  "/test-url":
    get:
      summary: Example endpoint
      responses:
        '200':
          description: Success!
          content:
            application/json:
              schema:
                "$ref": "#/components/schemas/bar"
components:
  schemas:
    bar:
      type: object
      additionalProperties: true
      properties:
        test:
          type: string
          description: Some existing value.
          default: ''
`

	right := `openapi: 3.0.3
paths:
  "/test-url":
    get:
      summary: Example endpoint
      responses:
        '200':
          description: Success!
          content:
            application/json:
              schema:
                "$ref": "#/components/schemas/bar"
components:
  schemas:
    bar:
      type: object
      additionalProperties: true
      properties:
        test:
          type: string
          description: Some existing value.
          default: ''
        foo:
          type: string
          description: Foo.
          default: ''`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("bar").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("bar").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, changes.TotalChanges(), 1)
	assert.Equal(t, changes.TotalBreakingChanges(), 0)
	assert.Len(t, changes.GetPropertyChanges(), 1)
}

func TestCompareSchemas_CheckNPE(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	var sc *SchemaChanges
	assert.Nil(t, sc.GetPropertyChanges())
}

func TestCompareSchemas_CheckRefChangeCircular_Right(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    Yo:
      type: int
    OK:
      $ref: '#/components/schemas/OK'`

	right := `openapi: 3.0
components:
  schemas:
    Yo:
      type: int
    OK:
      $ref: '#/components/schemas/OK'`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.Nil(t, changes)
}

func TestCompareSchemas_CheckRefChangeCircular_Left(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    Yo:
      type: int
    OK:
      $ref: '#/components/schemas/OK'`

	right := `openapi: 3.0
components:
  schemas:
    Yo:
      type: int
    OK:
      $ref: '#/components/schemas/Yo'`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.Nil(t, changes)
}

func TestCompareSchemas_CheckRefChangeCircular_HackIndex(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    Ho:
      type: int
    OK:
      $ref: '#/components/schemas/Ho'`

	right := `openapi: 3.0
components:
  schemas:
    Yo:
      type: int
    OK:
      $ref: '#/components/schemas/Yo'`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	//rSchemaProxy.GetIndex().SetAbsolutePath("/something/else.yaml")

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.Len(t, changes.GetAllChanges(), 1)

	rSchemaProxy.GetIndex().SetAbsolutePath("/something/else.yaml")
	changes = CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.Len(t, changes.GetAllChanges(), 1)

}

func TestCompareSchemas_CheckRefChange_HackIndex_LeftToRight(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    Ho:
      type: int
    OK:
      type: string`

	right := `openapi: 3.0
components:
  schemas:
    Yo:
      type: int
    OK:
      $ref: '#/components/schemas/Yo'`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	//rSchemaProxy.GetIndex().SetAbsolutePath("/something/else.yaml")

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.Len(t, changes.GetAllChanges(), 1)

	rSchemaProxy.GetIndex().SetAbsolutePath("/something/else.yaml")
	changes = CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.Len(t, changes.GetAllChanges(), 1)

	rSchemaProxy.GetIndex().SetAbsolutePath("")
	changes = CompareSchemas(rSchemaProxy, lSchemaProxy) // flip them back
	assert.Len(t, changes.GetAllChanges(), 1)

}

func TestCompareSchemas_CheckRefChangeCircular_HackIndex_LeftToRight(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    Ho:
      type: int
    OK:
      $ref: '#/components/schemas/OK'`

	right := `openapi: 3.0
components:
  schemas:
    Yo:
      type: int
    OK:
      type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	//rSchemaProxy.GetIndex().SetAbsolutePath("/something/else.yaml")

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.Nil(t, changes)
}

func TestCompareSchemas_CheckRefChange_HackIndex_RightToLeft(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    Ho:
      type: int
    OK:
      type: string`

	right := `openapi: 3.0
components:
  schemas:
    Yo:
      type: int
    OK:
      $ref: '#/components/schemas/Yo'`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	//rSchemaProxy.GetIndex().SetAbsolutePath("/something/else.yaml")

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.Len(t, changes.GetAllChanges(), 1)

	rSchemaProxy.GetIndex().SetAbsolutePath("/something/else.yaml")
	changes = CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.Len(t, changes.GetAllChanges(), 1)
}

func TestCompareSchemas_CheckRefChangeCircular_HackIndex_RightToLeft(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    Ho:
      type: int
    OK:
      type: string`

	right := `openapi: 3.0
components:
  schemas:
    Yo:
      type: int
    OK:
      $ref: '#/components/schemas/OK'`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	//rSchemaProxy.GetIndex().SetAbsolutePath("/something/else.yaml")

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.Nil(t, changes)

}

func TestCompareSchemas_DependentRequired_Added(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1.0
components:
  schemas:
    Something:
      type: object
      properties:
        name:
          type: string`

	right := `openapi: 3.1.0
components:
  schemas:
    Something:
      type: object
      dependentRequired:
        billingAddress:
          - street_address
          - locality
        creditCard:
          - billing_address
      properties:
        name:
          type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Something").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Something").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 2, len(changes.DependentRequiredChanges))

	// Check both properties were added
	foundBilling := false
	foundCredit := false
	for _, change := range changes.DependentRequiredChanges {
		if change.Property == "billingAddress" {
			assert.Equal(t, PropertyAdded, change.ChangeType)
			assert.False(t, change.Breaking)
			foundBilling = true
		}
		if change.Property == "creditCard" {
			assert.Equal(t, PropertyAdded, change.ChangeType)
			assert.False(t, change.Breaking)
			foundCredit = true
		}
	}
	assert.True(t, foundBilling)
	assert.True(t, foundCredit)
}

func TestCompareSchemas_DependentRequired_Removed(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1.0
components:
  schemas:
    Something:
      type: object
      dependentRequired:
        billingAddress:
          - street_address
          - locality
        creditCard:
          - billing_address
      properties:
        name:
          type: string`

	right := `openapi: 3.1.0
components:
  schemas:
    Something:
      type: object
      properties:
        name:
          type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Something").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Something").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 2, len(changes.DependentRequiredChanges))

	// Check both properties were removed and marked as breaking
	foundBilling := false
	foundCredit := false
	for _, change := range changes.DependentRequiredChanges {
		if change.Property == "billingAddress" {
			assert.Equal(t, PropertyRemoved, change.ChangeType)
			assert.True(t, change.Breaking) // Removing dependencies is breaking
			foundBilling = true
		}
		if change.Property == "creditCard" {
			assert.Equal(t, PropertyRemoved, change.ChangeType)
			assert.True(t, change.Breaking) // Removing dependencies is breaking
			foundCredit = true
		}
	}
	assert.True(t, foundBilling)
	assert.True(t, foundCredit)
}

func TestCompareSchemas_DependentRequired_Modified(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1.0
components:
  schemas:
    Something:
      type: object
      dependentRequired:
        billingAddress:
          - street_address
          - locality
      properties:
        name:
          type: string`

	right := `openapi: 3.1.0
components:
  schemas:
    Something:
      type: object
      dependentRequired:
        billingAddress:
          - street_address
          - locality
          - region
      properties:
        name:
          type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Something").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Something").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, len(changes.DependentRequiredChanges))

	change := changes.DependentRequiredChanges[0]
	assert.Equal(t, "billingAddress", change.Property)
	assert.Equal(t, Modified, change.ChangeType)
	assert.True(t, change.Breaking) // Adding new dependencies is breaking
}

func TestCompareSchemas_DependentRequired_NoChanges(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1.0
components:
  schemas:
    Something:
      type: object
      dependentRequired:
        billingAddress:
          - street_address
          - locality
      properties:
        name:
          type: string`

	right := `openapi: 3.1.0
components:
  schemas:
    Something:
      type: object
      dependentRequired:
        billingAddress:
          - street_address
          - locality
      properties:
        name:
          type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Something").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Something").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.Nil(t, changes)
}

func TestCompareSchemas_DependentRequired_EmptyArray(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1.0
components:
  schemas:
    Something:
      type: object
      dependentRequired:
        billingAddress: []
      properties:
        name:
          type: string`

	right := `openapi: 3.1.0
components:
  schemas:
    Something:
      type: object
      dependentRequired:
        billingAddress:
          - street_address
      properties:
        name:
          type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Something").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Something").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, len(changes.DependentRequiredChanges))

	change := changes.DependentRequiredChanges[0]
	assert.Equal(t, "billingAddress", change.Property)
	assert.Equal(t, Modified, change.ChangeType)
	assert.True(t, change.Breaking) // Adding dependencies is breaking
}

func TestCompareSchemas_DependentRequired_OrderMatters(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1.0
components:
  schemas:
    Something:
      type: object
      dependentRequired:
        billingAddress:
          - street_address
          - locality
      properties:
        name:
          type: string`

	right := `openapi: 3.1.0
components:
  schemas:
    Something:
      type: object
      dependentRequired:
        billingAddress:
          - locality
          - street_address
      properties:
        name:
          type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Something").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Something").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, len(changes.DependentRequiredChanges))

	change := changes.DependentRequiredChanges[0]
	assert.Equal(t, "billingAddress", change.Property)
	assert.Equal(t, Modified, change.ChangeType)
	assert.True(t, change.Breaking) // Order change is breaking as it changes validation behavior
}

func TestCompareSchemas_DependentRequired_TotalChangesCount(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1.0
components:
  schemas:
    Something:
      type: object
      properties:
        name:
          type: string`

	right := `openapi: 3.1.0
components:
  schemas:
    Something:
      type: object
      dependentRequired:
        billingAddress:
          - street_address
        creditCard:
          - billing_address
      properties:
        name:
          type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Something").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Something").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)

	// Test TotalChanges includes DependentRequired changes
	totalChanges := changes.TotalChanges()
	assert.Equal(t, 2, totalChanges) // 2 DependentRequired additions

	// Test TotalBreakingChanges doesn't include non-breaking additions
	totalBreaking := changes.TotalBreakingChanges()
	assert.Equal(t, 0, totalBreaking) // Adding dependencies is non-breaking
}

func TestCompareSchemas_DependentRequired_TotalBreakingChangesCount(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1.0
components:
  schemas:
    Something:
      type: object
      dependentRequired:
        billingAddress:
          - street_address
        creditCard:
          - billing_address
      properties:
        name:
          type: string`

	right := `openapi: 3.1.0
components:
  schemas:
    Something:
      type: object
      properties:
        name:
          type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Something").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Something").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)

	// Test TotalBreakingChanges includes DependentRequired removals
	totalBreaking := changes.TotalBreakingChanges()
	assert.Equal(t, 2, totalBreaking) // 2 DependentRequired removals
}

func TestSchemaChanges_GetAllChanges_WithDependentRequired(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1.0
components:
  schemas:
    Something:
      type: object
      properties:
        name:
          type: string
        billing:
          type: object`

	right := `openapi: 3.1.0
components:
  schemas:
    Something:
      type: object
      properties:
        name:
          type: string
        billing:
          type: object
      dependentRequired:
        billing: ["name"]`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Something").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Something").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)

	// Test GetAllChanges includes DependentRequired changes
	allChanges := changes.GetAllChanges()
	assert.Greater(t, len(allChanges), 0)

	// Verify at least one DependentRequired change is included
	foundDepReq := false
	for _, change := range allChanges {
		if change.Property == "billing" {
			foundDepReq = true
			break
		}
	}
	assert.True(t, foundDepReq)
}

func TestSchemaChanges_TotalBreakingChanges_WithDependentRequired(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.1.0
components:
  schemas:
    Something:
      type: object
      properties:
        name:
          type: string
        billing:
          type: object
      dependentRequired:
        billing: ["name", "email"]`

	right := `openapi: 3.1.0
components:
  schemas:
    Something:
      type: object
      properties:
        name:
          type: string
        billing:
          type: object`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Something").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Something").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)

	// Test TotalBreakingChanges includes DependentRequired breaking changes
	totalBreaking := changes.TotalBreakingChanges()
	assert.Greater(t, totalBreaking, 0)
}

func TestSlicesEqual_AllCases(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	// Test equal slices (this covers the return true case - line 1772)
	a := []string{"name", "email"}
	b := []string{"name", "email"}
	assert.True(t, slicesEqual(a, b))

	// Test different lengths
	c := []string{"name"}
	assert.False(t, slicesEqual(a, c))

	// Test different content
	d := []string{"name", "phone"}
	assert.False(t, slicesEqual(a, d))

	// Test empty slices
	assert.True(t, slicesEqual([]string{}, []string{}))
}

func TestGetNodeForProperty_EdgeCases(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	// Test with nil map (line 1778-1779)
	node := getNodeForProperty(nil, "test")
	assert.Nil(t, node)

	// Test with property not found (line 1785)
	depMap := orderedmap.New[low.KeyReference[string], low.ValueReference[[]string]]()
	depMap.Set(low.KeyReference[string]{Value: "billing"}, low.ValueReference[[]string]{Value: []string{"name"}})

	node = getNodeForProperty(depMap, "nonexistent")
	assert.Nil(t, node)

	// Test with property found (should return the node)
	node = getNodeForProperty(depMap, "billing")
	// Note: In this test case the node will be nil since we didn't set ValueNode,
	// but the function should still return it without error
	assert.Nil(t, node) // This is expected since we didn't populate ValueNode in our test map
}

func TestGetNodeForProperty_WithActualDocument(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	// Create a test with actual YAML nodes to hit the success path
	spec := `openapi: 3.1.0
components:
  schemas:
    Something:
      type: object
      dependentRequired:
        billing: ["name", "email"]`

	leftDoc, _ := test_BuildDoc(spec, spec)
	lSchemaProxy := leftDoc.Components.Value.FindSchema("Something").Value

	// Access the low-level DependentRequired to test with real nodes
	lowSchema := lSchemaProxy.Schema()
	if lowSchema.DependentRequired.Value != nil {
		// This should hit the successful path in getNodeForProperty
		node := getNodeForProperty(lowSchema.DependentRequired.Value, "billing")
		// The node should exist since we have real YAML nodes
		assert.NotNil(t, node)
	}
}

func TestSchemaChanges_GetPropertyChanges_WithDependentRequired(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	// This test specifically targets lines 73-74 in GetPropertyChanges() method
	left := `openapi: 3.1.0
components:
  schemas:
    Something:
      type: object
      properties:
        name:
          type: string
        billing:
          type: object`

	right := `openapi: 3.1.0
components:
  schemas:
    Something:
      type: object
      properties:
        name:
          type: string
        billing:
          type: object
      dependentRequired:
        billing: ["name"]`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Something").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Something").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Greater(t, len(changes.DependentRequiredChanges), 0)

	// This specifically calls GetPropertyChanges() which contains lines 73-74
	propertyChanges := changes.GetPropertyChanges()
	assert.Greater(t, len(propertyChanges), 0)

	// Verify that DependentRequired changes are included in property changes
	foundDepReq := false
	for _, change := range propertyChanges {
		if change.Property == "billing" {
			foundDepReq = true
			break
		}
	}
	assert.True(t, foundDepReq)
}

// TestCompareSchemas_TypeChange_ContextLines tests that type changes have proper context
func TestCompareSchemas_TypeChange_ContextLines(t *testing.T) {
	// Clear hash cache to ensure deterministic results
	low.ClearHashCache()
	left := `openapi: 3.0
components:
  schemas:
    Something:
      type: string`

	right := `openapi: 3.0
components:
  schemas:
    Something:
      type: integer`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Something").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Something").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())

	// Verify the type change has context lines set
	typeChange := changes.Changes[0]
	assert.Equal(t, "type", typeChange.Property)
	assert.NotNil(t, typeChange.Context)
	// Context lines should be set from the schema KeyNode
	assert.NotNil(t, typeChange.Context.OriginalLine)
	assert.NotNil(t, typeChange.Context.NewLine)
}

// TestCompareSchemas_DynamicAnchor_Added tests adding $dynamicAnchor (JSON Schema 2020-12)
func TestCompareSchemas_DynamicAnchor_Added(t *testing.T) {
	low.ClearHashCache()

	left := `openapi: "3.1.0"
components:
  schemas:
    TreeNode:
      type: object`

	right := `openapi: "3.1.0"
components:
  schemas:
    TreeNode:
      type: object
      $dynamicAnchor: node`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("TreeNode").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("TreeNode").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Equal(t, 0, changes.TotalBreakingChanges(), "Adding $dynamicAnchor should not be breaking by default")
	assert.Equal(t, PropertyAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, "$dynamicAnchor", changes.Changes[0].Property)
}

// TestCompareSchemas_DynamicAnchor_Modified tests modifying $dynamicAnchor
func TestCompareSchemas_DynamicAnchor_Modified(t *testing.T) {
	low.ClearHashCache()

	left := `openapi: "3.1.0"
components:
  schemas:
    TreeNode:
      type: object
      $dynamicAnchor: nodeOld`

	right := `openapi: "3.1.0"
components:
  schemas:
    TreeNode:
      type: object
      $dynamicAnchor: nodeNew`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("TreeNode").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("TreeNode").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Equal(t, 1, changes.TotalBreakingChanges(), "Modifying $dynamicAnchor should be breaking by default")
	assert.Equal(t, Modified, changes.Changes[0].ChangeType)
	assert.Equal(t, "$dynamicAnchor", changes.Changes[0].Property)
}

// TestCompareSchemas_DynamicAnchor_Removed tests removing $dynamicAnchor
func TestCompareSchemas_DynamicAnchor_Removed(t *testing.T) {
	low.ClearHashCache()

	left := `openapi: "3.1.0"
components:
  schemas:
    TreeNode:
      type: object
      $dynamicAnchor: node`

	right := `openapi: "3.1.0"
components:
  schemas:
    TreeNode:
      type: object`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("TreeNode").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("TreeNode").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Equal(t, 1, changes.TotalBreakingChanges(), "Removing $dynamicAnchor should be breaking by default")
	assert.Equal(t, PropertyRemoved, changes.Changes[0].ChangeType)
	assert.Equal(t, "$dynamicAnchor", changes.Changes[0].Property)
}

// TestCompareSchemas_DynamicRef_Added tests adding $dynamicRef (JSON Schema 2020-12)
func TestCompareSchemas_DynamicRef_Added(t *testing.T) {
	low.ClearHashCache()

	left := `openapi: "3.1.0"
components:
  schemas:
    TreeNode:
      type: object`

	right := `openapi: "3.1.0"
components:
  schemas:
    TreeNode:
      type: object
      $dynamicRef: "#node"`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("TreeNode").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("TreeNode").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Equal(t, 0, changes.TotalBreakingChanges(), "Adding $dynamicRef should not be breaking by default")
	assert.Equal(t, PropertyAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, "$dynamicRef", changes.Changes[0].Property)
}

// TestCompareSchemas_DynamicRef_Modified tests modifying $dynamicRef
func TestCompareSchemas_DynamicRef_Modified(t *testing.T) {
	low.ClearHashCache()

	left := `openapi: "3.1.0"
components:
  schemas:
    TreeNode:
      type: object
      $dynamicRef: "#oldNode"`

	right := `openapi: "3.1.0"
components:
  schemas:
    TreeNode:
      type: object
      $dynamicRef: "#newNode"`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("TreeNode").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("TreeNode").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Equal(t, 1, changes.TotalBreakingChanges(), "Modifying $dynamicRef should be breaking by default")
	assert.Equal(t, Modified, changes.Changes[0].ChangeType)
	assert.Equal(t, "$dynamicRef", changes.Changes[0].Property)
}

// TestCompareSchemas_DynamicRef_Removed tests removing $dynamicRef
func TestCompareSchemas_DynamicRef_Removed(t *testing.T) {
	low.ClearHashCache()

	left := `openapi: "3.1.0"
components:
  schemas:
    TreeNode:
      type: object
      $dynamicRef: "#node"`

	right := `openapi: "3.1.0"
components:
  schemas:
    TreeNode:
      type: object`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("TreeNode").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("TreeNode").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Equal(t, 1, changes.TotalBreakingChanges(), "Removing $dynamicRef should be breaking by default")
	assert.Equal(t, PropertyRemoved, changes.Changes[0].ChangeType)
	assert.Equal(t, "$dynamicRef", changes.Changes[0].Property)
}

// TestCompareSchemas_DynamicAnchor_ConfigurableBreakingRules tests that $dynamicAnchor
// breaking behavior can be configured
func TestCompareSchemas_DynamicAnchor_ConfigurableBreakingRules(t *testing.T) {
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	left := `openapi: "3.1.0"
components:
  schemas:
    TreeNode:
      type: object
      $dynamicAnchor: nodeOld`

	right := `openapi: "3.1.0"
components:
  schemas:
    TreeNode:
      type: object
      $dynamicAnchor: nodeNew`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("TreeNode").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("TreeNode").Value

	// Default behavior: modification should be breaking
	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalBreakingChanges(), "Modifying $dynamicAnchor should be breaking by default")

	// Now configure $dynamicAnchor modification as non-breaking
	customConfig := &BreakingRulesConfig{
		Schema: &SchemaRules{
			DynamicAnchor: &BreakingChangeRule{
				Added:    boolPtr(false),
				Modified: boolPtr(false), // Override: modification is not breaking
				Removed:  boolPtr(false),
			},
		},
	}
	SetActiveBreakingRulesConfig(customConfig)

	// Re-run comparison with custom config
	changes2 := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes2)
	assert.Equal(t, 0, changes2.TotalBreakingChanges(), "With custom config, modifying $dynamicAnchor should not be breaking")
}

// TestCompareSchemas_DynamicRef_ConfigurableBreakingRules tests that $dynamicRef
// breaking behavior can be configured
func TestCompareSchemas_DynamicRef_ConfigurableBreakingRules(t *testing.T) {
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	left := `openapi: "3.1.0"
components:
  schemas:
    TreeNode:
      type: object
      $dynamicRef: "#oldNode"`

	right := `openapi: "3.1.0"
components:
  schemas:
    TreeNode:
      type: object
      $dynamicRef: "#newNode"`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("TreeNode").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("TreeNode").Value

	// Default behavior: modification should be breaking
	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalBreakingChanges(), "Modifying $dynamicRef should be breaking by default")

	// Now configure $dynamicRef modification as non-breaking
	customConfig := &BreakingRulesConfig{
		Schema: &SchemaRules{
			DynamicRef: &BreakingChangeRule{
				Added:    boolPtr(false),
				Modified: boolPtr(false), // Override: modification is not breaking
				Removed:  boolPtr(false),
			},
		},
	}
	SetActiveBreakingRulesConfig(customConfig)

	// Re-run comparison with custom config
	changes2 := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes2)
	assert.Equal(t, 0, changes2.TotalBreakingChanges(), "With custom config, modifying $dynamicRef should not be breaking")
}

// TestCompareSchemas_Id_Added tests detection of $id being added
func TestCompareSchemas_Id_Added(t *testing.T) {
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	left := `openapi: "3.1.0"
info:
  title: left
  version: "1.0"
components:
  schemas:
    Pet:
      type: object`

	right := `openapi: "3.1.0"
info:
  title: right
  version: "1.0"
components:
  schemas:
    Pet:
      $id: "https://example.com/schemas/pet.json"
      type: object`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Pet").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Pet").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())

	// Find the $id change
	found := false
	for _, change := range changes.Changes {
		if change.Property == PropId {
			found = true
			assert.Equal(t, PropertyAdded, change.ChangeType)
			assert.Equal(t, "https://example.com/schemas/pet.json", change.New)
			break
		}
	}
	assert.True(t, found, "Should find $id property change")
}

// TestCompareSchemas_Id_Removed tests detection of $id being removed
func TestCompareSchemas_Id_Removed(t *testing.T) {
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	left := `openapi: "3.1.0"
info:
  title: left
  version: "1.0"
components:
  schemas:
    Pet:
      $id: "https://example.com/schemas/pet.json"
      type: object`

	right := `openapi: "3.1.0"
info:
  title: right
  version: "1.0"
components:
  schemas:
    Pet:
      type: object`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Pet").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Pet").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())

	// Find the $id change
	found := false
	for _, change := range changes.Changes {
		if change.Property == PropId {
			found = true
			assert.Equal(t, PropertyRemoved, change.ChangeType)
			assert.Equal(t, "https://example.com/schemas/pet.json", change.Original)
			break
		}
	}
	assert.True(t, found, "Should find $id property change")
}

// TestCompareSchemas_Id_Modified tests detection of $id being modified
func TestCompareSchemas_Id_Modified(t *testing.T) {
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	left := `openapi: "3.1.0"
info:
  title: left
  version: "1.0"
components:
  schemas:
    Pet:
      $id: "https://example.com/schemas/pet-v1.json"
      type: object`

	right := `openapi: "3.1.0"
info:
  title: right
  version: "1.0"
components:
  schemas:
    Pet:
      $id: "https://example.com/schemas/pet-v2.json"
      type: object`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Pet").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Pet").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())

	// Find the $id change
	found := false
	for _, change := range changes.Changes {
		if change.Property == PropId {
			found = true
			assert.Equal(t, Modified, change.ChangeType)
			assert.Equal(t, "https://example.com/schemas/pet-v1.json", change.Original)
			assert.Equal(t, "https://example.com/schemas/pet-v2.json", change.New)
			break
		}
	}
	assert.True(t, found, "Should find $id property change")
}

// TestCompareSchemas_Id_NoChange tests that identical $id produces no changes
func TestCompareSchemas_Id_NoChange(t *testing.T) {
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	left := `openapi: "3.1.0"
info:
  title: left
  version: "1.0"
components:
  schemas:
    Pet:
      $id: "https://example.com/schemas/pet.json"
      type: object`

	right := `openapi: "3.1.0"
info:
  title: right
  version: "1.0"
components:
  schemas:
    Pet:
      $id: "https://example.com/schemas/pet.json"
      type: object`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Pet").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Pet").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.Nil(t, changes)
}

// TestCompareSchemas_Comment_Added tests $comment addition detection
func TestCompareSchemas_Comment_Added(t *testing.T) {
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	left := `openapi: "3.1.0"
info:
  title: left
  version: "1.0"
components:
  schemas:
    Pet:
      type: object`

	right := `openapi: "3.1.0"
info:
  title: right
  version: "1.0"
components:
  schemas:
    Pet:
      $comment: "This is a comment"
      type: object`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Pet").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Pet").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())

	found := false
	for _, change := range changes.Changes {
		if change.Property == PropComment {
			found = true
			assert.Equal(t, PropertyAdded, change.ChangeType)
			assert.Equal(t, "This is a comment", change.New)
			assert.False(t, change.Breaking)
			break
		}
	}
	assert.True(t, found, "Should find $comment property change")
}

// TestCompareSchemas_Comment_Removed tests $comment removal detection
func TestCompareSchemas_Comment_Removed(t *testing.T) {
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	left := `openapi: "3.1.0"
info:
  title: left
  version: "1.0"
components:
  schemas:
    Pet:
      $comment: "This is a comment"
      type: object`

	right := `openapi: "3.1.0"
info:
  title: right
  version: "1.0"
components:
  schemas:
    Pet:
      type: object`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Pet").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Pet").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())

	found := false
	for _, change := range changes.Changes {
		if change.Property == PropComment {
			found = true
			assert.Equal(t, PropertyRemoved, change.ChangeType)
			assert.Equal(t, "This is a comment", change.Original)
			assert.False(t, change.Breaking)
			break
		}
	}
	assert.True(t, found, "Should find $comment property change")
}

// TestCompareSchemas_Comment_Modified tests $comment modification detection
func TestCompareSchemas_Comment_Modified(t *testing.T) {
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	left := `openapi: "3.1.0"
info:
  title: left
  version: "1.0"
components:
  schemas:
    Pet:
      $comment: "Original comment"
      type: object`

	right := `openapi: "3.1.0"
info:
  title: right
  version: "1.0"
components:
  schemas:
    Pet:
      $comment: "Modified comment"
      type: object`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Pet").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Pet").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())

	found := false
	for _, change := range changes.Changes {
		if change.Property == PropComment {
			found = true
			assert.Equal(t, Modified, change.ChangeType)
			assert.Equal(t, "Original comment", change.Original)
			assert.Equal(t, "Modified comment", change.New)
			assert.False(t, change.Breaking)
			break
		}
	}
	assert.True(t, found, "Should find $comment property change")
}

// TestCompareSchemas_Comment_NoChange tests identical $comment produces no changes
func TestCompareSchemas_Comment_NoChange(t *testing.T) {
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	left := `openapi: "3.1.0"
info:
  title: left
  version: "1.0"
components:
  schemas:
    Pet:
      $comment: "Same comment"
      type: object`

	right := `openapi: "3.1.0"
info:
  title: right
  version: "1.0"
components:
  schemas:
    Pet:
      $comment: "Same comment"
      type: object`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Pet").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Pet").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.Nil(t, changes)
}

// TestCompareSchemas_ContentSchema_Added tests contentSchema addition detection
func TestCompareSchemas_ContentSchema_Added(t *testing.T) {
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	left := `openapi: "3.1.0"
info:
  title: left
  version: "1.0"
components:
  schemas:
    Pet:
      type: string
      contentMediaType: application/json`

	right := `openapi: "3.1.0"
info:
  title: right
  version: "1.0"
components:
  schemas:
    Pet:
      type: string
      contentMediaType: application/json
      contentSchema:
        type: object`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Pet").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Pet").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())

	found := false
	for _, change := range changes.Changes {
		if change.Property == PropContentSchema {
			found = true
			assert.Equal(t, PropertyAdded, change.ChangeType)
			assert.True(t, change.Breaking)
			break
		}
	}
	assert.True(t, found, "Should find contentSchema property change")
}

// TestCompareSchemas_ContentSchema_Removed tests contentSchema removal detection
func TestCompareSchemas_ContentSchema_Removed(t *testing.T) {
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	left := `openapi: "3.1.0"
info:
  title: left
  version: "1.0"
components:
  schemas:
    Pet:
      type: string
      contentMediaType: application/json
      contentSchema:
        type: object`

	right := `openapi: "3.1.0"
info:
  title: right
  version: "1.0"
components:
  schemas:
    Pet:
      type: string
      contentMediaType: application/json`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Pet").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Pet").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())

	found := false
	for _, change := range changes.Changes {
		if change.Property == PropContentSchema {
			found = true
			assert.Equal(t, PropertyRemoved, change.ChangeType)
			assert.True(t, change.Breaking)
			break
		}
	}
	assert.True(t, found, "Should find contentSchema property change")
}

// TestCompareSchemas_ContentSchema_Modified tests contentSchema modification detection
func TestCompareSchemas_ContentSchema_Modified(t *testing.T) {
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	left := `openapi: "3.1.0"
info:
  title: left
  version: "1.0"
components:
  schemas:
    Pet:
      type: string
      contentMediaType: application/json
      contentSchema:
        type: object`

	right := `openapi: "3.1.0"
info:
  title: right
  version: "1.0"
components:
  schemas:
    Pet:
      type: string
      contentMediaType: application/json
      contentSchema:
        type: array`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Pet").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Pet").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.NotNil(t, changes.ContentSchemaChanges)
	assert.Equal(t, 1, changes.ContentSchemaChanges.TotalChanges())
}

// TestCompareSchemas_Vocabulary_Added tests $vocabulary entry addition detection
func TestCompareSchemas_Vocabulary_Added(t *testing.T) {
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	left := `openapi: "3.1.0"
info:
  title: left
  version: "1.0"
components:
  schemas:
    Pet:
      $vocabulary:
        "https://json-schema.org/draft/2020-12/vocab/core": true
      type: object`

	right := `openapi: "3.1.0"
info:
  title: right
  version: "1.0"
components:
  schemas:
    Pet:
      $vocabulary:
        "https://json-schema.org/draft/2020-12/vocab/core": true
        "https://json-schema.org/draft/2020-12/vocab/validation": true
      type: object`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Pet").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Pet").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.VocabularyChanges, 1)
	assert.Equal(t, PropertyAdded, changes.VocabularyChanges[0].ChangeType)
	assert.True(t, changes.VocabularyChanges[0].Breaking)
}

// TestCompareSchemas_Vocabulary_Removed tests $vocabulary entry removal detection
func TestCompareSchemas_Vocabulary_Removed(t *testing.T) {
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	left := `openapi: "3.1.0"
info:
  title: left
  version: "1.0"
components:
  schemas:
    Pet:
      $vocabulary:
        "https://json-schema.org/draft/2020-12/vocab/core": true
        "https://json-schema.org/draft/2020-12/vocab/validation": true
      type: object`

	right := `openapi: "3.1.0"
info:
  title: right
  version: "1.0"
components:
  schemas:
    Pet:
      $vocabulary:
        "https://json-schema.org/draft/2020-12/vocab/core": true
      type: object`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Pet").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Pet").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.VocabularyChanges, 1)
	assert.Equal(t, PropertyRemoved, changes.VocabularyChanges[0].ChangeType)
	assert.True(t, changes.VocabularyChanges[0].Breaking)
}

// TestCompareSchemas_Vocabulary_Modified tests $vocabulary value modification detection
func TestCompareSchemas_Vocabulary_Modified(t *testing.T) {
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	left := `openapi: "3.1.0"
info:
  title: left
  version: "1.0"
components:
  schemas:
    Pet:
      $vocabulary:
        "https://json-schema.org/draft/2020-12/vocab/core": true
      type: object`

	right := `openapi: "3.1.0"
info:
  title: right
  version: "1.0"
components:
  schemas:
    Pet:
      $vocabulary:
        "https://json-schema.org/draft/2020-12/vocab/core": false
      type: object`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Pet").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Pet").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.VocabularyChanges, 1)
	assert.Equal(t, Modified, changes.VocabularyChanges[0].ChangeType)
	assert.True(t, changes.VocabularyChanges[0].Breaking)
}

// TestCompareSchemas_Vocabulary_NoChange tests identical $vocabulary produces no changes
func TestCompareSchemas_Vocabulary_NoChange(t *testing.T) {
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	left := `openapi: "3.1.0"
info:
  title: left
  version: "1.0"
components:
  schemas:
    Pet:
      $vocabulary:
        "https://json-schema.org/draft/2020-12/vocab/core": true
      type: object`

	right := `openapi: "3.1.0"
info:
  title: right
  version: "1.0"
components:
  schemas:
    Pet:
      $vocabulary:
        "https://json-schema.org/draft/2020-12/vocab/core": true
      type: object`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Pet").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Pet").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.Nil(t, changes)
}

// TestCompareSchemas_Vocabulary_AddedFromNil tests $vocabulary added where none existed
func TestCompareSchemas_Vocabulary_AddedFromNil(t *testing.T) {
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	left := `openapi: "3.1.0"
info:
  title: left
  version: "1.0"
components:
  schemas:
    Pet:
      type: object`

	right := `openapi: "3.1.0"
info:
  title: right
  version: "1.0"
components:
  schemas:
    Pet:
      $vocabulary:
        "https://json-schema.org/draft/2020-12/vocab/core": true
      type: object`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Pet").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Pet").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.VocabularyChanges, 1)
	assert.Equal(t, PropertyAdded, changes.VocabularyChanges[0].ChangeType)
}

// TestCompareSchemas_Vocabulary_RemovedToNil tests $vocabulary removed to nil
func TestCompareSchemas_Vocabulary_RemovedToNil(t *testing.T) {
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	left := `openapi: "3.1.0"
info:
  title: left
  version: "1.0"
components:
  schemas:
    Pet:
      $vocabulary:
        "https://json-schema.org/draft/2020-12/vocab/core": true
      type: object`

	right := `openapi: "3.1.0"
info:
  title: right
  version: "1.0"
components:
  schemas:
    Pet:
      type: object`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Pet").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Pet").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.VocabularyChanges, 1)
	assert.Equal(t, PropertyRemoved, changes.VocabularyChanges[0].ChangeType)
}

// TestCheckVocabularyChanges_BothNil tests the checkVocabularyChanges helper with both nil
func TestCheckVocabularyChanges_BothNil(t *testing.T) {
	changes := checkVocabularyChanges(nil, nil)
	assert.Nil(t, changes)
}

// TestCompareSchemas_Vocabulary_MultipleChanges tests multiple vocabulary changes at once
func TestCompareSchemas_Vocabulary_MultipleChanges(t *testing.T) {
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	left := `openapi: "3.1.0"
info:
  title: left
  version: "1.0"
components:
  schemas:
    Pet:
      $vocabulary:
        "https://json-schema.org/draft/2020-12/vocab/core": true
        "https://json-schema.org/draft/2020-12/vocab/validation": true
      type: object`

	right := `openapi: "3.1.0"
info:
  title: right
  version: "1.0"
components:
  schemas:
    Pet:
      $vocabulary:
        "https://json-schema.org/draft/2020-12/vocab/core": false
        "https://json-schema.org/draft/2020-12/vocab/applicator": true
      type: object`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Pet").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Pet").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	// Should have 3 changes: core modified, validation removed, applicator added
	assert.Equal(t, 3, changes.TotalChanges())
	assert.Len(t, changes.VocabularyChanges, 3)
}

// TestSchemaChanges_TotalBreakingChanges_ContentSchema tests that TotalBreakingChanges
// correctly counts breaking changes from ContentSchemaChanges
func TestSchemaChanges_TotalBreakingChanges_ContentSchema(t *testing.T) {
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	left := `openapi: "3.1.0"
info:
  title: left
  version: "1.0"
components:
  schemas:
    EncodedData:
      type: string
      contentMediaType: application/json
      contentSchema:
        type: object
        properties:
          name:
            type: string`

	right := `openapi: "3.1.0"
info:
  title: right
  version: "1.0"
components:
  schemas:
    EncodedData:
      type: string
      contentMediaType: application/json
      contentSchema:
        type: object
        properties:
          name:
            type: integer`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("EncodedData").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("EncodedData").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.NotNil(t, changes.ContentSchemaChanges)
	// ContentSchemaChanges should have changes and TotalBreakingChanges should count them
	assert.GreaterOrEqual(t, changes.ContentSchemaChanges.TotalChanges(), 1)
	// TotalBreakingChanges on parent should include ContentSchemaChanges breaking changes
	assert.GreaterOrEqual(t, changes.TotalBreakingChanges(), 1)
}

// TestSchemaChanges_TotalBreakingChanges_Vocabulary tests that TotalBreakingChanges
// correctly counts breaking changes from VocabularyChanges
func TestSchemaChanges_TotalBreakingChanges_Vocabulary(t *testing.T) {
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	left := `openapi: "3.1.0"
info:
  title: left
  version: "1.0"
components:
  schemas:
    MetaSchema:
      type: object`

	right := `openapi: "3.1.0"
info:
  title: right
  version: "1.0"
components:
  schemas:
    MetaSchema:
      $vocabulary:
        "https://json-schema.org/draft/2020-12/vocab/core": true
      type: object`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("MetaSchema").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("MetaSchema").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.VocabularyChanges, 1)
	// Vocabulary addition is breaking by default
	assert.True(t, changes.VocabularyChanges[0].Breaking)
	// TotalBreakingChanges should count the vocabulary change
	assert.Equal(t, 1, changes.TotalBreakingChanges())
}
