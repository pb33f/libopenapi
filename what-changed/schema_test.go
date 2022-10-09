// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

import (
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/low/base"
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
