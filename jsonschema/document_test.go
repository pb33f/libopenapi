// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/pb33f/libopenapi/datamodel"
	wc "github.com/pb33f/libopenapi/what-changed/model"
	"github.com/stretchr/testify/require"
)

func loadTest(t *testing.T, input string) *Document {
	t.Helper()
	d, err := Load([]byte(input), "customer.yaml", nil, true)
	require.NoError(t, err)
	t.Cleanup(d.Release)
	return d
}

func TestAuthoredSchemaCompatibility(t *testing.T) {
	for _, tc := range []struct {
		name, left, right, path string
		breaking                bool
	}{
		{"required", `{"type":"object","properties":{"id":{"type":"string"}}}`, `{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}`, "$.required", true},
		{"optional constraint", "{}", `{"properties":{"x":{"type":"string"}}}`, "$.properties.x", true},
		{"minContains", `{"contains":{"type":"integer"},"minContains":2}`, `{"contains":{"type":"integer"},"minContains":3}`, "$.minContains", true},
		{"high precision", `{"minimum":1.2345678901234567890}`, `{"minimum":1.2345678901234567891}`, "$.minimum", true},
		{"ambiguous negation", `{"not":{"pattern":"a"}}`, `{"not":{"pattern":"b"}}`, "$.not.pattern", true},
		{"dependent requirements", `{"dependentRequired":{"x":["a"]}}`, `{"dependentRequired":{"x":["a","b"]}}`, "$.dependentRequired.x", true},
		{"reordered conjunction", `{"allOf":[{"type":"boolean"},{"description":"one"}]}`, `{"allOf":[{"description":"two"},{"type":"boolean"}]}`, "$.allOf[0].description", false},
		{"widen boolean", "false", "true", "$", false}, {"narrow boolean", "true", "false", "$", true},
		{"tighten length", `{"minLength":2}`, `{"minLength":3}`, "$.minLength", true},
		{"widen length", `{"minLength":3}`, `{"minLength":2}`, "$.minLength", false},
		{"annotation", `{"description":"old"}`, `{"description":"new"}`, "$.description", false},
		{"static anchor", `{"$anchor":"old"}`, `{"$anchor":"new"}`, `$["$anchor"]`, true},
		{"published definition", `{"$defs":{"a.b":{"minLength":1}}}`, `{"$defs":{"a.b":{"minLength":2}}}`, `$["$defs"]["a.b"].minLength`, true},
		{"reference siblings", `{"$ref":"#","minLength":3}`, `{"$ref":"#","minLength":2}`, "$.minLength", false},
		{"negated widening", `{"not":{"minLength":3}}`, `{"not":{"minLength":2}}`, "$.not.minLength", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			left, right := loadTest(t, tc.left), loadTest(t, tc.right)
			report, err := Compare(left, right, nil)
			require.NoError(t, err)
			require.NotNil(t, report)
			all := report.GetAllChanges()
			require.Len(t, all, 1)
			require.Equal(t, tc.path, all[0].Path)
			require.Equal(t, tc.breaking, all[0].Breaking)
			require.Equal(t, "customer.yaml", all[0].SchemaResource)
		})
	}
}

func TestExplicitPolicyOnWidening(t *testing.T) {
	rule := true
	diff, err := Compare(loadTest(t, `{"minLength":3}`), loadTest(t, `{"minLength":2}`), &wc.BreakingRulesConfig{Schema: &wc.SchemaRules{MinLength: &wc.BreakingChangeRule{Modified: &rule}}})
	require.NoError(t, err)
	require.Equal(t, 1, diff.TotalBreakingChanges())
}

func TestLoadReferencesFromEachSnapshot(t *testing.T) {
	root := []byte(`{"properties":{"billing":{"$ref":"amount.yaml#amount"},"shipping":{"$ref":"amount.yaml#amount"}}}`)
	load := func(limit int) *Document {
		cfg := datamodel.NewDocumentConfiguration()
		cfg.BasePath = "/snapshot"
		cfg.LocalFS = fstest.MapFS{"amount.yaml": {Data: []byte(fmt.Sprintf("$id: urn:amount\n$anchor: amount\nminLength: %d\n", limit))}}
		d, err := Load(root, "customer.json", cfg, true)
		require.NoError(t, err)
		t.Cleanup(d.Release)
		return d
	}
	a, b := load(2), load(3)
	require.Len(t, a.References, 2)
	require.Same(t, a.References[0].To, a.References[1].To)
	diff, err := Compare(a, b, nil)
	require.NoError(t, err)
	require.Len(t, diff.GetAllChanges(), 1)
	ch := diff.GetAllChanges()[0]
	require.Equal(t, "urn:amount", ch.SchemaResource)
	require.Equal(t, "amount.yaml", ch.Context.OriginalDocument)
	require.Equal(t, "amount.yaml", ch.Context.ModifiedDocument)
}

func TestEmbeddedResourceOccurrence(t *testing.T) {
	left := loadTest(t, `{"properties":{"x":{"$id":"urn:field","type":"string"}}}`)
	right := loadTest(t, `{"properties":{"y":{"$id":"urn:field","type":"string"}}}`)
	changes, err := Compare(left, right, nil)
	require.NoError(t, err)
	require.Equal(t, 2, changes.TotalChanges())
	require.Equal(t, "$.properties.x", changes.GetAllChanges()[0].Path)
	node, ok := left.Root.Source.Index.GetNode(left.Root.Node.Line, left.Root.Node.Column)
	require.True(t, ok)
	require.Same(t, left.Root.Node, node)
}
