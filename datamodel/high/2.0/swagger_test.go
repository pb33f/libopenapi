// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
    "github.com/pb33f/libopenapi/datamodel"
    v2 "github.com/pb33f/libopenapi/datamodel/low/2.0"
    "github.com/stretchr/testify/assert"

    "io/ioutil"
    "testing"
)

var doc *v2.Swagger

func initTest() {
    data, _ := ioutil.ReadFile("../../../test_specs/petstorev2-complete.yaml")
    info, _ := datamodel.ExtractSpecInfo(data)
    var err []error
    doc, err = v2.CreateDocument(info)
    if err != nil {
        panic("broken something")
    }
}

func TestNewSwaggerDocument(t *testing.T) {
    initTest()
    h := NewSwaggerDocument(doc)
    assert.NotNil(t, h)
}

func BenchmarkNewDocument(b *testing.B) {
    initTest()
    for i := 0; i < b.N; i++ {
        _ = NewSwaggerDocument(doc)
    }
}

func TestNewSwaggerDocument_Base(t *testing.T) {
    initTest()
    highDoc := NewSwaggerDocument(doc)
    assert.Equal(t, "2.0", highDoc.Swagger)
    assert.True(t, highDoc.Extensions["x-pet"].(bool))
    assert.Equal(t, "petstore.swagger.io", highDoc.Host)
    assert.Equal(t, "/v2", highDoc.BasePath)
    assert.Len(t, highDoc.Schemes, 2)
    assert.Equal(t, "https", highDoc.Schemes[0])
    assert.Len(t, highDoc.Consumes, 2)
    assert.Equal(t, "application/json", highDoc.Consumes[0])
    assert.Len(t, highDoc.Produces, 1)
    assert.Equal(t, "application/json", highDoc.Produces[0])

    wentLow := highDoc.GoLow()
    assert.Equal(t, 16, wentLow.Host.ValueNode.Line)
    assert.Equal(t, 7, wentLow.Host.ValueNode.Column)

}

func TestNewSwaggerDocument_Info(t *testing.T) {
    initTest()
    highDoc := NewSwaggerDocument(doc)
    assert.Equal(t, "1.0.6", highDoc.Info.Version)
    assert.NotEmpty(t, highDoc.Info.Description)
    assert.Equal(t, "Swagger Petstore", highDoc.Info.Title)
    assert.Equal(t, "Apache 2.0", highDoc.Info.License.Name)
    assert.Equal(t, "http://www.apache.org/licenses/LICENSE-2.0.html", highDoc.Info.License.URL)
    assert.Equal(t, "apiteam@swagger.io", highDoc.Info.Contact.Email)

    wentLow := highDoc.Info.Contact.GoLow()
    assert.Equal(t, 12, wentLow.Email.ValueNode.Line)
    assert.Equal(t, 12, wentLow.Email.ValueNode.Column)

    wentLowLic := highDoc.Info.License.GoLow()
    assert.Equal(t, 14, wentLowLic.Name.ValueNode.Line)
    assert.Equal(t, 11, wentLowLic.Name.ValueNode.Column)
}

func TestNewSwaggerDocument_Parameters(t *testing.T) {
    initTest()
    highDoc := NewSwaggerDocument(doc)
    params := highDoc.Parameters
    assert.Len(t, params.Definitions, 1)
    assert.Equal(t, "query", params.Definitions["simpleParam"].In)
    assert.Equal(t, "simple", params.Definitions["simpleParam"].Name)
    assert.Equal(t, "string", params.Definitions["simpleParam"].Type)
    assert.Equal(t, "nuggets", params.Definitions["simpleParam"].Extensions["x-chicken"])

    wentLow := params.GoLow()
    assert.Equal(t, 20, wentLow.FindParameter("simpleParam").ValueNode.Line)
    assert.Equal(t, 5, wentLow.FindParameter("simpleParam").ValueNode.Column)

    wentLower := params.Definitions["simpleParam"].GoLow()
    assert.Equal(t, 21, wentLower.Name.ValueNode.Line)
    assert.Equal(t, 11, wentLower.Name.ValueNode.Column)

}

func TestNewSwaggerDocument_Security(t *testing.T) {
    initTest()
    highDoc := NewSwaggerDocument(doc)
    assert.Len(t, highDoc.Security, 1)
    assert.Len(t, highDoc.Security[0].Requirements["global_auth"], 2)

    wentLow := highDoc.Security[0].GoLow()
    assert.Equal(t, 25, wentLow.Values.ValueNode.Line)
    assert.Equal(t, 5, wentLow.Values.ValueNode.Column)

}

func TestNewSwaggerDocument_Definitions_Security(t *testing.T) {
    initTest()
    highDoc := NewSwaggerDocument(doc)
    assert.Len(t, highDoc.SecurityDefinitions.Definitions, 3)
    assert.Equal(t, "oauth2", highDoc.SecurityDefinitions.Definitions["petstore_auth"].Type)
    assert.Equal(t, "https://petstore.swagger.io/oauth/authorize",
        highDoc.SecurityDefinitions.Definitions["petstore_auth"].AuthorizationUrl)
    assert.Equal(t, "implicit", highDoc.SecurityDefinitions.Definitions["petstore_auth"].Flow)
    assert.Len(t, highDoc.SecurityDefinitions.Definitions["petstore_auth"].Scopes.Values, 2)

    goLow := highDoc.SecurityDefinitions.GoLow()

    assert.Equal(t, 637, goLow.FindSecurityDefinition("petstore_auth").ValueNode.Line)
    assert.Equal(t, 5, goLow.FindSecurityDefinition("petstore_auth").ValueNode.Column)

    goLower := highDoc.SecurityDefinitions.Definitions["petstore_auth"].GoLow()
    assert.Equal(t, 640, goLower.Scopes.KeyNode.Line)
    assert.Equal(t, 5, goLower.Scopes.KeyNode.Column)

    goLowest := highDoc.SecurityDefinitions.Definitions["petstore_auth"].Scopes.GoLow()
    assert.Equal(t, 641, goLowest.FindScope("read:pets").ValueNode.Line)
    assert.Equal(t, 18, goLowest.FindScope("read:pets").ValueNode.Column)
}

func TestNewSwaggerDocument_Definitions_Responses(t *testing.T) {
    initTest()
    highDoc := NewSwaggerDocument(doc)
    assert.Len(t, highDoc.Responses.Definitions, 2)

    defs := highDoc.Responses.Definitions
    assert.Equal(t, "morning", defs["200"].Extensions["x-coffee"])
    assert.Equal(t, "OK", defs["200"].Description)
    assert.Equal(t, "a generic API response object",
        defs["200"].Schema.Schema().Description)
    assert.Len(t, defs["200"].Examples.Values, 3)

    exp := defs["200"].Examples.Values["application/json"]
    assert.Len(t, exp.(map[string]interface{}), 2)
    assert.Equal(t, "two", exp.(map[string]interface{})["one"])

    exp = defs["200"].Examples.Values["text/xml"]
    assert.Len(t, exp.([]interface{}), 3)
    assert.Equal(t, "two", exp.([]interface{})[1])

    exp = defs["200"].Examples.Values["text/plain"]
    assert.Equal(t, "something else.", exp)

    expWentLow := defs["200"].Examples.GoLow()
    assert.Equal(t, 678, expWentLow.FindExample("application/json").ValueNode.Line)
    assert.Equal(t, 9, expWentLow.FindExample("application/json").ValueNode.Column)

    wentLow := highDoc.Responses.GoLow()
    assert.Equal(t, 645, wentLow.FindResponse("200").ValueNode.Line)

}
