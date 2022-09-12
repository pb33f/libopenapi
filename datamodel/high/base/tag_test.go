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

func TestNewTag(t *testing.T) {

	var cNode yaml.Node

	yml := `name: chicken
description: nuggets
externalDocs:
  url: https://pb33f.io
x-hack: code`

	_ = yaml.Unmarshal([]byte(yml), &cNode)

	var lowTag lowbase.Tag
	_ = lowmodel.BuildModel(&cNode, &lowTag)
	_ = lowTag.Build(cNode.Content[0], nil)

	highTag := NewTag(&lowTag)

	assert.Equal(t, "chicken", highTag.Name)
	assert.Equal(t, "nuggets", highTag.Description)
	assert.Equal(t, "https://pb33f.io", highTag.ExternalDocs.URL)
	assert.Equal(t, "code", highTag.Extensions["x-hack"])

	wentLow := highTag.GoLow()
	assert.Equal(t, 5, wentLow.FindExtension("x-hack").ValueNode.Line)

}
