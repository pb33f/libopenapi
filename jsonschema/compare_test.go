// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"strconv"
	"strings"
	"testing"
	"time"

	wc "github.com/pb33f/libopenapi/what-changed/model"
	"github.com/pb33f/testify/require"
)

func TestNativeValidationDirections(t *testing.T) {
	for _, tc := range []struct {
		name, old, current string
		forward, reverse   int
	}{
		{"first type", `{}`, `{"type":"string"}`, 1, 0},
		{"integer to number", `{"type":"integer"}`, `{"type":"number"}`, 0, 1},
		{"disjoint type", `{"type":"string"}`, `{"type":"boolean"}`, 1, 1},
		{"type union", `{"type":["number","object"]}`, `{"type":["integer","object"]}`, 1, 0},
		{"new const", `{}`, `{"const":"blue"}`, 1, 0},
		{"const type", `{"const":1}`, `{"const":"1"}`, 1, 1},
		{"pattern", `{}`, `{"pattern":"^(usd|cad)$"}`, 1, 0},
		{"unique array", `{}`, `{"uniqueItems":true}`, 1, 0},
		{"nonunique default", `{}`, `{"uniqueItems":false}`, 0, 0},
		{"unique switch", `{"uniqueItems":false}`, `{"uniqueItems":true}`, 1, 0},
		{"new multiple", `{}`, `{"multipleOf":0.25}`, 1, 0},
		{"divisible multiple", `{"multipleOf":0.25}`, `{"multipleOf":1.5}`, 1, 0},
		{"incomparable multiple", `{"multipleOf":0.25}`, `{"multipleOf":0.3}`, 1, 1},
		{"new dependency", `{}`, `{"dependentRequired":{"a":["b"]}}`, 1, 0},
		{"added dependency", `{"dependentRequired":{"a":["b"]}}`, `{"dependentRequired":{"a":["b","c"]}}`, 1, 0},
		{"swapped dependency", `{"dependentRequired":{"a":["b"]}}`, `{"dependentRequired":{"a":["c"]}}`, 1, 1},
		{"first anyOf", `{}`, `{"anyOf":[{"type":"number"}]}`, 1, 0},
		{"another anyOf", `{"anyOf":[{"type":"number"}]}`, `{"anyOf":[{"type":"number"},{"type":"string"}]}`, 0, 1},
		{"first negated anyOf", `{"not":{}}`, `{"not":{"anyOf":[{"type":"number"}]}}`, 0, 1},
		{"additional negated anyOf", `{"not":{"anyOf":[{"type":"number"}]}}`, `{"not":{"anyOf":[{"type":"number"},{"type":"string"}]}}`, 1, 0},
		{"negated type", `{"not":{}}`, `{"not":{"type":"string"}}`, 0, 1},
		{"negated dependency", `{"not":{"dependentRequired":{"a":["b"]}}}`, `{"not":{"dependentRequired":{"a":["b","c"]}}}`, 0, 1},
		{"content annotation", `{"contentSchema":{"minLength":4}}`, `{"contentSchema":{"minLength":7}}`, 0, 0},
		{"referenced content schema", `{"contentSchema":{"minLength":4},"$ref":"#/contentSchema"}`, `{"contentSchema":{"minLength":7},"$ref":"#/contentSchema"}`, 1, 0},
		{"negated reference to content schema", `{"contentSchema":{"minLength":4},"not":{"$ref":"#/contentSchema"}}`, `{"contentSchema":{"minLength":7},"not":{"$ref":"#/contentSchema"}}`, 0, 1},
		{"content field addition", `{}`, `{"contentSchema":{"type":"string"}}`, 0, 0},
		{"bounded contains", `{"contains":{"minLength":2},"maxContains":1}`, `{"contains":{"minLength":4},"maxContains":1}`, 1, 1},
		{"new applied resource", `{}`, `{"properties":{"x":{"$id":"urn:field","type":"string"}}}`, 1, 1},
		{"new unused definition", `{}`, `{"$defs":{"x":{"$id":"urn:field","type":"string"}}}`, 0, 1},
		{"dialect declaration", `{}`, `{"$schema":"` + Dialect + `"}`, 0, 0},
		{"dialect spelling", `{"$schema":"` + Dialect + `"}`, `{"$schema":"` + Dialect + `#"}`, 0, 0},
		{"negated dialect declaration", `{"not":{}}`, `{"not":{"$schema":"` + Dialect + `"}}`, 0, 0},
		{"lower bound", `{"minimum":2}`, `{"minimum":3}`, 1, 0},
		{"upper bound", `{"maxLength":5}`, `{"maxLength":4}`, 1, 0},
		{"new upper bound", `{}`, `{"maxItems":5}`, 1, 0},
		{"negated lower bound", `{"not":{"minimum":2}}`, `{"not":{"minimum":3}}`, 0, 1},
		{"negated new upper bound", `{"not":{}}`, `{"not":{"maxItems":5}}`, 0, 1},
		{"bound beyond exact limits", `{"minimum":1e999999}`, `{"minimum":1e999998}`, 1, 1},
		{"negated bound beyond exact limits", `{"not":{"maximum":1e999999}}`, `{"not":{"maximum":1e999998}}`, 1, 1},
		{"multiple beyond exact limits", `{"multipleOf":1e-999999}`, `{"multipleOf":1e-999998}`, 1, 1},
		{"negated requirement", `{"not":{"required":["a"]}}`, `{"not":{"required":["a","b"]}}`, 0, 1},
		{"ambiguous requirement", `{"oneOf":[{"required":["a"]}]}`, `{"oneOf":[{"required":["a","b"]}]}`, 1, 1},
		{"closed object", `{}`, `{"additionalProperties":false}`, 1, 0},
		{"item schema", `{}`, `{"items":{"maxLength":5}}`, 1, 0},
		{"negation", `{}`, `{"not":{"type":"string"}}`, 1, 0},
		{"property names", `{}`, `{"propertyNames":{"maxLength":5}}`, 1, 0},
		{"condition", `{"then":{"minLength":1}}`, `{"if":{"type":"string"},"then":{"minLength":1}}`, 1, 0},
		{"last unevaluated keyword", `{}`, `{"unevaluatedProperties":false}`, 1, 0},
		{"negated item schema", `{"not":{}}`, `{"not":{"items":{"maxLength":5}}}`, 0, 1},
		{"ambiguous item schema", `{"oneOf":[{}]}`, `{"oneOf":[{"items":{"maxLength":5}}]}`, 1, 1},
		{"evaluated items", `{"unevaluatedItems":false}`, `{"unevaluatedItems":false,"items":{"maxLength":5}}`, 1, 1},
		{"composed branch", `{"allOf":[{"minLength":1}]}`, `{"allOf":[{"minLength":1},{"maxLength":5}]}`, 1, 0},
		{"evaluated branch", `{"allOf":[{"properties":{"a":{}}}],"unevaluatedProperties":false}`, `{"allOf":[{"properties":{"a":{}}},{"properties":{"b":{}}}],"unevaluatedProperties":false}`, 1, 1},
		{"property schema", `{"properties":{"a":{}}}`, `{"properties":{"a":{},"b":{}}}`, 1, 1},
		{"tuple slot", `{"prefixItems":[{}]}`, `{"prefixItems":[{},{}]}`, 1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := loadTest(t, tc.old), loadTest(t, tc.current)
			for i, sides := range [][2]*Document{{a, b}, {b, a}} {
				changes, err := Compare(sides[0], sides[1], nil)
				require.NoError(t, err)
				require.NotNil(t, changes)
				require.Equal(t, 1, changes.TotalChanges())
				want := []int{tc.forward, tc.reverse}[i]
				require.Equal(t, want, changes.TotalBreakingChanges(), "direction %d", i)
			}
		})
	}
}

