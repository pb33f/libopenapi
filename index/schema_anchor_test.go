// Copyright 2023-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
    Tuple:
      type: array
      prefixItems:
        - {$anchor: first, type: string}
        - {$anchor: second, type: integer}
    B:
      type: object
      properties:
        local: {$ref: '#foo'}
        outside: {$ref: 'https://example.com/r#bar'}
        first: {$ref: '#first'}
        second: {$ref: '#second'}
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
		{"percent-encoded name", "#f%6Fo", "string"},
		{"percent-encoded name in $id resource", "https://example.com/r#b%61r", "integer"},
		{"invalid percent-encoding", "#f%zzo", ""},
		{"array entry", "#second", "integer"},
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

	ref = idx.ResolveRefViaSchemaId("#second")
	require.NotNil(t, ref)
	assert.Equal(t, "1", ref.Name)
	assert.Equal(t, "#/components/schemas/Tuple/prefixItems/1", ref.Definition)

	// the encoded spelling finds the same anchor, and keeps the ref as written.
	ref = idx.ResolveRefViaSchemaId("#f%6Fo")
	require.NotNil(t, ref)
	assert.Equal(t, "#/components/schemas/A", ref.Definition)
	assert.Equal(t, "#f%6Fo", ref.RawRef)

	// a plain name the resource declares no anchor for is not read as one of its keywords.
	assert.Nil(t, idx.ResolveRefViaSchemaId("https://example.com/r#properties"))
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

	// anchors on two entries of one array are two targets, told apart by the index.
	require.Contains(t, mapped, "#/components/schemas/Tuple/prefixItems/0")
	assert.Equal(t, "string", schemaNodeType(mapped["#/components/schemas/Tuple/prefixItems/0"].Node))
	require.Contains(t, mapped, "#/components/schemas/Tuple/prefixItems/1")
	assert.Equal(t, "integer", schemaNodeType(mapped["#/components/schemas/Tuple/prefixItems/1"].Node))
	assert.NotContains(t, mapped, "#/components/schemas/Tuple/prefixItems")
}

func TestSchemaAnchorFragmentName(t *testing.T) {
	for fragment, want := range map[string]string{
		"#foo":     "foo",
		"#f%6Fo":   "foo",
		"#":        "",
		"foo":      "",
		"#/foo":    "",
		"#%2Ffoo":  "", // the JSON pointer "/foo"
		"#f%zzo":   "",
		"#1bad":    "",
		"#a%20b":   "",
		"#_a-b.c9": "_a-b.c9",
	} {
		name, ok := schemaAnchorFragmentName(fragment)
		assert.Equal(t, want, name, fragment)
		assert.Equal(t, want != "", ok, fragment)
		assert.Equal(t, want != "", isSchemaAnchorFragment(fragment), fragment)
		assert.Equal(t, want != "", IsSchemaAnchorFragment(fragment), fragment)
	}
}

