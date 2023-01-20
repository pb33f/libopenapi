// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"testing"

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
	_ = n.Build(idxNode.Content[0], idx)

	r := NewOperation(&n)

	assert.Equal(t, "https://pb33f.io", r.ExternalDocs.URL)
	assert.Equal(t, 1, r.GoLow().ExternalDocs.KeyNode.Line)
	assert.Contains(t, r.Callbacks, "testCallback")
	assert.Contains(t, r.Callbacks["testCallback"].Expression, "{$request.body#/callbackUrl}")
	assert.Equal(t, 3, r.GoLow().Callbacks.KeyNode.Line)
}
