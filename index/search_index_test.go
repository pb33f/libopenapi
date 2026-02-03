// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestSpecIndex_SearchIndexForReference(t *testing.T) {
	petstore, _ := os.ReadFile("../test_specs/petstorev3.json")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(petstore, &rootNode)

	c := CreateOpenAPIIndexConfig()
	idx := NewSpecIndexWithConfig(&rootNode, c)

	ref, _ := idx.SearchIndexForReference("#/components/schemas/Pet")
	assert.NotNil(t, ref)
}

func TestSpecIndex_SearchIndexForReferenceWithContext(t *testing.T) {
	petstore, _ := os.ReadFile("../test_specs/petstorev3.json")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(petstore, &rootNode)

	c := CreateOpenAPIIndexConfig()
	idx := NewSpecIndexWithConfig(&rootNode, c)

	ref, _, _ := idx.SearchIndexForReferenceWithContext(context.Background(), "#/components/schemas/Pet")
	assert.NotNil(t, ref)

	assert.NotNil(t, idx.GetRootNode())
	idx.SetRootNode(nil)
	assert.Nil(t, idx.GetRootNode())

}

func TestSearchIndexForReferenceByReferenceWithContext_SchemaIdBaseFromContext(t *testing.T) {
	spec := `openapi: "3.1.0"
info:
  title: Test
  version: "1.0.0"
components:
  schemas:
    Pet:
      $id: "https://example.com/schemas/pet.json"
      type: string
`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(spec), &rootNode)

	idx := NewSpecIndexWithConfig(&rootNode, CreateClosedAPIIndexConfig())
	scope := NewSchemaIdScope("https://example.com/schemas/")
	ctx := WithSchemaIdScope(context.Background(), scope)

	searchRef := &Reference{FullDefinition: "pet.json"}
	found, _, _ := idx.SearchIndexForReferenceByReferenceWithContext(ctx, searchRef)
	if assert.NotNil(t, found) {
		assert.Equal(t, "https://example.com/schemas/pet.json", found.FullDefinition)
	}
}

func TestSearchIndexForReferenceByReferenceWithContext_CacheHitOnNormalizedRef(t *testing.T) {
	spec := `openapi: "3.1.0"
info:
  title: Test
  version: "1.0.0"
components:
  schemas:
    Pet:
      $id: "https://example.com/schemas/pet.json"
      type: string
`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(spec), &rootNode)

	cfg := CreateClosedAPIIndexConfig()
	cfg.SpecAbsolutePath = "https://example.com/openapi.yaml"
	idx := NewSpecIndexWithConfig(&rootNode, cfg)

	cachedRef := &Reference{
		FullDefinition: "https://example.com/schemas/pet.json",
		Index:          idx,
		RemoteLocation: cfg.SpecAbsolutePath,
	}
	idx.cache.Store(cachedRef.FullDefinition, cachedRef)

	searchRef := &Reference{
		FullDefinition: "pet.json",
		SchemaIdBase:   "https://example.com/schemas/",
	}
	found, foundIdx, _ := idx.SearchIndexForReferenceByReferenceWithContext(context.Background(), searchRef)
	assert.Equal(t, cachedRef, found)
	assert.Equal(t, idx, foundIdx)
}

func TestSearchIndexForReferenceByReferenceWithContext_PathRefUsesRawRef(t *testing.T) {
	spec := `openapi: "3.1.0"
info:
  title: Test
  version: "1.0.0"
components:
  schemas:
    Integer:
      $id: "https://other.com/schemas/mixins/integer"
      type: integer
`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(spec), &rootNode)

	idx := NewSpecIndexWithConfig(&rootNode, CreateClosedAPIIndexConfig())

	searchRef := &Reference{
		FullDefinition: "/schemas/mixins/integer",
		SchemaIdBase:   "https://example.com/schemas/examples/non-negative-integer",
	}
	found, _, _ := idx.SearchIndexForReferenceByReferenceWithContext(context.Background(), searchRef)
	if assert.NotNil(t, found) {
		assert.Equal(t, "https://other.com/schemas/mixins/integer", found.FullDefinition)
	}

	if cached, ok := idx.cache.Load("/schemas/mixins/integer"); assert.True(t, ok) {
		assert.Equal(t, found, cached)
	}
	if cached, ok := idx.cache.Load("https://example.com/schemas/mixins/integer"); assert.True(t, ok) {
		assert.Equal(t, found, cached)
	}
}

