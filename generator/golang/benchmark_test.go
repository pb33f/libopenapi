// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"os"
	"reflect"
	"testing"

	"github.com/pb33f/libopenapi"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
)

func BenchmarkRenderTrainTravel(b *testing.B) {
	schemas := benchmarkTrainTravelSchemas(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := NewGenerator().RenderSchemas(schemas); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRenderTrainTravelTypedUnion(b *testing.B) {
	schemas := benchmarkTrainTravelSchemas(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := NewGenerator(WithOptionalConstDiscriminatorUnions(true)).RenderSchemas(schemas); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSchemasFromTypesComponentGraph(b *testing.B) {
	generator := NewGenerator(
		WithOneOfTypes((*PhaseTwoPaymentMethod)(nil), PhaseTwoCard{}, PhaseTwoBank{}),
		WithDiscriminatorMapping((*PhaseTwoPaymentMethod)(nil), "object", map[string]string{
			"bank": "#/components/schemas/PhaseTwoBank",
			"card": "#/components/schemas/PhaseTwoCard",
		}),
	)
	target := reflect.TypeOf(PhaseTwoCustomer{})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := generator.SchemasFromTypes(target); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRenderSyntheticLargeSchema(b *testing.B) {
	schemas := orderedmap.New[string, *highbase.SchemaProxy]()
	for i := 0; i < 75; i++ {
		props := orderedmap.New[string, *highbase.SchemaProxy]()
		props.Set("id", highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}}))
		props.Set("name", highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}}))
		props.Set("labels", highbase.CreateSchemaProxy(&highbase.Schema{
			Type: []string{"object"},
			AdditionalProperties: &highbase.DynamicValue[*highbase.SchemaProxy, bool]{
				A: highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}}),
			},
		}))
		schemas.Set("SyntheticModel"+intString(i), highbase.CreateSchemaProxy(&highbase.Schema{
			Type:       []string{"object"},
			Required:   []string{"id"},
			Properties: props,
		}))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := NewGenerator().RenderSchemas(schemas); err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkTrainTravelSchemas(tb testing.TB) *orderedmap.Map[string, *highbase.SchemaProxy] {
	tb.Helper()
	spec, err := os.ReadFile("testdata/train-travel.yaml")
	if err != nil {
		tb.Fatal(err)
	}
	doc, err := libopenapi.NewDocument(spec)
	if err != nil {
		tb.Fatal(err)
	}
	model, err := doc.BuildV3Model()
	if err != nil {
		tb.Fatal(err)
	}
	return model.Model.Components.Schemas
}
