// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"fmt"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/stretchr/testify/assert"
)

var doc *Swagger

func initTest() {
	if doc != nil {
		return
	}
	data, _ := os.ReadFile("../../../test_specs/petstorev2-complete.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)
	var err error
	doc, err = CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())
	wait := true
	for wait {
		select {
		case <-info.JsonParsingChannel:
			wait = false
		}
	}
	if err != nil {
		fmt.Print(err)
		panic(err)
	}
}

func BenchmarkCreateDocument(b *testing.B) {
	data, _ := os.ReadFile("../../../test_specs/petstorev2-complete.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)
	for i := 0; i < b.N; i++ {
		doc, _ = CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())
	}
}

func TestCreateDocument(t *testing.T) {
	initTest()
	doc := doc
	assert.Equal(t, "2.0", doc.SpecInfo.Version)
	assert.Equal(t, "1.0.6", doc.Info.Value.Version.Value)
	assert.Equal(t, "petstore.swagger.io", doc.Host.Value)
	assert.Equal(t, "/v2", doc.BasePath.Value)
	assert.Len(t, doc.Parameters.Value.Definitions, 1)
	assert.Len(t, doc.Tags.Value, 3)
	assert.Len(t, doc.Schemes.Value, 2)
	assert.Len(t, doc.Definitions.Value.Schemas, 6)
	assert.Len(t, doc.SecurityDefinitions.Value.Definitions, 3)
	assert.Len(t, doc.Paths.Value.PathItems, 15)
	assert.Len(t, doc.Responses.Value.Definitions, 2)
	assert.Equal(t, "http://swagger.io", doc.ExternalDocs.Value.URL.Value)
	assert.Equal(t, true, doc.FindExtension("x-pet").Value)
	assert.Equal(t, true, doc.FindExtension("X-Pet").Value)
	assert.NotNil(t, doc.GetExternalDocs())
	assert.Len(t, doc.GetExtensions(), 1)
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
	assert.Equal(t, "nuggets", simpleParam.Value.FindExtension("x-chicken").Value)

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

func TestCreateDocument_Definitions(t *testing.T) {
	initTest()
	apiResp := doc.Definitions.Value.FindSchema("ApiResponse").Value.Schema()
	assert.NotNil(t, apiResp)
	assert.Len(t, apiResp.Properties.Value, 3)
	assert.Equal(t, "integer", apiResp.FindProperty("code").Value.Schema().Type.Value.A)

	pet := doc.Definitions.Value.FindSchema("Pet").Value.Schema()
	assert.NotNil(t, pet)
	assert.Len(t, pet.Required.Value, 2)

	// perform a deep inline lookup on a schema to ensure chains work
	assert.Equal(t, "Category", pet.FindProperty("category").Value.Schema().XML.Value.Name.Value)

	// check enums
	assert.Len(t, pet.FindProperty("status").Value.Schema().Enum.Value, 3)
}

func TestCreateDocument_ResponseDefinitions(t *testing.T) {
	initTest()
	apiResp := doc.Responses.Value.FindResponse("200")
	assert.NotNil(t, apiResp)
	assert.Equal(t, "OK", apiResp.Value.Description.Value)
	assert.Equal(t, "morning", apiResp.Value.FindExtension("x-coffee").Value)

	header := apiResp.Value.FindHeader("noHeader")
	assert.NotNil(t, header)
	assert.True(t, header.Value.FindExtension("x-empty").Value.(bool))

	header = apiResp.Value.FindHeader("myHeader")
	if k, ok := header.Value.Items.Value.Default.Value.(map[string]interface{}); ok {
		assert.Equal(t, "here", k["something"])
	} else {
		panic("should not fail.")
	}
	if k, ok := header.Value.Items.Value.Items.Value.Default.Value.([]interface{}); ok {
		assert.Len(t, k, 2)
		assert.Equal(t, "two", k[1])
	} else {
		panic("should not fail.")
	}

	header = apiResp.Value.FindHeader("yourHeader")
	assert.Equal(t, "somethingSimple", header.Value.Items.Value.Default.Value)

	assert.NotNil(t, apiResp.Value.Examples.Value.FindExample("application/json").Value)

}

func TestCreateDocument_Paths(t *testing.T) {
	initTest()
	uploadImage := doc.Paths.Value.FindPath("/pet/{petId}/uploadImage").Value
	assert.NotNil(t, uploadImage)
	assert.Nil(t, doc.Paths.Value.FindPath("/nothing-nowhere-nohow"))
	assert.Equal(t, "man", uploadImage.FindExtension("x-potato").Value)
	assert.Equal(t, "fresh", doc.Paths.Value.FindExtension("x-minty").Value)
	assert.Equal(t, "successful operation",
		uploadImage.Post.Value.Responses.Value.FindResponseByCode("200").Value.Description.Value)

}

func TestCreateDocument_Bad(t *testing.T) {

	yml := `swagger:
  $ref: bork`

	info, err := datamodel.ExtractSpecInfo([]byte(yml))
	assert.Nil(t, info)
	assert.Error(t, err)
}

func TestCreateDocument_ExternalDocsBad(t *testing.T) {

	yml := `externalDocs:
  $ref: bork`

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	var err error
	doc, err = CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())
	wait := true
	for wait {
		select {
		case <-info.JsonParsingChannel:
			wait = false
		}
	}
	assert.Len(t, utils.UnwrapErrors(err), 2)
}

