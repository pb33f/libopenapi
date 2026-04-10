// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

type countingFS struct {
	opens int
	err   error
}

func (c *countingFS) Open(name string) (fs.File, error) {
	c.opens++
	if c.err != nil {
		return nil, c.err
	}
	return nil, fs.ErrNotExist
}

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
		assert.Equal(t, "#/components/schemas/Pet", found.FullDefinition)
		assert.Equal(t, "#/components/schemas/Pet", found.Definition)
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
		assert.Equal(t, "#/components/schemas/Integer", found.FullDefinition)
		assert.Equal(t, "#/components/schemas/Integer", found.Definition)
	}

	if cached, ok := idx.cache.Load("/schemas/mixins/integer"); assert.True(t, ok) {
		assert.Equal(t, found, cached)
	}
	if cached, ok := idx.cache.Load("https://example.com/schemas/mixins/integer"); assert.True(t, ok) {
		assert.Equal(t, found, cached)
	}
}

func TestSearchIndexForReferenceByReferenceWithContext_LocalSchemaIdCanonicalizesTarget(t *testing.T) {
	spec := `{
  "openapi": "3.2.0",
  "info": { "title": "Test", "version": "0" },
  "paths": {
    "/widgets": {
      "post": {
        "operationId": "postWidget",
        "requestBody": {
          "content": {
            "application/json": {
              "schema": { "$ref": "#/components/schemas/widget" }
            }
          }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "_0_xxxx.schema.json": {
        "$schema": "https://json-schema.org/draft/2020-12/schema",
        "$id": "https://example.com/widget/widget.schema.json",
        "title": "Widget",
        "type": "object"
      },
      "widget": {
        "additionalProperties": false,
        "properties": {
          "body": {
            "$ref": "https://example.com/widget/widget.schema.json"
          }
        },
        "type": "object"
      }
    }
  }
}`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(spec), &rootNode)
	assert.NoError(t, err)

	cfg := CreateOpenAPIIndexConfig()
	cfg.SpecAbsolutePath = "https://example.com/openapi.json"
	rolodex := NewRolodex(cfg)
	remote := &countingFS{err: fs.ErrNotExist}
	rolodex.AddRemoteFS("https://example.com", remote)

	idx := NewSpecIndexWithConfig(&rootNode, cfg)
	rolodex.SetRootIndex(idx)

	assert.Equal(t, 0, remote.opens, "local $id match should not trigger remote lookup")
	assert.Empty(t, idx.GetReferenceIndexErrors())

	var resolvedViaID *ReferenceMapped
	for _, mapped := range idx.GetMappedReferencesSequenced() {
		if mapped == nil || mapped.OriginalReference == nil {
			continue
		}
		if mapped.OriginalReference.RawRef == "https://example.com/widget/widget.schema.json" {
			resolvedViaID = mapped
			break
		}
	}
	if assert.NotNil(t, resolvedViaID) {
		assert.Equal(t, "https://example.com/widget/widget.schema.json", resolvedViaID.OriginalReference.RawRef)
		assert.Equal(t, "#/components/schemas/_0_xxxx.schema.json", resolvedViaID.Reference.Definition)
		assert.Equal(t,
			"https://example.com/openapi.json#/components/schemas/_0_xxxx.schema.json",
			resolvedViaID.Reference.FullDefinition,
		)
	}

	mappedRefs := idx.GetMappedReferences()
	target := mappedRefs["https://example.com/openapi.json#/components/schemas/_0_xxxx.schema.json"]
	if assert.NotNil(t, target) {
		assert.Equal(t, "#/components/schemas/_0_xxxx.schema.json", target.Definition)
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

// TestSearchIndexForReference_RootIndexFallback tests the last-ditch code path
// where a child index cannot resolve a ref like /path/to/file.yaml#/components/schemas/Name
// but the component exists in the root index. The fix adds root index to the search.
// Uses real files on disk so the rolodex can open them, matching production behavior.
func TestSearchIndexForReference_RootIndexFallback(t *testing.T) {
	tmpDir := t.TempDir()

	// Root spec WITH the Workspace schema
	rootSpec := `openapi: 3.0.1
info:
  title: Root
  version: "1.0"
components:
  schemas:
    Workspace:
      type: object
      properties:
        name:
          type: string
paths:
  /workspaces:
    $ref: './paths/list.yaml'`

	// Child spec (a paths file) that does NOT have the Workspace schema
	childSpec := `get:
  summary: List workspaces
  responses:
    '200':
      description: OK`

	// write files to disk
	err := os.MkdirAll(filepath.Join(tmpDir, "paths"), 0o755)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "root.yaml"), []byte(rootSpec), 0o644)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "paths", "list.yaml"), []byte(childSpec), 0o644)
	assert.NoError(t, err)

	rootPath := filepath.Join(tmpDir, "root.yaml")
	childPath := filepath.Join(tmpDir, "paths", "list.yaml")

	// build root index
	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(rootSpec), &rootNode)

	c := CreateOpenAPIIndexConfig()
	c.SpecAbsolutePath = rootPath
	c.BasePath = tmpDir
	rootIdx := NewSpecIndexWithConfig(&rootNode, c)

	// build child index
	var childRoot yaml.Node
	_ = yaml.Unmarshal([]byte(childSpec), &childRoot)

	childConfig := CreateOpenAPIIndexConfig()
	childConfig.SpecAbsolutePath = childPath
	childConfig.BasePath = filepath.Join(tmpDir, "paths")
	childIdx := NewSpecIndexWithConfig(&childRoot, childConfig)

	// create rolodex with filesystem access
	rolo := NewRolodex(c)
	localFS, fsErr := NewLocalFSWithConfig(&LocalFSConfig{
		BaseDirectory: tmpDir,
		IndexConfig:   c,
	})
	assert.NoError(t, fsErr)
	rolo.AddLocalFS(tmpDir, localFS)
	rolo.SetRootIndex(rootIdx)
	rolo.AddIndex(childIdx)
	childIdx.SetRolodex(rolo)

	// search for a ref that has the child file path + fragment for a component in the ROOT.
	// this simulates: resolver expanded #/components/schemas/Workspace in list.yaml to
	// /abs/path/to/paths/list.yaml#/components/schemas/Workspace
	searchRef := childPath + "#/components/schemas/Workspace"
	ref, foundIdx := childIdx.SearchIndexForReference(searchRef)

	assert.NotNil(t, ref, "reference should be found via root index fallback")
	assert.NotNil(t, foundIdx, "index should be returned")
	if ref != nil {
		assert.Equal(t, "Workspace", ref.Name)
	}
}

