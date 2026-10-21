package index

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/pb33f/go-yaml"
)

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
			}{{"stack_only", wideShared}, {"both", wideUnique}, {"visited_only_cycle", deep}, {"pending_siblings", pending}}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					ClearHashCache()
					defer ClearHashCache()
					want := frozenUncachedHashNode(tc.root)
					for repeat := 0; repeat < 3; repeat++ {
						if got := HashNode(tc.root); got != want {
							t.Fatalf("repeat %d got %s want %s", repeat, got, want)
						}
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
			if got, want := HashNode(root), frozenUncachedHashNode(root); got != want {
				t.Fatalf("got %s want %s", got, want)
			}
			root.Content[0], root.Content[len(root.Content)-1] = root.Content[len(root.Content)-1], root.Content[0]
			ClearHashCache()
			if got, want := HashNode(root), frozenUncachedHashNode(root); got != want {
				t.Fatalf("reordered got %s want %s", got, want)
			}
		})
	}
}

func BenchmarkHashNodeSmallTraversalBoundaries(b *testing.B) {
	for _, size := range []int{8, 16, 17, 32} {
		b.Run(fmt.Sprintf("deep_unique_%d", size), func(b *testing.B) {
			root := hashNodeFastBenchmarkChain(size)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				hashNodeFastBenchmarkSink = HashNode(root)
			}
		})
		b.Run(fmt.Sprintf("wide_duplicates_%d", size), func(b *testing.B) {
			shared := &yaml.Node{Value: "same"}
			wide := &yaml.Node{Content: make([]*yaml.Node, size)}
			for i := range wide.Content {
				wide.Content[i] = shared
			}
			// the narrow root avoids cache admission and measures each traversal.
			root := &yaml.Node{Content: []*yaml.Node{wide}}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				hashNodeFastBenchmarkSink = HashNode(root)
			}
		})
	}
}
