// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"context"
	"github.com/pb33f/libopenapi/datamodel/low"
	v2 "github.com/pb33f/libopenapi/datamodel/low/v2"
	"github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestCompareComponents_Swagger_Definitions_Equal(t *testing.T) {

	left := `thing1:
 type: int
 description: a thing
thing2:
 type: string
 description: another thing.`

	right := `thing1:
 type: int
 description: a thing
thing2:
 type: string
 description: another thing.`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Definitions
	var rDoc v2.Definitions
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareComponents_Swagger_Definitions_Modified(t *testing.T) {

	left := `thing1:
 type: int
 description: a thing
thing2:
 type: int
 description: another thing.`

	right := `thing1:
 type: int
 description: a thing that changed`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Definitions
	var rDoc v2.Definitions
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 2)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())

}

func TestCompareComponents_Swagger_Definitions_Added(t *testing.T) {

	left := `thing1:
 type: int
 description: a thing
thing2:
 type: string
 description: another thing.`

	right := `thing1:
 type: int
 description: a thing
thing2:
 type: string
 description: another thing.
thing3:
 type: int
 description: added a thing`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Definitions
	var rDoc v2.Definitions
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)

}

func TestCompareComponents_Swagger_Definitions_Removed(t *testing.T) {

	left := `thing1:
 type: int
 description: a thing
thing2:
 type: string
 description: another thing.`

	right := `thing1:
 type: int
 description: a thing
thing2:
 type: string
 description: another thing.
thing3:
 type: int
 description: added a thing`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Definitions
	var rDoc v2.Definitions
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "thing3", extChanges.Changes[0].Original)
}

func TestCompareComponents_Swagger_Parameters_Added(t *testing.T) {

	left := `param1:
 name: nap
param2:
 name: sleep
param3:
 name: snooze
`
	right := `param1:
 name: nap
param2:
 name: sleep
param3:
 name: snooze
param4:
 name: I woke up!`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.ParameterDefinitions
	var rDoc v2.ParameterDefinitions
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "param4", extChanges.Changes[0].New)
}

func TestCompareComponents_Swagger_Parameters_Removed(t *testing.T) {

	left := `param1:
 name: nap
param2:
 name: sleep
param3:
 name: snooze
`
	right := `param1:
 name: nap
param2:
 name: sleep
param3:
 name: snooze
param4:
 name: I woke up!`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.ParameterDefinitions
	var rDoc v2.ParameterDefinitions
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "param4", extChanges.Changes[0].Original)
}

func TestCompareComponents_Swagger_Responses_Added(t *testing.T) {

	left := `resp1:
 description: hi!
resp2:
 description: bye!
`
	right := `resp1:
 description: hi!
resp2:
 description: bye!
resp3:
 description: another response!`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.ResponsesDefinitions
	var rDoc v2.ResponsesDefinitions
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)

	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, "resp3", extChanges.Changes[0].New)
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v2.ResponsesLabel, extChanges.Changes[0].Property)
}

func TestCompareComponents_Swagger_Responses_Removed(t *testing.T) {

	left := `resp1:
 description: hi!
resp2:
 description: bye!
`
	right := `resp1:
 description: hi!
resp2:
 description: bye!
resp3:
 description: another response!`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.ResponsesDefinitions
	var rDoc v2.ResponsesDefinitions
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&rDoc, &lDoc)

	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, "resp3", extChanges.Changes[0].Original)
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v2.ResponsesLabel, extChanges.Changes[0].Property)
}

func TestCompareComponents_Swagger_SecurityDefinitions_Modified(t *testing.T) {

	left := `scheme1:
 description: hi!`

	right := `scheme1:
 description: hi! again
scheme2:
 description: bye! again!`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.SecurityDefinitions
	var rDoc v2.SecurityDefinitions
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 2)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, 1, extChanges.SecuritySchemeChanges["scheme1"].TotalChanges())
	assert.Equal(t, v3.DescriptionLabel, extChanges.SecuritySchemeChanges["scheme1"].Changes[0].Property)
}

func TestCompareComponents_OpenAPI_Schemas_Equal(t *testing.T) {

	left := `
schemas:
  coffee:
    description: tasty
  tv:
    description: mostly boring.`

	right := `schemas:
  coffee:
    description: tasty
  tv:
    description: mostly boring.`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Components
	var rDoc v3.Components
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareComponents_OpenAPI_Schemas_Refs_FullBuild(t *testing.T) {

	left := `components:
  schemas:
    coffee:
      description: tasty
    tv:
      $ref: '#/components/schemas/coffee'`

	right := left

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Components
	var rDoc v3.Components

	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)

	idx := index.NewSpecIndex(&lNode)

	_ = lDoc.Build(context.Background(), lNode.Content[0], idx)
	_ = rDoc.Build(context.Background(), rNode.Content[0], idx)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareComponents_OpenAPI_Schemas_Modify(t *testing.T) {

	left := `
schemas:
  coffee:
    description: tasty
  tv:
    description: mostly boring.`

	right := `schemas:
  coffee:
    description: tasty
  tv:
    description: mostly boring, except when it is sci-fi`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Components
	var rDoc v3.Components
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, 1, extChanges.SchemaChanges["tv"].TotalChanges())
	assert.Equal(t, v3.DescriptionLabel, extChanges.SchemaChanges["tv"].Changes[0].Property)
}

