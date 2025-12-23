// Copyright 2022-2025 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package libopenapi

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/overlay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOverlayDocument(t *testing.T) {
	overlayYAML := `overlay: 1.0.0
info:
  title: Test Overlay
  version: 1.0.0
actions:
  - target: $.info
    update:
      title: Updated Title`

	ov, err := NewOverlayDocument([]byte(overlayYAML))
	require.NoError(t, err)
	assert.NotNil(t, ov)
	assert.Equal(t, "1.0.0", ov.Overlay)
	assert.Equal(t, "Test Overlay", ov.Info.Title)
	assert.Len(t, ov.Actions, 1)
}

func TestNewOverlayDocument_InvalidYAML(t *testing.T) {
	ov, err := NewOverlayDocument([]byte(`invalid: yaml: content:`))
	assert.Error(t, err)
	assert.Nil(t, ov)
}

func TestNewOverlayDocument_EmptyDocument(t *testing.T) {
	ov, err := NewOverlayDocument([]byte(``))
	assert.ErrorIs(t, err, overlay.ErrInvalidOverlay)
	assert.Nil(t, ov)
}

func TestNewOverlayDocument_InvalidOverlay(t *testing.T) {
	// Missing required fields
	overlayYAML := `foo: bar`

	ov, err := NewOverlayDocument([]byte(overlayYAML))
	// BuildModel should succeed but Build should return an error
	// Actually, lowoverlay.Build returns nil for missing fields,
	// and validation happens in overlay.Apply
	require.NoError(t, err)
	assert.NotNil(t, ov)
}

func TestNewOverlayDocument_SequenceRoot(t *testing.T) {
	// Sequence at root - BuildModel is lenient and returns empty overlay
	overlayYAML := `- item1
- item2`

	ov, err := NewOverlayDocument([]byte(overlayYAML))
	// BuildModel is lenient and doesn't fail, but the overlay will be empty
	require.NoError(t, err)
	assert.NotNil(t, ov)
	assert.Empty(t, ov.Overlay)
}

func TestNewOverlayDocument_WithExtensions(t *testing.T) {
	overlayYAML := `overlay: 1.0.0
info:
  title: Test Overlay
  version: 1.0.0
x-custom: custom value
actions:
  - target: $.info
    update:
      title: Updated`

	ov, err := NewOverlayDocument([]byte(overlayYAML))
	require.NoError(t, err)
	assert.NotNil(t, ov)
	assert.Equal(t, "1.0.0", ov.Overlay)
	assert.NotNil(t, ov.Extensions)
}

// Tests for ApplyOverlay (Document, Overlay)

func TestApplyOverlay(t *testing.T) {
	targetYAML := `openapi: 3.0.0
info:
  title: Original Title
  version: 1.0.0
paths: {}`

	overlayYAML := `overlay: 1.0.0
info:
  title: Test Overlay
  version: 1.0.0
actions:
  - target: $.info
    update:
      title: Updated Title`

	doc, err := NewDocument([]byte(targetYAML))
	require.NoError(t, err)

	ov, err := NewOverlayDocument([]byte(overlayYAML))
	require.NoError(t, err)

	result, err := ApplyOverlay(doc, ov)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, string(result.Bytes), "Updated Title")
	assert.Len(t, result.Warnings, 0)

	// Verify OverlayDocument is populated and ready to use
	assert.NotNil(t, result.OverlayDocument)
	assert.Equal(t, "3.0.0", result.OverlayDocument.GetVersion())
}

func TestApplyOverlay_PreservesConfiguration(t *testing.T) {
	targetYAML := `openapi: 3.0.0
info:
  title: Original Title
  version: 1.0.0
paths: {}`

	overlayYAML := `overlay: 1.0.0
info:
  title: Test Overlay
  version: 1.0.0
actions:
  - target: $.info
    update:
      title: Updated Title`

	// Create document with custom configuration
	config := &datamodel.DocumentConfiguration{
		AllowFileReferences:   true,
		AllowRemoteReferences: false,
	}
	doc, err := NewDocumentWithConfiguration([]byte(targetYAML), config)
	require.NoError(t, err)

	ov, err := NewOverlayDocument([]byte(overlayYAML))
	require.NoError(t, err)

	result, err := ApplyOverlay(doc, ov)
	require.NoError(t, err)

	// Verify configuration is preserved in the resulting document
	resultConfig := result.OverlayDocument.GetConfiguration()
	assert.NotNil(t, resultConfig)
	assert.True(t, resultConfig.AllowFileReferences)
	assert.False(t, resultConfig.AllowRemoteReferences)
}

