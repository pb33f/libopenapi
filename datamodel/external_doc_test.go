package datamodel

import (
	"github.com/pb33f/libopenapi/datamodel/low/3.0"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestExternalDoc_Build(t *testing.T) {

	yml := `url: https://pb33f.io
description: the ranch
x-b33f: princess`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n v3.ExternalDoc
	err := v3.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "https://pb33f.io", n.URL.Value)
	assert.Equal(t, "the ranch", n.Description.Value)
	ext := n.FindExtension("x-b33f")
	assert.NotNil(t, ext)
	assert.Equal(t, "princess", ext.Value)

}
