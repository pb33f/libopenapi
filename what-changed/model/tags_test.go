// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"github.com/pkg-base/libopenapi/datamodel"
	"github.com/pkg-base/libopenapi/datamodel/low"
	lowv3 "github.com/pkg-base/libopenapi/datamodel/low/v3"
	"github.com/stretchr/testify/assert"
)

func TestCompareTags(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0.1
tags:
  - name: a tag
    description: a lovely tag
    x-tag: something
    externalDocs:
      url: https://quobix.com
      description: cool`

	right := `openapi: 3.0.1
tags:
  - name: a tag
    description: a lovelier tag description
    x-tag: something else
    externalDocs:
      url: https://pb33f.io
      description: cooler`

	// create document (which will create our correct tags low level structures)
	lInfo, _ := datamodel.ExtractSpecInfo([]byte(left))
	rInfo, _ := datamodel.ExtractSpecInfo([]byte(right))
	lDoc, _ := lowv3.CreateDocumentFromConfig(lInfo, datamodel.NewDocumentConfiguration())
	rDoc, _ := lowv3.CreateDocumentFromConfig(rInfo, datamodel.NewDocumentConfiguration())

	// compare.
	changes := CompareTags(lDoc.Tags.Value, rDoc.Tags.Value)

	// evaluate.
	assert.Len(t, changes[0].Changes, 1)
	assert.Len(t, changes[0].ExternalDocs.Changes, 2)
	assert.Len(t, changes[0].ExtensionChanges.Changes, 1)
	assert.Equal(t, 4, changes[0].TotalChanges())
	assert.Len(t, changes[0].GetAllChanges(), 4)

	descChange := changes[0].Changes[0]
	assert.Equal(t, "a lovelier tag description", descChange.New)
	assert.Equal(t, "a lovely tag", descChange.Original)
	assert.Equal(t, Modified, descChange.ChangeType)
	assert.False(t, descChange.Context.HasChanged())
}

func TestCompareTags_AddNewTag(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0.1
tags:
  - name: a tag
    description: a lovelier tag description
    x-tag: something else
    externalDocs:
      url: https://pb33f.io
      description: cooler`

	right := `openapi: 3.0.1
tags:
  - name: a tag
    description: a lovelier tag description
    x-tag: something else
    externalDocs:
      url: https://pb33f.io
      description: cooler
  - name: a new tag
    description: a cool new tag`

	// create document (which will create our correct tags low level structures)
	lInfo, _ := datamodel.ExtractSpecInfo([]byte(left))
	rInfo, _ := datamodel.ExtractSpecInfo([]byte(right))
	lDoc, _ := lowv3.CreateDocumentFromConfig(lInfo, datamodel.NewDocumentConfiguration())
	rDoc, _ := lowv3.CreateDocumentFromConfig(rInfo, datamodel.NewDocumentConfiguration())

	// compare.
	changes := CompareTags(lDoc.Tags.Value, rDoc.Tags.Value)

	// evaluate.
	assert.Len(t, changes[0].Changes, 1)
	assert.Equal(t, 1, changes[0].TotalChanges())
	assert.Len(t, changes[0].GetAllChanges(), 1)

	descChange := changes[0].Changes[0]
	assert.Equal(t, ObjectAdded, descChange.ChangeType)
}

func TestCompareTags_AddDeleteTag(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0.1
tags:
  - name: a tag
    description: a lovelier tag description
    x-tag: something else
    externalDocs:
      url: https://pb33f.io
      description: cooler`

	right := `openapi: 3.0.1
tags:
  - name: a new tag
    description: a cool new tag`

	// create document (which will create our correct tags low level structures)
	lInfo, _ := datamodel.ExtractSpecInfo([]byte(left))
	rInfo, _ := datamodel.ExtractSpecInfo([]byte(right))
	lDoc, _ := lowv3.CreateDocumentFromConfig(lInfo, datamodel.NewDocumentConfiguration())
	rDoc, _ := lowv3.CreateDocumentFromConfig(rInfo, datamodel.NewDocumentConfiguration())

	// compare.
	changes := CompareTags(lDoc.Tags.Value, rDoc.Tags.Value)

	// evaluate.
	assert.Len(t, changes, 2)
	assert.Equal(t, 1, changes[0].TotalChanges())
	assert.Len(t, changes[0].GetAllChanges(), 1)
	assert.Equal(t, 1, changes[1].TotalChanges())
	assert.Equal(t, 1, changes[0].TotalBreakingChanges())
}

func TestCompareTags_DescriptionMoved(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0.1
tags:
  - description: a lovelier tag description
    name: a tag
    x-tag: something else
    externalDocs:
      url: https://pb33f.io
      description: cooler`

	right := `openapi: 3.0.1
tags:
  - name: a tag
    x-tag: something else
    description: a lovelier tag description
    externalDocs:
      url: https://pb33f.io
      description: cooler`

	// create document (which will create our correct tags low level structures)
	lInfo, _ := datamodel.ExtractSpecInfo([]byte(left))
	rInfo, _ := datamodel.ExtractSpecInfo([]byte(right))
	lDoc, _ := lowv3.CreateDocumentFromConfig(lInfo, datamodel.NewDocumentConfiguration())
	rDoc, _ := lowv3.CreateDocumentFromConfig(rInfo, datamodel.NewDocumentConfiguration())

	// compare.
	changes := CompareTags(lDoc.Tags.Value, rDoc.Tags.Value)

	// evaluate.
	assert.Nil(t, changes)
}

