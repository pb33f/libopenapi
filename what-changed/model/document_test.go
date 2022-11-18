// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/low"
	v2 "github.com/pb33f/libopenapi/datamodel/low/v2"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestCompareDocuments_Swagger_BaseProperties_Identical(t *testing.T) {
	left := `swagger: 2.0
x-diet: tough
host: https://pb33f.io
basePath: /api
schemes: 
  - http
  - https
consumes:
  - application/json
  - apple/pie
produces:
  - application/json
  - fat/belly`

	right := left

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Swagger
	var rDoc v2.Swagger
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)

	// compare.
	extChanges := CompareDocuments(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareDocuments_Swagger_BaseProperties_Modified(t *testing.T) {
	left := `swagger: 2.0
x-diet: coke
host: https://pb33f.io
basePath: /api
schemes: 
  - http
  - https
consumes:
  - application/json
  - apple/pie
produces:
  - application/json
  - fat/belly`

	right := `swagger: 2.0.1
x-diet: pepsi
host: https://quobix.com
basePath: /new-api
schemes: 
  - ws
  - https
consumes:
  - application/json
  - apple/ice-cream
produces:
  - application/json
  - very-fat/belly`

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v2.CreateDocument(siLeft)
	rDoc, _ := v2.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(lDoc, rDoc)
	assert.Equal(t, 10, extChanges.TotalChanges())
	assert.Equal(t, 6, extChanges.TotalBreakingChanges())
}

func TestCompareDocuments_Swagger_Info_Modified(t *testing.T) {
	left := `swagger: 2.0
info:
  title: a doc
  contact:
    name: buckaroo
  license:
    url: https://pb33f.io`

	right := `swagger: 2.0
info:
  title: a doc that changed
  contact:
    name: chief buckaroo
  license:
    url: https://pb33f.io/license`

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v2.CreateDocument(siLeft)
	rDoc, _ := v2.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(lDoc, rDoc)
	assert.Equal(t, 3, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, 3, extChanges.InfoChanges.TotalChanges())
}

func TestCompareDocuments_Swagger_Info_Added(t *testing.T) {
	left := `swagger: 2.0`

	right := `swagger: 2.0
info:
  title: a doc that changed
  contact:
    name: chief buckaroo
  license:
    url: https://pb33f.io/license`

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v2.CreateDocument(siLeft)
	rDoc, _ := v2.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(lDoc, rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, v3.InfoLabel, extChanges.Changes[0].Property)
	assert.Equal(t, PropertyAdded, extChanges.Changes[0].ChangeType)
}

func TestCompareDocuments_Swagger_Info_Removed(t *testing.T) {
	left := `swagger: 2.0`

	right := `swagger: 2.0
info:
  title: a doc that changed
  contact:
    name: chief buckaroo
  license:
    url: https://pb33f.io/license`

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v2.CreateDocument(siLeft)
	rDoc, _ := v2.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(rDoc, lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, v3.InfoLabel, extChanges.Changes[0].Property)
	assert.Equal(t, PropertyRemoved, extChanges.Changes[0].ChangeType)
}

func TestCompareDocuments_Swagger_ExternalDocs_Modified(t *testing.T) {
	left := `swagger: 2.0
externalDocs:
  url: https://pb33f.io
  description: the ranch`

	right := `swagger: 2.0
externalDocs:
  url: https://pb33f.io/new
  description: the bunkhouse`

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v2.CreateDocument(siLeft)
	rDoc, _ := v2.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(lDoc, rDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, 2, extChanges.ExternalDocChanges.TotalChanges())
}

func TestCompareDocuments_Swagger_ExternalDocs_Added(t *testing.T) {
	left := `swagger: 2.0`

	right := `swagger: 2.0
externalDocs:
  url: https://pb33f.io
  description: the ranch`

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v2.CreateDocument(siLeft)
	rDoc, _ := v2.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(lDoc, rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, v3.ExternalDocsLabel, extChanges.Changes[0].Property)
	assert.Equal(t, PropertyAdded, extChanges.Changes[0].ChangeType)
	assert.NotNil(t, lDoc.GetExternalDocs())

}

func TestCompareDocuments_Swagger_ExternalDocs_Removed(t *testing.T) {
	left := `swagger: 2.0`

	right := `swagger: 2.0
externalDocs:
  url: https://pb33f.io
  description: the ranch`

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v2.CreateDocument(siLeft)
	rDoc, _ := v2.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(rDoc, lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, v3.ExternalDocsLabel, extChanges.Changes[0].Property)
	assert.Equal(t, PropertyRemoved, extChanges.Changes[0].ChangeType)
}

func TestCompareDocuments_Swagger_Security_Identical(t *testing.T) {
	left := `swagger: 2.0
security:
  - nice:
    - rice
    - spice
    - fries
  - bad:
    - glad
    - sad
    - bag`

	right := `swagger: 2.0
security:
  - bad:
    - sad
    - bag
    - glad
  - nice:
    - spice
    - rice
    - fries`

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v2.CreateDocument(siLeft)
	rDoc, _ := v2.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(lDoc, rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareDocuments_Swagger_Security_Changed(t *testing.T) {
	left := `swagger: 2.0
security:
  - nice:
    - rice
    - spice
    - fries
  - bad:
    - glad
    - sad
    - bag`

	right := `swagger: 2.0
security:
  - bad:
    - sad
    - bag
    - odd
  - nice:
    - spice
    - lego
    - fries`

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v2.CreateDocument(siLeft)
	rDoc, _ := v2.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(lDoc, rDoc)
	assert.Equal(t, 4, extChanges.TotalChanges())
	assert.Equal(t, 2, extChanges.TotalBreakingChanges())
}

func TestCompareDocuments_Swagger_Components_Schemas(t *testing.T) {
	left := `swagger: 2.0
definitions:
  burgers:
    type: int
    description: num burgers
  cakes:
    type: string
    description: your favorite cake`

	right := `swagger: 2.0
definitions:
  burgers:
    type: bool
    description: allow burgers?
  cakes:
    type: int
    description: how many cakes?`

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v2.CreateDocument(siLeft)
	rDoc, _ := v2.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(lDoc, rDoc)
	assert.Equal(t, 4, extChanges.TotalChanges())
	assert.Equal(t, 2, extChanges.TotalBreakingChanges())
}

func TestCompareDocuments_Swagger_Components_SecurityDefinitions_Identical(t *testing.T) {
	left := `swagger: 2.0
securityDefinitions:
  letMeIn:
    type: password
    description: a password that lets you in
  letMeOut:
    type: 2fa
    description: are you a robot?`

	right := `swagger: 2.0
securityDefinitions:
  letMeIn:
    type: password
    description: a password that lets you in
  letMeOut:
    type: 2fa
    description: are you a robot?`

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v2.CreateDocument(siLeft)
	rDoc, _ := v2.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(lDoc, rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareDocuments_Swagger_Components_SecurityDefinitions_Changed(t *testing.T) {
	left := `swagger: 2.0
securityDefinitions:
  letMeIn:
    type: password
    description: a password that lets you in
  letMeOut:
    type: 2fa
    description: are you a robot?`

	right := `swagger: 2.0
securityDefinitions:
  letMeIn:
    type: secret
    description: what is a secret?
  letMeOut:
    type: 2fa
    description: are you a robot?`

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v2.CreateDocument(siLeft)
	rDoc, _ := v2.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(lDoc, rDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
}

func TestCompareDocuments_Swagger_Components_Parameters_Identical(t *testing.T) {
	left := `swagger: 2.0
parameters:
  letMeIn:
    name: letsGo
    type: int
    description: lets go my friends.
  letMeOut:
    name: whyNow
    type: string
    description: why?`

	right := `swagger: 2.0
parameters:
  letMeIn:
    name: letsGo
    type: int
    description: lets go my friends.
  letMeOut:
    name: whyNow
    type: string
    description: why?`

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v2.CreateDocument(siLeft)
	rDoc, _ := v2.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(lDoc, rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareDocuments_Swagger_Components_Parameters_Changed(t *testing.T) {
	left := `swagger: 2.0
parameters:
  letMeIn:
    name: letsGo
    type: int
    description: lets go my friends. now.
  letMeOut:
    name: whyNow
    type: string
    description: why?`

	right := `swagger: 2.0
parameters:
  letMeIn:
    name: letsGoNow
    type: string
    description: lets go my friends.
  letMeOut:
    name: whyNowPlease
    type: int
    description: why should we do it?`

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v2.CreateDocument(siLeft)
	rDoc, _ := v2.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(lDoc, rDoc)
	assert.Equal(t, 6, extChanges.TotalChanges())
	assert.Equal(t, 4, extChanges.TotalBreakingChanges())
}

func TestCompareDocuments_Swagger_Components_Responses_Identical(t *testing.T) {
	left := `swagger: 2.0
responses:
  tacos:
    description: who wants a taco?
    schema:
      type: int
  pizza:
    description: who wants a pizza?
    schema:
      type: string`

	right := `swagger: 2.0
responses:
  tacos:
    description: who wants a taco?
    schema:
      type: int
  pizza:
    description: who wants a pizza?
    schema:
      type: string`

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v2.CreateDocument(siLeft)
	rDoc, _ := v2.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(lDoc, rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareDocuments_Swagger_Components_Responses_Modified(t *testing.T) {
	left := `swagger: 2.0
responses:
  tacos:
    description: who wants a taco?
    schema:
      type: int
  pizza:
    description: who wants a pizza?
    schema:
      type: string`

	right := `swagger: 2.0
responses:
  tacos:
    description: I want a taco
    schema:
      type: bool
  pizza:
    description: I need a pizza
    schema:
      type: int`

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v2.CreateDocument(siLeft)
	rDoc, _ := v2.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(lDoc, rDoc)
	assert.Equal(t, 4, extChanges.TotalChanges())
	assert.Equal(t, 2, extChanges.TotalBreakingChanges())
}

func TestCompareDocuments_Swagger_Components_Paths_Identical(t *testing.T) {
	left := `swagger: 2.0
paths:
  /nice/rice:
    get:
      description: nice rice?
  /lovely/horse:
    post:
      description: what a lovely horse.`

	right := `swagger: 2.0
paths:
  /lovely/horse:
    post:
      description: what a lovely horse.
  /nice/rice:
    get:
      description: nice rice?`

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v2.CreateDocument(siLeft)
	rDoc, _ := v2.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(lDoc, rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareDocuments_Swagger_Components_Paths_Modified(t *testing.T) {
	left := `swagger: 2.0
paths:
  /nice/rice:
    get:
      description: nice rice?
  /lovely/horse:
    post:
      description: what a lovely horse.`

	right := `swagger: 2.0
paths:
  /lovely/horse:
    put:
      description: what a lovely horse.
  /nice/rice:
    get:
      description: nice rice, but changed`

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v2.CreateDocument(siLeft)
	rDoc, _ := v2.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(lDoc, rDoc)
	assert.Equal(t, 3, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
}

func TestCompareDocuments_Swagger_Components_Paths_Modified_AgainForFun(t *testing.T) {
	left := `swagger: 2.0
paths:
  /nice/rice:
    get:
      description: nice rice?
  /lovely/horse:
    post:
      description: what a lovely horse.`

	right := `swagger: 2.0
paths:
  /lovely/horse:
    put:
      description: what a lovely horse, and shoes.
  /nice/rice:
    post:
      description: nice rice, but changed`

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v2.CreateDocument(siLeft)
	rDoc, _ := v2.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(lDoc, rDoc)
	assert.Equal(t, 4, extChanges.TotalChanges())
	assert.Equal(t, 2, extChanges.TotalBreakingChanges())
}

func TestCompareDocuments_Swagger_Components_Tags_Identical(t *testing.T) {
	left := `swagger: 2.0
tags:
  - name: a tag
    description: a nice tag
  - name: another tag
    description: this is another tag?`

	right := `swagger: 2.0
tags:
  - name: another tag
    description: this is another tag?
  - name: a tag
    description: a nice tag
  `

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v2.CreateDocument(siLeft)
	rDoc, _ := v2.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(lDoc, rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareDocuments_Swagger_Components_Tags_Modified(t *testing.T) {
	left := `swagger: 2.0
tags:
  - name: a tag
    description: a nice tag
  - name: another tag
    description: this is another tag?`

	right := `swagger: 2.0
tags:
  - name: another tag
    externalDocs:
      url: https://pb33f.io
    description: this is another tag, that changed
  - name: a tag
    description: a nice tag, modified
  `

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v2.CreateDocument(siLeft)
	rDoc, _ := v2.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(lDoc, rDoc)
	assert.Equal(t, 3, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
}

func TestCompareDocuments_OpenAPI_BaseProperties_Identical(t *testing.T) {
	left := `openapi: 3.1
x-diet: tough
jsonSchemaDialect: https://pb33f.io/schema`

	right := left

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v3.CreateDocument(siLeft)
	rDoc, _ := v3.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
	assert.NotNil(t, lDoc.GetExternalDocs())
	assert.Nil(t, lDoc.FindSecurityRequirement("chewy")) // because why not.
}

func TestCompareDocuments_OpenAPI_BaseProperties_Modified(t *testing.T) {

	left := `openapi: 3.1
x-diet: tough
jsonSchemaDialect: https://pb33f.io/schema`

	right := `openapi: 3.1.0
x-diet: fat
jsonSchemaDialect: https://pb33f.io/schema/changed`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v3.CreateDocument(siLeft)
	rDoc, _ := v3.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(lDoc, rDoc)

	assert.Equal(t, 3, extChanges.TotalChanges())
	assert.Equal(t, 2, extChanges.TotalBreakingChanges())
}

func TestCompareDocuments_OpenAPI_AddComponents(t *testing.T) {

	left := `openapi: 3.1`

	right := `openapi: 3.1
components:
  schemas:
    thing:
      type: int`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v3.CreateDocument(siLeft)
	rDoc, _ := v3.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(lDoc, rDoc)

	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyAdded, extChanges.Changes[0].ChangeType)
}

func TestCompareDocuments_OpenAPI_Removed(t *testing.T) {

	left := `openapi: 3.1`

	right := `openapi: 3.1
components:
  schemas:
    thing:
      type: int`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v3.CreateDocument(siLeft)
	rDoc, _ := v3.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(rDoc, lDoc)

	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyRemoved, extChanges.Changes[0].ChangeType)
}

func TestCompareDocuments_OpenAPI_ModifyPaths(t *testing.T) {

	left := `openapi: 3.1
paths:
  /brown/cow:
    get:
      description: brown cow
  /brown/hen:
    get:
      description: brown hen`

	right := `openapi: 3.1
paths:
  /brown/cow:
    get:
      description: brown cow modified
  /brown/hen:
    get:
      description: brown hen modified`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v3.CreateDocument(siLeft)
	rDoc, _ := v3.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(lDoc, rDoc)

	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
}

func TestCompareDocuments_OpenAPI_Identical_Security(t *testing.T) {

	left := `openapi: 3.1
security:
  - cakes:
    - chocolate
    - vanilla
  - shoes:
    - white
    - black`

	right := `openapi: 3.1
security:
  - shoes:
    - black
    - white
  - cakes:
    - vanilla
    - chocolate `

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v3.CreateDocument(siLeft)
	rDoc, _ := v3.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(lDoc, rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareDocuments_OpenAPI_ModifyComponents(t *testing.T) {

	left := `openapi: 3.1
components:
  schemas:
    athing:
      description: a schema
    nothing:
      description: nothing`

	right := `openapi: 3.1
components:
  schemas:
    athing:
      description: a schema that changed
    nothing:
      description: nothing with an update`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v3.CreateDocument(siLeft)
	rDoc, _ := v3.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(lDoc, rDoc)

	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
}

func TestCompareDocuments_OpenAPI_ModifyServers(t *testing.T) {

	left := `openapi: 3.1
servers:
  - url: https://pb33f.io
  - url: https://quobix.com`

	right := `openapi: 3.1
servers:
  - url: https://pb33f.io
    description: hello!
  - url: https://api.pb33f.io`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v3.CreateDocument(siLeft)
	rDoc, _ := v3.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(lDoc, rDoc)

	assert.Equal(t, 3, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
}

func TestCompareDocuments_OpenAPI_ModifyWebhooks(t *testing.T) {

	left := `openapi: 3.1
webhooks:
  bHook:
    get:
      description: coffee  
  aHook:
    get:
      description: jazz`

	right := `openapi: 3.1
webhooks:
  bHook:
    get:
      description: coffee in the morning
  aHook:
    get:
      description: jazz in the evening`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// have to build docs fully to get access to objects
	siLeft, _ := datamodel.ExtractSpecInfo([]byte(left))
	siRight, _ := datamodel.ExtractSpecInfo([]byte(right))

	lDoc, _ := v3.CreateDocument(siLeft)
	rDoc, _ := v3.CreateDocument(siRight)

	// compare.
	extChanges := CompareDocuments(lDoc, rDoc)

	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
}
