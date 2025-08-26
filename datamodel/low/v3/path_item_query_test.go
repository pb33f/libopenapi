// Copyright 2024 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestPathItem_BuildQuery(t *testing.T) {
	yml := `query:
  summary: Query resources
  description: Query resources with complex criteria
  operationId: queryResources
  requestBody:
    required: true
    content:
      application/json:
        schema:
          type: object
          properties:
            filters:
              type: array
              items:
                type: string
  responses:
    '200':
      description: Query results
      content:
        application/json:
          schema:
            type: array
            items:
              type: object`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())
	ctx := context.Background()

	var n PathItem
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(ctx, nil, idxNode.Content[0], idx)
	assert.NoError(t, err)

	assert.NotNil(t, n.Query.Value)
	assert.Equal(t, "Query resources", n.Query.Value.Summary.Value)
	assert.Equal(t, "Query resources with complex criteria", n.Query.Value.Description.Value)
	assert.Equal(t, "queryResources", n.Query.Value.OperationId.Value)
	assert.NotNil(t, n.Query.Value.RequestBody.Value)
	assert.True(t, n.Query.Value.RequestBody.Value.Required.Value)
}

func TestPathItem_HashWithQuery(t *testing.T) {
	yml1 := `query:
  summary: Query resources
  operationId: queryResources
  responses:
    '200':
      description: OK`

	yml2 := `query:
  summary: Query different resources
  operationId: queryResources
  responses:
    '200':
      description: OK`

	var idxNode1, idxNode2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml1), &idxNode1)
	_ = yaml.Unmarshal([]byte(yml2), &idxNode2)

	idx1 := index.NewSpecIndexWithConfig(&idxNode1, index.CreateOpenAPIIndexConfig())
	idx2 := index.NewSpecIndexWithConfig(&idxNode2, index.CreateOpenAPIIndexConfig())
	ctx := context.Background()

	var n1, n2 PathItem
	_ = low.BuildModel(&idxNode1, &n1)
	_ = low.BuildModel(&idxNode2, &n2)

	_ = n1.Build(ctx, nil, idxNode1.Content[0], idx1)
	_ = n2.Build(ctx, nil, idxNode2.Content[0], idx2)

	// Different summaries should produce different hashes
	hash1 := n1.Hash()
	hash2 := n2.Hash()
	assert.NotEqual(t, hash1, hash2)
}

func TestPathItem_MultipleOperationsIncludingQuery(t *testing.T) {
	yml := `get:
  summary: Get resource
  operationId: getResource
  responses:
    '200':
      description: OK
post:
  summary: Create resource
  operationId: createResource
  responses:
    '201':
      description: Created
query:
  summary: Query resources
  operationId: queryResources
  requestBody:
    required: true
    content:
      application/json:
        schema:
          type: object
  responses:
    '200':
      description: OK`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())
	ctx := context.Background()

	var n PathItem
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(ctx, nil, idxNode.Content[0], idx)
	assert.NoError(t, err)

	// Verify all operations are present
	assert.NotNil(t, n.Get.Value)
	assert.Equal(t, "Get resource", n.Get.Value.Summary.Value)

	assert.NotNil(t, n.Post.Value)
	assert.Equal(t, "Create resource", n.Post.Value.Summary.Value)

	assert.NotNil(t, n.Query.Value)
	assert.Equal(t, "Query resources", n.Query.Value.Summary.Value)
	assert.NotNil(t, n.Query.Value.RequestBody.Value)
}