func TestCompareTags_NameMoved(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0.1
tags:
  - description: a lovelier tag description
    name: a tag
    x-tag: something else
    externalDocs:
      url: https://pb33f.io
      description: cooler`

	right := `openapi: 3.0.1
tags:
  - description: a lovelier tag description
    x-tag: something else
    externalDocs:
      url: https://pb33f.io
      description: cooler
    name: a tag`

	// create document (which will create our correct tags low level structures)
	lInfo, _ := datamodel.ExtractSpecInfo([]byte(left))
	rInfo, _ := datamodel.ExtractSpecInfo([]byte(right))
	lDoc, _ := lowv3.CreateDocumentFromConfig(lInfo, datamodel.NewDocumentConfiguration())
	rDoc, _ := lowv3.CreateDocumentFromConfig(rInfo, datamodel.NewDocumentConfiguration())

	// compare.
	changes := CompareTags(lDoc.Tags.Value, rDoc.Tags.Value)

	// evaluate.
	assert.Nil(t, changes)
}

func TestCompareTags_ModifiedAndMoved(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0.1
tags:
  - description: a lovelier tag description
    name: a tag
    x-tag: something else
    externalDocs:
      url: https://pb33f.io
      description: cooler`

	right := `openapi: 3.0.1
tags:
  - name: a tag
    x-tag: something else
    description: a different tag description
    externalDocs:
      url: https://pb33f.io
      description: cooler`

	// create document (which will create our correct tags low level structures)
	lInfo, _ := datamodel.ExtractSpecInfo([]byte(left))
	rInfo, _ := datamodel.ExtractSpecInfo([]byte(right))
	lDoc, _ := lowv3.CreateDocumentFromConfig(lInfo, datamodel.NewDocumentConfiguration())
	rDoc, _ := lowv3.CreateDocumentFromConfig(rInfo, datamodel.NewDocumentConfiguration())
	// compare.
	changes := CompareTags(lDoc.Tags.Value, rDoc.Tags.Value)

	// evaluate.
	assert.Len(t, changes[0].Changes, 1)
	assert.Equal(t, 1, changes[0].TotalChanges())
	assert.Len(t, changes[0].GetAllChanges(), 1)

	descChange := changes[0].Changes[0]
	assert.Equal(t, Modified, descChange.ChangeType)
	assert.Equal(t, "a lovelier tag description", descChange.Original)
	assert.Equal(t, "a different tag description", descChange.New)
	assert.True(t, descChange.Context.HasChanged())
}

func TestCompareTags_Identical(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0.1
tags:
  - description: a lovelier tag description
    name: a tag
    x-tag: something else
    externalDocs:
      url: https://pb33f.io
      description: cooler`

	right := `openapi: 3.0.1
tags:
  - description: a lovelier tag description
    name: a tag
    x-tag: something else
    externalDocs:
      url: https://pb33f.io
      description: cooler`

	// create document (which will create our correct tags low level structures)
	lInfo, _ := datamodel.ExtractSpecInfo([]byte(left))
	rInfo, _ := datamodel.ExtractSpecInfo([]byte(right))
	lDoc, _ := lowv3.CreateDocumentFromConfig(lInfo, datamodel.NewDocumentConfiguration())
	rDoc, _ := lowv3.CreateDocumentFromConfig(rInfo, datamodel.NewDocumentConfiguration())

	// compare.
	changes := CompareTags(lDoc.Tags.Value, rDoc.Tags.Value)

	// evaluate.
	assert.Nil(t, changes)
}

func TestCompareTags_AddExternalDocs(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0.1
tags:
  - name: something else`

	right := `openapi: 3.0.1
tags:
  - name: something else
    externalDocs:
      url: https://pb33f.io
      description: cooler`

	// create document (which will create our correct tags low level structures)
	lInfo, _ := datamodel.ExtractSpecInfo([]byte(left))
	rInfo, _ := datamodel.ExtractSpecInfo([]byte(right))
	lDoc, _ := lowv3.CreateDocumentFromConfig(lInfo, datamodel.NewDocumentConfiguration())
	rDoc, _ := lowv3.CreateDocumentFromConfig(rInfo, datamodel.NewDocumentConfiguration())

	// compare.
	changes := CompareTags(lDoc.Tags.Value, rDoc.Tags.Value)

	// evaluate.
	assert.Equal(t, 1, changes[0].TotalChanges())
	assert.Len(t, changes[0].GetAllChanges(), 1)
	assert.Equal(t, ObjectAdded, changes[0].Changes[0].ChangeType)
}

