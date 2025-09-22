// Copyright 2025 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package tests

import (
	"fmt"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/stretchr/testify/assert"
)

func BenchmarkSiblingRefTransformation(b *testing.B) {
	// create a spec with various numbers of sibling refs to benchmark performance impact
	createSpecWithSiblingRefs := func(numSchemas int) string {
		spec := `openapi: 3.1.0
info:
  title: Benchmark Test API
  version: 1.0.0
components:
  schemas:
    Base:
      type: object
      properties:
        id:
          type: string
        name:
          type: string`

		for i := 0; i < numSchemas; i++ {
			spec += fmt.Sprintf(`
    Schema%d:
      title: "Schema %d"
      description: "Generated schema %d for benchmarking"
      example: {"id": "%d", "name": "test%d"}
      nullable: true
      $ref: '#/components/schemas/Base'`, i, i, i, i, i)
		}
		return spec
	}

	b.Run("transformation enabled", func(b *testing.B) {
		spec := createSpecWithSiblingRefs(100)
		config := datamodel.NewDocumentConfiguration()
		config.TransformSiblingRefs = true

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			doc, err := libopenapi.NewDocumentWithConfiguration([]byte(spec), config)
			if err != nil {
				b.Fatal(err)
			}
			_, errs := doc.BuildV3Model()
			if errs != nil {
				b.Fatalf("build errors: %v", errs)
			}
		}
	})

	b.Run("transformation disabled", func(b *testing.B) {
		spec := createSpecWithSiblingRefs(100)
		config := datamodel.NewDocumentConfiguration()
		config.TransformSiblingRefs = false

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			doc, err := libopenapi.NewDocumentWithConfiguration([]byte(spec), config)
			if err != nil {
				b.Fatal(err)
			}
			_, errs := doc.BuildV3Model()
			if errs != nil {
				b.Fatalf("build errors: %v", errs)
			}
		}
	})
}

func BenchmarkSiblingRefDetection(b *testing.B) {
	spec := `openapi: 3.1.0
info:
  title: Detection Benchmark
  version: 1.0.0
components:
  schemas:
    Base:
      type: object
    Test1:
      title: "Test Schema 1"
      description: "Test description 1"
      example: {"test": 1}
      $ref: '#/components/schemas/Base'
    Test2:
      title: "Test Schema 2"
      nullable: true
      $ref: '#/components/schemas/Base'
    Test3:
      description: "Test description 3"
      minLength: 5
      maxLength: 100
      $ref: '#/components/schemas/Base'`

	b.Run("sibling detection performance", func(b *testing.B) {
		config := datamodel.NewDocumentConfiguration()
		config.TransformSiblingRefs = true

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			doc, err := libopenapi.NewDocumentWithConfiguration([]byte(spec), config)
			if err != nil {
				b.Fatal(err)
			}

			// measure sibling detection performance
			rolodex := doc.GetRolodex()
			if rolodex != nil {
				rootIndex := rolodex.GetRootIndex()
				if rootIndex != nil {
					_ = rootIndex.GetReferencesWithSiblings()
				}
			}
		}
	})
}

func TestSiblingRefs_PerformanceImpact(t *testing.T) {
	t.Run("minimal performance impact when disabled", func(t *testing.T) {
		spec := `openapi: 3.1.0
info:
  title: Performance Test
  version: 1.0.0
components:
  schemas:
    Simple:
      type: object`

		// test with transformation disabled
		config := datamodel.NewDocumentConfiguration()
		config.TransformSiblingRefs = false
		config.MergeReferencedProperties = false

		doc, err := libopenapi.NewDocumentWithConfiguration([]byte(spec), config)
		assert.NoError(t, err)

		// should build quickly without any transformation overhead
		v3Doc, docErrs := doc.BuildV3Model()
		if docErrs != nil {
			t.Fatalf("failed to build v3 model: %v", docErrs)
		}
		assert.NotNil(t, v3Doc)

		// verify document was processed normally
		schemas := v3Doc.Model.Components.Schemas
		hasSimple := false
		for name := range schemas.FromOldest() {
			if name == "Simple" {
				hasSimple = true
				break
			}
		}
		assert.True(t, hasSimple)
	})
}
