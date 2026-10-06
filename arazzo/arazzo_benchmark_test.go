// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"fmt"
	"strings"
	"testing"

	libopenapi "github.com/pb33f/libopenapi"
)

func largeArazzoDocument(stepCount, outputCount int) []byte {
	var source strings.Builder
	source.Grow(256 + stepCount*80 + outputCount*120)
	source.WriteString(`arazzo: 1.1.0
info:
  title: large benchmark
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: openapi.yaml
    type: openapi
workflows:
  - workflowId: benchmark
    steps:
`)
	for index := range stepCount {
		fmt.Fprintf(
			&source,
			"      - stepId: step-%d\n        operationId: operation-%d\n",
			index,
			index,
		)
		if index == 0 && outputCount > 0 {
			source.WriteString("        outputs:\n")
			for outputIndex := range outputCount {
				fmt.Fprintf(
					&source,
					"          output-%d:\n            context: $response.body\n            selector: $.items[%d]\n            type: jsonpath\n",
					outputIndex,
					outputIndex,
				)
			}
		}
	}
	return []byte(source.String())
}

func BenchmarkNewLargeArazzoDocument(b *testing.B) {
	source := largeArazzoDocument(1_000, 0)
	b.ReportAllocs()
	b.SetBytes(int64(len(source)))
	b.ResetTimer()
	for range b.N {
		document, err := libopenapi.NewArazzoDocument(source)
		if err != nil {
			b.Fatal(err)
		}
		if len(document.Workflows) != 1 ||
			len(document.Workflows[0].Steps) != 1_000 {
			b.Fatal("large Arazzo document was not fully parsed")
		}
	}
}

func BenchmarkRenderLargeArazzoSelectorMap(b *testing.B) {
	document, err := libopenapi.NewArazzoDocument(largeArazzoDocument(1, 1_000))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		rendered, renderErr := document.Render()
		if renderErr != nil {
			b.Fatal(renderErr)
		}
		if len(rendered) == 0 {
			b.Fatal("large Arazzo render was empty")
		}
	}
}
