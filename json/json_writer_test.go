package json

import (
	"encoding/json"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

// requireMatchesMarshal asserts YAMLNodeToJSON gives exactly what converting the tree and marshaling it
// with encoding/json gives: the same bytes, or the same error.
func requireMatchesMarshal(t testing.TB, node *yaml.Node, indentation string) {
	t.Helper()
	want, wantErr := convertAndMarshal(node, indentation)
	got, gotErr := YAMLNodeToJSON(node, indentation)
	if wantErr != nil {
		require.EqualError(t, gotErr, wantErr.Error())
		return
	}
	require.NoError(t, gotErr)
	require.Equal(t, string(want), string(got))
}

func requireYAMLMatchesMarshal(t testing.TB, src string) {
	t.Helper()
	var node yaml.Node
	if yaml.Unmarshal([]byte(src), &node) != nil {
		return
	}
	for _, indentation := range []string{"", "  ", "\t"} {
		requireMatchesMarshal(t, &node, indentation)
	}
}

// Every fixture in the repository converts exactly as encoding/json marshals it.
func TestYAMLNodeToJSON_RepositoryFixtures(t *testing.T) {
	var converted int
	err := filepath.WalkDir(filepath.Join(".."), func(path string, d fs.DirEntry, err error) error {
		require.NoError(t, err)
		if d.IsDir() && (d.Name() == ".git" || d.Name() == ".claude") {
			return filepath.SkipDir
		}
		switch filepath.Ext(path) {
		case ".yaml", ".yml", ".json":
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			requireYAMLMatchesMarshal(t, string(data))
			converted++
		}
		return nil
	})
	require.NoError(t, err)
	require.Greater(t, converted, 100)
}

func TestYAMLNodeToJSON_MatchesMarshal(t *testing.T) {
	var many strings.Builder
	for i := 0; i < 3*dupScanLimit; i++ {
		many.WriteString("k" + strings.Repeat("x", i) + ": 1\n")
	}
	manyWithRepeat := many.String() + "kxxx: 2\n"
	for _, src := range []string{
		"a: 1\nb: [1, 2.5, -3e4, true, false, null, s]\nc: {d: e}\n",
		"html: '<a href=\"x\">&amp;</a>'\ncontrol: \"\\b\\f\\n\\r\\t\\x01\\x7f\"\nsep: \"\\u2028\\u2029\"\n",
		"unicode: \"é 日本 😀\"\n",
		"ints: [0, -1, 123, 007, 0x1F, 0o17, 1_000, +5, -0, 9223372036854775807, 9223372036854775808, 18446744073709551616]\n",
		"floats: [0.5, -0.5, 1e5, 1E-7, 1.0e21, 1e20, 123456789012345678901234567890, .5, 5., -0.0, 1e400, .inf, -.inf, .nan]\n",
		"tagged: [!!int 5, !!int -0, !!int 1.5, !!float 5, !!float 9223372036854775808, !!float 1e400, !!float -0, !!float abc]\n",
		"bools: [true, True, TRUE, false, False, FALSE, !!bool yes]\nnulls: [~, null, Null, NULL, !!null '', !!null x]\n",
		"stamp: 2001-12-14\nstampt: 2001-12-14t21:59:43.10-05:00\nbin: !!binary aGVsbG8=\nbadbin: !!binary '@@'\n",
		"strs: [!!str 5, '5', \"true\", !custom thing, !!str]\n",
		"200: ok\n1.5: float\ntrue: bool\n~: null\n2001-12-14: date\n.nan: nan\n[a, b]: seq\n{x: y}: map\n",
		"dup: 1\ndup: 2\nother: 3\n",
		manyWithRepeat,
		many.String(),
		"base: &b {x: 1}\nuse: *b\nlist: &l [1, 2]\nagain: *l\n",
		"<<: {merged: true}\nplain: 1\n",
		"empty: {}\nemptyList: []\nnested: [[], {}]\n",
		"just a scalar\n",
		"- a\n- b\n",
		"",
	} {
		requireYAMLMatchesMarshal(t, src)
	}
}

// Node trees yaml.Unmarshal never builds: untagged scalars, broken structure and recursion.
func TestYAMLNodeToJSON_ConstructedTrees(t *testing.T) {
	scalar := func(tag, value string, style yaml.Style) *yaml.Node {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: value, Style: style}
	}
	mapping := func(content ...*yaml.Node) *yaml.Node {
		return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: content}
	}
	recursive := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Anchor: "x"}
	recursive.Content = []*yaml.Node{{Kind: yaml.AliasNode, Value: "x", Alias: recursive}}
	recursiveKey := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	recursiveKey.Content = []*yaml.Node{{Kind: yaml.AliasNode, Value: "k", Alias: recursiveKey}, scalar("!!str", "v", 0)}
	nanNode := scalar("!!float", ".nan", 0)
	for _, node := range []*yaml.Node{
		nil,
		{Kind: yaml.DocumentNode},
		{Kind: yaml.Kind(99)},
		{Kind: yaml.DocumentNode, Content: []*yaml.Node{mapping(scalar("", "5", 0), scalar("", "true", 0))}},
		mapping(scalar("", "quoted", yaml.DoubleQuotedStyle), scalar("tag:yaml.org,2002:str", "long", 0)),
		mapping(nil, scalar("!!str", "v", 0)),
		mapping(scalar("!!str", "k", 0), nil),
		mapping(scalar("!!str", "odd", 0)),
		mapping(scalar("!!str", "a", 0), nanNode, scalar("!!str", "b", 0), &yaml.Node{Kind: yaml.Kind(99)}),
		mapping(nanNode, scalar("!!str", "nan key", 0)),
		mapping(scalar("!!str", "a", 0), nanNode, scalar("!!str", "a", 0), scalar("!!str", "dup", 0)),
		mapping(scalar("!!str", "a", 0), scalar("!!str", "1", 0), scalar("!!str", "a", 0), &yaml.Node{Kind: yaml.Kind(99)}),
		{Kind: yaml.SequenceNode, Tag: "!!seq", Content: []*yaml.Node{scalar("!!int", "1", 0), nil}},
		recursive,
		recursiveKey,
		nanNode,
		{Kind: yaml.SequenceNode, Tag: "!!seq", Content: []*yaml.Node{
			scalar("!!float", "1e", 0), scalar("!!float", "1.5e+", 0), scalar("!!float", "1.", 0)}},
	} {
		for _, indentation := range []string{"", "  "} {
			requireMatchesMarshal(t, node, indentation)
		}
	}
}

