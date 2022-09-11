// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
    lowmodel "github.com/pb33f/libopenapi/datamodel/low"
    lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
    "github.com/stretchr/testify/assert"
    "gopkg.in/yaml.v3"
    "testing"
)

func TestNewDiscriminator(t *testing.T) {

    var cNode yaml.Node

    yml := `propertyName: coffee
mapping:
  fogCleaner: in the morning`

    _ = yaml.Unmarshal([]byte(yml), &cNode)

    // build low
    var lowDiscriminator lowbase.Discriminator
    _ = lowmodel.BuildModel(&cNode, &lowDiscriminator)

    // build high
    highDiscriminator := NewDiscriminator(&lowDiscriminator)

    assert.Equal(t, "coffee", highDiscriminator.PropertyName)
    assert.Equal(t, "in the morning", highDiscriminator.Mapping["fogCleaner"])
    assert.Equal(t, 3, highDiscriminator.GoLow().FindMappingValue("fogCleaner").ValueNode.Line)

}
