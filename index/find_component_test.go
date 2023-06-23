// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
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
	assert.NotNil(t, index.children[0].uri)
	assert.NotNil(t, index.children[0].children[0].uri)
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

func TestSpecIndex_LocateRemoteDocsWithNoBaseURLSupplied(t *testing.T) {
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
	index := NewSpecIndexWithConfig(&rootNode, c)

	// extract crs param from index
	crsParam := index.GetMappedReferences()["https://schemas.opengis.net/ogcapi/features/part2/1.0/openapi/ogcapi-features-2.yaml#/components/parameters/crs"]
	assert.NotNil(t, crsParam)
	assert.True(t, crsParam.IsRemote)
	assert.Equal(t, "crs", crsParam.Node.Content[1].Value)
	assert.Equal(t, "query", crsParam.Node.Content[3].Value)
	assert.Equal(t, "form", crsParam.Node.Content[9].Value)
}

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