func TestApplyOverlay_WithWarnings(t *testing.T) {
	targetYAML := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0`

	overlayYAML := `overlay: 1.0.0
info:
  title: Test Overlay
  version: 1.0.0
actions:
  - target: $.nonexistent
    update:
      value: test`

	doc, err := NewDocument([]byte(targetYAML))
	require.NoError(t, err)

	ov, err := NewOverlayDocument([]byte(overlayYAML))
	require.NoError(t, err)

	result, err := ApplyOverlay(doc, ov)
	require.NoError(t, err)
	assert.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0].Message, "zero nodes")

	// OverlayDocument should still be populated
	assert.NotNil(t, result.OverlayDocument)
}

func TestApplyOverlay_NilOverlay(t *testing.T) {
	targetYAML := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0`

	doc, err := NewDocument([]byte(targetYAML))
	require.NoError(t, err)

	result, err := ApplyOverlay(doc, nil)
	assert.ErrorIs(t, err, overlay.ErrInvalidOverlay)
	assert.Nil(t, result)
}

// Tests for ApplyOverlayFromBytes (Document, overlayBytes)

func TestApplyOverlayFromBytes(t *testing.T) {
	targetYAML := `openapi: 3.0.0
info:
  title: Original Title
  version: 1.0.0
paths: {}`

	overlayYAML := `overlay: 1.0.0
info:
  title: Test Overlay
  version: 1.0.0
actions:
  - target: $.info
    update:
      title: Updated Title`

	doc, err := NewDocument([]byte(targetYAML))
	require.NoError(t, err)

	result, err := ApplyOverlayFromBytes(doc, []byte(overlayYAML))
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, string(result.Bytes), "Updated Title")

	// Verify OverlayDocument is populated
	assert.NotNil(t, result.OverlayDocument)
	assert.Equal(t, "3.0.0", result.OverlayDocument.GetVersion())
}

