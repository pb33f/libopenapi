// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package renderer

import (
	"encoding/json"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
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

func TestMockGeneratorPreservesNullStringExamples(t *testing.T) {
	const fixture = `type: object
required: [expiresAt, revokedAt, lastUsed, literal, values]
properties:
  expiresAt:
    type: [string, 'null']
    format: date-time
    examples: [null]
  revokedAt:
    type: [string, 'null']
    format: date-time
    examples: [null]
  lastUsed:
    type: [string, 'null']
    format: date-time
    examples: [null]
  literal:
    type: [string, 'null']
    examples: ['<nil>']
  values:
    type: array
    items:
      type: [string, 'null']
      examples: [null, '<nil>', actual-string]
`
	for _, format := range []MockType{JSON, YAML} {
		t.Run(map[MockType]string{JSON: "JSON", YAML: "YAML"}[format], func(t *testing.T) {
			mockable := createFakeMockWithoutProxy(fixture, nil, nil)
			payload, err := NewMockGenerator(format).GenerateMock(mockable, "")
			require.NoError(t, err)
			var decoded map[string]any
			if format == JSON {
				require.NoError(t, json.Unmarshal(payload, &decoded))
			} else {
				require.NoError(t, yaml.Unmarshal(payload, &decoded))
			}
			for _, name := range []string{"expiresAt", "revokedAt", "lastUsed"} {
				require.Contains(t, decoded, name)
				assert.Nil(t, decoded[name], name)
			}
			assert.Equal(t, "<nil>", decoded["literal"])
			assert.Equal(t, []any{nil, "<nil>", "actual-string"}, decoded["values"])
		})
	}
}
