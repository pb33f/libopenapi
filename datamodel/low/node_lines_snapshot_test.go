package low

import (
	"fmt"
	"github.com/pb33f/go-yaml"
	"reflect"
	"slices"
	"sync"
	"testing"
)

func TestNodeMapGetNodeLines(t *testing.T) {
	cases := []struct {
		name   string
		values map[int]any
	}{
		{"nil nodes", nil}, {"empty", map[int]any{}}, {"single", map[int]any{4: &yaml.Node{}}},
		{"mixed", map[int]any{-2: &yaml.Node{}, 0: []*yaml.Node{{}, {}}, 7: (*yaml.Node)(nil), 9: []*yaml.Node(nil), 11: []*yaml.Node{}, 13: "ignored", 14: nil}},
		{"invalid only", map[int]any{5: "invalid", 9: 42}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			nm := &NodeMap{}
			if tc.values != nil {
				nm.Nodes = &NodeLines{}
				for k, v := range tc.values {
					nm.Nodes.Store(k, v)
				}
			}
			got := nm.GetNodeLines()
			want := []int{}
			for k := range nm.GetNodes() {
				want = append(want, k)
			}
			slices.Sort(got)
			slices.Sort(want)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v want %v", got, want)
			}
			got[0] = 999
			fresh := nm.GetNodeLines()
			slices.Sort(fresh)
			if !reflect.DeepEqual(fresh, want) {
				t.Fatalf("snapshot aliases storage: %v", fresh)
			}
		})
	}
}

func TestNodeMapGetNodeLinesMutation(t *testing.T) {
	nm := &NodeMap{Nodes: &NodeLines{}}
	nm.Nodes.Store(2, "discard")
	nm.AddNode(3, &yaml.Node{})
	nm.Nodes.Store(2, &yaml.Node{})
	nm.AddNode(3, &yaml.Node{})
	before := nm.GetNodeLines()
	slices.Sort(before)
	nm.Nodes.Store(2, "now invalid")
	nm.Nodes.Store(7, (*yaml.Node)(nil))
	nm.AddNode(9, &yaml.Node{})
	after := nm.GetNodeLines()
	slices.Sort(after)
	if !reflect.DeepEqual(before, []int{2, 3}) || !reflect.DeepEqual(after, []int{3, 7, 9}) {
		t.Fatalf("before %v after %v", before, after)
	}
}

func TestNodeMapGetNodeLinesConcurrent(t *testing.T) {
	nm := &NodeMap{Nodes: &NodeLines{}}
	var wg sync.WaitGroup
	for worker := 0; worker < 4; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				nm.Nodes.Store(worker*1000+i, &yaml.Node{})
				lines := nm.GetNodeLines()
				seen := make(map[int]bool, len(lines))
				for _, line := range lines {
					if seen[line] {
						t.Errorf("duplicate line %d", line)
					}
					seen[line] = true
				}
			}
		}(worker)
	}
	wg.Wait()
	if len(nm.GetNodeLines()) != 4000 {
		t.Fatal("missing writes")
	}
}

func BenchmarkNodeMapLineNumbers(b *testing.B) {
	for _, size := range []int{1, 32, 1024} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			nm := &NodeMap{Nodes: &NodeLines{}}
			for i := 0; i < size; i++ {
				nm.Nodes.Store(i, &yaml.Node{})
			}
			nm.GetNodes()
			b.Run("map", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					n := 0
					for range nm.GetNodes() {
						n++
					}
					if n != size {
						b.Fatal(n)
					}
				}
			})
			b.Run("lines", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					n := 0
					for range nm.GetNodeLines() {
						n++
					}
					if n != size {
						b.Fatal(n)
					}
				}
			})
		})
	}
}
