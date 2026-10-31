package model

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/pb33f/testify/require"
)

func diamondComparisonSpec(prefix string, depth int) string {
	var spec strings.Builder
	spec.WriteString("openapi: 3.0.3\ninfo: {title: test, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n    Root:\n      type: object\n      properties:\n        entry:\n          $ref: '#/components/schemas/" + prefix + "0'\n")
	for i := 0; i < depth; i++ {
		fmt.Fprintf(&spec, "    %s%d:\n      type: object\n      properties:\n        a: {$ref: '#/components/schemas/%s%d'}\n        b: {$ref: '#/components/schemas/%s%d'}\n", prefix, i, prefix, i+1, prefix, i+1)
	}
	fmt.Fprintf(&spec, "    %s%d: {type: string}\n", prefix, depth)
	return spec.String()
}

func TestCompareSchemas_RetargetedDiamondIsBounded(t *testing.T) {
	for _, depth := range []int{8, 16, 24} {
		t.Run(fmt.Sprint(depth), func(t *testing.T) {
			left, right := test_BuildDoc(diamondComparisonSpec("L", depth), diamondComparisonSpec("R", depth))
			l, r := left.Components.Value.FindSchema("Root").Value, right.Components.Value.FindSchema("Root").Value
			for i := 0; i < 3; i++ {
				changes := CompareSchemas(l, r)
				require.Equal(t, 2*depth+1, changes.TotalChanges())
				current := changes.SchemaPropertyChanges["entry"]
				for level := 0; level < depth; level++ {
					require.Equal(t, 1, current.SchemaPropertyChanges["b"].TotalChanges())
					current = current.SchemaPropertyChanges["a"]
				}
			}
			var wg sync.WaitGroup
			counts := make([]int, 4)
			for i := range counts {
				wg.Add(1)
				go func() { defer wg.Done(); counts[i] = CompareSchemas(l, r).TotalChanges() }()
			}
			wg.Wait()
			for _, count := range counts {
				require.Equal(t, 2*depth+1, count)
			}
		})
	}
}

func TestCompareSchemas_CandidateScoringHasIndependentPairs(t *testing.T) {
	left := `openapi: 3.0.3
components:
  schemas:
    Root:
      type: object
      properties:
        first:
          anyOf:
            - {$ref: '#/components/schemas/A'}
            - {$ref: '#/components/schemas/B'}
            - {$ref: '#/components/schemas/X'}
        second:
          anyOf:
            - {$ref: '#/components/schemas/A'}
            - {$ref: '#/components/schemas/B'}
            - {$ref: '#/components/schemas/X'}
    A: {type: object, properties: {a: {type: string}}}
    B: {type: object, properties: {b: {type: integer}}}
    X: {type: object, properties: {x: {type: boolean}}}
`
	right := `openapi: 3.0.3
components:
  schemas:
    Root:
      type: object
      properties:
        first:
          anyOf:
            - {$ref: '#/components/schemas/B2'}
            - {$ref: '#/components/schemas/A2'}
        second:
          anyOf:
            - {$ref: '#/components/schemas/B2'}
            - {$ref: '#/components/schemas/A2'}
    A2: {type: object, properties: {a: {type: integer}}}
    B2: {type: object, properties: {b: {type: string}}}
`
	l, r := test_BuildDoc(left, right)
	for i := 0; i < 3; i++ {
		changes := CompareSchemas(l.Components.Value.FindSchema("Root").Value, r.Components.Value.FindSchema("Root").Value)
		require.Equal(t, 8, changes.TotalChanges())
		for _, name := range []string{"first", "second"} {
			union := changes.SchemaPropertyChanges[name]
			require.Len(t, union.AnyOfChanges, 2)
			for _, branch := range union.AnyOfChanges {
				for _, property := range branch.SchemaPropertyChanges {
					require.Len(t, property.Changes, 1)
					require.Equal(t, "type", property.Changes[0].Property)
				}
			}
		}
		require.Equal(t, 5, changes.SchemaPropertyChanges["first"].TotalChanges())
		require.Equal(t, 3, changes.SchemaPropertyChanges["second"].TotalChanges())
	}
}
