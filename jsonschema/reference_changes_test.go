// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"fmt"
	"strings"
	"testing"

	wc "github.com/pb33f/libopenapi/what-changed/model"
	"github.com/pb33f/testify/require"
)

func TestUnchangedReferenceRetargetsAfterResourceMoves(t *testing.T) {
	snapshot := func(a, b string) *Document {
		return loadMembers(t, t.TempDir(), map[string][]byte{
			"a.yml":   []byte("$id: " + a + "\nproperties: {x: true}"),
			"b.yml":   []byte("$id: " + b + "\nproperties: {x: true}"),
			"use.yml": []byte("prefixItems:\n  - properties:\n      '0':\n        $ref: a.yml#/properties/x"),
		})
	}
	left := snapshot("urn:one", "urn:two")
	right := snapshot("urn:two", "urn:one")
	changes, err := Compare(left, right, nil)
	require.NoError(t, err)
	require.NotNil(t, changes)
	require.Len(t, changes.GetAllChanges(), 1)
	c := changes.GetAllChanges()[0]
	require.Equal(t, "use.yml", c.SchemaResource)
	require.Equal(t, `$.prefixItems[0].properties["0"]["$ref"]`, c.Path)
	require.Equal(t, "$ref", c.Property)
	require.Equal(t, "urn:one#/properties/x", c.Original)
	require.Equal(t, "urn:two#/properties/x", c.New)
	require.Equal(t, "a.yml#/properties/x", c.Reference)
	require.True(t, c.Breaking)
	require.Equal(t, 4, *c.Context.OriginalLine)
	require.Equal(t, "use.yml", c.Context.ModifiedDocument)
}

func TestReferenceFollowsUnorderedOperand(t *testing.T) {
	for _, keyword := range []string{"anyOf", "allOf", "oneOf"} {
		t.Run(keyword, func(t *testing.T) {
			const use = `{"properties":{"field":{"$ref":"a.json"}}}`
			const other = `{"type":"boolean"}`
			snapshot := func(first, second, a, b string) *Document {
				return loadMembers(t, t.TempDir(), map[string][]byte{
					"a.json":   []byte(fmt.Sprintf(`{"$id":%q}`, a)),
					"b.json":   []byte(fmt.Sprintf(`{"$id":%q}`, b)),
					"use.json": []byte(fmt.Sprintf(`{%q:[%s,%s]}`, keyword, first, second)),
				})
			}
			left := snapshot(use, other, "urn:A", "urn:B")
			for _, tc := range []struct {
				right    *Document
				retarget bool
			}{
				{snapshot(other, use, "urn:A", "urn:B"), false},
				{snapshot(other, use, "urn:B", "urn:A"), true},
			} {
				diff, err := Compare(left, tc.right, nil)
				require.NoError(t, err)
				if !tc.retarget {
					require.Nil(t, diff)
					continue
				}
				require.NotNil(t, diff)
				require.Len(t, diff.GetAllChanges(), 1)
				change := diff.GetAllChanges()[0]
				require.Equal(t, "$."+keyword+`[1].properties.field["$ref"]`, change.Path)
				require.Equal(t, "use.json", change.SchemaResource)
				require.Equal(t, "urn:A", change.Original)
				require.Equal(t, "urn:B", change.New)
				require.True(t, change.Breaking)
			}
		})
	}
}

func TestEmbeddedResourceKeepsContainingContext(t *testing.T) {
	for _, tc := range []struct {
		name, schema, path string
		breaking           bool
	}{
		{"negated", `{"not":{"$id":"urn:shared","minimum":%d}}`, "$.minimum", true},
		{"negated descendant", `{"not":{"$id":"urn:shared","properties":{"v":{"minimum":%d}}}}`, "$.properties.v.minimum", true},
		{"bounded contains", `{"maxContains":1,"contains":{"$id":"urn:shared","minimum":%d}}`, "$.minimum", true},
		{"conditional", `{"if":{"$id":"urn:shared","minimum":%d},"then":false}`, "$.minimum", true},
		{"positive", `{"properties":{"v":{"$id":"urn:shared","minimum":%d}}}`, "$.minimum", false},
		{"double negative", `{"not":{"not":{"$id":"urn:shared","minimum":%d}}}`, "$.minimum", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := loadTest(t, fmt.Sprintf(tc.schema, 4)), loadTest(t, fmt.Sprintf(tc.schema, 3))
			require.Empty(t, a.References)
			diff, err := Compare(a, b, nil)
			require.NoError(t, err)
			require.Len(t, diff.GetAllChanges(), 1)
			ch := diff.GetAllChanges()[0]
			require.Equal(t, "urn:shared", ch.SchemaResource)
			require.Equal(t, tc.path, ch.Path)
			require.Equal(t, tc.breaking, ch.Breaking)
		})
	}
}

