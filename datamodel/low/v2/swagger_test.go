// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/pkg-base/libopenapi/index"
	"github.com/pkg-base/libopenapi/utils"

	"github.com/pkg-base/libopenapi/datamodel"
	"github.com/pkg-base/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.Equal(t, 1, orderedmap.Len(doc.Parameters.Value.Definitions))
	assert.Len(t, doc.Tags.Value, 3)
	assert.Len(t, doc.Schemes.Value, 2)
	assert.Equal(t, 6, orderedmap.Len(doc.Definitions.Value.Schemas))
	assert.Equal(t, 3, orderedmap.Len(doc.SecurityDefinitions.Value.Definitions))
	assert.Equal(t, 15, orderedmap.Len(doc.Paths.Value.PathItems))
	assert.Equal(t, 2, orderedmap.Len(doc.Responses.Value.Definitions))
	assert.Equal(t, "http://swagger.io", doc.ExternalDocs.Value.URL.Value)

	var xPet bool
	_ = doc.FindExtension("x-pet").Value.Decode(&xPet)

	assert.Equal(t, true, xPet)
	assert.NotNil(t, doc.GetExternalDocs())
	assert.Equal(t, 1, orderedmap.Len(doc.GetExtensions()))
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

	var xChicken string
	_ = simpleParam.Value.FindExtension("x-chicken").Value.Decode(&xChicken)

	assert.Equal(t, "nuggets", xChicken)
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
	assert.Equal(t, 2, orderedmap.Len(petStoreAuth.Value.Scopes.Value.Values))
	assert.Equal(t, "read your pets", petStoreAuth.Value.Scopes.Value.FindScope("read:pets").Value)
}

func TestCreateDocument_Definitions(t *testing.T) {
	initTest()
	apiResp := doc.Definitions.Value.FindSchema("ApiResponse").Value.Schema()
	assert.NotNil(t, apiResp)
	assert.Equal(t, 3, orderedmap.Len(apiResp.Properties.Value))
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

	var xCoffee string
	_ = apiResp.Value.FindExtension("x-coffee").Value.Decode(&xCoffee)

	assert.Equal(t, "morning", xCoffee)

	header := apiResp.Value.FindHeader("noHeader")
	assert.NotNil(t, header)

	var xEmpty bool
	_ = header.Value.FindExtension("x-empty").Value.Decode(&xEmpty)

	assert.True(t, xEmpty)

	header = apiResp.Value.FindHeader("myHeader")

	var m map[string]any
	err := header.Value.Items.Value.Default.Value.Decode(&m)
	require.NoError(t, err)

	assert.Equal(t, "here", m["something"])

	var a []any
	err = header.Value.Items.Value.Items.Value.Default.Value.Decode(&a)
	require.NoError(t, err)

	assert.Len(t, a, 2)
	assert.Equal(t, "two", a[1])

	header = apiResp.Value.FindHeader("yourHeader")

	var def string
	_ = header.Value.Items.Value.Default.Value.Decode(&def)

	assert.Equal(t, "somethingSimple", def)

	assert.NotNil(t, apiResp.Value.Examples.Value.FindExample("application/json").Value)
}

func TestCreateDocument_Paths(t *testing.T) {
	initTest()
	uploadImage := doc.Paths.Value.FindPath("/pet/{petId}/uploadImage").Value
	assert.NotNil(t, uploadImage)
	assert.Nil(t, doc.Paths.Value.FindPath("/nothing-nowhere-nohow"))

	var xPotato string
	_ = uploadImage.FindExtension("x-potato").Value.Decode(&xPotato)

	assert.Equal(t, "man", xPotato)

	var xMinty string
	_ = doc.Paths.Value.FindExtension("x-minty").Value.Decode(&xMinty)

	assert.Equal(t, "fresh", xMinty)
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
	assert.Len(t, utils.UnwrapErrors(err), 2)
}

func TestCreateDocument_TagsBad(t *testing.T) {
	yml := `tags:
  $ref: bork`

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	var err error
	doc, err = CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())
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
	assert.Len(t, utils.UnwrapErrors(err), 2)
}

func TestCreateDocument_SecurityBad(t *testing.T) {
	yml := `security:
  $ref: `

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	var err error
	doc, err = CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())
	assert.Len(t, utils.UnwrapErrors(err), 1)
}

func TestCreateDocument_SecurityDefinitionsBad(t *testing.T) {
	yml := `securityDefinitions:
  $ref: `

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	var err error
	doc, err = CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())
	assert.Len(t, utils.UnwrapErrors(err), 1)
}

func TestCreateDocument_ResponsesBad(t *testing.T) {
	yml := `responses:
  $ref: `

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	var err error
	doc, err = CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())
	assert.Len(t, utils.UnwrapErrors(err), 1)
}

func TestCreateDocument_ParametersBad(t *testing.T) {
	yml := `parameters:
  $ref: `

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	var err error
	doc, err = CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())
	assert.Len(t, utils.UnwrapErrors(err), 1)
}

func TestCreateDocument_DefinitionsBad(t *testing.T) {
	yml := `definitions:
  $ref: `

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	var err error
	doc, err = CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())
	assert.Len(t, utils.UnwrapErrors(err), 1)
}

func TestCreateDocument_InfoBad(t *testing.T) {
	yml := `info:
  $ref: `

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	var err error
	doc, err = CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())
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
	assert.NotNil(t, lDoc)
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
