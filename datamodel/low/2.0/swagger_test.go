// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

var doc *Swagger

func initTest() {
	if doc != nil {
		return
	}
	data, _ := ioutil.ReadFile("../../../test_specs/petstorev2-complete.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)
	var err []error
	doc, err = CreateDocument(info)
	wait := true
	for wait {
		select {
		case <-info.JsonParsingChannel:
			wait = false
		}
	}
	if err != nil {
		panic("broken something")
	}
}

func BenchmarkCreateDocument(b *testing.B) {
	data, _ := ioutil.ReadFile("../../../test_specs/petstorev2-complete.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)
	for i := 0; i < b.N; i++ {
		doc, _ = CreateDocument(info)
	}
}

func TestCreateDocument(t *testing.T) {
	initTest()
	assert.Equal(t, "2.0", doc.SpecInfo.Version)
	assert.Equal(t, "1.0.6", doc.Info.Value.Version.Value)
	assert.Equal(t, "petstore.swagger.io", doc.Host.Value)
	assert.Equal(t, "/v2", doc.BasePath.Value)
	assert.Len(t, doc.Parameters.Value.Definitions, 1)
	assert.Len(t, doc.Tags.Value, 3)
	assert.Len(t, doc.Schemes.Value, 2)
	assert.Len(t, doc.Definitions.Value.Schemas, 6)
	assert.Len(t, doc.SecurityDefinitions.Value.Definitions, 2)
	assert.Len(t, doc.Paths.Value.PathItems, 14)
	assert.Len(t, doc.Responses.Value.Definitions, 2)
	assert.Equal(t, "http://swagger.io", doc.ExternalDocs.Value.URL.Value)
	assert.Equal(t, true, doc.FindExtension("x-pet").Value)
	assert.Equal(t, true, doc.FindExtension("X-Pet").Value)
}

func TestCreateDocument_Info(t *testing.T) {
	initTest()
	assert.Equal(t, "Swagger Petstore", doc.Info.Value.Title.Value)
	assert.Equal(t, "apiteam@swagger.io", doc.Info.Value.Contact.Value.Email.Value)
	assert.Equal(t, "Apache 2.0", doc.Info.Value.License.Value.Name.Value)
}

func TestCreateDocument_Parameters(t *testing.T) {
	initTest()
	simpleParam := doc.Parameters.Value.FindParameter("simpleParam")
	assert.NotNil(t, simpleParam)
	assert.Equal(t, "simple", simpleParam.Value.Name.Value)
}

func TestCreateDocument_Tags(t *testing.T) {
	initTest()
	assert.Equal(t, "pet", doc.Tags.Value[0].Value.Name.Value)
	assert.Equal(t, "http://swagger.io", doc.Tags.Value[0].Value.ExternalDocs.Value.URL.Value)
	assert.Equal(t, "store", doc.Tags.Value[1].Value.Name.Value)
	assert.Equal(t, "user", doc.Tags.Value[2].Value.Name.Value)
	assert.Equal(t, "http://swagger.io", doc.Tags.Value[2].Value.ExternalDocs.Value.URL.Value)
}

func TestCreateDocument_SecurityDefinitions(t *testing.T) {
	initTest()
	apiKey := doc.SecurityDefinitions.Value.FindSecurityDefinition("api_key")
	assert.Equal(t, "apiKey", apiKey.Value.Type.Value)
	petStoreAuth := doc.SecurityDefinitions.Value.FindSecurityDefinition("petstore_auth")
	assert.Equal(t, "oauth2", petStoreAuth.Value.Type.Value)
	assert.Equal(t, "implicit", petStoreAuth.Value.Flow.Value)
	assert.Len(t, petStoreAuth.Value.Scopes.Value.Values, 2)
	assert.Equal(t, "read your pets", petStoreAuth.Value.Scopes.Value.FindScope("read:pets").Value)
}
