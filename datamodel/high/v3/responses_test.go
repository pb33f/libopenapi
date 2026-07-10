// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestResponses_MarshalYAMLInlineWithContext_PropagatesDefaultError(t *testing.T) {
	proxy := base.CreateSchemaProxyRefWithSchema("#/Cycle", &base.Schema{Description: "cycle"})
	ctx := base.NewInlineRenderContext()
	require.False(t, ctx.StartRendering("#/Cycle"))
	content := orderedmap.New[string, *MediaType]()
	content.Set("application/json", &MediaType{Schema: proxy})
	responses := &Responses{Codes: orderedmap.New[string, *Response](), Default: &Response{Description: "default", Content: content}}

	_, err := responses.MarshalYAMLInlineWithContext(ctx)
	require.ErrorContains(t, err, "circular reference")
}

// this test exists because the sample contract doesn't contain a
// responses with *everything* populated, I had already written a ton of tests
// with hard coded line and column numbers in them, changing the spec above the bottom will
// create pointless test changes. So here is a standalone test. you know... for science.

func TestNewResponses(t *testing.T) {
	yml := `default:
  description: default response`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n v3.Responses
	_ = low.BuildModel(&idxNode, &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	r := NewResponses(&n)

	assert.Equal(t, "default response", r.Default.Description)
	assert.Equal(t, 1, r.GoLow().Default.KeyNode.Line)
}

func TestResponses_MarshalYAML(t *testing.T) {
	yml := `"201":
    description: this is a response
    content:
        something/thing:
            example: cake
"404":
    description: this is a 404
    content:
        something/thing:
            example: why do you need an example?
"200":
    description: OK! not bad.`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	var n v3.Responses
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	r := NewResponses(&n)

	rend, _ := r.Render()
	assert.Equal(t, yml, strings.TrimSpace(string(rend)))
}

func TestResponses_MarshalYAMLInline(t *testing.T) {
	yml := `"201":
    description: this is a response
    content:
        something/thing:
            example: cake
"404":
    description: this is a 404
    content:
        something/thing:
            example: why do you need an example?
"200":
    description: OK! not bad.`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	var n v3.Responses
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	r := NewResponses(&n)

	rend, _ := r.RenderInline()
	assert.Equal(t, yml, strings.TrimSpace(string(rend)))
}
