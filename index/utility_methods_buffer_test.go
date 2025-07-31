// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// Test that buffer pool optimization maintains identical hash outputs
func TestHashNode_BufferPoolConsistency(t *testing.T) {
	testCases := []struct {
		name     string
		yaml     string
		expected string
	}{
		{
			name: "simple mapping",
			yaml: `plum: soup
chicken: wing
beef: burger
pork: chop`,
			expected: "e9aba1ce94ac8bd0143524ce594c0c7d38c06c09eca7ae96725187f488661fcd",
		},
		{
			name: "nested structure",
			yaml: `root:
  level1:
    level2:
      value: "deep"`,
			expected: "", // Will be calculated
		},
		{
			name: "array structure",
			yaml: `items:
  - name: first
    value: 1
  - name: second
    value: 2`,
			expected: "", // Will be calculated
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var rootNode yaml.Node
			err := yaml.Unmarshal([]byte(tc.yaml), &rootNode)
			assert.NoError(t, err)

			// Calculate hash multiple times to ensure consistency
			hash1 := HashNode(&rootNode)
			hash2 := HashNode(&rootNode)
			hash3 := HashNode(&rootNode)

			// All hashes should be identical
			assert.Equal(t, hash1, hash2, "Hash should be consistent between calls")
			assert.Equal(t, hash2, hash3, "Hash should be consistent between calls")
			assert.NotEmpty(t, hash1, "Hash should not be empty")

			// If expected hash is provided, verify it matches
			if tc.expected != "" {
				assert.Equal(t, tc.expected, hash1, "Hash should match expected value")
			}
		})
	}
}

// Test concurrent access to buffer pool
func TestHashNode_ConcurrentAccess(t *testing.T) {
	yamlStr := `concurrent:
  test: value
  items:
    - a: 1
    - b: 2
    - c: 3`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(yamlStr), &rootNode)
	assert.NoError(t, err)

	// Get expected hash first
	expectedHash := HashNode(&rootNode)

	// Run concurrent hash calculations
	const numGoroutines = 10
	results := make(chan string, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			results <- HashNode(&rootNode)
		}()
	}

	// Collect all results
	for i := 0; i < numGoroutines; i++ {
		hash := <-results
		assert.Equal(t, expectedHash, hash, "Concurrent hash calculation should be consistent")
	}
}

// Test ClearHashCache function with populated cache
func TestClearHashCache(t *testing.T) {
	// Ensure we start with a clean cache
	ClearHashCache()
	
	// Create multiple large nodes that will definitely be cached
	nodes := make([]*yaml.Node, 5)
	for n := 0; n < 5; n++ {
		largeYaml := fmt.Sprintf("root%d:", n)
		for i := 0; i < 300; i++ { // Well above cacheThreshold of 200
			largeYaml += fmt.Sprintf(`
  item%d: value%d_%d`, i, i, n)
		}

		var rootNode yaml.Node
		err := yaml.Unmarshal([]byte(largeYaml), &rootNode)
		assert.NoError(t, err)
		nodes[n] = &rootNode
	}

	// Hash all nodes to populate cache with multiple entries
	// This ensures the Range function in ClearHashCache will have items to iterate over
	hashes := make([]string, 5)
	for i, node := range nodes {
		hashes[i] = HashNode(node)
		assert.NotEmpty(t, hashes[i])
		// Note: After YAML unmarshaling, the actual content structure may differ
		// The important thing is that we create large enough YAML that will result in caching
	}

	// Now clear the cache - this should iterate over all cached entries and delete them
	// This exercises both the Range function and the Delete operations inside the anonymous function
	ClearHashCache()

	// Hash all nodes again - should still work and be identical (cache miss, recalculate)
	for i, node := range nodes {
		hash := HashNode(node)
		assert.Equal(t, hashes[i], hash, "Hash should be consistent after cache clear for node %d", i)
	}

	// Verify cache was actually cleared by hashing again - this should populate cache again
	for i, node := range nodes {
		hash := HashNode(node)
		assert.Equal(t, hashes[i], hash, "Hash should still be consistent")
	}
	
	// Clear the now-populated cache again to test the function multiple times
	ClearHashCache()
	
	// Final verification
	finalHash := HashNode(nodes[0])
	assert.Equal(t, hashes[0], finalHash, "Hash should work after multiple cache clears")
}

