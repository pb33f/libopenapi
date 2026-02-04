package bundler

import (
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/bundler/test/specs/rootrefs"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundleDocument_Embedded_RootRelativeRefs(t *testing.T) {
	doc, err := libopenapi.NewDocumentWithConfiguration(rootrefs.Schema, &datamodel.DocumentConfiguration{
		BasePath:            ".",
		LocalFS:             rootrefs.Files,
		AllowFileReferences: true,
	})
	require.NoError(t, err)

	v3, err := doc.BuildV3Model()
	require.NoError(t, err)

	bundled, err := BundleDocument(&v3.Model)
	require.NoError(t, err)

	bundledStr := string(bundled)
	assert.Contains(t, bundledStr, "id:")
	assert.Contains(t, bundledStr, "type: string")
	assert.NotContains(t, bundledStr, "resources/paths/resources")
	assert.NotContains(t, bundledStr, "resources/resources")
}