func TestCreateDocument_TagsBad(t *testing.T) {

	yml := `tags:
  $ref: bork`

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	var err error
	doc, err = CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())
	wait := true
	for wait {
		select {
		case <-info.JsonParsingChannel:
			wait = false
		}
	}
	assert.Len(t, utils.UnwrapErrors(err), 2)
}

func TestCreateDocument_PathsBad(t *testing.T) {

	yml := `paths:
  "/hey":
    post:
      responses:
        "200":
          $ref: bork`

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	var err error
	doc, err = CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())
	wait := true
	for wait {
		select {
		case <-info.JsonParsingChannel:
			wait = false
		}
	}
	assert.Len(t, utils.UnwrapErrors(err), 2)
}

func TestCreateDocument_SecurityBad(t *testing.T) {

	yml := `security:
  $ref: `

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	var err error
	doc, err = CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())
	wait := true
	for wait {
		select {
		case <-info.JsonParsingChannel:
			wait = false
		}
	}
	assert.Len(t, utils.UnwrapErrors(err), 1)
}

func TestCreateDocument_SecurityDefinitionsBad(t *testing.T) {

	yml := `securityDefinitions:
  $ref: `

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	var err error
	doc, err = CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())
	wait := true
	for wait {
		select {
		case <-info.JsonParsingChannel:
			wait = false
		}
	}
	assert.Len(t, utils.UnwrapErrors(err), 1)
}

func TestCreateDocument_ResponsesBad(t *testing.T) {

	yml := `responses:
  $ref: `

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	var err error
	doc, err = CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())
	wait := true
	for wait {
		select {
		case <-info.JsonParsingChannel:
			wait = false
		}
	}
	assert.Len(t, utils.UnwrapErrors(err), 1)
}

func TestCreateDocument_ParametersBad(t *testing.T) {

	yml := `parameters:
  $ref: `

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	var err error
	doc, err = CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())
	wait := true
	for wait {
		select {
		case <-info.JsonParsingChannel:
			wait = false
		}
	}
	assert.Len(t, utils.UnwrapErrors(err), 1)
}

func TestCreateDocument_DefinitionsBad(t *testing.T) {

	yml := `definitions:
  $ref: `

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	var err error
	doc, err = CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())
	wait := true
	for wait {
		select {
		case <-info.JsonParsingChannel:
			wait = false
		}
	}
	assert.Len(t, utils.UnwrapErrors(err), 1)
}

func TestCreateDocument_InfoBad(t *testing.T) {

	yml := `info:
  $ref: `

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	var err error
	doc, err = CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())
	wait := true
	for wait {
		select {
		case <-info.JsonParsingChannel:
			wait = false
		}
	}
	assert.Len(t, utils.UnwrapErrors(err), 1)
}

func TestCircularReferenceError(t *testing.T) {

	data, _ := os.ReadFile("../../../test_specs/swagger-circular-tests.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)
	circDoc, err := CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())
	assert.NotNil(t, circDoc)
	assert.Len(t, utils.UnwrapErrors(err), 3)

}

