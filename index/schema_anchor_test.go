// Copyright 2023-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

const schemaAnchorSpec = `openapi: 3.1.0
info: {title: anchors, version: '1'}
paths: {}
components:
  schemas:
    A:
      $anchor: foo
      type: string
    First:
      $anchor: dup
      type: boolean
    Second:
      $anchor: dup
      type: number
    Invalid:
      $anchor: 1bad
      type: string
    Example:
      type: object
      example:
        $anchor: payload
    R:
      $id: https://example.com/r
      type: object
      $defs:
        X: {$anchor: bar, type: integer}
      properties:
        scoped: {$ref: '#bar'}
        absolute: {$ref: 'https://example.com/r#bar'}
    Self:
      $id: https://example.com/self
      $anchor: self
      type: array
    B:
      type: object
      properties:
        local: {$ref: '#foo'}
        outside: {$ref: 'https://example.com/r#bar'}
`

func schemaNodeType(node *yaml.Node) string {
	for i := 0; node != nil && i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == "type" {
			return node.Content[i+1].Value
		}
	}
	return ""
}

func TestSchemaAnchorResolution(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(schemaAnchorSpec), &root))
	idx := NewSpecIndexWithConfig(&root, CreateClosedAPIIndexConfig())

	for _, tc := range []struct{ name, ref, kind string }{
		{"document anchor", "#foo", "string"},
		{"anchor in $id resource", "https://example.com/r#bar", "integer"},
		{"anchor beside $id", "https://example.com/self#self", "array"},
		{"first duplicate wins", "#dup", "boolean"},
		{"scoped anchor outside its resource", "#bar", ""},
		{"document anchor under an $id", "https://example.com/r#foo", ""},
		{"invalid name", "#1bad", ""},
		{"example payload", "#payload", ""},
		{"undeclared", "#missing", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, lookup := range []struct {
				name string
				call func() *Reference
			}{
				{"registry", func() *Reference { return idx.ResolveRefViaSchemaId(tc.ref) }},
				{"component", func() *Reference { return idx.FindComponent(context.Background(), tc.ref) }},
				{"search", func() *Reference {
					ref, _ := idx.SearchIndexForReference(tc.ref)
					return ref
				}},
			} {
				t.Run(lookup.name, func(t *testing.T) {
					ref := lookup.call()
					if tc.kind == "" {
						assert.Nil(t, ref)
						return
					}
					require.NotNil(t, ref)
					assert.Equal(t, tc.kind, schemaNodeType(ref.Node))
				})
			}
		})
	}
}

func TestSchemaAnchorResolvedReference(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(schemaAnchorSpec), &root))
	idx := NewSpecIndexWithConfig(&root, CreateClosedAPIIndexConfig())

	ref := idx.ResolveRefViaSchemaId("#foo")
	require.NotNil(t, ref)
	assert.Equal(t, "A", ref.Name)
	assert.Equal(t, "#/components/schemas/A", ref.Definition)
	assert.Equal(t, "#foo", ref.RawRef)
	assert.Empty(t, ref.SchemaIdBase)

	// an anchor inside an $id resource keeps that resource as the base for its own refs.
	ref = idx.ResolveRefViaSchemaId("https://example.com/r#bar")
	require.NotNil(t, ref)
	assert.Equal(t, "X", ref.Name)
	assert.Equal(t, "#/components/schemas/R/$defs/X", ref.Definition)
	assert.Equal(t, "https://example.com/r", ref.SchemaIdBase)
}

func TestSchemaAnchorExtractedRefs(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(schemaAnchorSpec), &root))
	idx := NewSpecIndexWithConfig(&root, CreateClosedAPIIndexConfig())

	assert.Empty(t, idx.GetReferenceIndexErrors())
	mapped := idx.GetMappedReferences()
	require.Contains(t, mapped, "#/components/schemas/A")
	assert.Equal(t, "#foo", mapped["#/components/schemas/A"].RawRef)
	require.Contains(t, mapped, "#/components/schemas/R/$defs/X")
	assert.Equal(t, "integer", schemaNodeType(mapped["#/components/schemas/R/$defs/X"].Node))
}

