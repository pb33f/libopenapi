// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"fmt"
	"github.com/pb33f/libopenapi/datamodel"
	"io/ioutil"
)

// How to create a low-level Swagger / OpenAPI 2 Document from a specification
func Example_createLowLevelSwaggerDocument() {

	// How to create a low-level OpenAPI 2 Document

	// load petstore into bytes
	petstoreBytes, _ := ioutil.ReadFile("../../../test_specs/petstorev2.json")

	// read in specification
	info, _ := datamodel.ExtractSpecInfo(petstoreBytes)

	// build low-level document model
	document, errors := CreateDocument(info)

	// if something went wrong, a slice of errors is returned
	if len(errors) > 0 {
		for i := range errors {
			fmt.Printf("error: %s\n", errors[i].Error())
		}
		panic("cannot build document")
	}

	// print out email address from the info > contact object.
	fmt.Print(document.Info.Value.Contact.Value.Email.Value)
	// Output: apiteam@swagger.io

}

// How to create a low-level Swagger / OpenAPI 2 Document from a specification
func ExampleCreateDocument() {

	// How to create a low-level OpenAPI 2 Document

	// load petstore into bytes
	petstoreBytes, _ := ioutil.ReadFile("../../../test_specs/petstorev2.json")

	// read in specification
	info, _ := datamodel.ExtractSpecInfo(petstoreBytes)

	// build low-level document model
	document, errors := CreateDocument(info)

	// if something went wrong, a slice of errors is returned
	if len(errors) > 0 {
		for i := range errors {
			fmt.Printf("error: %s\n", errors[i].Error())
		}
		panic("cannot build document")
	}

	// print out email address from the info > contact object.
	fmt.Print(document.Info.Value.Contact.Value.Email.Value)
	// Output: apiteam@swagger.io

}
