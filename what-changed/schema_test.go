// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

import (
	"fmt"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	v2 "github.com/pb33f/libopenapi/datamodel/low/v2"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/stretchr/testify/assert"
	"testing"
)

// These tests require full documents to be tested properly. schemas are perhaps the most complex
// of all the things in OpenAPI, to ensure correctness, we must test the whole document structure.
//
// To test this correctly, we need a simulated document with inline schemas for recursive
// checking, as well as a couple of references, so we can avoid that disaster.
// in our model, components/definitions will be checked independently for changes
// and references will be checked only for value changes (points to a different reference)
func TestCompareSchemas_PropertyModification(t *testing.T) {
	left := `components:
  schemas:
    OK:
      title: an OK message`

	right := `components:
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
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Equal(t, Modified, changes.Changes[0].ChangeType)
	assert.Equal(t, "an OK message Changed", changes.Changes[0].New)
}

func TestCompareSchemas_PropertyAdd(t *testing.T) {
	left := `components:
  schemas:
    OK:
      title: an OK message`

	right := `components:
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
	assert.Equal(t, PropertyAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, "a thing", changes.Changes[0].New)
	assert.Equal(t, v3.DescriptionLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_PropertyRemove(t *testing.T) {
	left := `components:
  schemas:
    OK:
      title: an OK message
      description: a thing`

	right := `components:
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
	assert.Equal(t, PropertyRemoved, changes.Changes[0].ChangeType)
	assert.Equal(t, "a thing", changes.Changes[0].Original)
	assert.Equal(t, v3.DescriptionLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_Removed(t *testing.T) {
	left := `components:
  schemas:
    OK:
      title: an OK message
      description: a thing`

	right := `components:
  schemas:`

	leftDoc, _ := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	var rSchemaProxy *base.SchemaProxy

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.Changes, 1)
	assert.Equal(t, ObjectRemoved, changes.Changes[0].ChangeType)
}

func TestCompareSchemas_Added(t *testing.T) {
	left := `components:
  schemas:
    OK:
      title: an OK message
      description: a thing`

	right := `components:
  schemas:`

	leftDoc, _ := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	var rSchemaProxy *base.SchemaProxy

	changes := CompareSchemas(rSchemaProxy, lSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.Changes, 1)
	assert.Equal(t, ObjectAdded, changes.Changes[0].ChangeType)
}

func test_BuildDoc(l, r string) (*v3.Document, *v3.Document) {

	leftInfo, _ := datamodel.ExtractSpecInfo([]byte(l))
	rightInfo, _ := datamodel.ExtractSpecInfo([]byte(r))

	leftDoc, _ := v3.CreateDocument(leftInfo)
	rightDoc, _ := v3.CreateDocument(rightInfo)
	return leftDoc, rightDoc
}

func test_BuildDocv2(l, r string) (*v2.Swagger, *v2.Swagger) {

	leftInfo, _ := datamodel.ExtractSpecInfo([]byte(l))
	rightInfo, _ := datamodel.ExtractSpecInfo([]byte(r))

	var err []error
	var leftDoc, rightDoc *v2.Swagger
	leftDoc, err = v2.CreateDocument(leftInfo)
	rightDoc, err = v2.CreateDocument(rightInfo)

	if len(err) > 0 {
		for i := range err {
			fmt.Printf("error: %v\n", err[i])
		}
		panic("failed to create doc")
	}
	return leftDoc, rightDoc
}

func TestCompareSchemas_RefIgnore(t *testing.T) {
	left := `components:
  schemas:
    Yo:
      type: int
    OK:
      $ref: '#/components/schemas/Yo'`

	right := `components:
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
	left := `components:
  schemas:
    Yo:
      type: int
    OK:
      $ref: '#/components/schemas/No'`

	right := `components:
  schemas:
    Yo:
      type: int
    OK:
      $ref: '#/components/schemas/Woah'`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Components.Value.FindSchema("OK").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("OK").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Len(t, changes.Changes, 1)
	assert.Equal(t, Modified, changes.Changes[0].ChangeType)
	assert.Equal(t, "#/components/schemas/Woah", changes.Changes[0].New)
}

func TestCompareSchemas_RefToInline(t *testing.T) {
	left := `components:
  schemas:
    Yo:
      type: int
    OK:
      $ref: '#/components/schemas/No'`

	right := `components:
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
	assert.Equal(t, Modified, changes.Changes[0].ChangeType)
	assert.Equal(t, v3.RefLabel, changes.Changes[0].Property)
	assert.Equal(t, "#/components/schemas/No", changes.Changes[0].Original)

}

func TestCompareSchemas_InlineToRef(t *testing.T) {
	left := `components:
  schemas:
    Yo:
      type: int
    OK:
      $ref: '#/components/schemas/No'`

	right := `components:
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
	assert.Equal(t, Modified, changes.Changes[0].ChangeType)
	assert.Equal(t, v3.RefLabel, changes.Changes[0].Property)
	assert.Equal(t, "#/components/schemas/No", changes.Changes[0].New)

}

func TestCompareSchemas_Identical(t *testing.T) {
	left := `components:
  schemas:
    Yo:
      type: int
    OK:
      type: string`

	right := `components:
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

func TestCompareSchemas_RequiredAdded(t *testing.T) {
	left := `components:
  schemas:
    OK:
      title: an OK message
      description: a thing
      required:
        - one`

	right := `components:
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
	assert.Equal(t, PropertyAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, "two", changes.Changes[0].New)
	assert.Equal(t, v3.RequiredLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_RequiredRemoved(t *testing.T) {
	left := `components:
  schemas:
    OK:
      required:
        - one`

	right := `components:
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
	assert.Equal(t, PropertyRemoved, changes.Changes[0].ChangeType)
	assert.Equal(t, "two", changes.Changes[0].Original)
	assert.Equal(t, v3.RequiredLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_EnumAdded(t *testing.T) {
	left := `components:
  schemas:
    OK:
      enum: [a,b,c]`

	right := `components:
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
	assert.Equal(t, PropertyAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, "d", changes.Changes[0].New)
	assert.Equal(t, v3.EnumLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_EnumRemoved(t *testing.T) {
	left := `components:
  schemas:
    OK:
      enum: [a,b,c]`

	right := `components:
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
	assert.Equal(t, PropertyRemoved, changes.Changes[0].ChangeType)
	assert.Equal(t, "d", changes.Changes[0].Original)
	assert.Equal(t, v3.EnumLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_PropertyAdded(t *testing.T) {
	left := `components:
  schemas:
    OK:
      properties:
        propA:
          type: int`

	right := `components:
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
	assert.Equal(t, ObjectAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, "propB", changes.Changes[0].New)
	assert.Equal(t, v3.PropertiesLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_PropertyRemoved(t *testing.T) {
	left := `components:
  schemas:
    OK:
      properties:
        propA:
          type: int`

	right := `components:
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
	assert.Equal(t, ObjectRemoved, changes.Changes[0].ChangeType)
	assert.Equal(t, "propB", changes.Changes[0].Original)
	assert.Equal(t, v3.PropertiesLabel, changes.Changes[0].Property)
}

func TestCompareSchemas_PropertyChanged(t *testing.T) {
	left := `components:
  schemas:
    OK:
      properties:
        propA:
          type: int`

	right := `components:
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
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, Modified, changes.SchemaPropertyChanges["propA"].Changes[0].ChangeType)
	assert.Equal(t, "string", changes.SchemaPropertyChanges["propA"].Changes[0].New)
	assert.Equal(t, "int", changes.SchemaPropertyChanges["propA"].Changes[0].Original)
}

func TestCompareSchemas_PropertySwap(t *testing.T) {
	left := `components:
  schemas:
    OK:
      properties:
        propA:
          type: int`

	right := `components:
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
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, "propN", changes.Changes[0].New)
	assert.Equal(t, v3.PropertiesLabel, changes.Changes[0].Property)
	assert.Equal(t, ObjectRemoved, changes.Changes[1].ChangeType)
	assert.Equal(t, "propA", changes.Changes[1].Original)
	assert.Equal(t, v3.PropertiesLabel, changes.Changes[1].Property)
}

func TestCompareSchemas_AnyOfModifyAndAddItem(t *testing.T) {
	left := `components:
  schemas:
    OK:
      anyOf:
        - type: bool`

	right := `components:
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
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, v3.AnyOfLabel, changes.Changes[0].Property)
	assert.Equal(t, Modified, changes.AnyOfChanges[0].Changes[0].ChangeType)
	assert.Equal(t, "string", changes.AnyOfChanges[0].Changes[0].New)
	assert.Equal(t, "bool", changes.AnyOfChanges[0].Changes[0].Original)
}

func TestCompareSchemas_AnyOfModifyAndRemoveItem(t *testing.T) {
	left := `components:
  schemas:
    OK:
      anyOf:
        - type: bool`

	right := `components:
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
	assert.Equal(t, 2, changes.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, changes.Changes[0].ChangeType)
	assert.Equal(t, v3.AnyOfLabel, changes.Changes[0].Property)
	assert.Equal(t, Modified, changes.AnyOfChanges[0].Changes[0].ChangeType)
	assert.Equal(t, "bool", changes.AnyOfChanges[0].Changes[0].New)
	assert.Equal(t, "string", changes.AnyOfChanges[0].Changes[0].Original)
}

func TestCompareSchemas_AnyOfModified(t *testing.T) {
	left := `components:
  schemas:
    OK:
      anyOf:
        - type: bool`

	right := `components:
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
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, Modified, changes.AnyOfChanges[0].Changes[0].ChangeType)
	assert.Equal(t, "string", changes.AnyOfChanges[0].Changes[0].New)
	assert.Equal(t, "bool", changes.AnyOfChanges[0].Changes[0].Original)

}

func TestCompareSchemas_OneOfModifyAndAddItem(t *testing.T) {
	left := `components:
  schemas:
    OK:
      oneOf:
        - type: bool`

	right := `components:
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
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, v3.OneOfLabel, changes.Changes[0].Property)
	assert.Equal(t, Modified, changes.OneOfChanges[0].Changes[0].ChangeType)
	assert.Equal(t, "string", changes.OneOfChanges[0].Changes[0].New)
	assert.Equal(t, "bool", changes.OneOfChanges[0].Changes[0].Original)
}

func TestCompareSchemas_AllOfModifyAndAddItem(t *testing.T) {
	left := `components:
  schemas:
    OK:
      allOf:
        - type: bool`

	right := `components:
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
	assert.Equal(t, 2, changes.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, v3.AllOfLabel, changes.Changes[0].Property)
	assert.Equal(t, Modified, changes.AllOfChanges[0].Changes[0].ChangeType)
	assert.Equal(t, "string", changes.AllOfChanges[0].Changes[0].New)
	assert.Equal(t, "bool", changes.AllOfChanges[0].Changes[0].Original)
}

func TestCompareSchemas_ItemsModifyAndAddItem(t *testing.T) {
	left := `components:
  schemas:
    OK:
      type: string
      items:
        type: bool`

	right := `components:
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
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.TypeLabel, changes.ItemsChanges[0].Changes[0].Property)
	assert.Equal(t, Modified, changes.ItemsChanges[0].Changes[0].ChangeType)
	assert.Equal(t, "string", changes.ItemsChanges[0].Changes[0].New)
	assert.Equal(t, "bool", changes.ItemsChanges[0].Changes[0].Original)
}

func TestCompareSchemas_ItemsModifyAndAddItemArray(t *testing.T) {
	left := `components:
  schemas:
    OK:
      type: string
      items:
        - type: bool`

	right := `components:
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
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.TypeLabel, changes.ItemsChanges[0].Changes[0].Property)
	assert.Equal(t, Modified, changes.ItemsChanges[0].Changes[0].ChangeType)
	assert.Equal(t, "string", changes.ItemsChanges[0].Changes[0].New)
	assert.Equal(t, "bool", changes.ItemsChanges[0].Changes[0].Original)
}

func TestCompareSchemas_NotModifyAndAddItem(t *testing.T) {
	left := `components:
  schemas:
    OK:
      type: string
      not:
        type: bool`

	right := `components:
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
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.TypeLabel, changes.NotChanges[0].Changes[0].Property)
	assert.Equal(t, Modified, changes.NotChanges[0].Changes[0].ChangeType)
	assert.Equal(t, "string", changes.NotChanges[0].Changes[0].New)
	assert.Equal(t, "bool", changes.NotChanges[0].Changes[0].Original)
}

func TestCompareSchemas_DiscriminatorChange(t *testing.T) {
	left := `components:
  schemas:
    OK:
      type: string
      discriminator:
        propertyName: melody`

	right := `components:
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
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.PropertyNameLabel, changes.DiscriminatorChanges.Changes[0].Property)
	assert.Equal(t, Modified, changes.DiscriminatorChanges.Changes[0].ChangeType)
	assert.Equal(t, "maddox", changes.DiscriminatorChanges.Changes[0].New)
	assert.Equal(t, "melody", changes.DiscriminatorChanges.Changes[0].Original)
}

func TestCompareSchemas_DiscriminatorAdd(t *testing.T) {
	left := `components:
  schemas:
    OK:
      type: string`

	right := `components:
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
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.DiscriminatorLabel, changes.Changes[0].Property)
	assert.Equal(t, ObjectAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, "0e563831440581c713657dd857a0ec3af1bd7308a43bd3cae9184f61d61b288f",
		low.HashToString(changes.Changes[0].NewObject.(*base.Discriminator).Hash()))

}

func TestCompareSchemas_DiscriminatorRemove(t *testing.T) {
	left := `components:
  schemas:
    OK:
      type: string`

	right := `components:
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
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.DiscriminatorLabel, changes.Changes[0].Property)
	assert.Equal(t, ObjectRemoved, changes.Changes[0].ChangeType)
	assert.Equal(t, "0e563831440581c713657dd857a0ec3af1bd7308a43bd3cae9184f61d61b288f",
		low.HashToString(changes.Changes[0].OriginalObject.(*base.Discriminator).Hash()))

}

func TestCompareSchemas_ExternalDocsChange(t *testing.T) {
	left := `components:
  schemas:
    OK:
      type: string
      externalDocs:
        url: https://pb33f.io`

	right := `components:
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
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Equal(t, v3.URLLabel, changes.ExternalDocChanges.Changes[0].Property)
	assert.Equal(t, Modified, changes.ExternalDocChanges.Changes[0].ChangeType)
	assert.Equal(t, "https://pb33f.io/new", changes.ExternalDocChanges.Changes[0].New)
	assert.Equal(t, "https://pb33f.io", changes.ExternalDocChanges.Changes[0].Original)
}

func TestCompareSchemas_ExternalDocsAdd(t *testing.T) {
	left := `components:
  schemas:
    OK:
      type: string`

	right := `components:
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
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ExternalDocsLabel, changes.Changes[0].Property)
	assert.Equal(t, ObjectAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, "2b7adf30f2ea3a7617ccf429a099617a9c03e8b5f3a23a89dba4b90f760010d7",
		low.HashToString(changes.Changes[0].NewObject.(*base.ExternalDoc).Hash()))

}

func TestCompareSchemas_ExternalDocsRemove(t *testing.T) {
	left := `components:
  schemas:
    OK:
      type: string`

	right := `components:
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
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ExternalDocsLabel, changes.Changes[0].Property)
	assert.Equal(t, ObjectRemoved, changes.Changes[0].ChangeType)
	assert.Equal(t, "2b7adf30f2ea3a7617ccf429a099617a9c03e8b5f3a23a89dba4b90f760010d7",
		low.HashToString(changes.Changes[0].OriginalObject.(*base.ExternalDoc).Hash()))

}

func TestCompareSchemas_AddExtension(t *testing.T) {
	left := `components:
  schemas:
    OK:
      type: string`

	right := `components:
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
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Equal(t, "x-melody", changes.ExtensionChanges.Changes[0].Property)
	assert.Equal(t, ObjectAdded, changes.ExtensionChanges.Changes[0].ChangeType)
	assert.Equal(t, "song", changes.ExtensionChanges.Changes[0].New)
}

func TestCompareSchemas_ExampleChange(t *testing.T) {
	left := `components:
  schemas:
    OK:
      example: sausages`

	right := `components:
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
	left := `components:
  schemas:
    OK:
      title: nice`

	right := `components:
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
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ExampleLabel, changes.Changes[0].Property)
	assert.Equal(t, PropertyAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, "yellow boat", changes.Changes[0].New)
}

