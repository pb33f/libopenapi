// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT
package libopenapi

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/what-changed/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	petstore, _ := os.ReadFile("test_specs/petstorev3.json")

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
	h.Paths.PathItems.GetOrZero("/pet/findByStatus").Get.OperationId = "findACakeInABakery"
	h.Paths.PathItems.GetOrZero("/pet/findByStatus").Get.Responses.Codes.GetOrZero("400").Description = "a nice bucket of mice"
	h.Paths.PathItems.GetOrZero("/pet/findByTags").Get.Tags =
		append(h.Paths.PathItems.GetOrZero("/pet/findByTags").Get.Tags, "gurgle", "giggle")

	h.Paths.PathItems.GetOrZero("/pet/{petId}").Delete.Security = append(h.Paths.PathItems.GetOrZero("/pet/{petId}").Delete.Security,
		&base.SecurityRequirement{Requirements: orderedmap.ToOrderedMap(map[string][]string{
			"pizza-and-cake": {"read:abook", "write:asong"},
		})},
	)

	h.Components.Schemas.GetOrZero("Order").Schema().Properties.GetOrZero("status").Schema().Example = "I am a teapot, filled with love."
	h.Components.SecuritySchemes.GetOrZero("petstore_auth").Flows.Implicit.AuthorizationUrl = "https://pb33f.io"

	bytes, _, newDocModel, e := doc.RenderAndReload()
	assert.Nil(t, e)
	assert.NotNil(t, bytes)

	h = newDocModel.Model
	assert.Equal(t, "findACakeInABakery", h.Paths.PathItems.GetOrZero("/pet/findByStatus").Get.OperationId)
	assert.Equal(t, "a nice bucket of mice",
		h.Paths.PathItems.GetOrZero("/pet/findByStatus").Get.Responses.Codes.GetOrZero("400").Description)
	assert.Len(t, h.Paths.PathItems.GetOrZero("/pet/findByTags").Get.Tags, 3)

	assert.Len(t, h.Paths.PathItems.GetOrZero("/pet/findByTags").Get.Tags, 3)
	yu := h.Paths.PathItems.GetOrZero("/pet/{petId}").Delete.Security
	assert.Equal(t, "read:abook", yu[len(yu)-1].Requirements.GetOrZero("pizza-and-cake")[0])
	assert.Equal(t, "I am a teapot, filled with love.",
		h.Components.Schemas.GetOrZero("Order").Schema().Properties.GetOrZero("status").Schema().Example)

	assert.Equal(t, "https://pb33f.io",
		h.Components.SecuritySchemes.GetOrZero("petstore_auth").Flows.Implicit.AuthorizationUrl)
}

func TestDocument_Render(t *testing.T) {

	// load an OpenAPI 3 specification from bytes
	petstore, _ := os.ReadFile("test_specs/petstorev3.json")

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
	h.Paths.PathItems.GetOrZero("/pet/findByStatus").Get.OperationId = "findACakeInABakery"
	h.Paths.PathItems.GetOrZero("/pet/findByStatus").
		Get.Responses.Codes.GetOrZero("400").Description = "a nice bucket of mice"
	h.Paths.PathItems.GetOrZero("/pet/findByTags").Get.Tags =
		append(h.Paths.PathItems.GetOrZero("/pet/findByTags").Get.Tags, "gurgle", "giggle")

	h.Paths.PathItems.GetOrZero("/pet/{petId}").Delete.Security = append(h.Paths.PathItems.GetOrZero("/pet/{petId}").Delete.Security,
		&base.SecurityRequirement{Requirements: orderedmap.ToOrderedMap(map[string][]string{
			"pizza-and-cake": {"read:abook", "write:asong"},
		})},
	)

	h.Components.Schemas.GetOrZero("Order").Schema().Properties.GetOrZero("status").Schema().Example = "I am a teapot, filled with love."
	h.Components.SecuritySchemes.GetOrZero("petstore_auth").Flows.Implicit.AuthorizationUrl = "https://pb33f.io"

	bytes, e := doc.Render()
	assert.NoError(t, e)
	assert.NotNil(t, bytes)

	newDoc, docErr := NewDocument(bytes)

	assert.NoError(t, docErr)

	newDocModel, docErrs := newDoc.BuildV3Model()
	assert.Len(t, docErrs, 0)

	h = newDocModel.Model
	assert.Equal(t, "findACakeInABakery", h.Paths.PathItems.GetOrZero("/pet/findByStatus").Get.OperationId)
	assert.Equal(t, "a nice bucket of mice",
		h.Paths.PathItems.GetOrZero("/pet/findByStatus").Get.Responses.Codes.GetOrZero("400").Description)
	assert.Len(t, h.Paths.PathItems.GetOrZero("/pet/findByTags").Get.Tags, 3)

	assert.Len(t, h.Paths.PathItems.GetOrZero("/pet/findByTags").Get.Tags, 3)
	yu := h.Paths.PathItems.GetOrZero("/pet/{petId}").Delete.Security
	assert.Equal(t, "read:abook", yu[len(yu)-1].Requirements.GetOrZero("pizza-and-cake")[0])
	assert.Equal(t, "I am a teapot, filled with love.",
		h.Components.Schemas.GetOrZero("Order").Schema().Properties.GetOrZero("status").Schema().Example)

	assert.Equal(t, "https://pb33f.io",
		h.Components.SecuritySchemes.GetOrZero("petstore_auth").Flows.Implicit.AuthorizationUrl)
}

