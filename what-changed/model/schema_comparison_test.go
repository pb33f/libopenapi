package model

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low/base"
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

func TestCompareSchemas_NestedUnionScoresAreShared(t *testing.T) {
	for _, union := range []string{"anyOf", "oneOf"} {
		for _, depth := range []int{0, 1, 2, 3, 12} {
			t.Run(fmt.Sprintf("%s/%d", union, depth), func(t *testing.T) {
				makeSpec := func(prefix, kind string) string {
					var spec strings.Builder
					spec.WriteString("openapi: 3.0.3\ninfo: {title: Union, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n")
					fmt.Fprintf(&spec, "    Root: {$ref: '#/components/schemas/%s0'}\n", prefix)
					for i := 0; i < depth; i++ {
						fmt.Fprintf(&spec, "    %s%d:\n      %s:\n        - {$ref: '#/components/schemas/%s%d'}\n        - {$ref: '#/components/schemas/%s%d'}\n", prefix, i, union, prefix, i+1, prefix, i+1)
					}
					fmt.Fprintf(&spec, "    %s%d: {type: %s}\n", prefix, depth, kind)
					return spec.String()
				}
				l, r := test_BuildDoc(makeSpec("L", "string"), makeSpec("R", "integer"))
				left, right := l.Components.Value.FindSchema("Root").Value, r.Components.Value.FindSchema("Root").Value
				comparison := new(schemaComparison)
				changes := comparison.compareSchemas(left, right)
				require.Equal(t, 2*depth+2, changes.TotalChanges())
				// Four candidates at each level share one complete score. Nested
				// candidates reuse the same cache instead of starting four more trees.
				require.Len(t, comparison.scores, depth)
				require.Equal(t, changes.TotalChanges(), CompareSchemas(left, right).TotalChanges())
				if depth == 12 {
					var wg sync.WaitGroup
					counts := make([]int, 4)
					for i := range counts {
						wg.Add(1)
						go func() { defer wg.Done(); counts[i] = CompareSchemas(left, right).TotalChanges() }()
					}
					wg.Wait()
					for _, count := range counts {
						require.Equal(t, 2*depth+2, count)
					}
				}
			})
		}
	}
}

func TestCompareSchemas_CandidateScoresPreserveReferenceIdentity(t *testing.T) {
	makeSpec := func(kind string) string {
		return fmt.Sprintf(`openapi: 3.0.3
components:
  schemas:
    Target: {type: %s}
    Alias: {$ref: '#/components/schemas/Target'}
    Ref: {$ref: '#/components/schemas/Target'}
    AliasRef: {$ref: '#/components/schemas/Alias'}
`, kind)
	}
	l, r := test_BuildDoc(makeSpec("string"), makeSpec("integer"))
	left := l.Components.Value.FindSchema("Ref").Value
	right := r.Components.Value.FindSchema("Ref").Value
	alias := r.Components.Value.FindSchema("AliasRef").Value
	inlineLeft := l.Components.Value.FindSchema("Target").Value
	inlineRight := r.Components.Value.FindSchema("Target").Value
	require.Same(t, schemaDefinitionNode(right), schemaDefinitionNode(alias))
	require.Same(t, schemaDefinitionNode(right), schemaDefinitionNode(inlineRight))
	comparison := new(schemaComparison)
	for _, pair := range [][2]*base.SchemaProxy{{left, right}, {left, alias}, {inlineLeft, right}, {left, inlineRight}} {
		var expected schemaScore
		if changes := CompareSchemas(pair[0], pair[1]); changes != nil {
			expected = schemaScore{changes.TotalChanges(), changes.TotalBreakingChanges()}
		}
		require.Equal(t, expected, comparison.score(pair[0], pair[1]))
		require.Equal(t, expected, comparison.score(pair[0], pair[1]))
	}
	require.Len(t, comparison.scores, 4)
}

func TestCompareSchemas_CircularUnionCandidatesStopAtReferences(t *testing.T) {
	for _, union := range []string{"anyOf", "oneOf"} {
		for _, mutual := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/mutual=%t", union, mutual), func(t *testing.T) {
				makeSpec := func(prefix string) string {
					next := prefix + "A"
					if mutual {
						next = prefix + "B"
					}
					return fmt.Sprintf(`openapi: 3.0.3
components:
  schemas:
    Root:
      %s:
        - {$ref: '#/components/schemas/%sA'}
        - {$ref: '#/components/schemas/%sA'}
    %sA:
      %s:
        - {$ref: '#/components/schemas/%s'}
        - {type: string}
    %sB:
      %s:
        - {$ref: '#/components/schemas/%sA'}
        - {type: integer}
`, union, prefix, prefix, prefix, union, next, prefix, union, prefix)
				}
				l, r := test_BuildDoc(makeSpec("L"), makeSpec("R"))
				left, right := l.Components.Value.FindSchema("Root").Value, r.Components.Value.FindSchema("Root").Value
				comparison := new(schemaComparison)
				changes := comparison.compareSchemas(left, right)
				require.Equal(t, 2, changes.TotalChanges())
				require.Len(t, comparison.scores, 1)
				for _, score := range comparison.scores {
					require.Equal(t, 1, score.total)
				}
			})
		}
	}
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

func TestCompareSchemas_RetargetedDiamondWithPropertyAddRemove(t *testing.T) {
	const depth = 8
	left := diamondComparisonSpec("L", depth)
	right := strings.Replace(diamondComparisonSpec("R", depth), "        entry:", "        extra: {type: string}\n        entry:", 1)
	l, r := test_BuildDoc(left, right)
	for _, reverse := range []bool{false, true} {
		lp, rp := l.Components.Value.FindSchema("Root").Value, r.Components.Value.FindSchema("Root").Value
		if reverse {
			lp, rp = rp, lp
		}
		changes := CompareSchemas(lp, rp)
		require.Equal(t, 2*depth+2, changes.TotalChanges())
		require.Equal(t, 2*depth+1, changes.SchemaPropertyChanges["entry"].TotalChanges())
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
