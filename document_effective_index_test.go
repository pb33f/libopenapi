package libopenapi

import (
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/utils"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestBuildV3ModelEffectiveComponentIndex(t *testing.T) {
	const prefix = `openapi: 3.1.1
info: {title: Index, version: '1'}
paths:
  /bound:
    get:
      responses:
        '200':
          description: Bound
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Bound'}
`
	for _, tc := range []struct{ name, components string }{
		{"plain", "components:\n  schemas:\n    Bound: {type: integer, minimum: 5}\n"},
		{"components merge", "x-base: &base\n  schemas:\n    Bound: {type: integer, minimum: 5}\ncomponents:\n  <<: *base\n"},
		{"schemas merge", "x-base: &base\n  Bound: {type: integer, minimum: 5}\ncomponents:\n  schemas:\n    <<: *base\n"},
		{"components alias", "x-base: &base\n  schemas:\n    Bound: {type: integer, minimum: 5}\ncomponents: *base\n"},
		{"schemas alias", "x-base: &base\n  Bound: {type: integer, minimum: 5}\ncomponents:\n  schemas: *base\n"},
		{"schema value merge", "x-base: &base {type: integer, minimum: 5}\ncomponents:\n  schemas:\n    Bound:\n      <<: *base\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := NewDocument([]byte(prefix + tc.components))
			require.NoError(t, err)
			root := doc.GetSpecInfo().RootNode
			before, err := yaml.Marshal(root)
			require.NoError(t, err)
			model, err := doc.BuildV3Model()
			require.NoError(t, err)
			proxy, ok := model.Model.Components.Schemas.Get("Bound")
			require.True(t, ok)
			schema := proxy.Schema()
			require.NotNil(t, schema)
			require.NotNil(t, schema.Minimum)
			assert.Equal(t, float64(5), *schema.Minimum)
			refs := model.Index.GetAllComponentSchemas()
			require.Len(t, refs, 1)
			ref := refs["#/components/schemas/Bound"]
			require.NotNil(t, ref)
			assert.Equal(t, "Bound", ref.KeyNode.Value)
			_, min := utils.FindKeyNodeTop("minimum", utils.ResolveMergeNodes(ref.Node).Content)
			assert.Same(t, min, schema.GoLow().Minimum.ValueNode)
			after, err := yaml.Marshal(root)
			require.NoError(t, err)
			assert.Equal(t, before, after)
			rendered, err := doc.Render()
			require.NoError(t, err)
			reloaded, err := NewDocument(rendered)
			require.NoError(t, err)
			_, err = reloaded.BuildV3Model()
			require.NoError(t, err)
		})
	}
}
