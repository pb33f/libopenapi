// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestSpecIndex_performExternalLookup(t *testing.T) {
	yml := `{
    "openapi": "3.1.0",
    "paths": [
        {"/": {
            "get": {}
        }}
    ]
}`
	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	c := CreateOpenAPIIndexConfig()
	index := NewSpecIndexWithConfig(&rootNode, c)
	assert.Len(t, index.GetPathsNode().Content, 1)
}

func TestSpecIndex_CheckCircularIndex(t *testing.T) {
	yml, _ := os.ReadFile("../test_specs/first.yaml")
	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	c := CreateOpenAPIIndexConfig()
	c.BasePath = "../test_specs"
	index := NewSpecIndexWithConfig(&rootNode, c)
	assert.Nil(t, index.uri)
	assert.NotNil(t, index.SearchIndexForReference("second.yaml#/properties/property2"))
	assert.NotNil(t, index.SearchIndexForReference("second.yaml"))
	assert.Nil(t, index.SearchIndexForReference("fourth.yaml"))
}

func TestSpecIndex_performExternalLookup_invalidURL(t *testing.T) {
	yml := `openapi: 3.1.0
components:
  schemas:
    thing:
      properties:
        thong:
          $ref: 'httpssss://not-gonna-work.com'`
	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	c := CreateOpenAPIIndexConfig()
	index := NewSpecIndexWithConfig(&rootNode, c)
	assert.Len(t, index.GetReferenceIndexErrors(), 2)
}

func TestSpecIndex_FindComponentInRoot(t *testing.T) {
	yml := `openapi: 3.1.0
components:
  schemas:
    thing:
      properties:
        thong: hi!`
	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	c := CreateOpenAPIIndexConfig()
	index := NewSpecIndexWithConfig(&rootNode, c)

	thing := index.FindComponentInRoot("#/$splish/$.../slash#$///./")
	assert.Nil(t, thing)
	assert.Len(t, index.GetReferenceIndexErrors(), 0)
}

func TestSpecIndex_FailLookupRemoteComponent_badPath(t *testing.T) {
	yml := `openapi: 3.1.0
components:
  schemas:
    thing:
      properties:
        thong:
          $ref: 'https://pb33f.io/site.webmanifest#/....$.ok../oh#/$$_-'`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	c := CreateOpenAPIIndexConfig()
	index := NewSpecIndexWithConfig(&rootNode, c)

	thing := index.FindComponentInRoot("#/$splish/$.../slash#$///./")
	assert.Nil(t, thing)
	assert.Len(t, index.GetReferenceIndexErrors(), 2)
}

func TestSpecIndex_FailLookupRemoteComponent_Ok_butNotFound(t *testing.T) {
	yml := `openapi: 3.1.0
components:
  schemas:
    thing:
      properties:
        thong:
          $ref: 'https://pb33f.io/site.webmanifest#/valid-but-missing'`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	c := CreateOpenAPIIndexConfig()
	index := NewSpecIndexWithConfig(&rootNode, c)

	thing := index.FindComponentInRoot("#/valid-but-missing")
	assert.Nil(t, thing)
	assert.Len(t, index.GetReferenceIndexErrors(), 1)
}

// disabled test because remote host is flaky.
//func TestSpecIndex_LocateRemoteDocsWithNoBaseURLSupplied(t *testing.T) {
//	// This test will push the index to do try and locate remote references that use relative references
//	spec := `openapi: 3.0.2
//info:
//  title: Test
//  version: 1.0.0
//paths:
//  /test:
//    get:
//      parameters:
//        - $ref: "https://schemas.opengis.net/ogcapi/features/part2/1.0/openapi/ogcapi-features-2.yaml#/components/parameters/crs"`
//
//	var rootNode yaml.Node
//	_ = yaml.Unmarshal([]byte(spec), &rootNode)
//
//	c := CreateOpenAPIIndexConfig()
//	index := NewSpecIndexWithConfig(&rootNode, c)
//
//	// extract crs param from index
//	crsParam := index.GetMappedReferences()["https://schemas.opengis.net/ogcapi/features/part2/1.0/openapi/ogcapi-features-2.yaml#/components/parameters/crs"]
//	assert.NotNil(t, crsParam)
//	assert.True(t, crsParam.IsRemote)
//	assert.Equal(t, "crs", crsParam.Node.Content[1].Value)
//	assert.Equal(t, "query", crsParam.Node.Content[3].Value)
//	assert.Equal(t, "form", crsParam.Node.Content[9].Value)
//}

