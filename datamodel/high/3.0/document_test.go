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

	wentLow := highDoc.GoLow()
	assert.Equal(t, 1, wentLow.Version.ValueNode.Line)
	assert.Equal(t, 3, wentLow.Info.Value.Title.KeyNode.Line)

}

func TestNewDocument_Servers(t *testing.T) {
	h := NewDocument(doc)
	assert.Len(t, h.Servers, 2)
	assert.Equal(t, "{scheme}://api.pb33f.io", h.Servers[0].URL)
	assert.Equal(t, "this is our main API server, for all fun API things.", h.Servers[0].Description)
	assert.Len(t, h.Servers[0].Variables, 1)
	assert.Equal(t, "https", h.Servers[0].Variables["scheme"].Default)
	assert.Len(t, h.Servers[0].Variables["scheme"].Enum, 2)

	assert.Equal(t, "https://{domain}.{host}.com", h.Servers[1].URL)
	assert.Equal(t, "this is our second API server, for all fun API things.", h.Servers[1].Description)
	assert.Len(t, h.Servers[1].Variables, 2)
	assert.Equal(t, "api", h.Servers[1].Variables["domain"].Default)
	assert.Equal(t, "pb33f.io", h.Servers[1].Variables["host"].Default)

	wentLow := h.GoLow()
	assert.Equal(t, 45, wentLow.Servers.Value[0].Value.Description.KeyNode.Line)
	assert.Equal(t, 5, wentLow.Servers.Value[0].Value.Description.KeyNode.Column)
	assert.Equal(t, 45, wentLow.Servers.Value[0].Value.Description.ValueNode.Line)
	assert.Equal(t, 18, wentLow.Servers.Value[0].Value.Description.ValueNode.Column)
	// holy shit! the perfect Golang OpenAPI Model! high and low! fuck yeah!
}

func TestNewDocument_Tags(t *testing.T) {
	h := NewDocument(doc)
	assert.Len(t, h.Tags, 2)
}
