// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"strings"
	"testing"
)

func TestDynamicValue_Render_A(t *testing.T) {
	dv := &DynamicValue[string, int]{N: 0, A: "hello"}
	dvb, _ := dv.Render()
	assert.Equal(t, "hello", strings.TrimSpace(string(dvb)))
}

func TestDynamicValue_Render_B(t *testing.T) {
	dv := &DynamicValue[string, int]{N: 1, B: 12345}
	dvb, _ := dv.Render()
	assert.Equal(t, "12345", strings.TrimSpace(string(dvb)))
}

func TestDynamicValue_Render_Bool(t *testing.T) {
	dv := &DynamicValue[string, bool]{N: 1, B: true}
	dvb, _ := dv.Render()
	assert.Equal(t, "true", strings.TrimSpace(string(dvb)))
}

func TestDynamicValue_Render_Int64(t *testing.T) {
	dv := &DynamicValue[string, int64]{N: 1, B: 12345567810}
	dvb, _ := dv.Render()
	assert.Equal(t, "12345567810", strings.TrimSpace(string(dvb)))
}

func TestDynamicValue_Render_Int32(t *testing.T) {
	dv := &DynamicValue[string, int32]{N: 1, B: 1234567891}
	dvb, _ := dv.Render()
	assert.Equal(t, "1234567891", strings.TrimSpace(string(dvb)))
}

func TestDynamicValue_Render_Float32(t *testing.T) {
	dv := &DynamicValue[string, float32]{N: 1, B: 23456.123}
	dvb, _ := dv.Render()
	assert.Equal(t, "23456.123", strings.TrimSpace(string(dvb)))
}

func TestDynamicValue_Render_Float64(t *testing.T) {
	dv := &DynamicValue[string, float64]{N: 1, B: 23456.1233456778}
	dvb, _ := dv.Render()
	assert.Equal(t, "23456.1233456778", strings.TrimSpace(string(dvb)))
}

func TestDynamicValue_Render_Ptr(t *testing.T) {

	type cake struct {
		Cake string
	}

	dv := &DynamicValue[string, *cake]{N: 1, B: &cake{Cake: "vanilla"}}
	dvb, _ := dv.Render()
	assert.Equal(t, "cake: vanilla", strings.TrimSpace(string(dvb)))
}

func TestDynamicValue_Render_PtrRenderable(t *testing.T) {

	tag := &Tag{
		Name: "cake",
	}

	dv := &DynamicValue[string, *Tag]{N: 1, B: tag}
	dvb, _ := dv.Render()
	assert.Equal(t, "name: cake", strings.TrimSpace(string(dvb)))
}

func TestDynamicValue_RenderInline(t *testing.T) {

	tag := &Tag{
		Name: "cake",
	}

	dv := &DynamicValue[string, *Tag]{N: 1, B: tag}
	dvb, _ := dv.RenderInline()
	assert.Equal(t, "name: cake", strings.TrimSpace(string(dvb)))
}

func TestDynamicValue_MarshalYAMLInline(t *testing.T) {

	const ymlComponents = `components:
    schemas:
     rice:
       type: array
       items:
         $ref: '#/components/schemas/ice'
     nice:
       properties:
         rice:
           $ref: '#/components/schemas/rice'
     ice:
       type: string`

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
	err := lowProxy.Build(node.Content[0], idx)
	assert.NoError(t, err)

	lowRef := low.NodeReference[*lowbase.SchemaProxy]{
		Value: lowProxy,
	}

	sp := NewSchemaProxy(&lowRef)

	rend, _ := sp.MarshalYAMLInline()

	// convert node into yaml
	bits, _ := yaml.Marshal(rend)
	assert.Equal(t, "properties:\n    rice:\n        type: array\n        items:\n            type: string", strings.TrimSpace(string(bits)))
}

func TestDynamicValue_MarshalYAMLInline_Error(t *testing.T) {

	const ymlComponents = `components:
    schemas:
     rice:
       type: array
       items:
         $ref: '#/components/schemas/bork'
     nice:
       properties:
         rice:
           $ref: '#/components/schemas/berk'
     ice:
       type: string`

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
	err := lowProxy.Build(node.Content[0], idx)
	assert.NoError(t, err)

	lowRef := low.NodeReference[*lowbase.SchemaProxy]{
		Value: lowProxy,
	}

	sp := NewSchemaProxy(&lowRef)

	rend, er := sp.MarshalYAMLInline()
	assert.Nil(t, rend)
	assert.Error(t, er)
}
