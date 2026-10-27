package libopenapi

import (
	"fmt"
	"testing"

	"github.com/pb33f/testify/require"
)

func TestBuildV3ModelPreservesFragmentMemberNames(t *testing.T) {
	for _, tc := range []struct{ name, fragment, kind string }{
		{"empty", "", "string"},
		{"literal plus", "A+B%20C", "integer"},
		{"escaped plus", "A%2BB%20C", "integer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := fmt.Sprintf(`openapi: 3.1.1
info: {title: Pointer, version: '1'}
paths: {}
components:
  schemas:
    Holder:
      type: object
      properties:
        '': {type: string}
        'A+B C': {type: integer}
    Chosen:
      $ref: '#/components/schemas/Holder/properties/%s'
`, tc.fragment)
			doc, err := NewDocument([]byte(source))
			require.NoError(t, err)
			model, err := doc.BuildV3Model()
			require.NoError(t, err)
			chosen, ok := model.Model.Components.Schemas.Get("Chosen")
			require.True(t, ok)
			schema := chosen.Schema()
			require.NotNil(t, schema)
			require.Equal(t, []string{tc.kind}, schema.Type)
		})
	}
}