func TestSpecIndex_LocateRemoteDocsWithRemoteURLHandler(t *testing.T) {
	// This test will push the index to do try and locate remote references that use relative references
	spec := `openapi: 3.0.2
info:
  title: Test
  version: 1.0.0
paths:
  /test:
    get:
      parameters:
        - $ref: "https://schemas.opengis.net/ogcapi/features/part2/1.0/openapi/ogcapi-features-2.yaml#/components/parameters/crs"`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(spec), &rootNode)

	c := CreateOpenAPIIndexConfig()
	c.RemoteURLHandler = httpClient.Get

	index := NewSpecIndexWithConfig(&rootNode, c)

	// extract crs param from index
	crsParam := index.GetMappedReferences()["https://schemas.opengis.net/ogcapi/features/part2/1.0/openapi/ogcapi-features-2.yaml#/components/parameters/crs"]
	assert.NotNil(t, crsParam)
	assert.True(t, crsParam.IsRemote)
	assert.Equal(t, "crs", crsParam.Node.Content[1].Value)
	assert.Equal(t, "query", crsParam.Node.Content[3].Value)
	assert.Equal(t, "form", crsParam.Node.Content[9].Value)
}

func TestSpecIndex_LocateRemoteDocsWithMalformedEscapedCharacters(t *testing.T) {
	spec := `openapi: 3.0.2
info:
  title: Test
  version: 1.0.0
paths:
  /test:
    get:
      parameters:
        - $ref: "https://petstore3.swagger.io/api/v3/openapi.yaml#/paths/~1pet~1%$petId%7D/get/parameters"`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(spec), &rootNode)

	c := CreateOpenAPIIndexConfig()
	c.RemoteURLHandler = httpClient.Get

	index := NewSpecIndexWithConfig(&rootNode, c)
	assert.Len(t, index.GetReferenceIndexErrors(), 2)
	assert.Equal(t, `invalid URL escape "%$p"`, index.GetReferenceIndexErrors()[0].Error())
	assert.Equal(t, "component 'https://petstore3.swagger.io/api/v3/openapi.yaml#/paths/~1pet~1%$petId%7D/get/parameters' does not exist in the specification", index.GetReferenceIndexErrors()[1].Error())
}

func TestSpecIndex_LocateRemoteDocsWithEscapedCharacters(t *testing.T) {
	spec := `openapi: 3.0.2
info:
  title: Test
  version: 1.0.0
paths:
  /test:
    get:
      parameters:
        - $ref: "https://petstore3.swagger.io/api/v3/openapi.yaml#/paths/~1pet~1%7BpetId%7D/get/parameters"`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(spec), &rootNode)

	c := CreateOpenAPIIndexConfig()
	c.RemoteURLHandler = httpClient.Get

	index := NewSpecIndexWithConfig(&rootNode, c)
	assert.Len(t, index.GetReferenceIndexErrors(), 0)
}

func TestGetRemoteDoc(t *testing.T) {
	// Mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(`OK`))
	}))
	// Close the server when test finishes
	defer server.Close()

	// Channel for data and error
	dataChan := make(chan []byte)
	errorChan := make(chan error)

	go getRemoteDoc(http.Get, server.URL, dataChan, errorChan)

	data := <-dataChan
	err := <-errorChan

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	expectedData := []byte(`OK`)
	if !reflect.DeepEqual(data, expectedData) {
		t.Errorf("Expected %v, got %v", expectedData, data)
	}
}

type FS struct{}
type FSBadOpen struct{}
type FSBadRead struct{}

type file struct {
	name string
	data string
}

type openFile struct {
	f      *file
	offset int64
}

func (f *openFile) Close() error               { return nil }
func (f *openFile) Stat() (fs.FileInfo, error) { return nil, nil }
func (f *openFile) Read(b []byte) (int, error) {
	if f.offset >= int64(len(f.f.data)) {
		return 0, io.EOF
	}
	if f.offset < 0 {
		return 0, &fs.PathError{Op: "read", Path: f.f.name, Err: fs.ErrInvalid}
	}
	n := copy(b, f.f.data[f.offset:])
	f.offset += int64(n)
	return n, nil
}

type badFileOpen struct{}

func (f *badFileOpen) Close() error               { return errors.New("bad file close") }
func (f *badFileOpen) Stat() (fs.FileInfo, error) { return nil, errors.New("bad file stat") }
func (f *badFileOpen) Read(b []byte) (int, error) {
	return 0, nil
}

