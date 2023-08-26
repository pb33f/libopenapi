// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

var testComponentsYaml = `
  x-pizza: crispy
  schemas:
    one:
      description: one of many
    two:
      description: two of many
  responses:
    three:
      description: three of many
    four:
      description: four of many
  parameters:
    five:
      description: five of many
    six:
      description: six of many
  examples:
    seven:
      description: seven of many
    eight:
      description: eight of many
  requestBodies:
    nine:
      description: nine of many
    ten:
      description: ten of many
  headers: 
    eleven:
      description: eleven of many
    twelve:
      description: twelve of many
  securitySchemes:
    thirteen:
      description: thirteen of many
    fourteen:
      description: fourteen of many
  links:
    fifteen:
      description: fifteen of many
    sixteen:
      description: sixteen of many
  callbacks: 
    seventeen:
      '{reference}':
        post:
          description: seventeen of many
    eighteen:
      '{raference}':
        post:
          description: eighteen of many`

func TestComponents_Build_Success(t *testing.T) {
	t.Parallel()
	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(testComponentsYaml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Components
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.NoError(t, err)

	assert.Equal(t, "one of many", n.FindSchema("one").Value.Schema().Description.Value)
	assert.Equal(t, "two of many", n.FindSchema("two").Value.Schema().Description.Value)
	assert.Equal(t, "three of many", n.FindResponse("three").Value.Description.Value)
	assert.Equal(t, "four of many", n.FindResponse("four").Value.Description.Value)
	assert.Equal(t, "five of many", n.FindParameter("five").Value.Description.Value)
	assert.Equal(t, "six of many", n.FindParameter("six").Value.Description.Value)
	assert.Equal(t, "seven of many", n.FindExample("seven").Value.Description.Value)
	assert.Equal(t, "eight of many", n.FindExample("eight").Value.Description.Value)
	assert.Equal(t, "nine of many", n.FindRequestBody("nine").Value.Description.Value)
	assert.Equal(t, "ten of many", n.FindRequestBody("ten").Value.Description.Value)
	assert.Equal(t, "eleven of many", n.FindHeader("eleven").Value.Description.Value)
	assert.Equal(t, "twelve of many", n.FindHeader("twelve").Value.Description.Value)
	assert.Equal(t, "thirteen of many", n.FindSecurityScheme("thirteen").Value.Description.Value)
	assert.Equal(t, "fourteen of many", n.FindSecurityScheme("fourteen").Value.Description.Value)
	assert.Equal(t, "fifteen of many", n.FindLink("fifteen").Value.Description.Value)
	assert.Equal(t, "sixteen of many", n.FindLink("sixteen").Value.Description.Value)
	assert.Equal(t, "seventeen of many",
		n.FindCallback("seventeen").Value.FindExpression("{reference}").Value.Post.Value.Description.Value)
	assert.Equal(t, "eighteen of many",
		n.FindCallback("eighteen").Value.FindExpression("{raference}").Value.Post.Value.Description.Value)

	assert.Equal(t, "7add1a6c63a354b1a8ffe22552c213fe26d1229beb0b0cbe7c7ca06e63f9a364",
		low.GenerateHashString(&n))

}

func TestComponents_Build_Success_Skip(t *testing.T) {
	t.Parallel()
	yml := `components:`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Components
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.NoError(t, err)

}

func TestComponents_Build_Fail(t *testing.T) {
	t.Parallel()
	yml := `
  parameters: 
    schema:
      $ref: '#/this is a problem.'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Components
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)

}

func TestComponents_Build_ParameterFail(t *testing.T) {
	t.Parallel()
	yml := `
  parameters:
    pizza:
      schema:
        $ref: '#/this is a problem.'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Components
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)

}

func TestComponents_Build_Fail_TypeFail(t *testing.T) {
	t.Parallel()
	yml := `
  parameters: 
    - schema:
        $ref: #/this is a problem.`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Components
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestComponents_Build_ExtensionTest(t *testing.T) {
	t.Parallel()
	yml := `x-curry: seagull
headers:
  x-curry-gull: vinadloo`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Components
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "seagull", n.FindExtension("x-curry").Value)

}

func TestComponents_Build_HashEmpty(t *testing.T) {

	yml := `x-curry: seagull`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Components
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "seagull", n.FindExtension("x-curry").Value)
	assert.Len(t, n.GetExtensions(), 1)
	assert.Equal(t, "9cf2c6ab3f9ff7e5231fcb391c8af5c47406711d2ca366533f21a8bb2f67edfe",
		low.GenerateHashString(&n))

}
