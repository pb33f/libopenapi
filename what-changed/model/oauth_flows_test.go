// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestCompareOAuthFlow(t *testing.T) {

	left := `authorizationUrl: cheese
tokenUrl: biscuits
refreshUrl: cake
scopes:
 riff: raff`

	right := `authorizationUrl: cheese
tokenUrl: biscuits
refreshUrl: cake
scopes:
 riff: raff`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.OAuthFlow
	var rDoc v3.OAuthFlow
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareOAuthFlow(&lDoc, &rDoc)
	assert.Nil(t, extChanges)

}

func TestCompareOAuthFlow_Modified(t *testing.T) {

	left := `authorizationUrl: toast
tokenUrl: biscuits
refreshUrl: roast
scopes:
 riff: raff
x-burgers: nice`

	right := `authorizationUrl: cheese
tokenUrl: biscuits
refreshUrl: cake
scopes:
 riff: raff
x-burgers: crispy`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.OAuthFlow
	var rDoc v3.OAuthFlow
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareOAuthFlow(&lDoc, &rDoc)
	assert.Equal(t, 3, extChanges.TotalChanges())
	assert.Equal(t, 2, extChanges.TotalBreakingChanges())
}

func TestCompareOAuthFlow_AddScope(t *testing.T) {

	left := `authorizationUrl: toast
tokenUrl: biscuits
refreshUrl: roast
scopes:
 riff: raff
x-burgers: nice`

	right := `authorizationUrl: toast
tokenUrl: biscuits
refreshUrl: roast
scopes:
  riff: raff
  tiff: taff
x-burgers: nice`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.OAuthFlow
	var rDoc v3.OAuthFlow
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareOAuthFlow(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, "taff", extChanges.Changes[0].New)
	assert.Equal(t, "tiff", extChanges.Changes[0].NewObject)
}

func TestCompareOAuthFlow_RemoveScope(t *testing.T) {

	left := `authorizationUrl: toast
tokenUrl: biscuits
refreshUrl: roast
scopes:
 riff: raff
x-burgers: nice`

	right := `authorizationUrl: toast
tokenUrl: biscuits
refreshUrl: roast
scopes:
  riff: raff
  tiff: taff
x-burgers: nice`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.OAuthFlow
	var rDoc v3.OAuthFlow
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareOAuthFlow(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, "taff", extChanges.Changes[0].Original)
	assert.Equal(t, "tiff", extChanges.Changes[0].OriginalObject)
}

func TestCompareOAuthFlow_ModifyScope(t *testing.T) {

	left := `authorizationUrl: toast
tokenUrl: biscuits
refreshUrl: roast
scopes:
 riff: ruffles
x-burgers: nice`

	right := `authorizationUrl: toast
tokenUrl: biscuits
refreshUrl: roast
scopes:
  riff: raff
x-burgers: nice`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.OAuthFlow
	var rDoc v3.OAuthFlow
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareOAuthFlow(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, "raff", extChanges.Changes[0].New)
	assert.Equal(t, "raff", extChanges.Changes[0].NewObject)
	assert.Equal(t, "ruffles", extChanges.Changes[0].Original)
	assert.Equal(t, "ruffles", extChanges.Changes[0].OriginalObject)
}

func TestCompareOAuthFlows(t *testing.T) {
	left := `implicit:
  authorizationUrl: cheese
password: 
  authorizationUrl: cake
clientCredentials:
  authorizationUrl: chicken
authorizationCode:
  authorizationUrl: chalk
x-coke: cola`

	right := `implicit:
  authorizationUrl: cheese
password: 
  authorizationUrl: cake
clientCredentials:
  authorizationUrl: chicken
authorizationCode:
  authorizationUrl: chalk
x-coke: cola`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.OAuthFlows
	var rDoc v3.OAuthFlows
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareOAuthFlows(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareOAuthFlows_AddEverything(t *testing.T) {
	left := `x-coke: cola`

	right := `implicit:
  authorizationUrl: cheese
password: 
  authorizationUrl: cake
clientCredentials:
  authorizationUrl: chicken
authorizationCode:
  authorizationUrl: chalk
x-coke: cola`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.OAuthFlows
	var rDoc v3.OAuthFlows
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareOAuthFlows(&lDoc, &rDoc)
	assert.Equal(t, 4, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
}

func TestCompareOAuthFlows_RemoveEverything(t *testing.T) {
	left := `x-coke: cola`

	right := `implicit:
  authorizationUrl: cheese
password: 
  authorizationUrl: cake
clientCredentials:
  authorizationUrl: chicken
authorizationCode:
  authorizationUrl: chalk
x-coke: cola`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.OAuthFlows
	var rDoc v3.OAuthFlows
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareOAuthFlows(&rDoc, &lDoc)
	assert.Equal(t, 4, extChanges.TotalChanges())
	assert.Equal(t, 4, extChanges.TotalBreakingChanges())
}

func TestCompareOAuthFlows_ModifyEverything(t *testing.T) {
	left := `implicit:
  authorizationUrl: cheese
password: 
  authorizationUrl: cake
clientCredentials:
  authorizationUrl: chicken
authorizationCode:
  authorizationUrl: chalk
x-coke: cola`

	right := `implicit:
  authorizationUrl: herbs
password: 
  authorizationUrl: coffee
clientCredentials:
  authorizationUrl: tea
authorizationCode:
  authorizationUrl: pasta
x-coke: cherry`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.OAuthFlows
	var rDoc v3.OAuthFlows
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareOAuthFlows(&lDoc, &rDoc)
	assert.Equal(t, 5, extChanges.TotalChanges())
	assert.Equal(t, 4, extChanges.TotalBreakingChanges())
}
