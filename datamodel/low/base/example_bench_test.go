// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"go.yaml.in/yaml/v4"
)

func benchmarkExampleRootNode(b *testing.B) *yaml.Node {
	b.Helper()

	yml := `summary: hot
description: cakes
value:
  pizza:
    kind: oven
    toppings:
      - cheese
      - herbs
  yummy:
    yes: pizza
dataValue:
  payload:
    nested:
      flag: true
serializedValue: '{"pizza":true}'
x-cake:
  sweet:
    maybe: yes`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yml), &root); err != nil {
		b.Fatalf("failed to unmarshal benchmark example: %v", err)
	}
	if len(root.Content) == 0 || root.Content[0] == nil {
		b.Fatal("failed to unmarshal benchmark example: empty root")
	}
	return root.Content[0]
}

func BenchmarkExample_Build(b *testing.B) {
	rootNode := benchmarkExampleRootNode(b)
	idx := index.NewSpecIndex(rootNode)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var ex Example
		if err := low.BuildModel(rootNode, &ex); err != nil {
			b.Fatalf("build model failed: %v", err)
		}
		if err := ex.Build(ctx, nil, rootNode, idx); err != nil {
			b.Fatalf("example build failed: %v", err)
		}
	}
}
