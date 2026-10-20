package index

import (
	"fmt"
	"testing"

	"github.com/pb33f/go-yaml"
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
	r := &Rolodex{rootIndex: originIdentityIndex("root.yaml", key, value), indexes: []*SpecIndex{originIdentityIndex("key.yaml", extKey), originIdentityIndex("value.yaml", extValue)}}
	clone := *value
	mismatch := *value
	mismatch.Value = "different"
	missing := &yaml.Node{Kind: yaml.ScalarNode, Value: "missing", Line: 99, Column: 99}
	for _, tc := range []struct {
		name       string
		key, value *yaml.Node
	}{
		{"root pointer", key, value}, {"root clone", key, &clone}, {"different content", key, &mismatch},
		{"external pointer", extKey, extValue}, {"root to external", key, extValue},
		{"nil key", nil, value}, {"nil value", key, nil}, {"missing value", key, missing},
		{"missing key", missing, extValue},
	} {
		for _, refNode := range []*yaml.Node{nil, key} {
			for _, ref := range []string{"", "#/schemas/Item"} {
				t.Run(fmt.Sprintf("%s/refNode=%t/ref=%t", tc.name, refNode != nil, ref != ""), func(t *testing.T) {
					call := func(f func(*yaml.Node, *yaml.Node, *yaml.Node, string) *NodeOrigin) (out *NodeOrigin, panicked bool) {
						defer func() {
							if recover() != nil {
								panicked = true
							}
						}()
						out = f(tc.key, tc.value, refNode, ref)
						return
					}
					ClearHashCache()
					want, wp := call(r.findNodeOriginWithValueReference)
					ClearHashCache()
					got, gp := call(r.FindNodeOriginWithValue)
					if gp != wp || (got == nil) != (want == nil) || (got != nil && *got != *want) {
						t.Fatalf("got %+v panic=%v, want %+v panic=%v", got, gp, want, wp)
					}
					if tc.name == "external pointer" && refNode == nil && ref == "" {
						if got.AbsoluteLocation != "key.yaml" || got.AbsoluteLocationValue != "value.yaml" || got.LineValue != 30 || got.ColumnValue != 2 || got.ValueNode != extValue {
							t.Fatalf("external metadata lost: %+v", got)
						}
					}
				})
			}
		}
	}
}

func TestOriginIdentityPreservesHashCachePopulation(t *testing.T) {
	for _, external := range []bool{false, true} {
		for _, withRef := range []bool{false, true} {
			t.Run(fmt.Sprintf("external=%t/ref=%t", external, withRef), func(t *testing.T) {
				key := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "key", Line: 1, Column: 1}
				value := originIdentityValue(2, hashCacheThreshold)
				r := &Rolodex{rootIndex: originIdentityIndex("root.yaml", key, value)}
				if external {
					r.rootIndex = originIdentityIndex("root.yaml")
					r.indexes = []*SpecIndex{originIdentityIndex("key.yaml", key), originIdentityIndex("value.yaml", value)}
				}
				var refNode *yaml.Node
				ref := ""
				if withRef {
					refNode = key
					ref = "#/schemas/Item"
				}
				ClearHashCache()
				if r.FindNodeOriginWithValue(key, value, refNode, ref) == nil {
					t.Fatal("missing origin")
				}
				cached, ok := nodeHashCache.Load(value)
				// The existing external-reference early return occurs before hashing the value.
				if external && withRef {
					if ok {
						t.Fatal("external ref unexpectedly populated value cache")
					}
					return
				}
				if !ok {
					t.Fatal("origin lookup no longer populates the value hash cache")
				}
				value.Content[0].Value = "changed"
				if HashNode(value) != cached {
					t.Fatal("cached hash changed without invalidation")
				}
				ClearHashCache()
				if HashNode(value) == cached {
					t.Fatal("cache invalidation ignored mutation")
				}
			})
		}
	}
}
