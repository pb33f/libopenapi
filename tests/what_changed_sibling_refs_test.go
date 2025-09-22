// Copyright 2025 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package tests

import (
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/stretchr/testify/assert"
)

func TestSiblingRefs_WhatChanged_Integration(t *testing.T) {
	t.Run("what-changed works with sibling ref transformations", func(t *testing.T) {
		originalSpec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Base:
      type: object
      properties:
        id:
          type: string
    Enhanced:
      title: "Original Title"
      description: "Original Description"
      $ref: '#/components/schemas/Base'`

		modifiedSpec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Base:
      type: object
      properties:
        id:
          type: string
        name:
          type: string
    Enhanced:
      title: "Modified Title"
      description: "Modified Description"
      example: {"id": "123"}
      $ref: '#/components/schemas/Base'`

		config := datamodel.NewDocumentConfiguration()
		config.TransformSiblingRefs = true

		// create documents
		originalDoc, err := libopenapi.NewDocumentWithConfiguration([]byte(originalSpec), config)
		assert.NoError(t, err)
		modifiedDoc, err := libopenapi.NewDocumentWithConfiguration([]byte(modifiedSpec), config)
		assert.NoError(t, err)

		// compare using what-changed
		changes, errs := libopenapi.CompareDocuments(originalDoc, modifiedDoc)
		if errs != nil {
			t.Fatalf("comparison failed: %v", errs)
		}

		assert.NotNil(t, changes)
		assert.Greater(t, changes.TotalChanges(), 0, "should detect changes in sibling-ref enhanced schemas")

		// verify specific changes are detected
		if changes.TotalChanges() > 0 {
			t.Logf("Detected %d total changes", changes.TotalChanges())
			if changes.ComponentsChanges != nil && changes.ComponentsChanges.SchemaChanges != nil {
				t.Logf("Schema changes detected: %v", len(changes.ComponentsChanges.SchemaChanges))
			}
		}
	})

	t.Run("what-changed backwards compatibility", func(t *testing.T) {
		originalSpec := `openapi: 3.0.0
info:
  title: Legacy API
  version: 1.0.0
components:
  schemas:
    Base:
      type: object
    Legacy:
      $ref: '#/components/schemas/Base'`

		modifiedSpec := `openapi: 3.0.0
info:
  title: Legacy API
  version: 2.0.0
components:
  schemas:
    Base:
      type: object
      properties:
        newProp:
          type: string
    Legacy:
      $ref: '#/components/schemas/Base'`

		// test with all features disabled for backwards compatibility
		config := datamodel.NewDocumentConfiguration()
		config.TransformSiblingRefs = false
		config.MergeReferencedProperties = false

		originalDoc, err := libopenapi.NewDocumentWithConfiguration([]byte(originalSpec), config)
		assert.NoError(t, err)
		modifiedDoc, err := libopenapi.NewDocumentWithConfiguration([]byte(modifiedSpec), config)
		assert.NoError(t, err)

		// what-changed should work exactly as before
		changes, errs := libopenapi.CompareDocuments(originalDoc, modifiedDoc)
		if errs != nil {
			t.Fatalf("comparison failed: %v", errs)
		}

		assert.NotNil(t, changes)
		assert.Greater(t, changes.TotalChanges(), 0, "should detect version and schema changes")
	})
}