func FuzzYAMLNodeToJSON(f *testing.F) {
	for _, seed := range []string{
		"a: 1\n", "a: [1, 2.5, true, null, 'x']\n", "{\"a\": {\"b\": \"<&>\"}}", "200: x\n1.5: y\n",
		"a: &x [1]\nb: *x\n", "a: 1\na: 2\n", "x: !!float 1e5\ny: !!int 7\n", "- 2001-12-14\n- !!binary aGk=\n",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, src string) {
		requireYAMLMatchesMarshal(t, src)
	})
}

// appendJSONString and appendJSONFloat write exactly what encoding/json writes.
func FuzzAppendJSONString(f *testing.F) {
	for _, seed := range []string{"", "plain", "<a&b>", "\"\\\b\f\n\r\t\x00\x1f\x7f", "é😀  ", "\xff\xfe bad"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		want, err := json.Marshal(s)
		require.NoError(t, err)
		require.Equal(t, string(want), string(appendJSONString(nil, s)))
	})
}

func FuzzAppendJSONFloat(f *testing.F) {
	for _, seed := range []float64{0, math.Copysign(0, -1), 1, -1.5, 1e-7, 1e-6, 1e20, 1e21, 123456789.125, math.MaxFloat64, math.SmallestNonzeroFloat64} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, v float64) {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return
		}
		want, err := json.Marshal(v)
		require.NoError(t, err)
		require.Equal(t, string(want), string(appendJSONFloat(nil, v)))
	})
}
