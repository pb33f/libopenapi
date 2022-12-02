// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestSchemaProxy_Build(t *testing.T) {

	yml := `x-windows: washed
description: something`

	var sch SchemaProxy
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	err := sch.Build(idxNode.Content[0], nil)
	assert.NoError(t, err)

	assert.Equal(t, "db2a35dd6fb3d9481d0682571b9d687616bb2a34c1887f7863f0b2e769ca7b23",
		low.GenerateHashString(&sch))

	assert.Equal(t, "something", sch.Schema().Description.Value)
	assert.Empty(t, sch.GetSchemaReference())
	assert.NotNil(t, sch.GetValueNode())
	assert.False(t, sch.IsSchemaReference())

	// already rendered, should spit out the same
	assert.Equal(t, "db2a35dd6fb3d9481d0682571b9d687616bb2a34c1887f7863f0b2e769ca7b23",
		low.GenerateHashString(&sch))

	assert.Len(t, sch.Schema().GetExtensions(), 1)

}

func TestSchemaProxy_Build_CheckRef(t *testing.T) {

	yml := `$ref: wat`

	var sch SchemaProxy
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	err := sch.Build(idxNode.Content[0], nil)
	assert.NoError(t, err)
	assert.True(t, sch.IsSchemaReference())
	assert.Equal(t, "wat", sch.GetSchemaReference())
	assert.Equal(t, "f00a787f7492a95e165b470702f4fe9373583fbdc025b2c8bdf0262cc48fcff4",
		low.GenerateHashString(&sch))
}

func TestSchemaProxy_Build_HashInline(t *testing.T) {

	yml := `type: int`

	var sch SchemaProxy
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	err := sch.Build(idxNode.Content[0], nil)
	assert.NoError(t, err)
	assert.False(t, sch.IsSchemaReference())
	assert.NotNil(t, sch.Schema())
	assert.Equal(t, "6da88c34ba124c41f977db66a4fc5c1a951708d285c81bb0d47c3206f4c27ca8",
		low.GenerateHashString(&sch))
}