// TestSearchIndexForReference_RootIndexFallback_Negative verifies that the root index
// fallback returns nil when the component genuinely doesn't exist anywhere.
func TestSearchIndexForReference_RootIndexFallback_Negative(t *testing.T) {
	tmpDir := t.TempDir()

	rootSpec := `openapi: 3.0.1
info:
  title: Root
  version: "1.0"
components:
  schemas:
    Workspace:
      type: object
paths:
  /workspaces:
    $ref: './paths/list.yaml'`

	childSpec := `get:
  summary: List workspaces
  responses:
    '200':
      description: OK`

	err := os.MkdirAll(filepath.Join(tmpDir, "paths"), 0o755)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "root.yaml"), []byte(rootSpec), 0o644)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "paths", "list.yaml"), []byte(childSpec), 0o644)
	assert.NoError(t, err)

	rootPath := filepath.Join(tmpDir, "root.yaml")
	childPath := filepath.Join(tmpDir, "paths", "list.yaml")

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(rootSpec), &rootNode)
	c := CreateOpenAPIIndexConfig()
	c.SpecAbsolutePath = rootPath
	c.BasePath = tmpDir
	rootIdx := NewSpecIndexWithConfig(&rootNode, c)

	var childRoot yaml.Node
	_ = yaml.Unmarshal([]byte(childSpec), &childRoot)
	childConfig := CreateOpenAPIIndexConfig()
	childConfig.SpecAbsolutePath = childPath
	childConfig.BasePath = filepath.Join(tmpDir, "paths")
	childIdx := NewSpecIndexWithConfig(&childRoot, childConfig)

	rolo := NewRolodex(c)
	localFS, fsErr := NewLocalFSWithConfig(&LocalFSConfig{
		BaseDirectory: tmpDir,
		IndexConfig:   c,
	})
	assert.NoError(t, fsErr)
	rolo.AddLocalFS(tmpDir, localFS)
	rolo.SetRootIndex(rootIdx)
	rolo.AddIndex(childIdx)
	childIdx.SetRolodex(rolo)

	// search for a component that doesn't exist anywhere
	searchRef := childPath + "#/components/schemas/NonExistent"
	ref, foundIdx := childIdx.SearchIndexForReference(searchRef)

	assert.Nil(t, ref, "non-existent component should not be found")
	assert.Equal(t, childIdx, foundIdx, "should return the searching index when not found")
}

