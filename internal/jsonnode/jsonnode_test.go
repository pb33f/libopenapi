// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package jsonnode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

// sameTree reports the first difference between two node trees, comparing every field yaml sets.
func sameTree(want, got *yaml.Node, path string) error {
	switch {
	case want.Kind != got.Kind:
		return fmt.Errorf("%s: kind %v != %v", path, got.Kind, want.Kind)
	case want.Style != got.Style:
		return fmt.Errorf("%s: style %v != %v", path, got.Style, want.Style)
	case want.Tag != got.Tag:
		return fmt.Errorf("%s: tag %q != %q", path, got.Tag, want.Tag)
	case want.Value != got.Value:
		return fmt.Errorf("%s: value %q != %q", path, got.Value, want.Value)
	case want.Line != got.Line || want.Column != got.Column:
		return fmt.Errorf("%s: position %d:%d != %d:%d", path, got.Line, got.Column, want.Line, want.Column)
	case want.Anchor != got.Anchor || want.Alias != got.Alias || want.Stream != got.Stream:
		return fmt.Errorf("%s: anchor, alias or stream differ", path)
	case want.HeadComment != got.HeadComment || want.LineComment != got.LineComment ||
		want.FootComment != got.FootComment:
		return fmt.Errorf("%s: comments differ", path)
	case (want.Content == nil) != (got.Content == nil) || len(want.Content) != len(got.Content):
		return fmt.Errorf("%s: content %d (nil %v) != %d (nil %v)", path,
			len(got.Content), got.Content == nil, len(want.Content), want.Content == nil)
	}
	for i := range want.Content {
		if err := sameTree(want.Content[i], got.Content[i], fmt.Sprintf("%s/%d", path, i)); err != nil {
			return err
		}
	}
	return nil
}

// checkParse asserts the parser either declines the input or builds exactly the tree yaml builds.
func checkParse(t testing.TB, data []byte) bool {
	t.Helper()
	got, ok := Parse(data)
	if !ok {
		return false
	}
	var want yaml.Node
	require.NoError(t, yaml.Unmarshal(data, &want), "accepted input that yaml rejects: %q", data)
	require.NoError(t, sameTree(&want, got, "doc"), "input: %q", data)
	return true
}

