// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/3.0"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

// this test exists because the sample contract doesn't contain a
// response with *everything* populated, I had already written a ton of tests
// with hard coded line and column numbers in them, changing the spec above the bottom will
// create pointless test changes. So here is a standalone test. you know... for science.

func TestNewResponse(t *testing.T) {

	yml := `description: this is a response
headers:
  someHeader:
    description: a header
content:
  something/thing:
    description: a thing
x-pizza-man: pizza!
links:
  someLink:
    description: a link!        `

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n v3.Response
	_ = low.BuildModel(&idxNode, &n)
	_ = n.Build(idxNode.Content[0], idx)

	r := NewResponse(&n)

	assert.Len(t, r.Headers, 1)
	assert.Len(t, r.Content, 1)
	assert.Equal(t, "pizza!", r.Extensions["x-pizza-man"])
	assert.Len(t, r.Links, 1)
	assert.Equal(t, 1, r.GoLow().Description.KeyNode.Line)

}
