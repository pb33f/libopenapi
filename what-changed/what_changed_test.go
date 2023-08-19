// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

import (
	"fmt"
	"os"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	v2 "github.com/pb33f/libopenapi/datamodel/low/v2"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/stretchr/testify/assert"
)

func TestCompareOpenAPIDocuments(t *testing.T) {

	original, _ := os.ReadFile("../test_specs/burgershop.openapi.yaml")
	modified, _ := os.ReadFile("../test_specs/burgershop.openapi-modified.yaml")
	infoOrig, _ := datamodel.ExtractSpecInfo(original)
	infoMod, _ := datamodel.ExtractSpecInfo(modified)

	origDoc, _ := v3.CreateDocument(infoOrig)
	modDoc, _ := v3.CreateDocument(infoMod)

	changes := CompareOpenAPIDocuments(origDoc, modDoc)
	assert.Equal(t, 72, changes.TotalChanges())
	assert.Equal(t, 17, changes.TotalBreakingChanges())
	//out, _ := json.MarshalIndent(changes, "", "  ")
	//_ = ioutil.WriteFile("outputv3.json", out, 0776)
}

func TestCompareSwaggerDocuments(t *testing.T) {

	original, _ := os.ReadFile("../test_specs/petstorev2-complete.yaml")
	modified, _ := os.ReadFile("../test_specs/petstorev2-complete-modified.yaml")
	infoOrig, _ := datamodel.ExtractSpecInfo(original)
	infoMod, _ := datamodel.ExtractSpecInfo(modified)

	origDoc, _ := v2.CreateDocument(infoOrig)
	modDoc, _ := v2.CreateDocument(infoMod)

	changes := CompareSwaggerDocuments(origDoc, modDoc)
	assert.Equal(t, 52, changes.TotalChanges())
	assert.Equal(t, 27, changes.TotalBreakingChanges())

	//out, _ := json.MarshalIndent(changes, "", "  ")
	//_ = ioutil.WriteFile("output.json", out, 0776)

}

func Benchmark_CompareOpenAPIDocuments(b *testing.B) {

	original, _ := os.ReadFile("../test_specs/burgershop.openapi.yaml")
	modified, _ := os.ReadFile("../test_specs/burgershop.openapi-modified.yaml")

	infoOrig, _ := datamodel.ExtractSpecInfo(original)
	infoMod, _ := datamodel.ExtractSpecInfo(modified)
	origDoc, _ := v3.CreateDocument(infoOrig)
	modDoc, _ := v3.CreateDocument(infoMod)

	for i := 0; i < b.N; i++ {
		CompareOpenAPIDocuments(origDoc, modDoc)
	}
}

func Benchmark_CompareSwaggerDocuments(b *testing.B) {

	original, _ := os.ReadFile("../test_specs/petstorev2-complete.yaml")
	modified, _ := os.ReadFile("../test_specs/petstorev2-complete-modified.yaml")
	infoOrig, _ := datamodel.ExtractSpecInfo(original)
	infoMod, _ := datamodel.ExtractSpecInfo(modified)

	origDoc, _ := v2.CreateDocument(infoOrig)
	modDoc, _ := v2.CreateDocument(infoMod)

	for i := 0; i < b.N; i++ {
		CompareSwaggerDocuments(origDoc, modDoc)
	}
}

func Benchmark_CompareOpenAPIDocuments_NoChange(b *testing.B) {

	original, _ := os.ReadFile("../test_specs/burgershop.openapi.yaml")
	modified, _ := os.ReadFile("../test_specs/burgershop.openapi.yaml")

	infoOrig, _ := datamodel.ExtractSpecInfo(original)
	infoMod, _ := datamodel.ExtractSpecInfo(modified)
	origDoc, _ := v3.CreateDocument(infoOrig)
	modDoc, _ := v3.CreateDocument(infoMod)

	for i := 0; i < b.N; i++ {
		CompareOpenAPIDocuments(origDoc, modDoc)
	}
}

func Benchmark_CompareK8s(b *testing.B) {

	original, _ := os.ReadFile("../test_specs/k8s.json")
	modified, _ := os.ReadFile("../test_specs/k8s.json")

	infoOrig, _ := datamodel.ExtractSpecInfo(original)
	infoMod, _ := datamodel.ExtractSpecInfo(modified)
	origDoc, _ := v2.CreateDocument(infoOrig)
	modDoc, _ := v2.CreateDocument(infoMod)

	for i := 0; i < b.N; i++ {
		CompareSwaggerDocuments(origDoc, modDoc)
	}
}

func Benchmark_CompareStripe(b *testing.B) {

	original, _ := os.ReadFile("../test_specs/stripe.yaml")
	modified, _ := os.ReadFile("../test_specs/stripe.yaml")

	infoOrig, _ := datamodel.ExtractSpecInfo(original)
	infoMod, _ := datamodel.ExtractSpecInfo(modified)
	origDoc, _ := v3.CreateDocument(infoOrig)
	modDoc, _ := v3.CreateDocument(infoMod)

	for i := 0; i < b.N; i++ {
		CompareOpenAPIDocuments(origDoc, modDoc)
	}
}

func ExampleCompareOpenAPIDocuments() {

	// Read in a 'left' (original) OpenAPI specification
	original, _ := os.ReadFile("../test_specs/burgershop.openapi.yaml")

	// Read in a 'right' (modified) OpenAPI specification
	modified, _ := os.ReadFile("../test_specs/burgershop.openapi-modified.yaml")

	// Extract SpecInfo from bytes
	infoOriginal, _ := datamodel.ExtractSpecInfo(original)
	infoModified, _ := datamodel.ExtractSpecInfo(modified)

	// Build OpenAPI Documents from SpecInfo
	origDocument, _ := v3.CreateDocument(infoOriginal)
	modDocDocument, _ := v3.CreateDocument(infoModified)

	// Compare OpenAPI Documents and extract to *DocumentChanges
	changes := CompareOpenAPIDocuments(origDocument, modDocDocument)

	// Extract SchemaChanges from components changes.
	schemaChanges := changes.ComponentsChanges.SchemaChanges

	// Print out some interesting stats.
	fmt.Printf("There are %d changes, of which %d are breaking. %v schemas have changes.",
		changes.TotalChanges(), changes.TotalBreakingChanges(), len(schemaChanges))
	//Output: There are 72 changes, of which 17 are breaking. 5 schemas have changes.
}
