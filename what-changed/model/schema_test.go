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
	assert.Equal(t, "0e563831440581c713657dd857a0ec3af1bd7308a43bd3cae9184f61d61b288f",
		low.HashToString(changes.Changes[0].NewObject.(*base.Discriminator).Hash()))
}

func TestCompareSchemas_DiscriminatorRemove(t *testing.T) {
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
	assert.Equal(t, "0e563831440581c713657dd857a0ec3af1bd7308a43bd3cae9184f61d61b288f",
		low.HashToString(changes.Changes[0].OriginalObject.(*base.Discriminator).Hash()))
}

func TestCompareSchemas_ExternalDocsChange(t *testing.T) {
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
	assert.Equal(t, "2b7adf30f2ea3a7617ccf429a099617a9c03e8b5f3a23a89dba4b90f760010d7",
		low.HashToString(changes.Changes[0].NewObject.(*base.ExternalDoc).Hash()))
}

func TestCompareSchemas_ExternalDocsRemove(t *testing.T) {
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
	assert.Equal(t, "2b7adf30f2ea3a7617ccf429a099617a9c03e8b5f3a23a89dba4b90f760010d7",
		low.HashToString(changes.Changes[0].OriginalObject.(*base.ExternalDoc).Hash()))
}

func TestCompareSchemas_AddExtension(t *testing.T) {
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
	assert.Equal(t, 3, changes.TotalBreakingChanges())
	assert.Len(t, changes.SchemaPropertyChanges["name"].PropertyChanges.Changes, 2)
	assert.Len(t, changes.PropertyChanges.Changes, 2)
}

func TestSchemaChanges_TotalChanges_NoNilPanic(t *testing.T) {
	var changes *SchemaChanges
	assert.Equal(t, 0, changes.TotalChanges())
}

func TestSchemaChanges_TotalBreakingChanges_NoNilPanic(t *testing.T) {
	var changes *SchemaChanges
	assert.Equal(t, 0, changes.TotalBreakingChanges())
}

func TestCompareSchemas_Nil(t *testing.T) {
	assert.Nil(t, CompareSchemas(nil, nil))
}

// Test for issue https://github.com/pb33f/libopenapi/issues/218
func TestCompareSchemas_PropertyRefChange_Identical(t *testing.T) {
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

func TestCompareSchemas_fireNilCheck(t *testing.T) {
	checkSchemaXML(nil, nil, nil, nil)
	checkSchemaPropertyChanges(nil, nil, nil, nil)
	checkExamples(nil, nil, nil)
}

func TestCompareSchemas_TestProps(t *testing.T) {
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
