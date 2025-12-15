// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package high

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestExtractExtensions(t *testing.T) {
	n := orderedmap.New[low.KeyReference[string], low.ValueReference[*yaml.Node]]()
	n.Set(low.KeyReference[string]{
		Value: "pb33f",
	}, low.ValueReference[*yaml.Node]{
		Value: utils.CreateStringNode("new cowboy in town"),
	})
	ext := ExtractExtensions(n)

	var pb33f string
	err := ext.GetOrZero("pb33f").Decode(&pb33f)
	require.NoError(t, err)

	assert.Equal(t, "new cowboy in town", pb33f)
}

type textExtension struct {
	Cowboy string
	Power  int
}

type parent struct {
	low *child
}

func (p *parent) GoLow() *child {
	return p.low
}

type child struct {
	Extensions *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]]
}

func (c *child) GetExtensions() *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]] {
	return c.Extensions
}

func TestUnpackExtensions(t *testing.T) {
	var resultA, resultB yaml.Node

	ymlA := `
cowboy: buckaroo
power: 100`

	ymlB := `
cowboy: frogman
power: 2`

	err := yaml.Unmarshal([]byte(ymlA), &resultA)
	assert.NoError(t, err)
	err = yaml.Unmarshal([]byte(ymlB), &resultB)
	assert.NoError(t, err)

	n := orderedmap.New[low.KeyReference[string], low.ValueReference[*yaml.Node]]()
	n.Set(low.KeyReference[string]{
		Value: "x-rancher-a",
	}, low.ValueReference[*yaml.Node]{
		ValueNode: resultA.Content[0],
	})

	n.Set(low.KeyReference[string]{
		Value: "x-rancher-b",
	}, low.ValueReference[*yaml.Node]{
		ValueNode: resultB.Content[0],
	})

	c := new(child)
	c.Extensions = n

	p := new(parent)
	p.low = c

	res, err := UnpackExtensions[textExtension, *child](p)
	assert.NoError(t, err)
	assert.NotEmpty(t, res)
	assert.Equal(t, "buckaroo", res.GetOrZero("x-rancher-a").Cowboy)
	assert.Equal(t, 100, res.GetOrZero("x-rancher-a").Power)
	assert.Equal(t, "frogman", res.GetOrZero("x-rancher-b").Cowboy)
	assert.Equal(t, 2, res.GetOrZero("x-rancher-b").Power)
}

func TestUnpackExtensions_Fail(t *testing.T) {
	var resultA, resultB yaml.Node

	ymlA := `
cowboy: buckaroo
power: 100`

	// this is incorrect types, unpacking will fail.
	ymlB := `
cowboy: 0
power: hello`

	err := yaml.Unmarshal([]byte(ymlA), &resultA)
	assert.NoError(t, err)
	err = yaml.Unmarshal([]byte(ymlB), &resultB)
	assert.NoError(t, err)

	n := orderedmap.New[low.KeyReference[string], low.ValueReference[*yaml.Node]]()
	n.Set(low.KeyReference[string]{
		Value: "x-rancher-a",
	}, low.ValueReference[*yaml.Node]{
		ValueNode: resultA.Content[0],
	})

	n.Set(low.KeyReference[string]{
		Value: "x-rancher-b",
	}, low.ValueReference[*yaml.Node]{
		ValueNode: resultB.Content[0],
	})

	c := new(child)
	c.Extensions = n

	p := new(parent)
	p.low = c

	res, er := UnpackExtensions[textExtension, *child](p)
	assert.Error(t, er)
	assert.Empty(t, res)
}

func TestRenderInline(t *testing.T) {
	// Create a simple struct to test rendering
	type testStruct struct {
		Name    string `yaml:"name,omitempty"`
		Version string `yaml:"version,omitempty"`
	}

	high := &testStruct{Name: "test", Version: "1.0.0"}
	result, err := RenderInline(high, nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify the result is a yaml.Node
	node, ok := result.(*yaml.Node)
	require.True(t, ok)
	assert.Equal(t, yaml.MappingNode, node.Kind)
}

func TestRenderInline_WithLow(t *testing.T) {
	// Test with both high and low models (typical use case)
	type testStruct struct {
		Name string `yaml:"name,omitempty"`
	}

	high := &testStruct{Name: "test"}
	low := &testStruct{Name: "low-test"} // low model, should be passed through

	result, err := RenderInline(high, low)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}
