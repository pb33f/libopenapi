// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	v2 "github.com/pb33f/libopenapi/datamodel/low/v2"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestCompareOperations_V2(t *testing.T) {
	left := `tags:
  - one
  - two
summary: hello
description: hello there my pal
operationId: mintyFresh
consumes:
  - pizza
  - cake
produces:
  - toast
  - jam
parameters:
  - name: jimmy
  - name: jammy
    in: freezer`

	right := left

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Operation
	var rDoc v2.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareOperations_V2_ModifyParam(t *testing.T) {
	left := `tags:
  - one
  - two
summary: hello
description: hello there my pal
operationId: mintyFresh
consumes:
  - pizza
  - cake
produces:
  - toast
  - jam
parameters:
  - name: jimmy
  - name: jammy
    in: freezer`

	right := `tags:
  - one
  - two
summary: hello
description: hello there my pal
operationId: mintyFresh
consumes:
  - pizza
  - cake
produces:
  - toast
  - jam
parameters:
  - name: jimmy
  - name: jammy
    in: fridge`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Operation
	var rDoc v2.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
}

func TestCompareOperations_V2_AddParamProperty(t *testing.T) {
	left := `tags:
  - one
  - two
summary: hello
description: hello there my pal
operationId: mintyFresh
consumes:
  - pizza
  - cake
produces:
  - toast
  - jam`

	right := `tags:
  - one
  - two
summary: hello
description: hello there my pal
operationId: mintyFresh
consumes:
  - pizza
  - cake
produces:
  - toast
  - jam
parameters:
  - name: jimmy
  - name: jammy
    in: freezer`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Operation
	var rDoc v2.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "parameters", extChanges.Changes[0].Property)
}

func TestCompareOperations_V2_RemoveParamProperty(t *testing.T) {
	left := `tags:
  - one
  - two
summary: hello
description: hello there my pal
operationId: mintyFresh
consumes:
  - pizza
  - cake
produces:
  - toast
  - jam`

	right := `tags:
  - one
  - two
summary: hello
description: hello there my pal
operationId: mintyFresh
consumes:
  - pizza
  - cake
produces:
  - toast
  - jam
parameters:
  - name: jimmy
  - name: jammy
    in: freezer`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Operation
	var rDoc v2.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "parameters", extChanges.Changes[0].Property)
}

func TestCompareOperations_V2_AddParam(t *testing.T) {
	left := `tags:
  - one
  - two
summary: hello
description: hello there my pal
operationId: mintyFresh
consumes:
  - pizza
  - cake
produces:
  - toast
  - jam
parameters:
  - name: jimmy
  - name: jammy
    in: freezer`

	right := `tags:
  - one
  - two
summary: hello
description: hello there my pal
operationId: mintyFresh
consumes:
  - pizza
  - cake
produces:
  - toast
  - jam
parameters:
  - name: jimmy
  - name: jammy
    in: freezer
  - name: jummy
    in: oven`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Operation
	var rDoc v2.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges()) // Adding param without required field (defaults to false) is not breaking
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
}

func TestCompareOperations_V2_AddRequiredParam(t *testing.T) {
	left := `tags:
  - one
  - two
summary: hello
description: hello there my pal
operationId: mintyFresh
consumes:
  - pizza
  - cake
produces:
  - toast
  - jam
parameters:
  - name: jimmy
  - name: jammy
    in: freezer`

	right := `tags:
  - one
  - two
summary: hello
description: hello there my pal
operationId: mintyFresh
consumes:
  - pizza
  - cake
produces:
  - toast
  - jam
parameters:
  - name: jimmy
  - name: jammy
    in: freezer
  - name: jummyRequired
    in: oven
    required: true`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Operation
	var rDoc v2.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges()) // The required param should be breaking
}

