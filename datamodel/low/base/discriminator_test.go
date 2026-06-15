// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/utils"
	"github.com/pb33f/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestDiscriminator_FindMappingValue(t *testing.T) {
	yml := `propertyName: freshCakes
mapping:
  something: nothing`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)

	var n Discriminator
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)
	assert.Equal(t, "nothing", n.FindMappingValue("something").Value)
	assert.Nil(t, n.FindMappingValue("freshCakes"))
}

func TestDiscriminator_Hash(t *testing.T) {
	left := `propertyName: freshCakes
mapping:
  something: nothing`

	right := `mapping:
  something: nothing
propertyName: freshCakes`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc Discriminator
	var rDoc Discriminator
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	lDoc.RootNode = &lNode
	lDoc.KeyNode = &rNode

	assert.Equal(t, lDoc.Hash(), rDoc.Hash())
	assert.NotNil(t, lDoc.GetRootNode())
	assert.NotNil(t, lDoc.GetKeyNode())
}

func TestDiscriminator_DefaultMapping_OpenAPI32(t *testing.T) {
	yml := `propertyName: petType
mapping:
  dog: '#/components/schemas/Dog'
  cat: '#/components/schemas/Cat'
defaultMapping: '#/components/schemas/UnknownPet'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)

	var n Discriminator
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)
	assert.Equal(t, "petType", n.PropertyName.Value)
	assert.Equal(t, "#/components/schemas/UnknownPet", n.DefaultMapping.Value)
	assert.Equal(t, "#/components/schemas/Dog", n.FindMappingValue("dog").Value)
	assert.Equal(t, "#/components/schemas/Cat", n.FindMappingValue("cat").Value)
}

func TestValidateDiscriminatorMappingValueNodesRejectsNonStringValues(t *testing.T) {
	tests := []struct {
		name    string
		mapping string
		wantErr string
	}{
		{
			name: "object",
			mapping: `propertyName: type
mapping:
  properties:
    type: object`,
			wantErr: "discriminator.mapping.properties must be a string",
		},
		{
			name: "array",
			mapping: `propertyName: type
mapping:
  required:
    - type`,
			wantErr: "discriminator.mapping.required must be a string",
		},
		{
			name: "boolean",
			mapping: `propertyName: type
mapping:
  additionalProperties: false`,
			wantErr: "discriminator.mapping.additionalProperties must be a string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var idxNode yaml.Node
			mErr := yaml.Unmarshal([]byte(tt.mapping), &idxNode)
			assert.NoError(t, mErr)

			err := ValidateDiscriminatorMappingValueNodes(idxNode.Content[0])
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestValidateDiscriminatorMappingValueNodesAcceptsStringValues(t *testing.T) {
	yml := `propertyName: type
mapping:
  dog: '#/components/schemas/Dog'
  cat: Cat`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)

	assert.NoError(t, ValidateDiscriminatorMappingValueNodes(idxNode.Content[0]))
}

func TestValidateDiscriminatorMappingValueNodesAcceptsAliasMapping(t *testing.T) {
	yml := `petMap: &petMap
  dog: '#/components/schemas/Dog'
  cat: '#/components/schemas/Cat'
discriminator:
  propertyName: type
  mapping: *petMap`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)

	var discriminatorNode *yaml.Node
	for i := 0; i < len(idxNode.Content[0].Content); i += 2 {
		if idxNode.Content[0].Content[i].Value == "discriminator" {
			discriminatorNode = idxNode.Content[0].Content[i+1]
			break
		}
	}

	assert.NoError(t, ValidateDiscriminatorMappingValueNodes(discriminatorNode))

	var n Discriminator
	err := low.BuildModel(discriminatorNode, &n)
	assert.NoError(t, err)
	assert.Equal(t, "#/components/schemas/Dog", n.FindMappingValue("dog").Value)
	assert.Equal(t, "#/components/schemas/Cat", n.FindMappingValue("cat").Value)
}

func TestValidateDiscriminatorMappingValueNodesAcceptsMergedMapping(t *testing.T) {
	yml := `petMap: &petMap
  dog: '#/components/schemas/Dog'
  cat: '#/components/schemas/Cat'
type: object
discriminator:
  propertyName: type
  mapping:
    <<: *petMap
    lizard: '#/components/schemas/Lizard'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)

	var schema Schema
	err := schema.Build(context.Background(), idxNode.Content[0], nil)
	assert.NoError(t, err)
	assert.Equal(t, "#/components/schemas/Dog", schema.Discriminator.Value.FindMappingValue("dog").Value)
	assert.Equal(t, "#/components/schemas/Cat", schema.Discriminator.Value.FindMappingValue("cat").Value)
	assert.Equal(t, "#/components/schemas/Lizard", schema.Discriminator.Value.FindMappingValue("lizard").Value)
}

