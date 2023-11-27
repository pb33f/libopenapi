// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
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

	err := sch.Build(context.Background(), &idxNode, idxNode.Content[0], nil)
	assert.NoError(t, err)

	assert.Equal(t, "db2a35dd6fb3d9481d0682571b9d687616bb2a34c1887f7863f0b2e769ca7b23",
		low.GenerateHashString(&sch))

	assert.Equal(t, "something", sch.Schema().Description.Value)
	assert.Empty(t, sch.GetSchemaReference())
	assert.NotNil(t, sch.GetKeyNode())
	assert.NotNil(t, sch.GetValueNode())
	assert.False(t, sch.IsSchemaReference())
	assert.False(t, sch.IsReference())
	assert.Empty(t, sch.GetReference())
	sch.SetReference("coffee")
	assert.Equal(t, "coffee", sch.GetReference())

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

	err := sch.Build(context.Background(), nil, idxNode.Content[0], nil)
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

	err := sch.Build(context.Background(), nil, idxNode.Content[0], nil)
	assert.NoError(t, err)
	assert.False(t, sch.IsSchemaReference())
	assert.NotNil(t, sch.Schema())
	assert.Equal(t, "6da88c34ba124c41f977db66a4fc5c1a951708d285c81bb0d47c3206f4c27ca8",
		low.GenerateHashString(&sch))
}

func TestSchemaProxy_Build_UsingMergeNodes(t *testing.T) {

	yml := `
x-common-definitions:
  life_cycle_types: &life_cycle_types_def
    type: string
    enum: ["Onboarding", "Monitoring", "Re-Assessment"]
    description: The type of life cycle
<<: *life_cycle_types_def`

	var sch SchemaProxy
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	err := sch.Build(context.Background(), nil, idxNode.Content[0], nil)
	assert.NoError(t, err)
	assert.Len(t, sch.Schema().Enum.Value, 3)
	assert.Equal(t, "The type of life cycle", sch.Schema().Description.Value)

}

func TestSchemaProxy_GetSchemaReferenceLocation(t *testing.T) {

	yml := `type: object
properties:
  name:
    type: string
    description: thing`

	var idxNodeA yaml.Node
	e := yaml.Unmarshal([]byte(yml), &idxNodeA)
	assert.NoError(t, e)

	yml = `
type: object
properties:
  name:
    type: string
    description: thang`

	var schA SchemaProxy
	var schB SchemaProxy
	var schC SchemaProxy
	var idxNodeB yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNodeB)

	c := index.CreateOpenAPIIndexConfig()
	rolo := index.NewRolodex(c)
	rolo.SetRootNode(&idxNodeA)
	_ = rolo.IndexTheRolodex()

	err := schA.Build(context.Background(), nil, idxNodeA.Content[0], rolo.GetRootIndex())
	assert.NoError(t, err)
	err = schB.Build(context.Background(), nil, idxNodeB.Content[0].Content[3].Content[1], rolo.GetRootIndex())
	assert.NoError(t, err)

	rolo.GetRootIndex().SetAbsolutePath("/rooty/rootster")
	origin := schA.GetSchemaReferenceLocation()
	assert.NotNil(t, origin)
	assert.Equal(t, "/rooty/rootster", origin.AbsoluteLocation)

	// mess things up so it cannot be found
	schA.vn = schB.vn
	origin = schA.GetSchemaReferenceLocation()
	assert.Nil(t, origin)

	// create a new index
	idx := index.NewSpecIndexWithConfig(&idxNodeB, c)
	idx.SetAbsolutePath("/boaty/mcboatface")

	// add the index to the rolodex
	rolo.AddIndex(idx)

	// can now find the origin
	origin = schA.GetSchemaReferenceLocation()
	assert.NotNil(t, origin)
	assert.Equal(t, "/boaty/mcboatface", origin.AbsoluteLocation)

	// do it again, but with no index
	err = schC.Build(context.Background(), nil, idxNodeA.Content[0], nil)
	origin = schC.GetSchemaReferenceLocation()
	assert.Nil(t, origin)

}