func TestCompareOperations_V2_RemoveParam(t *testing.T) {
	left := `tags:
  - one
  - two
summary: hello
description: hello there my pal
operationId: mintyFresh
consumes:
  - pizza
  - cake
produces:
  - toast
  - jam
parameters:
  - name: jimmy
  - name: jammy
    in: freezer`

	right := `tags:
  - one
  - two
summary: hello
description: hello there my pal
operationId: mintyFresh
consumes:
  - pizza
  - cake
produces:
  - toast
  - jam
parameters:
  - name: jimmy
  - name: jammy
    in: freezer
  - name: jummy
    in: oven`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Operation
	var rDoc v2.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "jummy", extChanges.Changes[0].Original)
}

func TestCompareOperations_V2_ModifyTag(t *testing.T) {
	left := `tags:
  - one
  - two
summary: hello
description: hello there my pal
operationId: mintyFresh
consumes:
  - pizza
  - cake
produces:
  - toast
  - jam
parameters:
  - name: jimmy
  - name: jammy
    in: freezer`

	right := `tags:
  - one
  - twenty
summary: hello
description: hello there my pal
operationId: mintyFresh
consumes:
  - pizza
  - cake
produces:
  - toast
  - jam
parameters:
  - name: jimmy
  - name: jammy
    in: freezer`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Operation
	var rDoc v2.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 2)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, PropertyAdded, extChanges.Changes[1].ChangeType)
}

func TestCompareOperations_V2_AddTag(t *testing.T) {
	left := `tags:
  - one
  - two
summary: hello
description: hello there my pal
operationId: mintyFresh
consumes:
  - pizza
  - cake
produces:
  - toast
  - jam
parameters:
  - name: jimmy
  - name: jammy
    in: freezer`

	right := `tags:
  - one
  - two
  - three
summary: hello
description: hello there my pal
operationId: mintyFresh
consumes:
  - pizza
  - cake
produces:
  - toast
  - jam
parameters:
  - name: jimmy
  - name: jammy
    in: freezer`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Operation
	var rDoc v2.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.TagsLabel, extChanges.Changes[0].Property)
}

func TestCompareOperations_V2_Modify_ProducesConsumesSchemes(t *testing.T) {
	left := `produces:
  - electricity
consumes:
  - oil
schemes:
  - burning`

	right := `produces:
  - heat
consumes:
  - wind
schemes:
  - blowing`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Operation
	var rDoc v2.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 6, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 6)
	assert.Equal(t, 3, extChanges.TotalBreakingChanges())
}

func TestCompareOperations_V2_Modify_ExtDocs_Responses(t *testing.T) {
	left := `externalDocs:
  url: https://pb33f.io/old
responses:
  200:
    description: OK matey!
`

	right := `externalDocs:
  url: https://pb33f.io/new
responses:
  200:
    description: OK me matey!`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Operation
	var rDoc v2.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 2)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, Modified, extChanges.ResponsesChanges.ResponseChanges["200"].Changes[0].ChangeType)
	assert.Equal(t, Modified, extChanges.ExternalDocChanges.Changes[0].ChangeType)
}

func TestCompareOperations_V2_AddRemoveResponses(t *testing.T) {
	left := `operationId: updatePet
responses:
  '400':
    description: Invalid ID supplied
  '404':
    description: Pet not found
  '405':
    description: Validation exception`

	right := `operationId: updatePet
responses:
  '401':
    description: Invalid ID supplied
  '404':
    description: Pet not found
  '405':
    description: Validation exception`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Operation
	var rDoc v2.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 2)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
}

func TestCompareOperations_V2_Add_ExtDocs_Responses(t *testing.T) {
	left := `operationId: nuggets`

	right := `operationId: nuggets
externalDocs:
  url: https://pb33f.io/new
responses:
  200:
    description: OK me matey!`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Operation
	var rDoc v2.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 2)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, PropertyAdded, extChanges.Changes[1].ChangeType)
}

func TestCompareOperations_V2_Remove_ExtDocs_Responses(t *testing.T) {
	left := `operationId: nuggets`

	right := `operationId: nuggets
externalDocs:
  url: https://pb33f.io/new
responses:
  200:
    description: OK me matey!`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Operation
	var rDoc v2.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&rDoc, &lDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 2)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, PropertyRemoved, extChanges.Changes[1].ChangeType)
}

