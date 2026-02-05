// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package libopenapi

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"testing"

	v2high "github.com/pb33f/libopenapi/datamodel/high/v2"
	what_changed "github.com/pb33f/libopenapi/what-changed"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"

	"github.com/pb33f/libopenapi/datamodel/high"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
	low "github.com/pb33f/libopenapi/datamodel/low/base"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
)

func ExampleNewDocument_fromOpenAPI3Document() {
	// How to read in an OpenAPI 3 Specification, into a Document.

	// load an OpenAPI 3 specification from bytes
	petstore, _ := os.ReadFile("test_specs/petstorev3.json")

	// create a new document from specification bytes
	document, err := NewDocument(petstore)
	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	// because we know this is a v3 spec, we can build a ready to go model from it.
	v3Model, err := document.BuildV3Model()

	// if anything went wrong when building the v3 model, an error will be returned.
	if err != nil {
		fmt.Printf("error: %e\n", err)
		panic(fmt.Sprintf("cannot create v3 model from document: %e", err))
	}

	// get a count of the number of paths and schemas.
	paths := orderedmap.Len(v3Model.Model.Paths.PathItems)
	schemas := orderedmap.Len(v3Model.Model.Components.Schemas)

	// print the number of paths and schemas in the document
	fmt.Printf("There are %d paths and %d schemas in the document", paths, schemas)
	// Output: There are 13 paths and 8 schemas in the document
}

func ExampleNewDocument_fromWithDocumentConfigurationFailure() {
	// This example shows how to create a document that prevents the loading of external references/
	// from files or the network

	// load in the Digital Ocean OpenAPI specification
	digitalOcean, _ := os.ReadFile("test_specs/digitalocean.yaml")

	// create a DocumentConfiguration that prevents loading file and remote references
	config := datamodel.NewDocumentConfiguration()

	// create a new structured logger to capture error logs that will be spewed out by the rolodex
	// when it tries to load external references. We're going to create a byte buffer to capture the logs
	// and then look at them after the document is built.
	var logs []byte
	buf := bytes.NewBuffer(logs)
	logger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	config.Logger = logger // set the config logger to our new logger.

	// Do not set any baseURL, as this will allow the rolodex to resolve relative references.
	// without a baseURL (for remote references, or a basePath for local references) the rolodex
	// will consider the reference to be local, and will not attempt to load it from the network.

	// create a new document from specification bytes
	doc, err := NewDocumentWithConfiguration(digitalOcean, config)
	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	// only errors will be thrown, so just capture them and print the number of errors.
	_, err = doc.BuildV3Model()

	// there should be 475 errors logs
	logItems := strings.Split(buf.String(), "\n")
	fmt.Printf("There are %d errors logged\n", len(logItems))

	if err != nil {
		fmt.Println("Error building Digital Ocean spec errors reported")
	}
	// Output: There are 475 errors logged
	// Error building Digital Ocean spec errors reported
}

func ExampleNewDocument_fromWithDocumentConfigurationSuccess() {
	// load in the Digital Ocean OpenAPI specification
	digitalOcean, _ := os.ReadFile("test_specs/digitalocean.yaml")

	// Digital Ocean needs a baseURL to be set, so we can resolve relative references.
	// baseURL, _ := url.Parse("https://raw.githubusercontent.com/digitalocean/openapi/main/specification")
	// locked this in to a release, because the spec is throwing 404's occasionally.
	baseURL, _ := url.Parse("https://raw.githubusercontent.com/digitalocean/openapi/ed0958267922794ec8cf540e19131a2d9664bfc7/specification")

	// create a DocumentConfiguration that allows loading file and remote references, and sets the baseURL
	// to somewhere that can resolve the relative references.
	config := datamodel.DocumentConfiguration{
		BaseURL: baseURL,
		Logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelError,
		})),
	}

	// create a new document from specification bytes
	doc, err := NewDocumentWithConfiguration(digitalOcean, &config)
	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, err := doc.BuildV3Model()

	// if anything went wrong when building the v3 model, a slice of errors will be returned
	if err != nil {
		fmt.Println("Error building Digital Ocean spec errors reported")
	} else {
		fmt.Println("Digital Ocean spec built successfully")
	}

	// running this through a change detection, will render out the entire model and
	// any stage two rendering for the model will be caught.
	what_changed.CompareOpenAPIDocuments(m.Model.GoLow(), m.Model.GoLow())
	// Output: Digital Ocean spec built successfully
}

