// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLoadInputBoundaries(t *testing.T) {
	for _, tc := range []struct {
		input    string
		explicit bool
		want     string
	}{
		{`{"$vocabulary":{"urn:unsupported":true}}`, true, "unsupported required vocabulary"},
		{"{}", false, "--schema"}, {`{"$schema":"https://example.org/app"}`, false, "unsupported"},
		{"null", true, "object or boolean"}, {"type: object\ntype: string", true, "already defined"},
		{`{"openapi":"3.1.0","$schema":"https://json-schema.org/draft/2020-12/schema"}`, true, "conflicting"},
		{`{"$defs":{"a":{"$id":"urn:money"},"b":{"$id":"urn:money"}}}`, true, "duplicate"},
		{`{"$dynamicRef":"#scope"}`, true, "dynamic scope"}, {`{"$ref":"#/missing"}`, true, "no schema"},
		{"true\n---\nfalse", true, "one schema"},
	} {
		_, err := Load([]byte(tc.input), "schema.yaml", nil, tc.explicit)
		require.ErrorContains(t, err, tc.want)
	}
	d, err := Load([]byte(`{"$schema":"https://json-schema.org/draft/2020-12/schema","examples":[{"$ref":"missing","$id":"urn:x"},{"$id":"urn:x"}]}`), "schema.json", nil, false)
	require.NoError(t, err)
	d.Release()
}
