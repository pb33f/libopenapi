// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	v2 "github.com/pb33f/libopenapi/datamodel/low/v2"
	v3low "github.com/pb33f/libopenapi/datamodel/low/v3"
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
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

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
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, 1, extChanges.SchemaChanges["thing2"].TotalChanges())

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
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
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
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "thing3", extChanges.Changes[0].Original)
}

func TestCompareComponents_Swagger_Parameters_Equal(t *testing.T) {

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
  name: snooze`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.ParameterDefinitions
	var rDoc v2.ParameterDefinitions
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareComponents_Swagger_Parameters_Modified(t *testing.T) {

	left := `param1:
  name: nap
param2:
  name: sleep
param3: 
  name: snooze
`
	right := `param1:
  name: WIDE AWAKE
param2:
  name: sleep
param3: 
  name: KINDA SNOOZ`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.ParameterDefinitions
	var rDoc v2.ParameterDefinitions
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Equal(t, 2, extChanges.TotalBreakingChanges())
	assert.Equal(t, Modified, extChanges.ParameterChanges["param1"].Changes[0].ChangeType)
	assert.Equal(t, "WIDE AWAKE", extChanges.ParameterChanges["param1"].Changes[0].New)
	assert.Equal(t, "KINDA SNOOZ", extChanges.ParameterChanges["param3"].Changes[0].New)
	assert.Equal(t, v3low.NameLabel, extChanges.ParameterChanges["param1"].Changes[0].Property)

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
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
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
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "param4", extChanges.Changes[0].Original)
}

func TestCompareComponents_Swagger_Responses_Equal(t *testing.T) {

	left := `resp1:
  description: hi!
resp2:
  description: bye!
`
	right := `resp1:
  description: hi!
resp2:
  description: bye!`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.ResponsesDefinitions
	var rDoc v2.ResponsesDefinitions
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareComponents_Swagger_Responses_Modified(t *testing.T) {

	left := `resp1:
  description: hi!
resp2:
  description: bye!
`
	right := `resp1:
  description: hi!
resp2:
  description: oh, so you want to change huh?`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.ResponsesDefinitions
	var rDoc v2.ResponsesDefinitions
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)

	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, 1, extChanges.ResponsesChanges["resp2"].TotalChanges())
	assert.Equal(t, v3low.DescriptionLabel, extChanges.ResponsesChanges["resp2"].Changes[0].Property)
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
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)

	assert.Equal(t, 1, extChanges.TotalChanges())
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
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&rDoc, &lDoc)

	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, "resp3", extChanges.Changes[0].Original)
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v2.ResponsesLabel, extChanges.Changes[0].Property)
}

func TestCompareComponents_Swagger_SecurityDefinitions_Equal(t *testing.T) {

	left := `scheme1:
  description: hi!
scheme2:
  description: bye!
`
	right := `scheme1:
  description: hi!
scheme2:
  description: bye!`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.SecurityDefinitions
	var rDoc v2.SecurityDefinitions
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareComponents_Swagger_SecurityDefinitions_Modified(t *testing.T) {

	left := `scheme1:
  description: hi!
scheme2:
  description: bye!
`
	right := `scheme1:
  description: hi!
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
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareComponents(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, 1, extChanges.SecuritySchemeChanges["scheme2"].TotalChanges())
	assert.Equal(t, v3low.DescriptionLabel, extChanges.SecuritySchemeChanges["scheme2"].Changes[0].Property)
}
