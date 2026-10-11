package utils

import (
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

func TestCheckEnumForDuplicates_Values(t *testing.T) {
	for _, tc := range []struct {
		name       string
		yaml       string
		duplicates []int
	}{
		{"distinct arrays", `[[a], [b], [a, b], [b, a], []]`, nil},
		{"duplicate arrays", `[[a], [b], [a], [a]]`, []int{2, 3}},
		{"distinct objects", `[{a: 1}, {a: 2}, {b: 1}, {}]`, nil},
		{"object order", `[{a: 1, b: [2, 3]}, {b: [2.0, 3], a: 1.0}]`, []int{1}},
		{"nested types", `[{a: [1]}, {a: ["1"]}]`, nil},
		{"scalar types", `[1, "1", true, "true", null, "null", "", [], {}]`, nil},
		{"octal integers", `[012, 10, 12, -012, -10, 0o12, 10.0]`, []int{1, 4, 5, 6}},
		{"numbers", `[1, 1.0, 1e0, -0, 0.0, 0, 0.1, 1e-1]`, []int{1, 2, 4, 5, 7}},
		{"precision", `[9007199254740992, 9007199254740993, 9007199254740993.0]`, []int{2}},
		{"yaml scalars", `[true, TRUE, null, ~, 0x10, 16]`, []int{1, 3, 5}},
		{"aliases", `[&a {x: [1]}, *a, {x: [1.0]}]`, []int{1, 2}},
		{"repeated aliases", `[&a [1], [*a, *a], [[1], [1]]]`, []int{2}},
		{"delimiters", `[["a,b", c], [a, "b,c"], {"a:b": c}, {a: "b:c"}]`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var root yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(tc.yaml), &root))
			values := root.Content[0].Content
			duplicates := CheckEnumForDuplicates(values)
			require.Len(t, duplicates, len(tc.duplicates))
			for i, index := range tc.duplicates {
				require.Same(t, values[index], duplicates[i])
			}
		})
	}
}

func TestCheckEnumForDuplicates_NonJSONValues(t *testing.T) {
	for _, source := range []string{
		`[&a [*a], &b [*b]]`,
		`[{1: a}, {1: a}]`,
		`[{a: 1, a: 2}, {a: 1, a: 2}]`,
		`[.inf, .inf, .nan, .nan]`,
		`[!custom a, !custom a]`,
		`[!custom [], []]`,
		`[!custom {}, {}]`,
		`[{x: !custom a}, {x: !custom a}]`,
	} {
		t.Run(source, func(t *testing.T) {
			var root yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(source), &root))
			require.Empty(t, CheckEnumForDuplicates(root.Content[0].Content))
		})
	}
	for _, node := range []*yaml.Node{
		nil,
		{Kind: yaml.AliasNode},
		{Kind: yaml.MappingNode, Content: []*yaml.Node{nil}},
		{Kind: yaml.MappingNode, Content: []*yaml.Node{nil, nil}},
		{Kind: yaml.SequenceNode, Content: []*yaml.Node{nil}},
		{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "invalid"},
		{Kind: yaml.ScalarNode, Tag: "!!int", Value: "invalid"},
		{Kind: yaml.DocumentNode},
	} {
		require.Empty(t, CheckEnumForDuplicates([]*yaml.Node{node, node}))
	}
}

func TestCheckEnumForDuplicates_SharedAliases(t *testing.T) {
	value := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "1"}
	// The graph is linear in size; expanding its aliases would create 2^128 leaves.
	for range 128 {
		value = &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{
			{Kind: yaml.AliasNode, Alias: value},
			{Kind: yaml.AliasNode, Alias: value},
		}}
	}
	alias := &yaml.Node{Kind: yaml.AliasNode, Alias: value}
	duplicates := CheckEnumForDuplicates([]*yaml.Node{value, alias})
	require.Len(t, duplicates, 1)
	require.Same(t, alias, duplicates[0])
}
