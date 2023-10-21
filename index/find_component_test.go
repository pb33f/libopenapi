// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"os"
	"testing"
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

	cFile := "../test_specs/first.yaml"
	yml, _ := os.ReadFile(cFile)
	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	cf := CreateOpenAPIIndexConfig()
	cf.AvoidCircularReferenceCheck = true
	cf.BasePath = "../test_specs"

	rolo := NewRolodex(cf)
	rolo.SetRootNode(&rootNode)
	cf.Rolodex = rolo

	fsCfg := LocalFSConfig{
		BaseDirectory: cf.BasePath,
		FileFilters:   []string{"first.yaml", "second.yaml", "third.yaml", "fourth.yaml"},
		DirFS:         os.DirFS(cf.BasePath),
	}

	fileFS, err := NewLocalFSWithConfig(&fsCfg)

	assert.NoError(t, err)
	rolo.AddLocalFS(cf.BasePath, fileFS)

	indexedErr := rolo.IndexTheRolodex()
	rolo.BuildIndexes()

	assert.NoError(t, indexedErr)

	index := rolo.GetRootIndex()

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
	assert.Len(t, index.GetReferenceIndexErrors(), 1)
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

	// create a new config that allows remote lookups.
	cf := &SpecIndexConfig{}
	cf.AllowRemoteLookup = true
	cf.AvoidCircularReferenceCheck = true

	// create a new rolodex
	rolo := NewRolodex(cf)

	// set the rolodex root node to the root node of the spec.
	rolo.SetRootNode(&rootNode)

	// create a new remote fs and set the config for indexing.
	remoteFS, _ := NewRemoteFSWithConfig(cf)

	// add remote filesystem
	rolo.AddRemoteFS("", remoteFS)

	// index the rolodex.
	indexedErr := rolo.IndexTheRolodex()

	assert.NoError(t, indexedErr)

	index := rolo.GetRootIndex()

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

	index := NewSpecIndexWithConfig(&rootNode, c)
	assert.Len(t, index.GetReferenceIndexErrors(), 1)
	assert.Equal(t, "component '#/paths/~1pet~1%$petId%7D/get/parameters' does not exist in the specification", index.GetReferenceIndexErrors()[0].Error())
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

	index := NewSpecIndexWithConfig(&rootNode, c)
	assert.Len(t, index.GetReferenceIndexErrors(), 1)
}
