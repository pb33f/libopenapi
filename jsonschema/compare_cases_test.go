// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"fmt"
	"testing"

	"github.com/pb33f/go-yaml"
	wc "github.com/pb33f/libopenapi/what-changed/model"
	"github.com/pb33f/testify/require"
)

func TestComparisonInputs(t *testing.T) {
	single := loadTest(t, "true")
	collection := loadMembers(t, "snapshot", map[string][]byte{})
	for _, tc := range []struct {
		left, right *Document
		message     string
	}{
		{nil, single, "both JSON Schema inputs are required"},
		{single, nil, "both JSON Schema inputs are required"},
		{single, collection, "collection on both sides"},
		{collection, single, "collection on both sides"},
	} {
		diff, err := Compare(tc.left, tc.right, nil)
		require.ErrorContains(t, err, tc.message)
		require.Nil(t, diff)
	}
	require.Nil(t, MatchResources(nil, single))
	require.Nil(t, MatchResources(single, nil))
	diff, err := Compare(single, single, nil)
	require.NoError(t, err)
	require.Nil(t, diff)
}

func TestResourceMoveCanReplaceRemovedFile(t *testing.T) {
	original := loadMembers(t, "before", map[string][]byte{
		"gone.yaml":  []byte("$id: urn:gone"),
		"moved.yaml": []byte("$id: urn:kept"),
	})
	current := loadMembers(t, "after", map[string][]byte{"gone.yaml": []byte("$id: urn:kept")})
	matches := MatchResources(original, current)
	require.Len(t, matches, 1)
	for old, newResource := range matches {
		require.Equal(t, "urn:kept", old.ID)
		require.Equal(t, old.ID, newResource.ID)
	}
	diff, err := Compare(original, current, nil)
	require.NoError(t, err)
	require.Len(t, diff.GetAllChanges(), 1)
	removed := diff.GetAllChanges()[0]
	require.Equal(t, "urn:gone", removed.SchemaResource)
	require.Equal(t, wc.ObjectRemoved, removed.ChangeType)
	require.True(t, removed.Breaking)
}

