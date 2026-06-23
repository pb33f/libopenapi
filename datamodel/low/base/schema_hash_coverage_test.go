// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestWriteSchemaBoolMap(t *testing.T) {
	var sb strings.Builder

	writeSchemaBoolMap(&sb, nil)
	assert.Equal(t, "", sb.String())

	empty := orderedmap.New[low.KeyReference[string], low.ValueReference[bool]]()
	writeSchemaBoolMap(&sb, empty)
	assert.Equal(t, "", sb.String())

	values := orderedmap.New[low.KeyReference[string], low.ValueReference[bool]]()
	values.Set(low.KeyReference[string]{Value: "zeta"}, low.ValueReference[bool]{Value: true})
	values.Set(low.KeyReference[string]{Value: "alpha"}, low.ValueReference[bool]{Value: false})

	writeSchemaBoolMap(&sb, values)
	assert.Equal(t, "alpha:false|zeta:true|", sb.String())
}

func TestWriteSchemaDependentRequired(t *testing.T) {
	var sb strings.Builder

	writeSchemaDependentRequired(&sb, nil)
	assert.Equal(t, "", sb.String())

	empty := orderedmap.New[low.KeyReference[string], low.ValueReference[[]string]]()
	writeSchemaDependentRequired(&sb, empty)
	assert.Equal(t, "", sb.String())

	values := orderedmap.New[low.KeyReference[string], low.ValueReference[[]string]]()
	values.Set(low.KeyReference[string]{Value: "omega"}, low.ValueReference[[]string]{Value: []string{"z", "a"}})
	values.Set(low.KeyReference[string]{Value: "alpha"}, low.ValueReference[[]string]{Value: []string{"x"}})

	writeSchemaDependentRequired(&sb, values)
	assert.Equal(t, "alpha:x|omega:z,a|", sb.String())
}

func TestWriteSortedSchemaStrings(t *testing.T) {
	var sb strings.Builder

	writeSortedSchemaStrings(&sb, nil, false)
	assert.Equal(t, "", sb.String())

	writeSortedSchemaStrings(&sb, []string{"zeta", "alpha"}, false)
	assert.Equal(t, "alphazeta|", sb.String())
}

func TestSchemaHashIncludesDefs(t *testing.T) {
	build := func(source string) *Schema {
		t.Helper()
		var root yaml.Node
		require.NoError(t, yaml.Unmarshal([]byte(source), &root))

		var schema Schema
		require.NoError(t, schema.Build(context.Background(), root.Content[0], nil))
		return &schema
	}

	a := build(`type: object
$defs:
  shared:
    type: string`)
	b := build(`type: object
$defs:
  shared:
    type: integer`)

	assert.NotEqual(t, a.Hash(), b.Hash())
}