func ExampleNewDocument_fromSwaggerDocument() {
	// How to read in a Swagger / OpenAPI 2 Specification, into a Document.

	// load a Swagger specification from bytes
	petstore, _ := os.ReadFile("test_specs/petstorev2.json")

	// create a new document from specification bytes
	document, err := NewDocument(petstore)
	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	// because we know this is a v2 spec, we can build a ready to go model from it.
	v2Model, err := document.BuildV2Model()

	// if anything went wrong when building the v3 model, and error will be returned
	if err != nil {
		fmt.Printf("error: %e\n", err)
		panic(fmt.Sprintf("cannot create v3 model from document: %e", err))
	}

	// get a count of the number of paths and schemas.
	paths := orderedmap.Len(v2Model.Model.Paths.PathItems)
	schemas := orderedmap.Len(v2Model.Model.Definitions.Definitions)

	// print the number of paths and schemas in the document
	fmt.Printf("There are %d paths and %d schemas in the document", paths, schemas)
	// Output: There are 14 paths and 6 schemas in the document
}

func ExampleNewDocument_fromUnknownVersion() {
	// load an unknown version of an OpenAPI spec
	burgershop, _ := os.ReadFile("test_specs/burgershop.openapi.yaml")

	var paths, schemas int
	var err error

	// create a new document from specification bytes
	document, err := NewDocument(burgershop)
	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	// We don't know which type of document this is, so we can use the spec info to inform us
	if document.GetSpecInfo().SpecType == utils.OpenApi3 {
		var v3Model *DocumentModel[v3high.Document]
		v3Model, err = document.BuildV3Model()
		if err == nil {
			paths = orderedmap.Len(v3Model.Model.Paths.PathItems)
			schemas = orderedmap.Len(v3Model.Model.Components.Schemas)
		}
	}
	if document.GetSpecInfo().SpecType == utils.OpenApi2 {
		var v2Model *DocumentModel[v2high.Swagger]
		v2Model, err = document.BuildV2Model()
		if err == nil {
			paths = orderedmap.Len(v2Model.Model.Paths.PathItems)
			schemas = orderedmap.Len(v2Model.Model.Definitions.Definitions)
		}
	}

	// if anything went wrong when building the model, report errors.
	if err != nil {
		fmt.Printf("error: %e\n", err)
		panic(fmt.Sprintf("cannot create v3 model from document: %e", err))
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
    url: https://some-place-on-the-internet.com/license
`
	// create a new document from specification bytes
	document, err := NewDocument([]byte(spec))
	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	// because we know this is a v3 spec, we can build a ready to go model from it.
	v3Model, err := document.BuildV3Model()

	// if anything went wrong when building the v3 model, a slice of errors will be returned
	if err != nil {
		fmt.Printf("error: %e\n", err)
		panic(fmt.Sprintf("cannot create v3 model from document: %e", err))
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
	// info:
	//     title: A new title for a useless spec
	//     contact:
	//         name: Buckaroo
	//         email: buckaroo@pb33f.io
	//     license:
	//         url: https://pb33f.io/license
}

func TestExampleCompareDocuments_openAPI(t *testing.T) {
	// How to compare two different OpenAPI specifications.

	// load an original OpenAPI 3 specification from bytes
	burgerShopOriginal, _ := os.ReadFile("test_specs/burgershop.openapi.yaml")

	// load an **updated** OpenAPI 3 specification from bytes
	burgerShopUpdated, _ := os.ReadFile("test_specs/burgershop.openapi-modified.yaml")

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
	if errs != nil {
		fmt.Printf("error: %e\n", errs)
		panic(fmt.Sprintf("cannot compare documents: %e", errs))
	}

	// Extract SchemaChanges from components changes.
	schemaChanges := documentChanges.ComponentsChanges.SchemaChanges

	// Print out some interesting stats about the OpenAPI document changes.
	assert.Equal(t, `There are 77 changes, of which 19 are breaking. 6 schemas have changes.`, fmt.Sprintf("There are %d changes, of which %d are breaking. %v schemas have changes.",
		documentChanges.TotalChanges(), documentChanges.TotalBreakingChanges(), len(schemaChanges)))
}

func TestExampleCompareDocuments_swagger(t *testing.T) {
	// How to compare two different Swagger specifications.

	// load an original OpenAPI 3 specification from bytes
	petstoreOriginal, _ := os.ReadFile("test_specs/petstorev2-complete.yaml")

	// load an **updated** OpenAPI 3 specification from bytes
	petstoreUpdated, _ := os.ReadFile("test_specs/petstorev2-complete-modified.yaml")

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
	if errs != nil {
		fmt.Printf("error: %e\n", errs)
		panic(fmt.Sprintf("cannot compare documents: %e", err))
	}

	// Extract SchemaChanges from components changes.
	schemaChanges := documentChanges.ComponentsChanges.SchemaChanges

	// Print out some interesting stats about the Swagger document changes.
	assert.Equal(t, `There are 52 changes, of which 27 are breaking. 5 schemas have changes.`, fmt.Sprintf("There are %d changes, of which %d are breaking. %v schemas have changes.",
		documentChanges.TotalChanges(), documentChanges.TotalBreakingChanges(), len(schemaChanges)))
}

func TestDocument_Paths_As_Array(t *testing.T) {
	// This test has invalid JSON (paths as array with object literal inside)
	// Testing that we properly reject invalid JSON after fix for issue #355
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
	// After fix #355, invalid JSON should now produce an error
	assert.Error(t, err, "Invalid JSON should produce an error")
	assert.Nil(t, doc, "Document should be nil when JSON is invalid")
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
	resolvingError := errs

	// resolving error is a pointer to *resolver.ResolvingError
	// which provides access to rich details about the error.

	var circularReference *index.CircularReferenceResult
	unwrapped := utils.UnwrapErrors(resolvingError)
	circularReference = unwrapped[0].(*index.ResolvingError).CircularReference

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

	assert.NoError(t, errs)
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
	schemaOne := docModel.Model.Components.Schemas.GetOrZero("SchemaOne").Schema()
	parameterOne := docModel.Model.Components.Parameters.GetOrZero("ParameterOne")

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
	customCakes := schemaOneExtensions.GetOrZero("x-custom-cakes")

	// extract extension by name for schemaOne
	customBurgers := parameterOneExtensions.GetOrZero("x-custom-burgers")

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
	// parameterOne 'x-custom-burgers' (some burgers) has 2 burgers, 'anotherBurger' has mayo sauce and a lamb patty
}

func ExampleNewDocument_modifyAndReRender() {
	// How to read in an OpenAPI 3 Specification, into a Document,
	// modify the document and then re-render it back to YAML bytes.

	// load an OpenAPI 3 specification from bytes
	petstore, _ := os.ReadFile("test_specs/petstorev3.json")

	// create a new document from specification bytes
	doc, err := NewDocument(petstore)
	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	// because we know this is a v3 spec, we can build a ready to go model from it.
	v3Model, errors := doc.BuildV3Model()

	// if anything went wrong when building the v3 model, a slice of errors will be returned
	if errors != nil {
		fmt.Printf("error: %e\n", errors)
		panic(fmt.Sprintf("cannot create v3 model from document: %e", errors))
	}

	// create a new path item and operation.
	newPath := &v3high.PathItem{
		Description: "this is a new path item",
		Get: &v3high.Operation{
			Description: "this is a get operation",
			OperationId: "getNewThing",
			RequestBody: &v3high.RequestBody{
				Description: "this is a new request body",
			},
		},
	}

	// capture original number of paths
	originalPaths := orderedmap.Len(v3Model.Model.Paths.PathItems)

	// add the path to the document
	v3Model.Model.Paths.PathItems.Set("/new/path", newPath)

	// render the document back to bytes and reload the model.
	rawBytes, _, newModel, errs := doc.RenderAndReload()

	// if anything went wrong when re-rendering the v3 model, a slice of errors will be returned
	if errors != nil {
		panic(fmt.Sprintf("cannot re-render document: %e", errs))
	}

	// capture new number of paths after re-rendering
	newPaths := orderedmap.Len(newModel.Model.Paths.PathItems)

	// print the number of paths and schemas in the document
	fmt.Printf("There were %d original paths. There are now %d paths in the document\n", originalPaths, newPaths)
	fmt.Printf("The new spec has %d bytes\n", len(rawBytes))
	// Output: There were 13 original paths. There are now 14 paths in the document
	// The new spec has 31406 bytes
}

// TestDocument_SkipExternalRefResolution_Issue519 verifies that when SkipExternalRefResolution is enabled,
// external $ref references are left unresolved while the rest of the model is built correctly.
// This is the scenario from https://github.com/pb33f/libopenapi/issues/519 — code generators like
// oapi-codegen need to parse specs with external refs without actually resolving them, using import
// mappings instead.
func TestDocument_SkipExternalRefResolution_Issue519(t *testing.T) {
	// An OpenAPI 3.1 spec where:
	// - Order schema has a property "product" that is an external $ref to ./models/product.yaml
	// - Order schema has a local property "id" (integer) that should be fully accessible
	// - Pet schema is a top-level external $ref to ./models/pet.yaml
	// - There is a local schema (ErrorResponse) with no external refs at all
	spec := `openapi: "3.1.0"
info:
  title: Test External Refs
  version: "1.0"
paths:
  /orders:
    get:
      summary: List orders
      operationId: listOrders
      responses:
        '200':
          description: A list of orders
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Order'
        '500':
          description: Server error
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'
  /pets:
    get:
      summary: List pets
      operationId: listPets
      responses:
        '200':
          description: A list of pets
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Pet'
components:
  schemas:
    Order:
      type: object
      properties:
        id:
          type: integer
          description: The order ID
        product:
          $ref: './models/product.yaml'
        customer:
          $ref: 'https://example.com/schemas/customer.yaml'
        warehouse:
          $ref: 'https://example.com/schemas/warehouse.yaml#/components/schemas/Warehouse'
      required:
        - id
        - product
        - warehouse
    Pet:
      $ref: './models/pet.yaml'
    ErrorResponse:
      type: object
      properties:
        code:
          type: integer
        message:
          type: string
`

	// ---- First: demonstrate the problem WITHOUT the flag ----
	// Without SkipExternalRefResolution, the rolodex tries to resolve external refs,
	// fails because no BasePath/BaseURL is set, and the model build either fails entirely
	// or produces schemas where Schema() returns nil for anything containing external refs.
	configWithout := datamodel.NewDocumentConfiguration()

	docWithout, err := NewDocumentWithConfiguration([]byte(spec), configWithout)
	assert.NoError(t, err, "document creation should succeed")

	modelWithout, buildErr := docWithout.BuildV3Model()
	// The build produces errors because external refs can't be resolved
	if buildErr == nil && modelWithout != nil {
		// If somehow the model builds, the Order schema with external refs in properties
		// should have Schema() returning nil — this is the core problem from issue #519
		orderWithout := modelWithout.Model.Components.Schemas.GetOrZero("Order")
		if orderWithout != nil {
			orderSchemaWithout := orderWithout.Schema()
			assert.Nil(t, orderSchemaWithout, "Without SkipExternalRefResolution, Order.Schema() "+
				"should be nil because external refs cannot be resolved")
		}
	}
	// Either way, the build is broken when external refs can't be resolved.
	// This is what issue #519 is about.

	// ---- Now: enable SkipExternalRefResolution and verify the fix ----
	config := datamodel.NewDocumentConfiguration()
	config.SkipExternalRefResolution = true

	doc, err := NewDocumentWithConfiguration([]byte(spec), config)
	assert.NoError(t, err, "document creation should succeed with SkipExternalRefResolution")

	model, errs := doc.BuildV3Model()
	assert.NoError(t, errs, "building model should not produce errors with SkipExternalRefResolution")
	assert.NotNil(t, model, "model should be non-nil")

	// Verify we can access components
	assert.NotNil(t, model.Model.Components, "Components should be non-nil")
	assert.NotNil(t, model.Model.Components.Schemas, "Schemas should be non-nil")

	// Verify paths are parsed correctly
	assert.NotNil(t, model.Model.Paths, "Paths should be non-nil")
	assert.Equal(t, 2, orderedmap.Len(model.Model.Paths.PathItems), "Should have 2 paths")

	// ---- Check the Order schema (object with external refs in properties) ----
	orderProxy := model.Model.Components.Schemas.GetOrZero("Order")
	assert.NotNil(t, orderProxy, "Order SchemaProxy should exist")

	// THIS is the key assertion from issue #519: Schema() should NOT be nil
	orderSchema := orderProxy.Schema()
	assert.NotNil(t, orderSchema, "Order.Schema() should NOT be nil when SkipExternalRefResolution is enabled")

	if orderSchema != nil {
		// The local "id" property should be fully accessible
		idProxy := orderSchema.Properties.GetOrZero("id")
		assert.NotNil(t, idProxy, "Order.properties.id should exist")
		if idProxy != nil {
			idSchema := idProxy.Schema()
			assert.NotNil(t, idSchema, "id schema should be buildable")
			if idSchema != nil {
				assert.Equal(t, "integer", idSchema.Type[0], "id should be type integer")
				assert.Equal(t, "The order ID", idSchema.Description, "id description should be set")
			}
		}

		// The "product" property should be an unresolved external ref
		productProxy := orderSchema.Properties.GetOrZero("product")
		assert.NotNil(t, productProxy, "Order.properties.product should exist")
		if productProxy != nil {
			assert.True(t, productProxy.IsReference(), "product should report IsReference()=true")
			assert.Equal(t, "./models/product.yaml", productProxy.GetReference(),
				"product GetReference() should return the external ref string")
			// Schema() should be nil for unresolved external refs — the ref is preserved but not resolved
			assert.Nil(t, productProxy.Schema(), "product.Schema() should be nil (external ref not resolved)")
		}

		// The "customer" property should also be an unresolved external ref (remote URL)
		customerProxy := orderSchema.Properties.GetOrZero("customer")
		assert.NotNil(t, customerProxy, "Order.properties.customer should exist")
		if customerProxy != nil {
			assert.True(t, customerProxy.IsReference(), "customer should report IsReference()=true")
			assert.Equal(t, "https://example.com/schemas/customer.yaml", customerProxy.GetReference(),
				"customer GetReference() should return the remote ref URL")
			assert.Nil(t, customerProxy.Schema(), "customer.Schema() should be nil (external ref not resolved)")
		}

		// Verify required fields are preserved
		assert.Contains(t, orderSchema.Required, "id")
		assert.Contains(t, orderSchema.Required, "product")
	}

	// ---- Check the Pet schema (top-level external $ref) ----
	petProxy := model.Model.Components.Schemas.GetOrZero("Pet")
	assert.NotNil(t, petProxy, "Pet SchemaProxy should exist")
	if petProxy != nil {
		assert.True(t, petProxy.IsReference(), "Pet should report IsReference()=true")
		assert.Equal(t, "./models/pet.yaml", petProxy.GetReference(),
			"Pet GetReference() should return the external ref string")
		// Top-level ref: Schema() is nil because the entire schema is an unresolved external ref
		assert.Nil(t, petProxy.Schema(), "Pet.Schema() should be nil (entire schema is external ref)")
	}

	// ---- Check the ErrorResponse schema (fully local, no external refs) ----
	errorProxy := model.Model.Components.Schemas.GetOrZero("ErrorResponse")
	assert.NotNil(t, errorProxy, "ErrorResponse SchemaProxy should exist")
	if errorProxy != nil {
		errorSchema := errorProxy.Schema()
		assert.NotNil(t, errorSchema, "ErrorResponse.Schema() should be non-nil (no external refs)")
		if errorSchema != nil {
			codeProxy := errorSchema.Properties.GetOrZero("code")
			assert.NotNil(t, codeProxy, "ErrorResponse.properties.code should exist")
			if codeProxy != nil && codeProxy.Schema() != nil {
				assert.Equal(t, "integer", codeProxy.Schema().Type[0])
			}

			msgProxy := errorSchema.Properties.GetOrZero("message")
			assert.NotNil(t, msgProxy, "ErrorResponse.properties.message should exist")
			if msgProxy != nil && msgProxy.Schema() != nil {
				assert.Equal(t, "string", msgProxy.Schema().Type[0])
			}
		}
	}

	// ---- Check that the response schema references work via paths ----
	ordersPath := model.Model.Paths.PathItems.GetOrZero("/orders")
	assert.NotNil(t, ordersPath, "/orders path should exist")
	if ordersPath != nil && ordersPath.Get != nil {
		resp200 := ordersPath.Get.Responses.Codes.GetOrZero("200")
		assert.NotNil(t, resp200, "200 response should exist")
		if resp200 != nil {
			jsonContent := resp200.Content.GetOrZero("application/json")
			assert.NotNil(t, jsonContent, "application/json content should exist")
			if jsonContent != nil && jsonContent.Schema != nil {
				arrSchema := jsonContent.Schema.Schema()
				assert.NotNil(t, arrSchema, "array schema should be non-nil")
				if arrSchema != nil {
					assert.Equal(t, "array", arrSchema.Type[0])
					// Items should reference Order via local $ref
					assert.NotNil(t, arrSchema.Items, "items should exist")
					if arrSchema.Items != nil && arrSchema.Items.A != nil {
						assert.True(t, arrSchema.Items.A.IsReference(), "items should be a reference to Order")
					}
				}
			}
		}
	}
}
