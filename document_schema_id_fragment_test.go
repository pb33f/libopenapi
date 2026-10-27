package libopenapi

import (
	"fmt"
	"testing"

	"github.com/pb33f/testify/require"
)

func TestBuildV3ModelResolvesFragmentsWithinSchemaIDResource(t *testing.T) {
	for _, tc := range []struct {
		name, fragment string
		valid          bool
	}{
		{"resource local", "#/$defs/Target", true},
		{"outside resource", "#/components/schemas/Target", false},
		{"missing member", "#/$defs/Missing", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := fmt.Sprintf(`openapi: 3.1.1
info: {title: Example, version: '1'}
paths: {}
components:
  schemas:
    Target: {type: boolean}
    Holder:
      $id: https://example.com/holder
      type: object
      $defs:
        Target: {type: integer}
      properties:
        value:
          $ref: '%s'
`, tc.fragment)
			doc, err := NewDocument([]byte(source))
			require.NoError(t, err)
			model, err := doc.BuildV3Model()
			if !tc.valid {
				require.ErrorContains(t, err, "cannot resolve reference")
				return
			}
			require.NoError(t, err)
			holder, ok := model.Model.Components.Schemas.Get("Holder")
			require.True(t, ok)
			schema := holder.Schema()
			require.NotNil(t, schema)
			value, ok := schema.Properties.Get("value")
			require.True(t, ok)
			resolved := value.Schema()
			require.NotNil(t, resolved)
			require.Equal(t, []string{"integer"}, resolved.Type)
		})
	}
}