func TestSchemaAnchorResolver(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(schemaAnchorSpec), &root))
	idx := NewSpecIndexWithConfig(&root, CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.Empty(t, resolver.Resolve())
	assert.Empty(t, resolver.GetResolvingErrors())
}

func TestSchemaAnchorAcrossIndexes(t *testing.T) {
	const shared = `components:
  schemas:
    S:
      $id: https://example.com/shared
      $defs:
        Name: {$anchor: name, type: string}
`
	const local = `openapi: 3.1.0
info: {title: anchors, version: '1'}
paths: {}
components:
  schemas:
    Local: {$anchor: name, type: integer}
`
	cfg := CreateClosedAPIIndexConfig()
	rolodex := NewRolodex(cfg)

	var sharedRoot, localRoot yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(shared), &sharedRoot))
	require.NoError(t, yaml.Unmarshal([]byte(local), &localRoot))

	sharedCfg := CreateClosedAPIIndexConfig()
	sharedCfg.SpecAbsolutePath = "/specs/shared.yaml"
	sharedIdx := NewSpecIndexWithConfig(&sharedRoot, sharedCfg)
	rolodex.AddExternalIndex(sharedIdx, "/specs/shared.yaml")

	localCfg := CreateClosedAPIIndexConfig()
	localCfg.SpecAbsolutePath = "/specs/root.yaml"
	localIdx := NewSpecIndexWithConfig(&localRoot, localCfg)
	localIdx.rolodex = rolodex

	ref := localIdx.ResolveRefViaSchemaId("https://example.com/shared#name")
	require.NotNil(t, ref)
	assert.Equal(t, "string", schemaNodeType(ref.Node))
	assert.True(t, ref.IsRemote)
	assert.Same(t, sharedIdx, ref.Index)

	// a document anchor of the same name stays local.
	ref = localIdx.ResolveRefViaSchemaId("#name")
	require.NotNil(t, ref)
	assert.Equal(t, "integer", schemaNodeType(ref.Node))
	assert.False(t, ref.IsRemote)

	// first registration wins in the global registry too.
	var otherRoot yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(strings.Replace(shared, "type: string", "type: integer", 1)), &otherRoot))
	otherCfg := CreateClosedAPIIndexConfig()
	otherCfg.SpecAbsolutePath = "/specs/other.yaml"
	rolodex.AddExternalIndex(NewSpecIndexWithConfig(&otherRoot, otherCfg), "/specs/other.yaml")
	ref = localIdx.ResolveRefViaSchemaId("https://example.com/shared#name")
	require.NotNil(t, ref)
	assert.Same(t, sharedIdx, ref.Index)
}

