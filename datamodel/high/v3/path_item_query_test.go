// Copyright 2024 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	lowV3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestNewPathItem_WithQuery(t *testing.T) {
	yml := `query:
  summary: Query resources with complex criteria
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
  responses:
    '200':
      description: Query results`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())
	ctx := context.Background()

	var n lowV3.PathItem
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(ctx, nil, idxNode.Content[0], idx)
	assert.NoError(t, err)

	// Create high-level PathItem
	highPath := NewPathItem(&n)

	assert.NotNil(t, highPath.Query)
	assert.Equal(t, "Query resources with complex criteria", highPath.Query.Summary)
	assert.Equal(t, "queryResources", highPath.Query.OperationId)
	assert.NotNil(t, highPath.Query.RequestBody)
	assert.NotNil(t, highPath.Query.RequestBody.Required)
	assert.True(t, *highPath.Query.RequestBody.Required)
}

func TestPathItem_GetOperations_WithQuery(t *testing.T) {
	yml := `get:
  summary: Get resource
  operationId: getResource
post:
  summary: Create resource
  operationId: createResource
query:
  summary: Query resources
  operationId: queryResources
  requestBody:
    required: true
    content:
      application/json:
        schema:
          type: object`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())
	ctx := context.Background()

	var n lowV3.PathItem
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(ctx, nil, idxNode.Content[0], idx)
	assert.NoError(t, err)

	// Create high-level PathItem
	highPath := NewPathItem(&n)

	// Get operations map
	ops := highPath.GetOperations()

	// Should have 3 operations
	assert.Equal(t, 3, ops.Len())

	// Check that query operation is included
	queryOp, ok := ops.Get(lowV3.QueryLabel)
	assert.True(t, ok)
	assert.NotNil(t, queryOp)
	assert.Equal(t, "Query resources", queryOp.Summary)
	assert.Equal(t, "queryResources", queryOp.OperationId)

	// Check other operations
	getOp, ok := ops.Get(lowV3.GetLabel)
	assert.True(t, ok)
	assert.NotNil(t, getOp)
	assert.Equal(t, "Get resource", getOp.Summary)

	postOp, ok := ops.Get(lowV3.PostLabel)
	assert.True(t, ok)
	assert.NotNil(t, postOp)
	assert.Equal(t, "Create resource", postOp.Summary)
}

func TestPathItem_MarshalYAML_WithQuery(t *testing.T) {
	yml := `description: Resource operations
get:
  summary: Get resource
query:
  summary: Query resources
  requestBody:
    required: true
    content:
      application/json:
        schema:
          type: object`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())
	ctx := context.Background()

	var n lowV3.PathItem
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(ctx, nil, idxNode.Content[0], idx)
	assert.NoError(t, err)

	// Create high-level PathItem
	highPath := NewPathItem(&n)

	// Render to YAML
	rendered, err := highPath.Render()
	assert.NoError(t, err)

	// Parse the rendered YAML
	var parsed map[string]interface{}
	err = yaml.Unmarshal(rendered, &parsed)
	assert.NoError(t, err)

	// Verify query operation is present
	queryOp, ok := parsed["query"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "Query resources", queryOp["summary"])

	// Verify requestBody is present in query operation
	reqBody, ok := queryOp["requestBody"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, true, reqBody["required"])
}
