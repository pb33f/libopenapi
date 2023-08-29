// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package reports

import (
	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/what-changed/model"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func createDiff() *model.DocumentChanges {
	burgerShopOriginal, _ := ioutil.ReadFile("../../test_specs/burgershop.openapi.yaml")
	burgerShopUpdated, _ := ioutil.ReadFile("../../test_specs/burgershop.openapi-modified.yaml")
	originalDoc, _ := libopenapi.NewDocument(burgerShopOriginal)
	updatedDoc, _ := libopenapi.NewDocument(burgerShopUpdated)
	documentChanges, _ := libopenapi.CompareDocuments(originalDoc, updatedDoc)
	return documentChanges
}

func TestCreateSummary_OverallReport(t *testing.T) {
	changes := createDiff()
	report := CreateOverallReport(changes)
	assert.Equal(t, 1, report.ChangeReport[v3.InfoLabel].Total)
	assert.Equal(t, 43, report.ChangeReport[v3.PathsLabel].Total)
	assert.Equal(t, 9, report.ChangeReport[v3.PathsLabel].Breaking)
	assert.Equal(t, 3, report.ChangeReport[v3.TagsLabel].Total)
	assert.Equal(t, 1, report.ChangeReport[v3.ExternalDocsLabel].Total)
	assert.Equal(t, 2, report.ChangeReport[v3.WebhooksLabel].Total)
	assert.Equal(t, 2, report.ChangeReport[v3.ServersLabel].Total)
	assert.Equal(t, 1, report.ChangeReport[v3.ServersLabel].Breaking)
	assert.Equal(t, 1, report.ChangeReport[v3.SecurityLabel].Total)
	assert.Equal(t, 20, report.ChangeReport[v3.ComponentsLabel].Total)
	assert.Equal(t, 8, report.ChangeReport[v3.ComponentsLabel].Breaking)
}
