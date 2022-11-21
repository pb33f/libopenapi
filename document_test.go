// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package libopenapi

import (
	"fmt"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestLoadDocument_Simple_V2(t *testing.T) {

	yml := `swagger: 2.0.1`
	doc, err := NewDocument([]byte(yml))
	assert.NoError(t, err)
	assert.Equal(t, "2.0.1", doc.GetVersion())

	v2Doc, docErr := doc.BuildV2Model()
	assert.Len(t, docErr, 0)
	assert.NotNil(t, v2Doc)
	assert.NotNil(t, doc.GetSpecInfo())

	fmt.Print()

}

func TestLoadDocument_Simple_V2_Error(t *testing.T) {

	yml := `swagger: 2.0`
	doc, err := NewDocument([]byte(yml))
	assert.NoError(t, err)

	v2Doc, docErr := doc.BuildV3Model()
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

	v2Doc, docErr := doc.BuildV2Model()
	assert.Len(t, docErr, 1)
	assert.Nil(t, v2Doc)
}

func TestLoadDocument_Simple_V3_Error(t *testing.T) {

	yml := `openapi: 3.0.1`
	doc, err := NewDocument([]byte(yml))
	assert.NoError(t, err)

	v2Doc, docErr := doc.BuildV2Model()
	assert.Len(t, docErr, 1)
	assert.Nil(t, v2Doc)
}

func TestLoadDocument_Error_V2NoSpec(t *testing.T) {

	doc := new(document) // not how this should be instantiated.
	_, err := doc.BuildV2Model()
	assert.Len(t, err, 1)
}

func TestLoadDocument_Error_V3NoSpec(t *testing.T) {

	doc := new(document) // not how this should be instantiated.
	_, err := doc.BuildV3Model()
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

	v3Doc, docErr := doc.BuildV3Model()
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

	v3Doc, docErr := doc.BuildV3Model()
	assert.Len(t, docErr, 1)
	assert.Nil(t, v3Doc)
}

func TestDocument_Serialize_Error(t *testing.T) {
	doc := new(document) // not how this should be instantiated.
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

	v3Doc, _ := doc.BuildV3Model()

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

	v3Doc, _ := doc.BuildV3Model()

	// eventually this will be encapsulated up high.
	// mutation does not replace low model, eventually pointers will be used.
	newTitle := v3Doc.Model.Info.GoLow().Title.Mutate("The magic API - but now, altered!")
	v3Doc.Model.Info.GoLow().Title = newTitle

	assert.Equal(t, "The magic API - but now, altered!", v3Doc.Model.Info.GoLow().Title.Value)

	serial, err := doc.Serialize()
	assert.NoError(t, err)
	assert.Equal(t, jsonModified, string(serial))
}

func ExampleNewDocument_fromOpenAPI3Document() {

	// How to read in an OpenAPI 3 Specification, into a Document.

	// load an OpenAPI 3 specification from bytes
	petstore, _ := ioutil.ReadFile("test_specs/petstorev3.json")

	// create a new document from specification bytes
	document, err := NewDocument(petstore)

	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	// because we know this is a v3 spec, we can build a ready to go model from it.
	v3Model, errors := document.BuildV3Model()

	// if anything went wrong when building the v3 model, a slice of errors will be returned
	if len(errors) > 0 {
		for i := range errors {
			fmt.Printf("error: %e\n", errors[i])
		}
		panic(fmt.Sprintf("cannot create v3 model from document: %d errors reported", len(errors)))
	}

	// get a count of the number of paths and schemas.
	paths := len(v3Model.Model.Paths.PathItems)
	schemas := len(v3Model.Model.Components.Schemas)

	// print the number of paths and schemas in the document
	fmt.Printf("There are %d paths and %d schemas in the document", paths, schemas)
	// Output: There are 13 paths and 8 schemas in the document
}

func ExampleNewDocument_fromSwaggerDocument() {

	// How to read in a Swagger / OpenAPI 2 Specification, into a Document.

	// load a Swagger specification from bytes
	petstore, _ := ioutil.ReadFile("test_specs/petstorev2.json")

	// create a new document from specification bytes
	document, err := NewDocument(petstore)

	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	// because we know this is a v2 spec, we can build a ready to go model from it.
	v2Model, errors := document.BuildV2Model()

	// if anything went wrong when building the v3 model, a slice of errors will be returned
	if len(errors) > 0 {
		for i := range errors {
			fmt.Printf("error: %e\n", errors[i])
		}
		panic(fmt.Sprintf("cannot create v3 model from document: %d errors reported", len(errors)))
	}

	// get a count of the number of paths and schemas.
	paths := len(v2Model.Model.Paths.PathItems)
	schemas := len(v2Model.Model.Definitions.Definitions)

	// print the number of paths and schemas in the document
	fmt.Printf("There are %d paths and %d schemas in the document", paths, schemas)
	// Output: There are 14 paths and 6 schemas in the document
}

func ExampleNewDocument_fromUnknownVersion() {

	// load an unknown version of an OpenAPI spec
	petstore, _ := ioutil.ReadFile("test_specs/burgershop.openapi.yaml")

	// create a new document from specification bytes
	document, err := NewDocument(petstore)

	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	var paths, schemas int
	var errors []error

	// We don't know which type of document this is, so we can use the spec info to inform us
	if document.GetSpecInfo().SpecType == utils.OpenApi3 {
		v3Model, errs := document.BuildV3Model()
		if len(errs) > 0 {
			errors = errs
		}
		if len(errors) <= 0 {
			paths = len(v3Model.Model.Paths.PathItems)
			schemas = len(v3Model.Model.Components.Schemas)
		}
	}
	if document.GetSpecInfo().SpecType == utils.OpenApi2 {
		v2Model, errs := document.BuildV2Model()
		if len(errs) > 0 {
			errors = errs
		}
		if len(errors) <= 0 {
			paths = len(v2Model.Model.Paths.PathItems)
			schemas = len(v2Model.Model.Definitions.Definitions)
		}
	}

	// if anything went wrong when building the model, report errors.
	if len(errors) > 0 {
		for i := range errors {
			fmt.Printf("error: %e\n", errors[i])
		}
		panic(fmt.Sprintf("cannot create v3 model from document: %d errors reported", len(errors)))
	}

	// print the number of paths and schemas in the document
	fmt.Printf("There are %d paths and %d schemas in the document", paths, schemas)
	// Output: There are 5 paths and 6 schemas in the document
}

func ExampleNewDocument_mutateValuesAndSerialize() {

	// How to mutate values in an OpenAPI Specification, without re-ordering original content.

	// create very small, and useless spec that does nothing useful, except showcase this feature.
	spec := `
openapi: 3.1.0
info:
  title: This is a title
  contact:
    name: Some Person
    email: some@emailaddress.com
  license:
    url: http://some-place-on-the-internet.com/license
`
	// create a new document from specification bytes
	document, err := NewDocument([]byte(spec))

	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	// because we know this is a v3 spec, we can build a ready to go model from it.
	v3Model, errors := document.BuildV3Model()

	// if anything went wrong when building the v3 model, a slice of errors will be returned
	if len(errors) > 0 {
		for i := range errors {
			fmt.Printf("error: %e\n", errors[i])
		}
		panic(fmt.Sprintf("cannot create v3 model from document: %d errors reported", len(errors)))
	}

	// mutate the title, to do this we currently need to drop down to the low-level API.
	v3Model.Model.GoLow().Info.Value.Title.Mutate("A new title for a useless spec")

	// mutate the email address in the contact object.
	v3Model.Model.GoLow().Info.Value.Contact.Value.Email.Mutate("buckaroo@pb33f.io")

	// mutate the name in the contact object.
	v3Model.Model.GoLow().Info.Value.Contact.Value.Name.Mutate("Buckaroo")

	// mutate the URL for the license object.
	v3Model.Model.GoLow().Info.Value.License.Value.URL.Mutate("https://pb33f.io/license")

	// serialize the document back into the original YAML or JSON
	mutatedSpec, serialError := document.Serialize()

	// if something went wrong serializing
	if serialError != nil {
		panic(fmt.Sprintf("cannot serialize document: %e", serialError))
	}

	// print our modified spec!
	fmt.Println(string(mutatedSpec))
	// Output: openapi: 3.1.0
	//info:
	//     title: A new title for a useless spec
	//     contact:
	//         name: Buckaroo
	//         email: buckaroo@pb33f.io
	//     license:
	//         url: https://pb33f.io/license
}

func ExampleCompareDocuments_openAPI() {

	// How to compare two different OpenAPI specifications.

	// load an original OpenAPI 3 specification from bytes
	burgerShopOriginal, _ := ioutil.ReadFile("test_specs/burgershop.openapi.yaml")

	// load an **updated** OpenAPI 3 specification from bytes
	burgerShopUpdated, _ := ioutil.ReadFile("test_specs/burgershop.openapi-modified.yaml")

	// create a new document from original specification bytes
	originalDoc, err := NewDocument(burgerShopOriginal)

	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	// create a new document from updated specification bytes
	updatedDoc, err := NewDocument(burgerShopUpdated)

	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	// Compare documents for all changes made
	documentChanges, errs := CompareDocuments(originalDoc, updatedDoc)

	// If anything went wrong when building models for documents.
	if len(errs) > 0 {
		for i := range errs {
			fmt.Printf("error: %e\n", errs[i])
		}
		panic(fmt.Sprintf("cannot compare documents: %d errors reported", len(errs)))
	}

	// Extract SchemaChanges from components changes.
	schemaChanges := documentChanges.ComponentsChanges.SchemaChanges

	// Print out some interesting stats about the OpenAPI document changes.
	fmt.Printf("There are %d changes, of which %d are breaking. %v schemas have changes.",
		documentChanges.TotalChanges(), documentChanges.TotalBreakingChanges(), len(schemaChanges))
	//Output: There are 67 changes, of which 17 are breaking. 5 schemas have changes.

}

func ExampleCompareDocuments_swagger() {

	// How to compare two different Swagger specifications.

	// load an original OpenAPI 3 specification from bytes
	petstoreOriginal, _ := ioutil.ReadFile("test_specs/petstorev2-complete.yaml")

	// load an **updated** OpenAPI 3 specification from bytes
	petstoreUpdated, _ := ioutil.ReadFile("test_specs/petstorev2-complete-modified.yaml")

	// create a new document from original specification bytes
	originalDoc, err := NewDocument(petstoreOriginal)

	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	// create a new document from updated specification bytes
	updatedDoc, err := NewDocument(petstoreUpdated)

	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	// Compare documents for all changes made
	documentChanges, errs := CompareDocuments(originalDoc, updatedDoc)

	// If anything went wrong when building models for documents.
	if len(errs) > 0 {
		for i := range errs {
			fmt.Printf("error: %e\n", errs[i])
		}
		panic(fmt.Sprintf("cannot compare documents: %d errors reported", len(errs)))
	}

	// Extract SchemaChanges from components changes.
	schemaChanges := documentChanges.ComponentsChanges.SchemaChanges

	// Print out some interesting stats about the Swagger document changes.
	fmt.Printf("There are %d changes, of which %d are breaking. %v schemas have changes.",
		documentChanges.TotalChanges(), documentChanges.TotalBreakingChanges(), len(schemaChanges))
	//Output: There are 52 changes, of which 27 are breaking. 5 schemas have changes.

}

func TestDocument_Paths_As_Array(t *testing.T) {

	// paths can now be wrapped in an array.
	spec := `{
    "openapi": "3.1.0",
    "paths": [
        "/": {
            "get": {}
        }
    ]
}
`
	// create a new document from specification bytes
	doc, err := NewDocument([]byte(spec))

	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}
	v3Model, _ := doc.BuildV3Model()
	assert.NotNil(t, v3Model)
}
