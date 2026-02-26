// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package expression

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestResolveComponents_WithDeepTail(t *testing.T) {
	ctx := &Context{
		Components: &ComponentsContext{
			Inputs: map[string]any{
				"i1": map[string]any{
					"inner": map[string]any{
						"value": "ok",
					},
				},
			},
		},
	}

	v, err := EvaluateString("$components.inputs.i1.inner.value", ctx)
	require.NoError(t, err)
	assert.Equal(t, "ok", v)
}

func TestResolveDeepValue_PropertyMissing(t *testing.T) {
	_, err := resolveDeepValue(map[string]any{"a": 1}, "b", "parameters", "p1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "property")
}

func TestResolveDeepValue_CannotTraverse(t *testing.T) {
	_, err := resolveDeepValue("x", "b", "parameters", "p1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot traverse")
}

func TestYAMLNodeToValue_DefaultCase(t *testing.T) {
	n := &yaml.Node{Kind: yaml.AliasNode}
	out := yamlNodeToValue(n)
	assert.Same(t, n, out)
}

func TestParse_ResponseUnknownBranch(t *testing.T) {
	_, err := Parse("$random")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown expression")
}

func TestParseEmbedded_InvalidEmbeddedExpression(t *testing.T) {
	_, err := ParseEmbedded("prefix {$badExpression}")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid embedded expression")
}
