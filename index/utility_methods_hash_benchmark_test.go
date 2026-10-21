package index

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/pb33f/go-yaml"
)

var hashNodeFastBenchmarkSink string

// BenchmarkHashNodeTraversalWorkloads distinguishes scalar traffic, uncached
// narrow roots, cold wide roots, and map reuse after a large traversal.
func BenchmarkHashNodeTraversalWorkloads(b *testing.B) {
	leaf := &yaml.Node{Tag: "!!str", Value: "property", Line: 13, Column: 5}
	small := &yaml.Node{Tag: "!!map", Content: []*yaml.Node{{Value: "type"}, {Value: "string"}}}
	narrow := hashNodeFastBenchmarkChain(16384)
	wide := &yaml.Node{Tag: "!!seq", Content: make([]*yaml.Node, 16384)}
	for i := range wide.Content {
		wide.Content[i] = &yaml.Node{Value: strconv.Itoa(i), Line: i}
	}
	for _, workload := range []struct {
		name string
		root *yaml.Node
	}{{"leaf", leaf}, {"small_composite", small}, {"large_narrow_uncached_repeat", narrow}} {
		b.Run(workload.name, func(b *testing.B) {
			ClearHashCache()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				hashNodeFastBenchmarkSink = HashNode(workload.root)
			}
			b.StopTimer()
			b.ReportMetric(1, "hashes/op")
		})
	}
	for _, workload := range []struct {
		name  string
		small *yaml.Node
	}{{"leaf_after_large_warmup", leaf}, {"composite_after_large_warmup", small}} {
		b.Run(workload.name, func(b *testing.B) {
			ClearHashCache()
			HashNode(narrow)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				hashNodeFastBenchmarkSink = HashNode(workload.small)
			}
			b.StopTimer()
			b.ReportMetric(1, "hashes/op")
		})
	}
	b.Run("large_wide_cold_with_cache_clear", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ClearHashCache()
			hashNodeFastBenchmarkSink = HashNode(wide)
		}
		b.StopTimer()
		ClearHashCache()
		b.ReportMetric(1, "hashes/op")
	})
	for _, workload := range []struct {
		name  string
		small *yaml.Node
	}{{"large_then_128_leaves", leaf}, {"large_then_128_composites", small}} {
		b.Run(workload.name, func(b *testing.B) {
			ClearHashCache()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				hashNodeFastBenchmarkSink = HashNode(narrow)
				for j := 0; j < 128; j++ {
					hashNodeFastBenchmarkSink = HashNode(workload.small)
				}
			}
			b.StopTimer()
			b.ReportMetric(129, "hashes/op")
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
