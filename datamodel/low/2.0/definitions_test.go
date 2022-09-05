// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestDefinitions_Schemas_Build_Error(t *testing.T) {

	yml := `gonna:
  $ref: break`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Definitions
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)

}

func TestDefinitions_Parameters_Build_Error(t *testing.T) {

	yml := `gonna:
  $ref: break`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n ParameterDefinitions
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)

}

func TestDefinitions_Responses_Build_Error(t *testing.T) {

	yml := `gonna:
  $ref: break`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n ResponsesDefinitions
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)

}

func TestDefinitions_Security_Build_Error(t *testing.T) {

	yml := `gonna:
  $ref: break`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n SecurityDefinitions
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)

}
