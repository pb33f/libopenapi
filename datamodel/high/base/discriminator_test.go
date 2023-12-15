// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"fmt"
	"strings"
	"testing"

	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestNewDiscriminator(t *testing.T) {
	var cNode yaml.Node

	yml := `propertyName: coffee
mapping:
    fogCleaner: in the morning`

	_ = yaml.Unmarshal([]byte(yml), &cNode)

	// build low
	var lowDiscriminator lowbase.Discriminator
	_ = lowmodel.BuildModel(cNode.Content[0], &lowDiscriminator)

	// build high
	highDiscriminator := NewDiscriminator(&lowDiscriminator)

	assert.Equal(t, "coffee", highDiscriminator.PropertyName)
	assert.Equal(t, "in the morning", highDiscriminator.Mapping.GetOrZero("fogCleaner"))
	assert.Equal(t, 3, highDiscriminator.GoLow().FindMappingValue("fogCleaner").ValueNode.Line)

	// render the example as YAML
	rendered, _ := highDiscriminator.Render()
	assert.Equal(t, strings.TrimSpace(string(rendered)), yml)
}

func ExampleNewDiscriminator() {
	// create a yaml representation of a discriminator (can be JSON, doesn't matter)
	yml := `propertyName: coffee
mapping:
  coffee: in the morning`

	// unmarshal into a *yaml.Node
	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	// build low-level model
	var lowDiscriminator lowbase.Discriminator
	_ = lowmodel.BuildModel(node.Content[0], &lowDiscriminator)

	// build high-level model
	highDiscriminator := NewDiscriminator(&lowDiscriminator)

	// print out a mapping defined for the discriminator.
	fmt.Print(highDiscriminator.Mapping.GetOrZero("coffee"))
	// Output: in the morning
}