func TestCompareComponents_OpenAPI_Schemas_Add(t *testing.T) {

	left := `
schemas:
  coffee:
    description: tasty
  tv:
    description: mostly boring.`

	right := `schemas:
  coffee:
    description: tasty
  tv:
    description: mostly boring.
  herbs:
    description: need a massive slowdown`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Components
	var rDoc v3.Components
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, "herbs", extChanges.Changes[0].New)
}

func TestCompareComponents_OpenAPI_Schemas_Remove(t *testing.T) {

	left := `
schemas:
  coffee:
    description: tasty
  tv:
    description: mostly boring.`

	right := `schemas:
  coffee:
    description: tasty
  tv:
    description: mostly boring.
  herbs:
    description: need a massive slowdown`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Components
	var rDoc v3.Components
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, "herbs", extChanges.Changes[0].Original)
}

func TestCompareComponents_OpenAPI_Responses_Equal(t *testing.T) {

	left := `
responses:
  niceResponse:
    description: hello
  badResponse:
    description: go away please`

	right := left

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Components
	var rDoc v3.Components
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareComponents_OpenAPI_Responses_FullBuild(t *testing.T) {
	left := `responses:
  coffee:
    description: tasty
  tv:
    $ref: '#/components/responses/coffee'`

	right := left

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Components
	var rDoc v3.Components

	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)

	idx := index.NewSpecIndex(&lNode)

	_ = lDoc.Build(context.Background(), lNode.Content[0], idx)
	_ = rDoc.Build(context.Background(), rNode.Content[0], idx)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareComponents_OpenAPI_ResponsesAdd_FullBuild(t *testing.T) {
	left := `responses:
  coffee:
    description: tasty
  tv:
    $ref: '#/responses/coffee'`

	right := `responses:
    herbs:
      description: refreshing
    coffee:
      description: tasty
    tv:
      $ref: '#/responses/coffee'`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Components
	var rDoc v3.Components

	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)

	idx := index.NewSpecIndex(&lNode)

	_ = lDoc.Build(context.Background(), lNode.Content[0], idx)
	_ = rDoc.Build(context.Background(), rNode.Content[0], idx)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())

}

func TestCompareComponents_OpenAPI_Responses_FullBuild_IdenticalRef(t *testing.T) {
	left := `components:
  responses:
    coffee:
      description: tasty
    tv:
      $ref: '#/components/responses/coffee'`

	right := `components:
  responses:
    coffee:
      $ref: '#/components/responses/tv'
    tv:
      description: tasty`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Components
	var rDoc v3.Components

	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)

	idx := index.NewSpecIndex(&lNode)
	idx2 := index.NewSpecIndex(&rNode)

	_ = lDoc.Build(context.Background(), lNode.Content[0], idx)
	_ = rDoc.Build(context.Background(), rNode.Content[0], idx2)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareComponents_OpenAPI_Responses_FullBuild_CircularRef(t *testing.T) {
	left := `components:
  responses:
    coffee:
      $ref: '#/components/responses/tv'
    tv:
      $ref: '#/components/responses/coffee'`

	right := `components:
  responses:
    coffee:
      $ref: '#/components/responses/tv'
    tv:
      $ref: '#/components/responses/coffee'`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Components
	var rDoc v3.Components

	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)

	idx := index.NewSpecIndex(&lNode)
	idx2 := index.NewSpecIndex(&rNode)

	// resolver required to check circular refs.
	re1 := index.NewResolver(idx)
	re2 := index.NewResolver(idx2)

	re1.CheckForCircularReferences()
	re2.CheckForCircularReferences()

	_ = lDoc.Build(context.Background(), lNode.Content[0], idx)
	_ = rDoc.Build(context.Background(), rNode.Content[0], idx2)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

