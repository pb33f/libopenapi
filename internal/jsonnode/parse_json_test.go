// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package jsonnode

import (
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

func TestParseJSONCoordinatesAndTypes(t *testing.T) {
	const source = "\t{\r\n\"values\"\r:\t[true, null, -0,\n\t1e9999, 18446744073709551617000]}\t"
	doc, ok := ParseJSON([]byte(source))
	require.True(t, ok)
	root := doc.Content[0]
	require.Equal(t, yaml.MappingNode, root.Kind)
	require.Equal(t, 1, root.Line)
	require.Equal(t, 2, root.Column)
	key, values := root.Content[0], root.Content[1]
	require.Equal(t, "values", key.Value)
	require.Equal(t, 2, key.Line)
	require.Equal(t, 1, key.Column)
	require.Equal(t, 3, values.Line)
	require.Equal(t, 3, values.Column)
	for i, want := range []struct {
		value, tag   string
		line, column int
	}{
		{"true", "!!bool", 3, 4}, {"null", "!!null", 3, 10}, {"-0", "!!int", 3, 16},
		{"1e9999", "!!float", 4, 2}, {"18446744073709551617000", "!!int", 4, 10},
	} {
		got := values.Content[i]
		require.Equal(t, want.tag, got.Tag)
		require.Equal(t, want.value, got.Value)
		require.Equal(t, want.line, got.Line)
		require.Equal(t, want.column, got.Column)
	}
	doc, ok = ParseJSON([]byte("\tfalse\r"))
	require.True(t, ok)
	require.Equal(t, "false", doc.Content[0].Value)
	require.Equal(t, "!!bool", doc.Content[0].Tag)
	_, ok = Parse([]byte(source))
	require.False(t, ok)
}

func TestParseJSONEscapesPreserveFollowingCoordinates(t *testing.T) {
	for _, tc := range []struct{ name, token, decoded string }{
		{"surrogate pair", `"\uD83D\uDE00"`, "😀"},
		{"lone high", `"\uD800"`, "\ufffd"},
		{"lone low", `"\uDC00"`, "\ufffd"},
		{"two high then low", `"\uD800\uD83D\uDE00"`, "\ufffd😀"},
		{"high and BMP", `"\uD800\u0041"`, "\ufffdA"},
		{"escaped backslash", `"\\uD83D\\uDE00"`, `\uD83D\uDE00`},
		{"JSON characters outside YAML repertoire", "\"x\\/\x7f\u0085\u2028😀\"", "x/\x7f\u0085\u2028😀"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prefix := "[" + tc.token + ","
			doc, ok := ParseJSON([]byte(prefix + "42]"))
			require.True(t, ok)
			items := doc.Content[0].Content
			require.Equal(t, tc.decoded, items[0].Value)
			require.Equal(t, 1, items[1].Line)
			require.Equal(t, len([]rune(prefix))+1, items[1].Column)
		})
	}
}

func TestParseJSONChecksTheWholeInput(t *testing.T) {
	for _, input := range []string{`"\uD800\uZZZZ"`, "\"\xff\"", "\"\x1f\"", "true false", `{"a": true}[]`, `[1,]`} {
		doc, ok := ParseJSON([]byte(input))
		require.False(t, ok, "%q", input)
		require.Nil(t, doc)
	}
}

func TestParseJSONNestingLimit(t *testing.T) {
	nested := func(levels int) []byte {
		return []byte(`{"v":` + strings.Repeat("[", levels) + "true" + strings.Repeat("]", levels) + "}")
	}
	source := nested(9999) // root object + 9,999 arrays
	doc, ok := ParseJSON(source)
	require.True(t, ok)
	node := doc.Content[0].Content[1]
	for node.Kind == yaml.SequenceNode {
		require.Len(t, node.Content, 1)
		node = node.Content[0]
	}
	require.Equal(t, "true", node.Value)
	_, ok = Parse(source)
	require.False(t, ok)
	_, ok = ParseJSON(nested(10000))
	require.False(t, ok)
}
