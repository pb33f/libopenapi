// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	lowV2 "github.com/pb33f/libopenapi/datamodel/low/v2"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/assert"
	"go.yaml.in/yaml/v4"
)

func buildHeader(t *testing.T, yml string) *Header {
	t.Helper()
	var idxNode yaml.Node
	assert.NoError(t, yaml.Unmarshal([]byte(yml), &idxNode))
	idx := index.NewSpecIndex(&idxNode)

	var n lowV2.Header
	assert.NoError(t, low.BuildModel(idxNode.Content[0], &n))
	assert.NoError(t, n.Build(context.Background(), nil, idxNode.Content[0], idx))
	return NewHeader(&n)
}

// https://github.com/pb33f/libopenapi/issues/630
func TestNewHeader_AllFields(t *testing.T) {
	yml := `type: integer
format: int64
description: a header
collectionFormat: csv
default: 5
maximum: 10
exclusiveMaximum: true
minimum: 1
exclusiveMinimum: true
maxLength: 20
minLength: 2
pattern: '^\d+$'
maxItems: 4
minItems: 1
uniqueItems: true
enum: [1, 5, 10]
multipleOf: 1
x-header: yes
items:
  type: string`

	h := buildHeader(t, yml)

	assert.Equal(t, "integer", h.Type)
	assert.Equal(t, "int64", h.Format)
	assert.Equal(t, "a header", h.Description)
	assert.Equal(t, "csv", h.CollectionFormat)
	assert.Equal(t, "5", h.Default.(*yaml.Node).Value)
	assert.Equal(t, 10, h.Maximum)
	assert.True(t, h.ExclusiveMaximum)
	assert.Equal(t, 1, h.Minimum)
	assert.True(t, h.ExclusiveMinimum)
	assert.Equal(t, 20, h.MaxLength)
	assert.Equal(t, 2, h.MinLength)
	assert.Equal(t, `^\d+$`, h.Pattern)
	assert.Equal(t, 4, h.MaxItems)
	assert.Equal(t, 1, h.MinItems)
	assert.True(t, h.UniqueItems)
	assert.Len(t, h.Enum, 3)
	assert.Equal(t, 1, h.MultipleOf)
	assert.Equal(t, "string", h.Items.Type)
	assert.Equal(t, 1, h.Extensions.Len())
	assert.Equal(t, 20, h.GoLow().MaxLength.Value)
}

func TestNewHeader_ExplicitFalseFlags(t *testing.T) {
	yml := `type: string
exclusiveMaximum: false
exclusiveMinimum: false
uniqueItems: false`

	h := buildHeader(t, yml)

	assert.Equal(t, "string", h.Type)
	assert.Empty(t, h.Format)
	assert.False(t, h.ExclusiveMaximum)
	assert.False(t, h.ExclusiveMinimum)
	assert.False(t, h.UniqueItems)
}