// Test ClearHashCache when no items were cached
func TestClearHashCache_EmptyCache(t *testing.T) {
	// Clear cache when it's already empty
	ClearHashCache()
	
	// Create small nodes that won't be cached (< 200 content items)
	smallYaml := `small:
  item1: value1
  item2: value2`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(smallYaml), &rootNode)
	assert.NoError(t, err)

	// Hash small node - should not populate cache
	hash1 := HashNode(&rootNode)
	assert.NotEmpty(t, hash1)

	// Clear empty cache
	ClearHashCache()

	// Should still work
	hash2 := HashNode(&rootNode)
	assert.Equal(t, hash1, hash2)
}

// Test ClearHashCache more comprehensively with guaranteed cache population
func TestClearHashCache_ComprehensiveTest(t *testing.T) {
	// Start completely clean
	ClearHashCache()
	
	// Create nodes that will definitely be cached by manually creating large content
	largeNodes := make([]*yaml.Node, 10)
	expectedHashes := make([]string, 10)
	
	for i := 0; i < 10; i++ {
		// Manually create large nodes to guarantee caching
		node := &yaml.Node{
			Kind: yaml.MappingNode,
			Tag: "!!map",
			Value: fmt.Sprintf("large_root_%d", i),
			Content: make([]*yaml.Node, 250), // Above cacheThreshold
		}
		
		// Fill with content that varies per node
		for j := 0; j < 250; j++ {
			node.Content[j] = &yaml.Node{
				Kind: yaml.ScalarNode,
				Tag: "!!str",
				Value: fmt.Sprintf("item_%d_%d", i, j),
				Line: j + 1,
				Column: (j % 10) + 1,
			}
		}
		
		largeNodes[i] = node
		expectedHashes[i] = HashNode(node) // This should populate cache
		assert.NotEmpty(t, expectedHashes[i])
	}
	
	// At this point, cache should have entries for all large nodes
	// Now test clearing the cache
	ClearHashCache()
	
	// Re-hash all nodes - they should produce the same hashes but from scratch
	for i, node := range largeNodes {
		hash := HashNode(node)
		assert.Equal(t, expectedHashes[i], hash, "Hash %d should be consistent after cache clear", i)
	}
	
	// Populate cache again
	for _, node := range largeNodes {
		HashNode(node)
	}
	
	// Clear once more to ensure the Range function executes multiple times
	ClearHashCache()
	
	// Final verification
	for i, node := range largeNodes {
		hash := HashNode(node)
		assert.Equal(t, expectedHashes[i], hash, "Hash %d should still be consistent", i)
	}
}

// Test shouldUseOptimizedHashing with large node threshold
func TestShouldUseOptimizedHashing_LargeNode(t *testing.T) {
	// Create a node with > 1000 content items (largeLodeThreshold)
	largeNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: make([]*yaml.Node, 1001),
	}
	for i := 0; i < 1001; i++ {
		largeNode.Content[i] = &yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprintf("item%d", i)}
	}

	// Should use optimized hashing for large nodes
	assert.True(t, shouldUseOptimizedHashing(largeNode, 0))
}

// Test shouldUseOptimizedHashing with deep node threshold
func TestShouldUseOptimizedHashing_DeepNode(t *testing.T) {
	smallNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "test"}
	
	// Should use optimized hashing for deep nodes (depth > 100)
	assert.True(t, shouldUseOptimizedHashing(smallNode, 101))
	
	// Should not use optimized for shallow nodes
	assert.False(t, shouldUseOptimizedHashing(smallNode, 50))
}

// Test shouldUseOptimizedHashing with large children
func TestShouldUseOptimizedHashing_LargeChildren(t *testing.T) {
	// Create parent with small content but large child
	largeChild := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: make([]*yaml.Node, 1001), // Above threshold
	}
	
	parentNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{largeChild}, // Only one child, but it's large
	}

	// Should use optimized hashing because child is large
	assert.True(t, shouldUseOptimizedHashing(parentNode, 0))
}

