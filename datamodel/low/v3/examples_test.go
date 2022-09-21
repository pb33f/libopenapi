// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"fmt"
	"github.com/pb33f/libopenapi/datamodel"
	"io/ioutil"
)

func Example_createLowDocument() {
	// How to create a low-level OpenAPI 3 Document

	// load petstore into bytes
	petstoreBytes, _ := ioutil.ReadFile("../../../test_specs/petstorev3.json")

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