// TestSearchIndexForReference_RootIndexSelfGuard verifies that when the searching
// index IS the root index, it does not recurse into itself (the rootIdx != index guard).
func TestSearchIndexForReference_RootIndexSelfGuard(t *testing.T) {
	rootSpec := `openapi: 3.0.1
info:
  title: Root
  version: "1.0"
paths: {}`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(rootSpec), &rootNode)

	c := CreateOpenAPIIndexConfig()
	c.SpecAbsolutePath = "/tmp/root.yaml"
	rootIdx := NewSpecIndexWithConfig(&rootNode, c)

	rolo := NewRolodex(c)
	rolo.SetRootIndex(rootIdx)
	rootIdx.SetRolodex(rolo)

	// search from the root index for something that doesn't exist.
	// the rootIdx == index guard should prevent infinite recursion.
	ref, foundIdx := rootIdx.SearchIndexForReference("/tmp/root.yaml#/components/schemas/Missing")

	assert.Nil(t, ref, "should not find non-existent component")
	assert.Equal(t, rootIdx, foundIdx, "should return the searching index")
}

// TestSearchIndexForReference_NoLoggerStillFindsViaRootFallback verifies that
// the root index fallback works even when no logger is set (decoupled from logger guard).
func TestSearchIndexForReference_NoLoggerStillFindsViaRootFallback(t *testing.T) {
	tmpDir := t.TempDir()

	rootSpec := `openapi: 3.0.1
info:
  title: Root
  version: "1.0"
components:
  schemas:
    Widget:
      type: object
      properties:
        id:
          type: integer
paths: {}`

	childSpec := `get:
  summary: List widgets
  responses:
    '200':
      description: OK`

	err := os.MkdirAll(filepath.Join(tmpDir, "paths"), 0o755)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "root.yaml"), []byte(rootSpec), 0o644)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "paths", "widgets.yaml"), []byte(childSpec), 0o644)
	assert.NoError(t, err)

	rootPath := filepath.Join(tmpDir, "root.yaml")
	childPath := filepath.Join(tmpDir, "paths", "widgets.yaml")

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(rootSpec), &rootNode)

	// no logger set on config
	c := CreateOpenAPIIndexConfig()
	c.SpecAbsolutePath = rootPath
	c.BasePath = tmpDir
	rootIdx := NewSpecIndexWithConfig(&rootNode, c)

	var childRoot yaml.Node
	_ = yaml.Unmarshal([]byte(childSpec), &childRoot)
	childConfig := CreateOpenAPIIndexConfig()
	childConfig.SpecAbsolutePath = childPath
	childConfig.BasePath = filepath.Join(tmpDir, "paths")
	childIdx := NewSpecIndexWithConfig(&childRoot, childConfig)

	rolo := NewRolodex(c)
	localFS, fsErr := NewLocalFSWithConfig(&LocalFSConfig{
		BaseDirectory: tmpDir,
		IndexConfig:   c,
	})
	assert.NoError(t, fsErr)
	rolo.AddLocalFS(tmpDir, localFS)
	rolo.SetRootIndex(rootIdx)
	rolo.AddIndex(childIdx)
	childIdx.SetRolodex(rolo)

	// explicitly ensure no logger is set
	childIdx.logger = nil

	// search for a component in root from the child index — should still work
	searchRef := childPath + "#/components/schemas/Widget"
	ref, foundIdx := childIdx.SearchIndexForReference(searchRef)

	assert.NotNil(t, ref, "should find via root index fallback even without logger")
	assert.NotNil(t, foundIdx, "index should be returned")
	if ref != nil {
		assert.Equal(t, "Widget", ref.Name)
	}
}

