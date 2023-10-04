package libopenapi

import (
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestFileRef(t *testing.T) {
	var expectedRender = `schema:
    type: object
    properties:
        one:
            type: object
            title: One
            additionalProperties: false
            properties:
                name:
                    type: string
                    minLength: 2
                    maxLength: 2
        two:
            type: object
            title: Two
            additionalProperties: false
            properties:
                name:
                    type: string
                    minLength: 2
                    maxLength: 2
        three:
            type: object
            title: Three
            additionalProperties: false
            properties:
                name:
                    type: string
                    minLength: 2
                    maxLength: 2`

	file, _ := os.ReadFile("test_specs/file-ref/index.yaml")

	config := &datamodel.DocumentConfiguration{
		AllowFileReferences:   true,
		AllowRemoteReferences: true,
		BasePath:              "test_specs/file-ref/",
	}
	doc, err := NewDocumentWithConfiguration(file, config)
	require.NoError(t, err)

	m, errs := doc.BuildV3Model()
	require.Empty(t, errs)
	require.NotEmpty(t, m)

	// Fetch the rendered inline schema for the path, we only need to compare that.
	out, err := m.Model.Paths.PathItems["/"].Post.RequestBody.Content["application/json"].RenderInline()
	require.NoError(t, err)

	assert.Equal(t, expectedRender, string(out))
}
