// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestLicense_Hash(t *testing.T) {

	left := `url: https://pb33f.io
description: the ranch`

	right := `url: https://pb33f.io
description: the ranch`

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
	l.Build(nil, &lNode, &rNode, nil)
	assert.NotNil(t, l.GetRootNode())
	assert.NotNil(t, l.GetKeyNode())

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