func TestSchemaIdentifierDefinitionPath(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(`a:
  b/c:
    - {$id: 'https://example.com/x'}
    - - {$id: 'https://example.com/y'}
      - {$anchor: z}
  alias: &shared {$anchor: w}
  ref: *shared
  merged:
    <<: *shared
    v: {$id: 'https://example.com/v'}
  top: {$anchor: u, other: 1}
  t~/: {$anchor: tl}
  named:
    properties:
      $id: {$anchor: d}
      $ref: {properties: {o: {$id: 'https://example.com/o'}}}
  self: &self
    $anchor: s
    loop: *self
m:
  <<: {$anchor: mw, type: string}
  k: 1
`), &root))
	cfg := CreateClosedAPIIndexConfig()
	cfg.SpecAbsolutePath = "/specs/root.yaml"
	idx := NewSpecIndexWithConfig(&root, cfg)
	anchor := func(name string) string {
		entry := idx.schemaAnchorRegistry["/specs/root.yaml#"+name]
		require.NotNil(t, entry, name)
		return entry.DefinitionPath
	}
	id := func(uri string) string {
		entry := idx.GetSchemaById(uri)
		require.NotNil(t, entry, uri)
		return entry.DefinitionPath
	}

	// the path in the tree, with the sequence indices, and each key with "~" written as "~0" and
	// "/" as "~1".
	assert.Equal(t, "#/a/b~1c/0", id("https://example.com/x"))
	assert.Equal(t, "#/a/b~1c/1/0", id("https://example.com/y"))
	assert.Equal(t, "#/a/b~1c/1/1", anchor("z"))
	assert.Equal(t, "#/a/top", anchor("u"))
	assert.Equal(t, "#/a/t~0~1", anchor("tl"))
	// a property named $id or $ref is a segment like any other.
	assert.Equal(t, "#/a/named/properties/$id", anchor("d"))
	assert.Equal(t, "#/a/named/properties/$ref/properties/o", id("https://example.com/o"))
	// a key beside a merge key keeps its own segment.
	assert.Equal(t, "#/a/merged/v", id("https://example.com/v"))
	// a value reached three times, at its declaration and through an alias and a merge key,
	// registers at its declaration.
	assert.Equal(t, "#/a/alias", anchor("w"))
	// a mapping that holds an alias of itself, at its declaration rather than through the loop.
	assert.Equal(t, "#/a/self", anchor("s"))
	// an identifier written under a merge key belongs to the mapping that merges it, the one a
	// reader of m sees; the merged-in mapping has no pointer of its own.
	assert.Equal(t, "#/m", anchor("mw"))
	resolved := idx.ResolveRefViaSchemaId("#mw")
	require.NotNil(t, resolved)
	assert.Equal(t, "#/m", resolved.Definition)
	assert.Equal(t, "string", schemaNodeType(resolved.Node))
}

func TestUnderSchemaDataPath(t *testing.T) {
	for _, path := range []string{
		"components/schemas/A/default",
		"components/schemas/A/const",
		"components/schemas/A/enum/0",
		"components/schemas/A/examples/0",
		"components/schemas/A/example",
		"components/schemas/A/x-data",
		"components/x-ext/A",
		"components/examples/A/value",
		"x-ext",
		"paths/~1p/get/responses/200/content/application~1json/examples/E/value",
		"components/schemas/default/default",
		"components/schemas/A/properties/default/default",
		"components/schemas/A/properties/x-named/x-data",
		"components/parameters/P/schema/default",
		"paths/~1p/get/parameters/0/schema/x-data",
		"paths/~1p/get/responses/200/links/L/parameters",
		"paths/~1p/get/responses/200/links/L/parameters/p",
		"paths/~1p/get/responses/200/links/L/requestBody",
		"paths/~1p/get/responses/200/links/parameters/requestBody",
		"components/responses/R/links/L/parameters/p",
		"components/links/L/parameters/p",
		"components/links/L/requestBody",
		"paths/~1p/get/responses/x-shadow/content/application~1json/schema",
		"components/pathItems/P/get/responses/x-shadow",
		"webhooks/hook/post/responses/x-shadow/content/application~1json/schema",
	} {
		assert.True(t, underSchemaDataPath(strings.Split(path, "/"), false, true), path)
	}
	for _, path := range []string{
		"",
		"components/schemas/A",
		"components/schemas/default",
		"components/schemas/x-Named",
		"components/schemas/x-Named/properties/x-named",
		"components/schemas/A/properties/default",
		"components/schemas/A/properties/enum/items",
		"components/schemas/A/patternProperties/x-",
		"components/schemas/A/$defs/const",
		"components/schemas/A/definitions/example",
		"components/schemas/A/dependentSchemas/examples",
		"components/schemas/A/dependencies/x-dep",
		"components/schemas/A/prefixItems/0",
		"components/responses/default/content/application~1json/schema",
		"components/headers/x-rate/schema",
		"paths/~1p/get/responses/default/content/application~1json/schema",
		"paths/~1p/get/responses/200/headers/x-rate/schema",
		"paths/~1p/get/requestBody/content/x-custom/schema",
		"paths/~1p/post/requestBody/content/multipart~1form-data/encoding/x-part/headers/h/schema",
		"paths/~1p/get/callbacks/x-cb",
		"paths/~1p/get/responses/200/links/default",
		"webhooks/x-hook/post",
		"definitions/default",
		"parameters/x-param/schema",
		"responses/default/schema",
		"paths/~1p/get/parameters/0/schema",
		"paths/~1p/get/responses/200/links/parameters",
		"paths/~1p/get/responses/200/links/requestBody/server",
		"components/links/parameters",
		"components/links/requestBody",
		"components/schemas/A/properties/links/properties/parameters",
		"components/responses/x-foo/content/application~1json/schema",
		"paths/~1p/get/responses/default/links/x-link",
		"responses/x-foo/schema",
	} {
		segments := strings.Split(path, "/")
		if path == "" {
			segments = nil
		}
		assert.False(t, underSchemaDataPath(segments, false, true), path)
	}

	// from the root of an $id resource, which is a schema, only the schema's keywords count.
	for _, path := range []string{
		"default",
		"const",
		"enum/0",
		"examples/0",
		"example",
		"x-data",
		"properties/p/default",
		"$defs/S/x-data",
		"components/schemas/x-A",
		"headers/x-h",
		"content/x-c",
		"responses/x-r",
		"parameters/x-p",
	} {
		assert.True(t, underSchemaDataPath(strings.Split(path, "/"), true, true), path)
	}
	for _, path := range []string{
		"$defs/S",
		"$defs/default",
		"properties/x-named",
		"properties/default/properties/enum",
		"patternProperties/x-",
		"definitions/example",
		"dependentSchemas/const",
		"dependencies/x-dep",
		"prefixItems/0",
		"items/properties/links/properties/parameters",
		"links/L/parameters",
		"links/L/requestBody",
	} {
		assert.False(t, underSchemaDataPath(strings.Split(path, "/"), true, true), path)
	}
}

