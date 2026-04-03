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

func benchmarkInfoRootNode(b *testing.B) *yaml.Node {
	b.Helper()

	yml := `title: Pizza API
summary: pizza summary
description: pizza description
termsOfService: https://example.com/tos
contact:
  name: Pizza Team
  url: https://example.com/contact
  email: pizza@example.com
license:
  name: MIT
  url: https://example.com/license
version: 1.0.0
x-info:
  hot: true`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yml), &root); err != nil {
		b.Fatalf("failed to unmarshal benchmark info: %v", err)
	}
	if len(root.Content) == 0 || root.Content[0] == nil {
		b.Fatal("failed to unmarshal benchmark info: empty root")
	}
	return root.Content[0]
}

func benchmarkContactRootNode(b *testing.B) *yaml.Node {
	b.Helper()

	yml := `name: Pizza Team
url: https://example.com/contact
email: pizza@example.com
x-contact:
  warm: true`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yml), &root); err != nil {
		b.Fatalf("failed to unmarshal benchmark contact: %v", err)
	}
	if len(root.Content) == 0 || root.Content[0] == nil {
		b.Fatal("failed to unmarshal benchmark contact: empty root")
	}
	return root.Content[0]
}

func benchmarkLicenseRootNode(b *testing.B) *yaml.Node {
	b.Helper()

	yml := `name: Apache-2.0
url: https://example.com/license
x-license:
  approved: true`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yml), &root); err != nil {
		b.Fatalf("failed to unmarshal benchmark license: %v", err)
	}
	if len(root.Content) == 0 || root.Content[0] == nil {
		b.Fatal("failed to unmarshal benchmark license: empty root")
	}
	return root.Content[0]
}

func benchmarkExternalDocRootNode(b *testing.B) *yaml.Node {
	b.Helper()

	yml := `description: more docs
url: https://example.com/docs
x-docs:
  bright: true`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yml), &root); err != nil {
		b.Fatalf("failed to unmarshal benchmark external doc: %v", err)
	}
	if len(root.Content) == 0 || root.Content[0] == nil {
		b.Fatal("failed to unmarshal benchmark external doc: empty root")
	}
	return root.Content[0]
}

func benchmarkXMLRootNode(b *testing.B) *yaml.Node {
	b.Helper()

	yml := `name: item
namespace: https://example.com/ns
prefix: ex
attribute: false
nodeType: element
wrapped: true
x-xml:
  rich: true`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yml), &root); err != nil {
		b.Fatalf("failed to unmarshal benchmark xml: %v", err)
	}
	if len(root.Content) == 0 || root.Content[0] == nil {
		b.Fatal("failed to unmarshal benchmark xml: empty root")
	}
	return root.Content[0]
}

func benchmarkSecurityRequirementRootNode(b *testing.B) *yaml.Node {
	b.Helper()

	yml := `oauth:
  - read
  - write
apiKey:
  - admin`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yml), &root); err != nil {
		b.Fatalf("failed to unmarshal benchmark security requirement: %v", err)
	}
	if len(root.Content) == 0 || root.Content[0] == nil {
		b.Fatal("failed to unmarshal benchmark security requirement: empty root")
	}
	return root.Content[0]
}

func benchmarkTagRootNode(b *testing.B) *yaml.Node {
	b.Helper()

	yml := `name: partner
summary: Partner API
description: Operations available to the partners network
parent: external
kind: audience
externalDocs:
  url: https://example.com/docs
  description: more docs
x-tag:
  warm: true`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yml), &root); err != nil {
		b.Fatalf("failed to unmarshal benchmark tag: %v", err)
	}
	if len(root.Content) == 0 || root.Content[0] == nil {
		b.Fatal("failed to unmarshal benchmark tag: empty root")
	}
	return root.Content[0]
}

func BenchmarkInfo_Build(b *testing.B) {
	rootNode := benchmarkInfoRootNode(b)
	idx := index.NewSpecIndex(rootNode)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var info Info
		if err := low.BuildModel(rootNode, &info); err != nil {
			b.Fatalf("build model failed: %v", err)
		}
		if err := info.Build(ctx, nil, rootNode, idx); err != nil {
			b.Fatalf("info build failed: %v", err)
		}
	}
}

func BenchmarkContact_Build(b *testing.B) {
	rootNode := benchmarkContactRootNode(b)
	idx := index.NewSpecIndex(rootNode)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var contact Contact
		if err := low.BuildModel(rootNode, &contact); err != nil {
			b.Fatalf("build model failed: %v", err)
		}
		if err := contact.Build(ctx, nil, rootNode, idx); err != nil {
			b.Fatalf("contact build failed: %v", err)
		}
	}
}

func BenchmarkLicense_Build(b *testing.B) {
	rootNode := benchmarkLicenseRootNode(b)
	idx := index.NewSpecIndex(rootNode)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var license License
		if err := low.BuildModel(rootNode, &license); err != nil {
			b.Fatalf("build model failed: %v", err)
		}
		if err := license.Build(ctx, nil, rootNode, idx); err != nil {
			b.Fatalf("license build failed: %v", err)
		}
	}
}

func BenchmarkExternalDoc_Build(b *testing.B) {
	rootNode := benchmarkExternalDocRootNode(b)
	idx := index.NewSpecIndex(rootNode)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var ex ExternalDoc
		if err := low.BuildModel(rootNode, &ex); err != nil {
			b.Fatalf("build model failed: %v", err)
		}
		if err := ex.Build(ctx, nil, rootNode, idx); err != nil {
			b.Fatalf("external doc build failed: %v", err)
		}
	}
}

func BenchmarkXML_Build(b *testing.B) {
	rootNode := benchmarkXMLRootNode(b)
	idx := index.NewSpecIndex(rootNode)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var x XML
		if err := low.BuildModel(rootNode, &x); err != nil {
			b.Fatalf("build model failed: %v", err)
		}
		if err := x.Build(rootNode, idx); err != nil {
			b.Fatalf("xml build failed: %v", err)
		}
	}
}

func BenchmarkSecurityRequirement_Build(b *testing.B) {
	rootNode := benchmarkSecurityRequirementRootNode(b)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var req SecurityRequirement
		if err := req.Build(ctx, nil, rootNode, nil); err != nil {
			b.Fatalf("security requirement build failed: %v", err)
		}
	}
}

func BenchmarkTag_Build(b *testing.B) {
	rootNode := benchmarkTagRootNode(b)
	idx := index.NewSpecIndex(rootNode)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var tag Tag
		if err := low.BuildModel(rootNode, &tag); err != nil {
			b.Fatalf("build model failed: %v", err)
		}
		if err := tag.Build(ctx, nil, rootNode, idx); err != nil {
			b.Fatalf("tag build failed: %v", err)
		}
	}
}
