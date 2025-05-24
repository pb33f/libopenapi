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
