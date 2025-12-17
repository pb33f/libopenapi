// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"fmt"
	"testing"

	"github.com/pb33f/libopenapi/utils"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	v2 "github.com/pb33f/libopenapi/datamodel/low/v2"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
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
        - type: bool`

	right := `openapi: 3.0
components:
  schemas:
    OK:
      type: string
      items:
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
	assert.Equal(t, "d998db65844824d9fe1c4b3fe13d9d969697a3f5353611dc7f2a6a158da77de1",
		low.HashToString(changes.Changes[0].NewObject.(*base.Discriminator).Hash()))
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
	assert.Equal(t, "d998db65844824d9fe1c4b3fe13d9d969697a3f5353611dc7f2a6a158da77de1",
		low.HashToString(changes.Changes[0].OriginalObject.(*base.Discriminator).Hash()))
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
	assert.Equal(t, "cc072505e1639fd745ccaf6d2d4188db0c0475d4e9a48a6b4d1b33a77183a882",
		low.HashToString(changes.Changes[0].NewObject.(*base.ExternalDoc).Hash()))
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
	assert.Equal(t, "cc072505e1639fd745ccaf6d2d4188db0c0475d4e9a48a6b4d1b33a77183a882",
		low.HashToString(changes.Changes[0].OriginalObject.(*base.ExternalDoc).Hash()))
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
	checkSchemaPropertyChanges(nil, nil, nil, nil, nil, nil)
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
