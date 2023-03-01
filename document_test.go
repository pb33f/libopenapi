// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package libopenapi

import (
	"fmt"
	"github.com/pb33f/libopenapi/datamodel"
	"io/ioutil"
	"net/url"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/base"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/resolver"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
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

func TestDocument_RenderAndReload(t *testing.T) {

	yml := `openapi: 3.0
info:
    title: The magic API
`
	doc, _ := NewDocument([]byte(yml))
	v3Doc, _ := doc.BuildV3Model()

	v3Doc.Model.Info.Title = "The magic API - but now, altered!"
	bytes, _, newDocModel, err := doc.RenderAndReload()
	assert.Nil(t, err)
	assert.NotNil(t, bytes)
	assert.Equal(t, "The magic API - but now, altered!",
		newDocModel.Model.Info.Title)
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
		operation.Parameters[0].Description, operation.GoLow().Parameters.Value[0].IsReference,
		operation.GoLow().Parameters.Value[0].Reference)
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

func ExampleNewDocument_fromWithDocumentConfigurationFailure() {

	// This example shows how to create a document that prevents the loading of external references/
	// from files or the network

	// load in the Digital Ocean OpenAPI specification
	digitalOcean, _ := ioutil.ReadFile("test_specs/digitalocean.yaml")

	// create a DocumentConfiguration that prevents loading file and remote references
	config := datamodel.DocumentConfiguration{
		AllowFileReferences:   false,
		AllowRemoteReferences: false,
	}

	// create a new document from specification bytes
	doc, err := NewDocumentWithConfiguration(digitalOcean, &config)

	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	// only errors will be thrown, so just capture them and print the number of errors.
	_, errors := doc.BuildV3Model()

	// if anything went wrong when building the v3 model, a slice of errors will be returned
	if len(errors) > 0 {
		fmt.Println("Error building Digital Ocean spec errors reported")
	}
	// Output: Error building Digital Ocean spec errors reported
}

func ExampleNewDocument_fromWithDocumentConfigurationSuccess() {

	// This example shows how to create a document that prevents the loading of external references/
	// from files or the network

	// load in the Digital Ocean OpenAPI specification
	digitalOcean, _ := ioutil.ReadFile("test_specs/digitalocean.yaml")

	// Digital Ocean needs a baseURL to be set, so we can resolve relative references.
	baseURL, _ := url.Parse("https://raw.githubusercontent.com/digitalocean/openapi/main/specification")

	// create a DocumentConfiguration that allows loading file and remote references, and sets the baseURL
	// to somewhere that can resolve the relative references.
	config := datamodel.DocumentConfiguration{
		AllowFileReferences:   true,
		AllowRemoteReferences: true,
		BaseURL:               baseURL,
	}

	// create a new document from specification bytes
	doc, err := NewDocumentWithConfiguration(digitalOcean, &config)

	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	// only errors will be thrown, so just capture them and print the number of errors.
	_, errors := doc.BuildV3Model()

	// if anything went wrong when building the v3 model, a slice of errors will be returned
	if len(errors) > 0 {
		fmt.Println("Error building Digital Ocean spec errors reported")
	} else {
		fmt.Println("Digital Ocean spec built successfully")
	}
	// Output: Digital Ocean spec built successfully
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

// If you want to know more about circular references that have been found
// during the parsing/indexing/building of a document, you can capture the
// []errors thrown which are pointers to *resolver.ResolvingError
func ExampleNewDocument_infinite_circular_references() {

	// create a specification with an obvious and deliberate circular reference
	spec := `openapi: "3.1"
components:
  schemas:
    One:
      description: "test one"
      properties:
        things:
          "$ref": "#/components/schemas/Two"
      required:
        - things
    Two:
      description: "test two"
      properties:
        testThing:
          "$ref": "#/components/schemas/One"
      required:
        - testThing
`
	// create a new document from specification bytes
	doc, err := NewDocument([]byte(spec))

	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}
	_, errs := doc.BuildV3Model()

	// extract resolving error
	resolvingError := errs[0]

	// resolving error is a pointer to *resolver.ResolvingError
	// which provides access to rich details about the error.
	circularReference := resolvingError.(*resolver.ResolvingError).CircularReference

	// capture the journey with all details
	var buf strings.Builder
	for n := range circularReference.Journey {

		// add the full definition name to the journey.
		buf.WriteString(circularReference.Journey[n].Definition)
		if n < len(circularReference.Journey)-1 {
			buf.WriteString(" -> ")
		}
	}

	// print out the journey and the loop point.
	fmt.Printf("Journey: %s\n", buf.String())
	fmt.Printf("Loop Point: %s", circularReference.LoopPoint.Definition)
	// Output: Journey: #/components/schemas/Two -> #/components/schemas/One -> #/components/schemas/Two
	// Loop Point: #/components/schemas/Two
}

// This tests checks that circular references which are _not_ marked as required pass correctly
func TestNewDocument_terminable_circular_references(t *testing.T) {

	// create a specification with an obvious and deliberate circular reference
	spec := `openapi: "3.1"
components:
  schemas:
    One:
      description: "test one"
      properties:
        things:
          "$ref": "#/components/schemas/Two"
    Two:
      description: "test two"
      properties:
        testThing:
          "$ref": "#/components/schemas/One"
`
	// create a new document from specification bytes
	doc, err := NewDocument([]byte(spec))

	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}
	_, errs := doc.BuildV3Model()

	assert.Len(t, errs, 0)
}

