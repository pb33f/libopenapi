// Copyright 2026 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"github.com/pb33f/testify/require"
)

func TestCompareBooleanSchemas(t *testing.T) {
	for _, tc := range []struct {
		left, right string
		changed     bool
	}{
		{"true", "true", false}, {"false", "false", false},
		{"true", "false", true}, {"false", "true", true},
		{"false", "{}", true}, {"{}", "false", true},
		{"true", "type: string", true}, {"type: string", "true", true},
	} {
		t.Run(tc.left+" to "+tc.right, func(t *testing.T) {
			left := buildSchemaProxyFromYAML(t, tc.left)
			right := buildSchemaProxyFromYAML(t, tc.right)
			changes := CompareSchemas(left, right)
			if !tc.changed {
				require.Nil(t, changes)
				return
			}
			require.NotNil(t, changes)
			require.Equal(t, 1, changes.TotalChanges())
			require.Equal(t, 1, changes.TotalBreakingChanges())
			require.Equal(t, "schema", changes.Changes[0].Property)
			require.Equal(t, Modified, changes.Changes[0].ChangeType)
		})
	}
}

func TestCompareBooleanSchemaPropertyAndPolicy(t *testing.T) {
	left := buildSchemaProxyFromYAML(t, "type: object\nproperties:\n  options: true")
	right := buildSchemaProxyFromYAML(t, "type: object\nproperties:\n  options: false")
	changes := CompareSchemas(left, right)
	require.NotNil(t, changes)
	require.NotNil(t, changes.SchemaPropertyChanges["options"])
	require.Equal(t, 1, changes.TotalBreakingChanges())
	rules := &BreakingRulesConfig{Schema: &SchemaRules{Type: rule(false, false, false)}}
	SetActiveBreakingRulesConfig(rules)
	t.Cleanup(ResetActiveBreakingRulesConfig)
	changes = CompareSchemas(left, right)
	require.NotNil(t, changes)
	require.Equal(t, 1, changes.TotalChanges())
	require.Zero(t, changes.TotalBreakingChanges())
}
