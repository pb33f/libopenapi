// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"fmt"
	"io/ioutil"

	"github.com/pb33f/libopenapi/datamodel"
	lowv3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/orderedmap"
)

// An example of how to create a new high-level OpenAPI 3+ document from an OpenAPI specification.
func Example_createHighLevelOpenAPIDocument() {
	// Load in an OpenAPI 3+ specification as a byte slice.
	data, _ := ioutil.ReadFile("../../../test_specs/petstorev3.json")

	// Create a new *datamodel.SpecInfo from bytes.
	info, _ := datamodel.ExtractSpecInfo(data)

	var err []error

	// Create a new low-level Document, capture any errors thrown during creation.
	lowDoc, err = lowv3.CreateDocument(info)

	// Get upset if any errors were thrown.
	if len(err) > 0 {
		for i := range err {
			fmt.Printf("error: %e", err[i])
		}
		panic("something went wrong")
	}

	// Create a high-level Document from the low-level one.
	doc := NewDocument(lowDoc)

	// Print out some details
	fmt.Printf("Petstore contains %d paths and %d component schemas",
		orderedmap.Len(doc.Paths.PathItems), orderedmap.Len(doc.Components.Schemas))
	// Output: Petstore contains 13 paths and 8 component schemas
}
