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
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestParameter_MarshalYAML(t *testing.T) {
	ext := orderedmap.New[string, *yaml.Node]()
	ext.Set("x-burgers", utils.CreateStringNode("why not?"))

	explode := true
	param := Parameter{
		Name:          "chicken",
		In:            "nuggets",
		Description:   "beefy",
		Deprecated:    true,
		Style:         "simple",
		Explode:       &explode,
		AllowReserved: true,
		Example:       utils.CreateStringNode("example"),
		Examples: orderedmap.ToOrderedMap(map[string]*base.Example{
			"example": {Value: utils.CreateStringNode("example")},
		}),
		Extensions: ext,
	}

	rend, _ := param.Render()

	desired := `name: chicken
in: nuggets
description: beefy
deprecated: true
style: simple
explode: true
allowReserved: true
example: example
examples:
    example:
        value: example
x-burgers: why not?`

	assert.Equal(t, desired, strings.TrimSpace(string(rend)))
}

func TestParameter_MarshalYAMLInline(t *testing.T) {
	ext := orderedmap.New[string, *yaml.Node]()
	ext.Set("x-burgers", utils.CreateStringNode("why not?"))

	explode := true
	param := Parameter{
		Name:          "chicken",
		In:            "nuggets",
		Description:   "beefy",
		Deprecated:    true,
		Style:         "simple",
		Explode:       &explode,
		AllowReserved: true,
		Example:       utils.CreateStringNode("example"),
		Examples: orderedmap.ToOrderedMap(map[string]*base.Example{
			"example": {Value: utils.CreateStringNode("example")},
		}),
		Extensions: ext,
	}

	rend, _ := param.RenderInline()

	desired := `name: chicken
in: nuggets
description: beefy
deprecated: true
style: simple
explode: true
allowReserved: true
example: example
examples:
    example:
        value: example
x-burgers: why not?`

	assert.Equal(t, desired, strings.TrimSpace(string(rend)))
}

func TestParameter_IsExploded(t *testing.T) {
	explode := true
	param := Parameter{
		Explode: &explode,
	}

	assert.True(t, param.IsExploded())

	explode = false
	param = Parameter{
		Explode: &explode,
	}

	assert.False(t, param.IsExploded())

	param = Parameter{}

	assert.False(t, param.IsExploded())
}

func TestParameter_IsDefaultFormEncoding(t *testing.T) {
	param := Parameter{}
	assert.True(t, param.IsDefaultFormEncoding())

	param = Parameter{Style: "form"}
	assert.True(t, param.IsDefaultFormEncoding())

	explode := false
	param = Parameter{
		Explode: &explode,
	}
	assert.False(t, param.IsDefaultFormEncoding())

	explode = true
	param = Parameter{
		Explode: &explode,
	}
	assert.True(t, param.IsDefaultFormEncoding())

	param = Parameter{
		Explode: &explode,
		Style:   "simple",
	}
	assert.False(t, param.IsDefaultFormEncoding())
}

func TestParameter_IsDefaultHeaderEncoding(t *testing.T) {
	param := Parameter{}
	assert.True(t, param.IsDefaultHeaderEncoding())

	param = Parameter{Style: "simple"}
	assert.True(t, param.IsDefaultHeaderEncoding())

	explode := false
	param = Parameter{
		Explode: &explode,
		Style:   "simple",
	}
	assert.True(t, param.IsDefaultHeaderEncoding())

	explode = true
	param = Parameter{
		Explode: &explode,
		Style:   "simple",
	}
	assert.False(t, param.IsDefaultHeaderEncoding())

	explode = false
	param = Parameter{
		Explode: &explode,
		Style:   "form",
	}
	assert.False(t, param.IsDefaultHeaderEncoding())
}

func TestParameter_IsDefaultPathEncoding(t *testing.T) {
	param := Parameter{}
	assert.True(t, param.IsDefaultPathEncoding())
}

func TestParameter_Examples(t *testing.T) {
	yml := `examples:
    pbjBurger:
        summary: A horrible, nutty, sticky mess.
        value:
            name: Peanut And Jelly
            numPatties: 3
    cakeBurger:
        summary: A sickly, sweet, atrocity
        value:
            name: Chocolate Cake Burger
            numPatties: 5`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	var n v3.Parameter
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	r := NewParameter(&n)

	assert.Equal(t, 2, orderedmap.Len(r.Examples))
}

