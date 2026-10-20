package index

import (
	"hash/maphash"
	"math/rand"
	"strconv"
	"sync"
	"testing"

	"github.com/pb33f/go-yaml"
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
	cases := []*yaml.Node{nil, {}, leaf, cycle, alias, {Content: []*yaml.Node{nil, nil}}, {Content: []*yaml.Node{leaf, alias, cycle}}}
	rng := rand.New(rand.NewSource(424242))
	for iteration := 0; iteration < 400; iteration++ {
		nodes := make([]*yaml.Node, 1+rng.Intn(80))
		for i := range nodes {
			nodes[i] = &yaml.Node{Kind: yaml.Kind(1 << rng.Intn(5)), Tag: []string{"", "!!str", "!!map", "!!seq"}[rng.Intn(4)], Value: strconv.Itoa(rng.Intn(20)), Line: rng.Intn(200) - 10, Column: rng.Intn(30) - 5}
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
		cases = append(cases, nodes[0])
	}
	for i, n := range cases {
		want := frozenUncachedHashNode(n)
		if got := HashNode(n); got != want {
			t.Fatalf("case %d: got %s want %s", i, got, want)
		}
		if got := HashNode(n); got != want {
			t.Fatalf("repeat case %d: got %s want %s", i, got, want)
		}
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
			_, cached := nodeHashCache.Load(root)
			if cached != (size >= hashCacheThreshold) {
				t.Fatalf("cache threshold changed at %d", size)
			}
			root.Content = nil
			root.Value = "mutated to leaf"
			want := frozenUncachedHashNode(root)
			if cached {
				want = original
			}
			if got := HashNode(root); got != want {
				t.Fatalf("cached-large-to-leaf semantics changed: got %s want %s", got, want)
			}
			ClearHashCache()
			if got, want := HashNode(root), frozenUncachedHashNode(root); got != want {
				t.Fatalf("clear did not expose mutation: got %s want %s", got, want)
			}
			root.Value = "mutated leaf again"
			if got, want := HashNode(root), frozenUncachedHashNode(root); got != want {
				t.Fatalf("uncached leaf mutation: got %s want %s", got, want)
			}
		})
	}
}

func TestHashNode_FastPathPoolsReleaseNodes(t *testing.T) {
	root := &yaml.Node{Content: make([]*yaml.Node, 17)}
	for i := range root.Content {
		root.Content[i] = &yaml.Node{Content: []*yaml.Node{root, nil}, Value: strconv.Itoa(i)}
	}
	HashNode(root)
	visited := visitedPool.Get().(map[*yaml.Node]struct{})
	if len(visited) != 0 {
		t.Errorf("pooled visited map retains %d nodes", len(visited))
	}
	visitedPool.Put(visited)
	stack := stackPool.Get().(*[]*yaml.Node)
	if len(*stack) != 0 {
		t.Errorf("pooled stack length is %d", len(*stack))
	}
	for i, n := range (*stack)[:cap(*stack)] {
		if n != nil {
			t.Errorf("pooled stack retains node at %d", i)
		}
	}
	stackPool.Put(stack)
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
					if got := HashNode(n); got != wants[i] {
						t.Errorf("concurrent mismatch: got %s want %s", got, wants[i])
						return
					}
				}
				ClearNodePools()
			}
		}()
	}
	wg.Wait()
}
