// Copyright 2023-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

// an $anchor inside an instance value is data, not a declaration, so it must not claim the name
// before the schema that declares it.
func TestSchemaAnchorInInstanceValues(t *testing.T) {
	const spec = `openapi: 3.1.0
info: {title: anchors, version: '1'}
paths:
  /p:
    get:
      responses:
        default:
          description: a response named default
          content:
            application/json:
              schema: {$anchor: defaultResponse, type: string}
              examples:
                Sample:
                  value: {$anchor: viaMediaExample, type: integer}
x-root: {$anchor: viaRootExtension, type: integer}
components:
  x-ext:
    Store: {$anchor: viaComponentsExtension, type: integer}
  schemas:
    AnchorPayloads:
      type: object
      default: {$anchor: viaDefault, type: integer}
      const: {$anchor: viaConst, type: integer}
      enum:
        - {$anchor: viaEnum, type: integer}
      examples:
        - {$anchor: viaExamples, type: integer}
      x-data: {$anchor: viaExtension, type: integer}
    MergedPayloads:
      <<: {default: {$anchor: viaMergedDefault, type: integer}}
      type: object
    Default: {$anchor: viaDefault, type: string}
    Const: {$anchor: viaConst, type: string}
    Enum: {$anchor: viaEnum, type: string}
    Examples: {$anchor: viaExamples, type: string}
    Extension: {$anchor: viaExtension, type: string}
    MediaExample: {$anchor: viaMediaExample, type: string}
    RootExtension: {$anchor: viaRootExtension, type: string}
    ComponentsExtension: {$anchor: viaComponentsExtension, type: string}
    MergedDefault: {$anchor: viaMergedDefault, type: string}
    Named:
      properties:
        x-named: {$anchor: namedProperty, type: string}
        default: {$anchor: defaultProperty, type: string}
        enum: {$id: 'https://example.com/enumProperty', type: string}
    x-Named:
      properties:
        id: {$id: 'https://example.com/namedComponent', type: string}
        x-named: {$anchor: namedPropertyOfNamedComponent, type: string}
    default: {$anchor: defaultComponent, type: string}
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(spec), &root))
	cfg := CreateClosedAPIIndexConfig()
	cfg.SpecAbsolutePath = "/specs/root.yaml"
	idx := NewSpecIndexWithConfig(&root, cfg)

	for _, ref := range []string{
		"#viaDefault", "#viaConst", "#viaEnum", "#viaExamples", "#viaExtension",
		"#viaMediaExample", "#viaRootExtension", "#viaComponentsExtension", "#viaMergedDefault",
	} {
		resolved := idx.ResolveRefViaSchemaId(ref)
		require.NotNil(t, resolved, ref)
		assert.Equal(t, "string", schemaNodeType(resolved.Node), ref)
	}

	// a component, property or response may be named "default" or "x-...": that is a name, not
	// a value or an extension.
	for ref, definition := range map[string]string{
		"#namedProperty":                     "#/components/schemas/Named/properties/x-named",
		"#defaultProperty":                   "#/components/schemas/Named/properties/default",
		"https://example.com/enumProperty":   "#/components/schemas/Named/properties/enum",
		"https://example.com/namedComponent": "#/components/schemas/x-Named/properties/id",
		"#namedPropertyOfNamedComponent":     "#/components/schemas/x-Named/properties/x-named",
		"#defaultComponent":                  "#/components/schemas/default",
		"#defaultResponse":                   "#/paths/~1p/get/responses/default/content/application~1json/schema",
	} {
		resolved := idx.ResolveRefViaSchemaId(ref)
		require.NotNil(t, resolved, ref)
		assert.Equal(t, "string", schemaNodeType(resolved.Node), ref)
		assert.Equal(t, definition, resolved.Definition, ref)
	}
}

// in a sequence such as required, "$anchor" and "$id" are entries, not keys, so the entry after
// them is not an identifier.
func TestSchemaIdentifierInSequence(t *testing.T) {
	const spec = `openapi: 3.1.0
info: {title: t, version: '1'}
paths: {}
components:
  schemas:
    Thing:
      type: object
      required: [$anchor, kind, $id, 'https://example.com/thing']
      properties:
        $anchor: {type: string}
        kind: {type: string}
        $id: {type: string}
    Ref:
      $ref: '#kind'
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(spec), &root))
	cfg := CreateClosedAPIIndexConfig()
	cfg.SpecAbsolutePath = "/specs/root.yaml"
	idx := NewSpecIndexWithConfig(&root, cfg)

	assert.Nil(t, idx.ResolveRefViaSchemaId("#kind"))
	assert.NotContains(t, idx.GetAllSchemaIds(), "https://example.com/thing")
}