func TestDocument_RenderWithLargeIndention(t *testing.T) {

	json := `{
      "openapi": "3.0"
}`
	doc, _ := NewDocument([]byte(json))

	doc.BuildV3Model()
	bytes, _ := doc.Render()
	assert.Equal(t, json, string(bytes))
}

func TestDocument_Render_ChangeCheck_Burgershop(t *testing.T) {

	bs, _ := os.ReadFile("test_specs/burgershop.openapi.yaml")
	doc, _ := NewDocument(bs)
	doc.BuildV3Model()

	rend, _ := doc.Render()

	newDoc, _ := NewDocument(rend)

	// compare documents
	compReport, errs := CompareDocuments(doc, newDoc)

	// should noth be nil.
	assert.Nil(t, errs)
	assert.NotNil(t, rend)
	assert.Nil(t, compReport)

}

func TestDocument_RenderAndReload_Swagger(t *testing.T) {
	petstore, _ := os.ReadFile("test_specs/petstorev2.json")
	doc, _ := NewDocument(petstore)
	doc.BuildV2Model()
	doc.BuildV2Model()
	_, _, _, e := doc.RenderAndReload()
	assert.Len(t, e, 1)
	assert.Equal(t, "this method only supports OpenAPI 3 documents, not Swagger", e[0].Error())

}

func TestDocument_Render_Swagger(t *testing.T) {
	petstore, _ := os.ReadFile("test_specs/petstorev2.json")
	doc, _ := NewDocument(petstore)
	doc.BuildV2Model()
	doc.BuildV2Model()
	_, e := doc.Render()
	assert.Error(t, e)
	assert.Equal(t, "this method only supports OpenAPI 3 documents, not Swagger", e.Error())

}

func TestDocument_BuildModelPreBuild(t *testing.T) {
	petstore, _ := os.ReadFile("test_specs/petstorev3.json")
	doc, e := NewDocument(petstore)
	assert.NoError(t, e)
	doc.BuildV3Model()
	doc.BuildV3Model()
	_, _, _, er := doc.RenderAndReload()
	assert.Len(t, er, 0)
}

func TestDocument_AnyDoc(t *testing.T) {
	anything := []byte(`{"chickens": "3.0.0", "burgers": {"title": "hello"}}`)
	_, e := NewDocumentWithTypeCheck(anything, true)
	assert.NoError(t, e)
}

func TestDocument_AnyDocWithConfig(t *testing.T) {
	anything := []byte(`{"chickens": "3.0.0", "burgers": {"title": "hello"}}`)
	_, e := NewDocumentWithConfiguration(anything, &datamodel.DocumentConfiguration{
		BypassDocumentCheck: true,
	})
	assert.NoError(t, e)
}

func TestDocument_BuildModelCircular(t *testing.T) {
	petstore, _ := os.ReadFile("test_specs/circular-tests.yaml")
	doc, _ := NewDocument(petstore)
	m, e := doc.BuildV3Model()
	assert.NotNil(t, m)
	assert.Len(t, e, 3)
}

func TestDocument_BuildModelBad(t *testing.T) {
	petstore, _ := os.ReadFile("test_specs/badref-burgershop.openapi.yaml")
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
	operation := result.Model.Paths.PathItems.GetOrZero("/something").Get

	// print it out.
	fmt.Printf("param1: %s, is reference? %t, original reference %s",
		operation.Parameters[0].Description, operation.GoLow().Parameters.Value[0].IsReference(),
		operation.GoLow().Parameters.Value[0].Reference)
}

