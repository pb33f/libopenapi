// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package base

import (
	"testing"

	"github.com/pkg-base/libopenapi/datamodel/low"
	"github.com/pkg-base/libopenapi/index"
	"github.com/pkg-base/yaml"
	"github.com/stretchr/testify/assert"
)

func TestXML_Build(t *testing.T) {
	yml := `name: a thing
namespace: somewhere
wrapped: true`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n XML
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(&idxNode, idx)
	assert.NoError(t, err)
	assert.Equal(t, "a thing", n.Name.Value)
	assert.Equal(t, "somewhere", n.Namespace.Value)
	assert.True(t, n.Wrapped.Value)
	assert.NotNil(t, n.GetRootNode())
}
