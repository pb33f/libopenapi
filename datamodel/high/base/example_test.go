// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"fmt"
	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestNewExample(t *testing.T) {

	var cNode yaml.Node

	yml := `summary: an example
description: something more
value: a thing
externalValue: https://pb33f.io
x-hack: code`

	_ = yaml.Unmarshal([]byte(yml), &cNode)

	// build low
	var lowExample lowbase.Example
	_ = lowmodel.BuildModel(&cNode, &lowExample)

	_ = lowExample.Build(cNode.Content[0], nil)

	// build high
	highExample := NewExample(&lowExample)

	assert.Equal(t, "an example", highExample.Summary)
	assert.Equal(t, "something more", highExample.Description)
	assert.Equal(t, "https://pb33f.io", highExample.ExternalValue)
	assert.Equal(t, "code", highExample.Extensions["x-hack"])
	assert.Equal(t, "a thing", highExample.Value)
	assert.Equal(t, 4, highExample.GoLow().ExternalValue.ValueNode.Line)
}

func TestExtractExamples(t *testing.T) {
	var cNode yaml.Node

	yml := `summary: herbs`

	_ = yaml.Unmarshal([]byte(yml), &cNode)

	// build low
	var lowExample lowbase.Example
	_ = lowmodel.BuildModel(&cNode, &lowExample)

	_ = lowExample.Build(cNode.Content[0], nil)

	examplesMap := make(map[lowmodel.KeyReference[string]]lowmodel.ValueReference[*lowbase.Example])
	examplesMap[lowmodel.KeyReference[string]{
		Value: "green",
	}] = lowmodel.ValueReference[*lowbase.Example]{
		Value: &lowExample,
	}

	assert.Equal(t, "herbs", ExtractExamples(examplesMap)["green"].Summary)

}

func ExampleNewExample() {

	// create some example yaml (or can be JSON, it does not matter)
	yml := `summary: something interesting
description: something more interesting with detail
externalValue: https://pb33f.io
x-hack: code`

	// unmarshal into a *yaml.Node
	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	// build low-level example
	var lowExample lowbase.Example
	_ = lowmodel.BuildModel(&node, &lowExample)

	// build out low-level example
	_ = lowExample.Build(node.Content[0], nil)

	// create a new high-level example
	highExample := NewExample(&lowExample)

	fmt.Print(highExample.ExternalValue)
	// Output: https://pb33f.io

}