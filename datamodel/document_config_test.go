// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package datamodel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClosedDocumentConfiguration(t *testing.T) {
	t.Parallel()
	cfg := NewClosedDocumentConfiguration()
	assert.False(t, cfg.AllowRemoteReferences)
	assert.False(t, cfg.AllowFileReferences)
}

func TestNewOpenDocumentConfiguration(t *testing.T) {
	t.Parallel()
	cfg := NewOpenDocumentConfiguration()
	assert.True(t, cfg.AllowRemoteReferences)
	assert.True(t, cfg.AllowFileReferences)
}