type badFileRead struct {
	f      *file
	offset int64
}

func (f *badFileRead) Close() error               { return errors.New("bad file close") }
func (f *badFileRead) Stat() (fs.FileInfo, error) { return nil, errors.New("bad file stat") }
func (f *badFileRead) Read(b []byte) (int, error) {
	return 0, fmt.Errorf("bad file read")
}

func (f FS) Open(name string) (fs.File, error) {

	data := `type: string
name: something
in: query`

	return &openFile{&file{"test.yaml", data}, 0}, nil
}

func (f FSBadOpen) Open(name string) (fs.File, error) {
	return nil, errors.New("bad file open")
}

func (f FSBadRead) Open(name string) (fs.File, error) {
	return &badFileRead{&file{}, 0}, nil
}

func TestSpecIndex_UseRemoteHandler(t *testing.T) {

	spec := `openapi: 3.1.0
info:
  title: Test Remote Handler
  version: 1.0.0
paths:
  /test:
    get:
      parameters:
        - $ref: "https://i-dont-exist-but-it-does-not-matter.com/some-place/some-file.yaml"`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(spec), &rootNode)

	c := CreateOpenAPIIndexConfig()
	c.FSHandler = FS{}

	index := NewSpecIndexWithConfig(&rootNode, c)

	// extract crs param from index
	crsParam := index.GetMappedReferences()["https://i-dont-exist-but-it-does-not-matter.com/some-place/some-file.yaml"]
	assert.NotNil(t, crsParam)
	assert.True(t, crsParam.IsRemote)
	assert.Equal(t, "string", crsParam.Node.Content[1].Value)
	assert.Equal(t, "something", crsParam.Node.Content[3].Value)
	assert.Equal(t, "query", crsParam.Node.Content[5].Value)
}

func TestSpecIndex_UseFileHandler(t *testing.T) {

	spec := `openapi: 3.1.0
info:
  title: Test Remote Handler
  version: 1.0.0
paths:
  /test:
    get:
      parameters:
        - $ref: "some-file-that-does-not-exist.yaml"`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(spec), &rootNode)

	c := CreateOpenAPIIndexConfig()
	c.FSHandler = FS{}

	index := NewSpecIndexWithConfig(&rootNode, c)

	// extract crs param from index
	crsParam := index.GetMappedReferences()["some-file-that-does-not-exist.yaml"]
	assert.NotNil(t, crsParam)
	assert.True(t, crsParam.IsRemote)
	assert.Equal(t, "string", crsParam.Node.Content[1].Value)
	assert.Equal(t, "something", crsParam.Node.Content[3].Value)
	assert.Equal(t, "query", crsParam.Node.Content[5].Value)
}

func TestSpecIndex_UseRemoteHandler_Error_Open(t *testing.T) {

	spec := `openapi: 3.1.0
info:
  title: Test Remote Handler
  version: 1.0.0
paths:
  /test:
    get:
      parameters:
        - $ref: "https://-i-cannot-be-opened.com"`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(spec), &rootNode)

	c := CreateOpenAPIIndexConfig()
	c.FSHandler = FSBadOpen{}
	c.RemoteURLHandler = httpClient.Get

	index := NewSpecIndexWithConfig(&rootNode, c)

	assert.Len(t, index.GetReferenceIndexErrors(), 2)
	assert.Equal(t, "unable to open remote file: bad file open", index.GetReferenceIndexErrors()[0].Error())
	assert.Equal(t, "component 'https://-i-cannot-be-opened.com' does not exist in the specification", index.GetReferenceIndexErrors()[1].Error())
}

func TestSpecIndex_UseFileHandler_Error_Open(t *testing.T) {

	spec := `openapi: 3.1.0
info:
  title: Test File Handler
  version: 1.0.0
paths:
  /test:
    get:
      parameters:
        - $ref: "I-can-never-be-opened.yaml"`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(spec), &rootNode)

	c := CreateOpenAPIIndexConfig()
	c.FSHandler = FSBadOpen{}
	c.RemoteURLHandler = httpClient.Get

	index := NewSpecIndexWithConfig(&rootNode, c)

	assert.Len(t, index.GetReferenceIndexErrors(), 2)
	assert.Equal(t, "unable to open file: bad file open", index.GetReferenceIndexErrors()[0].Error())
	assert.Equal(t, "component 'I-can-never-be-opened.yaml' does not exist in the specification", index.GetReferenceIndexErrors()[1].Error())
}

