// Copyright 2026 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package libopenapi

import (
	"github.com/pb33f/testify/require"
	"testing"
)

func TestBuildV2Model_ExternalPathItems(t *testing.T) {
	config := externalRegressionConfig(t, map[string]string{
		"paths/list.yaml": "get:\n  responses:\n    '200': {description: ok}\n",
		"paths/item.yaml": "parameters:\n  - {$ref: ./id.yaml}\nget:\n  responses:\n    '200': {description: ok}\n",
		"paths/id.yaml":   "name: id\nin: path\nrequired: true\ntype: string\n",
	})
	doc, err := NewDocumentWithConfiguration([]byte(`swagger: "2.0"
info: {title: External paths, version: "1"}
paths:
  /items:
    $ref: ./paths/list.yaml
  /items/{id}:
    $ref: ./paths/item.yaml
`), config)
	require.NoError(t, err)
	model, err := doc.BuildV2Model()
	require.NoError(t, err)
	list := model.Model.Paths.PathItems.GetOrZero("/items")
	require.NotNil(t, list.Get)
	require.Empty(t, list.Parameters)
	require.Equal(t, "./paths/list.yaml", list.Ref)
	item := model.Model.Paths.PathItems.GetOrZero("/items/{id}")
	require.NotNil(t, item.Get)
	require.Len(t, item.Parameters, 1)
	require.Equal(t, "id", item.Parameters[0].Name)
	require.Equal(t, "./paths/item.yaml", item.Ref)
}

func TestBuildV2Model_MissingExternalPathItem(t *testing.T) {
	config := externalRegressionConfig(t, map[string]string{
		"paths.yaml": "paths: {}\n",
	})
	doc, err := NewDocumentWithConfiguration([]byte(`swagger: "2.0"
info: {title: Missing path, version: "1"}
paths:
  /items: {$ref: './paths.yaml#/paths/missing'}
`), config)
	require.NoError(t, err)
	_, err = doc.BuildV2Model()
	require.Error(t, err)
}