func TestRolodexLocalFileSystem(t *testing.T) {
	data, _ := os.ReadFile("../../../test_specs/first.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)

	cf := datamodel.NewDocumentConfiguration()
	cf.BasePath = "../../../test_specs"
	cf.FileFilter = []string{"first.yaml", "second.yaml", "third.yaml"}
	lDoc, err := CreateDocumentFromConfig(info, cf)
	assert.NotNil(t, lDoc)
	assert.NoError(t, err)
}

func TestRolodexLocalFileSystem_ProvideNonRolodexFS(t *testing.T) {
	data, _ := os.ReadFile("../../../test_specs/first.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)
	baseDir := "../../../test_specs"

	cf := datamodel.NewDocumentConfiguration()
	cf.BasePath = baseDir
	cf.FileFilter = []string{"first.yaml", "second.yaml", "third.yaml"}
	cf.LocalFS = os.DirFS(baseDir)
	lDoc, err := CreateDocumentFromConfig(info, cf)
	assert.NotNil(t, lDoc)
	assert.Error(t, err)
}

func TestRolodexLocalFileSystem_ProvideRolodexFS(t *testing.T) {
	data, _ := os.ReadFile("../../../test_specs/first.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)
	baseDir := "../../../test_specs"
	cf := datamodel.NewDocumentConfiguration()
	cf.BasePath = baseDir
	cf.FileFilter = []string{"first.yaml", "second.yaml", "third.yaml"}

	localFS, lErr := index.NewLocalFSWithConfig(&index.LocalFSConfig{
		BaseDirectory: baseDir,
		DirFS:         os.DirFS(baseDir),
		FileFilters:   cf.FileFilter,
	})
	cf.LocalFS = localFS

	assert.NoError(t, lErr)
	lDoc, err := CreateDocumentFromConfig(info, cf)
	assert.NotNil(t, lDoc)
	assert.NoError(t, err)
}

func TestRolodexLocalFileSystem_BadPath(t *testing.T) {
	data, _ := os.ReadFile("../../../test_specs/first.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)

	cf := datamodel.NewDocumentConfiguration()
	cf.BasePath = "/NOWHERE"
	cf.FileFilter = []string{"first.yaml", "second.yaml", "third.yaml"}
	lDoc, err := CreateDocumentFromConfig(info, cf)
	assert.Nil(t, lDoc)
	assert.Error(t, err)
}

func TestRolodexRemoteFileSystem(t *testing.T) {
	data, _ := os.ReadFile("../../../test_specs/first.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)

	cf := datamodel.NewDocumentConfiguration()

	baseUrl := "https://raw.githubusercontent.com/pb33f/libopenapi/main/test_specs"
	u, _ := url.Parse(baseUrl)
	cf.BaseURL = u
	lDoc, err := CreateDocumentFromConfig(info, cf)
	assert.NotNil(t, lDoc)
	assert.NoError(t, err)
}

func TestRolodexRemoteFileSystem_BadBase(t *testing.T) {
	data, _ := os.ReadFile("../../../test_specs/first.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)

	cf := datamodel.NewDocumentConfiguration()

	baseUrl := "https://no-no-this-will-not-work-it-just-will-not-get-the-job-done-mate.com"
	u, _ := url.Parse(baseUrl)
	cf.BaseURL = u
	lDoc, err := CreateDocumentFromConfig(info, cf)
	assert.NotNil(t, lDoc)
	assert.Error(t, err)
}

func TestRolodexRemoteFileSystem_CustomRemote_NoBaseURL(t *testing.T) {
	data, _ := os.ReadFile("../../../test_specs/first.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)

	cf := datamodel.NewDocumentConfiguration()
	cf.RemoteFS, _ = index.NewRemoteFSWithConfig(&index.SpecIndexConfig{})
	lDoc, err := CreateDocumentFromConfig(info, cf)
	assert.NotNil(t, lDoc)
	assert.Error(t, err)
}

func TestRolodexRemoteFileSystem_CustomHttpHandler(t *testing.T) {
	data, _ := os.ReadFile("../../../test_specs/first.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)

	cf := datamodel.NewDocumentConfiguration()
	cf.RemoteURLHandler = http.Get
	baseUrl := "https://no-no-this-will-not-work-it-just-will-not-get-the-job-done-mate.com"
	u, _ := url.Parse(baseUrl)
	cf.BaseURL = u

	pizza := func(url string) (resp *http.Response, err error) {
		return nil, nil
	}
	cf.RemoteURLHandler = pizza
	lDoc, err := CreateDocumentFromConfig(info, cf)
	assert.NotNil(t, lDoc)
	assert.Error(t, err)
}

func TestRolodexRemoteFileSystem_FailRemoteFS(t *testing.T) {
	data, _ := os.ReadFile("../../../test_specs/first.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)

	cf := datamodel.NewDocumentConfiguration()
	cf.RemoteURLHandler = http.Get
	baseUrl := "https://no-no-this-will-not-work-it-just-will-not-get-the-job-done-mate.com"
	u, _ := url.Parse(baseUrl)
	cf.BaseURL = u

	pizza := func(url string) (resp *http.Response, err error) {
		return nil, nil
	}
	cf.RemoteURLHandler = pizza
	lDoc, err := CreateDocumentFromConfig(info, cf)
	assert.NotNil(t, lDoc)
	assert.Error(t, err)
}
