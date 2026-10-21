package index

import (
	"fmt"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func originIdentityIndex(path string, nodes ...*yaml.Node) *SpecIndex {
	idx := &SpecIndex{specAbsolutePath: path}
	for _, n := range nodes {
		for len(idx.nodeLines) <= n.Line {
			idx.nodeLines = append(idx.nodeLines, nil)
		}
		idx.nodeLines[n.Line] = append(idx.nodeLines[n.Line], nodeLineEntry{column: int32(n.Column), node: n})
	}
	sortNodeLines(idx.nodeLines)
	return idx
}

func originIdentityValue(line int, children int) *yaml.Node {
	n := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Line: line, Column: 2}
	for i := 0; i < children; i++ {
		n.Content = append(n.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: fmt.Sprint(i), Line: line + i + 1, Column: 4})
	}
	return n
}

func TestOriginIdentityEquivalent(t *testing.T) {
	key := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "root", Line: 1, Column: 1}
	value := originIdentityValue(2, 2)
	extKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "external", Line: 20, Column: 1}
	extValue := originIdentityValue(30, 2)
	rootIndex := originIdentityIndex("root.yaml", key, value)
	keyIndex := originIdentityIndex("key.yaml", extKey)
	valueIndex := originIdentityIndex("value.yaml", extValue)
	r := &Rolodex{rootIndex: rootIndex, indexes: []*SpecIndex{keyIndex, valueIndex}}
	rootOrigin := &NodeOrigin{Node: key, Line: 1, Column: 1, AbsoluteLocation: "root.yaml", Index: rootIndex}
	externalOrigin := &NodeOrigin{Node: extKey, Line: 20, Column: 1, AbsoluteLocation: "key.yaml", Index: keyIndex}
	rootWithValue := *rootOrigin
	rootWithValue.AbsoluteLocationValue = "value.yaml"
	rootWithValue.LineValue, rootWithValue.ColumnValue, rootWithValue.ValueNode = 30, 2, extValue
	externalWithValue := *externalOrigin
	externalWithValue.AbsoluteLocationValue = "value.yaml"
	externalWithValue.LineValue, externalWithValue.ColumnValue, externalWithValue.ValueNode = 30, 2, extValue
	clone := *value
	mismatch := *value
	mismatch.Value = "different"
	missing := &yaml.Node{Kind: yaml.ScalarNode, Value: "missing", Line: 99, Column: 99}
	refs := []struct {
		name  string
		node  *yaml.Node
		value string
	}{
		{name: "none"},
		{name: "node only", node: key},
		{name: "value only", value: "#/schemas/Item"},
		{name: "both", node: key, value: "#/schemas/Item"},
	}
	cases := []struct {
		name       string
		key, value *yaml.Node
		want       [4]*NodeOrigin
		panics     [4]bool
	}{
		{
			name:  "root pointer",
			key:   key,
			value: value,
			want:  [4]*NodeOrigin{rootOrigin, rootOrigin, rootOrigin, rootOrigin},
		},
		{
			name:  "root clone",
			key:   key,
			value: &clone,
			want:  [4]*NodeOrigin{rootOrigin, rootOrigin, rootOrigin, rootOrigin},
		},
		{
			name:  "different content",
			key:   key,
			value: &mismatch,
			want:  [4]*NodeOrigin{nil, nil, nil, rootOrigin},
		},
		{
			name:  "external pointer",
			key:   extKey,
			value: extValue,
			want:  [4]*NodeOrigin{&externalWithValue, externalOrigin, externalOrigin, externalOrigin},
		},
		{
			name:  "root to external",
			key:   key,
			value: extValue,
			want:  [4]*NodeOrigin{&rootWithValue, rootOrigin, rootOrigin, rootOrigin},
		},
		{
			name:  "nil key",
			value: value,
		},
		{
			name: "nil value",
			key:  key,
			want: [4]*NodeOrigin{rootOrigin, rootOrigin, rootOrigin, rootOrigin},
		},
		{
			name:  "missing value",
			key:   key,
			value: missing,
			want:  [4]*NodeOrigin{nil, nil, nil, rootOrigin},
		},
		{
			name:   "missing key",
			key:    missing,
			value:  extValue,
			panics: [4]bool{true, true, true, false},
		},
	}
	for _, tc := range cases {
		for i, ref := range refs {
			t.Run(tc.name+"/"+ref.name, func(t *testing.T) {
				ClearHashCache()
				t.Cleanup(ClearHashCache)
				call := func() { assert.Equal(t, tc.want[i], r.FindNodeOriginWithValue(tc.key, tc.value, ref.node, ref.value)) }
				if tc.panics[i] {
					assert.Panics(t, call)
					return
				}
				assert.NotPanics(t, call)
			})
		}
	}
}

func TestOriginIdentityPopulatesHashCache(t *testing.T) {
	for _, external := range []bool{false, true} {
		t.Run(fmt.Sprintf("external=%t", external), func(t *testing.T) {
			key := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "key", Line: 1, Column: 1}
			value := originIdentityValue(2, hashCacheThreshold)
			r := &Rolodex{rootIndex: originIdentityIndex("root.yaml", key, value)}
			if external {
				r.rootIndex = originIdentityIndex("root.yaml")
				r.indexes = []*SpecIndex{originIdentityIndex("key.yaml", key), originIdentityIndex("value.yaml", value)}
			}
			ClearHashCache()
			t.Cleanup(ClearHashCache)
			require.NotNil(t, r.FindNodeOriginWithValue(key, value, nil, ""))
			cached, ok := nodeHashCache.Load(value)
			require.True(t, ok, "origin lookup should warm cache-eligible values")
			value.Content[0].Value = "changed"
			ClearHashCache()
			assert.NotEqual(t, cached, HashNode(value), "invalidation must expose mutation")
		})
	}
}

func BenchmarkOriginIdentity(b *testing.B) {
	for _, external := range []bool{false, true} {
		for _, children := range []int{0, 2, hashCacheThreshold} {
			b.Run(fmt.Sprintf("external=%t/children=%d", external, children), func(b *testing.B) {
				key := &yaml.Node{Value: "key", Line: 1, Column: 1}
				value := originIdentityValue(2, children)
				r := &Rolodex{rootIndex: originIdentityIndex("root.yaml", key, value)}
				if external {
					r.rootIndex = originIdentityIndex("root.yaml")
					r.indexes = []*SpecIndex{originIdentityIndex("external.yaml", key, value)}
				}
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					if r.FindNodeOriginWithValue(key, value, nil, "") == nil {
						b.Fatal("missing origin")
					}
				}
			})
		}
	}
}