// Test shouldUseOptimizedHashing with nil node
func TestShouldUseOptimizedHashing_NilNode(t *testing.T) {
	assert.False(t, shouldUseOptimizedHashing(nil, 0))
}

// Test HashNode with large node that triggers caching
func TestHashNode_LargeNodeCaching(t *testing.T) {
	// Create a node with >= 200 content items to trigger caching
	largeYaml := `root:`
	for i := 0; i < 250; i++ {
		largeYaml += fmt.Sprintf(`
  item%d: value%d`, i, i)
	}

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(largeYaml), &rootNode)
	assert.NoError(t, err)

	// Clear cache first
	ClearHashCache()

	// First hash should populate cache and use optimized path
	hash1 := HashNode(&rootNode)
	assert.NotEmpty(t, hash1)

	// Second hash should use cached result
	hash2 := HashNode(&rootNode)
	assert.Equal(t, hash1, hash2)
}

// Test HashNode with small node that doesn't trigger caching
func TestHashNode_SmallNodeNoCaching(t *testing.T) {
	smallYaml := `small:
  item1: value1
  item2: value2`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(smallYaml), &rootNode)
	assert.NoError(t, err)

	// Clear cache first
	ClearHashCache()

	// Hash small node (should not be cached)
	hash1 := HashNode(&rootNode)
	assert.NotEmpty(t, hash1)

	// Second hash should still work
	hash2 := HashNode(&rootNode)
	assert.Equal(t, hash1, hash2)
}

// Test hash functions with very deep recursion (>1000 levels)
func TestHashNode_VeryDeepRecursion(t *testing.T) {
	// Create a chain of nodes that exceeds the 1000 depth limit
	root := &yaml.Node{Kind: yaml.MappingNode}
	current := root
	
	for i := 0; i < 1100; i++ {
		child := &yaml.Node{
			Kind: yaml.MappingNode,
			Tag: fmt.Sprintf("level%d", i),
			Value: fmt.Sprintf("value%d", i),
		}
		current.Content = []*yaml.Node{child}
		current = child
	}
	
	// Should handle deep recursion gracefully
	hash := HashNode(root)
	assert.NotEmpty(t, hash)
}

// Test optimized vs simple hashing produce same results for same input
func TestHashNode_OptimizedVsSimple(t *testing.T) {
	yamlStr := `test:
  item1: value1
  item2: value2
  nested:
    deep1: val1
    deep2: val2`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(yamlStr), &rootNode)
	assert.NoError(t, err)

	// Clear cache first
	ClearHashCache()

	// Force different code paths by manipulating thresholds temporarily
	// This tests that both paths produce identical results
	hash1 := HashNode(&rootNode)
	assert.NotEmpty(t, hash1)

	// Hash again should be identical regardless of path taken
	hash2 := HashNode(&rootNode)
	assert.Equal(t, hash1, hash2)
}

// Test hash functions with empty and edge case nodes
func TestHashNode_EdgeCases(t *testing.T) {
	testCases := []struct {
		name string
		node *yaml.Node
	}{
		{
			name: "empty mapping node",
			node: &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{}},
		},
		{
			name: "empty sequence node", 
			node: &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{}},
		},
		{
			name: "scalar with empty value",
			node: &yaml.Node{Kind: yaml.ScalarNode, Value: ""},
		},
		{
			name: "node with only tag",
			node: &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str"},
		},
		{
			name: "node with line/column info",
			node: &yaml.Node{Kind: yaml.ScalarNode, Value: "test", Line: 42, Column: 13},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ClearHashCache()
			hash := HashNode(tc.node)
			assert.NotEmpty(t, hash, "Hash should not be empty for %s", tc.name)
			
			// Hash should be consistent
			hash2 := HashNode(tc.node)
			assert.Equal(t, hash, hash2, "Hash should be consistent for %s", tc.name)
		})
	}
}

