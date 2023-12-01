// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"fmt"
	"strings"
	"testing"

	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
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
	_ = lowmodel.BuildModel(cNode.Content[0], &lowTag)
	_ = lowTag.Build(context.Background(), nil, cNode.Content[0], nil)

	highTag := NewTag(&lowTag)

	var xHack string
	_ = highTag.Extensions.GetOrZero("x-hack").Decode(&xHack)

	assert.Equal(t, "chicken", highTag.Name)
	assert.Equal(t, "nuggets", highTag.Description)
	assert.Equal(t, "https://pb33f.io", highTag.ExternalDocs.URL)
	assert.Equal(t, "code", xHack)

	wentLow := highTag.GoLow()
	assert.Equal(t, 5, wentLow.FindExtension("x-hack").ValueNode.Line)
	assert.NotNil(t, highTag.GoLowUntyped())

	// render the tag as YAML
	highTagBytes, _ := highTag.Render()
	assert.Equal(t, strings.TrimSpace(string(highTagBytes)), yml)
}

func TestTag_RenderInline(t *testing.T) {
	tag := &Tag{
		Name: "cake",
	}

	tri, _ := tag.RenderInline()

	assert.Equal(t, "name: cake", strings.TrimSpace(string(tri)))
}

func ExampleNewTag() {
	// create an example schema object
	// this can be either JSON or YAML.
	yml := `
name: Purchases
description: All kinds of purchase related operations
externalDocs:
  url: https://pb33f.io/purchases
x-hack: code`

	// unmarshal raw bytes
	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	// build out the low-level model
	var lowTag lowbase.Tag
	_ = lowmodel.BuildModel(node.Content[0], &lowTag)
	_ = lowTag.Build(context.Background(), nil, node.Content[0], nil)

	// build the high level tag
	highTag := NewTag(&lowTag)

	// print out the tag name
	fmt.Print(highTag.Name)
	// Output: Purchases
}
