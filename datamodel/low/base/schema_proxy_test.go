// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestSchemaProxy_Build(t *testing.T) {
	yml := `x-windows: washed
description: something`

	var sch SchemaProxy
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	ctx := context.WithValue(context.Background(), "key", "value")

	err := sch.Build(ctx, &idxNode, idxNode.Content[0], nil)
	assert.NoError(t, err)
	assert.Equal(t, "value", sch.GetContext().Value("key"))

	assert.Equal(t, "e20c009d370944d177c0b46e8fa29e15fadc3a6f9cca6bb251ff9e120265fc96",
		low.GenerateHashString(&sch))

	assert.Equal(t, "something", sch.Schema().Description.GetValue())
	assert.Empty(t, sch.GetReference())
	assert.NotNil(t, sch.GetKeyNode())
	assert.NotNil(t, sch.GetValueNode())
	assert.False(t, sch.IsReference())
	sch.SetReference("coffee", nil)
	assert.Equal(t, "coffee", sch.GetReference())

	// already rendered, should spit out the same
	assert.Equal(t, "37290d74ac4d186e3a8e5785d259d2ec04fac91ae28092e7620ec8bc99e830aa",
		low.GenerateHashString(&sch))

	assert.Equal(t, 1, orderedmap.Len(sch.Schema().GetExtensions()))
	assert.Nil(t, sch.GetIndex())
}

func TestSchemaProxy_Build_CheckRef(t *testing.T) {
	yml := `$ref: wat`

	var sch SchemaProxy
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	err := sch.Build(context.Background(), nil, idxNode.Content[0], nil)
	assert.NoError(t, err)
	assert.True(t, sch.IsReference())
	assert.Equal(t, "wat", sch.GetReference())
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
	assert.False(t, sch.IsReference())
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
	_ = rolo.IndexTheRolodex(context.Background())

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
	assert.NoError(t, err)
	origin = schC.GetSchemaReferenceLocation()
	assert.Nil(t, origin)
}

func TestSchemaProxy_Build_HashFail(t *testing.T) {
	sp := new(SchemaProxy)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	idx := index.NewSpecIndexWithConfig(nil, &index.SpecIndexConfig{Logger: logger})
	sp.idx = idx
	v := sp.Hash()
	assert.Equal(t, [32]byte{}, v)
}

func TestSchemaProxy_AddNodePassthrough(t *testing.T) {
	yml := `type: int
description: cakes`

	sch := SchemaProxy{}
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	err := sch.Build(context.Background(), nil, idxNode.Content[0], nil)
	assert.NoError(t, err)

	n, f := sch.Nodes.Load(3)
	assert.False(t, f)
	assert.Nil(t, n)

	sch.AddNode(3, &yaml.Node{Value: "3"})
	s := sch.Schema()
	sch.AddNode(4, &yaml.Node{Value: "4"})

	n, f = s.Nodes.Load(3)
	assert.True(t, f)
	assert.NotNil(t, n)

	n, f = s.Nodes.Load(4)
	assert.True(t, f)
	assert.NotNil(t, n)
}
