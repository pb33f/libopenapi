// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
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

	indexedErr := rolo.IndexTheRolodex(context.Background())
	rolo.BuildIndexes()

	assert.NoError(t, indexedErr)

	index := rolo.GetRootIndex()

	assert.Nil(t, index.uri)

	a, _ := index.SearchIndexForReference("second.yaml#/properties/property2")
	b, _ := index.SearchIndexForReference("second.yaml")
	c, _ := index.SearchIndexForReference("fourth.yaml")

	assert.NotNil(t, a)
	assert.NotNil(t, b)
	assert.Nil(t, c)
}

func TestSpecIndex_CheckCircularIndex_NoDirFS(t *testing.T) {
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
		IndexConfig:   cf,
	}

	fileFS, err := NewLocalFSWithConfig(&fsCfg)

	assert.NoError(t, err)
	rolo.AddLocalFS(cf.BasePath, fileFS)

	indexedErr := rolo.IndexTheRolodex(context.Background())
	rolo.BuildIndexes()

	assert.NoError(t, indexedErr)

	index := rolo.GetRootIndex()

	assert.Nil(t, index.uri)

	a, _ := index.SearchIndexForReference("second.yaml#/properties/property2")
	b, _ := index.SearchIndexForReference("second.yaml")
	c, _ := index.SearchIndexForReference("fourth.yaml")
	assert.NotNil(t, a)
	assert.NotNil(t, b)
	assert.Nil(t, c)
}

func TestFindComponent_RolodexFileParseError_Recovery(t *testing.T) {
	badData := "I cannot be parsed: \"I am not a YAML file or a JSON file"
	_ = os.WriteFile("bad.yaml", []byte(badData), 0o644)
	defer os.Remove("bad.yaml")

	badRef := `openapi: 3.1.0
components:
  schemas:
    thing:
      type: object
      properties:
        thong:
          $ref: 'bad.yaml'
`
	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(badRef), &rootNode)

	cf := CreateOpenAPIIndexConfig()
	cf.AvoidCircularReferenceCheck = true
	cf.BasePath = "."

	rolo := NewRolodex(cf)
	rolo.SetRootNode(&rootNode)
	cf.Rolodex = rolo

	fsCfg := LocalFSConfig{
		BaseDirectory: cf.BasePath,
		FileFilters:   []string{"bad.yaml"},
		DirFS:         os.DirFS(cf.BasePath),
	}

	fileFS, err := NewLocalFSWithConfig(&fsCfg)

	assert.NoError(t, err)
	rolo.AddLocalFS(cf.BasePath, fileFS)

	indexedErr := rolo.IndexTheRolodex(context.Background())
	rolo.BuildIndexes()

	// should no longer error
	assert.NoError(t, indexedErr)

	index := rolo.GetRootIndex()

	assert.Nil(t, index.uri)

	// can still be found.
	a, _ := index.SearchIndexForReference("bad.yaml")
	assert.NotNil(t, a)
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

	thing := index.FindComponentInRoot(context.Background(), "#/$splish/$.../slash#$///./")
	assert.Nil(t, thing)
	assert.Len(t, index.GetReferenceIndexErrors(), 0)
}

func TestSpecIndex_FailFindComponentInRoot(t *testing.T) {
	index := NewTestSpecIndex().Load().(*SpecIndex)
	assert.Nil(t, index.FindComponentInRoot(context.Background(), "does it even matter? of course not. no"))
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

	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	var idx *SpecIndex
	done := make(chan struct{})
	go func() {

		// index the rolodex.
		indexedErr := rolo.IndexTheRolodex(ctx)

		assert.NoError(t, indexedErr)
		idx = rolo.GetRootIndex()
		done <- struct{}{}
	}()

	complete := false
	for !complete {
		select {
		case <-ctx.Done():
			complete = true

			break
		case <-done:
			crsParam := idx.GetMappedReferences()["https://schemas.opengis.net/ogcapi/features/part2/1.0/openapi/ogcapi-features-2.yaml#/components/parameters/crs"]
			assert.NotNil(t, crsParam)
			assert.True(t, crsParam.IsRemote)
			assert.Equal(t, "crs", crsParam.Node.Content[1].Value)
			assert.Equal(t, "query", crsParam.Node.Content[3].Value)
			assert.Equal(t, "form", crsParam.Node.Content[9].Value)
			complete = true
		}
	}

	// extract crs param from index

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
	assert.Equal(t, "component `#/paths/~1pet~1%$petId%7D/get/parameters` does not exist in the specification", index.GetReferenceIndexErrors()[0].Error())
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

func TestFindComponent_LookupRolodex_GrabRoot(t *testing.T) {
	spec := `openapi: 3.0.2
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    thang:
      type: object
`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(spec), &rootNode)

	c := CreateOpenAPIIndexConfig()

	index := NewSpecIndexWithConfig(&rootNode, c)
	r := NewRolodex(c)
	index.rolodex = r

	n := index.lookupRolodex(context.Background(), []string{"bingobango"})

	// if the reference is not found, it should return the root.
	assert.NotNil(t, n)
}

func TestFindComponentInRoot_GrabDocRoot(t *testing.T) {
	spec := `openapi: 3.0.2
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    thang:
      type: object
`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(spec), &rootNode)

	c := CreateOpenAPIIndexConfig()

	index := NewSpecIndexWithConfig(&rootNode, c)
	r := NewRolodex(c)
	index.rolodex = r

	n := index.FindComponentInRoot(context.Background(), "#/")

	// if the reference is not found, it should return the root.
	assert.NotNil(t, n)
}

