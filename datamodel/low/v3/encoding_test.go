// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
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

func TestEncoding_Build_Success(t *testing.T) {
	yml := `contentType: hot/cakes
headers: 
  ohMyStars:
    description: this is a header
    required: true
    allowEmptyValue: true
allowReserved: true    
explode: true`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Encoding
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "hot/cakes", n.ContentType.Value)
	assert.Equal(t, true, n.AllowReserved.Value)
	assert.Equal(t, true, n.Explode.Value)

	header := n.FindHeader("ohMyStars")
	assert.NotNil(t, header.Value)
	assert.Equal(t, "this is a header", header.Value.Description.Value)
	assert.Equal(t, true, header.Value.Required.Value)
	assert.Equal(t, true, header.Value.AllowEmptyValue.Value)
	assert.NotNil(t, n.GetRootNode())
	assert.Nil(t, n.GetKeyNode())
	assert.NotNil(t, n.GetIndex())
	assert.NotNil(t, n.GetContext())
}

func TestEncoding_Build_Error(t *testing.T) {
	yml := `contentType: hot/cakes
headers: 
  $ref: #/borked`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Encoding
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestEncoding_Hash(t *testing.T) {
	yml := `contentType: application/waffle
headers:
  heady:
    description: a header
style: post modern
explode: true
allowReserved: true`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Encoding
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	yml2 := `explode: true
contentType: application/waffle
allowReserved: true
headers:
  heady:
    description: a header
style: post modern
`

	var idxNode2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml2), &idxNode2)
	idx2 := index.NewSpecIndex(&idxNode2)

	var n2 Encoding
	_ = low.BuildModel(idxNode2.Content[0], &n2)
	_ = n2.Build(context.Background(), nil, idxNode2.Content[0], idx2)

	// hash
	assert.Equal(t, n.Hash(), n2.Hash())
}

func TestEncoding_Build_ScalarRoot(t *testing.T) {
	var scalar yaml.Node
	_ = yaml.Unmarshal([]byte("hello"), &scalar)

	var n Encoding
	err := low.BuildModel(scalar.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, scalar.Content[0], nil)
	assert.NoError(t, err)

	nodes := n.GetNodes()
	assert.Len(t, nodes[scalar.Content[0].Line], 1)
	assert.Equal(t, "hello", nodes[scalar.Content[0].Line][0].Value)
}

func TestEncoding_Build_Nested(t *testing.T) {
	yml := `contentType: multipart/mixed
encoding:
  meta:
    contentType: application/json
prefixEncoding:
  - contentType: application/xml
  - contentType: image/png
itemEncoding:
  contentType: text/plain`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Encoding
	_ = low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, n.Build(context.Background(), nil, idxNode.Content[0], idx))

	assert.Equal(t, "application/json", n.FindEncoding("meta").Value.ContentType.Value)
	assert.Nil(t, n.FindEncoding("missing"))
	assert.Len(t, n.PrefixEncoding.Value, 2)
	assert.Equal(t, "image/png", n.PrefixEncoding.Value[1].Value.ContentType.Value)
	assert.Equal(t, "text/plain", n.ItemEncoding.Value.ContentType.Value)
}

func TestEncoding_Build_Error_Nested(t *testing.T) {
	for name, yml := range map[string]string{
		"encoding":       "encoding:\n  part:\n    $ref: '#/bork'",
		"prefixEncoding": "prefixEncoding:\n  - $ref: '#/bork'",
		"itemEncoding":   "itemEncoding:\n  $ref: '#/bork'",
	} {
		t.Run(name, func(t *testing.T) {
			var idxNode yaml.Node
			_ = yaml.Unmarshal([]byte(yml), &idxNode)
			idx := index.NewSpecIndex(&idxNode)

			var n Encoding
			_ = low.BuildModel(idxNode.Content[0], &n)
			assert.Error(t, n.Build(context.Background(), nil, idxNode.Content[0], idx))
		})
	}
}
