// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package high

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestExtractExtensions(t *testing.T) {
	t.Parallel()
	n := make(map[low.KeyReference[string]]low.ValueReference[any])
	n[low.KeyReference[string]{
		Value: "pb33f",
	}] = low.ValueReference[any]{
		Value: "new cowboy in town",
	}
	ext := ExtractExtensions(n)
	assert.Equal(t, "new cowboy in town", ext["pb33f"])
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
	Extensions map[low.KeyReference[string]]low.ValueReference[any]
}

func (c *child) GetExtensions() map[low.KeyReference[string]]low.ValueReference[any] {
	return c.Extensions
}

func TestUnpackExtensions(t *testing.T) {
	t.Parallel()

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

	n := make(map[low.KeyReference[string]]low.ValueReference[any])
	n[low.KeyReference[string]{
		Value: "x-rancher-a",
	}] = low.ValueReference[any]{
		ValueNode: resultA.Content[0],
	}

	n[low.KeyReference[string]{
		Value: "x-rancher-b",
	}] = low.ValueReference[any]{
		ValueNode: resultB.Content[0],
	}

	c := new(child)
	c.Extensions = n

	p := new(parent)
	p.low = c

	res, err := UnpackExtensions[textExtension, *child](p)
	assert.NoError(t, err)
	assert.NotEmpty(t, res)
	assert.Equal(t, "buckaroo", res["x-rancher-a"].Cowboy)
	assert.Equal(t, 100, res["x-rancher-a"].Power)
	assert.Equal(t, "frogman", res["x-rancher-b"].Cowboy)
	assert.Equal(t, 2, res["x-rancher-b"].Power)
}

func TestUnpackExtensions_Fail(t *testing.T) {
	t.Parallel()

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

	n := make(map[low.KeyReference[string]]low.ValueReference[any])
	n[low.KeyReference[string]{
		Value: "x-rancher-a",
	}] = low.ValueReference[any]{
		ValueNode: resultA.Content[0],
	}

	n[low.KeyReference[string]{
		Value: "x-rancher-b",
	}] = low.ValueReference[any]{
		ValueNode: resultB.Content[0],
	}

	c := new(child)
	c.Extensions = n

	p := new(parent)
	p.low = c

	res, er := UnpackExtensions[textExtension, *child](p)
	assert.Error(t, er)
	assert.Empty(t, res)
}
