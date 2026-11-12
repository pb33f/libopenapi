// Copyright 2025 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/low"
	lowv3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/assert"
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
  contentType: application/json
  allowReserved: true
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

	// itemEncoding is a single Encoding Object (OpenAPI 3.2)
	assert.NotNil(t, highMediaType.ItemEncoding)
	assert.Equal(t, "application/json", highMediaType.ItemEncoding.ContentType)
	assert.True(t, highMediaType.ItemEncoding.AllowReserved)
	assert.Equal(t, 1, highMediaType.ItemEncoding.Headers.Len())
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

	// Check regular encoding
	regularEnc := highMediaType.Encoding.GetOrZero("regularField")
	assert.NotNil(t, regularEnc)
	assert.Equal(t, "text/plain", regularEnc.ContentType)
	assert.Equal(t, "form", regularEnc.Style)

	// Check item encoding
	itemEnc := highMediaType.ItemEncoding
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

// TestMediaType_OAS32Encoding_RoundTrip checks that prefixEncoding, a single-object itemEncoding and the
// nested Encoding Object fields build into the model and render back unchanged.
func TestMediaType_OAS32Encoding_RoundTrip(t *testing.T) {
	for name, yml := range map[string]string{
		"prefix and item": `prefixEncoding:
    - contentType: application/json
    - contentType: image/png
      headers:
        X-Part:
            schema:
                type: string
itemEncoding:
    contentType: text/plain
`,
		"nested": `encoding:
    batch:
        contentType: multipart/mixed
        prefixEncoding:
            - contentType: application/json
        itemEncoding:
            contentType: image/png
    form:
        contentType: multipart/form-data
        encoding:
            file:
                contentType: image/jpeg
`,
	} {
		t.Run(name, func(t *testing.T) {
			var idxNode yaml.Node
			_ = yaml.Unmarshal([]byte(yml), &idxNode)
			idx := index.NewSpecIndex(&idxNode)

			var lowMediaType lowv3.MediaType
			_ = low.BuildModel(&idxNode, &lowMediaType)
			assert.NoError(t, lowMediaType.Build(context.Background(), nil, idxNode.Content[0], idx))

			rendered, err := NewMediaType(&lowMediaType).Render()
			assert.NoError(t, err)
			assert.Equal(t, yml, string(rendered))
		})
	}
}

func TestMediaType_OAS32Encoding_Model(t *testing.T) {
	yml := `encoding:
  batch:
    contentType: multipart/mixed
    prefixEncoding:
      - contentType: application/json
      - contentType: image/png
    itemEncoding:
      contentType: text/plain
    encoding:
      inner:
        style: form
prefixEncoding:
  - contentType: application/xml
itemEncoding:
  contentType: image/gif`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var lowMediaType lowv3.MediaType
	_ = low.BuildModel(&idxNode, &lowMediaType)
	assert.NoError(t, lowMediaType.Build(context.Background(), nil, idxNode.Content[0], idx))
	mt := NewMediaType(&lowMediaType)

	assert.Len(t, mt.PrefixEncoding, 1)
	assert.Equal(t, "application/xml", mt.PrefixEncoding[0].ContentType)
	assert.Equal(t, "image/gif", mt.ItemEncoding.ContentType)

	batch := mt.Encoding.GetOrZero("batch")
	assert.Len(t, batch.PrefixEncoding, 2)
	assert.Equal(t, "image/png", batch.PrefixEncoding[1].ContentType)
	assert.Equal(t, "text/plain", batch.ItemEncoding.ContentType)
	assert.Equal(t, "form", batch.Encoding.GetOrZero("inner").Style)
}
