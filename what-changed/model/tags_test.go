// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/low"
	lowv3 "github.com/pb33f/libopenapi/datamodel/low/v3"
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

// TestCompareTags_ParentAddedConfigurable tests that the tag parent added rule
// can be configured via the breaking rules system.
func TestCompareTags_ParentAddedConfigurable(t *testing.T) {
	// Reset state
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	left := `openapi: 3.0.1
tags:
  - name: tag1
    description: a taggy tag`

	right := `openapi: 3.0.1
tags:
  - name: tag1
    description: a taggy tag
    parent: tag2`

	// create document (which will create our correct tags low level structures)
	lInfo, _ := datamodel.ExtractSpecInfo([]byte(left))
	rInfo, _ := datamodel.ExtractSpecInfo([]byte(right))
	lDoc, _ := lowv3.CreateDocumentFromConfig(lInfo, datamodel.NewDocumentConfiguration())
	rDoc, _ := lowv3.CreateDocumentFromConfig(rInfo, datamodel.NewDocumentConfiguration())

	// Test 1: With default rules, adding parent should be breaking
	changes := CompareTags(lDoc.Tags.Value, rDoc.Tags.Value)

	assert.Len(t, changes, 1)
	assert.Len(t, changes[0].Changes, 1)

	parentChange := changes[0].Changes[0]
	assert.Equal(t, "parent", parentChange.Property)
	assert.Equal(t, PropertyAdded, parentChange.ChangeType)
	assert.True(t, parentChange.Breaking, "Default: adding parent should be breaking")
	assert.Equal(t, 1, changes[0].TotalBreakingChanges())

	// Test 2: With custom config setting parent.added to false
	low.ClearHashCache()

	falseVal := false
	customConfig := &BreakingRulesConfig{
		Tag: &TagRules{
			Parent: &BreakingChangeRule{
				Added: &falseVal,
			},
		},
	}
	SetActiveBreakingRulesConfig(customConfig)

	changes2 := CompareTags(lDoc.Tags.Value, rDoc.Tags.Value)

	assert.Len(t, changes2, 1)
	assert.Len(t, changes2[0].Changes, 1)

	parentChange2 := changes2[0].Changes[0]
	assert.Equal(t, "parent", parentChange2.Property)
	assert.Equal(t, PropertyAdded, parentChange2.ChangeType)
	assert.False(t, parentChange2.Breaking, "Custom config: adding parent should NOT be breaking")
	assert.Equal(t, 0, changes2[0].TotalBreakingChanges())
}

// TestCompareTags_AllChangeTypesConfigurable tests that all three change types
// (added, modified, removed) for tag properties can be independently configured.
func TestCompareTags_AllChangeTypesConfigurable(t *testing.T) {
	// Reset state
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	// Test data: tag with parent added
	leftAdd := `openapi: 3.0.1
tags:
  - name: tag1
    description: a taggy tag`
	rightAdd := `openapi: 3.0.1
tags:
  - name: tag1
    description: a taggy tag
    parent: tag2`

	// Test data: tag with parent modified
	leftMod := `openapi: 3.0.1
tags:
  - name: tag1
    description: a taggy tag
    parent: tag2`
	rightMod := `openapi: 3.0.1
tags:
  - name: tag1
    description: a taggy tag
    parent: tag3`

	// Test data: tag with parent removed
	leftRem := `openapi: 3.0.1
tags:
  - name: tag1
    description: a taggy tag
    parent: tag2`
	rightRem := `openapi: 3.0.1
tags:
  - name: tag1
    description: a taggy tag`

	// Set up custom config: added=false, modified=false, removed=false (all non-breaking)
	falseVal := false
	customConfig := &BreakingRulesConfig{
		Tag: &TagRules{
			Parent: &BreakingChangeRule{
				Added:    &falseVal,
				Modified: &falseVal,
				Removed:  &falseVal,
			},
		},
	}
	SetActiveBreakingRulesConfig(customConfig)

	// Test added
	lInfoAdd, _ := datamodel.ExtractSpecInfo([]byte(leftAdd))
	rInfoAdd, _ := datamodel.ExtractSpecInfo([]byte(rightAdd))
	lDocAdd, _ := lowv3.CreateDocumentFromConfig(lInfoAdd, datamodel.NewDocumentConfiguration())
	rDocAdd, _ := lowv3.CreateDocumentFromConfig(rInfoAdd, datamodel.NewDocumentConfiguration())

	changesAdd := CompareTags(lDocAdd.Tags.Value, rDocAdd.Tags.Value)
	assert.Len(t, changesAdd[0].Changes, 1)
	assert.False(t, changesAdd[0].Changes[0].Breaking, "Custom config: adding parent should NOT be breaking")

	// Test modified
	low.ClearHashCache()
	lInfoMod, _ := datamodel.ExtractSpecInfo([]byte(leftMod))
	rInfoMod, _ := datamodel.ExtractSpecInfo([]byte(rightMod))
	lDocMod, _ := lowv3.CreateDocumentFromConfig(lInfoMod, datamodel.NewDocumentConfiguration())
	rDocMod, _ := lowv3.CreateDocumentFromConfig(rInfoMod, datamodel.NewDocumentConfiguration())

	changesMod := CompareTags(lDocMod.Tags.Value, rDocMod.Tags.Value)
	assert.Len(t, changesMod[0].Changes, 1)
	assert.False(t, changesMod[0].Changes[0].Breaking, "Custom config: modifying parent should NOT be breaking")

	// Test removed
	low.ClearHashCache()
	lInfoRem, _ := datamodel.ExtractSpecInfo([]byte(leftRem))
	rInfoRem, _ := datamodel.ExtractSpecInfo([]byte(rightRem))
	lDocRem, _ := lowv3.CreateDocumentFromConfig(lInfoRem, datamodel.NewDocumentConfiguration())
	rDocRem, _ := lowv3.CreateDocumentFromConfig(rInfoRem, datamodel.NewDocumentConfiguration())

	changesRem := CompareTags(lDocRem.Tags.Value, rDocRem.Tags.Value)
	assert.Len(t, changesRem[0].Changes, 1)
	assert.False(t, changesRem[0].Changes[0].Breaking, "Custom config: removing parent should NOT be breaking")
}
