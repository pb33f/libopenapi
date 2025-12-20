// Copyright 2024 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"os"
	"testing"

	"go.yaml.in/yaml/v4"
)

// TestDeterminism_ConsistentResults verifies that reference extraction produces
// identical results across multiple runs. This is the regression test for issue #441.
//
// The test runs the full indexing process multiple times on a large spec and verifies:
// 1. The same number of refs are found each time
// 2. The refs are in the exact same order each time
// 3. The same number of errors are reported each time
func TestDeterminism_ConsistentResults(t *testing.T) {
	specPath := "../test_specs/stripe.yaml"
	specBytes, err := os.ReadFile(specPath)
	if err != nil {
		t.Skipf("Could not load spec: %v", err)
	}

	const runs = 10
	var baselineOrder []string
	var baselineErrorCount int

	for i := 0; i < runs; i++ {
		var rootNode yaml.Node
		if err := yaml.Unmarshal(specBytes, &rootNode); err != nil {
			t.Fatal(err)
		}

		config := &SpecIndexConfig{
			AllowRemoteLookup: false,
			AllowFileLookup:   false,
		}

		idx := NewSpecIndexWithConfig(&rootNode, config)
		idx.BuildIndex()

		// Get sequenced refs - this is what must be deterministic
		sequenced := idx.GetMappedReferencesSequenced()
		order := make([]string, len(sequenced))
		for j, ref := range sequenced {
			order[j] = ref.FullDefinition
		}

		errorCount := len(idx.GetReferenceIndexErrors())

		if i == 0 {
			baselineOrder = order
			baselineErrorCount = errorCount
			t.Logf("Baseline: %d refs, %d errors", len(order), errorCount)
		} else {
			// Verify ref count is identical
			if len(order) != len(baselineOrder) {
				t.Fatalf("Run %d: different ref count: got %d, want %d", i, len(order), len(baselineOrder))
			}
			// Verify error count is identical
			if errorCount != baselineErrorCount {
				t.Errorf("Run %d: different error count: got %d, want %d", i, errorCount, baselineErrorCount)
			}
			// Verify order is identical
			for j := range order {
				if order[j] != baselineOrder[j] {
					t.Fatalf("Run %d: different order at index %d: got %s, want %s",
						i, j, order[j], baselineOrder[j])
				}
			}
		}
	}
	t.Logf("All %d runs produced identical results", runs)
}

// BenchmarkIndexing_Determinism benchmarks the indexing process while also
// verifying determinism. This ensures performance optimizations don't break
// the deterministic ordering guarantee.
func BenchmarkIndexing_Determinism(b *testing.B) {
	specPath := "../test_specs/stripe.yaml"
	specBytes, err := os.ReadFile(specPath)
	if err != nil {
		b.Skipf("Could not load spec: %v", err)
	}

	var baselineOrder []string

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var rootNode yaml.Node
		if err := yaml.Unmarshal(specBytes, &rootNode); err != nil {
			b.Fatal(err)
		}

		config := &SpecIndexConfig{
			AllowRemoteLookup: false,
			AllowFileLookup:   false,
		}

		idx := NewSpecIndexWithConfig(&rootNode, config)
		idx.BuildIndex()

		// Get sequenced refs and verify determinism
		sequenced := idx.GetMappedReferencesSequenced()
		order := make([]string, len(sequenced))
		for j, ref := range sequenced {
			order[j] = ref.FullDefinition
		}

		if i == 0 {
			baselineOrder = order
			if len(order) == 0 {
				b.Fatal("No references found")
			}
		} else {
			// Verify order is identical on every iteration
			if len(order) != len(baselineOrder) {
				b.Fatalf("Iteration %d: different ref count: got %d, want %d", i, len(order), len(baselineOrder))
			}
			for j := range order {
				if order[j] != baselineOrder[j] {
					b.Fatalf("Iteration %d: different order at index %d", i, j)
				}
			}
		}
	}
}
