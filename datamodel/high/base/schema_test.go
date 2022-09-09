// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestNewSchemaProxy(t *testing.T) {

	// check proxy
	yml := `components:
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

	var idxNode, compNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	yml = `properties:
    rice:
     $ref: '#/components/schemas/I-do-not-exist'`

	_ = yaml.Unmarshal([]byte(yml), &compNode)

	sp := new(v3.SchemaProxy)
	err := sp.Build(compNode.Content[0], idx)
	assert.NoError(t, err)

	lowproxy := low.NodeReference[*v3.SchemaProxy]{
		Value:     sp,
		ValueNode: idxNode.Content[0],
	}

	sch1 := SchemaProxy{schema: &lowproxy}
	assert.Nil(t, sch1.Schema())
	assert.Error(t, sch1.GetBuildError())

}