// TestSearchIndexForReference_ErrorLogWithRolodex verifies that the error log
// includes correct structured fields when a reference cannot be found.
func TestSearchIndexForReference_ErrorLogWithRolodex(t *testing.T) {
	var logBuf strings.Builder
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	rootSpec := `openapi: 3.0.1
info:
  title: Root
  version: "1.0"
paths: {}`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(rootSpec), &rootNode)

	c := CreateOpenAPIIndexConfig()
	c.SpecAbsolutePath = "/tmp/test-root.yaml"
	c.Logger = logger
	idx := NewSpecIndexWithConfig(&rootNode, c)

	// set up rolodex with root index but no child indexes
	rolo := NewRolodex(c)
	rolo.SetRootIndex(idx)
	idx.SetRolodex(rolo)

	// search for something that doesn't exist — triggers error log
	ref, foundIdx := idx.SearchIndexForReference("#/components/schemas/Ghost")

	assert.Nil(t, ref)
	assert.Equal(t, idx, foundIdx)

	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "unable to locate reference anywhere in the rolodex")
	assert.Contains(t, logOutput, "indexPath")
	assert.Contains(t, logOutput, "hasRolodex")
	assert.Contains(t, logOutput, "rolodexIndexCount")
	assert.Contains(t, logOutput, "rootIndexPath")
}

// TestSearchIndexForReference_ErrorLogWithoutRolodex verifies the error log
// uses default values when no rolodex is set.
func TestSearchIndexForReference_ErrorLogWithoutRolodex(t *testing.T) {
	var logBuf strings.Builder
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	spec := `openapi: 3.0.1
info:
  title: Test
  version: "1.0"
paths: {}`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(spec), &rootNode)

	c := CreateOpenAPIIndexConfig()
	c.Logger = logger
	idx := NewSpecIndexWithConfig(&rootNode, c)

	// no rolodex set — should log with defaults
	ref, foundIdx := idx.SearchIndexForReference("#/components/schemas/Missing")

	assert.Nil(t, ref)
	assert.Equal(t, idx, foundIdx)

	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "unable to locate reference anywhere in the rolodex")
	assert.Contains(t, logOutput, "rolodexIndexCount=-1")
	assert.Contains(t, logOutput, "rootIndexPath=<nil>")
}

// TestSearchIndexForReference_NoLoggerNoRolodex verifies that when neither
// logger nor rolodex is set, the function returns nil without panicking.
func TestSearchIndexForReference_NoLoggerNoRolodex(t *testing.T) {
	spec := `openapi: 3.0.1
info:
  title: Test
  version: "1.0"
paths: {}`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(spec), &rootNode)

	c := CreateOpenAPIIndexConfig()
	idx := NewSpecIndexWithConfig(&rootNode, c)
	idx.logger = nil

	// no rolodex, no logger — should return nil without panicking
	ref, foundIdx := idx.SearchIndexForReference("#/components/schemas/Phantom")

	assert.Nil(t, ref)
	assert.Equal(t, idx, foundIdx)
}

