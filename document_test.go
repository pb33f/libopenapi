// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT
package libopenapi

import (
	"bytes"
	stdContext "context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"github.com/pb33f/libopenapi/what-changed/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestLoadDocument_Simple_V2(t *testing.T) {
	yml := `swagger: 2.0.1`
	doc, err := NewDocument([]byte(yml))
	assert.NoError(t, err)
	assert.Equal(t, "2.0.1", doc.GetVersion())

	v2Doc, docErr := doc.BuildV2Model()
	assert.NoError(t, docErr)
	assert.NotNil(t, v2Doc)
	assert.NotNil(t, doc.GetSpecInfo())

	fmt.Print()
}

func TestLoadDocument_Simple_V2_Error(t *testing.T) {
	yml := `swagger: 2.0`
	doc, err := NewDocument([]byte(yml))
	assert.NoError(t, err)

	v2Doc, docErr := doc.BuildV3Model()
	assert.Len(t, utils.UnwrapErrors(docErr), 1)
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
	assert.Len(t, utils.UnwrapErrors(docErr), 3)
	assert.Nil(t, v2Doc)
}

func TestLoadDocument_WrongDoc(t *testing.T) {
	yml := `IAmNotAnOpenAPI: 3.1.0`
	doc, err := NewDocument([]byte(yml))
	assert.Error(t, err)
	assert.Nil(t, doc)
}

func TestLoadDocument_Simple_V3_Error(t *testing.T) {
	yml := `openapi: 3.0.1`
	doc, err := NewDocument([]byte(yml))
	assert.NoError(t, err)

	v2Doc, docErr := doc.BuildV2Model()
	assert.Len(t, utils.UnwrapErrors(docErr), 1)
	assert.Nil(t, v2Doc)
}

func TestLoadDocument_Error_V2NoSpec(t *testing.T) {
	doc := new(document) // not how this should be instantiated.
	_, err := doc.BuildV2Model()
	assert.Len(t, utils.UnwrapErrors(err), 1)
}

func TestLoadDocument_Error_V3NoSpec(t *testing.T) {
	doc := new(document) // not how this should be instantiated.
	_, err := doc.BuildV3Model()
	assert.Len(t, utils.UnwrapErrors(err), 1)
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
	assert.Len(t, utils.UnwrapErrors(docErr), 0)
	assert.NotNil(t, v3Doc)
}

func TestLoadDocument_Simple_V3_Error_BadSpec_BuildModel(t *testing.T) {
	yml := `openapi: 3.0
paths:
  "/some":
    $ref: bork`
	doc, err := NewDocument([]byte(yml))
	assert.NoError(t, err)

	doc.BuildV3Model()
	rolo := doc.GetRolodex()
	assert.Len(t, rolo.GetCaughtErrors(), 1)
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

func TestDocument_RoundTrip_JSON(t *testing.T) {
	bs, _ := os.ReadFile("test_specs/roundtrip.json")

	doc, err := NewDocument(bs)
	require.NoError(t, err)

	m, errs := doc.BuildV3Model()
	require.Empty(t, errs)

	out, _ := m.Model.RenderJSON("  ")

	// windows has to be different, it does not add carriage returns.
	if runtime.GOOS != "windows" {
		assert.Equal(t, string(bs), string(out))
	}
}

func TestDocument_RoundTrip_YAML(t *testing.T) {
	bs, _ := os.ReadFile("test_specs/roundtrip.yaml")

	doc, err := NewDocument(bs)
	require.NoError(t, err)

	_, errs := doc.BuildV3Model()
	require.Empty(t, errs)

	out, err := doc.Render()
	require.NoError(t, err)
	if runtime.GOOS != "windows" {
		assert.Equal(t, string(bs), string(out))
	}
}

func TestDocument_RoundTrip_YAML_To_JSON(t *testing.T) {
	y, _ := os.ReadFile("test_specs/roundtrip.yaml")
	j, _ := os.ReadFile("test_specs/roundtrip.json")

	doc, err := NewDocument(y)
	require.NoError(t, err)

	m, errs := doc.BuildV3Model()
	require.Empty(t, errs)

	out, _ := m.Model.RenderJSON("  ")
	require.NoError(t, err)
	if runtime.GOOS != "windows" {
		assert.Equal(t, string(j), string(out))
	}
}

func TestDocument_RenderAndReload_ChangeCheck_Burgershop(t *testing.T) {
	bs, _ := os.ReadFile("test_specs/burgershop.openapi.yaml")
	doc, _ := NewDocument(bs)
	doc.BuildV3Model()

	rend, newDoc, _, _ := doc.RenderAndReload()

	// compare documents
	compReport, errs := CompareDocuments(doc, newDoc)

	// should not be nil.
	assert.Nil(t, errs)
	assert.Nil(t, errs)
	assert.NotNil(t, rend)
	assert.Nil(t, compReport)
}

func TestDocument_RenderAndReload_ChangeCheck_Stripe(t *testing.T) {
	bs, _ := os.ReadFile("test_specs/stripe.yaml")
	doc, _ := NewDocumentWithConfiguration(bs, &datamodel.DocumentConfiguration{})
	doc.BuildV3Model()

	_, newDoc, _, _ := doc.RenderAndReload()

	ctx, cancel := stdContext.WithTimeout(stdContext.Background(), 10*time.Second)
	done := make(chan struct{})
	defer cancel()
	defer close(done)

	go func() {
		// compare documents
		compReport, errs := CompareDocuments(doc, newDoc)

		// get flat list of changes.
		flatChanges := compReport.GetAllChanges()

		// remove everything that is a description change
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
		assert.Equal(t, 9, tc)

		// there should be no other changes besides descriptions.
		assert.Equal(t, 0, len(filtered))
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatalf("timed out waiting for changes to be compared: %v", ctx.Err())
	}
}

func TestDocument_ResolveStripe(t *testing.T) {
	bs, _ := os.ReadFile("test_specs/stripe.yaml")
	docConfig := datamodel.NewDocumentConfiguration()
	docConfig.SkipCircularReferenceCheck = true
	docConfig.BasePath = "."
	docConfig.AllowRemoteReferences = true
	docConfig.AllowFileReferences = true
	doc, _ := NewDocumentWithConfiguration(bs, docConfig)
	model, _ := doc.BuildV3Model()

	rolo := model.Index.GetRolodex()
	rolo.Resolve()

	assert.Equal(t, 1, len(model.Index.GetRolodex().GetCaughtErrors()))
}

func TestDocument_RenderAndReload_ChangeCheck_Asana(t *testing.T) {
	bs, _ := os.ReadFile("test_specs/asana.yaml")
	doc, _ := NewDocument(bs)
	doc.BuildV3Model()

	dat, newDoc, _, _ := doc.RenderAndReload()
	assert.NotNil(t, dat)
	if runtime.GOOS != "windows" {
		assert.Equal(t, string(bs), string(dat))
	}
	// compare documents
	compReport, errs := CompareDocuments(doc, newDoc)

	// get flat list of changes.
	flatChanges := compReport.GetAllChanges()

	assert.Nil(t, errs)
	tc := compReport.TotalChanges()
	assert.Equal(t, 0, tc)

	// there are some properties re-rendered that trigger changes.
	assert.Equal(t, 0, len(flatChanges))
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
	h.Paths.PathItems.GetOrZero("/pet/findByTags").Get.Tags = append(h.Paths.PathItems.GetOrZero("/pet/findByTags").Get.Tags, "gurgle", "giggle")

	h.Paths.PathItems.GetOrZero("/pet/{petId}").Delete.Security = append(h.Paths.PathItems.GetOrZero("/pet/{petId}").Delete.Security,
		&base.SecurityRequirement{Requirements: orderedmap.ToOrderedMap(map[string][]string{
			"pizza-and-cake": {"read:abook", "write:asong"},
		})},
	)

	h.Components.Schemas.GetOrZero("Order").Schema().Properties.GetOrZero("status").Schema().Example = utils.CreateStringNode("I am a teapot, filled with love.")
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

	var example string
	_ = h.Components.Schemas.GetOrZero("Order").Schema().Properties.GetOrZero("status").Schema().Example.Decode(&example)
	assert.Equal(t, "I am a teapot, filled with love.", example)

	assert.Equal(t, "https://pb33f.io",
		h.Components.SecuritySchemes.GetOrZero("petstore_auth").Flows.Implicit.AuthorizationUrl)
}

func TestDocument_RenderAndReload_WithErrors(t *testing.T) {
	// load an OpenAPI 3 specification from bytes
	petstore, _ := os.ReadFile("test_specs/petstorev3.json")

	// create config
	config := datamodel.NewDocumentConfiguration()

	// create a new document from specification bytes
	doc, err := NewDocumentWithConfiguration(petstore, config)

	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	// because we know this is a v3 spec, we can build a ready to go model from it.
	m, _ := doc.BuildV3Model()

	// Remove a schema to make the model invalid
	_, present := m.Model.Components.Schemas.Delete("Pet")
	assert.True(t, present, "expected schema Pet to exist")

	_, _, _, errors := doc.RenderAndReload()
	assert.Len(t, utils.UnwrapErrors(errors), 2)
	assert.Contains(t, errors.Error(), "component `#/components/schemas/Pet` does not exist in the specification")
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
	h.Paths.PathItems.GetOrZero("/pet/findByTags").Get.Tags = append(h.Paths.PathItems.GetOrZero("/pet/findByTags").Get.Tags, "gurgle", "giggle")

	h.Paths.PathItems.GetOrZero("/pet/{petId}").Delete.Security = append(h.Paths.PathItems.GetOrZero("/pet/{petId}").Delete.Security,
		&base.SecurityRequirement{Requirements: orderedmap.ToOrderedMap(map[string][]string{
			"pizza-and-cake": {"read:abook", "write:asong"},
		})},
	)

	h.Components.Schemas.GetOrZero("Order").Schema().Properties.GetOrZero("status").Schema().Example = utils.CreateStringNode("I am a teapot, filled with love.")
	h.Components.SecuritySchemes.GetOrZero("petstore_auth").Flows.Implicit.AuthorizationUrl = "https://pb33f.io"

	bytes, e := doc.Render()
	assert.NoError(t, e)
	assert.NotNil(t, bytes)

	newDoc, docErr := NewDocument(bytes)

	assert.NoError(t, docErr)

	newDocModel, docErrs := newDoc.BuildV3Model()
	assert.NoError(t, docErrs)

	h = newDocModel.Model
	assert.Equal(t, "findACakeInABakery", h.Paths.PathItems.GetOrZero("/pet/findByStatus").Get.OperationId)
	assert.Equal(t, "a nice bucket of mice",
		h.Paths.PathItems.GetOrZero("/pet/findByStatus").Get.Responses.Codes.GetOrZero("400").Description)
	assert.Len(t, h.Paths.PathItems.GetOrZero("/pet/findByTags").Get.Tags, 3)

	assert.Len(t, h.Paths.PathItems.GetOrZero("/pet/findByTags").Get.Tags, 3)
	yu := h.Paths.PathItems.GetOrZero("/pet/{petId}").Delete.Security
	assert.Equal(t, "read:abook", yu[len(yu)-1].Requirements.GetOrZero("pizza-and-cake")[0])

	var example string
	_ = h.Components.Schemas.GetOrZero("Order").Schema().Properties.GetOrZero("status").Schema().Example.Decode(&example)
	assert.Equal(t, "I am a teapot, filled with love.", example)

	assert.Equal(t, "https://pb33f.io",
		h.Components.SecuritySchemes.GetOrZero("petstore_auth").Flows.Implicit.AuthorizationUrl)
}

func TestDocument_Render_Missing_Model_Error(t *testing.T) {
	// load an OpenAPI 3 specification from bytes
	petstore, _ := os.ReadFile("test_specs/petstorev3.json")

	// create a new document from specification bytes
	doc, err := NewDocument(petstore)
	assert.NoError(t, err)

	// instead of building the model, we will render the doc immediately - therefore no underlying v3 model exists on render
	_, e := doc.Render()
	assert.Error(t, e)
	assert.Equal(t, "unable to render, no openapi model has been built for the document", e.Error())
}

func TestDocument_Render_Missing_Info_Error(t *testing.T) {
	doc := &document{
		// set the highOpenAPI3Model to a non-nil model to mock an existing model
		highOpenAPI3Model: &DocumentModel[v3high.Document]{},
		// do not set the info property
		info: nil,
	}
	_, e := doc.Render()
	assert.Error(t, e)
	assert.Equal(t, "unable to render, no specification has been loaded", e.Error())
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

	// should not be nil.
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
	assert.Error(t, e)
	assert.Equal(t, "this method only supports OpenAPI 3 documents, not Swagger", e.Error())
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
	assert.Len(t, utils.UnwrapErrors(er), 0)
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
	doc.BuildV3Model()
	assert.Len(t, doc.GetRolodex().GetCaughtErrors(), 3)
}

func TestDocument_BuildModelBad(t *testing.T) {
	petstore, _ := os.ReadFile("test_specs/badref-burgershop.openapi.yaml")
	doc, _ := NewDocument(petstore)
	doc.BuildV3Model()
	assert.Len(t, doc.GetRolodex().GetCaughtErrors(), 6)
}

func TestDocument_Serialize_JSON_Modified(t *testing.T) {
	json := `{ "openapi": "3.0",
 "info": {
   "title": "The magic API"
 }
}
`
	jsonModified := `{"info":{"title":"The magic API - but now, altered!"},"openapi":"3.0"}`
	doc, err := NewDocument([]byte(json))
	assert.NoError(t, err)

	v3Doc, _ := doc.BuildV3Model()

	// eventually this will be encapsulated up high.
	// mutation does not replace low model, eventually pointers will be used.
	newTitle := v3Doc.Model.Info.GoLow().Title.Mutate("The magic API - but now, altered!")
	v3Doc.Model.Info.GoLow().Title = newTitle

	assert.Equal(t, "The magic API - but now, altered!", v3Doc.Model.Info.GoLow().Title.GetValue())

	serial, err := doc.Serialize()
	assert.NoError(t, err)
	assert.Equal(t, jsonModified, string(serial))
}

func TestExtractReference(t *testing.T) {
	data := `
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
	assert.NoError(t, errs)

	// extract operation.
	operation := result.Model.Paths.PathItems.GetOrZero("/something").Get

	// print it out.
	fmt.Printf("param1: %s, is reference? %t, original reference %s",
		operation.Parameters[0].Description, operation.GoLow().Parameters.Value[0].IsReference(),
		operation.GoLow().Parameters.Value[0].GetReference())
}

func TestDocument_BuildModel_CompareDocsV3_LeftError(t *testing.T) {
	burgerShopOriginal, _ := os.ReadFile("test_specs/badref-burgershop.openapi.yaml")
	burgerShopUpdated, _ := os.ReadFile("test_specs/burgershop.openapi-modified.yaml")
	originalDoc, _ := NewDocument(burgerShopOriginal)
	updatedDoc, _ := NewDocument(burgerShopUpdated)
	changes, errors := CompareDocuments(originalDoc, updatedDoc)
	assert.Error(t, errors)
	assert.Nil(t, changes)
}

func TestDocument_BuildModel_CompareDocsV3_RightError(t *testing.T) {
	burgerShopOriginal, _ := os.ReadFile("test_specs/badref-burgershop.openapi.yaml")
	burgerShopUpdated, _ := os.ReadFile("test_specs/burgershop.openapi-modified.yaml")
	originalDoc, _ := NewDocument(burgerShopOriginal)
	updatedDoc, _ := NewDocument(burgerShopUpdated)
	changes, errors := CompareDocuments(updatedDoc, originalDoc)
	assert.Error(t, errors)
	assert.Nil(t, changes)
}

func TestDocument_BuildModel_CompareDocsV2_Error(t *testing.T) {
	burgerShopOriginal, _ := os.ReadFile("test_specs/petstorev2-badref.json")
	burgerShopUpdated, _ := os.ReadFile("test_specs/petstorev2-badref.json")
	originalDoc, _ := NewDocument(burgerShopOriginal)
	updatedDoc, _ := NewDocument(burgerShopUpdated)
	changes, errors := CompareDocuments(updatedDoc, originalDoc)
	assert.Error(t, errors)
	assert.Nil(t, changes)
}

func TestDocument_BuildModel_CompareDocsV2V3Mix_Error(t *testing.T) {
	burgerShopOriginal, _ := os.ReadFile("test_specs/petstorev2.json")
	burgerShopUpdated, _ := os.ReadFile("test_specs/petstorev3.json")
	originalDoc, _ := NewDocument(burgerShopOriginal)
	updatedDoc, _ := NewDocument(burgerShopUpdated)
	changes, errors := CompareDocuments(updatedDoc, originalDoc)
	assert.Error(t, errors)
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
	if errors != nil {
		fmt.Printf("error: %e\n", errors)
		panic(fmt.Sprintf("cannot create v3 model from document: %e", errors))
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
	d := `openapi: "3.1"
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
	assert.NoError(t, errs)

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
//	config := datamodel.NewDocumentConfiguration()
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
	d := `openapi: "3.1"
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
	assert.NoError(t, errs)

	// render the document.
	rend, _ := result.Model.Render()

	assert.Len(t, rend, len(d))
}

func TestDocument_OperationsAsRefs(t *testing.T) {
	ae := `operationId: thisIsAnOperationId
summary: a test thing
description: this is a test, that does a test.`

	_ = os.WriteFile("test-operation.yaml", []byte(ae), 0o644)
	defer os.Remove("test-operation.yaml")

	d := `openapi: "3.1"
paths:
    /an/operation:
        get:
            $ref: test-operation.yaml`

	cf := datamodel.NewDocumentConfiguration()
	cf.BasePath = "."
	cf.FileFilter = []string{"test-operation.yaml"}

	doc, err := NewDocumentWithConfiguration([]byte(d), cf)
	if err != nil {
		panic(err)
	}

	assert.NotNil(t, doc.GetConfiguration())
	assert.Equal(t, doc.GetConfiguration(), cf)

	result, errs := doc.BuildV3Model()
	assert.NoError(t, errs)

	// render the document.
	rend, _ := result.Model.Render()

	assert.Equal(t, d, strings.TrimSpace(string(rend)))
}

func TestDocument_InputAsJSON(t *testing.T) {
	d := `{
  "openapi": "3.1",
  "paths": {
    "/an/operation": {
      "get": {
        "operationId": "thisIsAnOperationId"
      }
    }
  }
}`

	doc, err := NewDocumentWithConfiguration([]byte(d), datamodel.NewDocumentConfiguration())
	if err != nil {
		panic(err)
	}

	_, _ = doc.BuildV3Model()

	// render the document.
	rend, _, _, _ := doc.RenderAndReload()

	assert.Equal(t, d, strings.TrimSpace(string(rend)))
}

func TestDocument_InputAsJSON_LargeIndent(t *testing.T) {
	d := `{
    "openapi": "3.1",
    "paths": {
        "/an/operation": {
            "get": {
                "operationId": "thisIsAnOperationId"
            }
        }
    }
}`

	doc, err := NewDocumentWithConfiguration([]byte(d), datamodel.NewDocumentConfiguration())
	if err != nil {
		panic(err)
	}

	_, _ = doc.BuildV3Model()

	// render the document.
	rend, _, _, _ := doc.RenderAndReload()

	assert.Equal(t, d, strings.TrimSpace(string(rend)))
}

// TestDocument_AppendParameterAfterRef tests the bug fix for appending new parameters
// to an operation that has $ref parameters. Before the fix, if a $ref was the last
// parameter in the original spec, any appended parameters would be silently dropped
// during rendering because the 'skip' flag wasn't reset between slice iterations.
func TestDocument_AppendParameterAfterRef(t *testing.T) {
	spec := `openapi: "3.1.0"
info:
    title: Test
    version: 1.0.0
paths:
    /test/{id}:
        get:
            parameters:
                - $ref: '#/components/parameters/IdParam'
            responses:
                "200":
                    description: ok
components:
    parameters:
        IdParam:
            name: id
            in: path
            required: true
            schema:
                type: string`

	doc, err := NewDocument([]byte(spec))
	if err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	result, errs := doc.BuildV3Model()
	assert.NoError(t, errs)

	// Get the operation and append a new parameter
	pathItem := result.Model.Paths.PathItems.GetOrZero("/test/{id}")
	assert.NotNil(t, pathItem)
	assert.NotNil(t, pathItem.Get)

	// Verify we have the original $ref parameter
	assert.Len(t, pathItem.Get.Parameters, 1)

	// Append a new parameter (simulating what the user does)
	newParam := &v3high.Parameter{
		Name: "Host",
		In:   "header",
	}
	pathItem.Get.Parameters = append(pathItem.Get.Parameters, newParam)

	// Render and reload
	_, newDoc, _, err := doc.RenderAndReload()
	assert.NoError(t, err)

	// Build the new model
	newResult, errs := newDoc.BuildV3Model()
	assert.NoError(t, errs)

	// Check that the new parameter is present
	newPathItem := newResult.Model.Paths.PathItems.GetOrZero("/test/{id}")
	assert.NotNil(t, newPathItem)
	assert.NotNil(t, newPathItem.Get)

	// This was the bug: the new "Host" parameter was being dropped
	assert.Len(t, newPathItem.Get.Parameters, 2, "Expected 2 parameters (original $ref + appended Host)")

	// Verify the Host parameter is present
	foundHost := false
	for _, p := range newPathItem.Get.Parameters {
		if p.Name == "Host" && p.In == "header" {
			foundHost = true
			break
		}
	}
	assert.True(t, foundHost, "Expected to find the appended 'Host' header parameter")
}

// TestDocument_AppendParameterBeforeRef tests that appending works when $ref is not last
func TestDocument_AppendParameterBeforeRef(t *testing.T) {
	spec := `openapi: "3.1.0"
info:
    title: Test
    version: 1.0.0
paths:
    /test/{id}:
        get:
            parameters:
                - name: existing
                  in: query
                  schema:
                      type: string
                - $ref: '#/components/parameters/IdParam'
            responses:
                "200":
                    description: ok
components:
    parameters:
        IdParam:
            name: id
            in: path
            required: true
            schema:
                type: string`

	doc, err := NewDocument([]byte(spec))
	if err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	result, errs := doc.BuildV3Model()
	assert.NoError(t, errs)

	// Get the operation and append a new parameter
	pathItem := result.Model.Paths.PathItems.GetOrZero("/test/{id}")

	// Append a new parameter
	newParam := &v3high.Parameter{
		Name: "Host",
		In:   "header",
	}
	pathItem.Get.Parameters = append(pathItem.Get.Parameters, newParam)

	// Render and reload
	_, newDoc, _, err := doc.RenderAndReload()
	assert.NoError(t, err)

	newResult, errs := newDoc.BuildV3Model()
	assert.NoError(t, errs)

	newPathItem := newResult.Model.Paths.PathItems.GetOrZero("/test/{id}")

	// Should have 3 parameters: existing + $ref + Host
	assert.Len(t, newPathItem.Get.Parameters, 3, "Expected 3 parameters")

	foundHost := false
	for _, p := range newPathItem.Get.Parameters {
		if p.Name == "Host" && p.In == "header" {
			foundHost = true
			break
		}
	}
	assert.True(t, foundHost, "Expected to find the appended 'Host' header parameter")
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

	config := datamodel.NewDocumentConfiguration()

	doc, err := NewDocumentWithConfiguration([]byte(spec), config)
	if err != nil {
		panic(err)
	}

	_, _ = doc.BuildV3Model()

	rend, _, _, _ := doc.RenderAndReload()

	assert.Equal(t, spec, strings.TrimSpace(string(rend)))
}

func TestDocument_IgnorePolymorphicCircularReferences(t *testing.T) {
	d := `openapi: 3.1.0
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

	config := datamodel.NewDocumentConfiguration()
	config.IgnorePolymorphicCircularReferences = true

	doc, err := NewDocumentWithConfiguration([]byte(d), config)
	if err != nil {
		panic(err)
	}

	m, errs := doc.BuildV3Model()

	assert.NoError(t, errs)
	assert.Len(t, m.Index.GetCircularReferences(), 0)
	assert.Len(t, m.Index.GetResolver().GetIgnoredCircularPolyReferences(), 1)
}

func TestDocument_IgnoreArrayCircularReferences(t *testing.T) {
	d := `openapi: 3.1.0
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

	config := datamodel.NewDocumentConfiguration()
	config.IgnoreArrayCircularReferences = true

	doc, err := NewDocumentWithConfiguration([]byte(d), config)
	if err != nil {
		panic(err)
	}

	m, errs := doc.BuildV3Model()

	assert.NoError(t, errs)
	assert.Len(t, m.Index.GetCircularReferences(), 0)
	assert.Len(t, m.Index.GetResolver().GetIgnoredCircularArrayReferences(), 1)
}

func TestDocument_TestMixedReferenceOrigin(t *testing.T) {
	bs, _ := os.ReadFile("test_specs/mixedref-burgershop.openapi.yaml")

	config := datamodel.NewDocumentConfiguration()
	config.AllowRemoteReferences = true
	config.AllowFileReferences = true
	config.SkipCircularReferenceCheck = true
	config.BasePath = "test_specs"

	config.Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	doc, _ := NewDocumentWithConfiguration(bs, config)
	m, _ := doc.BuildV3Model()

	// extract something that can only exist after being located by the rolodex.
	mediaType := m.Model.Paths.PathItems.GetOrZero("/burgers/{burgerId}/dressings").
		Get.Responses.Codes.GetOrZero("200").Content.GetOrZero("application/json").Schema.Schema().Items

	items := mediaType.A.Schema()

	origin := items.ParentProxy.GetReferenceOrigin()
	assert.NotNil(t, origin)
	sep := string(os.PathSeparator)
	assert.True(t, strings.HasSuffix(origin.AbsoluteLocation, "test_specs"+sep+"burgershop.openapi.yaml"))
}

func BenchmarkReferenceOrigin(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {

		bs, _ := os.ReadFile("test_specs/mixedref-burgershop.openapi.yaml")

		config := datamodel.NewDocumentConfiguration()
		config.AllowRemoteReferences = true
		config.AllowFileReferences = true
		config.BasePath = "test_specs"
		config.Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

		doc, _ := NewDocumentWithConfiguration(bs, config)
		m, _ := doc.BuildV3Model()

		// extract something that can only exist after being located by the rolodex.
		mediaType := m.Model.Paths.PathItems.GetOrZero("/burgers/{burgerId}/dressings").
			Get.Responses.Codes.GetOrZero("200").Content.GetOrZero("application/json").Schema.Schema().Items

		items := mediaType.A.Schema()

		origin := items.ParentProxy.GetReferenceOrigin()
		assert.NotNil(b, origin)
		assert.True(b, strings.HasSuffix(origin.AbsoluteLocation, "test_specs/burgershop.openapi.yaml"))
	}
}

// Ensure document ordering is preserved after building, rendering, and reloading.
func TestDocument_Render_PreserveOrder(t *testing.T) {
	t.Run("Paths", func(t *testing.T) {
		const itemCount = 100
		doc, err := NewDocument([]byte(`openapi: 3.1.0`))
		require.NoError(t, err)
		model, errs := doc.BuildV3Model()
		require.Empty(t, errs)
		pathItems := orderedmap.New[string, *v3high.PathItem]()
		model.Model.Paths = &v3high.Paths{
			PathItems: pathItems,
		}
		for i := 0; i < itemCount; i++ {
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
			require.Equal(t, itemCount, orderedmap.Len(pathItems))

			var i int
			for path := range model.Model.Paths.PathItems.KeysFromOldest() {
				pathName := fmt.Sprintf("/foobar/%d", i)
				assert.Equal(t, pathName, path)
				i++
			}
			assert.Equal(t, itemCount, i)
		}

		t.Run("Check order before rendering", func(t *testing.T) {
			checkOrder(t, doc)
		})

		yamlBytes, doc, _, errs := doc.RenderAndReload()
		require.Empty(t, errs)

		// Reload YAML into new Document, verify ordering.
		t.Run("Unmarshalled YAML ordering", func(t *testing.T) {
			doc2, err := NewDocument(yamlBytes)
			require.NoError(t, err)
			checkOrder(t, doc2)
		})

		// Verify ordering of reloaded document after call to RenderAndReload().
		t.Run("Reloaded document ordering", func(t *testing.T) {
			checkOrder(t, doc)
		})
	})

	t.Run("Responses", func(t *testing.T) {
		t.Run("Codes", func(t *testing.T) {
			const itemCount = 100
			doc, err := NewDocument([]byte(`openapi: 3.1.0`))
			require.NoError(t, err)
			model, errs := doc.BuildV3Model()
			require.Empty(t, errs)
			pathItems := orderedmap.New[string, *v3high.PathItem]()
			model.Model.Paths = &v3high.Paths{
				PathItems: pathItems,
			}
			pathItem := &v3high.PathItem{
				Get: &v3high.Operation{
					Parameters: make([]*v3high.Parameter, 0),
				},
			}
			pathName := "/foobar"
			pathItems.Set(pathName, pathItem)
			responses := &v3high.Responses{
				Codes: orderedmap.New[string, *v3high.Response](),
			}
			pathItem.Get.Responses = responses

			for i := 0; i < itemCount; i++ {
				code := strconv.Itoa(200 + i)
				resp := &v3high.Response{}
				responses.Codes.Set(code, resp)
			}

			checkOrder := func(t *testing.T, doc Document) {
				model, errs := doc.BuildV3Model()
				require.Empty(t, errs)
				pathItem := model.Model.Paths.PathItems.GetOrZero(pathName)
				responses := pathItem.Get.Responses

				var i int
				for code := range responses.Codes.KeysFromOldest() {
					expectedCode := strconv.Itoa(200 + i)
					assert.Equal(t, expectedCode, code)
					i++
				}
				assert.Equal(t, itemCount, i)
			}

			t.Run("Check order before rendering", func(t *testing.T) {
				checkOrder(t, doc)
			})

			yamlBytes, doc, _, errs := doc.RenderAndReload()
			require.Empty(t, errs)

			// Reload YAML into new Document, verify ordering.
			t.Run("Unmarshalled YAML ordering", func(t *testing.T) {
				doc2, err := NewDocument(yamlBytes)
				require.NoError(t, err)
				checkOrder(t, doc2)
			})

			// Verify ordering of reloaded document after call to RenderAndReload().
			t.Run("Reloaded document ordering", func(t *testing.T) {
				checkOrder(t, doc)
			})
		})

		t.Run("Examples", func(t *testing.T) {
			const itemCount = 3
			doc, err := NewDocument([]byte(`openapi: 3.1.0`))
			require.NoError(t, err)
			model, errs := doc.BuildV3Model()
			require.Empty(t, errs)
			pathItems := orderedmap.New[string, *v3high.PathItem]()
			model.Model.Paths = &v3high.Paths{
				PathItems: pathItems,
			}
			pathItem := &v3high.PathItem{
				Get: &v3high.Operation{
					Parameters: make([]*v3high.Parameter, 0),
				},
			}
			const pathName = "/foobar"
			pathItems.Set(pathName, pathItem)
			responses := &v3high.Responses{
				Codes: orderedmap.New[string, *v3high.Response](),
			}
			pathItem.Get.Responses = responses
			response := &v3high.Response{
				Content: orderedmap.New[string, *v3high.MediaType](),
			}
			const respCode = "200"
			responses.Codes.Set(respCode, response)
			const mediaType = "application/json"
			mediaTypeResp := &v3high.MediaType{
				Examples: orderedmap.New[string, *base.Example](),
			}
			response.Content.Set(mediaType, mediaTypeResp)
			type testExampleDomain struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Type string `json:"type"`
			}
			type testExampleDetails struct {
				Message string            `json:"message"`
				Domain  testExampleDomain `json:"domain"`
			}

			for i := 0; i < itemCount; i++ {
				example := &base.Example{
					Summary:     fmt.Sprintf("Summary example %d", i),
					Description: "Description example",
					Value: utils.CreateYamlNode(testExampleDetails{
						Message: "Foobar message",
						Domain: testExampleDomain{
							ID:   "12345",
							Name: "example.com",
							Type: "Foobar type",
						},
					}),
				}
				exampleName := fmt.Sprintf("FoobarExample%d", i)
				mediaTypeResp.Examples.Set(exampleName, example)
			}

			checkOrder := func(t *testing.T, doc Document) {
				model, errs := doc.BuildV3Model()
				require.Empty(t, errs)
				pathItem := model.Model.Paths.PathItems.GetOrZero(pathName)
				responses := pathItem.Get.Responses
				respCode := responses.Codes.GetOrZero(respCode)
				mediaTypeResp := respCode.Content.GetOrZero(mediaType)

				var i int
				for exampleName, example := range mediaTypeResp.Examples.FromOldest() {
					assert.Equal(t, fmt.Sprintf("FoobarExample%d", i), exampleName)
					assert.Equal(t, fmt.Sprintf("Summary example %d", i), example.Summary)
					i++
				}
				assert.Equal(t, itemCount, i)
			}

			t.Run("Check order before rendering", func(t *testing.T) {
				checkOrder(t, doc)
			})

			_, _, _, errs = doc.RenderAndReload()
			require.Empty(t, errs)

			// Cannot test order of reloaded or unmarshalled examples.
			// The data type of `Example.Value` is `any`, and `yaml` package
			// will unmarshall associative array data to `map` objects, which
			// will lose consistent order.
		})
	})
}

func TestDocument_AdvanceCallbackReferences(t *testing.T) {
	bs, _ := os.ReadFile("test_specs/advancecallbackreferences/min-openapi.yaml")

	buf := bytes.NewBuffer([]byte{})

	config := datamodel.NewDocumentConfiguration()
	config.AllowRemoteReferences = true
	config.AllowFileReferences = true
	config.BasePath = "test_specs/advancecallbackreferences"
	config.Logger = slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelError}))

	doc, err := NewDocumentWithConfiguration(bs, config)
	require.NoError(t, err)

	_, errs := doc.BuildV3Model()
	require.Empty(t, errs)

	assert.Empty(t, buf.String())
}

func BenchmarkLoadDocTwice(b *testing.B) {
	for i := 0; i < b.N; i++ {

		spec, err := os.ReadFile("test_specs/speakeasy-test.yaml")
		require.NoError(b, err)

		doc, err := NewDocumentWithConfiguration(spec, &datamodel.DocumentConfiguration{
			BasePath:                            "./test_specs",
			IgnorePolymorphicCircularReferences: true,
			IgnoreArrayCircularReferences:       true,
			AllowFileReferences:                 true,
		})
		require.NoError(b, err)

		_, errs := doc.BuildV3Model()
		require.Empty(b, errs)

		doc, err = NewDocumentWithConfiguration(spec, &datamodel.DocumentConfiguration{
			BasePath:                            "./test_specs",
			IgnorePolymorphicCircularReferences: true,
			IgnoreArrayCircularReferences:       true,
			AllowFileReferences:                 true,
		})
		require.NoError(b, err)

		_, errs = doc.BuildV3Model()
		require.Empty(b, errs)

	}
}

func TestDocument_LoadDocTwice(t *testing.T) {
	spec, err := os.ReadFile("test_specs/speakeasy-test.yaml")
	require.NoError(t, err)

	doc, err := NewDocumentWithConfiguration(spec, &datamodel.DocumentConfiguration{
		BasePath:                            "./test_specs",
		IgnorePolymorphicCircularReferences: true,
		IgnoreArrayCircularReferences:       true,
		AllowFileReferences:                 true,
	})
	require.NoError(t, err)

	_, errs := doc.BuildV3Model()
	require.Empty(t, errs)

	doc, err = NewDocumentWithConfiguration(spec, &datamodel.DocumentConfiguration{
		BasePath:                            "./test_specs",
		IgnorePolymorphicCircularReferences: true,
		IgnoreArrayCircularReferences:       true,
		AllowFileReferences:                 true,
	})
	require.NoError(t, err)

	_, errs = doc.BuildV3Model()
	require.Empty(t, errs)
}

func TestSchemaTypeRef_Issue215(t *testing.T) {
	docBytes := []byte(`
openapi: 3.1.0
components:
  schemas:
    Foo:
      $schema: https://example.com/custom-json-schema-dialect
      type: string`)

	doc, err := NewDocument(docBytes)
	if err != nil {
		panic(err)
	}

	model, errs := doc.BuildV3Model()
	assert.NoError(t, errs)

	schema := model.Model.Components.Schemas.GetOrZero("Foo").Schema()
	schemaLow := schema.GoLow()

	// works as expected
	if v := schemaLow.SchemaTypeRef.Value; v != "https://example.com/custom-json-schema-dialect" {
		t.Errorf("low model: expected $schema to be 'https://example.com/custom-json-schema-dialect', but got '%v'", v)
	}

	// high model: expected $schema to be 'https://example.com/custom-json-schema-dialect', but got ''
	if v := schema.SchemaTypeRef; v != "https://example.com/custom-json-schema-dialect" {
		t.Errorf("high model: expected $schema to be 'https://example.com/custom-json-schema-dialect', but got '%v'", v)
	}
}

func TestMissingExtensions_Issue214(t *testing.T) {
	docBytes := []byte(`openapi: 3.1.0
x-time: 2020-12-24T12:00:00Z
x-string: test`)

	doc, err := NewDocument(docBytes)
	if err != nil {
		panic(err)
	}

	model, errs := doc.BuildV3Model()
	assert.NoError(t, errs)

	// x-string works as expected
	if extVal := model.Model.Extensions.GetOrZero("x-string"); extVal.Value != "test" {
		t.Errorf("expected x-string to be 'test', but got %v", extVal)
	}

	// expected x-time to be '2020-12-24T12:00:00Z', but got <nil>
	if extVal := model.Model.Extensions.GetOrZero("x-time"); extVal.Value != "2020-12-24T12:00:00Z" {
		t.Errorf("expected x-time to be '2020-12-24T12:00:00Z', but got %v", extVal)
	}
}

func TestDocument_TestNestedFiles(t *testing.T) {
	spec, err := os.ReadFile("test_specs/nested_files/openapi.yaml")
	require.NoError(t, err)

	doc, err := NewDocumentWithConfiguration(spec, &datamodel.DocumentConfiguration{
		BasePath:            "./test_specs/nested_files",
		AllowFileReferences: true,
	})
	require.NoError(t, err)

	_, errs := doc.BuildV3Model()
	require.Empty(t, errs)
}

func TestDocument_MinimalRemoteRefs(t *testing.T) {
	newRemoteHandlerFunc := func() utils.RemoteURLHandler {
		c := &http.Client{
			Timeout: time.Second * 120,
		}

		return func(url string) (*http.Response, error) {
			resp, err := c.Get(url)
			if err != nil {
				return nil, fmt.Errorf("fetch remote ref: %v", err)
			}

			return resp, nil
		}
	}

	spec, err := os.ReadFile("test_specs/minimal_remote_refs/openapi.yaml")
	require.NoError(t, err)

	baseURL, err := url.Parse("https://raw.githubusercontent.com/pb33f/libopenapi/refs/heads/main/test_specs/minimal_remote_refs/")
	require.NoError(t, err)

	doc, err := NewDocumentWithConfiguration(spec, &datamodel.DocumentConfiguration{
		BaseURL:               baseURL,
		AllowFileReferences:   false,
		AllowRemoteReferences: true,
		RemoteURLHandler:      newRemoteHandlerFunc(),
	})
	require.NoError(t, err)

	d, errs := doc.BuildV3Model()
	require.Empty(t, errs)

	o, err := d.Model.Render()
	require.NoError(t, err)
	fmt.Println(string(o))
}

func TestDocument_Issue264(t *testing.T) {
	openAPISpec := `{"openapi":"3.0.0","info":{"title":"dummy","version":"1.0.0"},"paths":{"/dummy":{"post":{"requestBody":{"content":{"application/json":{"schema":{"type":"object","properties":{"value":{"type":"number","format":"decimal","multipleOf":0.01,"minimum":-999.99}}}}}},"responses":{"200":{"description":"OK"}}}}}}`

	d, _ := NewDocument([]byte(openAPISpec))

	_, _ = d.BuildV3Model()

	_, _, _, errs := d.RenderAndReload() // code panics here
	assert.Nil(t, errs)
}

func TestDocument_Issue269(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: test
  version: "3"
paths: { }
components:
  schemas:
    Container:
      properties:
        pet:
          $ref: https://petstore3.swagger.io/api/v3/openapi.json#/components/schemas/Pet`

	doc, err := NewDocumentWithConfiguration([]byte(spec), &datamodel.DocumentConfiguration{
		AllowRemoteReferences: true,
	})
	if err != nil {
		panic(err)
	}
	_, errs := doc.BuildV3Model()
	assert.NoError(t, errs)
}

func TestDocument_Issue418(t *testing.T) {

	spec, _ := os.ReadFile("test_specs/nested_files/openapi-issue-418.yaml")

	doc, err := NewDocumentWithConfiguration(spec, &datamodel.DocumentConfiguration{
		AllowFileReferences: true,
		BasePath:            "test_specs/nested_files",
		SpecFilePath:        "test_specs/nested_files/openapi-issue-418.yaml",
	})
	if err != nil {
		panic(err)
	}
	m, errs := doc.BuildV3Model()
	assert.NoError(t, errs)
	assert.Len(t, m.Model.Index.GetResolver().GetResolvingErrors(), 0)
}

func TestDocument_Issue418_NoFile(t *testing.T) {

	spec, _ := os.ReadFile("test_specs/nested_files/openapi-issue-418.yaml")

	doc, err := NewDocumentWithConfiguration(spec, &datamodel.DocumentConfiguration{
		AllowFileReferences: true,
		BasePath:            "test_specs/nested_files",
	})
	if err != nil {
		panic(err)
	}
	m, errs := doc.BuildV3Model()
	assert.NoError(t, errs)
	assert.Len(t, m.Model.Index.GetResolver().GetResolvingErrors(), 0)
}

func TestDocument_RecursiveSchemaHash(t *testing.T) {
	data, err := os.ReadFile("test_specs/recursive-expression-test.yaml")
	assert.NoError(t, err)

	document, err := NewDocument(data)
	assert.NoError(t, err)

	model, errs := document.BuildV3Model()
	assert.Empty(t, errs)

	schema, found := model.Model.Components.Schemas.Get("RecursiveExpression")
	assert.True(t, found)
	assert.NotNil(t, schema)

	hash := schema.Schema().GoLow().Hash()
	assert.NotEmpty(t, hash)
}

// https://github.com/pb33f/libopenapi/issues/426
func TestDocument_MinimumZero(t *testing.T) {
	d := `openapi: "3.1"`

	doc, err := NewDocument([]byte(d))
	if err != nil {
		panic(err)
	}

	result, errs := doc.BuildV3Model()
	assert.NoError(t, errs)

	schema := &base.Schema{
		Type:       []string{"object"},
		Properties: orderedmap.New[string, *base.SchemaProxy](),
	}

	result.Model.Components = &v3high.Components{Schemas: orderedmap.New[string, *base.SchemaProxy]()}

	minimum := float64(0)
	fieldSchema := &base.Schema{
		Type:    []string{"number"},
		Minimum: &minimum,
	}

	schema.Properties.Set("minZeroField", base.CreateSchemaProxy(fieldSchema))

	result.Model.Components.Schemas.Set("minZeroSchema", base.CreateSchemaProxy(schema))

	rend, _ := result.Model.Render()

	expected := `openapi: "3.1"
components:
    schemas:
        minZeroSchema:
            type: object
            properties:
                minZeroField:
                    type: number
                    minimum: 0`

	assert.Equal(t, expected, strings.TrimSpace(string(rend)))
}

// Test sibling ref transformation - demonstrates how sibling properties with $ref
// get transformed into allOf structure when TransformSiblingRefs is enabled
func TestDocument_SiblingRefTransformation_LowLevel(t *testing.T) {
	// openapi spec with sibling ref (title alongside $ref)
	spec := `openapi: 3.1.0
info:
  title: Sibling Ref Test
  version: 1.0.0
components:
  schemas:
    destination-base:
      type: object
      properties:
        id:
          type: string
        name:
          type: string
    destination-amazon-sqs:
      title: destination-amazon-sqs
      $ref: '#/components/schemas/destination-base'`

	// create document with transformation enabled
	config := datamodel.NewDocumentConfiguration()
	config.TransformSiblingRefs = true

	doc, err := NewDocumentWithConfiguration([]byte(spec), config)
	assert.NoError(t, err)

	// debug: verify configuration was set correctly
	assert.True(t, config.TransformSiblingRefs, "configuration should have TransformSiblingRefs enabled")

	// build v3 model
	v3Doc, docErrs := doc.BuildV3Model()
	assert.Empty(t, docErrs)
	assert.NotNil(t, v3Doc)

	// debug: check if transformation occurred at low level
	lowDoc := v3Doc.Model.GoLow()
	if !lowDoc.Components.IsEmpty() && !lowDoc.Components.Value.Schemas.IsEmpty() {
		for pair := orderedmap.First(lowDoc.Components.Value.Schemas.Value); pair != nil; pair = pair.Next() {
			if pair.Key().Value == "destination-amazon-sqs" {
				lowSchema := pair.Value().Value.Schema()
				if lowSchema != nil {
					t.Logf("Low-level schema RootNode content: %v", lowSchema.RootNode)
					if lowSchema.RootNode != nil && len(lowSchema.RootNode.Content) > 0 {
						t.Logf("First element value: %v", lowSchema.RootNode.Content[0].Value)
					}
					// check if AllOf is populated at low level
					t.Logf("Low-level AllOf isEmpty: %v", lowSchema.AllOf.IsEmpty())
				}

				// also check the high-level schema
				if v3Doc.Model.Components != nil && v3Doc.Model.Components.Schemas != nil {
					if schema, found := v3Doc.Model.Components.Schemas.Get("destination-amazon-sqs"); found {
						highSchema := schema.Schema()
						if highSchema != nil {
							t.Logf("High-level schema AllOf length: %v", len(highSchema.AllOf))
							t.Logf("High-level schema Title: %v", highSchema.Title)
						}
					}
				}
				break
			}
		}
	}

	// The key verification: demonstrate that transformation works at low level
	// This shows the sibling ref was transformed from:
	//   title: destination-amazon-sqs
	//   $ref: '#/components/schemas/destination-base'
	// to:
	//   allOf:
	//     - title: destination-amazon-sqs
	//     - $ref: '#/components/schemas/destination-base'

	// First, verify transformation was applied at the low level
	found := false
	transformedContent := ""
	if !lowDoc.Components.IsEmpty() && !lowDoc.Components.Value.Schemas.IsEmpty() {
		for pair := orderedmap.First(lowDoc.Components.Value.Schemas.Value); pair != nil; pair = pair.Next() {
			if pair.Key().Value == "destination-amazon-sqs" {
				lowSchema := pair.Value().Value.Schema()
				if lowSchema != nil && lowSchema.RootNode != nil && len(lowSchema.RootNode.Content) > 0 {
					if lowSchema.RootNode.Content[0].Value == "allOf" {
						found = true
						// render the transformed node to show the structure
						transformedBytes, _ := yaml.Marshal(lowSchema.RootNode)
						transformedContent = string(transformedBytes)
					}
				}
				break
			}
		}
	}

	assert.True(t, found, "Transformation should have created allOf structure in low-level schema")
	assert.Contains(t, transformedContent, "allOf:")
	assert.Contains(t, transformedContent, "title: destination-amazon-sqs")
	assert.Contains(t, transformedContent, "$ref: '#/components/schemas/destination-base'")
}

// Test complete sibling ref transformation with full rendering support
// https://github.com/pb33f/libopenapi/issues/90
func TestDocument_SiblingRefTransformation_FullRender(t *testing.T) {
	// openapi spec with sibling ref (title alongside $ref)
	spec := `openapi: 3.1.0
info:
  title: Sibling Ref Test
  version: 1.0.0
components:
  schemas:
    destination-base:
      type: string
    destination-amazon-sqs:
      title: destination-amazon-sqs
      $ref: '#/components/schemas/destination-base'`

	// create document with transformation enabled
	config := datamodel.NewDocumentConfiguration()
	config.TransformSiblingRefs = true

	doc, err := NewDocumentWithConfiguration([]byte(spec), config)
	assert.NoError(t, err)

	// build v3 model
	v3Doc, docErrs := doc.BuildV3Model()
	assert.Empty(t, docErrs)
	assert.NotNil(t, v3Doc)

	// use RenderAndReload to render and reload the model
	renderedBytes, reloadedDoc, reloadedV3Doc, docErrs := doc.RenderAndReload()
	assert.Empty(t, docErrs)
	assert.NotNil(t, reloadedDoc)
	assert.NotNil(t, reloadedV3Doc)

	renderedStr := string(renderedBytes)

	// verify the rendered spec contains the transformed structure
	assert.Contains(t, renderedStr, "allOf:")
	assert.Contains(t, renderedStr, "- title: destination-amazon-sqs")
	assert.Contains(t, renderedStr, "- $ref: '#/components/schemas/destination-base'")

	// verify the reloaded model has the correct structure
	if reloadedV3Doc.Model.Components != nil && reloadedV3Doc.Model.Components.Schemas != nil {
		if destinationSchema, found := reloadedV3Doc.Model.Components.Schemas.Get("destination-amazon-sqs"); found {
			schema := destinationSchema.Schema()
			assert.NotNil(t, schema, "destination-amazon-sqs schema should exist")
			assert.Len(t, schema.AllOf, 2, "should have 2 allOf items")

			// verify first allOf item has title
			firstItem := schema.AllOf[0].Schema()
			assert.NotNil(t, firstItem)
			assert.Equal(t, "destination-amazon-sqs", firstItem.Title)

			// verify second allOf item is reference
			secondItem := schema.AllOf[1]
			assert.True(t, secondItem.IsReference())
			assert.Equal(t, "#/components/schemas/destination-base", secondItem.GetReference())
		} else {
			t.Fatal("destination-amazon-sqs schema not found in reloaded model")
		}
	} else {
		t.Fatal("components or schemas not found in reloaded model")
	}
}

func TestDocument_SiblingRefTransformation_FlippedOrder(t *testing.T) {
	spec := `openapi: 3.1.0
info:
  title: Sibling Ref Test
  version: 1.0.0
components:
  schemas:
    destination-base:
      description: hello
      type: string
    destination-amazon-sqs:
      $ref: '#/components/schemas/destination-base'
      description: destination-amazon-sqs`

	// create document with transformation enabled
	config := datamodel.NewDocumentConfiguration()
	config.TransformSiblingRefs = true

	doc, err := NewDocumentWithConfiguration([]byte(spec), config)
	assert.NoError(t, err)

	// build v3 model
	v3Doc, docErrs := doc.BuildV3Model()
	assert.Empty(t, docErrs)
	assert.NotNil(t, v3Doc)

	// use RenderAndReload to render and reload the model
	renderedBytes, reloadedDoc, reloadedV3Doc, docErrs := doc.RenderAndReload()
	assert.Empty(t, docErrs)
	assert.NotNil(t, reloadedDoc)
	assert.NotNil(t, reloadedV3Doc)

	renderedStr := string(renderedBytes)

	// verify the rendered spec contains the transformed structure
	assert.Contains(t, renderedStr, "allOf:")
	assert.Contains(t, renderedStr, "- description: destination-amazon-sqs")
	assert.Contains(t, renderedStr, "- $ref: '#/components/schemas/destination-base'")

	// verify the reloaded model has the correct structure
	if reloadedV3Doc.Model.Components != nil && reloadedV3Doc.Model.Components.Schemas != nil {
		if destinationSchema, found := reloadedV3Doc.Model.Components.Schemas.Get("destination-amazon-sqs"); found {
			schema := destinationSchema.Schema()
			assert.NotNil(t, schema, "destination-amazon-sqs schema should exist")
			assert.Len(t, schema.AllOf, 2, "should have 2 allOf items")

			// verify first allOf item has title
			firstItem := schema.AllOf[0].Schema()
			assert.NotNil(t, firstItem)
			assert.Equal(t, "destination-amazon-sqs", firstItem.Description)

			// verify second allOf item is reference
			secondItem := schema.AllOf[1]
			assert.True(t, secondItem.IsReference())
			assert.Equal(t, "#/components/schemas/destination-base", secondItem.GetReference())
		} else {
			t.Fatal("destination-amazon-sqs schema not found in reloaded model")
		}
	} else {
		t.Fatal("components or schemas not found in reloaded model")
	}
}