func TestSpecIndex_UseRemoteHandler_Error_Read(t *testing.T) {

	spec := `openapi: 3.1.0
info:
  title: Test Remote Handler
  version: 1.0.0
paths:
  /test:
    get:
      parameters:
        - $ref: "https://-i-cannot-be-opened.com"`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(spec), &rootNode)

	c := CreateOpenAPIIndexConfig()
	c.FSHandler = FSBadRead{}
	c.RemoteURLHandler = httpClient.Get

	index := NewSpecIndexWithConfig(&rootNode, c)

	assert.Len(t, index.GetReferenceIndexErrors(), 2)
	assert.Equal(t, "unable to read remote file bytes: bad file read", index.GetReferenceIndexErrors()[0].Error())
	assert.Equal(t, "component 'https://-i-cannot-be-opened.com' does not exist in the specification", index.GetReferenceIndexErrors()[1].Error())
}

func TestSpecIndex_UseFileHandler_Error_Read(t *testing.T) {

	spec := `openapi: 3.1.0
info:
  title: Test File Handler
  version: 1.0.0
paths:
  /test:
    get:
      parameters:
        - $ref: "I-am-impossible-to-open-forever.yaml"`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(spec), &rootNode)

	c := CreateOpenAPIIndexConfig()
	c.FSHandler = FSBadRead{}
	c.RemoteURLHandler = httpClient.Get

	index := NewSpecIndexWithConfig(&rootNode, c)

	assert.Len(t, index.GetReferenceIndexErrors(), 2)
	assert.Equal(t, "unable to read file bytes: bad file read", index.GetReferenceIndexErrors()[0].Error())
	assert.Equal(t, "component 'I-am-impossible-to-open-forever.yaml' does not exist in the specification", index.GetReferenceIndexErrors()[1].Error())
}

func TestSpecIndex_UseFileHandler_ErrorReference(t *testing.T) {

	spec := `openapi: 3.1.0
info:
  title: Test File Handler
  version: 1.0.0
paths:
  /test:
    get:
      parameters:
        - $ref: "exisiting.yaml#/paths/~1pet~1%$petId%7D/get/parameters"`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(spec), &rootNode)

	c := CreateOpenAPIIndexConfig()
	c.FSHandler = FS{}
	c.RemoteURLHandler = httpClient.Get

	index := NewSpecIndexWithConfig(&rootNode, c)

	assert.Len(t, index.GetReferenceIndexErrors(), 2)
	assert.Equal(t, `invalid URL escape "%$p"`, index.GetReferenceIndexErrors()[0].Error())
	assert.Equal(t, "component 'exisiting.yaml#/paths/~1pet~1%$petId%7D/get/parameters' does not exist in the specification", index.GetReferenceIndexErrors()[1].Error())
}

func TestSpecIndex_Complex_Local_File_Design(t *testing.T) {

	main := `openapi: 3.1.0
paths:
  /anything/circularReference:
    get:
      operationId: circularReferenceGet
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "components.yaml#/components/schemas/validCircularReferenceObject"
  /anything/oneOfCircularReference:
    get:
      operationId: oneOfCircularReferenceGet
      tags:
        - generation
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "components.yaml#/components/schemas/oneOfCircularReferenceObject"`

	components := `components:
  schemas:
    validCircularReferenceObject:
      type: object
      properties:
        circular:
          type: array
          items:
            $ref: "#/components/schemas/validCircularReferenceObject"
    oneOfCircularReferenceObject:
      type: object
      properties:
        child:
          oneOf:
            - $ref: "#/components/schemas/oneOfCircularReferenceObject"
            - $ref: "#/components/schemas/simpleObject"
      required:
        - child
    simpleObject:
      description: "simple"
      type: object
      properties:
        str:
          type: string
          description: "A string property."
          example: "example" `

	_ = os.WriteFile("components.yaml", []byte(components), 0644)

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(main), &rootNode)

	c := CreateOpenAPIIndexConfig()
	index := NewSpecIndexWithConfig(&rootNode, c)

	assert.Len(t, index.GetReferenceIndexErrors(), 2)
	assert.Equal(t, `invalid URL escape "%$p"`, index.GetReferenceIndexErrors()[0].Error())
	assert.Equal(t, "component 'exisiting.yaml#/paths/~1pet~1%$petId%7D/get/parameters' does not exist in the specification", index.GetReferenceIndexErrors()[1].Error())
}
