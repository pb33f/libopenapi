// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"fmt"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
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
	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(testComponentsYaml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Components
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), idxNode.Content[0], idx)
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

	assert.Equal(t, "76328a0e32a9989471d335734af04a37bdfad333cf8cd8aa8065998c3a1489a2",
		low.GenerateHashString(&n))
}

func TestComponents_Build_Success_Skip(t *testing.T) {
	yml := `components:`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Components
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)
}

func TestComponents_Build_Fail(t *testing.T) {
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

	err = n.Build(context.Background(), idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestComponents_Build_ParameterFail(t *testing.T) {
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

	err = n.Build(context.Background(), idxNode.Content[0], idx)
	assert.Error(t, err)
}

// Test parse failure among many parameters.
// This stresses `TranslatePipeline`'s error handling.
func TestComponents_Build_ParameterFail_Many(t *testing.T) {
	yml := `
  parameters:
`

	for i := 0; i < 1000; i++ {
		format := `
    pizza%d:
      schema:
        $ref: '#/this is a problem.'
`
		yml += fmt.Sprintf(format, i)
	}

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Components
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestComponents_Build_Fail_TypeFail(t *testing.T) {
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

	err = n.Build(context.Background(), idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestComponents_Build_ExtensionTest(t *testing.T) {
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

	err = n.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

	var xCurry string
	_ = n.FindExtension("x-curry").Value.Decode(&xCurry)

	assert.Equal(t, "seagull", xCurry)
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

	err = n.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

	var xCurry string
	_ = n.FindExtension("x-curry").Value.Decode(&xCurry)

	assert.Equal(t, "seagull", xCurry)
	assert.Equal(t, 1, orderedmap.Len(n.GetExtensions()))
	assert.Equal(t, "e45605d7361dbc9d4b9723257701bef1d283f8fe9566b9edda127fc66a6b8fdd",
		low.GenerateHashString(&n))
}