func TestCompareOperations_V2_AddSecurityReq_Role(t *testing.T) {
	left := `operationId: nuggets
security:
  - things:
    - stuff
    - junk`

	right := `operationId: nuggets
security:
  - things:
    - stuff
    - junk
    - crap`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Operation
	var rDoc v2.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.SecurityRequirementChanges[0].Changes[0].ChangeType)
	assert.Equal(t, "crap", extChanges.SecurityRequirementChanges[0].Changes[0].New)
}

func TestCompareOperations_V2_RemoveSecurityReq_Role(t *testing.T) {
	left := `operationId: nuggets
security:
  - things:
    - stuff
    - junk`

	right := `operationId: nuggets
security:
  - things:
    - stuff
    - junk
    - crap`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Operation
	var rDoc v2.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.SecurityRequirementChanges[0].Changes[0].ChangeType)
	assert.Equal(t, "crap", extChanges.SecurityRequirementChanges[0].Changes[0].Original)
}

func TestCompareOperations_V2_AddSecurityRequirement(t *testing.T) {
	left := `operationId: nuggets
security:
  - things:
    - stuff
    - junk`

	right := `operationId: nuggets
security:
  - things:
    - stuff
    - junk
  - thongs:
    - small
    - smelly`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Operation
	var rDoc v2.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "thongs", extChanges.Changes[0].New)
}

func TestCompareOperations_V2_RemoveSecurityRequirement(t *testing.T) {
	left := `operationId: nuggets
security:
  - things:
    - stuff
    - junk`

	right := `operationId: nuggets
security:
  - things:
    - stuff
    - junk
  - thongs:
    - small
    - smelly`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Operation
	var rDoc v2.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "thongs", extChanges.Changes[0].Original)
}

func TestCompareOperations_V3(t *testing.T) {
	left := `tags:
  - one
  - two
summary: hello
description: hello there buddy.
operationId: aTest
requestBody:
  description: a request
  content:
    fish/cake:
      schema:
        type: int
responses:
  "200":
    description: OK
security:
  - beer:
    - burgers
    - chips
    - beans
parameters:
  - name: honey
  - name: bunny
    in: fridge`

	right := left

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Operation
	var rDoc v3.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareOperations_V3_ModifyParam(t *testing.T) {
	left := `parameters:
  - name: honey
  - name: bunny
    in: fridge`

	right := `parameters:
  - name: honey
  - name: bunny
    in: attic`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Operation
	var rDoc v3.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, Modified, extChanges.ParameterChanges[0].Changes[0].ChangeType)
}

func TestCompareOperations_V3_AddParam(t *testing.T) {
	left := `parameters:
  - name: honey
  - name: bunny
    in: fridge`

	right := `parameters:
  - name: honey
  - name: bunny
    in: fridge
  - name: pb33f
    in: the_ranch`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Operation
	var rDoc v3.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges()) // Adding param without required field (defaults to false) is not breaking
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
}

func TestCompareOperations_V3_AddRequiredParam(t *testing.T) {
	left := `parameters:
  - name: honey
  - name: bunny
    in: fridge`

	right := `parameters:
  - name: honey
  - name: bunny
    in: fridge
  - name: pb33fRequired
    in: the_ranch
    required: true`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Operation
	var rDoc v3.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges()) // The required param should be breaking
}

func TestCompareOperations_V3_RemoveParam(t *testing.T) {
	left := `parameters:
  - name: honey
  - name: bunny
    in: fridge`

	right := `parameters:
  - name: honey
  - name: bunny
    in: fridge
  - name: pb33f
    in: the_ranch`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Operation
	var rDoc v3.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
}

func TestCompareOperations_V3_RemoveParams(t *testing.T) {
	left := `operationId: ohNoes
parameters:
  - name: honey
  - name: bunny
    in: fridge`

	right := `operationId: ohNoes`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Operation
	var rDoc v3.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyRemoved, extChanges.Changes[0].ChangeType)
}

