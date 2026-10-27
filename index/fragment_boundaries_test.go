package index

import (
	"context"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

func TestFindComponentFragmentValues(t *testing.T) {
	const source = `openapi: 3.1.1
info: {title: Pointer, version: '1'}
paths: {}
components:
  schemas:
    List: {oneOf: [{type: string}, {type: integer}]}
    Holder:
      type: object
      properties:
        '': {type: boolean, properties: {'': {type: 'null'}}}
        'A+B C': {type: number}
        'a/b~c': {type: string}
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(source), &root))
	idx := NewSpecIndex(&root)
	for _, tc := range []struct{ name, ref, kind string }{
		{"zero", "#/components/schemas/List/oneOf/0", "string"},
		{"last element", "#/components/schemas/List/oneOf/1", "integer"},
		{"empty member", "#/components/schemas/Holder/properties/", "boolean"},
		{"nested empty member", "#/components/schemas/Holder/properties//properties/", "null"},
		{"literal plus", "#/components/schemas/Holder/properties/A+B%20C", "number"},
		{"escaped plus", "#/components/schemas/Holder/properties/A%2BB%20C", "number"},
		{"pointer escapes", "#/components/schemas/Holder/properties/a~1b~0c", "string"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ref := idx.FindComponent(context.Background(), tc.ref)
			require.NotNil(t, ref)
			require.NotNil(t, ref.Node)
			require.GreaterOrEqual(t, len(ref.Node.Content), 2)
			require.Equal(t, "type", ref.Node.Content[0].Value)
			require.Equal(t, tc.kind, ref.Node.Content[1].Value)
		})
	}
}

func TestFindComponentRejectsOutOfRangeArrayFragments(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("items: [first, second]\n"), &root))
	idx := NewSpecIndex(&root)
	for _, segment := range []string{"2", "-1", "01", "", "4294967296", "9223372036854775808", "18446744073709551616", "999999999999999999999999999999999999999"} {
		t.Run(segment, func(t *testing.T) {
			var ref *Reference
			require.NotPanics(t, func() {
				ref = idx.FindComponent(context.Background(), "#/items/"+segment)
			})
			require.Nil(t, ref)
		})
	}
}

func TestFindComponentRejectsMalformedFragmentEscapes(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("items: [first]\n"), &root))
	idx := NewSpecIndex(&root)
	for _, fragment := range []string{"#/items/%", "#/items/%zz"} {
		t.Run(fragment, func(t *testing.T) {
			require.Nil(t, idx.FindComponent(context.Background(), fragment))
		})
	}
}

func TestFindComponentMissingEmptyMemberDoesNotResolveParent(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("items: [first]\n"), &root))
	idx := NewSpecIndex(&root)
	require.Nil(t, idx.FindComponent(context.Background(), "#/"))
}

func TestFindComponentRootEmptyMember(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("'': {type: integer}\n"), &root))
	ref := FindComponent(context.Background(), &root, "#/", "", nil)
	require.NotNil(t, ref)
	require.Same(t, root.Content[0].Content[1], ref.Node)
}

func TestSearchIndexRejectsMalformedFragmentEscape(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("items: [first]\n"), &root))
	idx := NewSpecIndex(&root)
	ref, _ := idx.SearchIndexForReference("#/items/%zz")
	require.Nil(t, ref)
}

func TestFindComponentNilRootWithEmptyPointer(t *testing.T) {
	require.Nil(t, FindComponent(context.Background(), nil, "", "", nil))
}

func TestResolverPreservesExternalEmptyMemberFragment(t *testing.T) {
	resolver := &Resolver{}
	parent := &Reference{FullDefinition: "https://example.com/root.yaml"}
	for _, ref := range []string{
		"https://example.com/schema.yaml",
		"https://example.com/schema.yaml#/",
		"https://example.com/schema.yaml#/properties/name",
	} {
		t.Run(ref, func(t *testing.T) {
			require.Equal(t, ref, resolver.buildDefPath(parent, ref))
		})
	}
}
