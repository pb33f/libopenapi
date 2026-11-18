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
	"github.com/pb33f/libopenapi/orderedmap"
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
		"item encoding with nested encoding": `itemEncoding:
    contentType: multipart/form-data
    encoding:
        photo:
            contentType: image/png
        x-file:
            contentType: application/pdf
`,
		"encoding keys inside example": `example:
    encoding:
        photo:
            contentType: image/png
    itemEncoding:
        contentType: text/plain
`,
		"nested part headers": `encoding:
    batch:
        contentType: multipart/mixed
        itemEncoding:
            headers:
                X-Part:
                    schema:
                        type: string
`,
		"item encoding extension": `itemEncoding:
    contentType: image/png
    x-vendor: abc
`,
		"extensions on encoding map levels": `encoding:
    batch:
        contentType: multipart/form-data
        x-batch: true
        encoding:
            x-file:
                contentType: application/pdf
                x-inner: 1
`,
		"extensions on positional encoding levels": `prefixEncoding:
    - contentType: application/xml
      x-part: top
itemEncoding:
    contentType: multipart/mixed
    x-batch: true
    prefixEncoding:
        - contentType: application/json
          x-part: first
    itemEncoding:
        contentType: image/png
        x-item: abc
`,
		"empty prefix encoding": `schema:
    type: array
prefixEncoding: []
`,
		"empty prefix encoding on an encoding map entry": `encoding:
    batch:
        contentType: multipart/mixed
        prefixEncoding: []
`,
		"empty prefix encoding on nested parts": `prefixEncoding:
    - contentType: multipart/mixed
      prefixEncoding: []
itemEncoding:
    contentType: multipart/mixed
    prefixEncoding: []
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
	mt := buildTestMediaType(t, []byte(`prefixEncoding:
  - contentType: application/xml
itemEncoding:
  contentType: image/gif`))

	assert.Len(t, mt.PrefixEncoding, 1)
	assert.Equal(t, "application/xml", mt.PrefixEncoding[0].ContentType)
	assert.Equal(t, "image/gif", mt.ItemEncoding.ContentType)

	mt = buildTestMediaType(t, []byte(`encoding:
  batch:
    contentType: multipart/mixed
    prefixEncoding:
      - contentType: application/json
      - contentType: image/png
    itemEncoding:
      contentType: text/plain
  form:
    contentType: multipart/form-data
    encoding:
      inner:
        style: form`))

	batch := mt.Encoding.GetOrZero("batch")
	assert.Len(t, batch.PrefixEncoding, 2)
	assert.Equal(t, "image/png", batch.PrefixEncoding[1].ContentType)
	assert.Equal(t, "text/plain", batch.ItemEncoding.ContentType)
	assert.Equal(t, "form", mt.Encoding.GetOrZero("form").Encoding.GetOrZero("inner").Style)
}

func TestMediaType_OAS32Encoding_Extensions(t *testing.T) {
	yml := `itemEncoding:
  contentType: image/png
  x-vendor: abc