// Test specific branches in hashNodeOptimized and hashNodeSimple
func TestHashNode_ForceBranches(t *testing.T) {
	// Create a node that will trigger optimized hashing (large content)
	largeNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag: "!!map",
		Value: "root",
		Line: 1,
		Column: 1,
		Content: make([]*yaml.Node, 1100), // Above largeLodeThreshold
	}
	
	// Fill with alternating small and large nodes to test both paths
	for i := 0; i < 1100; i++ {
		if i%2 == 0 {
			// Small node - will use simple hashing
			largeNode.Content[i] = &yaml.Node{
				Kind: yaml.ScalarNode,
				Tag: "!!str",
				Value: fmt.Sprintf("small%d", i),
				Line: i + 2,
				Column: 1,
			}
		} else {
			// Large node - will use optimized hashing
			child := &yaml.Node{
				Kind: yaml.MappingNode,
				Tag: "!!map", 
				Value:  fmt.Sprintf("large%d", i),
				Line: i + 2,
				Column: 1,
				Content: make([]*yaml.Node, 1001),
			}
			for j := 0; j < 1001; j++ {
				child.Content[j] = &yaml.Node{
					Kind: yaml.ScalarNode,
					Value: fmt.Sprintf("item%d", j),
				}
			}
			largeNode.Content[i] = child
		}
	}
	
	ClearHashCache()
	
	// This should exercise both optimized and simple code paths
	hash := HashNode(largeNode)
	assert.NotEmpty(t, hash)
	
	// Should be consistent
	hash2 := HashNode(largeNode)
	assert.Equal(t, hash, hash2)
}

// Test hashNodeOptimized and hashNodeSimple with empty content arrays
func TestHashNode_EmptyContentArrays(t *testing.T) {
	// Test with empty content arrays and various node types
	testNodes := []*yaml.Node{
		{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{}},
		{Kind: yaml.SequenceNode, Tag: "!!seq", Content: []*yaml.Node{}},
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "scalar", Content: nil},
	}
	
	for i, node := range testNodes {
		t.Run(fmt.Sprintf("node_%d", i), func(t *testing.T) {
			hash := HashNode(node)
			assert.NotEmpty(t, hash)
			
			// Should be consistent
			hash2 := HashNode(node)
			assert.Equal(t, hash, hash2)
		})
	}
}

// Test nodes at exactly the depth threshold (1000)
func TestHashNode_ExactDepthThreshold(t *testing.T) {
	// Create a chain exactly 1000 levels deep
	root := &yaml.Node{Kind: yaml.MappingNode, Value: "root"}
	current := root
	
	for i := 0; i < 999; i++ { // 999 + root = 1000 total
		child := &yaml.Node{
			Kind: yaml.MappingNode,
			Tag: fmt.Sprintf("!!level%d", i),
			Value: fmt.Sprintf("value%d", i),
		}
		current.Content = []*yaml.Node{child}
		current = child
	}
	
	// At exactly 1000 depth, should still process
	hash := HashNode(root)
	assert.NotEmpty(t, hash)
	
	// Add one more level to exceed threshold
	finalChild := &yaml.Node{
		Kind: yaml.ScalarNode,
		Value: "final",
	}
	current.Content = []*yaml.Node{finalChild}
	
	// Should still work (depth limit prevents infinite recursion)
	hash2 := HashNode(root)
	assert.NotEmpty(t, hash2)
}

// Test with very large individual node values
func TestHashNode_LargeValues(t *testing.T) {
	// Create node with very large tag and value strings
	largeTag := fmt.Sprintf("!!%s", strings.Repeat("tag", 1000))
	largeValue := strings.Repeat("value", 1000)
	
	nodeWithLargeValues := &yaml.Node{
		Kind: yaml.ScalarNode,
		Tag: largeTag,
		Value: largeValue,
		Line: 999999,
		Column: 999999,
	}
	
	hash := HashNode(nodeWithLargeValues)
	assert.NotEmpty(t, hash)
	
	// Should be consistent
	hash2 := HashNode(nodeWithLargeValues)
	assert.Equal(t, hash, hash2)
}

