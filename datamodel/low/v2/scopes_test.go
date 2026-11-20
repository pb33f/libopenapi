// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"context"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestScopes_Hash(t *testing.T) {
	yml := `burgers: chips
pizza: beans
x-men: needs a reboot or a refresh`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Scopes
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	yml2 := `x-men: needs a reboot or a refresh
pizza: beans
burgers: chips`

	var idxNode2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml2), &idxNode2)
	idx2 := index.NewSpecIndex(&idxNode2)

	var n2 Scopes
	_ = low.BuildModel(idxNode2.Content[0], &n2)
	_ = n2.Build(context.Background(), nil, idxNode2.Content[0], idx2)

	// hash
	assert.Equal(t, n.Hash(), n2.Hash())
	assert.Equal(t, 1, orderedmap.Len(n.GetExtensions()))
}

// a scope name that contains `x-` without starting with it is a scope, not an extension.
func TestScopes_Build_NameContainingExtensionPrefix(t *testing.T) {
	yml := `inbox-read: read the inbox
fax-send: send faxes
write: write things
x-ext: an extension`

	var idxNode yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yml), &idxNode))
	idx := index.NewSpecIndex(&idxNode)

	var n Scopes
	require.NoError(t, low.BuildModel(idxNode.Content[0], &n))
	require.NoError(t, n.Build(context.Background(), nil, idxNode.Content[0], idx))

	require.Equal(t, 3, orderedmap.Len(n.Values))
	assert.Equal(t, "read the inbox", n.FindScope("inbox-read").Value)
	assert.Equal(t, "send faxes", n.FindScope("fax-send").Value)
	assert.Equal(t, "write things", n.FindScope("write").Value)
	assert.Equal(t, 1, orderedmap.Len(n.Extensions))
}
