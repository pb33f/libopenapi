// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"testing"

	"github.com/pb33f/testify/require"
)

func TestReferenceInNamedOperandKeepsItsAddress(t *testing.T) {
	const named = `{"$id":"urn:operand","$ref":"urn:field"}`
	const number = `{"type":"number"}`
	const definitions = `"$defs":{"field":{"$id":"urn:field"}}`
	before := loadTest(t, `{`+definitions+`,"anyOf":[`+named+","+number+`]}`)
	after := loadTest(t, `{`+definitions+`,"anyOf":[`+number+","+named+`]}`)
	diff, err := Compare(before, after, nil)
	require.NoError(t, err)
	require.Nil(t, diff)
}

func TestReferenceInInlineOperandGainsAddress(t *testing.T) {
	const input = `{"$defs":{"field":{"$id":"urn:field"}},"anyOf":[`
	original := loadTest(t, input+`{"$ref":"urn:field"}]}`)
	modified := loadTest(t, input+`{"$id":"urn:operand","$ref":"urn:field"}]}`)
	diff, err := Compare(original, modified, nil)
	require.NoError(t, err)
	require.Len(t, diff.GetAllChanges(), 1)
	change := diff.GetAllChanges()[0]
	require.Equal(t, "$id", change.Property)
	require.Equal(t, "urn:operand", change.SchemaResource)
	require.Equal(t, "urn:operand", change.New)
	require.False(t, change.Breaking)
}
