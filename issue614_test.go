// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// https://pb33f.io
// SPDX-License-Identifier: MIT
package libopenapi

import (
	"os"
	"runtime"
	"testing"
	"time"
	"weak"

	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

// A document the caller has dropped must be reclaimable without ClearAllCaches: no process-wide cache may keep
// its index or YAML tree reachable, whatever the document was used for before it was dropped.
func TestIssue614DroppedDocumentIsCollected(t *testing.T) {
	original, err := os.ReadFile("test_specs/burgershop.openapi.yaml")
	require.NoError(t, err)
	modified, err := os.ReadFile("test_specs/burgershop.openapi-modified.yaml")
	require.NoError(t, err)

	var specIndex weak.Pointer[index.SpecIndex]
	var rootNode weak.Pointer[yaml.Node]

	func() {
		doc, err := NewDocument(original)
		require.NoError(t, err)
		model, err := doc.BuildV3Model()
		require.NoError(t, err)

		// hash every schema (as request validators do), render, and compare against another document.
		for _, schema := range model.Model.Components.Schemas.FromOldest() {
			_ = schema.Schema().GoLow().Hash()
		}
		_, err = model.Model.RenderInline()
		require.NoError(t, err)
		other, err := NewDocument(modified)
		require.NoError(t, err)
		changes, errs := CompareDocuments(doc, other)
		require.Empty(t, errs)
		require.NotNil(t, changes)

		specIndex = weak.Make(model.Index)
		rootNode = weak.Make(model.Index.GetRootNode())
		doc.Release()
	}()

	require.Eventually(t, func() bool {
		runtime.GC()
		return specIndex.Value() == nil && rootNode.Value() == nil
	}, 5*time.Second, 10*time.Millisecond, "a dropped document is still reachable from a global cache")
}
