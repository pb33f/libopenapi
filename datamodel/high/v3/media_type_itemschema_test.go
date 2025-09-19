// Copyright 2025 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	lowv3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestNewMediaType_WithItemSchema(t *testing.T) {
	yml := `schema:
  type: array
  items:
    type: string
itemSchema:
  type: object
  properties:
    id:
      type: integer
    message:
      type: string
example:
  - id: 1
    message: "Hello World"`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var lowMediaType lowv3.MediaType
	_ = low.BuildModel(&idxNode, &lowMediaType)
	_ = lowMediaType.Build(context.Background(), nil, idxNode.Content[0], idx)

	highMediaType := NewMediaType(&lowMediaType)

	// Check regular schema
	assert.NotNil(t, highMediaType.Schema)
	assert.Equal(t, "array", highMediaType.Schema.Schema().Type[0])

	// Check itemSchema
	assert.NotNil(t, highMediaType.ItemSchema)
	assert.Equal(t, "object", highMediaType.ItemSchema.Schema().Type[0])
	assert.NotNil(t, highMediaType.ItemSchema.Schema().Properties)
	assert.Equal(t, 2, highMediaType.ItemSchema.Schema().Properties.Len())
}

func TestNewMediaType_WithItemEncoding(t *testing.T) {
	yml := `schema:
  type: string
  format: binary
itemSchema:
  type: object
  properties:
    data:
      type: string
      format: binary
    metadata:
      type: object
itemEncoding:
  data:
    contentType: application/octet-stream
    allowReserved: true
  metadata:
    contentType: application/json
    headers:
      X-Meta-Type:
        schema:
          type: string`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var lowMediaType lowv3.MediaType
	_ = low.BuildModel(&idxNode, &lowMediaType)
	_ = lowMediaType.Build(context.Background(), nil, idxNode.Content[0], idx)

	highMediaType := NewMediaType(&lowMediaType)

	// Check itemEncoding exists
	assert.NotNil(t, highMediaType.ItemEncoding)
	assert.Equal(t, 2, highMediaType.ItemEncoding.Len())

	// Check data encoding
	dataEnc := highMediaType.ItemEncoding.GetOrZero("data")
	assert.NotNil(t, dataEnc)
	assert.Equal(t, "application/octet-stream", dataEnc.ContentType)
	assert.True(t, dataEnc.AllowReserved)

	// Check metadata encoding
	metaEnc := highMediaType.ItemEncoding.GetOrZero("metadata")
	assert.NotNil(t, metaEnc)
	assert.Equal(t, "application/json", metaEnc.ContentType)
	assert.NotNil(t, metaEnc.Headers)
	assert.Equal(t, 1, metaEnc.Headers.Len())
}

func TestNewMediaType_WithBothEncodingTypes(t *testing.T) {
	yml := `schema:
  type: object
itemSchema:
  type: object
  required:
    - id
encoding:
  regularField:
    contentType: text/plain
    style: form
itemEncoding:
  streamField:
    contentType: application/json
    explode: false`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var lowMediaType lowv3.MediaType
	_ = low.BuildModel(&idxNode, &lowMediaType)
	_ = lowMediaType.Build(context.Background(), nil, idxNode.Content[0], idx)

	highMediaType := NewMediaType(&lowMediaType)

	// Check both encoding types exist
	assert.NotNil(t, highMediaType.Encoding)
	assert.NotNil(t, highMediaType.ItemEncoding)
	assert.Equal(t, 1, highMediaType.Encoding.Len())
	assert.Equal(t, 1, highMediaType.ItemEncoding.Len())

	// Check regular encoding
	regularEnc := highMediaType.Encoding.GetOrZero("regularField")
	assert.NotNil(t, regularEnc)
	assert.Equal(t, "text/plain", regularEnc.ContentType)
	assert.Equal(t, "form", regularEnc.Style)

	// Check item encoding
	itemEnc := highMediaType.ItemEncoding.GetOrZero("streamField")
	assert.NotNil(t, itemEnc)
	assert.Equal(t, "application/json", itemEnc.ContentType)
	assert.NotNil(t, itemEnc.Explode)
	assert.False(t, *itemEnc.Explode)
}

func TestNewMediaType_EmptyItemFields(t *testing.T) {
	yml := `schema:
  type: array
example: [1, 2, 3]`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var lowMediaType lowv3.MediaType
	_ = low.BuildModel(&idxNode, &lowMediaType)
	_ = lowMediaType.Build(context.Background(), nil, idxNode.Content[0], idx)

	highMediaType := NewMediaType(&lowMediaType)

	// Check that itemSchema and itemEncoding are nil when not provided
	assert.Nil(t, highMediaType.ItemSchema)
	assert.Nil(t, highMediaType.ItemEncoding)

	// Check that regular fields are still populated
	assert.NotNil(t, highMediaType.Schema)
	assert.NotNil(t, highMediaType.Example)
}

func TestMediaType_Render_WithItemFields(t *testing.T) {
	yml := `schema:
  type: array
itemSchema:
  type: object
  properties:
    id:
      type: string
itemEncoding:
  id:
    contentType: text/plain`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var lowMediaType lowv3.MediaType
	_ = low.BuildModel(&idxNode, &lowMediaType)
	_ = lowMediaType.Build(context.Background(), nil, idxNode.Content[0], idx)

	highMediaType := NewMediaType(&lowMediaType)

	// Test rendering
	rendered, err := highMediaType.Render()
	assert.NoError(t, err)
	assert.Contains(t, string(rendered), "itemSchema:")
	assert.Contains(t, string(rendered), "itemEncoding:")
}

func TestMediaType_GoLow_WithItemFields(t *testing.T) {
	yml := `itemSchema:
  type: string
itemEncoding:
  field:
    contentType: application/json`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var lowMediaType lowv3.MediaType
	_ = low.BuildModel(&idxNode, &lowMediaType)
	_ = lowMediaType.Build(context.Background(), nil, idxNode.Content[0], idx)

	highMediaType := NewMediaType(&lowMediaType)

	// Test GoLow
	lowResult := highMediaType.GoLow()
	assert.NotNil(t, lowResult)
	assert.NotNil(t, lowResult.ItemSchema.Value)
	assert.NotNil(t, lowResult.ItemEncoding.Value)
}
