// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package datamodel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClosedDocumentConfiguration(t *testing.T) {
	cfg := NewDocumentConfiguration()
	assert.NotNil(t, cfg)
}

func TestNewDocumentConfiguration_DefaultSiblingRefTransformation(t *testing.T) {
	cfg := NewDocumentConfiguration()
	assert.True(t, cfg.TransformSiblingRefs, "TransformSiblingRefs should be enabled by default for OpenAPI 3.1 compliance")
}

func TestDocumentConfiguration_SiblingRefTransformationDisabled(t *testing.T) {
	cfg := NewDocumentConfiguration()
	cfg.TransformSiblingRefs = false
	assert.False(t, cfg.TransformSiblingRefs, "TransformSiblingRefs should be configurable")
}

func TestNewDocumentConfiguration_DefaultPropertyMerging(t *testing.T) {
	cfg := NewDocumentConfiguration()
	assert.True(t, cfg.MergeReferencedProperties, "MergeReferencedProperties should be enabled by default")
	assert.Equal(t, PreserveLocal, cfg.PropertyMergeStrategy, "PropertyMergeStrategy should default to PreserveLocal")
}

func TestDocumentConfiguration_PropertyMergeStrategies(t *testing.T) {
	cfg := NewDocumentConfiguration()

	t.Run("preserve local strategy", func(t *testing.T) {
		cfg.PropertyMergeStrategy = PreserveLocal
		assert.Equal(t, PreserveLocal, cfg.PropertyMergeStrategy)
	})

	t.Run("overwrite with remote strategy", func(t *testing.T) {
		cfg.PropertyMergeStrategy = OverwriteWithRemote
		assert.Equal(t, OverwriteWithRemote, cfg.PropertyMergeStrategy)
	})

	t.Run("reject conflicts strategy", func(t *testing.T) {
		cfg.PropertyMergeStrategy = RejectConflicts
		assert.Equal(t, RejectConflicts, cfg.PropertyMergeStrategy)
	})
}

func TestDocumentConfiguration_PropertyMergingDisabled(t *testing.T) {
	cfg := NewDocumentConfiguration()
	cfg.MergeReferencedProperties = false
	assert.False(t, cfg.MergeReferencedProperties, "MergeReferencedProperties should be configurable")
}

func TestDocumentConfiguration_BackwardsCompatibility(t *testing.T) {
	// verify that all new features can be disabled to preserve existing behavior
	cfg := NewDocumentConfiguration()
	cfg.TransformSiblingRefs = false
	cfg.MergeReferencedProperties = false

	assert.False(t, cfg.TransformSiblingRefs)
	assert.False(t, cfg.MergeReferencedProperties)
	// when disabled, should behave exactly like pre-enhancement versions
}
