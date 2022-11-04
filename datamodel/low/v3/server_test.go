// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestServer_Build(t *testing.T) {

	yml := `url: https://pb33f.io
description: high quality software for developers.
variables:
  var1: 
    default: hello
    description: a var
    enum: [one, two]`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Server
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "https://pb33f.io", n.URL.Value)
	assert.Equal(t, "high quality software for developers.", n.Description.Value)
	assert.Equal(t, "hello", n.FindVariable("var1").Value.Default.Value)
	assert.Equal(t, "a var", n.FindVariable("var1").Value.Description.Value)

}

func TestServer_Build_NoVars(t *testing.T) {

	yml := `url: https://pb33f.io
description: high quality software for developers.`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Server
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "https://pb33f.io", n.URL.Value)
	assert.Equal(t, "high quality software for developers.", n.Description.Value)
	assert.Len(t, n.Variables.Value, 0)

}