prefixEncoding:
  - contentType: application/json
    x-part: first`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var lowMediaType lowv3.MediaType
	_ = low.BuildModel(&idxNode, &lowMediaType)
	assert.NoError(t, lowMediaType.Build(context.Background(), nil, idxNode.Content[0], idx))
	mt := NewMediaType(&lowMediaType)

	assert.Equal(t, "abc", mt.ItemEncoding.Extensions.GetOrZero("x-vendor").Value)
	assert.Equal(t, "first", mt.PrefixEncoding[0].Extensions.GetOrZero("x-part").Value)
	assert.Equal(t, 0, mt.Extensions.Len())
}

func TestMediaType_OAS32Encoding_EmptyPrefixEncodingBuiltByCaller(t *testing.T) {
	// an empty non-nil slice renders as [], a nil slice is omitted.
	mt := &MediaType{
		PrefixEncoding: []*Encoding{},
		ItemEncoding:   &Encoding{ContentType: "multipart/mixed", PrefixEncoding: []*Encoding{}},
	}
	rendered, err := mt.Render()
	assert.NoError(t, err)
	assert.Equal(t, "prefixEncoding: []\nitemEncoding:\n    contentType: multipart/mixed\n    prefixEncoding: []\n", string(rendered))

	rendered, err = (&MediaType{ItemEncoding: &Encoding{ContentType: "multipart/mixed"}}).Render()
	assert.NoError(t, err)
	assert.Equal(t, "itemEncoding:\n    contentType: multipart/mixed\n", string(rendered))
}

// buildTestMediaType builds a low MediaType from yml and wraps it in a high MediaType.
func buildTestMediaType(t *testing.T, yml []byte) *MediaType {
	t.Helper()
	var idxNode yaml.Node
	assert.NoError(t, yaml.Unmarshal(yml, &idxNode))
	var lowMediaType lowv3.MediaType
	_ = low.BuildModel(&idxNode, &lowMediaType)
	assert.NoError(t, lowMediaType.Build(context.Background(), nil, idxNode.Content[0], index.NewSpecIndex(&idxNode)))
	return NewMediaType(&lowMediaType)
}

// TestMediaType_OAS32Encoding_ExplicitFalseRoundTrip checks that an explicit explode: false (whose default depends on
// style) and allowReserved: false survive a render and reload, while an omitted explode stays omitted.
func TestMediaType_OAS32Encoding_ExplicitFalseRoundTrip(t *testing.T) {
	yml := `encoding:
    photo:
        style: form
        explode: false
        allowReserved: false
    note:
        style: form
    batch:
        contentType: multipart/form-data
        encoding:
            photo:
                contentType: application/json
                style: form
                explode: false
                allowReserved: false
            note:
                style: form
`
	rendered, err := buildTestMediaType(t, []byte(yml)).Render()
	assert.NoError(t, err)
	assert.Equal(t, yml, string(rendered))

	reloaded := buildTestMediaType(t, rendered)
	for name, enc := range map[string]*orderedmap.Map[string, *Encoding]{
		"encoding":                reloaded.Encoding,
		"encoding.batch.encoding": reloaded.Encoding.GetOrZero("batch").Encoding,
	} {
		photo := enc.GetOrZero("photo")
		if assert.NotNil(t, photo.Explode, name) {
			assert.False(t, *photo.Explode, name)
		}
		assert.False(t, photo.GoLow().AllowReserved.IsEmpty(), name)
		assert.False(t, photo.AllowReserved, name)

		note := enc.GetOrZero("note")
		assert.Nil(t, note.Explode, name)
		assert.True(t, note.GoLow().AllowReserved.IsEmpty(), name)
	}
}

// TestMediaType_OAS32Encoding_AliasedKeys checks that encoding and headers keys written as YAML aliases build and
// render under the anchored name, not the anchor's label.
func TestMediaType_OAS32Encoding_AliasedKeys(t *testing.T) {
	mt := buildTestMediaType(t, []byte(`schema:
    type: object
    properties:
        &field photo:
            type: string
        &hdr X-Rate-Limit:
            type: integer
        batch:
            type: object
encoding:
    *field :
        contentType: image/png
        headers:
            *hdr :
                schema:
                    type: integer
    batch:
        contentType: multipart/form-data
        encoding:
            *field :
                contentType: image/jpeg
`))

	photo := mt.Encoding.GetOrZero("photo")
	if assert.NotNil(t, photo) {
		assert.Equal(t, "image/png", photo.ContentType)
		assert.NotNil(t, photo.Headers.GetOrZero("X-Rate-Limit"))
	}
	assert.Equal(t, "image/jpeg", mt.Encoding.GetOrZero("batch").Encoding.GetOrZero("photo").ContentType)

	rendered, err := mt.Render()
	assert.NoError(t, err)
	assert.Equal(t, `schema:
    type: object
    properties:
        photo:
            type: string
        X-Rate-Limit:
            type: integer
        batch:
            type: object
encoding:
    photo:
        contentType: image/png
        headers:
            X-Rate-Limit:
                schema:
                    type: integer
    batch:
        contentType: multipart/form-data
        encoding:
            photo:
                contentType: image/jpeg
`, string(rendered))
}