// TestSearchIndexForReference_LastDitchRolodexFallback tests the last-ditch effort
// code path where a reference is found by iterating through rolodex indexes
// after all other lookup methods fail.
func TestSearchIndexForReference_LastDitchRolodexFallback(t *testing.T) {
	// Primary index with NO components - searches will fail here
	primarySpec := `openapi: 3.0.1
info:
  title: Primary
  version: "1.0"`

	var primaryRoot yaml.Node
	_ = yaml.Unmarshal([]byte(primarySpec), &primaryRoot)

	c := CreateOpenAPIIndexConfig()
	primaryIdx := NewSpecIndexWithConfig(&primaryRoot, c)

	// Secondary index WITH the component we want to find
	secondarySpec := `openapi: 3.0.1
info:
  title: Secondary
  version: "1.0"
components:
  schemas:
    Pet:
      type: object
      properties:
        name:
          type: string`

	var secondaryRoot yaml.Node
	_ = yaml.Unmarshal([]byte(secondarySpec), &secondaryRoot)

	secondaryIdx := NewSpecIndexWithConfig(&secondaryRoot, c)

	// Create rolodex and add secondary index
	rolo := NewRolodex(c)
	rolo.AddIndex(secondaryIdx)

	// Set rolodex on primary index
	primaryIdx.SetRolodex(rolo)

	// Search for reference that:
	// 1. Doesn't exist in primary index's allMappedRefs
	// 2. Has roloLookup = "" (simple ref format)
	// 3. Should be found via last-ditch rolodex iteration
	ref, idx := primaryIdx.SearchIndexForReference("#/components/schemas/Pet")

	assert.NotNil(t, ref, "Reference should be found via rolodex fallback")
	assert.NotNil(t, idx, "Index should be returned")
	assert.Equal(t, "Pet", ref.Name)
}

func TestSearchIndexForReference_RolodexSuffixMatch(t *testing.T) {
	tempDir := t.TempDir()
	externalDir := filepath.Join(tempDir, "subdir")
	err := os.MkdirAll(externalDir, 0o755)
	assert.NoError(t, err)

	externalPath := filepath.Join(externalDir, "external.yaml")
	externalSpec := []byte(`openapi: "3.0.0"
info:
  title: External
  version: "1.0.0"
paths: {}`)
	err = os.WriteFile(externalPath, externalSpec, 0o644)
	assert.NoError(t, err)

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(`openapi: "3.0.0"
info:
  title: Root
  version: "1.0.0"
paths: {}`), &rootNode)

	config := CreateOpenAPIIndexConfig()
	config.SpecAbsolutePath = filepath.Join(tempDir, "root.yaml")
	config.SpecFilePath = config.SpecAbsolutePath
	config.BasePath = tempDir

	rolo := NewRolodex(config)
	localFS, fsErr := NewLocalFSWithConfig(&LocalFSConfig{
		BaseDirectory: tempDir,
		IndexConfig:   config,
	})
	assert.NoError(t, fsErr)
	rolo.AddLocalFS(tempDir, localFS)

	idx := NewSpecIndexWithConfig(&rootNode, config)
	idx.SetRolodex(rolo)

	ref := filepath.ToSlash(filepath.Join("subdir", "external.yaml"))
	found, _ := idx.SearchIndexForReference(ref)

	assert.NotNil(t, found)
	assert.True(t, found.IsRemote)
	assert.Equal(t, "external.yaml", filepath.Base(found.FullDefinition))
}

func TestIsFileBeingIndexed_HTTPPathMatch(t *testing.T) {
	// Test that HTTP paths match when the path portion is the same
	ctx := context.Background()

	// Add an HTTP file to the indexing context
	files := make(map[string]bool)
	files["https://example.com/schemas/pet.yaml"] = true
	ctx = context.WithValue(ctx, IndexingFilesKey, files)

	// Same path, different host - should match
	assert.True(t, IsFileBeingIndexed(ctx, "https://different-host.com/schemas/pet.yaml"))

	// Same exact URL - should match
	assert.True(t, IsFileBeingIndexed(ctx, "https://example.com/schemas/pet.yaml"))

	// Different path - should not match
	assert.False(t, IsFileBeingIndexed(ctx, "https://example.com/other/file.yaml"))
}
