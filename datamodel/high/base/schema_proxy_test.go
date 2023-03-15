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
        return index.NewSpecIndex(&idxNode)
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

    rend, _ := sp.Render()
    assert.Equal(t, "$ref: '#/components/schemas/nice'", strings.TrimSpace(string(rend)))

}