func TestApplyOverlayFromBytes_InvalidOverlay(t *testing.T) {
	targetYAML := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0`

	doc, err := NewDocument([]byte(targetYAML))
	require.NoError(t, err)

	result, err := ApplyOverlayFromBytes(doc, []byte(`invalid: yaml: content:`))
	assert.Error(t, err)
	assert.Nil(t, result)
}

// Tests for ApplyOverlayToSpecBytes (docBytes, Overlay)

func TestApplyOverlayToSpecBytes(t *testing.T) {
	targetYAML := `openapi: 3.0.0
info:
  title: Original Title
  version: 1.0.0
paths: {}`

	overlayYAML := `overlay: 1.0.0
info:
  title: Test Overlay
  version: 1.0.0
actions:
  - target: $.info
    update:
      title: Updated Title`

	ov, err := NewOverlayDocument([]byte(overlayYAML))
	require.NoError(t, err)

	result, err := ApplyOverlayToSpecBytes([]byte(targetYAML), ov)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, string(result.Bytes), "Updated Title")

	// Verify OverlayDocument is populated (with default config)
	assert.NotNil(t, result.OverlayDocument)
	assert.Equal(t, "3.0.0", result.OverlayDocument.GetVersion())
}

func TestApplyOverlayToSpecBytes_NilOverlay(t *testing.T) {
	result, err := ApplyOverlayToSpecBytes([]byte(`openapi: 3.0.0`), nil)
	assert.ErrorIs(t, err, overlay.ErrInvalidOverlay)
	assert.Nil(t, result)
}

func TestApplyOverlayToSpecBytes_InvalidTarget(t *testing.T) {
	targetYAML := `invalid: yaml: content:`

	overlayYAML := `overlay: 1.0.0
info:
  title: Test Overlay
  version: 1.0.0
actions:
  - target: $.info
    update:
      title: Updated`

	ov, err := NewOverlayDocument([]byte(overlayYAML))
	require.NoError(t, err)

	result, err := ApplyOverlayToSpecBytes([]byte(targetYAML), ov)
	assert.Error(t, err)
	assert.Nil(t, result)
}

// Tests for ApplyOverlayFromBytesToSpecBytes (docBytes, overlayBytes)

func TestApplyOverlayFromBytesToSpecBytes(t *testing.T) {
	targetYAML := `openapi: 3.0.0
info:
  title: Original Title
  version: 1.0.0
paths: {}`

	overlayYAML := `overlay: 1.0.0
info:
  title: Test Overlay
  version: 1.0.0
actions:
  - target: $.info
    update:
      title: Updated Title`

	result, err := ApplyOverlayFromBytesToSpecBytes([]byte(targetYAML), []byte(overlayYAML))
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, string(result.Bytes), "Updated Title")

	// Verify OverlayDocument is populated (with default config)
	assert.NotNil(t, result.OverlayDocument)
	assert.Equal(t, "3.0.0", result.OverlayDocument.GetVersion())
}

func TestApplyOverlayFromBytesToSpecBytes_InvalidOverlay(t *testing.T) {
	targetYAML := `openapi: 3.0.0`
	overlayYAML := `invalid: yaml: content:`

	result, err := ApplyOverlayFromBytesToSpecBytes([]byte(targetYAML), []byte(overlayYAML))
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestApplyOverlayFromBytesToSpecBytes_InvalidTarget(t *testing.T) {
	targetYAML := `invalid: yaml: content:`
	overlayYAML := `overlay: 1.0.0
info:
  title: Test
  version: 1.0.0
actions:
  - target: $.info
    update:
      title: Updated`

	result, err := ApplyOverlayFromBytesToSpecBytes([]byte(targetYAML), []byte(overlayYAML))
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestApplyOverlayFromBytesToSpecBytes_ComplexOverlay(t *testing.T) {
	targetYAML := `openapi: 3.0.0
info:
  title: Original
  version: 1.0.0
  description: Remove me
tags:
  - name: existing
paths: {}`

	overlayYAML := `overlay: 1.0.0
info:
  title: Complex Overlay
  version: 1.0.0
actions:
  - target: $.info
    update:
      title: Updated
  - target: $.info.description
    remove: true
  - target: $.tags
    update:
      - name: new-tag`

	result, err := ApplyOverlayFromBytesToSpecBytes([]byte(targetYAML), []byte(overlayYAML))
	require.NoError(t, err)
	assert.Contains(t, string(result.Bytes), "Updated")
	assert.NotContains(t, string(result.Bytes), "Remove me")
	assert.Contains(t, string(result.Bytes), "existing")
	assert.Contains(t, string(result.Bytes), "new-tag")

	// Verify OverlayDocument is populated
	assert.NotNil(t, result.OverlayDocument)
}

func TestApplyOverlay_CanBuildModel(t *testing.T) {
	targetYAML := `openapi: 3.0.0
info:
  title: Original Title
  version: 1.0.0
paths:
  /test:
    get:
      summary: Test endpoint`

	overlayYAML := `overlay: 1.0.0
info:
  title: Test Overlay
  version: 1.0.0
actions:
  - target: $.info
    update:
      title: Updated Title`

	doc, err := NewDocument([]byte(targetYAML))
	require.NoError(t, err)

	ov, err := NewOverlayDocument([]byte(overlayYAML))
	require.NoError(t, err)

	result, err := ApplyOverlay(doc, ov)
	require.NoError(t, err)

	// Verify we can build a model from the OverlayDocument
	model, errs := result.OverlayDocument.BuildV3Model()
	require.Empty(t, errs)
	assert.NotNil(t, model)
	assert.Equal(t, "Updated Title", model.Model.Info.Title)
}