func TestCompareTags_RemoveExternalDocs(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0.1
tags:
  - name: something else`

	right := `openapi: 3.0.1
tags:
  - name: something else
    externalDocs:
      url: https://pb33f.io
      description: cooler`

	// create document (which will create our correct tags low level structures)
	lInfo, _ := datamodel.ExtractSpecInfo([]byte(left))
	rInfo, _ := datamodel.ExtractSpecInfo([]byte(right))
	lDoc, _ := lowv3.CreateDocumentFromConfig(lInfo, datamodel.NewDocumentConfiguration())
	rDoc, _ := lowv3.CreateDocumentFromConfig(rInfo, datamodel.NewDocumentConfiguration())

	// compare.
	changes := CompareTags(rDoc.Tags.Value, lDoc.Tags.Value)

	// evaluate.
	assert.Equal(t, 1, changes[0].TotalChanges())
	assert.Len(t, changes[0].GetAllChanges(), 1)
	assert.Equal(t, ObjectRemoved, changes[0].Changes[0].ChangeType)
}

func TestCompareTags_OpenAPI32_NewFields(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0.1
tags:
  - name: partner
    description: Operations available to the partners network
    externalDocs:
      url: https://pb33f.io
      description: Find more info here`

	right := `openapi: 3.0.1
tags:
  - name: partner
    summary: Partner API
    description: Operations available to the partners network
    parent: external
    kind: audience
    externalDocs:
      url: https://pb33f.io
      description: Find more info here`

	// create document (which will create our correct tags low level structures)
	lInfo, _ := datamodel.ExtractSpecInfo([]byte(left))
	rInfo, _ := datamodel.ExtractSpecInfo([]byte(right))
	lDoc, _ := lowv3.CreateDocumentFromConfig(lInfo, datamodel.NewDocumentConfiguration())
	rDoc, _ := lowv3.CreateDocumentFromConfig(rInfo, datamodel.NewDocumentConfiguration())

	// compare.
	changes := CompareTags(lDoc.Tags.Value, rDoc.Tags.Value)

	// evaluate.
	assert.Len(t, changes[0].Changes, 3) // summary, parent, kind added
	assert.Equal(t, 3, changes[0].TotalChanges())
	assert.Len(t, changes[0].GetAllChanges(), 3)

	// Check the changes
	changeMap := make(map[string]*Change)
	for _, change := range changes[0].Changes {
		changeMap[change.Property] = change
	}

	// Summary was added
	summaryChange := changeMap["summary"]
	assert.NotNil(t, summaryChange)
	assert.Equal(t, PropertyAdded, summaryChange.ChangeType)
	assert.Equal(t, "Partner API", summaryChange.New)
	assert.False(t, summaryChange.Breaking)

	// Parent was added
	parentChange := changeMap["parent"]
	assert.NotNil(t, parentChange)
	assert.Equal(t, PropertyAdded, parentChange.ChangeType)
	assert.Equal(t, "external", parentChange.New)
	assert.True(t, parentChange.Breaking)

	// Kind was added
	kindChange := changeMap["kind"]
	assert.NotNil(t, kindChange)
	assert.Equal(t, PropertyAdded, kindChange.ChangeType)
	assert.Equal(t, "audience", kindChange.New)
	assert.False(t, kindChange.Breaking)
}

