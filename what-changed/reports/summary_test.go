// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package reports

import (
	"os"
	"testing"

	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/what-changed/model"
	"github.com/stretchr/testify/assert"
)

func createDiff() *model.DocumentChanges {
	burgerShopOriginal, _ := os.ReadFile("../../test_specs/burgershop.openapi.yaml")
	burgerShopUpdated, _ := os.ReadFile("../../test_specs/burgershop.openapi-modified.yaml")
	originalDoc, _ := libopenapi.NewDocument(burgerShopOriginal)
	updatedDoc, _ := libopenapi.NewDocument(burgerShopUpdated)
	documentChanges, _ := libopenapi.CompareDocuments(originalDoc, updatedDoc)
	return documentChanges
}

func TestCreateSummary_OverallReport(t *testing.T) {
	changes := createDiff()
	report := CreateOverallReport(changes)
	assert.Equal(t, 1, report.ChangeReport[v3.InfoLabel].Total)
	// Callbacks are now properly counted as individual expression changes
	// instead of a single PropertyAdded/PropertyRemoved change
	assert.Equal(t, 45, report.ChangeReport[v3.PathsLabel].Total)
	assert.Equal(t, 11, report.ChangeReport[v3.PathsLabel].Breaking)
	assert.Equal(t, 3, report.ChangeReport[v3.TagsLabel].Total)
	assert.Equal(t, 1, report.ChangeReport[v3.ExternalDocsLabel].Total)
	assert.Equal(t, 2, report.ChangeReport[v3.WebhooksLabel].Total)
	assert.Equal(t, 2, report.ChangeReport[v3.ServersLabel].Total)
	assert.Equal(t, 1, report.ChangeReport[v3.ServersLabel].Breaking)
	assert.Equal(t, 1, report.ChangeReport[v3.SecurityLabel].Total)
	assert.Equal(t, 20, report.ChangeReport[v3.ComponentsLabel].Total)
	assert.Equal(t, 7, report.ChangeReport[v3.ComponentsLabel].Breaking)
}

func TestCreateSummary_OverallReport_IncludesDocumentPropertyChanges(t *testing.T) {
	left := []byte("openapi: 3.0.0\ninfo:\n  title: test\n  version: 1.0.0\npaths: {}\n")
	right := []byte("openapi: 3.1.0\ninfo:\n  title: test\n  version: 2.0.0\npaths: {}\n")

	originalDoc, originalErr := libopenapi.NewDocument(left)
	assert.NoError(t, originalErr)
	updatedDoc, updatedErr := libopenapi.NewDocument(right)
	assert.NoError(t, updatedErr)

	documentChanges, compareErrs := libopenapi.CompareDocuments(originalDoc, updatedDoc)
	assert.Empty(t, compareErrs)

	report := CreateOverallReport(documentChanges)
	assert.Equal(t, 1, report.ChangeReport[v3.OpenAPILabel].Total)
	assert.Equal(t, 1, report.ChangeReport[v3.OpenAPILabel].Breaking)
	assert.Equal(t, 1, report.ChangeReport[v3.InfoLabel].Total)
	assert.Equal(t, 0, report.ChangeReport[v3.InfoLabel].Breaking)
}

func TestCreateSummary_OverallReport_CountsBreakingResponseExtensions(t *testing.T) {
	left := []byte(`openapi: 3.1.0
info:
  title: test
  version: 1.0.0
paths:
  /burgers:
    get:
      responses:
        "500":
          description: no burgers
          x-summary: legacy summary
`)
	right := []byte(`openapi: 3.1.0
info:
  title: test
  version: 1.0.0
paths:
  /burgers:
    get:
      responses:
        "500":
          description: no burgers
`)

	originalDoc, originalErr := libopenapi.NewDocument(left)
	assert.NoError(t, originalErr)
	updatedDoc, updatedErr := libopenapi.NewDocument(right)
	assert.NoError(t, updatedErr)

	documentChanges, compareErrs := libopenapi.CompareDocuments(originalDoc, updatedDoc)
	assert.Empty(t, compareErrs)

	report := CreateOverallReport(documentChanges)
	assert.Equal(t, 1, report.ChangeReport[v3.PathsLabel].Total)
	assert.Equal(t, 1, report.ChangeReport[v3.PathsLabel].Breaking)
}

func TestCreateSummary_OverallReport_NilChanges(t *testing.T) {
	report := CreateOverallReport(nil)
	assert.NotNil(t, report)
	assert.Empty(t, report.ChangeReport)
}

func TestMergeRootPropertyChangesAndHelpers(t *testing.T) {
	changedReport := make(map[string]*Changed)

	mergeRootPropertyChanges(changedReport, nil)
	assert.Empty(t, changedReport)

	propertyChanges := model.NewPropertyChanges([]*model.Change{
		nil,
		{Property: "", Breaking: true},
		{Property: v3.OpenAPILabel, Breaking: true},
		{Property: v3.OpenAPILabel, Breaking: false},
	})
	mergeRootPropertyChanges(changedReport, propertyChanges)
	assert.Equal(t, 2, changedReport[v3.OpenAPILabel].Total)
	assert.Equal(t, 1, changedReport[v3.OpenAPILabel].Breaking)

	mergeChangedModel(changedReport, v3.OpenAPILabel, &Changed{Total: 3, Breaking: 2})
	assert.Equal(t, 5, changedReport[v3.OpenAPILabel].Total)
	assert.Equal(t, 3, changedReport[v3.OpenAPILabel].Breaking)

	mergeChangedModel(changedReport, v3.InfoLabel, nil)
	assert.Nil(t, changedReport[v3.InfoLabel])

	assert.Equal(t, changedReport[v3.OpenAPILabel], getOrCreateChanged(changedReport, v3.OpenAPILabel))
}

func TestCreateChangedModelHelpers(t *testing.T) {
	changed := createChangedModel(&model.PropertyChanges{
		Changes: []*model.Change{
			{Breaking: true},
			{Breaking: false},
		},
	})
	assert.Equal(t, 2, changed.Total)
	assert.Equal(t, 1, changed.Breaking)

	fromSlice := createChangedModelFromSlice([]HasChanges{
		&model.PropertyChanges{Changes: []*model.Change{{Breaking: true}}},
		&model.PropertyChanges{Changes: []*model.Change{{Breaking: false}, {Breaking: false}}},
	})
	assert.Equal(t, 3, fromSlice.Total)
	assert.Equal(t, 1, fromSlice.Breaking)
}
