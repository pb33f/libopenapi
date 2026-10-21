package index

import (
	"fmt"
	"hash/maphash"
	"math/rand"
	"strconv"
	"sync"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

// frozenUncachedHashNode preserves the pre-optimization byte stream independently
// of the production writer, caches, and pools. Aliases are not traversed.
func frozenUncachedHashNode(root *yaml.Node) string {
	var h maphash.Hash
	h.SetSeed(globalHashSeed)
	stack := []*yaml.Node{root}
	seen := make(map[*yaml.Node]bool)
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if n == nil || seen[n] {
			continue
		}
		seen[n] = true
		h.WriteString(n.Tag)
		// The historical writer emits nothing for negative coordinates.
		if n.Line >= 0 {
			h.WriteString(strconv.Itoa(n.Line))
		}
		if n.Column >= 0 {
			h.WriteString(strconv.Itoa(n.Column))
		}
		h.WriteString(n.Value)
		for i := len(n.Content) - 1; i >= 0; i-- {
			stack = append(stack, n.Content[i])
		}
	}
	return strconv.FormatUint(h.Sum64(), 16)
}

func TestHashNode_FastPathFrozenOracle(t *testing.T) {
	ClearHashCache()
	defer ClearHashCache()
	leaf := &yaml.Node{Tag: "!!str", Value: "leaf", Line: 3, Column: 7}
	cycle := &yaml.Node{Tag: "!!map", Line: -3, Column: -1}
	cycle.Content = []*yaml.Node{nil, leaf, cycle, leaf}
	alias := &yaml.Node{Kind: yaml.AliasNode, Value: "alias", Alias: cycle, Line: 9}
	cases := []struct {
		name string
		node *yaml.Node
	}{
		{name: "nil"},
		{name: "empty", node: &yaml.Node{}},
		{name: "leaf", node: leaf},
		{name: "cycle", node: cycle},
		{name: "alias", node: alias},
		{name: "nil_children", node: &yaml.Node{Content: []*yaml.Node{nil, nil}}},
		{name: "shared_children", node: &yaml.Node{Content: []*yaml.Node{leaf, alias, cycle}}},
	}
	rng := rand.New(rand.NewSource(424242))
	for iteration := 0; iteration < 400; iteration++ {
		nodes := make([]*yaml.Node, 1+rng.Intn(80))
		for i := range nodes {
			nodes[i] = &yaml.Node{
				Kind:   yaml.Kind(1 << rng.Intn(5)),
				Tag:    []string{"", "!!str", "!!map", "!!seq"}[rng.Intn(4)],
				Value:  strconv.Itoa(rng.Intn(20)),
				Line:   rng.Intn(200) - 10,
				Column: rng.Intn(30) - 5,
			}
		}
		for _, n := range nodes {
			n.Alias = nodes[rng.Intn(len(nodes))]
			for j := rng.Intn(25); j > 0; j-- {
				if rng.Intn(7) == 0 {
					n.Content = append(n.Content, nil)
				} else {
					n.Content = append(n.Content, nodes[rng.Intn(len(nodes))])
				}
			}
		}
		cases = append(cases, struct {
			name string
			node *yaml.Node
		}{name: fmt.Sprintf("random_%03d", iteration), node: nodes[0]})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := frozenUncachedHashNode(tc.node)
			require.Equal(t, want, HashNode(tc.node))
			require.Equal(t, want, HashNode(tc.node), "repeated lookup")
		})
	}

}

func TestHashNode_FastPathMutationAndThreshold(t *testing.T) {
	for _, size := range []int{hashCacheThreshold - 1, hashCacheThreshold, hashCacheThreshold + 1} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			ClearHashCache()
			defer ClearHashCache()
			root := &yaml.Node{Tag: "!!seq", Content: make([]*yaml.Node, size)}
			for i := range root.Content {
				root.Content[i] = &yaml.Node{Value: strconv.Itoa(i)}
			}
			original := HashNode(root)
			require.Equal(t, original, HashNode(root), "unchanged node retains its hash")
			_, cached := nodeHashCache.Load(root)
			require.Equal(t, size >= hashCacheThreshold, cached, "cache admission threshold")
			root.Content = nil
			root.Value = "mutated to leaf"
			ClearHashCache()
			require.Equal(t, frozenUncachedHashNode(root), HashNode(root), "invalidation exposes mutation")
			root.Value = "mutated leaf again"
			require.Equal(t, frozenUncachedHashNode(root), HashNode(root), "uncached leaf mutation")

		})
	}
}

