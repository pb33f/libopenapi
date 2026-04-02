// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package low

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"go.yaml.in/yaml/v4"
)

const benchmarkLookupSpec = `openapi: 3.1.0
info:
  title: benchmark
  version: 1.0.0
paths:
  /burger/time:
    get:
      responses:
        '200':
          description: delicious
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Thing'
components:
  schemas:
    Thing:
      $id: "https://example.com/schemas/thing"
      type: object
      properties:
        name:
          type: string
    Scoped:
      $id: "https://example.com/schemas/base"
      $defs:
        child:
          type: string
`

func benchmarkLookupIndex(b *testing.B) *index.SpecIndex {
	b.Helper()

	var rootNode yaml.Node
	if err := yaml.Unmarshal([]byte(benchmarkLookupSpec), &rootNode); err != nil {
		b.Fatalf("failed to unmarshal benchmark lookup spec: %v", err)
	}
	config := index.CreateClosedAPIIndexConfig()
	config.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	return index.NewSpecIndexWithConfig(&rootNode, config)
}

func BenchmarkLocateRefNodeWithContext_MappedReference(b *testing.B) {
	idx := benchmarkLookupIndex(b)
	refNode := utils.CreateRefNode("#/components/schemas/Thing")
	ctx := context.Background()

	found, _, err, _ := LocateRefNodeWithContext(ctx, refNode, idx)
	if err != nil || found == nil {
		b.Fatalf("benchmark setup failed: found=%v err=%v", found != nil, err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		found, _, err, _ = LocateRefNodeWithContext(ctx, refNode, idx)
		if err != nil || found == nil {
			b.Fatalf("mapped lookup failed: found=%v err=%v", found != nil, err)
		}
	}
}

func BenchmarkLocateRefNodeWithContext_LocalPathFallback(b *testing.B) {
	idx := benchmarkLookupIndex(b)
	refNode := utils.CreateRefNode("#/paths/~1burger~1time")
	ctx := context.Background()

	found, _, err, _ := LocateRefNodeWithContext(ctx, refNode, idx)
	if err != nil || found == nil {
		b.Fatalf("benchmark setup failed: found=%v err=%v", found != nil, err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		found, _, err, _ = LocateRefNodeWithContext(ctx, refNode, idx)
		if err != nil || found == nil {
			b.Fatalf("path fallback lookup failed: found=%v err=%v", found != nil, err)
		}
	}
}

func BenchmarkLocateRefNodeWithContext_SchemaIDScope(b *testing.B) {
	idx := benchmarkLookupIndex(b)
	refNode := utils.CreateRefNode("#/$defs/child")
	ctx := index.WithSchemaIdScope(context.Background(), index.NewSchemaIdScope("https://example.com/schemas/base"))

	found, _, err, _ := LocateRefNodeWithContext(ctx, refNode, idx)
	if err != nil || found == nil {
		b.Fatalf("benchmark setup failed: found=%v err=%v", found != nil, err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		found, _, err, _ = LocateRefNodeWithContext(ctx, refNode, idx)
		if err != nil || found == nil {
			b.Fatalf("schema id lookup failed: found=%v err=%v", found != nil, err)
		}
	}
}