func TestParameter_Examples_NotFromSchema(t *testing.T) {
	yml := `schema:
  type: string
  examples:
    - example 1
    - example 2
    - example 3`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	var n v3.Parameter
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	r := NewParameter(&n)

	assert.Equal(t, 0, orderedmap.Len(r.Examples))
}

func TestCreateParameterRef(t *testing.T) {
	ref := "#/components/parameters/limitParam"
	p := CreateParameterRef(ref)

	assert.True(t, p.IsReference())
	assert.Equal(t, ref, p.GetReference())
	assert.Nil(t, p.GoLow())
}

func TestParameter_MarshalYAML_Reference(t *testing.T) {
	p := CreateParameterRef("#/components/parameters/limitParam")

	node, err := p.MarshalYAML()
	assert.NoError(t, err)

	yamlNode, ok := node.(*yaml.Node)
	assert.True(t, ok)
	assert.Equal(t, yaml.MappingNode, yamlNode.Kind)
	assert.Equal(t, 2, len(yamlNode.Content))
	assert.Equal(t, "$ref", yamlNode.Content[0].Value)
	assert.Equal(t, "#/components/parameters/limitParam", yamlNode.Content[1].Value)
}

func TestParameter_MarshalYAMLInline_Reference(t *testing.T) {
	p := CreateParameterRef("#/components/parameters/limitParam")

	node, err := p.MarshalYAMLInline()
	assert.NoError(t, err)

	yamlNode, ok := node.(*yaml.Node)
	assert.True(t, ok)
	assert.Equal(t, "$ref", yamlNode.Content[0].Value)
}

func TestParameter_Reference_TakesPrecedence(t *testing.T) {
	// When both Reference and content are set, Reference should take precedence
	p := &Parameter{
		Reference: "#/components/parameters/foo",
		Name:      "shouldBeIgnored",
		In:        "query",
	}

	assert.True(t, p.IsReference())

	node, err := p.MarshalYAML()
	assert.NoError(t, err)

	// Should render as $ref only, not full parameter
	rendered, _ := yaml.Marshal(node)
	assert.Contains(t, string(rendered), "$ref")
	assert.NotContains(t, string(rendered), "shouldBeIgnored")
}

func TestParameter_Render_Reference(t *testing.T) {
	p := CreateParameterRef("#/components/parameters/limitParam")

	rendered, err := p.Render()
	assert.NoError(t, err)

	assert.Contains(t, string(rendered), "$ref")
	assert.Contains(t, string(rendered), "#/components/parameters/limitParam")
}

func TestParameter_IsReference_False(t *testing.T) {
	p := &Parameter{
		Name: "limit",
		In:   "query",
	}
	assert.False(t, p.IsReference())
	assert.Equal(t, "", p.GetReference())
}

func TestParameter_Integration_MixedRefAndInline(t *testing.T) {
	// Build an operation with both ref and inline parameters
	op := &Operation{
		OperationId: "listUsers",
		Parameters: []*Parameter{
			CreateParameterRef("#/components/parameters/limitParam"),
			{
				Name:        "status",
				In:          "query",
				Description: "Filter by status",
			},
		},
	}

	rendered, err := op.Render()
	assert.NoError(t, err)

	output := string(rendered)
	assert.Contains(t, output, "$ref: '#/components/parameters/limitParam'")
	assert.Contains(t, output, "name: status")
	assert.Contains(t, output, "in: query")
}

