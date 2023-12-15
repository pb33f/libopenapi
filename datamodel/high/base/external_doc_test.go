// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"strings"
	"testing"

	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestNewExternalDoc(t *testing.T) {
	var cNode yaml.Node

	yml := `description: hack code
url: https://pb33f.io
x-hack: code`

	_ = yaml.Unmarshal([]byte(yml), &cNode)

	var lowExt lowbase.ExternalDoc
	_ = lowmodel.BuildModel(cNode.Content[0], &lowExt)

	_ = lowExt.Build(context.Background(), nil, cNode.Content[0], nil)

	highExt := NewExternalDoc(&lowExt)

	var xHack string
	_ = highExt.Extensions.GetOrZero("x-hack").Decode(&xHack)

	assert.Equal(t, "hack code", highExt.Description)
	assert.Equal(t, "https://pb33f.io", highExt.URL)
	assert.Equal(t, "code", xHack)

	wentLow := highExt.GoLow()
	assert.Equal(t, 2, wentLow.URL.ValueNode.Line)
	assert.Equal(t, 1, orderedmap.Len(highExt.GetExtensions()))

	// render the high-level object as YAML
	rendered, _ := highExt.Render()
	assert.Equal(t, strings.TrimSpace(string(rendered)), yml)
}

func TestExampleNewExternalDoc(t *testing.T) {
	// create a new external documentation spec reference
	// this can be YAML or JSON.
	yml := `description: hack code docs
url: https://pb33f.io/docs
x-hack: code`

	// unmarshal the raw bytes into a *yaml.Node
	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	// build low-level ExternalDoc
	var lowExt lowbase.ExternalDoc
	_ = lowmodel.BuildModel(node.Content[0], &lowExt)

	// build out low-level properties (like extensions)
	_ = lowExt.Build(context.Background(), nil, node.Content[0], nil)

	// create new high-level ExternalDoc
	highExt := NewExternalDoc(&lowExt)

	var xHack string
	_ = highExt.Extensions.GetOrZero("x-hack").Decode(&xHack)

	assert.Equal(t, "code", xHack)
}
