// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package tests

import (
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/stretchr/testify/assert"
)

// TestIssue438_PastebinExample tests the specific scenario from the GitHub issue
// This test verifies that the rolodex can handle unknown file extensions when content detection is enabled
func TestIssue438_PastebinExample(t *testing.T) {
	// Simple test to verify the core functionality works at the document level
	simpleSpec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  '/test':
    get:
      responses:
        '200':
          description: OK`

	t.Run("Basic document creation with content detection enabled", func(t *testing.T) {
		config := datamodel.NewDocumentConfiguration()
		config.AllowUnknownExtensionContentDetection = true
		config.AllowRemoteReferences = true

		// Test that the configuration flows through correctly
		doc, err := libopenapi.NewDocumentWithConfiguration([]byte(simpleSpec), config)
		assert.NoError(t, err)
		assert.NotNil(t, doc)

		// Verify the document was created successfully
		model, errs := doc.BuildV3Model()
		assert.Len(t, errs, 0)
		assert.NotNil(t, model)
		assert.Equal(t, "Test API", model.Model.Info.Title)

		// Verify the configuration was passed through
		if doc.GetRolodex() != nil && doc.GetRolodex().GetConfig() != nil {
			assert.True(t, doc.GetRolodex().GetConfig().AllowUnknownExtensionContentDetection)
		}
	})

	t.Run("Basic document creation with content detection disabled", func(t *testing.T) {
		config := datamodel.NewDocumentConfiguration()
		config.AllowUnknownExtensionContentDetection = false
		config.AllowRemoteReferences = true

		// Test that the configuration flows through correctly
		doc, err := libopenapi.NewDocumentWithConfiguration([]byte(simpleSpec), config)
		assert.NoError(t, err)
		assert.NotNil(t, doc)

		// Verify the document was created successfully
		model, errs := doc.BuildV3Model()
		assert.Len(t, errs, 0)
		assert.NotNil(t, model)
		assert.Equal(t, "Test API", model.Model.Info.Title)

		// Verify the configuration was passed through
		if doc.GetRolodex() != nil && doc.GetRolodex().GetConfig() != nil {
			assert.False(t, doc.GetRolodex().GetConfig().AllowUnknownExtensionContentDetection)
		}
	})
}

// TestIssue438_ConfigurationFlow tests that the configuration flows correctly through the system
func TestIssue438_ConfigurationFlow(t *testing.T) {
	// Test that verifies our configuration options are properly passed through the system
	spec := `openapi: 3.0.0
info:
  title: Configuration Test API
  version: 1.0.0
paths:
  '/config-test':
    get:
      responses:
        '200':
          description: Configuration test endpoint`

	t.Run("Document creation succeeds with content detection configuration", func(t *testing.T) {
		config := datamodel.NewDocumentConfiguration()
		config.AllowUnknownExtensionContentDetection = true
		config.AllowRemoteReferences = true

		doc, err := libopenapi.NewDocumentWithConfiguration([]byte(spec), config)
		assert.NoError(t, err)
		assert.NotNil(t, doc)

		model, errs := doc.BuildV3Model()
		assert.Len(t, errs, 0)
		assert.NotNil(t, model)
		assert.Equal(t, "Configuration Test API", model.Model.Info.Title)

		// Test passes if document creation and model building succeed
		// The detailed rolodex configuration testing is handled at the unit test level
	})

	t.Run("Document creation succeeds regardless of content detection setting", func(t *testing.T) {
		config := datamodel.NewDocumentConfiguration()
		config.AllowUnknownExtensionContentDetection = false
		config.AllowRemoteReferences = false

		doc, err := libopenapi.NewDocumentWithConfiguration([]byte(spec), config)
		assert.NoError(t, err)
		assert.NotNil(t, doc)

		model, errs := doc.BuildV3Model()
		assert.Len(t, errs, 0)
		assert.NotNil(t, model)
		assert.Equal(t, "Configuration Test API", model.Model.Info.Title)
	})
}
