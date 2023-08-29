// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"fmt"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestPaths_Build(t *testing.T) {

	yml := `"/fresh/code":
  $ref: break`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Paths
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.Error(t, err)

}

func TestPaths_FindPathAndKey(t *testing.T) {

	yml := `/no/sleep:
  get:
    description: til brooklyn
/no/pizza:
  post:
    description: because i'm fat`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Paths
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(nil, idxNode.Content[0], idx)
	_, k := n.FindPathAndKey("/no/pizza")
	assert.Equal(t, "because i'm fat", k.Value.Post.Value.Description.Value)

	_, k = n.FindPathAndKey("/I do not exist at all.")
	assert.Nil(t, k)
}

func TestPaths_Hash(t *testing.T) {

	yml := `/data/dog:
  get:
    description: does data kinda, ish.
/snow/flake:
  get:
    description: does data
/spl/unk:
  get:
    description: does data the best
x-milk: creamy`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Paths
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(nil, idxNode.Content[0], idx)

	yml2 := `x-milk: creamy
/spl/unk:
  get:
    description: does data the best
/data/dog:
  get:
    description: does data kinda, ish.
/snow/flake:
  get:
    description: does data
`

	var idxNode2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml2), &idxNode2)
	idx2 := index.NewSpecIndex(&idxNode2)

	var n2 Paths
	_ = low.BuildModel(idxNode2.Content[0], &n2)
	_ = n2.Build(nil, idxNode2.Content[0], idx2)

	// hash
	assert.Equal(t, n.Hash(), n2.Hash())
	assert.Len(t, n.GetExtensions(), 1)

}

// Test parse failure among many paths.
// This stresses `TranslatePipeline`'s error handling.
func TestPaths_Build_Fail_Many(t *testing.T) {
	var yml string
	for i := 0; i < 1000; i++ {
		format := `"/fresh/code%d":
  parameters:
    $ref: break
`
		yml += fmt.Sprintf(format, i)
	}

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Paths
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}