func TestCompareOperations_V3_AddParams(t *testing.T) {
	left := `operationId: ohNoes
parameters:
  - name: honey
  - name: bunny
    in: fridge`

	right := `operationId: ohNoes`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Operation
	var rDoc v3.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyAdded, extChanges.Changes[0].ChangeType)
}

func TestCompareOperations_V3_ModifyTag(t *testing.T) {
	left := `tags:
  - one
  - two
summary: hello
description: hello there my pal
operationId: mintyFresh
`

	right := `tags:
  - one
  - twenty
summary: hello
description: hello there my pal
operationId: mintyFresh
`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Operation
	var rDoc v3.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 2)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, PropertyAdded, extChanges.Changes[1].ChangeType)
}

func TestCompareOperations_V3_AddTag(t *testing.T) {
	left := `tags:
  - one
  - two
summary: hello
description: hello there my pal
operationId: mintyFresh
`

	right := `tags:
  - one
  - two
  - three
summary: hello
description: hello there my pal
operationId: mintyFresh
`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Operation
	var rDoc v3.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.TagsLabel, extChanges.Changes[0].Property)
}

func TestCompareOperations_V3_ModifyServers(t *testing.T) {
	left := `servers:
  - url: https://pb33f.io
  - description: nourl!`

	right := `servers:
  - url: https://pb33f.io
    description: new!
  - description: nourl!`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Operation
	var rDoc v3.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyAdded, extChanges.ServerChanges[0].Changes[0].ChangeType)
}

func TestCompareOperations_V3_ModifyCallback(t *testing.T) {
	left := `callbacks:
  myCallback:
    '{$request.query.queryUrl}':
      post:
        description: something old`

	right := `callbacks:
  myCallback:
    '{$request.query.queryUrl}':
      post:
        description: something new!`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Operation
	var rDoc v3.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, Modified, extChanges.
		CallbackChanges["myCallback"].
		ExpressionChanges["{$request.query.queryUrl}"].
		PostChanges.Changes[0].ChangeType)
}

func TestCompareOperations_V3_AddCallback(t *testing.T) {
	left := `callbacks:
  myCallback:
    '{$request.query.queryUrl}':
      post:
        description: something old`

	right := `callbacks:
  myCallback:
    '{$request.query.queryUrl}':
      post:
        description: something old
  aNewCallback:
    aLovelyHorse:
      post:
        description: something new!`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Operation
	var rDoc v3.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
}

func TestCompareOperations_V3_AddCallbacks(t *testing.T) {
	left := `operationId: 123`

	right := `operationId: 123
callbacks:
  myCallback:
    '{$request.query.queryUrl}':
      post:
        description: something old
  aNewCallback:
    aLovelyHorse:
      post:
        description: something new!`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Operation
	var rDoc v3.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyAdded, extChanges.Changes[0].ChangeType)
}

func TestCompareOperations_V3_RemoveCallbacks(t *testing.T) {
	left := `operationId: 123`

	right := `operationId: 123
callbacks:
  myCallback:
    '{$request.query.queryUrl}':
      post:
        description: something old
  aNewCallback:
    aLovelyHorse:
      post:
        description: something new!`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Operation
	var rDoc v3.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyRemoved, extChanges.Changes[0].ChangeType)
}

func TestCompareOperations_V3_RemoveCallback(t *testing.T) {
	left := `callbacks:
  myCallback:
    '{$request.query.queryUrl}':
      post:
        description: something old`

	right := `callbacks:
  myCallback:
    '{$request.query.queryUrl}':
      post:
        description: something old
  aNewCallback:
    aLovelyHorse:
      post:
        description: something new!`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Operation
	var rDoc v3.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
}

func TestCompareOperations_V3_AddServer(t *testing.T) {
	left := `servers:
  - url: https://pb33f.io`

	right := `servers:
  - url: https://pb33f.io
  - url: https://quobix.com`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Operation
	var rDoc v3.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.ServerChanges[0].Changes[0].ChangeType)
}

func TestCompareOperations_V3_RemoveServer(t *testing.T) {
	left := `servers:
  - url: https://pb33f.io`

	right := `servers:
  - url: https://pb33f.io
  - url: https://quobix.com`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Operation
	var rDoc v3.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.ServerChanges[0].Changes[0].ChangeType)
}

