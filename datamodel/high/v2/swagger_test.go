// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"os"

	"github.com/pb33f/libopenapi/datamodel"
	v2 "github.com/pb33f/libopenapi/datamodel/low/v2"
	"github.com/stretchr/testify/assert"

	"testing"
)

var doc *v2.Swagger

func initTest() {
	data, _ := os.ReadFile("../../../test_specs/petstorev2-complete.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)
	var err error
	doc, err = v2.CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())
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
	assert.Equal(t, 25, wentLow.Requirements.ValueNode.Line)
	assert.Equal(t, 5, wentLow.Requirements.ValueNode.Column)

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

	assert.Equal(t, 661, goLow.FindSecurityDefinition("petstore_auth").ValueNode.Line)
	assert.Equal(t, 5, goLow.FindSecurityDefinition("petstore_auth").ValueNode.Column)

	goLower := highDoc.SecurityDefinitions.Definitions["petstore_auth"].GoLow()
	assert.Equal(t, 664, goLower.Scopes.KeyNode.Line)
	assert.Equal(t, 5, goLower.Scopes.KeyNode.Column)

	goLowest := highDoc.SecurityDefinitions.Definitions["petstore_auth"].Scopes.GoLow()
	assert.Equal(t, 665, goLowest.FindScope("read:pets").ValueNode.Line)
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
	assert.Equal(t, 702, expWentLow.FindExample("application/json").ValueNode.Line)
	assert.Equal(t, 9, expWentLow.FindExample("application/json").ValueNode.Column)

	wentLow := highDoc.Responses.GoLow()
	assert.Equal(t, 669, wentLow.FindResponse("200").ValueNode.Line)

	y := defs["500"].Headers["someHeader"]
	assert.Len(t, y.Enum, 2)
	x := y.Items

	assert.Equal(t, "something", x.Format)
	assert.Equal(t, "array", x.Type)
	assert.Equal(t, "csv", x.CollectionFormat)
	assert.Equal(t, "cake", x.Default)
	assert.Equal(t, 10, x.Maximum)
	assert.Equal(t, 1, x.Minimum)
	assert.True(t, x.ExclusiveMaximum)
	assert.True(t, x.ExclusiveMinimum)
	assert.Equal(t, 5, x.MaxLength)
	assert.Equal(t, 1, x.MinLength)
	assert.Equal(t, "hi!", x.Pattern)
	assert.Equal(t, 1, x.MinItems)
	assert.True(t, x.UniqueItems)
	assert.Len(t, x.Enum, 2)

	wentQuiteLow := y.GoLow()
	assert.Equal(t, 729, wentQuiteLow.Type.KeyNode.Line)

	wentLowest := x.GoLow()
	assert.Equal(t, 745, wentLowest.UniqueItems.KeyNode.Line)
}

func TestNewSwaggerDocument_Definitions(t *testing.T) {
	initTest()
	highDoc := NewSwaggerDocument(doc)

	assert.Len(t, highDoc.Definitions.Definitions, 6)

	wentLow := highDoc.Definitions.GoLow()
	assert.Equal(t, 848, wentLow.FindSchema("User").ValueNode.Line)

}

func TestNewSwaggerDocument_Paths(t *testing.T) {
	initTest()
	highDoc := NewSwaggerDocument(doc)
	assert.Len(t, highDoc.Paths.PathItems, 15)

	upload := highDoc.Paths.PathItems["/pet/{petId}/uploadImage"]
	assert.Equal(t, "man", upload.Extensions["x-potato"])
	assert.Nil(t, upload.Get)
	assert.Nil(t, upload.Put)
	assert.Nil(t, upload.Patch)
	assert.Nil(t, upload.Delete)
	assert.Nil(t, upload.Head)
	assert.Nil(t, upload.Options)
	assert.Equal(t, "pet", upload.Post.Tags[0])
	assert.Equal(t, "uploads an image", upload.Post.Summary)
	assert.NotEmpty(t, upload.Post.Description)
	assert.Equal(t, "uploadFile", upload.Post.OperationId)
	assert.Equal(t, "multipart/form-data", upload.Post.Consumes[0])
	assert.Equal(t, "application/json", upload.Post.Produces[0])
	assert.Len(t, upload.Post.Parameters, 3)
	assert.Equal(t, "petId", upload.Post.Parameters[0].Name)
	assert.Equal(t, "path", upload.Post.Parameters[0].In)
	assert.Equal(t, "ID of pet to update", upload.Post.Parameters[0].Description)
	assert.True(t, *upload.Post.Parameters[0].Required)
	assert.Equal(t, "integer", upload.Post.Parameters[0].Type)
	assert.Equal(t, "int64", upload.Post.Parameters[0].Format)
	assert.True(t, *upload.Post.Parameters[0].ExclusiveMaximum)
	assert.True(t, *upload.Post.Parameters[0].ExclusiveMinimum)
	assert.Equal(t, 2, *upload.Post.Parameters[0].MaxLength)
	assert.Equal(t, 1, *upload.Post.Parameters[0].MinLength)
	assert.Equal(t, 1, *upload.Post.Parameters[0].Minimum)
	assert.Equal(t, 5, *upload.Post.Parameters[0].Maximum)
	assert.Equal(t, "hi!", upload.Post.Parameters[0].Pattern)
	assert.Equal(t, 1, *upload.Post.Parameters[0].MinItems)
	assert.Equal(t, 20, *upload.Post.Parameters[0].MaxItems)
	assert.True(t, *upload.Post.Parameters[0].UniqueItems)
	assert.Len(t, upload.Post.Parameters[0].Enum, 2)
	assert.Equal(t, "hello", upload.Post.Parameters[0].Enum[0])
	def := upload.Post.Parameters[0].Default.(map[string]interface{})
	assert.Equal(t, "here", def["something"])

	assert.Equal(t, "https://pb33f.io", upload.Post.ExternalDocs.URL)
	assert.Len(t, upload.Post.Schemes, 2)

	wentLow := highDoc.Paths.GoLow()
	assert.Equal(t, 52,
		wentLow.FindPath("/pet/{petId}/uploadImage").ValueNode.Line)
	assert.Equal(t, 5,
		wentLow.FindPath("/pet/{petId}/uploadImage").ValueNode.Column)

	wentLower := upload.GoLow()
	assert.Equal(t, 52, wentLower.FindExtension("x-potato").ValueNode.Line)
	assert.Equal(t, 15, wentLower.FindExtension("x-potato").ValueNode.Column)

	wentLowest := upload.Post.GoLow()
	assert.Equal(t, 55, wentLowest.Tags.KeyNode.Line)

}

func TestNewSwaggerDocument_Responses(t *testing.T) {

	initTest()
	highDoc := NewSwaggerDocument(doc)
	upload := highDoc.Paths.PathItems["/pet/{petId}/uploadImage"].Post

	assert.Len(t, upload.Responses.Codes, 1)

	OK := upload.Responses.Codes["200"]
	assert.Equal(t, "successful operation", OK.Description)
	assert.Equal(t, "a generic API response object", OK.Schema.Schema().Description)

	wentLow := upload.Responses.GoLow()
	assert.Equal(t, 106, wentLow.FindResponseByCode("200").ValueNode.Line)

	wentLower := OK.GoLow()
	assert.Equal(t, 107, wentLower.Schema.KeyNode.Line)
	assert.Equal(t, 11, wentLower.Schema.KeyNode.Column)

}
