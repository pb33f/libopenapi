// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"fmt"
	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"strings"
	"testing"
)

func TestNewExternalDoc(t *testing.T) {

	var cNode yaml.Node

	yml := `description: hack code
url: https://pb33f.io
x-hack: code`

	_ = yaml.Unmarshal([]byte(yml), &cNode)

	var lowExt lowbase.ExternalDoc
	_ = lowmodel.BuildModel(cNode.Content[0], &lowExt)

	_ = lowExt.Build(cNode.Content[0], nil)

	highExt := NewExternalDoc(&lowExt)

	assert.Equal(t, "hack code", highExt.Description)
	assert.Equal(t, "https://pb33f.io", highExt.URL)
	assert.Equal(t, "code", highExt.Extensions["x-hack"])

	wentLow := highExt.GoLow()
	assert.Equal(t, 2, wentLow.URL.ValueNode.Line)
	assert.Len(t, highExt.GetExtensions(), 1)

	// render the high-level object as YAML
	rendered, _ := highExt.Render()
	assert.Equal(t, strings.TrimSpace(string(rendered)), yml)

}

func ExampleNewExternalDoc() {

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
	_ = lowExt.Build(node.Content[0], nil)

	// create new high-level ExternalDoc
	highExt := NewExternalDoc(&lowExt)

	// print out a extension
	fmt.Print(highExt.Extensions["x-hack"])
	// Output: code
}
