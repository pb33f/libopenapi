// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package utils

import (
	"testing"

	"go.yaml.in/yaml/v4"
)

func benchmarkPetstoreRootNode(b *testing.B) *yaml.Node {
	b.Helper()

	root, err := FindNodes(getPetstore(), "$")
	if err != nil {
		b.Fatalf("failed to load benchmark root node: %v", err)
	}
	if len(root) == 0 || root[0] == nil {
		b.Fatal("failed to load benchmark root node: no root node found")
	}
	return root[0]
}

func BenchmarkFindNodesWithoutDeserializingWithOptions_Default(b *testing.B) {
	ClearJSONPathCache()
	root := benchmarkPetstoreRootNode(b)
	path := "$.info.contact"

	nodes, err := FindNodesWithoutDeserializingWithOptions(root, path, JSONPathLookupOptions{})
	if err != nil || len(nodes) == 0 {
		b.Fatalf("benchmark setup failed: len=%d err=%v", len(nodes), err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		nodes, err = FindNodesWithoutDeserializingWithOptions(root, path, JSONPathLookupOptions{})
		if err != nil || len(nodes) == 0 {
			b.Fatalf("lookup failed: len=%d err=%v", len(nodes), err)
		}
	}
}

func BenchmarkFindNodesWithoutDeserializingWithOptions_LazyDisabled(b *testing.B) {
	ClearJSONPathCache()
	root := benchmarkPetstoreRootNode(b)
	path := "$.info.contact"
	lazyDisabled := false
	options := JSONPathLookupOptions{LazyContextTracking: &lazyDisabled}

	nodes, err := FindNodesWithoutDeserializingWithOptions(root, path, options)
	if err != nil || len(nodes) == 0 {
		b.Fatalf("benchmark setup failed: len=%d err=%v", len(nodes), err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		nodes, err = FindNodesWithoutDeserializingWithOptions(root, path, options)
		if err != nil || len(nodes) == 0 {
			b.Fatalf("lookup failed: len=%d err=%v", len(nodes), err)
		}
	}
}
