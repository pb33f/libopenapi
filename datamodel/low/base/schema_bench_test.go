// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"go.yaml.in/yaml/v4"
)

func benchmarkSchemaRootNode(b *testing.B) *yaml.Node {
	b.Helper()

	var rootNode yaml.Node
	if err := yaml.Unmarshal([]byte(test_get_schema_blob()), &rootNode); err != nil {
		b.Fatalf("failed to unmarshal benchmark schema: %v", err)
	}
	if len(rootNode.Content) == 0 || rootNode.Content[0] == nil {
		b.Fatal("failed to unmarshal benchmark schema: empty root")
	}
	return rootNode.Content[0]
}

func benchmarkBuiltSchema(b *testing.B) *Schema {
	b.Helper()

	rootNode := benchmarkSchemaRootNode(b)
	schema := new(Schema)
	if err := low.BuildModel(rootNode, schema); err != nil {
		b.Fatalf("failed to build low-level schema model: %v", err)
	}
	if err := schema.Build(context.Background(), rootNode, nil); err != nil {
		b.Fatalf("failed to build schema: %v", err)
	}
	return schema
}

func BenchmarkSchema_Build(b *testing.B) {
	rootNode := benchmarkSchemaRootNode(b)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var schema Schema
		if err := low.BuildModel(rootNode, &schema); err != nil {
			b.Fatalf("build model failed: %v", err)
		}
		if err := schema.Build(ctx, rootNode, nil); err != nil {
			b.Fatalf("schema build failed: %v", err)
		}
	}
}

func BenchmarkSchema_QuickHash(b *testing.B) {
	schema := benchmarkBuiltSchema(b)

	ClearSchemaQuickHashMap()
	if hash := schema.QuickHash(); hash == 0 {
		b.Fatal("benchmark setup failed: quick hash returned zero")
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ClearSchemaQuickHashMap()
		if hash := schema.QuickHash(); hash == 0 {
			b.Fatal("quick hash returned zero")
		}
	}
}

func BenchmarkSchema_QuickHash_Cached(b *testing.B) {
	schema := benchmarkBuiltSchema(b)
	if hash := schema.QuickHash(); hash == 0 {
		b.Fatal("benchmark setup failed: quick hash returned zero")
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if hash := schema.QuickHash(); hash == 0 {
			b.Fatal("quick hash returned zero")
		}
	}
}

func BenchmarkSchema_HashSingle(b *testing.B) {
	schema := benchmarkBuiltSchema(b)
	if hash := schema.Hash(); hash == 0 {
		b.Fatal("benchmark setup failed: hash returned zero")
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if hash := schema.Hash(); hash == 0 {
			b.Fatal("hash returned zero")
		}
	}
}