func TestHashNode_FastPathConcurrent(t *testing.T) {
	ClearHashCache()
	defer ClearHashCache()
	leaf := &yaml.Node{Value: "leaf", Line: 8}
	root := &yaml.Node{Content: make([]*yaml.Node, hashCacheThreshold)}
	for i := range root.Content {
		root.Content[i] = leaf
	}
	root.Content[0] = root
	roots := []*yaml.Node{nil, leaf, root}
	wants := make([]string, len(roots))
	for i, n := range roots {
		wants[i] = frozenUncachedHashNode(n)
	}
	var wg sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for iteration := 0; iteration < 300; iteration++ {
				for i, n := range roots {
					if !assert.Equal(t, wants[i], HashNode(n), "concurrent lookup") {
						return
					}
				}
			}
		}()
	}
	wg.Wait()
}

func hashNodeFastBenchmarkChain(size int) *yaml.Node {
	root := &yaml.Node{Tag: "!!seq"}
	current := root
	for i := 1; i < size; i++ {
		next := &yaml.Node{Tag: "!!seq", Value: strconv.Itoa(i), Line: i}
		current.Content = []*yaml.Node{next}
		current = next
	}
	return root
}

func TestHashNode_SmallTraversalTransitions(t *testing.T) {
	for _, size := range []int{8, 15, 16, 17, 18, 31, 32, 33} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			leaf := &yaml.Node{Tag: "!!str", Value: "shared", Line: 4}
			wideShared := &yaml.Node{Content: make([]*yaml.Node, size)}
			wideUnique := &yaml.Node{Content: make([]*yaml.Node, size)}
			for i := 0; i < size; i++ {
				wideShared.Content[i] = leaf
				wideUnique.Content[i] = &yaml.Node{Value: strconv.Itoa(i), Line: i, Column: i % 3}
			}
			// stack-only promotion includes nils and duplicates in pending capacity.
			wideShared.Content[size/2] = nil
			deep := hashNodeFastBenchmarkChain(size)
			tail := deep
			for len(tail.Content) > 0 {
				tail = tail.Content[0]
			}
			// on size17 this ancestor cycle is reached directly after visited promotion.
			tail.Content = []*yaml.Node{deep, leaf, leaf, nil}
			// preserve existing pending siblings below newly pushed children on promotion.
			pending := &yaml.Node{Content: []*yaml.Node{wideUnique, leaf, deep, nil, wideShared}}
			cases := []struct {
				name string
				root *yaml.Node
			}{
				{name: "stack_only", root: wideShared},
				{name: "both", root: wideUnique},
				{name: "visited_only_cycle", root: deep},
				{name: "pending_siblings", root: pending},
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					ClearHashCache()
					defer ClearHashCache()
					want := frozenUncachedHashNode(tc.root)
					for repeat := 0; repeat < 3; repeat++ {
						require.Equal(t, want, HashNode(tc.root), "repeat %d", repeat)
					}
				})
			}
		})
	}
}

func TestHashNode_SmallTraversalOrderAndSharedNodes(t *testing.T) {
	for _, width := range []int{16, 17, 32} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			ClearHashCache()
			defer ClearHashCache()
			root := &yaml.Node{Tag: "root"}
			shared := &yaml.Node{Value: "shared"}
			for i := 0; i < width; i++ {
				root.Content = append(root.Content, &yaml.Node{Value: strconv.Itoa(i), Content: []*yaml.Node{shared, nil, root}})
			}
			require.Equal(t, frozenUncachedHashNode(root), HashNode(root))
			root.Content[0], root.Content[len(root.Content)-1] = root.Content[len(root.Content)-1], root.Content[0]
			ClearHashCache()
			require.Equal(t, frozenUncachedHashNode(root), HashNode(root), "reordered children")
		})
	}
}

// This is a smoke check only: sync.Pool may discard an item or return a fresh one.
// Cleanup relies on the traversal's explicit slot/map clearing; this test does
// not identify returned items or prove that every pooled item was cleared.
func TestHashNode_PooledStorageSmoke(t *testing.T) {
	root := &yaml.Node{Content: make([]*yaml.Node, 17)}
	for i := range root.Content {
		root.Content[i] = &yaml.Node{Content: []*yaml.Node{root, nil}, Value: strconv.Itoa(i)}
	}
	HashNode(root)
	visited := visitedPool.Get().(map[*yaml.Node]struct{})
	assert.Empty(t, visited)
	visitedPool.Put(visited)
	stack := stackPool.Get().(*[]*yaml.Node)
	assert.Empty(t, *stack)
	for i, node := range (*stack)[:cap(*stack)] {
		assert.Nil(t, node, "pooled backing slot %d", i)
	}
	stackPool.Put(stack)
}