func TestNativeLiteralValues(t *testing.T) {
	for _, tc := range []struct {
		name, key, old, current string
		want, breaking          int
	}{
		{"reordered types", "type", `["array","string"]`, `["string","array"]`, 0, 0},
		{"enum scalar type", "enum", `[1]`, `["1"]`, 2, 1},
		{"enum object", "enum", `[{"id":1}]`, `[{"id":2}]`, 2, 1},
		{"enum array", "enum", `[[1,2]]`, `[[2,1]]`, 2, 1},
		{"enum boolean and string", "enum", `[true,"false"]`, `[true,false]`, 2, 1},
		{"enum canonical equality", "enum", `[{"x":1,"y":true}, 1.00]`, `[1e0,{"y":true,"x":1.0}]`, 0, 0},
		{"object const", "const", `{"id":1}`, `{"id":"1"}`, 1, 1},
		{"structured examples", "examples", `[{"count":1}]`, `[{"count":2}]`, 1, 0},
		{"vocabulary", "$vocabulary", `{"urn:foo":false}`, `{"urn:bar":false}`, 1, 1},
		{"type array", "type", `["object","string"]`, `["array","string"]`, 1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			left, right := loadTest(t, `{"`+tc.key+`":`+tc.old+`}`), loadTest(t, `{"`+tc.key+`":`+tc.current+`}`)
			for _, pair := range [][2]*Document{{left, right}, {right, left}} {
				changes, err := Compare(pair[0], pair[1], nil)
				require.NoError(t, err)
				if tc.want == 0 {
					require.Nil(t, changes)
					continue
				}
				require.NotNil(t, changes)
				require.Equal(t, tc.want, changes.TotalChanges())
				require.Equal(t, tc.breaking, changes.TotalBreakingChanges())
				for _, ch := range changes.GetAllChanges() {
					require.Equal(t, tc.key, ch.Property)
					if ch.ChangeType != wc.PropertyAdded {
						require.NotEmpty(t, ch.OriginalEncoded)
					}
					if ch.ChangeType != wc.PropertyRemoved {
						require.NotEmpty(t, ch.NewEncoded)
					}
				}
			}
		})
	}
}

