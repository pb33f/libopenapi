// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"context"
	"testing"

	"github.com/pkg-base/libopenapi/datamodel/low"
	"github.com/pkg-base/libopenapi/index"
	"github.com/pkg-base/yaml"
	"github.com/stretchr/testify/assert"
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestDefinitions_Hash(t *testing.T) {
	yml := `nice:
  description: rice`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Definitions
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.Equal(t, "e32477b3f3c2dc0b95126b51a34564ad19d7d0b6b43cde4783fae4c4e04dfdf6",
		low.GenerateHashString(&n))
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}