// If you're using complex types with OpenAPI Extensions, it's simple to unpack extensions into complex
// types using `high.UnpackExtensions()`. libopenapi retains the original raw data in the low model (not the high)
// which means unpacking them can be a little complex.
//
// This example demonstrates how to use the `UnpackExtensions` with custom OpenAPI extensions.
func ExampleNewDocument_unpacking_extensions() {

	// define an example struct representing a cake
	type cake struct {
		Candles               int    `yaml:"candles"`
		Frosting              string `yaml:"frosting"`
		Some_Strange_Var_Name string `yaml:"someStrangeVarName"`
	}

	// define a struct that holds a map of cake pointers.
	type cakes struct {
		Description string
		Cakes       map[string]*cake
	}

	// define a struct representing a burger
	type burger struct {
		Sauce string
		Patty string
	}

	// define a struct that holds a map of cake pointers
	type burgers struct {
		Description string
		Burgers     map[string]*burger
	}

	// create a specification with a schema and parameter that use complex custom cakes and burgers extensions.
	spec := `openapi: "3.1"
components:
  schemas:
    SchemaOne:
      description: "Some schema with custom complex extensions"
      x-custom-cakes:
        description: some cakes
        cakes:
          someCake:
            candles: 10
            frosting: blue
            someStrangeVarName: something
          anotherCake:
            candles: 1
            frosting: green
  parameters:
    ParameterOne:
      description: "Some parameter also using complex extensions"
      x-custom-burgers:
        description: some burgers
        burgers:
          someBurger:
            sauce: ketchup
            patty: meat 
          anotherBurger:
            sauce: mayo
            patty: lamb`
	// create a new document from specification bytes
	doc, err := NewDocument([]byte(spec))

	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	// build a v3 model.
	docModel, errs := doc.BuildV3Model()

	// if anything went wrong building, indexing and resolving the model, an error is thrown
	if errs != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	// get a reference to SchemaOne and ParameterOne
	schemaOne := docModel.Model.Components.Schemas["SchemaOne"].Schema()
	parameterOne := docModel.Model.Components.Parameters["ParameterOne"]

	// unpack schemaOne extensions into complex `cakes` type
	schemaOneExtensions, schemaUnpackErrors := high.UnpackExtensions[cakes, *low.Schema](schemaOne)
	if schemaUnpackErrors != nil {
		panic(fmt.Sprintf("cannot unpack schema extensions: %e", err))
	}

	// unpack parameterOne into complex `burgers` type
	parameterOneExtensions, paramUnpackErrors := high.UnpackExtensions[burgers, *v3.Parameter](parameterOne)
	if paramUnpackErrors != nil {
		panic(fmt.Sprintf("cannot unpack parameter extensions: %e", err))
	}

	// extract extension by name for schemaOne
	customCakes := schemaOneExtensions["x-custom-cakes"]

	// extract extension by name for schemaOne
	customBurgers := parameterOneExtensions["x-custom-burgers"]

	// print out schemaOne complex extension details.
	fmt.Printf("schemaOne 'x-custom-cakes' (%s) has %d cakes, 'someCake' has %d candles and %s frosting\n",
		customCakes.Description,
		len(customCakes.Cakes),
		customCakes.Cakes["someCake"].Candles,
		customCakes.Cakes["someCake"].Frosting,
	)

	// print out parameterOne complex extension details.
	fmt.Printf("parameterOne 'x-custom-burgers' (%s) has %d burgers, 'anotherBurger' has %s sauce and a %s patty\n",
		customBurgers.Description,
		len(customBurgers.Burgers),
		customBurgers.Burgers["anotherBurger"].Sauce,
		customBurgers.Burgers["anotherBurger"].Patty,
	)

	// Output: schemaOne 'x-custom-cakes' (some cakes) has 2 cakes, 'someCake' has 10 candles and blue frosting
	//parameterOne 'x-custom-burgers' (some burgers) has 2 burgers, 'anotherBurger' has mayo sauce and a lamb patty

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