// TestSearchIndexForReference_RootFallbackFullRefMatch verifies that the root
// index fallback finds a component via the full ref (before fragment extraction).
// This covers the branch where rootIdx.FindComponent succeeds on the first try.
func TestSearchIndexForReference_RootFallbackFullRefMatch(t *testing.T) {
	rootSpec := `openapi: 3.0.1
info:
  title: Root
  version: "1.0"
components:
  schemas:
    Order:
      type: object
      properties:
        id:
          type: integer
paths: {}`

	childSpec := `get:
  summary: Get order
  responses:
    '200':
      description: OK`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(rootSpec), &rootNode)

	c := CreateOpenAPIIndexConfig()
	c.SpecAbsolutePath = "/tmp/root.yaml"
	c.BasePath = "/tmp"
	rootIdx := NewSpecIndexWithConfig(&rootNode, c)

	var childNode yaml.Node
	_ = yaml.Unmarshal([]byte(childSpec), &childNode)

	childConfig := CreateOpenAPIIndexConfig()
	childConfig.SpecAbsolutePath = "/tmp/paths/orders.yaml"
	childConfig.BasePath = "/tmp/paths"
	childIdx := NewSpecIndexWithConfig(&childNode, childConfig)

	rolo := NewRolodex(c)
	rolo.SetRootIndex(rootIdx)
	rolo.AddIndex(childIdx)
	childIdx.SetRolodex(rolo)

	// search using a simple #/components/schemas/Order ref from the child index.
	// this ref won't be found in the child (no components), won't be found in
	// rolodex indexes (child has no components), but WILL be found in root
	// via FindComponent with the full ref on the first try (no fragment extraction needed).
	ref, foundIdx := childIdx.SearchIndexForReference("#/components/schemas/Order")

	assert.NotNil(t, ref, "should find via root index full ref match")
	assert.NotNil(t, foundIdx, "index should be returned")
	if ref != nil {
		assert.Equal(t, "Order", ref.Name)
	}
}

// TestSearchIndexForReference_RootFallbackNoFragment verifies that the fragment
// extraction path is skipped when the ref doesn't contain "#/".
func TestSearchIndexForReference_RootFallbackNoFragment(t *testing.T) {
	rootSpec := `openapi: 3.0.1
info:
  title: Root
  version: "1.0"
paths: {}`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(rootSpec), &rootNode)

	c := CreateOpenAPIIndexConfig()
	c.SpecAbsolutePath = "/tmp/root.yaml"
	rootIdx := NewSpecIndexWithConfig(&rootNode, c)

	childSpec := `get:
  summary: Test
  responses:
    '200':
      description: OK`

	var childNode yaml.Node
	_ = yaml.Unmarshal([]byte(childSpec), &childNode)

	childConfig := CreateOpenAPIIndexConfig()
	childConfig.SpecAbsolutePath = "/tmp/child.yaml"
	childIdx := NewSpecIndexWithConfig(&childNode, childConfig)

	rolo := NewRolodex(c)
	rolo.SetRootIndex(rootIdx)
	rolo.AddIndex(childIdx)
	childIdx.SetRolodex(rolo)

	// ref with no "#/" — fragment extraction should be skipped
	ref, foundIdx := childIdx.SearchIndexForReference("some-ref-without-fragment")

	assert.Nil(t, ref)
	assert.Equal(t, childIdx, foundIdx)
}

// TestSearchIndexForReference_RolodexWithNilRootIndex verifies behavior when
// the rolodex exists but has no root index set.
func TestSearchIndexForReference_RolodexWithNilRootIndex(t *testing.T) {
	var logBuf strings.Builder
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	spec := `openapi: 3.0.1
info:
  title: Test
  version: "1.0"
paths: {}`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(spec), &rootNode)

	c := CreateOpenAPIIndexConfig()
	c.Logger = logger
	idx := NewSpecIndexWithConfig(&rootNode, c)

	// rolodex without a root index
	rolo := NewRolodex(c)
	idx.SetRolodex(rolo)

	// should skip root index fallback, log error with rolodex fields but nil root
	ref, foundIdx := idx.SearchIndexForReference("#/components/schemas/Missing")

	assert.Nil(t, ref)
	assert.Equal(t, idx, foundIdx)

	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "unable to locate reference anywhere in the rolodex")
	assert.Contains(t, logOutput, "rootIndexPath=<nil>")
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

func TestIsFileBeingIndexed_HTTPMatchesLocalFilename(t *testing.T) {
	ctx := context.Background()

	files := map[string]bool{
		"/tmp/specs/pet.yaml": true,
	}
	ctx = context.WithValue(ctx, IndexingFilesKey, files)

	assert.True(t, IsFileBeingIndexed(ctx, "https://different-host.com/schemas/pet.yaml"))
	assert.False(t, IsFileBeingIndexed(ctx, "https://different-host.com/schemas/cat.yaml"))
}
