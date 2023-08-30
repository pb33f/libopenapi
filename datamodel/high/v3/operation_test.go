// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high/base"

	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// this test exists because the sample contract doesn't contain an
// operation with *everything* populated, I had already written a ton of tests
// with hard coded line and column numbers in them, changing the spec above the bottom will
// create pointless test changes. So here is a standalone test. you know... for science.

func TestOperation(t *testing.T) {
	yml := `externalDocs:
  url: https://pb33f.io
callbacks:
  testCallback:
    '{$request.body#/callbackUrl}':
      post:
        requestBody:
          content:
            application/json:
              schema:
                type: object
        responses:
          '200':
            description: OK`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n v3.Operation
	_ = low.BuildModel(&idxNode, &n)
	_ = n.Build(nil, idxNode.Content[0], idx)

	r := NewOperation(&n)

	assert.Equal(t, "https://pb33f.io", r.ExternalDocs.URL)
	assert.Equal(t, 1, r.GoLow().ExternalDocs.KeyNode.Line)
	assert.Contains(t, r.Callbacks, "testCallback")
	assert.Contains(t, r.Callbacks.GetOrZero("testCallback").Expression, "{$request.body#/callbackUrl}")
	assert.Equal(t, 3, r.GoLow().Callbacks.KeyNode.Line)
}

func TestOperation_MarshalYAML(t *testing.T) {

	op := &Operation{
		Tags:        []string{"test"},
		Summary:     "nice",
		Description: "rice",
		ExternalDocs: &base.ExternalDoc{
			Description: "spice",
		},
		OperationId: "slice",
		Parameters: []*Parameter{
			{
				Name: "mice",
			},
		},
		RequestBody: &RequestBody{
			Description: "dice",
		},
	}

	rend, _ := op.Render()

	desired := `tags:
    - test
summary: nice
description: rice
externalDocs:
    description: spice
operationId: slice
parameters:
    - name: mice
requestBody:
    description: dice`

	assert.Equal(t, desired, strings.TrimSpace(string(rend)))

}

func TestOperation_MarshalYAMLInline(t *testing.T) {

	op := &Operation{
		Tags:        []string{"test"},
		Summary:     "nice",
		Description: "rice",
		ExternalDocs: &base.ExternalDoc{
			Description: "spice",
		},
		OperationId: "slice",
		Parameters: []*Parameter{
			{
				Name: "mice",
			},
		},
		RequestBody: &RequestBody{
			Description: "dice",
		},
	}

	rend, _ := op.RenderInline()

	desired := `tags:
    - test
summary: nice
description: rice
externalDocs:
    description: spice
operationId: slice
parameters:
    - name: mice
requestBody:
    description: dice`

	assert.Equal(t, desired, strings.TrimSpace(string(rend)))

}

func TestOperation_EmptySecurity(t *testing.T) {
	yml := `
security: []`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n v3.Operation
	_ = low.BuildModel(&idxNode, &n)
	_ = n.Build(nil, idxNode.Content[0], idx)

	r := NewOperation(&n)

	assert.NotNil(t, r.Security)
	assert.Len(t, r.Security, 0)

}

func TestOperation_NoSecurity(t *testing.T) {
	yml := `operationId: test`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n v3.Operation
	_ = low.BuildModel(&idxNode, &n)
	_ = n.Build(nil, idxNode.Content[0], idx)

	r := NewOperation(&n)

	assert.Nil(t, r.Security)

}
