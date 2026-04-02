// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package low

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestNavigateReferenceFragment(t *testing.T) {
	spec := `components:
  schemas:
    Thing:
      type: string
list:
  - zero
  - one`

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(spec), &root))

	cases := []struct {
		name     string
		node     *yaml.Node
		fragment string
		value    string
		nilNode  bool
	}{
		{name: "nil root", node: nil, fragment: "#/components", nilNode: true},
		{name: "empty fragment", node: &root, fragment: "", nilNode: true},
		{name: "invalid prefix", node: &root, fragment: "components/schemas/Thing", nilNode: true},
		{name: "root fragment", node: &root, fragment: "#/", nilNode: true},
		{name: "skip empty segment", node: &root, fragment: "#/components//schemas/Thing/type", value: "string"},
		{name: "array index", node: &root, fragment: "#/list/1", value: "one"},
		{name: "missing map key", node: &root, fragment: "#/components/schemas/Missing", nilNode: true},
		{name: "scalar terminal", node: &root, fragment: "#/components/schemas/Thing/type/extra", nilNode: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			found := navigateReferenceFragment(tc.node, tc.fragment)
			if tc.nilNode {
				assert.Nil(t, found)
				return
			}
			require.NotNil(t, found)
			assert.Equal(t, tc.value, found.Value)
		})
	}
}

func TestLookupFragmentSequenceValue(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("- zero\n- one\n"), &root))
	seq := root.Content[0]

	require.NotNil(t, lookupFragmentSequenceValue(seq, "0"))
	assert.Equal(t, "zero", lookupFragmentSequenceValue(seq, "0").Value)
	assert.Nil(t, lookupFragmentSequenceValue(seq, "x"))
	assert.Nil(t, lookupFragmentSequenceValue(seq, "9"))
}
