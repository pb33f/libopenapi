// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

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
	_ = n.Build(nil, idxNode.Content[0], idx)

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
	_ = n.Build(nil, idxNode.Content[0], idx)

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
	_ = n.Build(nil, idxNode.Content[0], idx)

	r := NewResponses(&n)

	rend, _ := r.RenderInline()
	assert.Equal(t, yml, strings.TrimSpace(string(rend)))

}