func TestRemovedOperandDoesNotInheritSurvivingReference(t *testing.T) {
	snapshot := func(branches, a, b string) *Document {
		return loadMembers(t, t.TempDir(), map[string][]byte{
			"a.yml": []byte("$id: " + a), "b.yml": []byte("$id: " + b),
			"use.yml": []byte("anyOf:\n" + branches),
		})
	}
	const removed = "  - {$ref: a.yml, title: removed}\n"
	const kept = "  - {$ref: a.yml, title: kept}\n"
	diff, err := Compare(snapshot(removed+kept, "urn:A", "urn:B"), snapshot(kept, "urn:B", "urn:A"), nil)
	require.NoError(t, err)
	require.Len(t, diff.GetAllChanges(), 2)
	require.Equal(t, wc.ObjectRemoved, diff.GetAllChanges()[0].ChangeType)
	retarget := diff.GetAllChanges()[1]
	require.Equal(t, `$.anyOf[0]["$ref"]`, retarget.Path)
	require.Equal(t, "urn:B", retarget.New)
	require.True(t, retarget.Breaking)
}

func TestEmbeddedMembershipUsesContainingKeywordOverride(t *testing.T) {
	empty := loadTest(t, "{}")
	for _, keyword := range []string{"$defs", "properties"} {
		for _, override := range []bool{false, true} {
			rule := &wc.BreakingChangeRule{Added: &override, Removed: &override}
			config := &wc.BreakingRulesConfig{Schema: &wc.SchemaRules{}}
			if keyword == "$defs" {
				config.Schema.Defs = rule
			} else {
				config.Schema.Properties = rule
			}
			member := loadTest(t, fmt.Sprintf(`{%q:{"v":{"$id":"urn:defined","type":"string"}}}`, keyword))
			for _, pair := range [][2]*Document{{empty, member}, {member, empty}} {
				diff, err := Compare(pair[0], pair[1], config)
				require.NoError(t, err)
				require.Len(t, diff.GetAllChanges(), 1)
				ch := diff.GetAllChanges()[0]
				require.Equal(t, "urn:defined", ch.SchemaResource)
				require.Equal(t, "$", ch.Path)
				require.Equal(t, "schema", ch.Property)
				require.Equal(t, override, ch.Breaking, "%s %v", keyword, ch.ChangeType)
			}
		}
	}
}

func TestReferenceAnchorRetarget(t *testing.T) {
	snapshot := func(a, b string) *Document {
		return loadMembers(t, t.TempDir(), map[string][]byte{
			"owner.yml": []byte("$id: urn:owner\n$defs:\n  'a/b': {$anchor: " + a + "}\n  b: {$anchor: " + b + "}"),
			"use.yml":   []byte("$dynamicRef: urn:owner#entry"),
		})
	}
	changes, err := Compare(snapshot("entry", "other"), snapshot("other", "entry"), nil)
	require.NoError(t, err)
	var ref *wc.Change
	for _, change := range changes.GetAllChanges() {
		if change.Property == "$dynamicRef" {
			require.Nil(t, ref)
			ref = change
		}
	}
	require.NotNil(t, ref)
	require.Equal(t, "urn:owner#/$defs/a~1b", ref.Original)
	require.Equal(t, "urn:owner#/$defs/b", ref.New)
	require.True(t, ref.Breaking)
}