func TestDocument_BuildModel_CompareDocsV3_LeftError(t *testing.T) {
	burgerShopOriginal, _ := os.ReadFile("test_specs/badref-burgershop.openapi.yaml")
	burgerShopUpdated, _ := os.ReadFile("test_specs/burgershop.openapi-modified.yaml")
	originalDoc, _ := NewDocument(burgerShopOriginal)
	updatedDoc, _ := NewDocument(burgerShopUpdated)
	changes, errors := CompareDocuments(originalDoc, updatedDoc)
	assert.Len(t, errors, 9)
	assert.Nil(t, changes)
}

func TestDocument_BuildModel_CompareDocsV3_RightError(t *testing.T) {

	burgerShopOriginal, _ := os.ReadFile("test_specs/badref-burgershop.openapi.yaml")
	burgerShopUpdated, _ := os.ReadFile("test_specs/burgershop.openapi-modified.yaml")
	originalDoc, _ := NewDocument(burgerShopOriginal)
	updatedDoc, _ := NewDocument(burgerShopUpdated)
	changes, errors := CompareDocuments(updatedDoc, originalDoc)
	assert.Len(t, errors, 9)
	assert.Nil(t, changes)

}

func TestDocument_BuildModel_CompareDocsV2_Error(t *testing.T) {

	burgerShopOriginal, _ := os.ReadFile("test_specs/petstorev2-badref.json")
	burgerShopUpdated, _ := os.ReadFile("test_specs/petstorev2-badref.json")
	originalDoc, _ := NewDocument(burgerShopOriginal)
	updatedDoc, _ := NewDocument(burgerShopUpdated)
	changes, errors := CompareDocuments(updatedDoc, originalDoc)
	assert.Len(t, errors, 2)
	assert.Nil(t, changes)

}

func TestDocument_BuildModel_CompareDocsV2V3Mix_Error(t *testing.T) {

	burgerShopOriginal, _ := os.ReadFile("test_specs/petstorev2.json")
	burgerShopUpdated, _ := os.ReadFile("test_specs/petstorev3.json")
	originalDoc, _ := NewDocument(burgerShopOriginal)
	updatedDoc, _ := NewDocument(burgerShopUpdated)
	changes, errors := CompareDocuments(updatedDoc, originalDoc)
	assert.Len(t, errors, 1)
	assert.Nil(t, changes)

}

