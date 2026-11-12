// Copyright 2025 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/assert"
)

func TestMediaType_Build_ItemSchema(t *testing.T) {
	yml := `schema:
  type: array
itemSchema:
  type: object
  properties:
    id:
      type: string
    name:
      type: string
example: 
  - id: "1"
    name: "test"`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n MediaType
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)

	// Check regular schema
	assert.NotNil(t, n.Schema.Value)
	assert.Equal(t, "array", n.Schema.Value.Schema().Type.Value.A)

	// Check itemSchema
	assert.NotNil(t, n.ItemSchema.Value)
	itemSchema := n.ItemSchema.Value.Schema()
	assert.Equal(t, "object", itemSchema.Type.Value.A)
	assert.NotNil(t, itemSchema.Properties.Value)
	assert.Equal(t, 2, itemSchema.Properties.Value.Len())
}

func TestMediaType_Build_ItemEncoding(t *testing.T) {
	yml := `schema:
  type: array
itemSchema:
  type: object
  properties:
    file:
      type: string
      format: binary
    metadata:
      type: object
itemEncoding:
  contentType: image/jpeg
  headers:
    X-Custom:
      schema:
        type: string`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n MediaType
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)

	// itemEncoding is a single Encoding Object (OpenAPI 3.2)
	assert.NotNil(t, n.ItemEncoding.Value)
	assert.Equal(t, "image/jpeg", n.ItemEncoding.Value.ContentType.Value)
	assert.Equal(t, 1, n.ItemEncoding.Value.Headers.Value.Len())
}

func TestMediaType_Build_ItemSchema_Bad(t *testing.T) {
	yml := `schema:
  type: array
itemSchema:
  $ref: #bork
example: 
  - id: "1"
    name: "test"`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n MediaType
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)

	assert.NotNil(t, n.Schema.Value)
	assert.Equal(t, "array", n.Schema.Value.Schema().Type.Value.A)

	assert.NotNil(t, n.ItemSchema.Value)
	itemSchema := n.ItemSchema.Value.Schema()
	assert.Nil(t, itemSchema)
}

func TestMediaType_Build_ItemSchema_AndEncoding_Complete(t *testing.T) {
	yml := `description: Stream of JSON Lines
schema:
  type: string
  format: binary
itemSchema:
  type: object
  properties:
    timestamp:
      type: string
      format: date-time
    event:
      type: string
    data:
      type: object
itemEncoding:
  contentType: application/json
encoding:
  mainField:
    style: form
example: '{"timestamp":"2025-01-01T00:00:00Z","event":"test","data":{"key":"value"}}'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n MediaType
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)

	// Check both schema and itemSchema exist
	assert.NotNil(t, n.Schema.Value)
	assert.NotNil(t, n.ItemSchema.Value)

	// Check both encoding and itemEncoding exist
	assert.NotNil(t, n.Encoding.Value)
	assert.NotNil(t, n.ItemEncoding.Value)
	assert.Equal(t, 1, n.Encoding.Value.Len())
	assert.Equal(t, "application/json", n.ItemEncoding.Value.ContentType.Value)
}

func TestMediaType_Hash_WithItemSchema(t *testing.T) {
	yml1 := `schema:
  type: array
itemSchema:
  type: object
  properties:
    id:
      type: string`

	yml2 := `schema:
  type: array
itemSchema:
  type: object
  properties:
    id:
      type: integer`

	var idxNode1, idxNode2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml1), &idxNode1)
	_ = yaml.Unmarshal([]byte(yml2), &idxNode2)
	idx1 := index.NewSpecIndex(&idxNode1)
	idx2 := index.NewSpecIndex(&idxNode2)

	var n1, n2 MediaType
	_ = low.BuildModel(&idxNode1, &n1)
	_ = low.BuildModel(&idxNode2, &n2)
	_ = n1.Build(context.Background(), nil, idxNode1.Content[0], idx1)
	_ = n2.Build(context.Background(), nil, idxNode2.Content[0], idx2)

	// Hashes should be different due to different itemSchema
	assert.NotEqual(t, n1.Hash(), n2.Hash())
}

func TestMediaType_Hash_WithItemEncoding(t *testing.T) {
	yml1 := `schema:
  type: array
itemEncoding:
  contentType: image/jpeg`

	yml2 := `schema:
  type: array
itemEncoding:
  contentType: image/png`

	var idxNode1, idxNode2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml1), &idxNode1)
	_ = yaml.Unmarshal([]byte(yml2), &idxNode2)
	idx1 := index.NewSpecIndex(&idxNode1)
	idx2 := index.NewSpecIndex(&idxNode2)

	var n1, n2 MediaType
	_ = low.BuildModel(&idxNode1, &n1)
	_ = low.BuildModel(&idxNode2, &n2)
	_ = n1.Build(context.Background(), nil, idxNode1.Content[0], idx1)
	_ = n2.Build(context.Background(), nil, idxNode2.Content[0], idx2)

	// Hashes should be different due to different itemEncoding
	assert.NotEqual(t, n1.Hash(), n2.Hash())
}
