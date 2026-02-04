// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestCompareOAuthFlow(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
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
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare
	extChanges := CompareOAuthFlow(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareOAuthFlow_Modified(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
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
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare
	extChanges := CompareOAuthFlow(&lDoc, &rDoc)
	assert.Equal(t, 3, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 3)
	assert.Equal(t, 2, extChanges.TotalBreakingChanges())
}

func TestCompareOAuthFlow_AddScope(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
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
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare
	extChanges := CompareOAuthFlow(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, "taff", extChanges.Changes[0].New)
	assert.Equal(t, "tiff", extChanges.Changes[0].NewObject)
}

func TestCompareOAuthFlow_RemoveScope(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
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
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare
	extChanges := CompareOAuthFlow(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, "taff", extChanges.Changes[0].Original)
	assert.Equal(t, "tiff", extChanges.Changes[0].OriginalObject)
}

func TestCompareOAuthFlow_ModifyScope(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
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
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare
	extChanges := CompareOAuthFlow(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, "raff", extChanges.Changes[0].New)
	assert.Equal(t, "raff", extChanges.Changes[0].NewObject)
	assert.Equal(t, "ruffles", extChanges.Changes[0].Original)
	assert.Equal(t, "ruffles", extChanges.Changes[0].OriginalObject)
}

func TestCompareOAuthFlows(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
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
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare
	extChanges := CompareOAuthFlows(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareOAuthFlows_AddEverything(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
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
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare
	extChanges := CompareOAuthFlows(&lDoc, &rDoc)
	assert.Equal(t, 4, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 4)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
}

func TestCompareOAuthFlows_RemoveEverything(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
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
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare
	extChanges := CompareOAuthFlows(&rDoc, &lDoc)
	assert.Equal(t, 4, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 4)
	assert.Equal(t, 4, extChanges.TotalBreakingChanges())
}

func TestCompareOAuthFlows_ModifyEverything(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
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
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare
	extChanges := CompareOAuthFlows(&lDoc, &rDoc)
	assert.Equal(t, 5, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 5)
	assert.Equal(t, 4, extChanges.TotalBreakingChanges())
}

func TestCompareOAuthFlows_DeviceFlowAdded(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `implicit:
  authorizationUrl: https://auth.example.com`

	right := `implicit:
  authorizationUrl: https://auth.example.com
device:
  tokenUrl: https://oauth2.example.com/device/token
  scopes:
    read: read access`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.OAuthFlows
	var rDoc v3.OAuthFlows
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare
	extChanges := CompareOAuthFlows(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges()) // adding device flow is not breaking

	allChanges := extChanges.GetAllChanges()
	assert.Equal(t, ObjectAdded, allChanges[0].ChangeType)
	assert.Equal(t, v3.DeviceLabel, allChanges[0].Property)
}

func TestCompareOAuthFlows_DeviceFlowRemoved(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `implicit:
  authorizationUrl: https://auth.example.com
device:
  tokenUrl: https://oauth2.example.com/device/token
  scopes:
    read: read access`

	right := `implicit:
  authorizationUrl: https://auth.example.com`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.OAuthFlows
	var rDoc v3.OAuthFlows
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare
	extChanges := CompareOAuthFlows(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges()) // removing device flow is breaking

	allChanges := extChanges.GetAllChanges()
	assert.Equal(t, ObjectRemoved, allChanges[0].ChangeType)
	assert.Equal(t, v3.DeviceLabel, allChanges[0].Property)

	// Test that DeviceChanges is not included in GetAllChanges when removed
	assert.Nil(t, extChanges.DeviceChanges)
}

func TestCompareOAuthFlows_DeviceFlowModified(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `device:
  tokenUrl: https://oauth2.example.com/device/token
  scopes:
    read: read access`

	right := `device:
  tokenUrl: https://oauth2.example.com/device/token-v2
  scopes:
    read: read access
    write: write access`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.OAuthFlows
	var rDoc v3.OAuthFlows
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare
	extChanges := CompareOAuthFlows(&lDoc, &rDoc)
	assert.NotNil(t, extChanges)
	assert.NotNil(t, extChanges.DeviceChanges)

	// DeviceChanges should be included in GetAllChanges and TotalChanges
	assert.Equal(t, 2, extChanges.TotalChanges())         // tokenUrl change + scope addition
	assert.Equal(t, 1, extChanges.TotalBreakingChanges()) // tokenUrl change is breaking

	allChanges := extChanges.GetAllChanges()
	assert.Len(t, allChanges, 2) // should include changes from DeviceChanges
}

func TestCompareOAuthFlows_WithAllFlowsIncludingDevice(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `implicit:
  authorizationUrl: cheese
password:
  tokenUrl: cake
clientCredentials:
  tokenUrl: chicken
authorizationCode:
  authorizationUrl: chalk
device:
  tokenUrl: https://oauth2.example.com/device/token
x-coke: cola`

	right := `implicit:
  authorizationUrl: herbs
password:
  tokenUrl: coffee
clientCredentials:
  tokenUrl: tea
authorizationCode:
  authorizationUrl: pasta
device:
  tokenUrl: https://oauth2.example.com/device/token-v2
x-coke: cherry`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.OAuthFlows
	var rDoc v3.OAuthFlows
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare
	extChanges := CompareOAuthFlows(&lDoc, &rDoc)
	assert.NotNil(t, extChanges)
	assert.NotNil(t, extChanges.DeviceChanges)

	// Should include all changes from all flows
	totalChanges := extChanges.TotalChanges()
	assert.Equal(t, 6, totalChanges) // 4 auth URL/token URL changes + 1 device token URL change + 1 extension change

	totalBreakingChanges := extChanges.TotalBreakingChanges()
	assert.Equal(t, 5, totalBreakingChanges) // all URL changes are breaking

	allChanges := extChanges.GetAllChanges()
	assert.Len(t, allChanges, 6) // should include all changes
}
