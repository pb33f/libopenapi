// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package overlay

import (
	"context"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"strings"
	"testing"
)

const overlay12Model = `overlay: 1.2.0
$self: https://example.com/overlay.yaml
info: {title: Reuse, version: '1'}
components:
  x-source: docs
  actions:
    shared:
      description: Shared metadata
      x-owner: docs
      fields:
        description: Action description
        update: {description: New}
        remove: true
        x-field: example
actions:
  - $ref: '#/components/actions/shared'
    target: $.info
    remove: false
`

func buildOverlay12(t *testing.T, text string) (*Overlay, error) {
	t.Helper()
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(text), &node))
	ov := new(Overlay)
	if err := low.BuildModel(node.Content[0], ov); err != nil {
		return nil, err
	}
	err := ov.Build(context.Background(), nil, node.Content[0], nil)
	return ov, err
}

func TestOverlay12ModelsAndHashes(t *testing.T) {
	ov, err := buildOverlay12(t, overlay12Model)
	require.NoError(t, err)
	assert.Equal(t, "$self", ov.Self.KeyNode.Value)
	c := ov.Components.Value
	require.NotNil(t, c)
	assert.Equal(t, "components", c.GetKeyNode().Value)
	assert.Equal(t, ov.Components.ValueNode, c.GetRootNode())
	assert.Equal(t, context.Background(), c.GetContext())
	assert.Nil(t, c.GetIndex())
	assert.NotNil(t, c.FindExtension("x-source"))
	assert.Equal(t, c.Extensions, c.GetExtensions())
	entry := low.FindItemInOrderedMap("shared", c.Actions.Value)
	require.NotNil(t, entry)
	reusable := entry.Value
	assert.Equal(t, "shared", reusable.GetKeyNode().Value)
	assert.Equal(t, entry.ValueNode, reusable.GetRootNode())
	assert.Equal(t, context.Background(), reusable.GetContext())
	assert.Nil(t, reusable.GetIndex())
	assert.NotNil(t, reusable.FindExtension("x-owner"))
	assert.Equal(t, reusable.Extensions, reusable.GetExtensions())
	assert.True(t, reusable.Fields.Value.Remove.Value)
	assert.Equal(t, "#/components/actions/shared", ov.Actions.Value[0].Value.Ref.Value)
	require.NotNil(t, ov.Actions.Value[0].Value.Remove.KeyNode)
	before := ov.Hash()
	for _, change := range [][2]string{
		{"https://example.com/overlay.yaml", "https://example.com/other.yaml"},
		{"Shared metadata", "Other metadata"},
		{"Action description", "Different action"},
		{"description: New", "description: Changed"},
		{"remove: false", "remove: true"},
		{"#/components/actions/shared", "#/components/actions/other"},
		{"x-source: docs", "x-source: other"},
		{"x-owner: docs", "x-owner: other"},
		{"x-field: example", "x-field: other"},
	} {
		modified, err := buildOverlay12(t, strings.Replace(overlay12Model, change[0], change[1], 1))
		require.NoError(t, err)
		assert.NotEqual(t, before, modified.Hash(), change[0])
	}
	again, err := buildOverlay12(t, overlay12Model)
	require.NoError(t, err)
	assert.Equal(t, before, again.Hash())
}

func TestOverlay12MalformedModels(t *testing.T) {
	for _, text := range []string{
		"components: {actions: {a: {}, a: {}}}", "components: {actions: {? [a,b]: {}}}",
		"overlay: []", "extends: {}", "info: {title: []}", "info: {version: []}", "info: {description: []}",
		"actions: [{remove: !!bool invalid}]", "actions: [{target: []}]", "actions: [{description: []}]", "actions: [{copy: []}]", "actions: [{remove: 'false'}]",
		"components: {actions: {a: {description: []}}}",
		"[]", "null", "$self: {}", "components: []", "components: {actions: []}",
		"components: {actions: {a: false}}", "components: {actions: {a: {fields: []}}}",
		"actions: [false]", "actions: [{$ref: {}}]", "info: []",
	} {
		t.Run(text, func(t *testing.T) { _, err := buildOverlay12(t, text); require.Error(t, err) })
	}
}

func TestComponentsHashOrderIndependent(t *testing.T) {
	a, err := buildOverlay12(t, "components: {actions: {a: {fields: {remove: true}}, b: {fields: {copy: '$.info'}}}}")
	require.NoError(t, err)
	b, err := buildOverlay12(t, "components: {actions: {b: {fields: {copy: '$.info'}}, a: {fields: {remove: true}}}}")
	require.NoError(t, err)
	assert.Equal(t, a.Hash(), b.Hash())
	empty, err := buildOverlay12(t, "components: {}")
	require.NoError(t, err)
	assert.NotZero(t, empty.Hash())
}