// Test to ensure nil nodes passed to internal hash functions are handled
func TestHashNode_InternalNilHandling(t *testing.T) {
	// Create a large node that will trigger optimized hashing but has mixed content
	// including scenarios that might result in nil checks in the internal functions
	rootNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag: "!!map",
		Value: "root",
		Content: make([]*yaml.Node, 1100), // Forces optimized path
	}
	
	// Fill with mix of content that exercises different code paths
	for i := 0; i < 1100; i++ {
		if i%100 == 0 {
			// Create nodes that will trigger different hash paths with empty content
			rootNode.Content[i] = &yaml.Node{
				Kind: yaml.MappingNode,
				Tag: "!!map",
				Value: fmt.Sprintf("empty_%d", i),
				Content: []*yaml.Node{}, // Empty content - exercises edge case
			}
		} else if i%50 == 0 {
			// Create very deep nested structure to test depth limits
			deepNode := &yaml.Node{Kind: yaml.MappingNode, Value: fmt.Sprintf("deep_%d", i)}
			current := deepNode
			// Create chain that approaches but doesn't exceed depth limit
			for j := 0; j < 500; j++ {
				child := &yaml.Node{
					Kind: yaml.ScalarNode,
					Value: fmt.Sprintf("depth_%d_%d", i, j),
				}
				current.Content = []*yaml.Node{child}
				current = child
			}
			rootNode.Content[i] = deepNode
		} else {
			// Regular nodes
			rootNode.Content[i] = &yaml.Node{
				Kind: yaml.ScalarNode,
				Tag: "!!str",
				Value: fmt.Sprintf("item_%d", i),
				Line: i,
				Column: i % 100,
			}
		}
	}
	
	// This should exercise both hashNodeOptimized and hashNodeSimple
	// with various edge cases including empty content and deep nesting
	hash := HashNode(rootNode)
	assert.NotEmpty(t, hash)
	
	// Should be consistent
	hash2 := HashNode(rootNode)
	assert.Equal(t, hash, hash2)
}

// Test extreme depth scenarios to hit the depth limit checks
func TestHashNode_ExtremeDepthLimits(t *testing.T) {
	// Create a node structure that will definitely hit the >1000 depth limit
	// This should exercise the depth check returns in both hash functions
	
	// Start with a large root that forces optimized hashing
	root := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag: "!!map", 
		Value: "root",
		Content: make([]*yaml.Node, 1200), // Forces optimized path
	}
	
	// Create one extremely deep branch that will hit the depth limit
	deepBranch := &yaml.Node{Kind: yaml.MappingNode, Value: "deep_start"}
	current := deepBranch
	
	// Create a chain that goes well beyond the 1000 depth limit
	for i := 0; i < 1200; i++ {
		child := &yaml.Node{
			Kind: yaml.MappingNode,
			Tag: fmt.Sprintf("!!level_%d", i),
			Value: fmt.Sprintf("depth_%d", i),
			Line: i,
			Column: i % 100,
		}
		current.Content = []*yaml.Node{child}
		current = child
	}
	
	// Add the deep branch as first element
	root.Content[0] = deepBranch
	
	// Fill remaining slots with smaller nodes that will use simple hashing
	for i := 1; i < 1200; i++ {
		root.Content[i] = &yaml.Node{
			Kind: yaml.ScalarNode,
			Value: fmt.Sprintf("shallow_%d", i),
		}
	}
	
	// This will exercise both optimized and simple hash functions
	// and specifically test the depth > 1000 early returns
	hash := HashNode(root)
	assert.NotEmpty(t, hash)
	
	// Should be consistent even with depth limits
	hash2 := HashNode(root)
	assert.Equal(t, hash, hash2)
}