// the same "#bar" in different resources points at different schemas.
func TestSchemaAnchorScopedByResource(t *testing.T) {
	const spec = `openapi: 3.1.0
info: {title: anchors, version: '1'}
paths: {}
components:
  schemas:
    R1:
      $id: https://example.com/r1
      $defs: {X: {$anchor: bar, type: integer}}
      properties: {p: {$ref: '#bar'}}
    D: {$anchor: bar, type: boolean}
    U:
      properties: {p: {$ref: '#bar'}}
    R2:
      $id: https://example.com/r2
      $defs: {X: {$anchor: bar, type: string}}
      properties: {p: {$ref: '#bar'}}
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(spec), &root))
	cfg := CreateClosedAPIIndexConfig()
	cfg.SpecAbsolutePath = "/specs/root.yaml"
	idx := NewSpecIndexWithConfig(&root, cfg)

	assert.Empty(t, idx.GetReferenceIndexErrors())
	mapped := idx.GetMappedReferences()
	for def, kind := range map[string]string{
		"/specs/root.yaml#/components/schemas/R1/$defs/X": "integer",
		"/specs/root.yaml#/components/schemas/D":          "boolean",
		"/specs/root.yaml#/components/schemas/R2/$defs/X": "string",
	} {
		require.Contains(t, mapped, def)
		assert.Equal(t, kind, schemaNodeType(mapped[def].Node), def)
	}

	// repeat lookups must not share a cache entry across resources.
	for _, tc := range []struct{ base, kind string }{
		{"https://example.com/r1", "integer"},
		{"https://example.com/r2", "string"},
		{"", "boolean"},
		{"https://example.com/r1", "integer"},
	} {
		ref, _, _ := idx.SearchIndexForReferenceByReferenceWithContext(context.Background(),
			&Reference{FullDefinition: "#bar", RawRef: "#bar", SchemaIdBase: tc.base})
		require.NotNil(t, ref, tc.base)
		assert.Equal(t, tc.kind, schemaNodeType(ref.Node), tc.base)

		ctx := context.Background()
		if tc.base != "" {
			ctx = WithSchemaIdScope(ctx, NewSchemaIdScope(tc.base))
		}
		ref, _, _ = idx.SearchIndexForReferenceWithContext(ctx, "#bar")
		require.NotNil(t, ref, tc.base)
		assert.Equal(t, tc.kind, schemaNodeType(ref.Node), tc.base)
	}
}

func TestRegisterSchemaAnchorAt_Skips(t *testing.T) {
	idx := NewTestSpecIndex().Load().(*SpecIndex)
	parse := func(src string) *yaml.Node {
		var n yaml.Node
		require.NoError(t, yaml.Unmarshal([]byte(src), &n))
		return n.Content[0]
	}

	idx.registerSchemaAnchorAt(parse("$anchor: a"), 0, []string{"components", "examples", "Sample", "value"}, "test.yaml")
	idx.registerSchemaAnchorAt(parse("$anchor: [a]"), 0, nil, "test.yaml")
	idx.registerSchemaAnchorAt(parse("$anchor: 'a b'"), 0, nil, "test.yaml")
	truncated := parse("$anchor: a")
	truncated.Content = truncated.Content[:1]
	idx.registerSchemaAnchorAt(truncated, 0, nil, "test.yaml")
	assert.Empty(t, idx.schemaAnchorRegistry)

	idx.registerSchemaAnchorAt(parse("$anchor: a"), 0, nil, "test.yaml")
	require.Contains(t, idx.schemaAnchorRegistry, "test.yaml#a")
	assert.Equal(t, "#", idx.schemaAnchorRegistry["test.yaml#a"].DefinitionPath)
}

// "#foo" inside an external file names an anchor in that file, not in the root document.
func TestSchemaAnchorInExternalFile(t *testing.T) {
	dir := t.TempDir()
	const root = `openapi: 3.1.0
info: {title: anchors, version: '1'}
paths: {}
components:
  schemas:
    Root:
      $anchor: name
      type: integer
    Pet:
      $ref: 'pet.yaml#/components/schemas/Pet'
`
	const pet = `components:
  schemas:
    Name:
      $anchor: name
      type: string
    Pet:
      type: object
      properties:
        name: {$ref: '#name'}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "root.yaml"), []byte(root), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pet.yaml"), []byte(pet), 0o600))

	var rootNode yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(root), &rootNode))

	cfg := CreateOpenAPIIndexConfig()
	cfg.BasePath = dir
	cfg.SpecAbsolutePath = filepath.Join(dir, "root.yaml")
	rolodex := NewRolodex(cfg)
	rolodex.SetRootNode(&rootNode)
	fileFS, err := NewLocalFSWithConfig(&LocalFSConfig{BaseDirectory: dir, IndexConfig: cfg})
	require.NoError(t, err)
	rolodex.AddLocalFS(dir, fileFS)

	require.NoError(t, rolodex.IndexTheRolodex(context.Background()))
	rolodex.Resolve()
	assert.Empty(t, rolodex.GetCaughtErrors())

	petIdx := rolodex.GetRootIndex().GetRolodex().GetIndexes()
	var found *Reference
	for _, idx := range petIdx {
		if idx.GetSpecAbsolutePath() == filepath.Join(dir, "pet.yaml") {
			found, _ = idx.SearchIndexForReference("#name")
		}
	}
	require.NotNil(t, found)
	assert.Equal(t, "string", schemaNodeType(found.Node))
}