// a schema declared under an extension, where its $anchor declares nothing, and aliased into
// components is anchored there.
func TestSchemaAnchorAliasedFromDataToSchema(t *testing.T) {
	const spec = `openapi: 3.1.0
info: {title: t, version: '1'}
x-templates:
  Str: &str
    $anchor: str
    type: string
paths: {}
components:
  schemas:
    Str: *str
    Uses:
      type: object
      properties:
        s: {$ref: '#str'}
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(spec), &root))
	cfg := CreateClosedAPIIndexConfig()
	cfg.SpecAbsolutePath = "/specs/root.yaml"
	idx := NewSpecIndexWithConfig(&root, cfg)
	assert.Empty(t, idx.GetReferenceIndexErrors())

	anchor := idx.schemaAnchorRegistry["/specs/root.yaml#str"]
	require.NotNil(t, anchor)
	assert.Equal(t, "#/components/schemas/Str", anchor.DefinitionPath)
	resolved := idx.ResolveRefViaSchemaId("#str")
	require.NotNil(t, resolved)
	assert.Equal(t, "string", schemaNodeType(resolved.Node))
}

// an $anchor in an $id resource is judged by its path from the mapping that holds the $id,
// whether or not that $id registers. Under an $id at the document root, the document's own rules
// still apply.
func TestSchemaAnchorResourceDepth(t *testing.T) {
	build := func(spec string) *SpecIndex {
		var root yaml.Node
		require.NoError(t, yaml.Unmarshal([]byte(spec), &root))
		cfg := CreateClosedAPIIndexConfig()
		cfg.SpecAbsolutePath = "/specs/root.yaml"
		return NewSpecIndexWithConfig(&root, cfg)
	}
	path := func(t *testing.T, idx *SpecIndex, uri string) string {
		entry := idx.schemaAnchorRegistry[uri]
		require.NotNil(t, entry, uri)
		return entry.DefinitionPath
	}

	t.Run("document root", func(t *testing.T) {
		idx := build(`$id: 'https://example.com/root'
$defs:
  S: {$anchor: s, type: string}
x-ext:
  E: {$anchor: e, type: integer}
`)
		assert.Equal(t, "#/$defs/S", path(t, idx, "https://example.com/root#s"))
		assert.NotContains(t, idx.schemaAnchorRegistry, "https://example.com/root#e")
	})

	t.Run("duplicate $id", func(t *testing.T) {
		idx := build(`components:
  schemas:
    First: {$id: 'https://example.com/dup', $defs: {A: {$anchor: a, type: string}}}
x-vendor:
  Second:
    $id: 'https://example.com/dup'
    default: {$anchor: d, type: integer}
    $defs:
      B: {$anchor: b, type: integer}
`)
		assert.Equal(t, "#/components/schemas/First", idx.GetSchemaById("https://example.com/dup").DefinitionPath)
		assert.Equal(t, "#/components/schemas/First/$defs/A", path(t, idx, "https://example.com/dup#a"))
		assert.Equal(t, "#/x-vendor/Second/$defs/B", path(t, idx, "https://example.com/dup#b"))
		assert.NotContains(t, idx.schemaAnchorRegistry, "https://example.com/dup#d")
	})

	t.Run("refused $id", func(t *testing.T) {
		// an $id holding a fragment is not registered, but it is still the base its anchors join.
		idx := build(`x-vendor:
  Bad:
    $id: 'https://example.com/bad#frag'
    default: {$anchor: d, type: integer}
    $defs:
      C: {$anchor: c, type: string}
`)
		assert.Nil(t, idx.GetSchemaById("https://example.com/bad#frag"))
		assert.Equal(t, "#/x-vendor/Bad/$defs/C", path(t, idx, "https://example.com/bad#frag#c"))
		assert.NotContains(t, idx.schemaAnchorRegistry, "https://example.com/bad#frag#d")
	})
}