func TestCompareTags_OpenAPI32_ModifiedFields(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0.1
tags:
  - name: partner
    summary: Partner
    description: Operations available to the partners network
    parent: external
    kind: audience`

	right := `openapi: 3.0.1
tags:
  - name: partner
    summary: Partner API
    description: Operations available to the partners network
    parent: internal
    kind: nav`

	// create document (which will create our correct tags low level structures)
	lInfo, _ := datamodel.ExtractSpecInfo([]byte(left))
	rInfo, _ := datamodel.ExtractSpecInfo([]byte(right))
	lDoc, _ := lowv3.CreateDocumentFromConfig(lInfo, datamodel.NewDocumentConfiguration())
	rDoc, _ := lowv3.CreateDocumentFromConfig(rInfo, datamodel.NewDocumentConfiguration())

	// compare.
	changes := CompareTags(lDoc.Tags.Value, rDoc.Tags.Value)

	// evaluate.
	assert.Len(t, changes[0].Changes, 3) // summary, parent, kind modified
	assert.Equal(t, 3, changes[0].TotalChanges())
	assert.Equal(t, 1, changes[0].TotalBreakingChanges()) // only parent change is breaking
	assert.Len(t, changes[0].GetAllChanges(), 3)

	// Check the changes
	changeMap := make(map[string]*Change)
	for _, change := range changes[0].Changes {
		changeMap[change.Property] = change
	}

	// Summary was modified (non-breaking)
	summaryChange := changeMap["summary"]
	assert.NotNil(t, summaryChange)
	assert.Equal(t, Modified, summaryChange.ChangeType)
	assert.Equal(t, "Partner", summaryChange.Original)
	assert.Equal(t, "Partner API", summaryChange.New)
	assert.False(t, summaryChange.Breaking)

	// Parent was modified (breaking)
	parentChange := changeMap["parent"]
	assert.NotNil(t, parentChange)
	assert.Equal(t, Modified, parentChange.ChangeType)
	assert.Equal(t, "external", parentChange.Original)
	assert.Equal(t, "internal", parentChange.New)
	assert.True(t, parentChange.Breaking)

	// Kind was modified (non-breaking)
	kindChange := changeMap["kind"]
	assert.NotNil(t, kindChange)
	assert.Equal(t, Modified, kindChange.ChangeType)
	assert.Equal(t, "audience", kindChange.Original)
	assert.Equal(t, "nav", kindChange.New)
	assert.False(t, kindChange.Breaking)
}

func TestCompareTags_OpenAPI32_RemovedFields(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `openapi: 3.0.1
tags:
  - name: partner
    summary: Partner API
    description: Operations available to the partners network
    parent: external
    kind: audience`

	right := `openapi: 3.0.1
tags:
  - name: partner
    description: Operations available to the partners network`

	// create document (which will create our correct tags low level structures)
	lInfo, _ := datamodel.ExtractSpecInfo([]byte(left))
	rInfo, _ := datamodel.ExtractSpecInfo([]byte(right))
	lDoc, _ := lowv3.CreateDocumentFromConfig(lInfo, datamodel.NewDocumentConfiguration())
	rDoc, _ := lowv3.CreateDocumentFromConfig(rInfo, datamodel.NewDocumentConfiguration())

	// compare.
	changes := CompareTags(lDoc.Tags.Value, rDoc.Tags.Value)

	// evaluate.
	assert.Len(t, changes[0].Changes, 3) // summary, parent, kind removed
	assert.Equal(t, 3, changes[0].TotalChanges())
	assert.Equal(t, 1, changes[0].TotalBreakingChanges()) // only parent removal is breaking
	assert.Len(t, changes[0].GetAllChanges(), 3)

	// Check the changes
	changeMap := make(map[string]*Change)
	for _, change := range changes[0].Changes {
		changeMap[change.Property] = change
	}

	// Summary was removed (non-breaking)
	summaryChange := changeMap["summary"]
	assert.NotNil(t, summaryChange)
	assert.Equal(t, PropertyRemoved, summaryChange.ChangeType)
	assert.Equal(t, "Partner API", summaryChange.Original)
	assert.False(t, summaryChange.Breaking)

	// Parent was removed (breaking)
	parentChange := changeMap["parent"]
	assert.NotNil(t, parentChange)
	assert.Equal(t, PropertyRemoved, parentChange.ChangeType)
	assert.Equal(t, "external", parentChange.Original)
	assert.True(t, parentChange.Breaking)

	// Kind was removed (non-breaking)
	kindChange := changeMap["kind"]
	assert.NotNil(t, kindChange)
	assert.Equal(t, PropertyRemoved, kindChange.ChangeType)
	assert.Equal(t, "audience", kindChange.Original)
	assert.False(t, kindChange.Breaking)
}
