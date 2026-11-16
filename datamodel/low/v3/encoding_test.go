// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"fmt"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
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
  ohMyStars:
    $ref: '#/borked'`

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

// TestEncoding_Build_DirectKeysOnly checks that headers and the nested encoding fields are read only from the
// Encoding's own keys, never from the keys of a nested part.
func TestEncoding_Build_DirectKeysOnly(t *testing.T) {
	yml := `contentType: multipart/mixed
itemEncoding:
  contentType: multipart/form-data
  headers:
    X-Part:
      description: a part header
  encoding:
    photo:
      contentType: image/png
  prefixEncoding:
    - contentType: text/plain`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Encoding
	_ = low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, n.Build(context.Background(), nil, idxNode.Content[0], idx))

	assert.True(t, n.Headers.IsEmpty())
	assert.True(t, n.Encoding.IsEmpty())
	assert.True(t, n.PrefixEncoding.IsEmpty())

	item := n.ItemEncoding.Value
	assert.NotNil(t, item.FindHeader("X-Part"))
	assert.Equal(t, "image/png", item.FindEncoding("photo").Value.ContentType.Value)
	assert.Len(t, item.PrefixEncoding.Value, 1)
	assert.Nil(t, item.ItemEncoding.Value)
}

// TestEncoding_Build_ExtensionPrefixedNames checks that encoding property names and header names starting with
// x- are kept, because they are user-defined names rather than specification extensions.
func TestEncoding_Build_ExtensionPrefixedNames(t *testing.T) {
	yml := `headers:
  x-trace:
    description: trace id
encoding:
  photo:
    contentType: image/png
  x-file:
    contentType: application/pdf`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Encoding
	_ = low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, n.Build(context.Background(), nil, idxNode.Content[0], idx))

	assert.Equal(t, "trace id", n.FindHeader("x-trace").Value.Description.Value)
	assert.Equal(t, "image/png", n.FindEncoding("photo").Value.ContentType.Value)
	assert.Equal(t, "application/pdf", n.FindEncoding("x-file").Value.ContentType.Value)
}

func TestEncoding_Build_ItemEncodingReference(t *testing.T) {
	yml := `itemEncoding:
  $ref: '#/part'
part:
  contentType: image/png`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Encoding
	_ = low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, n.Build(context.Background(), nil, idxNode.Content[0], idx))

	assert.Equal(t, "image/png", n.ItemEncoding.Value.ContentType.Value)
	assert.True(t, n.ItemEncoding.IsReference())
	assert.Equal(t, "#/part", n.ItemEncoding.GetReference())
	assert.True(t, n.ItemEncoding.Value.IsReference())
}

func buildTestEncoding(t *testing.T, yml string) *Encoding {
	t.Helper()
	var idxNode yaml.Node
	assert.NoError(t, yaml.Unmarshal([]byte(yml), &idxNode))
	var n Encoding
	_ = low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, n.Build(context.Background(), nil, idxNode.Content[0], index.NewSpecIndex(&idxNode)))
	return &n
}

func TestEncoding_Build_Extensions(t *testing.T) {
	n := buildTestEncoding(t, `contentType: multipart/mixed
x-vendor: abc
encoding:
  x-file:
    contentType: application/pdf
    x-inner: 1
prefixEncoding:
  - contentType: application/json
    x-part: first
itemEncoding:
  contentType: image/png
  x-item: abc`)

	assert.Equal(t, 1, orderedmap.Len(n.GetExtensions()))
	assert.Equal(t, "abc", n.FindExtension("x-vendor").Value.Value)
	assert.Equal(t, "1", n.FindEncoding("x-file").Value.FindExtension("x-inner").Value.Value)
	assert.Equal(t, "first", n.PrefixEncoding.Value[0].Value.FindExtension("x-part").Value.Value)
	assert.Equal(t, "abc", n.ItemEncoding.Value.FindExtension("x-item").Value.Value)

	// x- keys under encoding are part names, not extensions of the enclosing Encoding Object.
	assert.Nil(t, n.FindExtension("x-file"))
	assert.Nil(t, n.FindExtension("x-inner"))
}

func TestEncoding_Hash_Extensions(t *testing.T) {
	for name, yml := range map[string]string{
		"top":    "contentType: image/png\nx-vendor: %s",
		"item":   "itemEncoding:\n  contentType: image/png\n  x-vendor: %s",
		"prefix": "prefixEncoding:\n  - contentType: image/png\n    x-vendor: %s",
		"nested": "encoding:\n  photo:\n    contentType: image/png\n    x-vendor: %s",
	} {
		t.Run(name, func(t *testing.T) {
			a := buildTestEncoding(t, fmt.Sprintf(yml, "abc"))
			b := buildTestEncoding(t, fmt.Sprintf(yml, "xyz"))
			assert.NotEqual(t, a.Hash(), b.Hash())
			assert.Equal(t, a.Hash(), buildTestEncoding(t, fmt.Sprintf(yml, "abc")).Hash())
		})
	}
}

func TestEncoding_Hash_ExplodePresence(t *testing.T) {
	absent := buildTestEncoding(t, "style: form").Hash()
	explicitFalse := buildTestEncoding(t, "style: form\nexplode: false").Hash()
	explicitTrue := buildTestEncoding(t, "style: form\nexplode: true").Hash()

	assert.NotEqual(t, absent, explicitFalse)
	assert.NotEqual(t, absent, explicitTrue)
	assert.NotEqual(t, explicitFalse, explicitTrue)

	nested := buildTestEncoding(t, "itemEncoding:\n  encoding:\n    photo:\n      style: form").Hash()
	nestedFalse := buildTestEncoding(t, "itemEncoding:\n  encoding:\n    photo:\n      style: form\n      explode: false").Hash()
	assert.NotEqual(t, nested, nestedFalse)
}

func TestEncoding_Hash_EmptyPrefixEncodingPresence(t *testing.T) {
	low.ClearHashCache()
	absent := buildTestEncoding(t, "contentType: multipart/mixed").Hash()
	empty := buildTestEncoding(t, "contentType: multipart/mixed\nprefixEncoding: []").Hash()
	emptyAgain := buildTestEncoding(t, "contentType: multipart/mixed\nprefixEncoding: []").Hash()
	assert.NotEqual(t, absent, empty)
	assert.Equal(t, empty, emptyAgain)

	nested := buildTestEncoding(t, "itemEncoding:\n  contentType: multipart/mixed").Hash()
	nestedEmpty := buildTestEncoding(t, "itemEncoding:\n  contentType: multipart/mixed\n  prefixEncoding: []").Hash()
	assert.NotEqual(t, nested, nestedEmpty)

	buildMediaType := func(yml string) uint64 {
		var node yaml.Node
		assert.NoError(t, yaml.Unmarshal([]byte(yml), &node))
		var mt MediaType
		_ = low.BuildModel(node.Content[0], &mt)
		assert.NoError(t, mt.Build(context.Background(), nil, node.Content[0], index.NewSpecIndex(&node)))
		return mt.Hash()
	}
	assert.NotEqual(t, buildMediaType("schema:\n  type: array"),
		buildMediaType("schema:\n  type: array\nprefixEncoding: []"))
}
