// Copyright 2026 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package base

import (
	"testing"

	"github.com/pb33f/testify/require"
)

func TestBooleanSchemaValueAndHash(t *testing.T) {
	schemas := buildHashTestSchemas(t, "true", "false", "{}")
	require.False(t, schemas[0].BooleanSchema.IsEmpty())
	require.True(t, schemas[0].BooleanSchema.Value)
	require.False(t, schemas[1].BooleanSchema.IsEmpty())
	require.False(t, schemas[1].BooleanSchema.Value)
	require.True(t, schemas[2].BooleanSchema.IsEmpty())
	assertSchemaHashesDiffer(t, "true", "false", "{}")
	assertSchemaHashesMatch(t, "false", "false")
}

func TestBooleanSchemaObjectKeywordDoesNotSetRootShape(t *testing.T) {
	for _, keyword := range []string{"booleanSchema", "BooleanSchema", "booleanschema"} {
		for _, value := range []string{"true", "false"} {
			t.Run(keyword+"_"+value, func(t *testing.T) {
				source := "type: string\n" + keyword + ": " + value
				schemas := buildHashTestSchemas(t, source, "type: string")
				require.True(t, schemas[0].BooleanSchema.IsEmpty())
				require.Equal(t, schemas[1].Hash(), schemas[0].Hash())
			})
		}
	}
}