func TestCompareOperations_V3_AddServerToOp(t *testing.T) {
	left := `operationId: noServers!`

	right := `operationId: noServers!
servers:
  - url: https://pb33f.io
  - url: https://quobix.com`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Operation
	var rDoc v3.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyAdded, extChanges.ServerChanges[0].Changes[0].ChangeType)
}

func TestCompareOperations_V3_RemoveServerFromOp(t *testing.T) {
	left := `operationId: noServers!`

	right := `operationId: noServers!
servers:
  - url: https://pb33f.io
  - url: https://quobix.com`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Operation
	var rDoc v3.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyRemoved, extChanges.ServerChanges[0].Changes[0].ChangeType)
}

func TestCompareOperations_V3_ModifySecurity(t *testing.T) {
	left := `operationId: coldSecurity
security:
  - winter:
    - cold`

	right := `operationId: coldSecurity
security:
  - winter:
    - cold
    - brrr.`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Operation
	var rDoc v3.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.SecurityRequirementChanges[0].Changes[0].ChangeType)
}

func TestCompareOperations_V3_AddSecurity(t *testing.T) {
	left := `operationId: coldSecurity
security: []`

	right := `operationId: coldSecurity
security:
  - winter:
    - cold`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Operation
	var rDoc v3.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Empty(t, extChanges.SecurityRequirementChanges)
}

func TestCompareOperations_V3_RemoveSecurity(t *testing.T) {
	left := `operationId: coldSecurity
security:
  - winter:
    - cold`

	right := `operationId: coldSecurity
security: []`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Operation
	var rDoc v3.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Empty(t, extChanges.SecurityRequirementChanges)
}

func TestCompareOperations_V3_ModifyRequestBody(t *testing.T) {
	left := `requestBody:
  description: jammy`

	right := `requestBody:
  description: gooey`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Operation
	var rDoc v3.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, Modified, extChanges.RequestBodyChanges.Changes[0].ChangeType)
}

func TestCompareOperations_V3_AddRequestBody(t *testing.T) {
	left := `operationId: noRequestBody`

	right := `operationId: noRequestBody
requestBody:
  description: jammy`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Operation
	var rDoc v3.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyAdded, extChanges.Changes[0].ChangeType)
}

func TestCompareOperations_V3_ModifyExtension(t *testing.T) {
	left := `x-pizza: yummy`

	right := `x-pizza: yammy!`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Operation
	var rDoc v3.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, Modified, extChanges.ExtensionChanges.Changes[0].ChangeType)
}

func TestCompareOperations_V3_RemoveRequestBody(t *testing.T) {
	left := `operationId: noRequestBody`

	right := `operationId: noRequestBody
requestBody:
  description: jammy`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Operation
	var rDoc v3.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyRemoved, extChanges.Changes[0].ChangeType)
}

func TestComparePathItem_V3_AddOptionalParam(t *testing.T) {
	left := `operationId: listBurgerDressings`

	right := `operationId: listBurgerDressings
parameters:
  - in: head
    name: burgerId
    required: false`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Operation
	var rDoc v3.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
}

func TestComparePathItem_V2_AddOptionalParam(t *testing.T) {
	left := `operationId: listBurgerDressings`

	right := `operationId: listBurgerDressings
parameters:
  - in: head
    name: burgerId
    required: false`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Operation
	var rDoc v2.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
}

func TestComparePathItem_V3_AddRequiredParam(t *testing.T) {
	left := `operationId: listBurgerDressings`

	right := `operationId: listBurgerDressings
parameters:
  - in: head
    name: burgerId
    required: true`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Operation
	var rDoc v3.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
}

func TestComparePathItem_V2_AddRequiredParam(t *testing.T) {
	left := `operationId: listBurgerDressings`

	right := `operationId: listBurgerDressings
parameters:
  - in: head
    name: burgerId
    required: true`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Operation
	var rDoc v2.Operation
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareOperations(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
}
