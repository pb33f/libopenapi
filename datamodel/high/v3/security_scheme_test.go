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
    _ = n.Build(idxNode.Content[0], idx)

    r := NewSecurityScheme(&n)

    dat, _ = r.Render()

    desired := `type: apiKey
description: this is a description
name: superSecret
in: header
scheme: https`

    assert.Equal(t, desired, strings.TrimSpace(string(dat)))
}