//func TestCompareComponents_OpenAPI_Responses_Modify(t *testing.T) {
//
//	left := `responses:
// niceResponse:
//   description: hello
// badResponse:
//   description: go away please`
//
//	right := `responses:
// niceResponse:
//   description: hello my matey
// badResponse:
//   description: go away please, now!`
//
//	var lNode, rNode yaml.Node
//	_ = yaml.Unmarshal([]byte(left), &lNode)
//	_ = yaml.Unmarshal([]byte(right), &rNode)
//
//	// create low level objects
//	var lDoc v3.Components
//	var rDoc v3.Components
//	_ = low.BuildModel(lNode.Content[0], &lDoc)
//	_ = low.BuildModel(rNode.Content[0], &rDoc)
//	_ = lDoc.Build(context.Background(), lNode.Content[0], nil)
//	_ = rDoc.Build(context.Background(), rNode.Content[0], nil)
//
//	// compare.
//	extChanges := CompareComponents(&rDoc, &lDoc)
//	assert.Equal(t, 2, extChanges.TotalChanges())
//	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
//}

func TestCompareComponents_OpenAPI_Responses_Add(t *testing.T) {

	left := `responses:
  niceResponse:
    description: hello
  badResponse:
    description: go away please!`

	right := `responses:
  niceResponse:
    description: hello
  badResponse:
    description: go away please!
  indifferent:
    description: stay, or go, who cares?`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Components
	var rDoc v3.Components
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, "indifferent", extChanges.Changes[0].New)
}

func TestCompareComponents_OpenAPI_Responses_Remove(t *testing.T) {

	left := `responses:
  niceResponse:
    description: hello
  badResponse:
    description: go away please!`

	right := `responses:
  niceResponse:
    description: hello
  badResponse:
    description: go away please!
  indifferent:
    description: stay, or go, who cares?`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Components
	var rDoc v3.Components
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, "indifferent", extChanges.Changes[0].Original)
}

func TestCompareComponents_OpenAPI_Parameters_Equal(t *testing.T) {

	left := `parameters:
  param1:
    name: a parameter
  param2:
    name: another param`

	right := left

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Components
	var rDoc v3.Components
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareComponents_OpenAPI_Parameters_Added(t *testing.T) {

	left := `parameters:
  param1:
    name: a parameter
  param2:
    name: another param`

	right := `parameters:
  param1:
    name: a parameter
  param2:
    name: another param
  param3:
    name: do you like code?`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Components
	var rDoc v3.Components
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, "param3", extChanges.Changes[0].New)
}

func TestCompareComponents_OpenAPI_Parameters_Removed(t *testing.T) {

	left := `parameters:
  param1:
    name: a parameter
  param2:
    name: another param`

	right := `parameters:
  param1:
    name: a parameter
  param2:
    name: another param
  param3:
    name: do you like code?`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Components
	var rDoc v3.Components
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, "param3", extChanges.Changes[0].Original)
}

func TestCompareComponents_OpenAPI_RequestBodies_Modified(t *testing.T) {

	left := `requestBodies:
  body1:
    description: a request`

	right := `requestBodies:
  body1:
    description: a request but changed
  body2:
    description: another request, also changed`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Components
	var rDoc v3.Components
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
}

func TestCompareComponents_OpenAPI_Headers_Add(t *testing.T) {

	left := `headers:
  header1:
    description: a header`

	right := `headers:
  header1:
    description: a header
  header2:
    description: another header`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Components
	var rDoc v3.Components
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
}

func TestCompareComponents_OpenAPI_SecuritySchemes_Equal(t *testing.T) {

	left := `securitySchemes:
  scheme1:
    description: a scheme
  scheme2:
    description: another scheme`

	right := left

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Components
	var rDoc v3.Components
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareComponents_OpenAPI_SecuritySchemes_Modified(t *testing.T) {

	left := `securitySchemes:
  scheme1:
    description: a scheme
  scheme2:
    description: another scheme`

	right := `securitySchemes:
  scheme1:
    description: a scheme that changed
  scheme2:
    description: another scheme that also changed`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Components
	var rDoc v3.Components
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
}

func TestCompareComponents_OpenAPI_Links_Added(t *testing.T) {

	left := `links:
  link1:
    operationId: link1`

	right := `links:
  link1:
    operationId: somethingFresh
  link2:
    operationId: somethingNew`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Components
	var rDoc v3.Components
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
}

func TestCompareComponents_OpenAPI_Callbacks_Modified(t *testing.T) {

	left := `callbacks:
  link1:
    '{$request.query.queryUrl}':
      post:
        description: a nice callback`

	right := `callbacks:
  link1:
    '{$request.query.queryUrl}':
      post:
        description: a nice callback, but changed
  link2:
    '{$pizza.cake.burgers}':
      get:
        description: pizza and cake, and burgers, and ketchup.`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Components
	var rDoc v3.Components
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
}

func TestCompareComponents_OpenAPI_Extensions_Modified(t *testing.T) {

	left := `x-components: are done"`

	right := `x-components: I hope`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Components
	var rDoc v3.Components
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
}