func TestEquivalentConstraintSpellings(t *testing.T) {
	for _, tc := range []struct{ name, original, modified string }{
		{"boolean case", "FALSE", "false"},
		{"numeric spelling", `{"maximum":1.25}`, `{"maximum":125e-2}`},
		{"dependency order", `{"dependentRequired":{"card":["number","currency"]}}`, `{"dependentRequired":{"card":["currency","number"]}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			diff, err := Compare(loadTest(t, tc.original), loadTest(t, tc.modified), nil)
			require.NoError(t, err)
			require.Nil(t, diff)
		})
	}
}

func TestDependenciesKeepTheirAuthoredDialectNames(t *testing.T) {
	original := loadTest(t, `{"dependentRequired":{"schemaDialect":["oldA"],"$schema":["oldB"]}}`)
	modified := loadTest(t, `{"dependentRequired":{"schemaDialect":["newA"],"$schema":["newB"]}}`)
	diff, err := Compare(original, modified, nil)
	require.NoError(t, err)
	changes := diff.GetAllChanges()
	require.Len(t, changes, 2)
	require.Equal(t, 2, diff.TotalBreakingChanges())
	want := map[string]struct{ path, original, modified string }{
		"schemaDialect": {"$.dependentRequired.schemaDialect", `["oldA"]`, `["newA"]`},
		"$schema":       {`$.dependentRequired["$schema"]`, `["oldB"]`, `["newB"]`},
	}
	for _, ch := range changes {
		expected, exists := want[ch.Property]
		require.True(t, exists, "unexpected dependency %q", ch.Property)
		require.Equal(t, expected.path, ch.Path)
		require.Equal(t, expected.original, ch.OriginalEncoded)
		require.Equal(t, expected.modified, ch.NewEncoded)
		delete(want, ch.Property)
	}
	require.Empty(t, want)
}

func TestAnnotationEditsAlongsideEquivalentDependencies(t *testing.T) {
	original := loadTest(t, `{
        "dependentRequired":{"card":["issuer","currency"]},
        "same":{"a":1,"b":2},
        "former-kind":"legacy"
    }`)
	current := loadTest(t, `{
        "dependentRequired":{"card":["currency","issuer"]},
        "same":{"b":2,"a":1},
        "new-kind":"modern",
        "description":"Issued card"
    }`)
	diff, err := Compare(original, current, nil)
	require.NoError(t, err)
	changes := diff.GetAllChanges()
	require.Len(t, changes, 3)
	require.Zero(t, diff.TotalBreakingChanges())
	byKey := map[string]*wc.Change{}
	for _, ch := range changes {
		byKey[ch.Property] = ch
	}
	require.Contains(t, byKey, "description")
	require.Contains(t, byKey, "former-kind")
	require.Contains(t, byKey, "new-kind")
	require.Equal(t, `$["former-kind"]`, byKey["former-kind"].Path)
	require.Equal(t, wc.PropertyRemoved, byKey["former-kind"].ChangeType)
	require.Equal(t, "legacy", byKey["former-kind"].Original)
	require.Equal(t, `$["new-kind"]`, byKey["new-kind"].Path)
	require.Equal(t, wc.PropertyAdded, byKey["new-kind"].ChangeType)
	require.Equal(t, "modern", byKey["new-kind"].New)
}

func TestCompatibilityOfNestedConstraints(t *testing.T) {
	for _, tc := range []struct {
		name, original, modified, path string
		forward, reverse               bool
	}{
		{"negated boolean", `{"not":false}`, `{"not":true}`, "$.not", true, false},
		{"negated reference", `{"$defs":{"a":true,"b":false},"not":{"$ref":"#/$defs/a"}}`, `{"$defs":{"a":true,"b":false},"not":{"$ref":"#/$defs/b"}}`, `$.not["$ref"]`, true, true},
		{"maximum size", `{"maxLength":5}`, `{"maxLength":3}`, "$.maxLength", true, false},
		{"new conjunct", `{"allOf":[{"type":"number"}]}`, `{"allOf":[{"type":"number"},{"minimum":0}]}`, "$.allOf[1]", true, false},
		{"new dependency", `{}`, `{"dependentSchemas":{"card":{"required":["issuer"]}}}`, "$.dependentSchemas.card", true, false},
		{"new dependency in condition", `{"if":{},"then":false}`, `{"if":{"dependentSchemas":{"card":{"required":["issuer"]}}},"then":false}`, "$.if.dependentSchemas.card", true, true},
		{"content with both reference polarities", `{"contentSchema":{"minLength":1},"allOf":[{"$ref":"#/contentSchema"},{"not":{"$ref":"#/contentSchema"}}]}`, `{"contentSchema":{"minLength":2},"allOf":[{"$ref":"#/contentSchema"},{"not":{"$ref":"#/contentSchema"}}]}`, "$.contentSchema.minLength", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := loadTest(t, tc.original), loadTest(t, tc.modified)
			for _, pair := range []struct {
				l, r     *Document
				breaking bool
			}{{a, b, tc.forward}, {b, a, tc.reverse}} {
				diff, err := Compare(pair.l, pair.r, nil)
				require.NoError(t, err)
				require.Len(t, diff.GetAllChanges(), 1)
				change := diff.GetAllChanges()[0]
				require.Equal(t, tc.path, change.Path)
				require.Equal(t, pair.breaking, change.Breaking)
			}
		})
	}
}

func TestTypedLiteralBoundaryChanges(t *testing.T) {
	for _, tc := range []struct {
		name, original, modified, property, oldValue, newValue string
		breaking                                               bool
	}{
		{"empty enum", `{}`, `{"enum":[]}`, "enum", "", "[]", true},
		{"drop empty enum", `{"enum":[]}`, `{}`, "enum", "[]", "", false},
		{"null default", `{"default":null}`, `{"default":"null"}`, "default", "null", `"null"`, false},
		{"large exponent value", `{"const":1.0e-425}`, `{"const":2.0e-425}`, "const", "1.0e-425", "2.0e-425", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Compare(loadTest(t, tc.original), loadTest(t, tc.modified), nil)
			require.NoError(t, err)
			require.Len(t, result.GetAllChanges(), 1)
			ch := result.GetAllChanges()[0]
			require.Equal(t, tc.property, ch.Property)
			require.Equal(t, tc.oldValue, ch.OriginalEncoded)
			require.Equal(t, tc.newValue, ch.NewEncoded)
			require.Equal(t, tc.breaking, ch.Breaking)
		})
	}
}

func TestResourceAtAnExistingPropertyChangesIdentity(t *testing.T) {
	const first = `{"$id":"urn:first","type":"string"}`
	const second = `{"$id":"urn:second","type":"number"}`
	for _, alternative := range []struct{ name, schema string }{
		{"another address", second}, {"inline", `{"type":"number"}`},
	} {
		t.Run(alternative.name, func(t *testing.T) {
			const shape = `{"properties":{"x":%s,"y":%s}}`
			before := loadTest(t, fmt.Sprintf(shape, first, alternative.schema))
			after := loadTest(t, fmt.Sprintf(shape, alternative.schema, first))
			result, err := Compare(before, after, nil)
			require.NoError(t, err)
			edits := result.GetAllChanges()
			require.Len(t, edits, 2)
			for i, path := range []string{"$.properties.x", "$.properties.y"} {
				ch := edits[i]
				require.Equal(t, path, ch.Path)
				require.Equal(t, "customer.yaml", ch.SchemaResource)
				require.Equal(t, wc.Modified, ch.ChangeType)
				require.True(t, ch.Breaking)
			}
			if alternative.schema == second {
				require.Equal(t, "urn:first", edits[0].Original)
				require.Equal(t, "urn:second", edits[0].New)
			} else {
				require.Contains(t, edits[0].OriginalEncoded, "urn:first")
				require.Contains(t, edits[0].NewEncoded, "number")
			}
		})
	}
}

// An editor can change a source node after Load. An invalid edit must fail the
// whole comparison, including when the edited constraint is nested.
func TestInvalidInMemoryDependencyStopsComparison(t *testing.T) {
	for _, tc := range []struct {
		name, schema string
		locate       func(*yaml.Node) *yaml.Node
	}{
		{"in property", `{"properties":{"payment":{"dependentRequired":{"card":["issuer"]}}}}`,
			func(root *yaml.Node) *yaml.Node { return member(member(root, "properties"), "payment") }},
		{"in conjunct", `{"allOf":[{"dependentRequired":{"card":["issuer"]}}]}`,
			func(root *yaml.Node) *yaml.Node { return member(root, "allOf").Content[0] }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			good, bad := loadTest(t, tc.schema), loadTest(t, tc.schema)
			field := member(member(tc.locate(bad.Root.Node), "dependentRequired"), "card")
			field.Kind, field.Tag, field.Value, field.Content = yaml.ScalarNode, "!!str", "issuer", nil
			for _, pair := range [][2]*Document{{good, bad}, {bad, good}} {
				diff, err := Compare(pair[0], pair[1], nil)
				require.ErrorContains(t, err, "dependentRequired value must be an array")
				require.ErrorContains(t, err, "customer.yaml:1")
				require.Nil(t, diff)
			}
		})
	}
}

func TestCompareRejectsMalformedSchemaChildrenAfterLoad(t *testing.T) {
	for _, tc := range []struct {
		name, schema, path, diagnostic string
		edit                           func(*yaml.Node)
	}{
		{
			name: "property name",
			schema: `{
  "properties": {
    "1": { "type": "integer" }
  }
}`,
			path:       "$",
			diagnostic: "line 3: schema keys must be strings",
			edit: func(root *yaml.Node) {
				member(root, "properties").Content[0].Tag = "!!int"
			},
		},
		{
			name: "array applicator below a property",
			schema: `{
  "title":"Original",
  "properties":{
    "status":{
      "anyOf":[true, false]
    }
  }
}`,
			path:       "$.properties.status",
			diagnostic: "line 5: anyOf requires an array",
			edit: func(root *yaml.Node) {
				member(root, "title").Value = "Edited"
				choices := member(member(member(root, "properties"), "status"), "anyOf")
				choices.Kind, choices.Tag, choices.Content = yaml.MappingNode, "!!map", nil
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			good, bad := loadTest(t, tc.schema), loadTest(t, tc.schema)
			tc.edit(bad.Root.Node)
			for _, pair := range []struct {
				left, right *Document
				side        string
			}{{bad, good, "original"}, {good, bad, "modified"}} {
				diff, err := Compare(pair.left, pair.right, nil)
				require.ErrorContains(t, err, pair.side+" customer.yaml at "+tc.path)
				require.ErrorContains(t, err, tc.diagnostic)
				require.Nil(t, diff)
			}
		})
	}
}
