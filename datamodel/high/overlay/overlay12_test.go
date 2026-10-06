// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package overlay

import (
	"context"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/low"
	lowoverlay "github.com/pb33f/libopenapi/datamodel/low/overlay"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"testing"
)

func parseOverlay12(t *testing.T, text string) *Overlay {
	t.Helper()
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(text), &node))
	var model lowoverlay.Overlay
	require.NoError(t, low.BuildModel(node.Content[0], &model))
	require.NoError(t, model.Build(context.Background(), nil, node.Content[0], nil))
	return NewOverlay(&model)
}

func TestOverlay12RoundTrip(t *testing.T) {
	text := `overlay: 1.2.0
$self: https://example.com/overlay.yaml
extends: api.yaml
info: {title: Shared, version: '1'}
components:
  x-source: docs
  actions:
    shared:
      description: Reusable description
      x-owner: docs
      fields:
        description: Field description
        remove: true
        copy: $.source
        update: null
    noop: {}
actions:
  - $ref: '#/components/actions/shared'
    target: $.info
    description: ''
    copy: ''
    remove: false
`
	ov := parseOverlay12(t, text)
	c := ov.Components
	assert.Equal(t, c.GoLow(), c.GoLowUntyped())
	reusable, ok := c.Actions.Get("shared")
	require.True(t, ok)
	assert.Equal(t, reusable.GoLow(), reusable.GoLowUntyped())
	componentBytes, err := c.Render()
	require.NoError(t, err)
	assert.Contains(t, string(componentBytes), "x-source: docs")
	actionBytes, err := reusable.Render()
	require.NoError(t, err)
	assert.Contains(t, string(actionBytes), "x-owner: docs")
	raw, err := ov.Render()
	require.NoError(t, err)
	assert.YAMLEq(t, text, string(raw))
	again := parseOverlay12(t, string(raw))
	assert.Equal(t, ov.GoLow().Hash(), again.GoLow().Hash())
	assert.True(t, again.Actions[0].HasRemove())
	assert.False(t, again.Actions[0].Remove)
	ov.Actions[0].SetRemove(true)
	assert.True(t, ov.Actions[0].Remove)
	ov.Actions[0].SetRemove(false)
	raw, err = ov.Actions[0].Render()
	require.NoError(t, err)
	assert.Contains(t, string(raw), "remove: false")
	empty := parseOverlay12(t, "components: {}")
	raw, err = empty.Components.Render()
	require.NoError(t, err)
	assert.YAMLEq(t, "{}", string(raw))
}

func TestOverlay12ExplicitEmptyURIs(t *testing.T) {
	for _, text := range []string{
		"$self: https://example.com/overlay.yaml\nextends: ''",
		"$self: ''\nextends: ''",
	} {
		ov := parseOverlay12(t, text)
		got, err := ov.ResolveExtends("https://example.com/overlay.yaml")
		require.NoError(t, err)
		assert.Equal(t, "https://example.com/overlay.yaml", got)
		raw, err := ov.Render()
		require.NoError(t, err)
		assert.YAMLEq(t, text, string(raw))
	}
	absent := parseOverlay12(t, "$self: https://example.com/overlay.yaml")
	got, err := absent.ResolveExtends("")
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestOverlay12EmptyReferenceAndTargetPreserved(t *testing.T) {
	text := "actions: [{$ref: '', target: ''}]"
	ov := parseOverlay12(t, text)
	raw, err := ov.Render()
	require.NoError(t, err)
	assert.YAMLEq(t, text, string(raw))
}
