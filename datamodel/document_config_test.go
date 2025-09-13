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
