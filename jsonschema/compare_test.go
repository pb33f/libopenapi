// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"testing"

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
