// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package renderer

import (
	"testing"

	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/testify/assert"
)

func TestMockValueRendererHandlesNilSchemas(t *testing.T) {
	t.Parallel()

	wr := CreateRendererUsingDefaultDictionary()

	assert.Nil(t, wr.renderMockStringValue(nil, rootType, DefaultMaxGeneratedStringBytes))
	assert.Nil(t, wr.renderMockNumberValue(nil))
}

func TestMockValueRendererCapsIPv4Strings(t *testing.T) {
	t.Parallel()

	wr := emptyDictionarySchemaRenderer(1)
	wr.SetMockGenerationOptions(MockGenerationOptions{MaxGeneratedStringBytes: 4})

	value := wr.renderMockStringValue(&highbase.Schema{
		Type:   []string{stringType},
		Format: ipv4Type,
	}, rootType, wr.effectiveMockGenerationOptions().MaxGeneratedStringBytes)

	rendered, ok := value.(string)
	assert.True(t, ok)
	assert.LessOrEqual(t, len(rendered), 4)
}
