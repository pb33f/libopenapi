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
	"go.yaml.in/yaml/v4"
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

func TestNewDiscriminator_DefaultMapping_OpenAPI32(t *testing.T) {
	var cNode yaml.Node

	yml := `propertyName: petType
mapping:
  dog: '#/components/schemas/Dog'
  cat: '#/components/schemas/Cat'
defaultMapping: '#/components/schemas/UnknownPet'`

	_ = yaml.Unmarshal([]byte(yml), &cNode)

	// build low
	var lowDiscriminator lowbase.Discriminator
	_ = lowmodel.BuildModel(cNode.Content[0], &lowDiscriminator)

	// build high
	highDiscriminator := NewDiscriminator(&lowDiscriminator)

	assert.Equal(t, "petType", highDiscriminator.PropertyName)
	assert.Equal(t, "#/components/schemas/UnknownPet", highDiscriminator.DefaultMapping)
	assert.Equal(t, "#/components/schemas/Dog", highDiscriminator.Mapping.GetOrZero("dog"))
	assert.Equal(t, "#/components/schemas/Cat", highDiscriminator.Mapping.GetOrZero("cat"))

	// Test GoLow and GoLowUntyped
	assert.Equal(t, &lowDiscriminator, highDiscriminator.GoLow())
	assert.Equal(t, &lowDiscriminator, highDiscriminator.GoLowUntyped())

	// render the example as YAML - test structure, not exact formatting
	rendered, _ := highDiscriminator.Render()
	assert.Contains(t, string(rendered), "propertyName: petType")
	assert.Contains(t, string(rendered), "defaultMapping: '#/components/schemas/UnknownPet'")
	assert.Contains(t, string(rendered), "dog: '#/components/schemas/Dog'")
	assert.Contains(t, string(rendered), "cat: '#/components/schemas/Cat'")
}

func TestNewDiscriminator_NoDefaultMapping(t *testing.T) {
	var cNode yaml.Node

	yml := `propertyName: petType
mapping:
  dog: '#/components/schemas/Dog'`

	_ = yaml.Unmarshal([]byte(yml), &cNode)

	// build low
	var lowDiscriminator lowbase.Discriminator
	_ = lowmodel.BuildModel(cNode.Content[0], &lowDiscriminator)

	// build high
	highDiscriminator := NewDiscriminator(&lowDiscriminator)

	assert.Equal(t, "petType", highDiscriminator.PropertyName)
	assert.Equal(t, "", highDiscriminator.DefaultMapping) // Should be empty when not specified
	assert.Equal(t, "#/components/schemas/Dog", highDiscriminator.Mapping.GetOrZero("dog"))
}

func TestNewDiscriminator_MarshalYAML(t *testing.T) {
	var cNode yaml.Node

	yml := `propertyName: animalType
mapping:
  snake: '#/components/schemas/Snake'
  lizard: '#/components/schemas/Lizard'
defaultMapping: '#/components/schemas/UnknownReptile'`

	_ = yaml.Unmarshal([]byte(yml), &cNode)

	// build low
	var lowDiscriminator lowbase.Discriminator
	_ = lowmodel.BuildModel(cNode.Content[0], &lowDiscriminator)

	// build high
	highDiscriminator := NewDiscriminator(&lowDiscriminator)

	// Test MarshalYAML
	marshaled, err := highDiscriminator.MarshalYAML()
	assert.NoError(t, err)
	assert.NotNil(t, marshaled)
}