func TestNativeKeywordPolicyOverrides(t *testing.T) {
	cfg := &wc.BreakingRulesConfig{Schema: &wc.SchemaRules{
		SchemaDialect:        &wc.BreakingChangeRule{Added: new(false), Removed: new(false)},
		AdditionalProperties: &wc.BreakingChangeRule{Modified: new(false)},
	}}
	for _, tc := range [][2]string{{`{}`, `{"$schema":"` + Dialect + `"}`}, {`{"additionalProperties":true}`, `{"additionalProperties":false}`}} {
		a, b := loadTest(t, tc[0]), loadTest(t, tc[1])
		for _, pair := range [][2]*Document{{a, b}, {b, a}} {
			changes, err := Compare(pair[0], pair[1], cfg)
			require.NoError(t, err)
			require.NotNil(t, changes)
			require.Equal(t, 1, changes.TotalChanges())
			require.Zero(t, changes.TotalBreakingChanges())
		}
	}
}

// A rule replaces the verdict in both directions, so merged defaults must not
// stand in for the direction-aware ones.
func TestRulesAreExplicitOverrides(t *testing.T) {
	override := func(rule wc.SchemaRules) *wc.BreakingRulesConfig { return &wc.BreakingRulesConfig{Schema: &rule} }
	for _, tc := range []struct {
		name, old, current string
		rules              *wc.BreakingRulesConfig
		breaking           int
	}{
		{"widening", `{"minLength":3}`, `{"minLength":2}`, nil, 0},
		{"unrelated override", `{"minLength":3}`, `{"minLength":2}`, override(wc.SchemaRules{MaxLength: &wc.BreakingChangeRule{Modified: new(true)}}), 0},
		{"keyword override", `{"minLength":3}`, `{"minLength":2}`, override(wc.SchemaRules{MinLength: &wc.BreakingChangeRule{Modified: new(true)}}), 1},
		{"merged defaults", `{"minLength":3}`, `{"minLength":2}`, wc.GenerateDefaultBreakingRules(), 1},
		{"dialect override", `{}`, `{"$schema":"` + Dialect + `"}`, override(wc.SchemaRules{SchemaDialect: &wc.BreakingChangeRule{Added: new(true)}}), 1},
		{"root replacement has no keyword", `true`, `false`, override(wc.SchemaRules{Type: &wc.BreakingChangeRule{Modified: new(false)}}), 1},
		{"nested replacement", `{"properties":{"a":true}}`, `{"properties":{"a":false}}`, override(wc.SchemaRules{Properties: &wc.BreakingChangeRule{Modified: new(false)}}), 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changes, err := Compare(loadTest(t, tc.old), loadTest(t, tc.current), tc.rules)
			require.NoError(t, err)
			require.Equal(t, 1, changes.TotalChanges())
			require.Equal(t, tc.breaking, changes.TotalBreakingChanges())
		})
	}
}

// Before the limit, each of these exponents cost tens of milliseconds to expand.
func TestNumbersBeyondExactLimitsAreNotExpanded(t *testing.T) {
	schema := func(exponent string) *Document {
		var properties []string
		for i := range 200 {
			properties = append(properties, strconv.Quote("p"+strconv.Itoa(i))+`:{"minimum":1e`+exponent+`}`)
		}
		return loadTest(t, `{"properties":{`+strings.Join(properties, ",")+`}}`)
	}
	left, right := schema("999999"), schema("999998")
	start := time.Now()
	changes, err := Compare(left, right, nil)
	require.NoError(t, err)
	require.Less(t, time.Since(start), 2*time.Second)
	require.Equal(t, 200, changes.TotalChanges())
	require.Equal(t, 200, changes.TotalBreakingChanges())
}
