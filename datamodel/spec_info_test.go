// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package datamodel

import (
	"fmt"
	"os"
	"testing"

	"github.com/pkg-base/libopenapi/utils"
	"github.com/stretchr/testify/assert"
)

const (
	// OpenApi3 is used by all OpenAPI 3+ docs
	OpenApi3 = "openapi"

	// OpenApi2 is used by all OpenAPI 2 docs, formerly known as swagger.
	OpenApi2 = "swagger"

	// AsyncApi is used by akk AsyncAPI docs, all versions.
	AsyncApi = "asyncapi"
)

var (
	goodJSON = `{"name":"kitty", "noises":["meow","purrrr","gggrrraaaaaooooww"]}`
	badJSON  = `{"name":"kitty, "noises":[{"meow","purrrr","gggrrraaaaaooooww"]}}`
	goodYAML = `name: kitty
noises:
- meow
- purrr
- gggggrrraaaaaaaaaooooooowwwwwww
`
)

var badYAML = `name: kitty
  noises:
   - meow
    - purrr
    - gggggrrraaaaaaaaaooooooowwwwwww
`

var OpenApiWat = `openapi: 3.2
info:
  title: Test API, valid, but not quite valid
servers:
  - url: https://quobix.com/api`

var OpenApi31 = `openapi: 3.1
info:
  title: Test API, valid, but not quite valid
servers:
  - url: https://quobix.com/api`

var OpenApiFalse = `openapi: false
info:
  title: Test API version is a bool?
servers:
  - url: https://quobix.com/api`

var OpenApiOne = `openapi: 1.0.1
info:
  title: Test API version is what version?
servers:
  - url: https://quobix.com/api`

var OpenApi3Spec = `openapi: 3.0.1
info:
  title: Test API
tags:
  - name: "Test"
  - name: "Test 2"
servers:
  - url: https://quobix.com/api`

var OpenApi2Spec = `swagger: 2.0.1
info:
  title: Test API
tags:
  - name: "Test"
servers:
  - url: https://quobix.com/api`

var OpenApi2SpecOdd = `swagger: 3.0.1
info:
  title: Test API
tags:
  - name: "Test"
servers:
  - url: https://quobix.com/api`

var AsyncAPISpec = `asyncapi: 2.0.0
info:
  title: Hello world application
  version: '0.1.0'
channels:
  hello:
    publish:
      message:
        payload:
          type: string
          pattern: '^hello .+$'`

var AsyncAPISpecOdd = `asyncapi: 3.0.0
info:
  title: Hello world application
  version: '0.1.0'`

func TestExtractSpecInfo_ValidJSON(t *testing.T) {
	r, e := ExtractSpecInfo([]byte(goodJSON))
	assert.Greater(t, len(*r.SpecJSONBytes), 0)
	assert.Error(t, e)
}

func TestExtractSpecInfo_InvalidJSON(t *testing.T) {
	_, e := ExtractSpecInfo([]byte(badJSON))
	assert.Error(t, e)
}

func TestExtractSpecInfo_Nothing(t *testing.T) {
	_, e := ExtractSpecInfo([]byte(""))
	assert.Error(t, e)
}

func TestExtractSpecInfo_ValidYAML(t *testing.T) {
	r, e := ExtractSpecInfo([]byte(goodYAML))
	assert.Greater(t, len(*r.SpecJSONBytes), 0)
	assert.Error(t, e)
}

func TestExtractSpecInfo_InvalidYAML(t *testing.T) {
	_, e := ExtractSpecInfo([]byte(badYAML))
	assert.Error(t, e)
}

func TestExtractSpecInfo_InvalidOpenAPIVersion(t *testing.T) {
	_, e := ExtractSpecInfo([]byte(OpenApiOne))
	assert.Error(t, e)
}

func TestExtractSpecInfo_OpenAPI3(t *testing.T) {
	r, e := ExtractSpecInfo([]byte(OpenApi3Spec))
	assert.Nil(t, e)
	assert.Equal(t, utils.OpenApi3, r.SpecType)
	assert.Equal(t, "3.0.1", r.Version)
	assert.Greater(t, len(*r.SpecJSONBytes), 0)
	assert.Contains(t, r.APISchema, "https://spec.openapis.org/oas/3.0/schema/2021-09-28")
}

func TestExtractSpecInfo_OpenAPIWat(t *testing.T) {
	r, e := ExtractSpecInfo([]byte(OpenApiWat))
	assert.Nil(t, e)
	assert.Equal(t, OpenApi3, r.SpecType)
	assert.Equal(t, "3.2", r.Version)
}

func TestExtractSpecInfo_OpenAPI31(t *testing.T) {
	r, e := ExtractSpecInfo([]byte(OpenApi31))
	assert.Nil(t, e)
	assert.Equal(t, OpenApi3, r.SpecType)
	assert.Equal(t, "3.1", r.Version)
	assert.Contains(t, r.APISchema, "https://spec.openapis.org/oas/3.1/schema/2022-10-07")
}

func TestExtractSpecInfo_AnyDocument(t *testing.T) {
	random := `something: yeah
nothing:
  - one
  - two
why:
  yes: no`

	r, e := ExtractSpecInfoWithDocumentCheck([]byte(random), true)
	assert.Nil(t, e)
	assert.NotNil(t, r.RootNode)
	assert.Equal(t, "something", r.RootNode.Content[0].Content[0].Value)
	assert.Len(t, *r.SpecBytes, 55)
}