func TestParse_MatchesYAML(t *testing.T) {
	accepted := []string{
		`{}`,
		`{"a":1}`,
		"\n\n  {\"a\": {}, \"b\": [], \"c\": [1, -2.5e3, true, null, \"x\\u00e9\\n\"]}\n\n",
		"{\r\n  \"a\" : 1,\r\n  \"b\": [\r\n    2\r\n  ]\r\n}\r\n",
		"{\n\t\"a\":\t1\n}",
		`{"é": "ü", "日本": "語", "emoji": "😀", "x": "a\u00e9b"}`,
		`{"esc": "\"\\\b\f\n\r\t\u0041\u00e9\u20AC\u0000"}`,
		`{"n": [0, -0, -0.0, -0.00, 1, -1, 0.5, -0.5, 1e5, 1E5, 1e+5, 1e-5, 1.5E-10, 123456789012345678901234567890]}`,
		`{"range": [9223372036854775807, 9223372036854775808, 18446744073709551615, 18446744073709551616, -9223372036854775808, -9223372036854775809]}`,
		`{"inf": 1e400, "tiny": 1e-400}`,
		`{"nested": {"deeper": {"deepest": [[[{"a": [null]}]]]}}}`,
		`{"":""}`,
		`{"a":"b" , "c" :"d"}`,
		"{\"key\":\n  \"value on the next line\"}",
		`{"dup": 1, "dup": 2}`,
		`{"#": "# not a comment", "&a": "*a", "!tag": "- item", "k": "a: b", "q": "'"}`,
		`{"long": "` + strings.Repeat("long text ", 200) + `"}`,
		`{"` + strings.Repeat("k", 2000) + `": 1}`,
		`{"c1ok": "\u0085\u2028\u2029\ufeff\uffff"}`,
		`{"del": "\u007f"}`,
	}
	for _, src := range accepted {
		require.True(t, checkParse(t, []byte(src)), "expected the parser to accept %q", src)
	}

	declined := []string{
		``,
		`   `,
		`[]`,
		`"string"`,
		`{`,
		`{"a"`,
		`{"a":`,
		`{"a":1`,
		`{"a":1,`,
		`{"a":1,}`,
		`{"a" 1}`,
		`{a: 1}`,
		`{"a": 1} x`,
		`{"a": 1} {}`,
		"\t{}",
		"{}\t",
		"{\r}",
		"{\"a\"\r: 1}",
		"{\"a\":\r1}",
		"{\"a\":1,\r\"b\":2}",
		`{"a": "\u12`,
		"{\"a\"\n: 1}",
		`{"a": [1,]}`,
		`{"a": [1 2]}`,
		`{"a": tru}`,
		`{"a": truex}`,
		`{"a": nul}`,
		`{"a": fals}`,
		`{"a": -}`,
		`{"a": 01}`,
		`{"a": 1.}`,
		`{"a": 1.e5}`,
		`{"a": 1e}`,
		`{"a": 1e+}`,
		`{"a": .5}`,
		`{"a": +1}`,
		`{"a": 0x1F}`,
		`{"a": x}`,
		`{"a": "unterminated}`,
		`{"a": "bad \x escape"}`,
		`{"a": "solidus \/ escape"}`,
		`{"a": "bad \u12 escape"}`,
		`{"a": "bad \u12G4 escape"}`,
		`{"a": "\ud83d\ude00"}`,
		`{"a": "\udc00"}`,
		"{\"a\": \"raw\ttab\"}",
		"{\"a\": \"raw\nbreak\"}",
		"{\"a\": \"del\x7f\"}",
		"{\"a\": \"c1\u0085\"}",
		"{\"a\": \"ls\xe2\x80\xa8\"}",
		"{\"a\": \"ps\xe2\x80\xa9\"}",
		"{\"a\": \"bom\xef\xbb\xbf\"}",
		"{\"a\": \"nonchar\xef\xbf\xbf\"}",
		"{\"a\": \"bad \xff utf8\"}",
		"{\"a\": \"truncated \xe2\x82\"}",
		`{"a": "trailing backslash\`,
		strings.Repeat(`{"a":`, maxDepth+1) + "1" + strings.Repeat("}", maxDepth+1),
		strings.Repeat(`{"a":[`, maxDepth/2) + "1" + strings.Repeat("]}", maxDepth/2) + "x",
	}
	for _, src := range declined {
		_, ok := Parse([]byte(src))
		require.False(t, ok, "expected the parser to decline %q", src)
	}
}

// Every JSON fixture in the repository, and every YAML fixture re-encoded as JSON in several layouts,
// parses to exactly the tree yaml builds.
func TestParse_RepositoryFixtures(t *testing.T) {
	root := filepath.Join("..", "..")
	var jsonFiles, converted int
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		require.NoError(t, err)
		if d.IsDir() && (d.Name() == ".git" || d.Name() == ".claude") {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		switch filepath.Ext(path) {
		case ".json":
			trimmed := bytes.TrimSpace(data)
			if len(trimmed) > 0 && trimmed[0] == '{' && trimmed[len(trimmed)-1] == '}' {
				require.True(t, checkParse(t, data), "declined fixture %s", path)
				jsonFiles++
			}
		case ".yaml", ".yml":
			var decoded map[string]any
			if yaml.Unmarshal(data, &decoded) != nil || decoded == nil {
				return nil
			}
			compact, err := json.Marshal(decoded)
			if err != nil {
				return nil // yaml values json cannot represent, such as maps with non-string keys
			}
			indented, err := json.MarshalIndent(decoded, "", "  ")
			require.NoError(t, err)
			tabbed, err := json.MarshalIndent(decoded, "", "\t")
			require.NoError(t, err)
			for _, doc := range [][]byte{compact, indented, tabbed} {
				require.True(t, checkParse(t, doc), "declined %s re-encoded as JSON", path)
			}
			converted++
		}
		return nil
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, jsonFiles, 10)
	require.GreaterOrEqual(t, converted, 100)
}

// Unmarshal decodes JSON through Parse and everything else through yaml, matching yaml.Unmarshal either way.
func TestUnmarshal(t *testing.T) {
	for _, src := range []string{`{"a": [1, "b"]}`, "a: 1\nb: [2]\n", `{"a": 1} x`, `{a: 1}`, `{"a": "\/"}`} {
		var want, got yaml.Node
		wantErr := yaml.Unmarshal([]byte(src), &want)
		gotErr := Unmarshal([]byte(src), &got)
		require.Equal(t, wantErr, gotErr, "input %q", src)
		require.NoError(t, sameTree(&want, &got, "doc"), "input %q", src)
	}
}

func FuzzParse(f *testing.F) {
	for _, seed := range []string{
		`{}`, `{"a":1}`, `{"a":[1,2.5,-3e4,true,false,null,"s"]}`, `{"a":{"b":{"c":"\u00e9\n"}}}`,
		"{\r\n\t\"a\" : \"b\"\r\n}", `{"a":"😀"}`, `{"":0}`,
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		checkParse(t, data)
	})
}
