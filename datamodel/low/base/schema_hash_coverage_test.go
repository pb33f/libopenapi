// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
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
