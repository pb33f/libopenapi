// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT
package libopenapi

import (
	"fmt"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/what-changed/model"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"strings"
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
	assert.Len(t, docErr, 2)
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
	assert.Len(t, docErr, 2)
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

func TestDocument_RenderAndReload_ChangeCheck_Burgershop(t *testing.T) {

	bs, _ := os.ReadFile("test_specs/burgershop.openapi.yaml")
	doc, _ := NewDocument(bs)
	doc.BuildV3Model()

	rend, newDoc, _, _ := doc.RenderAndReload()

	// compare documents
	compReport, errs := CompareDocuments(doc, newDoc)

	// should noth be nil.
	assert.Nil(t, errs)
	assert.NotNil(t, rend)
	assert.Nil(t, compReport)

}

func TestDocument_RenderAndReload_ChangeCheck_Stripe(t *testing.T) {

	bs, _ := os.ReadFile("test_specs/stripe.yaml")
	doc, _ := NewDocument(bs)
	doc.BuildV3Model()

	_, newDoc, _, _ := doc.RenderAndReload()

	// compare documents
	compReport, errs := CompareDocuments(doc, newDoc)

	// get flat list of changes.
	flatChanges := compReport.GetAllChanges()

	// remove everything that is a description change (stripe has a lot of those from having 519 empty descriptions)
	var filtered []*model.Change
	for i := range flatChanges {
		if flatChanges[i].Property != "description" {
			filtered = append(filtered, flatChanges[i])
		}
	}

	assert.Nil(t, errs)
	tc := compReport.TotalChanges()
	bc := compReport.TotalBreakingChanges()
	assert.Equal(t, 0, bc)
	assert.Equal(t, 519, tc)

	// there should be no other changes than the 519 descriptions.
	assert.Equal(t, 0, len(filtered))

}

func TestDocument_RenderAndReload_ChangeCheck_Asana(t *testing.T) {

	bs, _ := os.ReadFile("test_specs/asana.yaml")
	doc, _ := NewDocument(bs)
	doc.BuildV3Model()

	dat, newDoc, _, _ := doc.RenderAndReload()
	assert.NotNil(t, dat)

	// compare documents
	compReport, errs := CompareDocuments(doc, newDoc)

	// get flat list of changes.
	flatChanges := compReport.GetAllChanges()

	assert.Nil(t, errs)
	tc := compReport.TotalChanges()
	assert.Equal(t, 21, tc)

	// there are some properties re-rendered that trigger changes.
	assert.Equal(t, 21, len(flatChanges))

}

func TestDocument_RenderAndReload(t *testing.T) {

	// load an OpenAPI 3 specification from bytes
	petstore, _ := ioutil.ReadFile("test_specs/petstorev3.json")

	// create a new document from specification bytes
	doc, err := NewDocument(petstore)

	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	// because we know this is a v3 spec, we can build a ready to go model from it.
	m, _ := doc.BuildV3Model()

	// mutate the model
	h := m.Model
	h.Paths.PathItems["/pet/findByStatus"].Get.OperationId = "findACakeInABakery"
	h.Paths.PathItems["/pet/findByStatus"].Get.Responses.Codes["400"].Description = "a nice bucket of mice"
	h.Paths.PathItems["/pet/findByTags"].Get.Tags =
		append(h.Paths.PathItems["/pet/findByTags"].Get.Tags, "gurgle", "giggle")

	h.Paths.PathItems["/pet/{petId}"].Delete.Security = append(h.Paths.PathItems["/pet/{petId}"].Delete.Security,
		&base.SecurityRequirement{Requirements: map[string][]string{
			"pizza-and-cake": {"read:abook", "write:asong"},
		}})

	h.Components.Schemas["Order"].Schema().Properties["status"].Schema().Example = "I am a teapot, filled with love."
	h.Components.SecuritySchemes["petstore_auth"].Flows.Implicit.AuthorizationUrl = "https://pb33f.io"

	bytes, _, newDocModel, e := doc.RenderAndReload()
	assert.Nil(t, e)
	assert.NotNil(t, bytes)

	h = newDocModel.Model
	assert.Equal(t, "findACakeInABakery", h.Paths.PathItems["/pet/findByStatus"].Get.OperationId)
	assert.Equal(t, "a nice bucket of mice",
		h.Paths.PathItems["/pet/findByStatus"].Get.Responses.Codes["400"].Description)
	assert.Len(t, h.Paths.PathItems["/pet/findByTags"].Get.Tags, 3)

	assert.Len(t, h.Paths.PathItems["/pet/findByTags"].Get.Tags, 3)
	yu := h.Paths.PathItems["/pet/{petId}"].Delete.Security
	assert.Equal(t, "read:abook", yu[len(yu)-1].Requirements["pizza-and-cake"][0])
	assert.Equal(t, "I am a teapot, filled with love.",
		h.Components.Schemas["Order"].Schema().Properties["status"].Schema().Example)

	assert.Equal(t, "https://pb33f.io",
		h.Components.SecuritySchemes["petstore_auth"].Flows.Implicit.AuthorizationUrl)

}
func TestDocument_RenderAndReload_Swagger(t *testing.T) {
	petstore, _ := ioutil.ReadFile("test_specs/petstorev2.json")
	doc, _ := NewDocument(petstore)
	doc.BuildV2Model()
	doc.BuildV2Model()
	_, _, _, e := doc.RenderAndReload()
	assert.Len(t, e, 1)
	assert.Equal(t, "this method only supports OpenAPI 3 documents, not Swagger", e[0].Error())

}

func TestDocument_BuildModelPreBuild(t *testing.T) {
	petstore, _ := ioutil.ReadFile("test_specs/petstorev3.json")
	doc, _ := NewDocument(petstore)
	doc.BuildV3Model()
	doc.BuildV3Model()
	_, _, _, e := doc.RenderAndReload()
	assert.Len(t, e, 0)
}

func TestDocument_BuildModelCircular(t *testing.T) {
	petstore, _ := ioutil.ReadFile("test_specs/circular-tests.yaml")
	doc, _ := NewDocument(petstore)
	m, e := doc.BuildV3Model()
	assert.NotNil(t, m)
	assert.Len(t, e, 3)
}

func TestDocument_BuildModelBad(t *testing.T) {
	petstore, _ := ioutil.ReadFile("test_specs/badref-burgershop.openapi.yaml")
	doc, _ := NewDocument(petstore)
	m, e := doc.BuildV3Model()
	assert.Nil(t, m)
	assert.Len(t, e, 9)
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

func TestExtractReference(t *testing.T) {
	var data = `
openapi: "3.1"
components:
  parameters:
    Param1:
      description: "I am a param"
paths:
  /something:
    get:
      parameters:
        - $ref: '#/components/parameters/Param1'`

	doc, err := NewDocument([]byte(data))
	if err != nil {
		panic(err)
	}

	result, errs := doc.BuildV3Model()
	if len(errs) > 0 {
		panic(errs)
	}

	// extract operation.
	operation := result.Model.Paths.PathItems["/something"].Get

	// print it out.
	fmt.Printf("param1: %s, is reference? %t, original reference %s",
		operation.Parameters[0].Description, operation.GoLow().Parameters.Value[0].IsReference(),
		operation.GoLow().Parameters.Value[0].Reference)
}

func TestDocument_BuildModel_CompareDocsV3_LeftError(t *testing.T) {
	burgerShopOriginal, _ := ioutil.ReadFile("test_specs/badref-burgershop.openapi.yaml")
	burgerShopUpdated, _ := ioutil.ReadFile("test_specs/burgershop.openapi-modified.yaml")
	originalDoc, _ := NewDocument(burgerShopOriginal)
	updatedDoc, _ := NewDocument(burgerShopUpdated)
	changes, errors := CompareDocuments(originalDoc, updatedDoc)
	assert.Len(t, errors, 9)
	assert.Nil(t, changes)
}

func TestDocument_BuildModel_CompareDocsV3_RightError(t *testing.T) {

	burgerShopOriginal, _ := ioutil.ReadFile("test_specs/badref-burgershop.openapi.yaml")
	burgerShopUpdated, _ := ioutil.ReadFile("test_specs/burgershop.openapi-modified.yaml")
	originalDoc, _ := NewDocument(burgerShopOriginal)
	updatedDoc, _ := NewDocument(burgerShopUpdated)
	changes, errors := CompareDocuments(updatedDoc, originalDoc)
	assert.Len(t, errors, 9)
	assert.Nil(t, changes)

}

func TestDocument_BuildModel_CompareDocsV2_Error(t *testing.T) {

	burgerShopOriginal, _ := ioutil.ReadFile("test_specs/petstorev2-badref.json")
	burgerShopUpdated, _ := ioutil.ReadFile("test_specs/petstorev2-badref.json")
	originalDoc, _ := NewDocument(burgerShopOriginal)
	updatedDoc, _ := NewDocument(burgerShopUpdated)
	changes, errors := CompareDocuments(updatedDoc, originalDoc)
	assert.Len(t, errors, 2)
	assert.Nil(t, changes)

}

func TestDocument_BuildModel_CompareDocsV2V3Mix_Error(t *testing.T) {

	burgerShopOriginal, _ := ioutil.ReadFile("test_specs/petstorev2.json")
	burgerShopUpdated, _ := ioutil.ReadFile("test_specs/petstorev3.json")
	originalDoc, _ := NewDocument(burgerShopOriginal)
	updatedDoc, _ := NewDocument(burgerShopUpdated)
	changes, errors := CompareDocuments(updatedDoc, originalDoc)
	assert.Len(t, errors, 1)
	assert.Nil(t, changes)

}

func TestSchemaRefIsFollowed(t *testing.T) {
	petstore, _ := ioutil.ReadFile("test_specs/ref-followed.yaml")

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
	schemas := v3Model.Model.Components.Schemas
	assert.Equal(t, 4, len(schemas))

	fp := schemas["FP"]
	fbsref := schemas["FBSRef"]

	assert.Equal(t, fp.Schema().Pattern, fbsref.Schema().Pattern)
	assert.Equal(t, fp.Schema().Example, fbsref.Schema().Example)

	byte := schemas["Byte"]
	uint64 := schemas["UInt64"]

	assert.Equal(t, uint64.Schema().Format, byte.Schema().Format)
	assert.Equal(t, uint64.Schema().Type, byte.Schema().Type)
	assert.Equal(t, uint64.Schema().Nullable, byte.Schema().Nullable)
	assert.Equal(t, uint64.Schema().Example, byte.Schema().Example)
	assert.Equal(t, uint64.Schema().Minimum, byte.Schema().Minimum)
}

func TestDocument_ParamsAndRefsRender(t *testing.T) {
	var d = `openapi: "3.1"
components:
    parameters:
        limit:
            description: I am a param
        offset:
            description: I am a param
paths:
    /webhooks:
        get:
            description: Get the compact representation of all webhooks your app has registered for the authenticated user in the given workspace.
            operationId: getWebhooks
            parameters:
                - $ref: '#/components/parameters/limit'
                - $ref: '#/components/parameters/offset'
                - description: The workspace to query for webhooks in.
                  example: "1331"
                  in: query
                  name: workspace
                  required: true
                  schema:
                    type: string
                - description: Only return webhooks for the given resource.
                  example: "51648"
                  in: query
                  name: resource
                  schema:
                    type: string`

	doc, err := NewDocument([]byte(d))
	if err != nil {
		panic(err)
	}

	result, errs := doc.BuildV3Model()
	if len(errs) > 0 {
		panic(errs)
	}

	// render the document.
	rend, _ := result.Model.Render()

	assert.Equal(t, d, strings.TrimSpace(string(rend)))
}

func TestDocument_RemoteWithoutBaseURL(t *testing.T) {

	// This test will push the index to do try and locate remote references that use relative references
	spec := `openapi: 3.0.2
info:
  title: Test
  version: 1.0.0
paths:
  /test:
    get:
      parameters:
        - $ref: "https://schemas.opengis.net/ogcapi/features/part2/1.0/openapi/ogcapi-features-2.yaml#/components/parameters/crs"`

	config := datamodel.NewOpenDocumentConfiguration()

	doc, err := NewDocumentWithConfiguration([]byte(spec), config)
	if err != nil {
		panic(err)
	}

	result, errs := doc.BuildV3Model()
	if len(errs) > 0 {
		panic(errs)
	}

	assert.Equal(t, "crs", result.Model.Paths.PathItems["/test"].Get.Parameters[0].Name)
}

func TestDocument_ExampleMap(t *testing.T) {
	var d = `openapi: "3.1"
components:
    schemas:
        ProjectRequest:
            allOf:
                - properties:
                    custom_fields:
                        additionalProperties:
                            description: '"{custom_field_gid}" => Value (Can be text, number, etc.)'
                            type: string
                        description: An object where each key is a Custom Field gid and each value is an enum gid, string, or number.
                        example:
                            "4578152156": Not Started
                            "5678904321": On Hold
                        type: object`

	doc, err := NewDocument([]byte(d))
	if err != nil {
		panic(err)
	}

	result, errs := doc.BuildV3Model()
	if len(errs) > 0 {
		panic(errs)
	}

	// render the document.
	rend, _ := result.Model.Render()

	assert.Len(t, rend, 644)
}

func TestDocument_OperationsAsRefs(t *testing.T) {

	ae := `operationId: thisIsAnOperationId
summary: a test thing
description: this is a test, that does a test.`

	_ = os.WriteFile("test-operation.yaml", []byte(ae), 0644)

	var d = `openapi: "3.1"
paths:
    /an/operation:
        get:
            $ref: test-operation.yaml`

	doc, err := NewDocumentWithConfiguration([]byte(d), datamodel.NewOpenDocumentConfiguration())
	if err != nil {
		panic(err)
	}

	result, errs := doc.BuildV3Model()
	if len(errs) > 0 {
		panic(errs)
	}

	// render the document.
	rend, _ := result.Model.Render()

	assert.Equal(t, d, strings.TrimSpace(string(rend)))
}