func TestFindComponentInRoot_SimulateWindows(t *testing.T) {
	spec := `openapi: 3.0.2
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    thang:
      type: object
`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(spec), &rootNode)

	c := CreateOpenAPIIndexConfig()

	index := NewSpecIndexWithConfig(&rootNode, c)
	r := NewRolodex(c)
	index.rolodex = r

	n := index.FindComponentInRoot(context.Background(), `C:\windows\you\annoy\me#\components\schemas\thang`)

	assert.NotNil(t, n)
}

func TestFindComponent_LookupRolodex_NoURL(t *testing.T) {
	spec := `openapi: 3.0.2
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    thang:
      type: object
`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(spec), &rootNode)

	c := CreateOpenAPIIndexConfig()

	index := NewSpecIndexWithConfig(&rootNode, c)
	r := NewRolodex(c)
	index.rolodex = r

	n := index.lookupRolodex(context.Background(), nil)

	// no url, no ref.
	assert.Nil(t, n)
}

func TestFindComponent_LookupRolodex_InvalidFile_NoBypass(t *testing.T) {
	spec := `i:am : not a yaml file:`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(spec), &rootNode)

	c := CreateOpenAPIIndexConfig()

	index := NewSpecIndexWithConfig(&rootNode, c)
	r := NewRolodex(c)
	index.rolodex = r

	n := index.lookupRolodex(context.Background(), []string{"bingobango"})

	// if the reference is not found, it should return the root.
	assert.NotNil(t, n)
}

func TestFindComponent_LookupRolodex_WithSpecAbsolutePath(t *testing.T) {
	// Test that triggers line 156: basePath = filepath.Dir(index.specAbsolutePath)
	// This happens when:
	// 1. A relative file reference is used (not absolute, not http)
	// 2. index.specAbsolutePath is set (non-empty)

	spec := `openapi: 3.0.2
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    thang:
      type: object
`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(spec), &rootNode)

	c := CreateOpenAPIIndexConfig()
	c.BasePath = "."

	index := NewSpecIndexWithConfig(&rootNode, c)
	r := NewRolodex(c)
	index.rolodex = r

	// Set specAbsolutePath to trigger the branch at line 156
	index.specAbsolutePath = "/some/absolute/path/to/spec.yaml"

	// Call lookupRolodex with a relative file reference (not absolute, not http)
	// This will hit the branch where specAbsolutePath is used to determine basePath
	n := index.lookupRolodex(context.Background(), []string{"relative_file.yaml"})

	// The file doesn't exist, so it returns nil, but the important thing is
	// that we hit the code path at line 156
	assert.Nil(t, n)
}

func TestFindComponent_LookupRolodex_FindComponentReturnsNil_DebugLog(t *testing.T) {
	// Test that triggers the debug log at lines 241-245 when FindComponent returns nil.
	// This happens when a file exists in the rolodex but the queried component doesn't exist.

	// Create a valid external file with some components
	externalContent := `type: object
properties:
  name:
    type: string
  age:
    type: integer`

	// Write the external file
	err := os.WriteFile("external_schema.yaml", []byte(externalContent), 0o644)
	assert.NoError(t, err)
	defer os.Remove("external_schema.yaml")

	// Create main spec that references a non-existent component in the external file
	mainSpec := `openapi: 3.1.0
components:
  schemas:
    MySchema:
      $ref: 'external_schema.yaml#/components/schemas/NonExistent'`

	var rootNode yaml.Node
	err = yaml.Unmarshal([]byte(mainSpec), &rootNode)
	assert.NoError(t, err)

	// Create a buffer to capture log output
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Create config with debug logger
	cf := CreateOpenAPIIndexConfig()
	cf.BasePath = "."
	cf.Logger = logger
	cf.AvoidCircularReferenceCheck = true

	// Create rolodex
	rolo := NewRolodex(cf)
	rolo.SetRootNode(&rootNode)
	cf.Rolodex = rolo

	// Add local filesystem
	fsCfg := LocalFSConfig{
		BaseDirectory: cf.BasePath,
		FileFilters:   []string{"external_schema.yaml"},
		DirFS:         os.DirFS(cf.BasePath),
		IndexConfig:   cf,
	}
	fileFS, err := NewLocalFSWithConfig(&fsCfg)
	assert.NoError(t, err)
	rolo.AddLocalFS(cf.BasePath, fileFS)

	// Index the rolodex - errors are expected because the component doesn't exist
	_ = rolo.IndexTheRolodex(context.Background())

	index := rolo.GetRootIndex()
	assert.NotNil(t, index)

	// The reference to NonExistent should trigger the debug log
	// because the file exists but the component path doesn't
	logOutput := logBuf.String()
	assert.True(t, strings.Contains(logOutput, "[lookupRolodex] FindComponent returned nil"),
		"Expected debug log about FindComponent returning nil, got: %s", logOutput)
	assert.True(t, strings.Contains(logOutput, "external_schema.yaml"),
		"Expected log to contain the file location")
}

func TestLookupRolodex_SkipExternalRefResolution(t *testing.T) {
	spec := `openapi: 3.0.2
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    thang:
      type: object`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(spec), &rootNode)

	c := CreateOpenAPIIndexConfig()
	c.SkipExternalRefResolution = true

	index := NewSpecIndexWithConfig(&rootNode, c)
	r := NewRolodex(c)
	index.rolodex = r

	// lookupRolodex should return nil immediately when SkipExternalRefResolution is set,
	// without attempting to open files via the rolodex
	n := index.lookupRolodex(context.Background(), []string{"./models/pet.yaml"})
	assert.Nil(t, n, "lookupRolodex should return nil when SkipExternalRefResolution is enabled")

	// Also test with a remote URL reference
	n = index.lookupRolodex(context.Background(), []string{"https://example.com/schemas/pet.yaml"})
	assert.Nil(t, n, "lookupRolodex should return nil for remote refs when SkipExternalRefResolution is enabled")
}
