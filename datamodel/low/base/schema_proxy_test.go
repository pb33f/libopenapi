// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestSchemaProxy_Build(t *testing.T) {

	yml := `description: something`

	var sch SchemaProxy
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	err := sch.Build(idxNode.Content[0], nil)
	assert.NoError(t, err)
	assert.Equal(t, "something", sch.Schema().Description.Value)
	assert.Empty(t, sch.GetSchemaReference())
	assert.NotNil(t, sch.GetValueNode())
	assert.False(t, sch.IsSchemaReference())
}

func TestSchemaProxy_Build_CheckRef(t *testing.T) {

	yml := `$ref: wat`

	var sch SchemaProxy
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	err := sch.Build(idxNode.Content[0], nil)
	assert.NoError(t, err)
	assert.True(t, sch.IsSchemaReference())
	assert.Equal(t, "wat", sch.GetSchemaReference())
}
