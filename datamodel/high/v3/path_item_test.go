// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// this test exists because the sample contract doesn't contain a
// response with *everything* populated, I had already written a ton of tests
// with hard coded line and column numbers in them, changing the spec above the bottom will
// create pointless test changes. So here is a standalone test. you know... for science.
func TestPathItem(t *testing.T) {
	yml := `servers:
  - description: so many options for things in places.`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n v3.PathItem
	_ = low.BuildModel(&idxNode, &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	r := NewPathItem(&n)

	assert.Len(t, r.Servers, 1)
	assert.Equal(t, "so many options for things in places.", r.Servers[0].Description)
	assert.Equal(t, 1, r.GoLow().Servers.KeyNode.Line)
}

func TestPathItem_GetOperations(t *testing.T) {
	yml := `get:
  description: get
put:
  description: put
post:
  description: post
patch:
  description: patch
delete:
  description: delete
head:
  description: head
options:
  description: options
trace:
  description: trace
`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n v3.PathItem
	_ = low.BuildModel(&idxNode, &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	r := NewPathItem(&n)

	assert.Equal(t, 8, r.GetOperations().Len())

	// test that the operations are in the correct order
	expectedOrder := []string{"get", "put", "post", "patch", "delete", "head", "options", "trace"}

	i := 0
	for pair := orderedmap.First(r.GetOperations()); pair != nil; pair = pair.Next() {
		assert.Equal(t, expectedOrder[i], pair.Value().Description)
		i++
	}
}

func TestPathItem_MarshalYAML(t *testing.T) {
	pi := &PathItem{
		Description: "a path item",
		Summary:     "It's a test, don't worry about it, Jim",
		Servers: []*Server{
			{
				Description: "a server",
			},
		},
		Parameters: []*Parameter{
			{
				Name: "I am a query parameter",
				In:   "query",
			},
		},
		Get: &Operation{
			Description: "a get operation",
		},
		Post: &Operation{
			Description: "a post operation",
		},
	}

	rend, _ := pi.Render()

	desired := `description: a path item
summary: It's a test, don't worry about it, Jim
get:
    description: a get operation
post:
    description: a post operation
servers:
    - description: a server
parameters:
    - name: I am a query parameter
      in: query`

	assert.Equal(t, desired, strings.TrimSpace(string(rend)))
}

func TestPathItem_MarshalYAMLInline(t *testing.T) {
	pi := &PathItem{
		Description: "a path item",
		Summary:     "It's a test, don't worry about it, Jim",
		Servers: []*Server{
			{
				Description: "a server",
			},
		},
		Parameters: []*Parameter{
			{
				Name: "I am a query parameter",
				In:   "query",
			},
		},
		Get: &Operation{
			Description: "a get operation",
		},
		Post: &Operation{
			Description: "a post operation",
		},
	}

	rend, _ := pi.RenderInline()

	desired := `description: a path item
summary: It's a test, don't worry about it, Jim
get:
    description: a get operation
post:
    description: a post operation
servers:
    - description: a server
parameters:
    - name: I am a query parameter
      in: query`

	assert.Equal(t, desired, strings.TrimSpace(string(rend)))
}
