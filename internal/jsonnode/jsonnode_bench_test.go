// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package jsonnode

import (
	"os"
	"path/filepath"
	"testing"

	"go.yaml.in/yaml/v4"
)

func benchmarkSpec(b *testing.B) []byte {
	b.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "test_specs", "docusignv3.1.json"))
	if err != nil {
		b.Fatal(err)
	}
	return data
}

func BenchmarkParse_DocuSign(b *testing.B) {
	data := benchmarkSpec(b)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	for b.Loop() {
		if _, ok := Parse(data); !ok {
			b.Fatal("declined")
		}
	}
}

func BenchmarkYAMLUnmarshal_DocuSign(b *testing.B) {
	data := benchmarkSpec(b)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	for b.Loop() {
		var node yaml.Node
		if err := yaml.Unmarshal(data, &node); err != nil {
			b.Fatal(err)
		}
	}
}
