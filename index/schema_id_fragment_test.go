package index

import (
	"context"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

func TestSchemaIDFragmentResolution(t *testing.T) {
	const source = `openapi: 3.1.1
info: {title: Example, version: '1'}
paths: {}
components:
  schemas:
    Holder:
      $id: https://example.com/holder
      type: object
      properties:
        normal: {type: boolean}
        'A+B C': {type: integer}
        'percent%name': {type: string}
`
	for _, tc := range []struct{ name, fragment, kind string }{
		{"whole schema", "", "object"},
		{"empty fragment", "#", "object"},
		{"ordinary member", "#/properties/normal", "boolean"},
		{"literal plus", "#/properties/A+B%20C", "integer"},
		{"encoded plus", "#/properties/A%2BB%20C", "integer"},
		{"encoded percent", "#/properties/percent%25name", "string"},
		{"missing member", "#/properties/missing", ""},
		{"malformed escape", "#/properties/%zz", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var root yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(source), &root))
			idx := NewSpecIndex(&root)
			for _, lookup := range []struct {
				name string
				call func() *Reference
			}{
				{"registry", func() *Reference {
					return idx.ResolveRefViaSchemaId("https://example.com/holder" + tc.fragment)
				}},
				{"component URI", func() *Reference {
					return idx.FindComponent(context.Background(), "https://example.com/holder"+tc.fragment)
				}},
				{"component path", func() *Reference {
					return idx.FindComponent(context.Background(), "/holder"+tc.fragment)
				}},
				{"search URI", func() *Reference {
					ref, _ := idx.SearchIndexForReference("https://example.com/holder" + tc.fragment)
					return ref
				}},
				{"search path", func() *Reference {
					ref, _ := idx.SearchIndexForReference("/holder" + tc.fragment)
					return ref
				}},
			} {
				t.Run(lookup.name, func(t *testing.T) {
					ref := lookup.call()
					if tc.kind == "" {
						require.Nil(t, ref)
						return
					}
					require.NotNil(t, ref)
					require.NotNil(t, ref.Node)
					var kind string
					for i := 0; i+1 < len(ref.Node.Content); i += 2 {
						if ref.Node.Content[i].Value == "type" {
							kind = ref.Node.Content[i+1].Value
						}
					}
					require.Equal(t, tc.kind, kind)
				})
			}
		})
	}
}
