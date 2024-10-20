// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestLicense_Hash(t *testing.T) {

	left := `url: https://pb33f.io
description: the ranch
x-happy: dance`

	right := `url: https://pb33f.io
description: the ranch
x-drink: beer`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc License
	var rDoc License
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)

	assert.Equal(t, lDoc.Hash(), rDoc.Hash())

	l := License{}
	l.Build(context.Background(), lNode.Content[0], rNode.Content[0], nil)
	assert.NotNil(t, l.GetRootNode())
	assert.NotNil(t, l.GetKeyNode())
	assert.Equal(t, 1, l.GetExtensions().Len())
	assert.Nil(t, l.GetIndex())
	assert.NotNil(t, l.GetContext())
}

func TestLicense_WithIdentifier_Hash(t *testing.T) {

	left := `identifier: MIT
description: the ranch`

	right := `identifier: MIT
description: the ranch`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc License
	var rDoc License
	err := low.BuildModel(lNode.Content[0], &lDoc)
	assert.NoError(t, err)

	err = low.BuildModel(rNode.Content[0], &rDoc)
	assert.NoError(t, err)

	assert.Equal(t, lDoc.Hash(), rDoc.Hash())

}