func TestInSchemaData(t *testing.T) {
	split := func(path string) []string { return strings.Split(path, "/") }

	// an $id under an extension starts a resource, while one in a literal value is data.
	assert.True(t, inSchemaData(split("x-vendor/Thing"), 0, true))
	assert.False(t, inSchemaData(split("x-vendor/Thing"), 0, false))
	assert.True(t, inSchemaData(split("paths/~1p/get/responses/x-r"), 0, true))
	assert.False(t, inSchemaData(split("paths/~1p/get/responses/x-r"), 0, false))
	assert.True(t, inSchemaData(split("x-vendor/Thing/default"), 0, false))
	assert.True(t, inSchemaData(split("components/links/L/parameters"), 0, false))
	assert.True(t, inSchemaData(split("components/schemas/W/default"), 0, false))

	// inside a resource the path from its root decides, where a schema name is a keyword.
	assert.True(t, inSchemaData(split("x-vendor/Thing/default"), 2, false))
	assert.False(t, inSchemaData(split("x-vendor/Thing/properties/default"), 2, false))
	assert.True(t, inSchemaData(split("components/schemas/W/$defs/S/const"), 3, false))
	// a depth past the pointer falls back to the document root.
	assert.False(t, inSchemaData(split("components/schemas/W"), 4, false))
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

	scope := NewSchemaIdScope("test.yaml")
	idx.registerSchemaAnchorAt(parse("$anchor: a"), 0, []string{"components", "examples", "Sample", "value"}, scope)
	idx.registerSchemaAnchorAt(parse("$anchor: [a]"), 0, nil, scope)
	idx.registerSchemaAnchorAt(parse("$anchor: 'a b'"), 0, nil, scope)
	truncated := parse("$anchor: a")
	truncated.Content = truncated.Content[:1]
	idx.registerSchemaAnchorAt(truncated, 0, nil, scope)
	// inside the $id resource at x-vendor/Thing, its default is data.
	resource := NewSchemaIdScope("test.yaml")
	resource.PushId("https://example.com/thing")
	resource.resourceDepth = 2
	idx.registerSchemaAnchorAt(parse("$anchor: d"), 0, []string{"x-vendor", "Thing", "default"}, resource)
	assert.Empty(t, idx.schemaAnchorRegistry)

	idx.registerSchemaAnchorAt(parse("$anchor: a"), 0, nil, scope)
	require.Contains(t, idx.schemaAnchorRegistry, "test.yaml#a")
	assert.Equal(t, "#", idx.schemaAnchorRegistry["test.yaml#a"].DefinitionPath)

	// the extension holding the resource does not count, only the path within it.
	idx.registerSchemaAnchorAt(parse("$anchor: s"), 0, []string{"x-vendor", "Thing", "$defs", "S"}, resource)
	require.Contains(t, idx.schemaAnchorRegistry, "https://example.com/thing#s")
	assert.Equal(t, "#/x-vendor/Thing/$defs/S", idx.schemaAnchorRegistry["https://example.com/thing#s"].DefinitionPath)
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

// the same "#item" under oneOf in different resources must map to each resource's own anchor.
func TestSchemaAnchorPolymorphicScopedByResource(t *testing.T) {
	build := func(t *testing.T, spec string) *SpecIndex {
		var root yaml.Node
		require.NoError(t, yaml.Unmarshal([]byte(spec), &root))
		cfg := CreateClosedAPIIndexConfig()
		cfg.SpecAbsolutePath = "/specs/root.yaml"
		return NewSpecIndexWithConfig(&root, cfg)
	}

	t.Run("both resources mapped", func(t *testing.T) {
		idx := build(t, `openapi: 3.1.0
info: {title: anchors, version: '1'}
paths: {}
components:
  schemas:
    R1:
      $id: https://example.com/r1
      $defs: {X: {$anchor: item, type: integer}}
      oneOf: [{$ref: '#item'}]
    R2:
      $id: https://example.com/r2
      $defs: {X: {$anchor: item, type: string}}
      oneOf: [{$ref: '#item'}]
`)
		assert.Empty(t, idx.GetReferenceIndexErrors())

		poly := idx.GetPolyReferences()
		assert.Len(t, poly, 2)
		assert.Contains(t, poly, "https://example.com/r1#item")
		assert.Contains(t, poly, "https://example.com/r2#item")

		mapped := idx.GetMappedReferences()
		for def, kind := range map[string]string{
			"/specs/root.yaml#/components/schemas/R1/$defs/X": "integer",
			"/specs/root.yaml#/components/schemas/R2/$defs/X": "string",
		} {
			require.Contains(t, mapped, def)
			assert.Equal(t, kind, schemaNodeType(mapped[def].Node), def)
		}
	})

	t.Run("missing anchor in the first resource is reported", func(t *testing.T) {
		idx := build(t, `openapi: 3.1.0
info: {title: anchors, version: '1'}
paths: {}
components:
  schemas:
    R1:
      $id: https://example.com/r1
      oneOf: [{$ref: '#item'}]
    R2:
      $id: https://example.com/r2
      $defs: {X: {$anchor: item, type: string}}
      oneOf: [{$ref: '#item'}]
`)
		errs := idx.GetReferenceIndexErrors()
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "#item")
	})
}