func TestParameter_MarshalYAMLInlineWithContext(t *testing.T) {
	ext := orderedmap.New[string, *yaml.Node]()
	ext.Set("x-burgers", utils.CreateStringNode("why not?"))

	explode := true
	param := Parameter{
		Name:          "chicken",
		In:            "nuggets",
		Description:   "beefy",
		Deprecated:    true,
		Style:         "simple",
		Explode:       &explode,
		AllowReserved: true,
		Example:       utils.CreateStringNode("example"),
		Examples: orderedmap.ToOrderedMap(map[string]*base.Example{
			"example": {Value: utils.CreateStringNode("example")},
		}),
		Extensions: ext,
	}

	ctx := base.NewInlineRenderContext()
	node, err := param.MarshalYAMLInlineWithContext(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, node)

	rend, _ := yaml.Marshal(node)

	desired := `name: chicken
in: nuggets
description: beefy
deprecated: true
style: simple
explode: true
allowReserved: true
example: example
examples:
    example:
        value: example
x-burgers: why not?`

	assert.Equal(t, desired, strings.TrimSpace(string(rend)))
}

func TestParameter_MarshalYAMLInlineWithContext_Reference(t *testing.T) {
	p := CreateParameterRef("#/components/parameters/limitParam")

	ctx := base.NewInlineRenderContext()
	node, err := p.MarshalYAMLInlineWithContext(ctx)
	assert.NoError(t, err)

	yamlNode, ok := node.(*yaml.Node)
	assert.True(t, ok)
	assert.Equal(t, "$ref", yamlNode.Content[0].Value)
}

func TestBuildLowParameter_Success(t *testing.T) {
	// Test the success path of buildLowParameter
	yml := `name: testParam
in: query
description: A test parameter
required: true`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	assert.NoError(t, err)

	result, err := buildLowParameter(node.Content[0], nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "testParam", result.Name.Value)
	assert.Equal(t, "query", result.In.Value)
}

func TestBuildLowParameter_BuildError(t *testing.T) {
	// Create a parameter with a schema that has an unresolvable $ref
	// This triggers an error in ExtractSchema during Build
	yml := `name: test
in: query
schema:
  $ref: '#/components/schemas/DoesNotExist'`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	assert.NoError(t, err)

	// Create an empty index - the ref won't be found
	config := index.CreateOpenAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(&node, config)

	result, err := buildLowParameter(node.Content[0], idx)

	// The schema extraction should fail because the ref doesn't exist
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestParameter_MarshalYAMLInline_ExternalRef(t *testing.T) {
	// Test that MarshalYAMLInline resolves external references properly
	// This covers the "if rendered != nil" path in MarshalYAMLInline
	yml := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
components:
  parameters:
    FilterParam:
      $ref: "#/components/parameters/InternalParam"
    InternalParam:
      name: filter
      in: query
      description: Filter query parameter
      schema:
        type: string
paths: {}`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	config := index.CreateOpenAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(&idxNode, config)
	resolver := index.NewResolver(idx)
	idx.SetResolver(resolver)
	errs := resolver.Resolve()
	assert.Empty(t, errs)

	// Build the low-level parameter that has an internal reference
	// When we call MarshalYAMLInline, it should resolve it
	var n v3.Parameter
	paramNode := idxNode.Content[0].Content[5].Content[1].Content[1] // components.parameters.FilterParam
	_ = low.BuildModel(paramNode, &n)
	_ = n.Build(context.Background(), nil, paramNode, idx)

	p := NewParameter(&n)

	// Call MarshalYAMLInline which should resolve the reference
	result, err := p.MarshalYAMLInline()
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestParameter_MarshalYAMLInlineWithContext_ExternalRef(t *testing.T) {
	// Test that MarshalYAMLInlineWithContext resolves external references properly
	yml := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
components:
  parameters:
    FilterParam:
      $ref: "#/components/parameters/InternalParam"
    InternalParam:
      name: filter
      in: query
      description: Filter query parameter
      schema:
        type: string
paths: {}`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	config := index.CreateOpenAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(&idxNode, config)
	resolver := index.NewResolver(idx)
	idx.SetResolver(resolver)
	errs := resolver.Resolve()
	assert.Empty(t, errs)

	var n v3.Parameter
	paramNode := idxNode.Content[0].Content[5].Content[1].Content[1] // components.parameters.FilterParam
	_ = low.BuildModel(paramNode, &n)
	_ = n.Build(context.Background(), nil, paramNode, idx)

	p := NewParameter(&n)

	ctx := base.NewInlineRenderContext()
	result, err := p.MarshalYAMLInlineWithContext(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}