func TestCompareSchemas_ExampleRemove(t *testing.T) {
	left := `components:
  schemas:
    OK:
      title: nice`

	right := `components:
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
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ExampleLabel, changes.Changes[0].Property)
	assert.Equal(t, PropertyRemoved, changes.Changes[0].ChangeType)
	assert.Equal(t, "yellow boat", changes.Changes[0].Original)
}

func TestCompareSchemas_ExamplesChange(t *testing.T) {
	left := `components:
  schemas:
    OK:
      title: nice
      examples:
        - sausages`

	right := `components:
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
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ExamplesLabel, changes.Changes[0].Property)
	assert.Equal(t, Modified, changes.Changes[0].ChangeType)
	assert.Equal(t, "yellow boat", changes.Changes[0].New)
	assert.Equal(t, "sausages", changes.Changes[0].Original)
}

func TestCompareSchemas_ExamplesAdd(t *testing.T) {
	left := `components:
  schemas:
    OK:
      title: nice`

	right := `components:
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
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ExamplesLabel, changes.Changes[0].Property)
	assert.Equal(t, ObjectAdded, changes.Changes[0].ChangeType)
	assert.Equal(t, "yellow boat", changes.Changes[0].New)
}

func TestCompareSchemas_ExamplesAddAndModify(t *testing.T) {
	left := `components:
  schemas:
    OK:
      title: nice
      examples:
        - sausages`

	right := `components:
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
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ExamplesLabel, changes.Changes[0].Property)
	assert.Equal(t, Modified, changes.Changes[0].ChangeType)
	assert.Equal(t, "yellow boat", changes.Changes[0].New)
	assert.Equal(t, "sausages", changes.Changes[0].Original)
	assert.Equal(t, ObjectAdded, changes.Changes[1].ChangeType)
	assert.Equal(t, "seal pup", changes.Changes[1].New)
}

