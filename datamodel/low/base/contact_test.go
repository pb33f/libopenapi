// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestContact_Hash(t *testing.T) {
	left := `url: https://pb33f.io
description: the ranch
email: buckaroo@pb33f.io
x-cake: yummy`

	right := `url: https://pb33f.io
description: the ranch
email: buckaroo@pb33f.io
x-beer: cold`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc Contact
	var rDoc Contact
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)

	assert.Equal(t, lDoc.Hash(), rDoc.Hash())

	c := Contact{}
	c.Build(context.Background(), lNode.Content[0], rNode.Content[0], nil)
	assert.NotNil(t, c.GetRootNode())
	assert.NotNil(t, c.GetKeyNode())
	assert.Equal(t, 1, c.GetExtensions().Len())
	assert.Equal(t, 1, c.GetExtensions().Len())
	assert.Nil(t, c.GetIndex())
	assert.NotNil(t, c.GetContext())
}

func TestContact_Build_ScalarRoot(t *testing.T) {
	var scalar yaml.Node
	_ = yaml.Unmarshal([]byte("hello"), &scalar)

	var c Contact
	err := low.BuildModel(scalar.Content[0], &c)
	assert.NoError(t, err)

	err = c.Build(context.Background(), nil, scalar.Content[0], nil)
	assert.NoError(t, err)

	nodes := c.GetNodes()
	assert.Len(t, nodes[scalar.Content[0].Line], 1)
	assert.Equal(t, "hello", nodes[scalar.Content[0].Line][0].Value)
}
