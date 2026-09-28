// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
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

// When a definition fails, the pipeline stops reading. With far more definitions than the pipeline
// buffers, the feeding goroutine is left mid-send and must exit via the done channel, not leak.
func TestDefinitions_Schemas_Build_ErrorStopsFeed(t *testing.T) {
	var yml strings.Builder
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&yml, "gonna%d:\n  $ref: break\n", i)
	}

	var idxNode yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yml.String()), &idxNode))
	idx := index.NewSpecIndex(&idxNode)

	var n Definitions
	err := n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.Error(t, err)
	assert.Nil(t, n.Schemas)
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
	// maphash uses random seed per process, so just test non-empty
	assert.NotEmpty(t, low.GenerateHashString(&n))
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
