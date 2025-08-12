// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"testing"

	"github.com/pkg-base/libopenapi/datamodel/low"
	"github.com/pkg-base/yaml"
	"github.com/stretchr/testify/assert"
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
