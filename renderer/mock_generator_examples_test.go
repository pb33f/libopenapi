// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package renderer

import (
	"fmt"
	"os"

	"github.com/pb33f/libopenapi"
)

func ExampleMockGenerator_generateBurgerMock_yaml() {

	// create a new YAML mock generator
	mg := NewMockGenerator(YAML)

	burgerShop, _ := os.ReadFile("../test_specs/burgershop.openapi.yaml")

	// create a new document from specification and build a v3 model.
	document, _ := libopenapi.NewDocument(burgerShop)
	v3Model, _ := document.BuildV3Model()

	// create a mock of the Burger model
	burgerModel := v3Model.Model.Components.Schemas.GetOrZero("Burger")
	burger := burgerModel.Schema()
	mock, err := mg.GenerateMock(burger, "")

	if err != nil {
		panic(err)
	}
	fmt.Println(string(mock))
	// Output: name: Big Mac
	//numPatties: 2
}

func ExampleMockGenerator_generateFriesMock_json() {

	// create a new YAML mock generator
	mg := NewMockGenerator(JSON)

	burgerShop, _ := os.ReadFile("../test_specs/burgershop.openapi.yaml")

	// create a new document from specification and build a v3 model.
	document, _ := libopenapi.NewDocument(burgerShop)
	v3Model, _ := document.BuildV3Model()

	// create a mock of the Fries model
	friesModel := v3Model.Model.Components.Schemas.GetOrZero("Fries")
	fries := friesModel.Schema()
	mock, err := mg.GenerateMock(fries, "")

	if err != nil {
		panic(err)
	}
	fmt.Println(string(mock))
	// Output: {"favoriteDrink":{"drinkType":"coke","size":"M"},"potatoShape":"Crispy Shoestring"}
}

func ExampleMockGenerator_generateRequestMock_json() {

	// create a new YAML mock generator
	mg := NewMockGenerator(JSON)

	burgerShop, _ := os.ReadFile("../test_specs/burgershop.openapi.yaml")

	// create a new document from specification and build a v3 model.
	document, _ := libopenapi.NewDocument(burgerShop)
	v3Model, _ := document.BuildV3Model()

	// create a mock of the burger request model, extracted from the operation directly.
	burgerRequestModel := v3Model.Model.Paths.PathItems.GetOrZero("/burgers").
		Post.RequestBody.Content.GetOrZero("application/json")

	// use the 'cakeBurger' example to generate a mock
	mock, err := mg.GenerateMock(burgerRequestModel, "cakeBurger")

	if err != nil {
		panic(err)
	}
	fmt.Println(string(mock))
	// Output: {"name":"Chocolate Cake Burger","numPatties":5}
}

func ExampleMockGenerator_generateResponseMock_json() {

	mg := NewMockGenerator(JSON)
	// create a new YAML mock generator

	burgerShop, _ := os.ReadFile("../test_specs/burgershop.openapi.yaml")

	// create a new document from specification and build a v3 model.
	document, _ := libopenapi.NewDocument(burgerShop)
	v3Model, _ := document.BuildV3Model()

	// create a mock of the burger response model, extracted from the operation directly.
	burgerResponseModel := v3Model.Model.Paths.PathItems.GetOrZero("/burgers").
		Post.Responses.Codes.GetOrZero("200").Content.GetOrZero("application/json")

	// use the 'filetOFish' example to generate a mock
	mock, err := mg.GenerateMock(burgerResponseModel, "filetOFish")

	if err != nil {
		panic(err)
	}
	fmt.Println(string(mock))
	// Output: {"name":"Filet-O-Fish","numPatties":1}
}

func ExampleMockGenerator_generatePolymorphicMock_json() {

	mg := NewMockGenerator(JSON)
	// create a new YAML mock generator

	burgerShop, _ := os.ReadFile("../test_specs/burgershop.openapi.yaml")

	// create a new document from specification and build a v3 model.
	document, _ := libopenapi.NewDocument(burgerShop)
	v3Model, _ := document.BuildV3Model()

	// create a mock of the SomePayload component, which uses polymorphism (incorrectly)
	payloadModel := v3Model.Model.Components.Schemas.GetOrZero("SomePayload")
	payload := payloadModel.Schema()
	mock, err := mg.GenerateMock(payload, "")

	if err != nil {
		panic(err)
	}
	fmt.Println(string(mock))
	// Output: {"drinkType":"coke","size":"M"}
}
