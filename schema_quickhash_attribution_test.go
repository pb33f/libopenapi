// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package libopenapi

import (
	"os"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

// The quick-hash cache key identifies a schema by file plus the position of its RootNode.
// Building a schema through a $ref re-attributes its Index to the file the reference resolves
// to, while RootNode stays behind in the referring file. Pairing the resolved file's path with
// the referring file's line and column would stop identifying the node: two $refs sitting at
// the same position in different files, both pointing into one shared file, would produce the
// same key and be handed each other's hash.
//
// The fixtures put the two $refs at exactly the same line and column on purpose.
func TestQuickHashKeyDistinguishesSamePositionRefs(t *testing.T) {
	lowbase.ClearSchemaQuickHashMap()

	specBytes, err := os.ReadFile("test_specs/index_attribution/root_collide.yaml")
	require.NoError(t, err)

	doc, err := NewDocumentWithConfiguration(specBytes, &datamodel.DocumentConfiguration{
		BasePath:            "test_specs/index_attribution",
		AllowFileReferences: true,
		UseSchemaQuickHash:  true,
	})
	require.NoError(t, err)

	model, errs := doc.BuildV3Model()
	require.Nil(t, errs)

	schemaAt := func(path string) *lowbase.Schema {
		item, ok := model.Model.Paths.PathItems.Get(path)
		require.Truef(t, ok, "path %s missing", path)
		response, ok := item.Get.Responses.Codes.Get("200")
		require.True(t, ok)
		mediaType, ok := response.Content.Get("application/json")
		require.True(t, ok)
		return mediaType.Schema.GoLow().Schema()
	}

	fromA := schemaAt("/a") // resolves to shared_two.yaml#/components/schemas/A, an object
	fromB := schemaAt("/b") // resolves to shared_two.yaml#/components/schemas/B, an integer

	// Preconditions. Without these the test would pass for the wrong reason.
	require.Equal(t, fromA.RootNode.Line, fromB.RootNode.Line,
		"fixtures must place both $ref nodes on the same line")
	require.Equal(t, fromA.RootNode.Column, fromB.RootNode.Column,
		"fixtures must place both $ref nodes in the same column")
	require.Equal(t, fromA.Index.GetSpecAbsolutePath(), fromB.Index.GetSpecAbsolutePath(),
		"both must be re-attributed to the same resolved file")

	assert.NotEqual(t, fromA.QuickHash(), fromB.QuickHash(),
		"an object schema and an integer schema must not share a quick hash")
}
