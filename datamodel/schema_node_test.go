// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package datamodel

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

func TestDecodeSchemaNodeReadsOneDocument(t *testing.T) {
	for _, tc := range []struct{ name, source, message string }{
		{"empty", "", "invalid schema document"},
		{"malformed YAML", "properties: [\n", "invalid schema document"},
		{"second document", "type: string\n---\nfalse\n", "expected one schema document"},
		{"malformed second document", "type: string\n---\nenum: [1\n", "yaml:"},
		{"malformed JSON string", "{\"title\":\"\xff\"}", "invalid JSON schema document"},
		{"duplicate escaped JSON key", "{\n \"properties\": {\n  \"name\": true,\n  \"na\\u006de\": false}}", `line 4: mapping key "name" already defined at line 3`},
		{"duplicate inside YAML example", "examples:\n  - name: first\n    name: second\n", `line 3: mapping key "name" already defined at line 2`},
		{"numeric YAML key", "properties:\n  42: false\n", "line 2: JSON object keys must be strings"},
		{"complex YAML key", "properties:\n  ? [a,b]\n  : false\n", "line 2: JSON object keys must be strings"},
		{"YAML timestamp", "const: 2026-11-25\n", "line 1: unsupported JSON scalar !!timestamp"},
		{"YAML alias", "$defs: &shape {}\nproperties:\n  x: *shape\n", "line 3: expected JSON-compatible YAML"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := DecodeSchemaNode([]byte(tc.source))
			require.Nil(t, doc)
			require.ErrorContains(t, err, tc.message)
			if tc.source == "" {
				require.ErrorIs(t, err, io.EOF)
			}
		})
	}
}

func TestDecodeSchemaNodePreservesYAMLNumberTypes(t *testing.T) {
	integer := strings.Repeat("9", 400)
	doc, err := DecodeSchemaNode([]byte("examples:\n  - " + integer + "\n  - -1.5e5000\n  - '1e5000'\n  - 1GB\n...\n"))
	require.NoError(t, err)
	require.Equal(t, yaml.DocumentNode, doc.Kind)
	require.Len(t, doc.Content, 1)
	values := doc.Content[0].Content[1]
	require.Equal(t, yaml.SequenceNode, values.Kind)
	require.Len(t, values.Content, 4)
	for i, want := range []struct{ value, tag string }{
		{integer, "!!int"}, {"-1.5e5000", "!!float"}, {"1e5000", "!!str"}, {"1GB", "!!str"},
	} {
		node := values.Content[i]
		require.Equal(t, want.value, node.Value)
		require.Equal(t, want.tag, node.Tag)
		require.Equal(t, i+2, node.Line)
		require.Equal(t, 5, node.Column)
	}
}

func TestUnsupportedSpecTypeDistinguishesSchemaFromInvalidAPI(t *testing.T) {
	info, err := ExtractSpecInfo([]byte(`{"type":"object","properties":{"id":{"type":"string"}}}`))
	require.ErrorIs(t, err, ErrUnsupportedSpecType)
	require.ErrorIs(t, info.Error, ErrUnsupportedSpecType)
	require.NotNil(t, info.RootNode)
	for _, doc := range []string{"openapi: [", "openapi: '2.0.0'\n"} {
		_, err = ExtractSpecInfo([]byte(doc))
		require.Error(t, err)
		require.False(t, errors.Is(err, ErrUnsupportedSpecType))
	}
}
