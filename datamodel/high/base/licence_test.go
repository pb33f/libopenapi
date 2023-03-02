// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
    lowmodel "github.com/pb33f/libopenapi/datamodel/low"
    lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
    "github.com/stretchr/testify/assert"
    "gopkg.in/yaml.v3"
    "testing"
)

func TestLicense_Render(t *testing.T) {

    highL := &License{Name: "MIT", URL: "https://pb33f.io"}
    dat, _ := highL.Render()

    // unmarshal yaml into a *yaml.Node instance
    var cNode yaml.Node
    _ = yaml.Unmarshal(dat, &cNode)

    // build low
    var lowLicense lowbase.License
    _ = lowmodel.BuildModel(cNode.Content[0], &lowLicense)

    // build high
    highLicense := NewLicense(&lowLicense)

    assert.Equal(t, "MIT", highLicense.Name)
    assert.Equal(t, "https://pb33f.io", highLicense.URL)

}

func TestLicense_RenderEqual(t *testing.T) {

    yml := `name: MIT
url: https://pb33f.io/not-real
`
    // unmarshal yaml into a *yaml.Node instance
    var cNode yaml.Node
    _ = yaml.Unmarshal([]byte(yml), &cNode)

    // build low
    var lowLicense lowbase.License
    _ = lowmodel.BuildModel(cNode.Content[0], &lowLicense)
    _ = lowLicense.Build(cNode.Content[0], nil)

    // build high
    highLicense := NewLicense(&lowLicense)

    assert.Equal(t, "MIT", highLicense.Name)
    assert.Equal(t, "https://pb33f.io/not-real", highLicense.URL)

    // re-render and ensure everything is in the same order as before.
    bytes, _ := highLicense.Render()
    assert.Equal(t, yml, string(bytes))

}