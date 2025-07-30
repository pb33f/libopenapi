// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
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