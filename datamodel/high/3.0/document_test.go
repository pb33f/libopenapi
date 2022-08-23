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

func TestNewDocument_Extensions(t *testing.T) {
	h := NewDocument(doc)
	assert.Equal(t, "darkside", h.Extensions["x-something-something"])
}

func TestNewDocument_ExternalDocs(t *testing.T) {
	h := NewDocument(doc)
	assert.Equal(t, "https://pb33f.io", h.ExternalDocs.URL)
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

	wentLower := highDoc.Info.Contact.GoLow()
	assert.Equal(t, 8, wentLower.Name.ValueNode.Line)
	assert.Equal(t, 11, wentLower.Name.ValueNode.Column)

	wentLowAgain := highDoc.Info.GoLow()
	assert.Equal(t, 3, wentLowAgain.Title.ValueNode.Line)
	assert.Equal(t, 10, wentLowAgain.Title.ValueNode.Column)

	wentOnceMore := highDoc.Info.License.GoLow()
	assert.Equal(t, 12, wentOnceMore.Name.ValueNode.Line)
	assert.Equal(t, 11, wentOnceMore.Name.ValueNode.Column)

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

	wentLower := h.Servers[0].GoLow()
	assert.Equal(t, 45, wentLower.Description.ValueNode.Line)
	assert.Equal(t, 18, wentLower.Description.ValueNode.Column)

	wentLowest := h.Servers[0].Variables["scheme"].GoLow()
	assert.Equal(t, 50, wentLowest.Description.ValueNode.Line)
	assert.Equal(t, 22, wentLowest.Description.ValueNode.Column)

}

func TestNewDocument_Tags(t *testing.T) {
	h := NewDocument(doc)
	assert.Len(t, h.Tags, 2)
	assert.Equal(t, "Burgers", h.Tags[0].Name)
	assert.Equal(t, "All kinds of yummy burgers.", h.Tags[0].Description)
	assert.Equal(t, "Find out more", h.Tags[0].ExternalDocs.Description)
	assert.Equal(t, "https://pb33f.io", h.Tags[0].ExternalDocs.URL)
	assert.Equal(t, "somethingSpecial", h.Tags[0].Extensions["x-internal-ting"])
	assert.Equal(t, int64(1), h.Tags[0].Extensions["x-internal-tong"])
	assert.Equal(t, 1.2, h.Tags[0].Extensions["x-internal-tang"])
	assert.True(t, h.Tags[0].Extensions["x-internal-tung"].(bool))

	wentLow := h.Tags[1].GoLow()
	assert.Equal(t, 39, wentLow.Description.KeyNode.Line)
	assert.Equal(t, 5, wentLow.Description.KeyNode.Column)

	wentLower := h.Tags[0].ExternalDocs.GoLow()
	assert.Equal(t, 23, wentLower.Description.ValueNode.Line)
	assert.Equal(t, 20, wentLower.Description.ValueNode.Column)
}

func TestNewDocument_Components_Links(t *testing.T) {
	h := NewDocument(doc)
	assert.Len(t, h.Components.Links, 2)
	assert.Equal(t, "locateBurger", h.Components.Links["LocateBurger"].OperationId)
	assert.Equal(t, "$response.body#/id", h.Components.Links["LocateBurger"].Parameters["burgerId"])

	wentLow := h.Components.Links["LocateBurger"].GoLow()
	assert.Equal(t, 305, wentLow.OperationId.ValueNode.Line)
	assert.Equal(t, 20, wentLow.OperationId.ValueNode.Column)

}

func TestNewDocument_Components_Callbacks(t *testing.T) {
	h := NewDocument(doc)
	assert.Len(t, h.Components.Callbacks, 1)
	assert.Equal(t, "Callback payload",
		h.Components.Callbacks["BurgerCallback"].Expression["{$request.query.queryUrl}"].Post.RequestBody.Description)

	assert.Equal(t, 293,
		h.Components.Callbacks["BurgerCallback"].GoLow().FindExpression("{$request.query.queryUrl}").ValueNode.Line)
	assert.Equal(t, 9,
		h.Components.Callbacks["BurgerCallback"].GoLow().FindExpression("{$request.query.queryUrl}").ValueNode.Column)

	assert.Equal(t, "please", h.Components.Callbacks["BurgerCallback"].Extensions["x-break-everything"])

	for k := range h.Components.GoLow().Callbacks.Value {
		if k.Value == "BurgerCallback" {
			assert.Equal(t, 290, k.KeyNode.Line)
			assert.Equal(t, 5, k.KeyNode.Column)
		}
	}
}

func TestNewDocument_Components_Schemas(t *testing.T) {
	h := NewDocument(doc)
	assert.Len(t, h.Components.Schemas, 6)

	goLow := h.Components.GoLow()

	a := h.Components.Schemas["Error"]
	assert.Equal(t, "No such burger as 'Big-Whopper'", a.Properties["message"].Example)
	assert.Equal(t, 428, goLow.Schemas.KeyNode.Line)
	assert.Equal(t, 3, goLow.Schemas.KeyNode.Column)
	assert.Equal(t, 431, a.GoLow().Description.KeyNode.Line)

	b := h.Components.Schemas["Burger"]
	assert.Len(t, b.Required, 2)
	assert.Equal(t, "golden slices of happy fun joy", b.Properties["fries"].Description)
	assert.Equal(t, int64(2), b.Properties["numPatties"].Example)
	assert.Equal(t, 443, goLow.FindSchema("Burger").Value.Properties.KeyNode.Line)
	assert.Equal(t, 7, goLow.FindSchema("Burger").Value.Properties.KeyNode.Column)
	assert.Equal(t, 445, b.GoLow().FindProperty("name").ValueNode.Line)

	f := h.Components.Schemas["Fries"]
	assert.Equal(t, "salt", f.Properties["seasoning"].Items.Example)
	assert.Len(t, f.Properties["favoriteDrink"].Properties["drinkType"].Enum, 2)

	d := h.Components.Schemas["Drink"]
	assert.Len(t, d.Required, 2)
	assert.True(t, d.AdditionalProperties.(bool))
	assert.Equal(t, "drinkType", d.Discriminator.PropertyName)
	assert.Equal(t, "some value", d.Discriminator.Mapping["drink"])

	ext := h.Components.Extensions
	assert.Equal(t, "loud", ext["x-screaming-baby"])
}

func TestNewDocument_Components_Headers(t *testing.T) {
	h := NewDocument(doc)
	assert.Len(t, h.Components.Headers, 1)
	assert.Equal(t, "this is a header", h.Components.Headers["UseOil"].Description)
	assert.Equal(t, 318, h.Components.Headers["UseOil"].GoLow().Description.ValueNode.Line)
	assert.Equal(t, 20, h.Components.Headers["UseOil"].GoLow().Description.ValueNode.Column)
}
