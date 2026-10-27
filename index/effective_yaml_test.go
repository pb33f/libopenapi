package index

import (
	"context"
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestEffectiveReferenceOccurrences(t *testing.T) {
	const source = `openapi: 3.1.1
info: {title: Index, version: '1'}
paths: {}
x-ref: &ref {$ref: '#/components/schemas/Target'}
x-props: &props
  nested: *ref
components:
  schemas:
    Target: {type: string}
    First: {type: object, properties: *props}
    Second: {type: object, properties: {<<: *props}}
    Example:
      type: object
      example: {nested: *ref}
  examples:
    Named:
      value: {nested: *ref}
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(source), &root))
	before, err := yaml.Marshal(&root)
	require.NoError(t, err)
	idx := NewSpecIndex(&root)
	paths := map[string]bool{}
	for _, ref := range idx.rawSequencedRefs {
		paths[strings.Join(ref.SourcePath, "/")] = true
	}
	assert.Contains(t, paths, "components/schemas/First/properties/nested")
	assert.Contains(t, paths, "components/schemas/Second/properties/nested")
	for path := range paths {
		assert.NotContains(t, path, "/Example/")
		assert.NotContains(t, path, "/examples/Named/value/")
	}
	assert.Empty(t, idx.GetReferenceIndexErrors())
	after, err := yaml.Marshal(&root)
	require.NoError(t, err)
	assert.Equal(t, before, after)
}

func TestEffectiveFragmentTraversal(t *testing.T) {
	const source = `base: &base
  escaped/key: &value {type: string}
components: {schemas: {<<: *base}}
array: [*value]
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(source), &root))
	for _, path := range []string{"#/components/schemas/escaped~1key/type", "#/array/0/type"} {
		got := navigateToFragment(&root, path)
		require.NotNil(t, got, path)
		assert.Equal(t, "string", got.Value)
	}
	assert.Nil(t, navigateToFragment(&root, "#/components/schemas/missing"))
	assert.Nil(t, navigateToFragment(&root, "#/array/1"))
}

func TestEffectiveTraversalCycles(t *testing.T) {
	for _, source := range []string{
		"x-map: &mapping {<<: *mapping, nested: *mapping, local: value}\n",
		"x-sequence: &sequence [*sequence]\n",
	} {
		var root yaml.Node
		require.NoError(t, yaml.Unmarshal([]byte(source), &root))
		idx := NewSpecIndex(&root)
		assert.Empty(t, idx.GetReferenceIndexErrors())
	}
}

func TestEffectiveComponentCollections(t *testing.T) {
	const source = `openapi: 3.1.1
info: {title: Index, version: '1'}
paths: {}
x-entry: &entry
  Named: {description: inherited}
x-collections: &collections
  schemas: {<<: *entry}
  parameters: {<<: *entry}
  requestBodies: {<<: *entry}
  responses: {<<: *entry}
  securitySchemes: {<<: *entry}
  headers: {<<: *entry}
  examples: {<<: *entry}
  links: {<<: *entry}
  callbacks: {<<: *entry}
  pathItems: {<<: *entry}
components: *collections
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(source), &root))
	idx := NewSpecIndex(&root)
	assert.Equal(t, 1, idx.GetComponentSchemaCount())
	assert.Equal(t, 1, idx.GetComponentParameterCount())
	for _, kind := range []string{"schemas", "parameters", "requestBodies", "responses", "securitySchemes", "headers", "examples", "links", "callbacks", "pathItems"} {
		ref := idx.FindComponent(context.Background(), "#/components/"+kind+"/Named")
		require.NotNil(t, ref, kind)
		assert.Equal(t, "Named", ref.KeyNode.Value, kind)
		assert.Nil(t, idx.FindComponent(context.Background(), "#/components/"+kind+"/<<"), kind)
	}
}

func TestEffectiveInlineSchemaCollections(t *testing.T) {
	const source = `openapi: 3.1.1
info: {title: Index, version: '1'}
paths: {}
x-schema: &schema {type: object, properties: {name: {type: string}}}
x-map: &map {child: *schema}
x-array: &array [*schema]
components:
  schemas:
    Single: {type: array, items: *schema}
    Mapping: {type: object, properties: {<<: *map}}
    Array: {allOf: *array}
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(source), &root))
	idx := NewSpecIndex(&root)
	definitions := map[string]*Reference{}
	for _, ref := range idx.allInlineSchemaObjectDefinitions {
		definitions[ref.Definition] = ref
	}
	for _, definition := range []string{
		"#/components/schemas/Single/items",
		"#/components/schemas/Mapping/properties/child",
		"#/components/schemas/Array/allOf/0",
	} {
		assert.Contains(t, definitions, definition)
	}
}

func TestEffectiveSchemaIDScope(t *testing.T) {
	const source = `openapi: 3.1.1
info: {title: Index, version: '1'}
paths: {}
x-scope: &scope {$id: 'https://example.test/scoped', type: object}
x-property: &property {$ref: '#/components/schemas/Target'}
components:
  schemas:
    Target: {type: string}
    Scoped:
      <<: *scope
      properties: {value: *property}
    Ordinary:
      type: object
      properties: {value: *property}
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(source), &root))
	idx := NewSpecIndex(&root)
	scopes := map[string]string{}
	for _, ref := range idx.rawSequencedRefs {
		scopes[strings.Join(ref.SourcePath, "/")] = ref.SchemaIdBase
	}
	assert.Equal(t, "https://example.test/scoped", scopes["components/schemas/Scoped/properties/value"])
	assert.Equal(t, "", scopes["components/schemas/Ordinary/properties/value"])
}

func TestEffectiveLookupWithAbsentNodes(t *testing.T) {
	t.Run("missing collection mapping", func(t *testing.T) {
		require.Nil(t, effectiveMappingValue(nil, "schemas"))
	})
	t.Run("document with absent content node", func(t *testing.T) {
		root := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{nil}}
		require.Nil(t, navigateToFragment(root, "#/type"))
	})
}
