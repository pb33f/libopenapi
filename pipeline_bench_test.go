// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package libopenapi_test

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"runtime"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/bundler"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/high/base"
)

// Pipeline benchmarks exercise the full public entry points against large, real-world specifications.
// Each benchmark also reports "retained-B/op": the live heap left behind by one iteration's result after a
// forced GC, which is the memory a long-lived consumer pays to hold the model.

func pipelineSpec(b *testing.B, path string) []byte {
	b.Helper()
	spec, err := os.ReadFile(path)
	if err != nil {
		b.Fatalf("read %s: %v", path, err)
	}
	return spec
}

func pipelineConfig() *datamodel.DocumentConfiguration {
	return &datamodel.DocumentConfiguration{
		IgnorePolymorphicCircularReferences: true,
		IgnoreArrayCircularReferences:       true,
		// keep resolution errors out of the benchmark output, where they break benchstat parsing.
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func heapInUse() uint64 {
	runtime.GC()
	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return ms.HeapAlloc
}

func benchmarkPipelineV3(b *testing.B, path string) {
	spec := pipelineSpec(b, path)
	b.ReportAllocs()
	b.ResetTimer()
	var keep any
	for i := 0; i < b.N; i++ {
		doc, err := libopenapi.NewDocumentWithConfiguration(spec, pipelineConfig())
		if err != nil {
			b.Fatal(err)
		}
		m, _ := doc.BuildV3Model()
		if m == nil {
			b.Fatal("nil model")
		}
		keep = m
	}
	b.StopTimer()
	keep = nil
	before := heapInUse()
	doc, _ := libopenapi.NewDocumentWithConfiguration(spec, pipelineConfig())
	m, _ := doc.BuildV3Model()
	keep = m
	after := heapInUse()
	runtime.KeepAlive(keep)
	runtime.KeepAlive(doc)
	b.ReportMetric(float64(after-before), "retained-B/op")
}

func benchmarkPipelineV2(b *testing.B, path string) {
	spec := pipelineSpec(b, path)
	b.ReportAllocs()
	b.ResetTimer()
	var keep any
	for i := 0; i < b.N; i++ {
		doc, err := libopenapi.NewDocumentWithConfiguration(spec, pipelineConfig())
		if err != nil {
			b.Fatal(err)
		}
		m, _ := doc.BuildV2Model()
		if m == nil {
			b.Fatal("nil model")
		}
		keep = m
	}
	b.StopTimer()
	keep = nil
	before := heapInUse()
	doc, _ := libopenapi.NewDocumentWithConfiguration(spec, pipelineConfig())
	m, _ := doc.BuildV2Model()
	keep = m
	after := heapInUse()
	runtime.KeepAlive(keep)
	runtime.KeepAlive(doc)
	b.ReportMetric(float64(after-before), "retained-B/op")
}

func BenchmarkPipeline_BuildV3_Stripe(b *testing.B) { benchmarkPipelineV3(b, "test_specs/stripe.yaml") }
func BenchmarkPipeline_BuildV3_DocuSign(b *testing.B) {
	benchmarkPipelineV3(b, "test_specs/docusignv3.1.json")
}
func BenchmarkPipeline_BuildV3_Asana(b *testing.B) { benchmarkPipelineV3(b, "test_specs/asana.yaml") }
func BenchmarkPipeline_BuildV2_K8s(b *testing.B)   { benchmarkPipelineV2(b, "test_specs/k8s.json") }
func BenchmarkPipeline_BuildV2_Xsoar(b *testing.B) { benchmarkPipelineV2(b, "test_specs/xsoar.json") }

func BenchmarkPipeline_Render_Stripe(b *testing.B) {
	spec := pipelineSpec(b, "test_specs/stripe.yaml")
	doc, err := libopenapi.NewDocumentWithConfiguration(spec, pipelineConfig())
	if err != nil {
		b.Fatal(err)
	}
	if m, _ := doc.BuildV3Model(); m == nil {
		b.Fatal("nil model")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, rErr := doc.Render()
		if rErr != nil || len(out) == 0 {
			b.Fatal(rErr)
		}
	}
}

func BenchmarkPipeline_Render_DocuSignJSON(b *testing.B) {
	spec := pipelineSpec(b, "test_specs/docusignv3.1.json")
	doc, err := libopenapi.NewDocumentWithConfiguration(spec, pipelineConfig())
	if err != nil {
		b.Fatal(err)
	}
	if m, _ := doc.BuildV3Model(); m == nil {
		b.Fatal("nil model")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, rErr := doc.Render()
		if rErr != nil || len(out) == 0 {
			b.Fatal(rErr)
		}
	}
}

func BenchmarkPipeline_Compare_Stripe(b *testing.B) {
	left := pipelineSpec(b, "test_specs/stripe.yaml")
	// mutate a slice of the spec so the comparison has real changes to report, not just an identity walk.
	right := bytes.Replace(left, []byte("type: string"), []byte("type: integer"), 200)
	right = bytes.Replace(right, []byte("nullable: true"), []byte("nullable: false"), 300)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l, err := libopenapi.NewDocumentWithConfiguration(left, pipelineConfig())
		if err != nil {
			b.Fatal(err)
		}
		r, err := libopenapi.NewDocumentWithConfiguration(right, pipelineConfig())
		if err != nil {
			b.Fatal(err)
		}
		changes, _ := libopenapi.CompareDocuments(l, r)
		if changes == nil {
			b.Fatal("no changes")
		}
	}
}

func BenchmarkPipeline_Bundle_Stripe(b *testing.B) {
	spec := pipelineSpec(b, "test_specs/stripe.yaml")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, _ := bundler.BundleBytes(spec, pipelineConfig())
		if len(out) == 0 {
			b.Fatal("empty bundle")
		}
	}
}

// BenchmarkPipeline_RenderSchemasInline_Stripe renders every component schema inline in validation mode,
// the way request/response validators compile schemas.
func BenchmarkPipeline_RenderSchemasInline_Stripe(b *testing.B) {
	spec := pipelineSpec(b, "test_specs/stripe.yaml")
	doc, err := libopenapi.NewDocumentWithConfiguration(spec, pipelineConfig())
	if err != nil {
		b.Fatal(err)
	}
	m, _ := doc.BuildV3Model()
	if m == nil {
		b.Fatal("nil model")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, proxy := range m.Model.Components.Schemas.FromOldest() {
			if _, rErr := proxy.Schema().RenderInlineWithContext(base.NewInlineRenderContextForValidation()); rErr != nil {
				b.Fatal(rErr)
			}
		}
	}
}
