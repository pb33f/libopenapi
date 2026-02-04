// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestSecurityScheme_MarshalYAML(t *testing.T) {
	ss := &SecurityScheme{
		Type:        "apiKey",
		Description: "this is a description",
		Name:        "superSecret",
		In:          "header",
		Scheme:      "https",
	}

	dat, _ := ss.Render()

	var idxNode yaml.Node
	_ = yaml.Unmarshal(dat, &idxNode)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	var n v3.SecurityScheme
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	r := NewSecurityScheme(&n)

	dat, _ = r.Render()

	desired := `type: apiKey
description: this is a description
name: superSecret
in: header
scheme: https`

	assert.Equal(t, desired, strings.TrimSpace(string(dat)))
}

func TestCreateSecuritySchemeRef(t *testing.T) {
	ref := "#/components/securitySchemes/BearerAuth"
	ss := CreateSecuritySchemeRef(ref)

	assert.True(t, ss.IsReference())
	assert.Equal(t, ref, ss.GetReference())
	assert.Nil(t, ss.GoLow())
}

func TestSecurityScheme_MarshalYAML_Reference(t *testing.T) {
	ss := CreateSecuritySchemeRef("#/components/securitySchemes/BearerAuth")

	node, err := ss.MarshalYAML()
	assert.NoError(t, err)

	yamlNode, ok := node.(*yaml.Node)
	assert.True(t, ok)
	assert.Equal(t, yaml.MappingNode, yamlNode.Kind)
	assert.Equal(t, 2, len(yamlNode.Content))
	assert.Equal(t, "$ref", yamlNode.Content[0].Value)
	assert.Equal(t, "#/components/securitySchemes/BearerAuth", yamlNode.Content[1].Value)
}

func TestSecurityScheme_MarshalYAMLInline_Reference(t *testing.T) {
	ss := CreateSecuritySchemeRef("#/components/securitySchemes/BearerAuth")

	node, err := ss.MarshalYAMLInline()
	assert.NoError(t, err)

	yamlNode, ok := node.(*yaml.Node)
	assert.True(t, ok)
	assert.Equal(t, "$ref", yamlNode.Content[0].Value)
}

func TestSecurityScheme_Reference_TakesPrecedence(t *testing.T) {
	// When both Reference and content are set, Reference should take precedence
	ss := &SecurityScheme{
		Reference:   "#/components/securitySchemes/foo",
		Description: "shouldBeIgnored",
	}

	assert.True(t, ss.IsReference())

	node, err := ss.MarshalYAML()
	assert.NoError(t, err)

	// Should render as $ref only, not full security scheme
	rendered, _ := yaml.Marshal(node)
	assert.Contains(t, string(rendered), "$ref")
	assert.NotContains(t, string(rendered), "shouldBeIgnored")
}

func TestSecurityScheme_Render_Reference(t *testing.T) {
	ss := CreateSecuritySchemeRef("#/components/securitySchemes/BearerAuth")

	rendered, err := ss.Render()
	assert.NoError(t, err)

	assert.Contains(t, string(rendered), "$ref")
	assert.Contains(t, string(rendered), "#/components/securitySchemes/BearerAuth")
}

func TestSecurityScheme_IsReference_False(t *testing.T) {
	ss := &SecurityScheme{
		Type: "apiKey",
		Name: "X-API-Key",
		In:   "header",
	}
	assert.False(t, ss.IsReference())
	assert.Equal(t, "", ss.GetReference())
}

func TestSecurityScheme_MarshalYAMLInlineWithContext(t *testing.T) {
	ss := &SecurityScheme{
		Type:        "apiKey",
		Description: "this is a description",
		Name:        "superSecret",
		In:          "header",
		Scheme:      "https",
	}

	ctx := base.NewInlineRenderContext()
	node, err := ss.MarshalYAMLInlineWithContext(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, node)

	dat, _ := yaml.Marshal(node)

	desired := `type: apiKey
description: this is a description
name: superSecret
in: header
scheme: https`

	assert.Equal(t, desired, strings.TrimSpace(string(dat)))
}

func TestSecurityScheme_MarshalYAMLInlineWithContext_Reference(t *testing.T) {
	ss := CreateSecuritySchemeRef("#/components/securitySchemes/BearerAuth")

	ctx := base.NewInlineRenderContext()
	node, err := ss.MarshalYAMLInlineWithContext(ctx)
	assert.NoError(t, err)

	yamlNode, ok := node.(*yaml.Node)
	assert.True(t, ok)
	assert.Equal(t, "$ref", yamlNode.Content[0].Value)
}

func TestBuildLowSecurityScheme_Success(t *testing.T) {
	yml := `type: apiKey
name: X-API-Key
in: header`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	assert.NoError(t, err)

	result, err := buildLowSecurityScheme(node.Content[0], nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "apiKey", result.Type.Value)
}

func TestBuildLowSecurityScheme_BuildNeverErrors(t *testing.T) {
	// SecurityScheme.Build never returns an error (no error return paths in the Build method)
	// This test verifies the success path
	yml := `type: http
scheme: bearer
bearerFormat: JWT`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	assert.NoError(t, err)

	result, err := buildLowSecurityScheme(node.Content[0], nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSecurityScheme_MarshalYAMLInline_ExternalRef(t *testing.T) {
	// Test that MarshalYAMLInline resolves external references properly
	yml := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
components:
  securitySchemes:
    BearerAuth:
      $ref: "#/components/securitySchemes/InternalAuth"
    InternalAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
paths: {}`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	config := index.CreateOpenAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(&idxNode, config)
	resolver := index.NewResolver(idx)
	idx.SetResolver(resolver)
	errs := resolver.Resolve()
	assert.Empty(t, errs)

	var n v3.SecurityScheme
	schemeNode := idxNode.Content[0].Content[5].Content[1].Content[1] // components.securitySchemes.BearerAuth
	_ = low.BuildModel(schemeNode, &n)
	_ = n.Build(context.Background(), nil, schemeNode, idx)

	ss := NewSecurityScheme(&n)

	result, err := ss.MarshalYAMLInline()
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSecurityScheme_MarshalYAMLInlineWithContext_ExternalRef(t *testing.T) {
	// Test that MarshalYAMLInlineWithContext resolves external references properly
	yml := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
components:
  securitySchemes:
    BearerAuth:
      $ref: "#/components/securitySchemes/InternalAuth"
    InternalAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
paths: {}`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	config := index.CreateOpenAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(&idxNode, config)
	resolver := index.NewResolver(idx)
	idx.SetResolver(resolver)
	errs := resolver.Resolve()
	assert.Empty(t, errs)

	var n v3.SecurityScheme
	schemeNode := idxNode.Content[0].Content[5].Content[1].Content[1] // components.securitySchemes.BearerAuth
	_ = low.BuildModel(schemeNode, &n)
	_ = n.Build(context.Background(), nil, schemeNode, idx)

	ss := NewSecurityScheme(&n)

	ctx := base.NewInlineRenderContext()
	result, err := ss.MarshalYAMLInlineWithContext(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}