// a oneOf that loops back through an $anchor is as circular as the same loop through a JSON pointer.
func TestSchemaAnchorPolymorphicCircularReference(t *testing.T) {
	circular := func(t *testing.T, ref string) []*CircularReferenceResult {
		var root yaml.Node
		require.NoError(t, yaml.Unmarshal([]byte(`openapi: 3.1.0
info: {title: anchors, version: '1'}
paths: {}
components:
  schemas:
    A:
      $anchor: a
      type: object
      oneOf: [{$ref: '`+ref+`'}]
`), &root))
		cfg := CreateClosedAPIIndexConfig()
		cfg.SpecAbsolutePath = "/specs/root.yaml"
		idx := NewSpecIndexWithConfig(&root, cfg)
		require.Empty(t, idx.GetReferenceIndexErrors())

		resolver := NewResolver(idx)
		resolver.CheckForCircularReferences()
		return resolver.GetCircularReferences()
	}

	byPointer := circular(t, "#/components/schemas/A")
	require.Len(t, byPointer, 1)
	assert.True(t, byPointer[0].IsPolymorphicResult)

	byAnchor := circular(t, "#a")
	require.Len(t, byAnchor, 1)
	assert.True(t, byAnchor[0].IsPolymorphicResult)
	assert.Equal(t, byPointer[0].LoopPoint.FullDefinition, byAnchor[0].LoopPoint.FullDefinition)
}