func TestValidateDiscriminatorMappingValueNodesDefensiveNilKeys(t *testing.T) {
	discriminatorNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			nil,
			utils.CreateStringNode("ignored"),
			utils.CreateStringNode("mapping"),
			{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					nil,
					utils.CreateStringNode("#/components/schemas/Ignored"),
					utils.CreateStringNode("dog"),
					utils.CreateStringNode("#/components/schemas/Dog"),
				},
			},
		},
	}

	assert.NoError(t, ValidateDiscriminatorMappingValueNodes(discriminatorNode))
}

func TestSchemaBuildRejectsInvalidDiscriminatorMappingValue(t *testing.T) {
	yml := `type: object
discriminator:
  propertyName: type
  mapping:
    dog:
      type: object`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)

	var schema Schema
	err := schema.Build(context.Background(), idxNode.Content[0], nil)
	assert.ErrorContains(t, err, "discriminator.mapping.dog must be a string")
}

func TestValidateDiscriminatorMappingValueNodesNonMappingCases(t *testing.T) {
	assert.NoError(t, ValidateDiscriminatorMappingValueNodes(nil))
	assert.NoError(t, ValidateDiscriminatorMappingValueNodes(&yaml.Node{Kind: yaml.SequenceNode}))

	yml := `propertyName: type`
	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	assert.NoError(t, ValidateDiscriminatorMappingValueNodes(idxNode.Content[0]))

	yml = `propertyName: type
mapping:
  - nope`
	mErr = yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	assert.ErrorContains(t, ValidateDiscriminatorMappingValueNodes(idxNode.Content[0]), "discriminator.mapping must be an object")
}

func TestDescribeDiscriminatorMappingNode(t *testing.T) {
	assert.Equal(t, "nil", describeDiscriminatorMappingNode(nil))
	assert.Equal(t, "document", describeDiscriminatorMappingNode(&yaml.Node{Kind: yaml.DocumentNode}))
	assert.Equal(t, "alias", describeDiscriminatorMappingNode(&yaml.Node{Kind: yaml.AliasNode}))
	assert.Equal(t, "kind 99", describeDiscriminatorMappingNode(&yaml.Node{Kind: 99}))
}

func TestDiscriminator_DefaultMapping_Hash(t *testing.T) {
	left := `propertyName: petType
mapping:
  dog: '#/components/schemas/Dog'
defaultMapping: '#/components/schemas/UnknownPet'`

	right := `propertyName: petType
defaultMapping: '#/components/schemas/UnknownPet'
mapping:
  dog: '#/components/schemas/Dog'`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	var lDoc Discriminator
	var rDoc Discriminator
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)

	// Same content, different order should produce same hash
	assert.Equal(t, lDoc.Hash(), rDoc.Hash())
}

func TestDiscriminator_DefaultMapping_HashDifferent(t *testing.T) {
	left := `propertyName: petType
defaultMapping: '#/components/schemas/UnknownPet'`

	right := `propertyName: petType
defaultMapping: '#/components/schemas/OtherPet'`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	var lDoc Discriminator
	var rDoc Discriminator
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)

	// Different defaultMapping should produce different hash
	assert.NotEqual(t, lDoc.Hash(), rDoc.Hash())
}