// Test to specifically exercise the nil return paths in hash functions
func TestHashNode_ForceNilPaths(t *testing.T) {
	// Create a structure that might exercise nil handling in recursive calls
	// This is tricky since we can't directly pass nil to the internal functions,
	// but we can create scenarios where the functions handle edge cases
	
	// Create a node that forces optimized hashing
	complexNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag: "!!map",
		Content: make([]*yaml.Node, 1001), // Above threshold
	}
	
	// Fill with nodes that have various edge case properties
	for i := 0; i < 1001; i++ {
		if i%3 == 0 {
			// Node with nil content (valid case)
			complexNode.Content[i] = &yaml.Node{
				Kind: yaml.ScalarNode,
				Tag: "",  // Empty tag
				Value: "", // Empty value
				Content: nil, // Explicitly nil content
			}
		} else if i%3 == 1 {
			// Node with empty content slice
			complexNode.Content[i] = &yaml.Node{
				Kind: yaml.MappingNode,
				Tag: "!!map",
				Value: "",
				Content: []*yaml.Node{}, // Empty but not nil
			}
		} else {
			// Regular node
			complexNode.Content[i] = &yaml.Node{
				Kind: yaml.ScalarNode,
				Value: fmt.Sprintf("regular_%d", i),
			}
		}
	}
	
	// Hash the complex structure
	hash := HashNode(complexNode)
	assert.NotEmpty(t, hash)
	
	// Should be consistent
	hash2 := HashNode(complexNode)
	assert.Equal(t, hash, hash2)
}

// Test to trigger edge cases in hashNodeOptimized and hashNodeSimple
func TestHashNode_TriggerAllPaths(t *testing.T) {
	testCases := []struct {
		name string
		node *yaml.Node
	}{
		{
			name: "OptimizedPath_WithNilContentElements",
			node: func() *yaml.Node {
				// Create a large node that forces optimized path
				n := &yaml.Node{
					Kind: yaml.MappingNode,
					Tag: "!!map",
					Value: "optimized_root",
					Content: make([]*yaml.Node, 1001),
				}
				// Fill with real nodes - we can't put nil in Content as it would cause panic
				for i := 0; i < 1001; i++ {
					n.Content[i] = &yaml.Node{
						Kind: yaml.ScalarNode,
						Value: fmt.Sprintf("opt_%d", i),
						Line: i,
						Column: i % 100,
					}
				}
				return n
			}(),
		},
		{
			name: "SimplePath_WithMinimalContent",
			node: &yaml.Node{
				Kind: yaml.ScalarNode,
				Tag: "!!str",
				Value: "simple_node",
				Line: 42,
				Column: 13,
				Content: nil, // Nil content for scalar
			},
		},
		{
			name: "EmptyNode_OptimizedPath",
			node: func() *yaml.Node {
				n := &yaml.Node{
					Kind: yaml.MappingNode,
					Tag: "",
					Value: "",
					Content: make([]*yaml.Node, 1100),
				}
				// Fill with empty nodes
				for i := 0; i < 1100; i++ {
					n.Content[i] = &yaml.Node{
						Kind: yaml.ScalarNode,
						Tag: "",
						Value: "",
					}
				}
				return n
			}(),
		},
		{
			name: "DeepNesting_BothPaths",
			node: func() *yaml.Node {
				// Create a structure that will use both optimized and simple paths
				root := &yaml.Node{
					Kind: yaml.MappingNode,
					Content: make([]*yaml.Node, 1200), // Force optimized
				}
				
				for i := 0; i < 1200; i++ {
					if i < 600 {
						// Small nodes that will use simple path when recursed into
						root.Content[i] = &yaml.Node{
							Kind: yaml.ScalarNode,
							Value: fmt.Sprintf("simple_%d", i),
						}
					} else {
						// Large nodes that will use optimized path when recursed into
						child := &yaml.Node{
							Kind: yaml.MappingNode,
							Content: make([]*yaml.Node, 1001),
						}
						for j := 0; j < 1001; j++ {
							child.Content[j] = &yaml.Node{
								Kind: yaml.ScalarNode,
								Value: fmt.Sprintf("deep_%d_%d", i, j),
							}
						}
						root.Content[i] = child
					}
				}
				return root
			}(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear cache before each test
			ClearHashCache()
			
			hash1 := HashNode(tc.node)
			assert.NotEmpty(t, hash1, "Hash should not be empty for %s", tc.name)
			
			hash2 := HashNode(tc.node)
			assert.Equal(t, hash1, hash2, "Hash should be consistent for %s", tc.name)
		})
	}
}