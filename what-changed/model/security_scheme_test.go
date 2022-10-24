// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/v2"
	"github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestCompareSecuritySchemes_v2(t *testing.T) {

	left := `type: string
description: a thing
flow: heavy
authorizationUrl: https://somewheremagicandnotreal.com
tokenUrl: https://amadeupplacefilledwithendlesstimeandbeer.com
x-beer: tasty`

	right := `type: string
description: a thing
flow: heavy
authorizationUrl: https://somewheremagicandnotreal.com
tokenUrl: https://amadeupplacefilledwithendlesstimeandbeer.com
x-beer: tasty`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.SecurityScheme
	var rDoc v2.SecurityScheme
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareSecuritySchemes(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareSecuritySchemes_v2_ModifyProps(t *testing.T) {

	left := `type: int
description: who cares if this changes?
flow: very heavy
x-beer: tasty`

	right := `type: string
description: a thing
flow: heavy
x-beer: very tasty`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.SecurityScheme
	var rDoc v2.SecurityScheme
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareSecuritySchemes(&lDoc, &rDoc)
	assert.Equal(t, 4, extChanges.TotalChanges())
	assert.Equal(t, 2, extChanges.TotalBreakingChanges())
	assert.Equal(t, Modified, extChanges.Changes[0].ChangeType)
	assert.Equal(t, Modified, extChanges.Changes[1].ChangeType)
	assert.Equal(t, Modified, extChanges.Changes[2].ChangeType)
	assert.Equal(t, Modified, extChanges.ExtensionChanges.Changes[0].ChangeType)
}

func TestCompareSecuritySchemes_v2_AddScope(t *testing.T) {

	left := `description: I am a thing`

	right := `description: I am a thing
scopes:
  pizza:pie
  lemon:sky`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.SecurityScheme
	var rDoc v2.SecurityScheme
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareSecuritySchemes(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.ScopesLabel, extChanges.Changes[0].Property)
}

func TestCompareSecuritySchemes_v2_RemoveScope(t *testing.T) {

	left := `description: I am a thing`

	right := `description: I am a thing
scopes:
  pizza:pie
  lemon:sky`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.SecurityScheme
	var rDoc v2.SecurityScheme
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareSecuritySchemes(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.ScopesLabel, extChanges.Changes[0].Property)
}

func TestCompareSecuritySchemes_v2_ModifyScope(t *testing.T) {

	left := `scopes:
  pizza: pie`

	right := `scopes:
  pizza: pie
  lemon: sky`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.SecurityScheme
	var rDoc v2.SecurityScheme
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareSecuritySchemes(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.ScopesChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.ScopesLabel, extChanges.ScopesChanges.Changes[0].Property)
}

func TestCompareSecuritySchemes_v3(t *testing.T) {

	left := `type: string
description: a thing
scheme: fishy
bearerFormat: golden
x-beer: tasty`

	right := `x-beer: tasty
type: string
bearerFormat: golden
scheme: fishy
description: a thing`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.SecurityScheme
	var rDoc v3.SecurityScheme
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareSecuritySchemes(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareSecuritySchemes_v3_ModifyProps(t *testing.T) {

	left := `type: string
description: a thing
scheme: fishy
bearerFormat: golden
x-beer: tasty`

	right := `type: int
description: a thing that can change without breaking
scheme: smokey
bearerFormat: amber
x-beer: cool`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.SecurityScheme
	var rDoc v3.SecurityScheme
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareSecuritySchemes(&lDoc, &rDoc)
	assert.Equal(t, 5, extChanges.TotalChanges())
	assert.Equal(t, 2, extChanges.TotalBreakingChanges())
	assert.Equal(t, Modified, extChanges.Changes[0].ChangeType)
	assert.Equal(t, Modified, extChanges.Changes[1].ChangeType)
	assert.Equal(t, Modified, extChanges.Changes[2].ChangeType)
	assert.Equal(t, Modified, extChanges.Changes[3].ChangeType)
	assert.Equal(t, Modified, extChanges.ExtensionChanges.Changes[0].ChangeType)
}

func TestCompareSecuritySchemes_v3_AddFlows(t *testing.T) {

	left := `type: oauth`

	right := `type: oauth
flows:
  implicit:
    tokenUrl: https://magichappyclappyland.com`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.SecurityScheme
	var rDoc v3.SecurityScheme
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareSecuritySchemes(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
}

func TestCompareSecuritySchemes_v3_RemoveFlows(t *testing.T) {

	left := `type: oauth`

	right := `type: oauth
flows:
  implicit:
    tokenUrl: https://magichappyclappyland.com`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.SecurityScheme
	var rDoc v3.SecurityScheme
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareSecuritySchemes(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
}

func TestCompareSecuritySchemes_v3_ModifyFlows(t *testing.T) {

	left := `type: oauth
flows:
  implicit:
    tokenUrl: https://magichappyclappyland.com`

	right := `type: oauth
flows:
  implicit:
    tokenUrl: https://chickennuggetsandchickensoup.com`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.SecurityScheme
	var rDoc v3.SecurityScheme
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareSecuritySchemes(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, Modified, extChanges.OAuthFlowChanges.ImplicitChanges.Changes[0].ChangeType)
}