func TestExtractSpecInfo_AnyDocument_Sync(t *testing.T) {
	random := `something: yeah
nothing:
  - one
  - two
why:
  yes: no`

	r, e := ExtractSpecInfoWithDocumentCheckSync([]byte(random), true)
	assert.Nil(t, e)
	assert.NotNil(t, r.RootNode)
	assert.Equal(t, "something", r.RootNode.Content[0].Content[0].Value)
	assert.Len(t, *r.SpecBytes, 55)
}

func TestExtractSpecInfo_AnyDocument_JSON(t *testing.T) {
	random := `{ "something" : "yeah"}`

	r, e := ExtractSpecInfoWithDocumentCheck([]byte(random), true)
	assert.Nil(t, e)
	assert.NotNil(t, r.RootNode)
	assert.Equal(t, "something", r.RootNode.Content[0].Content[0].Value)
	assert.Len(t, *r.SpecBytes, 23)
}

func TestExtractSpecInfo_AnyDocumentFromConfig(t *testing.T) {
	random := `something: yeah
nothing:
  - one
  - two
why:
  yes: no`

	r, e := ExtractSpecInfoWithConfig([]byte(random), &DocumentConfiguration{
		BypassDocumentCheck: true,
	})
	assert.Nil(t, e)
	assert.NotNil(t, r.RootNode)
	assert.Equal(t, "something", r.RootNode.Content[0].Content[0].Value)
	assert.Len(t, *r.SpecBytes, 55)
}

func TestExtractSpecInfo_OpenAPIFalse(t *testing.T) {
	spec, e := ExtractSpecInfo([]byte(OpenApiFalse))
	assert.NoError(t, e)
	assert.Equal(t, "false", spec.Version)
}

func TestExtractSpecInfo_OpenAPI2(t *testing.T) {
	r, e := ExtractSpecInfo([]byte(OpenApi2Spec))
	assert.Nil(t, e)
	assert.Equal(t, OpenApi2, r.SpecType)
	assert.Equal(t, "2.0.1", r.Version)
	assert.Greater(t, len(*r.SpecJSONBytes), 0)
	assert.Contains(t, r.APISchema, "http://swagger.io/v2/schema.json#")
}

func TestExtractSpecInfo_OpenAPI2_OddVersion(t *testing.T) {
	_, e := ExtractSpecInfo([]byte(OpenApi2SpecOdd))
	assert.NotNil(t, e)
	assert.Equal(t,
		"spec is defined as a swagger (openapi 2.0) spec, but is an openapi 3 or unknown version", e.Error())
}

func TestExtractSpecInfo_AsyncAPI(t *testing.T) {
	r, e := ExtractSpecInfo([]byte(AsyncAPISpec))
	assert.Nil(t, e)
	assert.Equal(t, AsyncApi, r.SpecType)
	assert.Equal(t, "2.0.0", r.Version)
	assert.Greater(t, len(*r.SpecJSONBytes), 0)
}

func TestExtractSpecInfo_AsyncAPI_OddVersion(t *testing.T) {
	_, e := ExtractSpecInfo([]byte(AsyncAPISpecOdd))
	assert.NotNil(t, e)
	assert.Equal(t,
		"spec is defined as asyncapi, but has a major version that is invalid", e.Error())
}

func TestExtractSpecInfo_BadVersion_OpenAPI3(t *testing.T) {
	yml := `openapi:
 should: fail`

	_, err := ExtractSpecInfo([]byte(yml))
	assert.Error(t, err)
}

func TestExtractSpecInfo_BadVersion_Swagger(t *testing.T) {
	yml := `swagger:
 should: fail`

	_, err := ExtractSpecInfo([]byte(yml))
	assert.Error(t, err)
}

func TestExtractSpecInfo_BadVersion_AsyncAPI(t *testing.T) {
	yml := `asyncapi:
 should: fail`

	_, err := ExtractSpecInfo([]byte(yml))
	assert.Error(t, err)
}

func ExampleExtractSpecInfo() {
	// load bytes from openapi spec file.
	bytes, _ := os.ReadFile("../test_specs/petstorev3.json")

	// create a new *SpecInfo instance from loaded bytes
	specInfo, err := ExtractSpecInfo(bytes)
	if err != nil {
		panic(fmt.Sprintf("cannot extract spec info: %e", err))
	}

	// print out the version, format and filetype
	fmt.Printf("the version of the spec is %s, the format is %s and the file type is %s",
		specInfo.Version, specInfo.SpecFormat, specInfo.SpecFileType)

	// Output: the version of the spec is 3.0.2, the format is oas3 and the file type is json
}

func TestExtractSpecInfoSync_Error(t *testing.T) {
	random := ``

	_, e := ExtractSpecInfoWithDocumentCheckSync([]byte(random), true)
	assert.Error(t, e)
}

func TestExtractSpecInfoWithDocumentCheck_Bypass_NonYAML(t *testing.T) {
	yml := `I am not: a parsable: yaml: file: at all.`

	info, err := ExtractSpecInfoWithDocumentCheck([]byte(yml), true)
	assert.Equal(t, "I am not: a parsable: yaml: file: at all.", info.RootNode.Content[0].Value)
	assert.NoError(t, err)
	assert.Equal(t, "I am not: a parsable: yaml: file: at all.", string(*info.SpecBytes))
}
