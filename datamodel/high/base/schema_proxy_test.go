// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestSchemaProxy_MarshalYAML(t *testing.T) {
	const ymlComponents = `components:
    schemas:
     rice:
       type: string
     nice:
       properties:
         rice:
           $ref: '#/components/schemas/rice'
     ice:
       properties:
         rice:
           $ref: '#/components/schemas/rice'`

	idx := func() *index.SpecIndex {
		var idxNode yaml.Node
		err := yaml.Unmarshal([]byte(ymlComponents), &idxNode)
		assert.NoError(t, err)
		return index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())
	}()

	const ref = "#/components/schemas/nice"
	const ymlSchema = `$ref: '` + ref + `'`
	var node yaml.Node
	_ = yaml.Unmarshal([]byte(ymlSchema), &node)

	lowProxy := new(lowbase.SchemaProxy)
	err := lowProxy.Build(context.Background(), nil, node.Content[0], idx)
	assert.NoError(t, err)

	lowRef := low.NodeReference[*lowbase.SchemaProxy]{
		Value: lowProxy,
	}

	sp := NewSchemaProxy(&lowRef)

	origin := sp.GetReferenceOrigin()
	assert.Nil(t, origin)

	rend, _ := sp.Render()
	assert.Equal(t, "$ref: '#/components/schemas/nice'", strings.TrimSpace(string(rend)))
}

func TestCreateSchemaProxy(t *testing.T) {
	sp := CreateSchemaProxy(&Schema{Description: "iAmASchema"})
	assert.Equal(t, "iAmASchema", sp.rendered.Description)
	assert.False(t, sp.IsReference())
}

func TestCreateSchemaProxyRef(t *testing.T) {
	sp := CreateSchemaProxyRef("#/components/schemas/MySchema")
	assert.Equal(t, "#/components/schemas/MySchema", sp.GetReference())
	assert.True(t, sp.IsReference())
}

func TestSchemaProxy_GetReference(t *testing.T) {
	refNode := utils.CreateStringNode("#/components/schemas/MySchema")

	ref := low.Reference{}
	ref.SetReference("#/components/schemas/MySchema", refNode)

	sp := &SchemaProxy{
		schema: &low.NodeReference[*lowbase.SchemaProxy]{
			Value: &lowbase.SchemaProxy{
				Reference: ref,
			},
		},
	}
	assert.Equal(t, "#/components/schemas/MySchema", sp.GetReference())
	assert.Equal(t, refNode, sp.GetReferenceNode())
}

func TestSchemaProxy_IsReference_Nil(t *testing.T) {
	var sp *SchemaProxy
	assert.False(t, sp.IsReference())
}

func TestSchemaProxy_NoSchema_GetOrigin(t *testing.T) {
	sp := &SchemaProxy{}
	assert.Nil(t, sp.GetReferenceOrigin())
}

func TestCreateSchemaProxyRef_GetReferenceNode(t *testing.T) {
	refNode := utils.CreateRefNode("#/components/schemas/MySchema")

	sp := CreateSchemaProxyRef("#/components/schemas/MySchema")
	assert.Equal(t, refNode, sp.GetReferenceNode())
}

func TestCreateRefNode_MarshalYAML(t *testing.T) {
	ref := low.Reference{}
	ref.SetReference("#/components/schemas/MySchema", nil)

	sp := &SchemaProxy{
		schema: &low.NodeReference[*lowbase.SchemaProxy]{
			Value: &lowbase.SchemaProxy{
				Reference: ref,
			},
		},
	}
	node, err := sp.MarshalYAML()
	require.NoError(t, err)
	assert.Equal(t, node, utils.CreateRefNode("#/components/schemas/MySchema"))
}

func TestSchemaProxy_MarshalYAML_InlineCircular(t *testing.T) {
	const ymlComponents = `openapi: 3.1
components:
  schemas:
    spice:
      properties:
        ice:
          $ref: '#/components/schemas/nice'
    nice:
      properties:
        rice:
          $ref: '#/components/schemas/nice'`

	idx := func() *index.SpecIndex {
		var idxNode yaml.Node
		err := yaml.Unmarshal([]byte(ymlComponents), &idxNode)
		assert.NoError(t, err)
		return index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())
	}()

	resolver := index.NewResolver(idx)
	resolver.CheckForCircularReferences()

	const ymlSchema = `properties:
  rice:
    $ref: '#/components/schemas/nice'`
	var node yaml.Node
	_ = yaml.Unmarshal([]byte(ymlSchema), &node)

	lowProxy := new(lowbase.SchemaProxy)
	err := lowProxy.Build(context.Background(), &node, node.Content[0], idx)
	assert.NoError(t, err)

	lowRef := low.NodeReference[*lowbase.SchemaProxy]{
		Value:   lowProxy,
		KeyNode: &node,
	}

	spEmpty := NewSchemaProxy(nil)
	assert.Nil(t, spEmpty.GetSchemaKeyNode())

	sp := NewSchemaProxy(&lowRef)
	assert.NotNil(t, sp.GetSchemaKeyNode())

	rend, _ := sp.MarshalYAMLInline()
	assert.NotNil(t, rend)
}
