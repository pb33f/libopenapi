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
	_ = n.Build(idxNode.Content[0], idx)

	r := NewResponses(&n)

	assert.Equal(t, "default response", r.Default.Description)
	assert.Equal(t, 1, r.GoLow().Default.KeyNode.Line)

}
