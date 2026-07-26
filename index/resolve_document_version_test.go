// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

func versionTestIndex(t *testing.T, version float32) *SpecIndex {
	t.Helper()
	var rootNode yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("openapi: 3.1.0"), &rootNode))

	config := CreateOpenAPIIndexConfig()
	if version > 0 {
		config.SpecInfo = &datamodel.SpecInfo{VersionNumeric: version}
	}
	return NewSpecIndexWithConfig(&rootNode, config)
}

// ResolveDocumentVersion decides how version-sensitive keywords are read, so its precedence
// matters: a file that states its own version describes its own keywords, and only a file with
// no version of its own should inherit the document being built.
func TestSpecIndex_ResolveDocumentVersion(t *testing.T) {
	t.Run("nil index reports nothing", func(t *testing.T) {
		var idx *SpecIndex
		version, ok := idx.ResolveDocumentVersion()
		assert.False(t, ok)
		assert.Zero(t, version)
	})

	t.Run("no spec info anywhere reports nothing", func(t *testing.T) {
		idx := versionTestIndex(t, 0)
		version, ok := idx.ResolveDocumentVersion()
		assert.False(t, ok)
		assert.Zero(t, version)
	})

	t.Run("own version is used when there is no rolodex", func(t *testing.T) {
		idx := versionTestIndex(t, 3.1)
		version, ok := idx.ResolveDocumentVersion()
		assert.True(t, ok)
		assert.Equal(t, float32(3.1), version)
	})

	// A file declaring its own version wins: it may be a complete document referenced from a
	// document of a different version, and its own key describes how its keywords are written.
	t.Run("own version wins over the rolodex root", func(t *testing.T) {
		root := versionTestIndex(t, 3.0)
		child := versionTestIndex(t, 3.1)

		rolodex := NewRolodex(CreateOpenAPIIndexConfig())
		rolodex.SetRootIndex(root)
		child.SetRolodex(rolodex)

		version, ok := child.ResolveDocumentVersion()
		assert.True(t, ok)
		assert.Equal(t, float32(3.1), version,
			"a file that declares its own version must not inherit the root's")
	})

	// A bare fragment carries no version key, so its own SpecInfo leaves VersionNumeric at zero.
	// Inheriting the document being built is the right reading for it.
	t.Run("versionless file inherits the rolodex root", func(t *testing.T) {
		root := versionTestIndex(t, 3.0)
		fragment := versionTestIndex(t, 0)

		rolodex := NewRolodex(CreateOpenAPIIndexConfig())
		rolodex.SetRootIndex(root)
		fragment.SetRolodex(rolodex)

		version, ok := fragment.ResolveDocumentVersion()
		assert.True(t, ok)
		assert.Equal(t, float32(3.0), version)
	})

	// A numeric zero means the file did not declare a version. It must remain unknown so
	// version-sensitive schema keywords can infer their representation from the node shape.
	t.Run("zero remains unknown when the rolodex root has no spec info", func(t *testing.T) {
		rootNoInfo := versionTestIndex(t, 0)
		rootNoInfo.GetConfig().SpecInfo = nil

		fragment := versionTestIndex(t, 0)
		fragment.GetConfig().SpecInfo = &datamodel.SpecInfo{VersionNumeric: 0}

		rolodex := NewRolodex(CreateOpenAPIIndexConfig())
		rolodex.SetRootIndex(rootNoInfo)
		fragment.SetRolodex(rolodex)

		version, ok := fragment.ResolveDocumentVersion()
		assert.False(t, ok)
		assert.Zero(t, version)
	})

	// A rolodex with no root index at all must not panic.
	t.Run("rolodex without a root index falls through", func(t *testing.T) {
		idx := versionTestIndex(t, 0)
		idx.GetConfig().SpecInfo = &datamodel.SpecInfo{VersionNumeric: 0}
		idx.SetRolodex(NewRolodex(CreateOpenAPIIndexConfig()))

		version, ok := idx.ResolveDocumentVersion()
		assert.False(t, ok)
		assert.Zero(t, version)
	})
}