func TestReferenceSurvivesInlineResourceBoundary(t *testing.T) {
	before := loadTest(t, `{"$defs":{"v":{"type":"string"}},"$ref":"#/$defs/v"}`)
	after := loadTest(t, `{"$defs":{"v":{"$id":"urn:v","type":"string"}},"$ref":"#/$defs/v"}`)
	for _, pair := range [][2]*Document{{before, after}, {after, before}} {
		changes, err := Compare(pair[0], pair[1], nil)
		require.NoError(t, err)
		require.Len(t, changes.GetAllChanges(), 1)
		require.Equal(t, "$id", changes.GetAllChanges()[0].Property)
	}
}

func TestUnchangedReferenceRetargetsAtNewEmbeddedBase(t *testing.T) {
	snapshot := func(id string) *Document {
		return loadMembers(t, t.TempDir(), map[string][]byte{
			"source.yml":     []byte("properties:\n  use:\n    $ref: target.yml" + id),
			"target.yml":     []byte("$id: urn:first"),
			"dir/target.yml": []byte("$id: urn:second"),
		})
	}
	changes, err := Compare(snapshot(""), snapshot("\n    $id: dir/use.yml"), nil)
	require.NoError(t, err)
	require.Len(t, changes.GetAllChanges(), 2)
	c := changes.GetAllChanges()[1]
	require.Equal(t, "$ref", c.Property)
	require.Equal(t, "dir/use.yml", c.SchemaResource)
	require.Equal(t, `$["$ref"]`, c.Path)
	require.Equal(t, "urn:first", c.Original)
	require.Equal(t, "urn:second", c.New)
	require.True(t, c.Breaking)
}

func TestReferenceUseChangesCompatibilityAtDefinition(t *testing.T) {
	for _, test := range []struct {
		name, use string
		breaking  bool
	}{
		{"ordinary", `{"$ref":"#/$defs/T"}`, false},
		{"negative", `{"not":{"$ref":"#/$defs/T"}}`, true},
		{"exclusive", `{"oneOf":[{"$ref":"#/$defs/T"},{}]}`, true},
		{"condition", `{"if":{"$ref":"#/$defs/T"},"then":false}`, true},
		{"bounded contains", `{"contains":{"$ref":"#/$defs/T"},"maxContains":1}`, true},
		{"transitive", `{"not":{"$ref":"#/$defs/A"}}`, true},
		{"nested positive", `{"not":{"not":{"$ref":"#/$defs/T"}}}`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			schema := strings.TrimSuffix(test.use, "}") + `,"$defs": {
				"T": {"minimum": %d, "description": "%s"},
				"A": {"$ref": "#/$defs/T", "properties": {"next": {"$ref": "#/$defs/A"}}}
			}}`
			a := loadTest(t, fmt.Sprintf(schema, 4, "old"))
			b := loadTest(t, fmt.Sprintf(schema, 1, "new"))
			changes, err := Compare(a, b, nil)
			require.NoError(t, err)
			require.Len(t, changes.GetAllChanges(), 2)
			for _, c := range changes.GetAllChanges() {
				require.Equal(t, `$["$defs"].T.`+c.Property, c.Path)
				if c.Property == "minimum" {
					require.Equal(t, test.breaking, c.Breaking)
				} else {
					require.Equal(t, "description", c.Property)
					require.False(t, c.Breaking)
				}
			}
		})
	}
}

func TestReferenceUsageCrossesResources(t *testing.T) {
	snapshot := func(limit int) *Document {
		return loadMembers(t, t.TempDir(), map[string][]byte{
			"use.yml":    []byte("not: {$ref: urn:intermediate}"),
			"shared.yml": []byte(fmt.Sprintf("$id: urn:target\nproperties: {v: {minimum: %d}}", limit)),
			"middle.yml": []byte("$id: urn:intermediate\n$ref: urn:target"),
		})
	}
	changes, err := Compare(snapshot(5), snapshot(2), nil)
	require.NoError(t, err)
	require.Len(t, changes.GetAllChanges(), 1)
	c := changes.GetAllChanges()[0]
	require.Equal(t, "urn:target", c.SchemaResource)
	require.Equal(t, "$.properties.v.minimum", c.Path)
	require.True(t, c.Breaking)
}
