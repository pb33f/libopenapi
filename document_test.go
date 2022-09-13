// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLoadDocument_Simple_V2(t *testing.T) {

	yml := `swagger: 2.0.1`
	doc, err := NewDocument([]byte(yml))
	assert.NoError(t, err)
	assert.Equal(t, "2.0.1", doc.GetVersion())

	v2Doc, docErr := doc.BuildV2Document()
	assert.Len(t, docErr, 0)
	assert.NotNil(t, v2Doc)
	assert.NotNil(t, doc.GetSpecInfo())

	fmt.Print()

}

func TestLoadDocument_Simple_V2_Error(t *testing.T) {

	yml := `swagger: 2.0`
	doc, err := NewDocument([]byte(yml))
	assert.NoError(t, err)

	v2Doc, docErr := doc.BuildV3Document()
	assert.Len(t, docErr, 1)
	assert.Nil(t, v2Doc)
}

func TestLoadDocument_Simple_V2_Error_BadSpec(t *testing.T) {

	yml := `swagger: 2.0
definitions:
  thing:
    $ref: bork`
	doc, err := NewDocument([]byte(yml))
	assert.NoError(t, err)

	v2Doc, docErr := doc.BuildV2Document()
	assert.Len(t, docErr, 1)
	assert.Nil(t, v2Doc)
}

func TestLoadDocument_Simple_V3_Error(t *testing.T) {

	yml := `openapi: 3.0.1`
	doc, err := NewDocument([]byte(yml))
	assert.NoError(t, err)

	v2Doc, docErr := doc.BuildV2Document()
	assert.Len(t, docErr, 1)
	assert.Nil(t, v2Doc)
}

func TestLoadDocument_Error_V2NoSpec(t *testing.T) {

	doc := new(Document) // not how this should be instantiated.
	_, err := doc.BuildV2Document()
	assert.Len(t, err, 1)
}

func TestLoadDocument_Error_V3NoSpec(t *testing.T) {

	doc := new(Document) // not how this should be instantiated.
	_, err := doc.BuildV3Document()
	assert.Len(t, err, 1)
}

func TestLoadDocument_Empty(t *testing.T) {
	yml := ``
	_, err := NewDocument([]byte(yml))
	assert.Error(t, err)
}

func TestLoadDocument_Simple_V3(t *testing.T) {

	yml := `openapi: 3.0.1`
	doc, err := NewDocument([]byte(yml))
	assert.NoError(t, err)
	assert.Equal(t, "3.0.1", doc.GetVersion())

	v3Doc, docErr := doc.BuildV3Document()
	assert.Len(t, docErr, 0)
	assert.NotNil(t, v3Doc)
}

func TestLoadDocument_Simple_V3_Error_BadSpec(t *testing.T) {

	yml := `openapi: 3.0
paths:
  "/some":
    $ref: bork`
	doc, err := NewDocument([]byte(yml))
	assert.NoError(t, err)

	v3Doc, docErr := doc.BuildV3Document()
	assert.Len(t, docErr, 1)
	assert.Nil(t, v3Doc)
}

func TestDocument_Serialize_Error(t *testing.T) {
	doc := new(Document) // not how this should be instantiated.
	_, err := doc.Serialize()
	assert.Error(t, err)
}

func TestDocument_Serialize(t *testing.T) {

	yml := `openapi: 3.0
info:
    title: The magic API
`
	doc, _ := NewDocument([]byte(yml))
	serial, err := doc.Serialize()
	assert.NoError(t, err)
	assert.Equal(t, yml, string(serial))
}

func TestDocument_Serialize_Modified(t *testing.T) {

	yml := `openapi: 3.0
info:
    title: The magic API
`

	ymlModified := `openapi: 3.0
info:
    title: The magic API - but now, altered!
`
	doc, _ := NewDocument([]byte(yml))

	v3Doc, _ := doc.BuildV3Document()

	v3Doc.Model.Info.GoLow().Title.Mutate("The magic API - but now, altered!")

	serial, err := doc.Serialize()
	assert.NoError(t, err)
	assert.Equal(t, ymlModified, string(serial))
}

func TestDocument_Serialize_JSON_Modified(t *testing.T) {

	json := `{ 'openapi': '3.0',
 'info': {
   'title': 'The magic API'
 }
}
`
	jsonModified := `{"info":{"title":"The magic API - but now, altered!"},"openapi":"3.0"}`
	doc, _ := NewDocument([]byte(json))

	v3Doc, _ := doc.BuildV3Document()

	// eventually this will be encapsulated up high.
	// mutation does not replace low model, eventually pointers will be used.
	newTitle := v3Doc.Model.Info.GoLow().Title.Mutate("The magic API - but now, altered!")
	v3Doc.Model.Info.GoLow().Title = newTitle

	assert.Equal(t, "The magic API - but now, altered!", v3Doc.Model.Info.GoLow().Title.Value)

	serial, err := doc.Serialize()
	assert.NoError(t, err)
	assert.Equal(t, jsonModified, string(serial))
}
