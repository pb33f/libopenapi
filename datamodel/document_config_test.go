// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package datamodel

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewClosedDocumentConfiguration(t *testing.T) {
	cfg := NewDocumentConfiguration()
	assert.NotNil(t, cfg)
}
