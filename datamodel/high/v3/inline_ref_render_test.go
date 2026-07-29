// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

const refRenderSpec = `openapi: 3.1.0
info:
  title: reference rendering
  version: 1.0.0
paths:
  /referenced:
    $ref: '#/components/pathItems/Target'
components:
  pathItems:
    Target:
      get:
        description: the referenced path item
        responses:
          '200':
            description: ok
  securitySchemes:
    Target:
      type: apiKey
      name: api_key
      in: header
    Referenced:
      $ref: '#/components/securitySchemes/Target'
`

func buildRefRenderDocument(t *testing.T) *Document {
	t.Helper()

	info, err := datamodel.ExtractSpecInfo([]byte(refRenderSpec))
	require.NoError(t, err)

	lowDoc, err := v3.CreateDocumentFromConfig(info, &datamodel.DocumentConfiguration{})
	require.NoError(t, err)

	return NewDocument(lowDoc)
}

// Inline rendering resolves a referenced object and renders its target in place. The high model
// carries no Reference of its own for these types, so the resolution happens through the low
// model rather than short-circuiting to a $ref node.
func TestPathItem_MarshalYAMLInline_ResolvesReference(t *testing.T) {
	doc := buildRefRenderDocument(t)

	pathItem := doc.Paths.PathItems.GetOrZero("/referenced")
	require.NotNil(t, pathItem)
	require.Empty(t, pathItem.Reference, "the high model holds no reference of its own here")
	require.True(t, pathItem.GoLow().IsReference(), "but the low model knows it is one")

	t.Run("without context", func(t *testing.T) {
		rendered, err := pathItem.MarshalYAMLInline()
		require.NoError(t, err)
		require.NotNil(t, rendered)

		out, err := yaml.Marshal(rendered)
		require.NoError(t, err)
		assert.Contains(t, string(out), "the referenced path item")
		assert.NotContains(t, string(out), "$ref", "the target is rendered in place")
	})

	t.Run("with context", func(t *testing.T) {
		rendered, err := pathItem.MarshalYAMLInlineWithContext(nil)
		require.NoError(t, err)
		require.NotNil(t, rendered)

		out, err := yaml.Marshal(rendered)
		require.NoError(t, err)
		assert.Contains(t, string(out), "the referenced path item")
		assert.NotContains(t, string(out), "$ref")
	})
}

func TestSecurityScheme_MarshalYAMLInline_ResolvesReference(t *testing.T) {
	doc := buildRefRenderDocument(t)

	scheme := doc.Components.SecuritySchemes.GetOrZero("Referenced")
	require.NotNil(t, scheme)
	require.Empty(t, scheme.Reference)
	require.True(t, scheme.GoLow().IsReference())

	t.Run("without context", func(t *testing.T) {
		rendered, err := scheme.MarshalYAMLInline()
		require.NoError(t, err)
		require.NotNil(t, rendered)

		out, err := yaml.Marshal(rendered)
		require.NoError(t, err)
		assert.Contains(t, string(out), "apiKey")
		assert.Contains(t, string(out), "api_key")
		assert.NotContains(t, string(out), "$ref")
	})

	t.Run("with context", func(t *testing.T) {
		rendered, err := scheme.MarshalYAMLInlineWithContext(nil)
		require.NoError(t, err)
		require.NotNil(t, rendered)

		out, err := yaml.Marshal(rendered)
		require.NoError(t, err)
		assert.Contains(t, string(out), "apiKey")
		assert.NotContains(t, string(out), "$ref")
	})
}

// RenderJSON converts the rendered YAML tree to JSON, which can fail on values YAML permits
// but JSON has no representation for. NaN is the reachable case: it is a legal YAML float and
// a legal mapping key, but json.Marshal refuses it. The error must surface rather than yield a
// half-written document.
func TestDocument_RenderJSON_PropagatesConversionError(t *testing.T) {
	spec := `openapi: 3.1.0
info:
  title: unconvertible extension
  version: 1.0.0
  x-weird:
    .nan: value
paths: {}
`
	info, err := datamodel.ExtractSpecInfo([]byte(spec))
	require.NoError(t, err)

	lowDoc, err := v3.CreateDocumentFromConfig(info, &datamodel.DocumentConfiguration{})
	require.NoError(t, err)

	rendered, err := NewDocument(lowDoc).RenderJSON("  ")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported value: NaN")
	assert.Empty(t, rendered)
}
