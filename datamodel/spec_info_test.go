// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package datamodel

import (
	"fmt"
	"os"
	"testing"

	"github.com/pb33f/libopenapi/utils"
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

// badYAMLDuplicateKey is the exact scenario from issue #355
// Duplicate mapping keys should trigger a decode error
var badYAMLDuplicateKey = `openapi: 3.0.1
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      summary: List all pets
      responses:
        '200':
          description: Success
    get:
      summary: Duplicate get operation (invalid!)
      responses:
        '200':
          description: This is a duplicate key`

var badYAMLDuplicateKey2 = `swagger: 2.0
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      summary: List all pets
      responses:
        '200':
          description: Success
    get:
      summary: Duplicate get operation (invalid!)
      responses:
        '200':
          description: This is a duplicate key`

var badYAMLDuplicateKeyAsync = `asyncapi: 3.0
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      summary: List all pets
      responses:
        '200':
          description: Success
    get:
      summary: Duplicate get operation (invalid!)
      responses:
        '200':
          description: This is a duplicate key`

var badYAMLDuplicateKeyUnknown = `chipchop: 3.0
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      summary: List all pets
      responses:
        '200':
          description: Success
    get:
      summary: Duplicate get operation (invalid!)
      responses:
        '200':
          description: This is a duplicate key`

var badYAMLDuplicateUnknownType = `chipchop: 3.0
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      summary: List all pets
      responses:
        '200':
          description: Success
    get:
      summary: Duplicate get operation (invalid!)
      responses:
        '200':
          description: This is a duplicate key`

var OpenApiWat = `openapi: 3.3
info:
  title: Test API, valid, but not quite valid
servers:
  - url: https://quobix.com/api`

var OpenApi31 = `openapi: 3.1
info:
  title: Test API, valid, but not quite valid
servers:
  - url: https://quobix.com/api`

var OpenApi32 = `openapi: 3.2
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

// TestExtractSpecInfo_InvalidYAML_DuplicateKey tests issue #355
// Malformed YAML with duplicate keys should return an error when bypass=false
func TestExtractSpecInfo_InvalidYAML_DuplicateKey(t *testing.T) {
	_, e := ExtractSpecInfo([]byte(badYAMLDuplicateKey))
	assert.Error(t, e, "Should error on YAML with duplicate keys")
	assert.Contains(t, e.Error(), "already defined", "Error should mention duplicate key")
}

// TestExtractSpecInfo_InvalidYAML_DuplicateKey_WithBypass tests that bypass mode
// still allows malformed YAML to be processed without errors
func TestExtractSpecInfo_InvalidYAML_DuplicateKey_WithBypass(t *testing.T) {
	r, e := ExtractSpecInfoWithDocumentCheck([]byte(badYAMLDuplicateKey), true)
	assert.NoError(t, e, "Bypass mode should not error on malformed YAML")
	assert.NotNil(t, r, "Should return SpecInfo even with malformed YAML in bypass mode")
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
	assert.Equal(t, "3.3", r.Version)
}

func TestExtractSpecInfo_OpenAPI31(t *testing.T) {
	r, e := ExtractSpecInfo([]byte(OpenApi31))
	assert.Nil(t, e)
	assert.Equal(t, OpenApi3, r.SpecType)
	assert.Equal(t, "3.1", r.Version)
	assert.Contains(t, r.APISchema, "https://spec.openapis.org/oas/3.1/schema/2022-10-07")
}

func TestExtractSpecInfo_OpenAPI32(t *testing.T) {
	r, e := ExtractSpecInfo([]byte(OpenApi32))
	assert.Nil(t, e)
	assert.Equal(t, OpenApi3, r.SpecType)
	assert.Equal(t, "3.2", r.Version)
	assert.Contains(t, r.APISchema, "https://spec.openapis.org/oas/3.2/schema/2025-09-17")
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

func TestExtractSpecInfo_CheckSelf_BackwardsCompat(t *testing.T) {
	random := `openapi: 3.1.0
$self: something`

	r, e := ExtractSpecInfoWithDocumentCheck([]byte(random), false)
	assert.Nil(t, e)
	assert.NotNil(t, r.RootNode)
	assert.Len(t, *r.SpecBytes, 31)
	assert.Equal(t, "something", r.Self)
}

func TestExtractSpecInfo_CheckSelf(t *testing.T) {
	random := `openapi: 3.2
$self: something`

	r, e := ExtractSpecInfoWithDocumentCheck([]byte(random), false)
	assert.Nil(t, e)
	assert.NotNil(t, r.RootNode)
	assert.Len(t, *r.SpecBytes, 29)
	assert.Equal(t, "something", r.Self)
}

// TestUnescapeJSONSlashes tests the unescapeJSONSlashes helper function
// This addresses issue #479 where JSON files with \/ escape sequences fail to parse
func TestUnescapeJSONSlashes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple escaped slash", `\/`, `/`},
		{"url with escapes", `https:\/\/example.com\/path`, `https://example.com/path`},
		// \\ followed by / - the \\ is kept as \\, then / stays as /
		{"escaped backslash then literal slash", `\\/`, `\\/`},
		// \\ followed by \/ - the \\ is kept as \\, then \/ becomes /
		{"escaped backslash then escaped slash", `\\\/`, `\\/`},
		// \\\\ followed by \/ - two \\ pairs kept, then \/ becomes /
		{"double escaped backslash then escaped slash", `\\\\\/`, `\\\\/`},
		{"no escapes", `hello`, `hello`},
		{"empty", ``, ``},
		{"other escapes preserved", `\n\t\/`, `\n\t/`},
		{"multiple escaped slashes", `\/one\/two\/three`, `/one/two/three`},
		{"mixed content", `{"path":"\/test","url":"https:\/\/example.com"}`, `{"path":"/test","url":"https://example.com"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := unescapeJSONSlashes([]byte(tt.input))
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

// TestUnescapeJSONSlashes_NoAllocation tests that the fast path returns original slice
func TestUnescapeJSONSlashes_NoAllocation(t *testing.T) {
	input := []byte(`{"path":"/test"}`)
	result := unescapeJSONSlashes(input)
	// Should return same slice when no \/ present
	assert.Equal(t, &input[0], &result[0], "Should return original slice when no \\/ present")
}

// TestExtractSpecInfo_JSON_EscapedSlashes tests issue #479
// JSON files containing \/ (escaped forward slash) should parse correctly
func TestExtractSpecInfo_JSON_EscapedSlashes(t *testing.T) {
	// Exact test case from issue #479
	jsonWithEscapedSlash := `{"openapi":"3.0.0","info":{"title":"Escaped Slash Test","description":"This spec contains escaped forward slashes (\\/) that cause parsing issues","version":"1.0.0"},"paths":{"\/test":{"get":{"summary":"Test endpoint with escaped slashes","description":"The path \/test\/ contains escaped forward slashes","responses":{"200":{"description":"OK","content":{"application\/json":{"schema":{"type":"object","properties":{"url":{"type":"string","example":"https:\/\/example.com\/api\/test"},"path":{"type":"string","example":"\/users\/{id}\/profile"}}}}}}}}}}}`

	r, e := ExtractSpecInfo([]byte(jsonWithEscapedSlash))
	assert.NoError(t, e)
	assert.Equal(t, "3.0.0", r.Version)
	assert.Equal(t, JSONFileType, r.SpecFileType)
	assert.Equal(t, utils.OpenApi3, r.SpecType)
}

// TestExtractSpecInfo_JSON_EscapedSlashes_URL tests URL paths with escaped slashes
func TestExtractSpecInfo_JSON_EscapedSlashes_URL(t *testing.T) {
	jsonWithURL := `{"openapi":"3.0.0","info":{"title":"Test","version":"1.0.0"},"servers":[{"url":"https:\/\/api.example.com\/v1"}],"paths":{}}`

	r, e := ExtractSpecInfo([]byte(jsonWithURL))
	assert.NoError(t, e)
	assert.Equal(t, "3.0.0", r.Version)
	assert.Equal(t, JSONFileType, r.SpecFileType)
}

// TestExtractSpecInfo_JSON_EscapedBackslashAndSlash tests edge case with both \\ and \/
func TestExtractSpecInfo_JSON_EscapedBackslashAndSlash(t *testing.T) {
	// \\/ in JSON is escaped backslash followed by literal slash = \/ in the value
	// This should NOT be transformed incorrectly
	jsonWithBoth := `{"openapi":"3.0.0","info":{"title":"Test with \\\\/path","version":"1.0.0"},"paths":{}}`

	r, e := ExtractSpecInfo([]byte(jsonWithBoth))
	assert.NoError(t, e)
	assert.Equal(t, "3.0.0", r.Version)
}

// TestExtractSpecInfo_JSON_NoEscapedSlashes verifies normal JSON still works
func TestExtractSpecInfo_JSON_NoEscapedSlashes(t *testing.T) {
	normalJSON := `{"openapi":"3.0.0","info":{"title":"Test","version":"1.0.0"},"paths":{"/test":{"get":{"summary":"Test","responses":{"200":{"description":"OK"}}}}}}`

	r, e := ExtractSpecInfo([]byte(normalJSON))
	assert.NoError(t, e)
	assert.Equal(t, "3.0.0", r.Version)
	assert.Equal(t, JSONFileType, r.SpecFileType)
}

// TestExtractSpecInfo_YAML_NotAffected verifies YAML files are not affected by the fix
func TestExtractSpecInfo_YAML_NotAffected(t *testing.T) {
	yamlSpec := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
paths:
  /test:
    get:
      summary: Test
      responses:
        '200':
          description: OK`

	r, e := ExtractSpecInfo([]byte(yamlSpec))
	assert.NoError(t, e)
	assert.Equal(t, "3.0.0", r.Version)
	assert.Equal(t, YAMLFileType, r.SpecFileType)
}

func TestExtractSpecInfo_NoConfig(t *testing.T) {
	normalJSON := []byte(badYAMLDuplicateKey)

	r, e := ExtractSpecInfoWithConfig([]byte(normalJSON), nil)
	assert.Error(t, e)
	assert.Nil(t, r)
}

func TestExtractSpecInfo_ConfigSkip(t *testing.T) {
	normalJSON := []byte(badYAMLDuplicateKey2)

	r, e := ExtractSpecInfoWithConfig([]byte(normalJSON), &DocumentConfiguration{
		SkipJSONConversion: false,
	})
	assert.Error(t, e)
	assert.Nil(t, r)
}

func TestExtractSpecInfo_ConfigSkipAsyncApi(t *testing.T) {
	normalJSON := []byte(badYAMLDuplicateKeyAsync)

	r, e := ExtractSpecInfoWithConfig([]byte(normalJSON), &DocumentConfiguration{
		SkipJSONConversion: false,
	})
	assert.Error(t, e)
	assert.Nil(t, r)
}

func TestExtractSpecInfo_ConfigSkipAsyncUnknown(t *testing.T) {
	normalJSON := []byte(badYAMLDuplicateKeyUnknown)

	r, e := ExtractSpecInfoWithConfig([]byte(normalJSON), &DocumentConfiguration{
		SkipJSONConversion: false,
	})
	assert.Error(t, e)
	assert.Nil(t, r)
}
