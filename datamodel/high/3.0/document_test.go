// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel"
	lowv3 "github.com/pb33f/libopenapi/datamodel/low/3.0"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

var doc *lowv3.Document

func init() {
	data, _ := ioutil.ReadFile("../../../test_specs/burgershop.openapi.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)
	var err []error
	doc, err = lowv3.CreateDocument(info)
	if err != nil {
		panic("broken something")
	}
}

func BenchmarkNewDocument(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewDocument(doc)
	}
}

func TestNewDocument_Info(t *testing.T) {
	highDoc := NewDocument(doc)
	assert.Equal(t, "3.0.1", highDoc.Version)
	assert.Equal(t, "Burger Shop", highDoc.Info.Title)
	assert.Equal(t, "https://pb33f.io", highDoc.Info.TermsOfService)
	assert.Equal(t, "pb33f", highDoc.Info.Contact.Name)
	assert.Equal(t, "buckaroo@pb33f.io", highDoc.Info.Contact.Email)
	assert.Equal(t, "https://pb33f.io", highDoc.Info.Contact.URL)
	assert.Equal(t, "pb33f", highDoc.Info.License.Name)
	assert.Equal(t, "https://pb33f.io/made-up", highDoc.Info.License.URL)
	assert.Equal(t, "1.2", highDoc.Info.Version)
}