// every use of "#name" is inlined by a destructive resolve, not just the first, and each
// resource's uses get that resource's anchor.
func TestSchemaAnchorResolverInlinesEveryUse(t *testing.T) {
	const spec = `openapi: 3.1.0
info: {title: anchors, version: '1'}
paths: {}
components:
  schemas:
    N: {$anchor: name, type: string}
    A:
      type: object
      properties:
        x: {$ref: '#name'}
        y: {$ref: '#name'}
    R:
      $id: https://example.com/r
      $defs: {X: {$anchor: name, type: integer}}
      properties:
        p: {$ref: '#name'}
        q: {$ref: '#name'}
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(spec), &root))
	cfg := CreateClosedAPIIndexConfig()
	cfg.SpecAbsolutePath = "/specs/root.yaml"
	idx := NewSpecIndexWithConfig(&root, cfg)

	resolver := NewResolver(idx)
	assert.Empty(t, resolver.Resolve())
	resolver.ResolvePendingNodes()

	out, err := yaml.Marshal(&root)
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(out, &doc))
	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	for schema, props := range map[string]map[string]string{
		"A": {"x": "string", "y": "string"},
		"R": {"p": "integer", "q": "integer"},
	} {
		properties := schemas[schema].(map[string]any)["properties"].(map[string]any)
		for prop, kind := range props {
			resolved := properties[prop].(map[string]any)
			assert.NotContains(t, resolved, "$ref", schema+"."+prop)
			assert.Equal(t, kind, resolved["type"], schema+"."+prop)
		}
	}
}

// "other.yaml#foo" names the $anchor foo in other.yaml's root resource: its $id, or the file.
func TestSchemaAnchorInOtherFile(t *testing.T) {
	dir := t.TempDir()
	const root = `openapi: 3.1.0
info: {title: anchors, version: '1'}
paths: {}
components:
  schemas:
    Root:
      $anchor: name
      type: boolean
    Foo: {$ref: 'other.yaml#foo'}
    Wid: {$ref: 'withid.yaml#wid'}
    Chain: {$ref: 'chain.yaml#/components/schemas/Chain'}
`
	files := map[string]string{
		"other.yaml": `components:
  schemas:
    Foo: {$anchor: foo, type: string}
`,
		"withid.yaml": `$id: 'https://example.com/wid'
$anchor: wid
type: integer
`,
		"chain.yaml": `components:
  schemas:
    Chain:
      type: object
      properties:
        leaf: {$ref: 'sub/leaf.yaml#leaf'}
`,
		"sub/leaf.yaml": `components:
  schemas:
    Leaf: {$anchor: leaf, type: number}
`,
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "root.yaml"), []byte(root), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "sub"), 0o700))
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
	}

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

	idx := rolodex.GetRootIndex()
	for ref, kind := range map[string]string{
		"other.yaml#foo":  "string",
		"withid.yaml#wid": "integer",
	} {
		found, _ := idx.SearchIndexForReference(ref)
		require.NotNil(t, found, ref)
		assert.Equal(t, kind, schemaNodeType(found.Node), ref)
	}

	// an anchor on the file's root names the whole file, and is searched for by its $id URI.
	wid := idx.FindComponent(context.Background(), "withid.yaml#wid")
	require.NotNil(t, wid)
	assert.Equal(t, filepath.Join(dir, "withid.yaml"), wid.FullDefinition)
	assert.Equal(t, "https://example.com/wid#wid", wid.RawRef)

	// "#leaf", reached through chain.yaml, resolves against sub/leaf.yaml.
	leaf := idx.FindComponent(context.Background(), "chain.yaml#/components/schemas/Chain/properties/leaf")
	require.NotNil(t, leaf)
	assert.Equal(t, "number", schemaNodeType(leaf.Node))

	found, _ := idx.SearchIndexForReference("withid.yaml#missing")
	assert.Nil(t, found)
	assert.Nil(t, idx.findSchemaAnchorInFile(context.Background(), filepath.Join(dir, "withid.yaml"), "withid.yaml#missing", "missing"))

	// the file is this document.
	self := idx.findSchemaAnchorInFile(context.Background(), cfg.SpecAbsolutePath, "root.yaml#name", "name")
	require.NotNil(t, self)
	assert.Equal(t, "boolean", schemaNodeType(self.Node))
}

func TestFindSchemaAnchorInFile_NoRolodex(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(schemaAnchorSpec), &root))
	cfg := CreateClosedAPIIndexConfig()
	cfg.SpecAbsolutePath = "/specs/root.yaml"
	idx := NewSpecIndexWithConfig(&root, cfg)
	assert.Nil(t, idx.findSchemaAnchorInFile(context.Background(), "/specs/other.yaml", "other.yaml#foo", "foo"))
}

func TestGetSchemaIdBaseForNode(t *testing.T) {
	var nilIndex *SpecIndex
	node := &yaml.Node{Kind: yaml.MappingNode}
	assert.Empty(t, nilIndex.GetSchemaIdBaseForNode(node))

	idx := &SpecIndex{}
	assert.Empty(t, idx.GetSchemaIdBaseForNode(nil))
	assert.Empty(t, idx.GetSchemaIdBaseForNode(node))

	// a node reached again keeps the base it was first seen with.
	idx.recordSchemaIdBase(node, "https://example.com/a")
	idx.recordSchemaIdBase(node, "https://example.com/b")
	assert.Equal(t, "https://example.com/a", idx.GetSchemaIdBaseForNode(node))
}

func TestResolveLocalFileRef(t *testing.T) {
	resolver := &Resolver{}
	// an absolute base on every platform, since the ref path is made absolute.
	base := t.TempDir()
	assert.Equal(t, filepath.Join(base, "sub", "leaf.yaml")+"#leaf", resolver.resolveLocalFileRef(base, "sub/leaf.yaml#leaf"))
	assert.Equal(t, filepath.Join(base, "sub", "leaf.yaml"), resolver.resolveLocalFileRef(base, "sub/leaf.yaml"))
}

// TestFindSchemaAnchorInPendingFile checks that an $anchor in a file still being indexed is found
// from the file's parsed YAML, without waiting for its index.
func TestFindSchemaAnchorInPendingFile(t *testing.T) {
	dir := t.TempDir()
	cfg := CreateOpenAPIIndexConfig()
	cfg.BasePath = dir
	cfg.SpecAbsolutePath = filepath.Join(dir, "root.yaml")
	rolo := NewRolodex(cfg)
	cfg.Rolodex = rolo
	files, err := NewLocalFSWithConfig(&LocalFSConfig{BaseDirectory: dir, IndexConfig: cfg})
	require.NoError(t, err)
	pending := func(name, content string) string {
		path := filepath.Join(dir, name)
		file := &LocalFile{fullPath: path, data: []byte(content), indexingComplete: make(chan struct{})}
		files.Files.Store(path, file)
		t.Cleanup(func() { close(file.indexingComplete) })
		return path
	}
	cat := pending("cat.yaml", "type: object\n$defs:\n  N: {$anchor: name, type: string}\n")
	tag := pending("tag.yaml", "$id: 'https://example.com/tag'\n$defs:\n  T: {$anchor: tag, type: integer}\n")
	rolo.AddLocalFS(dir, files)

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("openapi: 3.1.0\ncomponents: {}\n"), &root))
	idx := NewSpecIndexWithConfig(&root, cfg)

	result := make(chan [3]*Reference, 1)
	go func() {
		ctx := context.Background()
		result <- [3]*Reference{
			idx.findSchemaAnchorInFile(ctx, cat, "cat.yaml#name", "name"),
			idx.findSchemaAnchorInFile(ctx, tag, "tag.yaml#tag", "tag"),
			idx.findSchemaAnchorInFile(ctx, cat, "cat.yaml#missing", "missing"),
		}
	}()
	var found [3]*Reference
	select {
	case found = <-result:
	case <-time.After(5 * time.Second):
		t.Fatal("anchor lookup waited for the file's index")
	}

	require.NotNil(t, found[0])
	assert.Nil(t, found[0].Index)
	assert.Equal(t, cat+"#/$defs/N", found[0].FullDefinition)
	assert.Equal(t, "string", found[0].Node.Content[3].Value)

	require.NotNil(t, found[1])
	assert.Nil(t, found[1].Index)
	assert.Equal(t, "https://example.com/tag#tag", found[1].RawRef)
	assert.Equal(t, "integer", found[1].Node.Content[3].Value)

	assert.Nil(t, found[2])
}
