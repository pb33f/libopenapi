// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
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
	_ = lowmodel.BuildModel(cNode.Content[0], &lowExample)

	_ = lowExample.Build(context.Background(), &cNode, cNode.Content[0], nil)

	// build high
	highExample := NewExample(&lowExample)

	var xHack string
	_ = highExample.Extensions.GetOrZero("x-hack").Decode(&xHack)

	var example string
	_ = highExample.Value.Decode(&example)

	assert.Equal(t, "an example", highExample.Summary)
	assert.Equal(t, "something more", highExample.Description)
	assert.Equal(t, "https://pb33f.io", highExample.ExternalValue)
	assert.Equal(t, "code", xHack)
	assert.Equal(t, "a thing", example)
	assert.Equal(t, 4, highExample.GoLow().ExternalValue.ValueNode.Line)
	assert.NotNil(t, highExample.GoLowUntyped())

	// render the example as YAML
	rendered, _ := highExample.Render()
	assert.Equal(t, yml, strings.TrimSpace(string(rendered)))

	// render the example as JSON
	var err error
	rendered, err = json.Marshal(highExample)
	assert.NoError(t, err)

	var j map[string]any
	_ = json.Unmarshal(rendered, &j)

	assert.Equal(t, "an example", j["summary"])
	assert.Equal(t, "something more", j["description"])
	assert.Equal(t, "https://pb33f.io", j["externalValue"])
	assert.Equal(t, "code", j["x-hack"])
	assert.Equal(t, "a thing", j["value"])
}

func TestExtractExamples(t *testing.T) {
	var cNode yaml.Node

	yml := `summary: herbs`

	_ = yaml.Unmarshal([]byte(yml), &cNode)

	// build low
	var lowExample lowbase.Example
	_ = lowmodel.BuildModel(cNode.Content[0], &lowExample)

	_ = lowExample.Build(context.Background(), nil, cNode.Content[0], nil)

	examplesMap := orderedmap.New[lowmodel.KeyReference[string], lowmodel.ValueReference[*lowbase.Example]]()
	examplesMap.Set(
		lowmodel.KeyReference[string]{Value: "green"},
		lowmodel.ValueReference[*lowbase.Example]{Value: &lowExample},
	)

	assert.Equal(t, "herbs", ExtractExamples(examplesMap).GetOrZero("green").Summary)
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
	_ = lowmodel.BuildModel(node.Content[0], &lowExample)

	// build out low-level example
	_ = lowExample.Build(context.Background(), nil, node.Content[0], nil)

	// create a new high-level example
	highExample := NewExample(&lowExample)

	fmt.Print(highExample.ExternalValue)
	// Output: https://pb33f.io
}

func TestExample_GoLow(t *testing.T) {
	var example *Example
	assert.Nil(t, example.GoLow())
	assert.Nil(t, example.GoLowUntyped())
}