func TestCompareSchemas_ExamplesRemove(t *testing.T) {
	left := `components:
  schemas:
    OK:
      title: nice`

	right := `components:
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
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ExamplesLabel, changes.Changes[0].Property)
	assert.Equal(t, ObjectRemoved, changes.Changes[0].ChangeType)
	assert.Equal(t, "yellow boat", changes.Changes[0].Original)
}

func TestCompareSchemas_ExamplesRemoveAndModify(t *testing.T) {
	left := `components:
  schemas:
    OK:
      title: nice
      examples:
        - sausages`

	right := `components:
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
	assert.Equal(t, 0, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ExamplesLabel, changes.Changes[0].Property)
	assert.Equal(t, Modified, changes.Changes[0].ChangeType)
	assert.Equal(t, "yellow boat", changes.Changes[0].Original)
	assert.Equal(t, "sausages", changes.Changes[0].New)
	assert.Equal(t, ObjectRemoved, changes.Changes[1].ChangeType)
	assert.Equal(t, "seal pup", changes.Changes[1].Original)
}

func TestCompareSchemas_XMLChange(t *testing.T) {
	left := `components:
 schemas:
   OK:
     xml:
       name: baby xml`

	right := `components:
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
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.NameLabel, changes.XMLChanges.Changes[0].Property)
	assert.Equal(t, Modified, changes.XMLChanges.Changes[0].ChangeType)
	assert.Equal(t, "big xml", changes.XMLChanges.Changes[0].New)
	assert.Equal(t, "baby xml", changes.XMLChanges.Changes[0].Original)
}

func TestCompareSchemas_XMLAdd(t *testing.T) {
	left := `components:
  schemas:
    OK:
      description: OK`

	right := `components:
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
	left := `components:
 schemas:
   OK:`

	right := `components:
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
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.XMLLabel, changes.Changes[0].Property)
	assert.Equal(t, ObjectRemoved, changes.Changes[0].ChangeType)
	assert.Equal(t, "big xml", changes.Changes[0].OriginalObject.(*base.XML).Name.Value)
}