func TestSchemaRefIsFollowed(t *testing.T) {
	petstore, _ := os.ReadFile("test_specs/ref-followed.yaml")

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
	assert.Equal(t, 4, orderedmap.Len(schemas))

	fp := schemas.GetOrZero("FP")
	fbsref := schemas.GetOrZero("FBSRef")

	assert.Equal(t, fp.Schema().Pattern, fbsref.Schema().Pattern)
	assert.Equal(t, fp.Schema().Example, fbsref.Schema().Example)

	byte := schemas.GetOrZero("Byte")
	uint64 := schemas.GetOrZero("UInt64")

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

// disabled for now as the host is timing out
//func TestDocument_RemoteWithoutBaseURL(t *testing.T) {
//
//	// This test will push the index to do try and locate remote references that use relative references
//	spec := `openapi: 3.0.2
//info:
//  title: Test
//  version: 1.0.0
//paths:
//  /test:
//    get:
//      parameters:
//        - $ref: "https://schemas.opengis.net/ogcapi/features/part2/1.0/openapi/ogcapi-features-2.yaml#/components/parameters/crs"`
//
//	config := datamodel.NewOpenDocumentConfiguration()
//
//	doc, err := NewDocumentWithConfiguration([]byte(spec), config)
//	if err != nil {
//		panic(err)
//	}
//
//	result, errs := doc.BuildV3Model()
//	if len(errs) > 0 {
//		panic(errs)
//	}
//
//	assert.Equal(t, "crs", result.Model.Paths.PathItems.GetOrZero("/test").Get.Parameters[0].Name)
//}

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
                        type: object
`

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

	assert.Len(t, rend, len(d))
}

func TestDocument_OperationsAsRefs(t *testing.T) {

	ae := `operationId: thisIsAnOperationId
summary: a test thing
description: this is a test, that does a test.`

	_ = os.WriteFile("test-operation.yaml", []byte(ae), 0644)
	defer os.Remove("test-operation.yaml")

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

func TestDocument_InputAsJSON(t *testing.T) {

	var d = `{
  "openapi": "3.1",
  "paths": {
    "/an/operation": {
      "get": {
        "operationId": "thisIsAnOperationId"
      }
    }
  }
}`

	doc, err := NewDocumentWithConfiguration([]byte(d), datamodel.NewOpenDocumentConfiguration())
	if err != nil {
		panic(err)
	}

	_, _ = doc.BuildV3Model()

	// render the document.
	rend, _, _, _ := doc.RenderAndReload()

	assert.Equal(t, d, strings.TrimSpace(string(rend)))
}

func TestDocument_InputAsJSON_LargeIndent(t *testing.T) {

	var d = `{
    "openapi": "3.1",
    "paths": {
        "/an/operation": {
            "get": {
                "operationId": "thisIsAnOperationId"
            }
        }
    }
}`

	doc, err := NewDocumentWithConfiguration([]byte(d), datamodel.NewOpenDocumentConfiguration())
	if err != nil {
		panic(err)
	}

	_, _ = doc.BuildV3Model()

	// render the document.
	rend, _, _, _ := doc.RenderAndReload()

	assert.Equal(t, d, strings.TrimSpace(string(rend)))
}

func TestDocument_RenderWithIndention(t *testing.T) {

	spec := `openapi: "3.1.0"
info:
      title: Test
      version: 1.0.0
paths:
      /test:
            get:
                  operationId: 'test'`

	config := datamodel.NewOpenDocumentConfiguration()

	doc, err := NewDocumentWithConfiguration([]byte(spec), config)
	if err != nil {
		panic(err)
	}

	_, _ = doc.BuildV3Model()

	rend, _, _, _ := doc.RenderAndReload()

	assert.Equal(t, spec, strings.TrimSpace(string(rend)))
}

func TestDocument_IgnorePolymorphicCircularReferences(t *testing.T) {

	var d = `openapi: 3.1.0
components:
  schemas:
    ProductCategory:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "object"
          anyOf:
            - $ref: "#/components/schemas/ProductCategory"
          description: "Array of sub-categories in the same format."
      required:
        - "name"
        - "children"`

	config := datamodel.NewClosedDocumentConfiguration()
	config.IgnorePolymorphicCircularReferences = true

	doc, err := NewDocumentWithConfiguration([]byte(d), config)
	if err != nil {
		panic(err)
	}

	m, errs := doc.BuildV3Model()

	assert.Len(t, errs, 0)
	assert.Len(t, m.Index.GetCircularReferences(), 0)

}

func TestDocument_IgnoreArrayCircularReferences(t *testing.T) {

	var d = `openapi: 3.1.0
components:
  schemas:
    ProductCategory:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "array"
          items:
            $ref: "#/components/schemas/ProductCategory"
          description: "Array of sub-categories in the same format."
      required:
        - "name"
        - "children"`

	config := datamodel.NewClosedDocumentConfiguration()
	config.IgnoreArrayCircularReferences = true

	doc, err := NewDocumentWithConfiguration([]byte(d), config)
	if err != nil {
		panic(err)
	}

	m, errs := doc.BuildV3Model()

	assert.Len(t, errs, 0)
	assert.Len(t, m.Index.GetCircularReferences(), 0)

}

// Ensure document ordering is preserved after building and loading.
func TestDocument_Render_PreserveOrder(t *testing.T) {
	t.Run("Paths", func(t *testing.T) {
		const pathCount = 100
		doc, err := NewDocument([]byte(`openapi: 3.1.0`))
		require.NoError(t, err)
		model, errs := doc.BuildV3Model()
		require.Empty(t, errs)
		pathItems := orderedmap.New[string, *v3high.PathItem]()
		model.Model.Paths = &v3high.Paths{
			PathItems: pathItems,
		}
		for i := 0; i < pathCount; i++ {
			pathItem := &v3high.PathItem{
				Get: &v3high.Operation{
					Parameters: make([]*v3high.Parameter, 0),
				},
			}
			pathName := fmt.Sprintf("/foobar/%d", i)
			pathItems.Set(pathName, pathItem)
		}

		checkOrder := func(t *testing.T, doc Document) {
			model, errs := doc.BuildV3Model()
			require.Empty(t, errs)
			pathItems := model.Model.Paths.PathItems
			require.Equal(t, pathCount, orderedmap.Len(pathItems))

			var i int
			for pair := orderedmap.First(model.Model.Paths.PathItems); pair != nil; pair = pair.Next() {
				pathName := fmt.Sprintf("/foobar/%d", i)
				assert.Equal(t, pathName, pair.Key())
				i++
			}
			assert.Equal(t, pathCount, i)
		}

		checkOrder(t, doc)
		yamlBytes, doc, _, errs := doc.RenderAndReload()
		require.Empty(t, errs)

		t.Run("Unmarshalled YAML ordering", func(t *testing.T) {
			// Reload YAML into new Document, verify ordering.
			doc2, err := NewDocument(yamlBytes)
			require.NoError(t, err)
			checkOrder(t, doc2)
		})

		t.Run("Reloaded document ordering", func(t *testing.T) {
			// Verify ordering of reloaded document after call to RenderAndReload().
			checkOrder(t, doc)
		})
	})
}
