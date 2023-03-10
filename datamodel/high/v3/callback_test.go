// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
    "github.com/pb33f/libopenapi/index"
    "github.com/stretchr/testify/assert"
    "gopkg.in/yaml.v3"
    "strings"
    "testing"
)

func TestCallback_MarshalYAML(t *testing.T) {

    cb := &Callback{
        Expression: map[string]*PathItem{
            "https://pb33f.io": {
                Get: &Operation{
                    OperationId: "oneTwoThree",
                },
            },
            "https://pb33f.io/libopenapi": {
                Get: &Operation{
                    OperationId: "openaypeeeye",
                },
            },
        },
        Extensions: map[string]any{
            "x-burgers": "why not?",
        },
    }

    rend, _ := cb.Render()

    desired := `https://pb33f.io:
    get:
        operationId: oneTwoThree
https://pb33f.io/libopenapi:
    get:
        operationId: openaypeeeye
x-burgers: why not?`

    assert.Equal(t, desired, strings.TrimSpace(string(rend)))

    // mutate
    cb.Expression["https://pb33f.io"].Get.OperationId = "blim-blam"
    cb.Extensions = map[string]interface{}{"x-burgers": "yes please!"}

    desired = `https://pb33f.io:
    get:
        operationId: blim-blam
https://pb33f.io/libopenapi:
    get:
        operationId: openaypeeeye
x-burgers: yes please!`

    rend, _ = cb.Render()
    assert.Equal(t, desired, strings.TrimSpace(string(rend)))

    k := `x-break-everything: please
"{$request.query.queryUrl}":
  post:
    requestBody: null
    description: Callback payload
    content:
      application/json:
        schema:
          type: string
    responses:
      "200":
        description: callback successfully processes
`

    var idxNode yaml.Node
    err := yaml.Unmarshal([]byte(k), &idxNode)
    assert.NoError(t, err)
    idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

    var n v3.Callback
    _ = low.BuildModel(idxNode.Content[0], &n)
    _ = n.Build(idxNode.Content[0], idx)

    r := NewCallback(&n)

    assert.Equal(t, "please", r.Extensions["x-break-everything"])

    rend, _ = r.Render()
    assert.Equal(t, k, strings.TrimSpace(string(rend)))
}
