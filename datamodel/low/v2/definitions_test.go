// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestDefinitions_Schemas_Build_Error(t *testing.T) {
	t.Parallel()
	yml := `gonna:
  $ref: break`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Definitions
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.Error(t, err)

}

func TestDefinitions_Parameters_Build_Error(t *testing.T) {
	t.Parallel()
	yml := `gonna:
  $ref: break`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n ParameterDefinitions
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.Error(t, err)

}

func TestDefinitions_Hash(t *testing.T) {
	t.Parallel()
	yml := `nice:
  description: rice`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Definitions
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	_ = n.Build(nil, idxNode.Content[0], idx)
	assert.Equal(t, "26d23786e6873e1a337f8e9be85f7de1490e4ff6cd303c3b15e593a25a6a149d",
		low.GenerateHashString(&n))

}

func TestDefinitions_Responses_Build_Error(t *testing.T) {
	t.Parallel()
	yml := `gonna:
  $ref: break`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n ResponsesDefinitions
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.Error(t, err)

}

func TestDefinitions_Security_Build_Error(t *testing.T) {
	t.Parallel()
	yml := `gonna:
  $ref: break`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n SecurityDefinitions
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.Error(t, err)

}
