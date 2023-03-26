// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package datamodel

import (
    "github.com/stretchr/testify/assert"
    "testing"
)

func TestNewClosedDocumentConfiguration(t *testing.T) {
    cfg := NewClosedDocumentConfiguration()
    assert.False(t, cfg.AllowRemoteReferences)
    assert.False(t, cfg.AllowFileReferences)
}

func TestNewOpenDocumentConfiguration(t *testing.T) {
    cfg := NewOpenDocumentConfiguration()
    assert.True(t, cfg.AllowRemoteReferences)
    assert.True(t, cfg.AllowFileReferences)
}
