// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

import (
	"github.com/pb33f/libopenapi/datamodel"
	v2 "github.com/pb33f/libopenapi/datamodel/low/v2"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestCompareOpenAPIDocuments(t *testing.T) {

	original, _ := ioutil.ReadFile("../test_specs/burgershop.openapi.yaml")
	modified, _ := ioutil.ReadFile("../test_specs/burgershop.openapi-modified.yaml")
	infoOrig, _ := datamodel.ExtractSpecInfo(original)
	infoMod, _ := datamodel.ExtractSpecInfo(modified)

	origDoc, _ := v3.CreateDocument(infoOrig)
	modDoc, _ := v3.CreateDocument(infoMod)

	changes := CompareOpenAPIDocuments(origDoc, modDoc)
	assert.Equal(t, 67, changes.TotalChanges())
	assert.Equal(t, 17, changes.TotalBreakingChanges())
	//
	//out, _ := json.MarshalIndent(changes, "", "  ")
	//_ = ioutil.WriteFile("outputv3.json", out, 0776)

}

func TestCompareSwaggerDocuments(t *testing.T) {

	original, _ := ioutil.ReadFile("../test_specs/petstorev2-complete.yaml")
	modified, _ := ioutil.ReadFile("../test_specs/petstorev2-complete-modified.yaml")
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

	original, _ := ioutil.ReadFile("../test_specs/burgershop.openapi.yaml")
	modified, _ := ioutil.ReadFile("../test_specs/burgershop.openapi-modified.yaml")

	infoOrig, _ := datamodel.ExtractSpecInfo(original)
	infoMod, _ := datamodel.ExtractSpecInfo(modified)
	origDoc, _ := v3.CreateDocument(infoOrig)
	modDoc, _ := v3.CreateDocument(infoMod)

	for i := 0; i < b.N; i++ {
		CompareOpenAPIDocuments(origDoc, modDoc)
	}
}

func Benchmark_CompareSwaggerDocuments(b *testing.B) {

	original, _ := ioutil.ReadFile("../test_specs/petstorev2-complete.yaml")
	modified, _ := ioutil.ReadFile("../test_specs/petstorev2-complete-modified.yaml")
	infoOrig, _ := datamodel.ExtractSpecInfo(original)
	infoMod, _ := datamodel.ExtractSpecInfo(modified)

	origDoc, _ := v2.CreateDocument(infoOrig)
	modDoc, _ := v2.CreateDocument(infoMod)

	for i := 0; i < b.N; i++ {
		CompareSwaggerDocuments(origDoc, modDoc)
	}
}

func Benchmark_CompareOpenAPIDocuments_NoChange(b *testing.B) {

	original, _ := ioutil.ReadFile("../test_specs/burgershop.openapi.yaml")
	modified, _ := ioutil.ReadFile("../test_specs/burgershop.openapi.yaml")

	infoOrig, _ := datamodel.ExtractSpecInfo(original)
	infoMod, _ := datamodel.ExtractSpecInfo(modified)
	origDoc, _ := v3.CreateDocument(infoOrig)
	modDoc, _ := v3.CreateDocument(infoMod)

	for i := 0; i < b.N; i++ {
		CompareOpenAPIDocuments(origDoc, modDoc)
	}
}

func Benchmark_CompareK8s(b *testing.B) {

	original, _ := ioutil.ReadFile("../test_specs/k8s.json")
	modified, _ := ioutil.ReadFile("../test_specs/k8s.json")

	infoOrig, _ := datamodel.ExtractSpecInfo(original)
	infoMod, _ := datamodel.ExtractSpecInfo(modified)
	origDoc, _ := v2.CreateDocument(infoOrig)
	modDoc, _ := v2.CreateDocument(infoMod)

	for i := 0; i < b.N; i++ {
		CompareSwaggerDocuments(origDoc, modDoc)
	}
}

func Benchmark_CompareStripe(b *testing.B) {

	original, _ := ioutil.ReadFile("../test_specs/stripe.yaml")
	modified, _ := ioutil.ReadFile("../test_specs/stripe.yaml")

	infoOrig, _ := datamodel.ExtractSpecInfo(original)
	infoMod, _ := datamodel.ExtractSpecInfo(modified)
	origDoc, _ := v3.CreateDocument(infoOrig)
	modDoc, _ := v3.CreateDocument(infoMod)

	for i := 0; i < b.N; i++ {
		CompareOpenAPIDocuments(origDoc, modDoc)
	}
}
