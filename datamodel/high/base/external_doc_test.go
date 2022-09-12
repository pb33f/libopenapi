// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestNewExternalDoc(t *testing.T) {

	var cNode yaml.Node

	yml := `description: hack code
url: https://pb33f.io
x-hack: code`

	_ = yaml.Unmarshal([]byte(yml), &cNode)

	var lowExt lowbase.ExternalDoc
	_ = lowmodel.BuildModel(&cNode, &lowExt)

	_ = lowExt.Build(cNode.Content[0], nil)

	highExt := NewExternalDoc(&lowExt)

	assert.Equal(t, "hack code", highExt.Description)
	assert.Equal(t, "https://pb33f.io", highExt.URL)
	assert.Equal(t, "code", highExt.Extensions["x-hack"])

	wentLow := highExt.GoLow()
	assert.Equal(t, 2, wentLow.URL.ValueNode.Line)
}
