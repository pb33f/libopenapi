// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package bundler

import (
	"testing"

	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestInferComponentTypeFromSourcePath(t *testing.T) {
	tests := []struct {
		name       string
		sourcePath []string
		wantType   string
		wantOK     bool
	}{
		{
			name:       "empty path",
			sourcePath: nil,
			wantOK:     false,
		},
		{
			name:       "operation response",
			sourcePath: []string{"paths", "/pets", "get", "responses", "200"},
			wantType:   v3.ResponsesLabel,
			wantOK:     true,
		},
		{
			name:       "response content schema",
			sourcePath: []string{"paths", "/pets", "get", "responses", "200", "content", "application/json", "schema"},
			wantType:   v3.SchemasLabel,
			wantOK:     true,
		},
		{
			name:       "response media type",
			sourcePath: []string{"paths", "/pets", "get", "responses", "200", "content", "application/json"},
			wantType:   v3.MediaTypesLabel,
			wantOK:     true,
		},
		{
			name:       "operation parameter",
			sourcePath: []string{"paths", "/pets", "get", "parameters", "0"},
			wantType:   v3.ParametersLabel,
			wantOK:     true,
		},
		{
			name:       "operation request body",
			sourcePath: []string{"paths", "/pets", "post", "requestBody"},
			wantType:   v3.RequestBodiesLabel,
			wantOK:     true,
		},
		{
			name:       "response header",
			sourcePath: []string{"paths", "/pets", "get", "responses", "200", "headers", "X-Rate-Limit"},
			wantType:   v3.HeadersLabel,
			wantOK:     true,
		},
		{
			name:       "media type example",
			sourcePath: []string{"paths", "/pets", "get", "responses", "200", "content", "application/json", "examples", "sample"},
			wantType:   v3.ExamplesLabel,
			wantOK:     true,
		},
		{
			name:       "singular example wrapper under schema",
			sourcePath: []string{"components", "schemas", "Pet", "example"},
			wantType:   v3.ExamplesLabel,
			wantOK:     true,
		},
		{
			name:       "schema property named example",
			sourcePath: []string{"components", "schemas", "Pet", "properties", "example"},
			wantType:   v3.SchemasLabel,
			wantOK:     true,
		},
		{
			name:       "response link",
			sourcePath: []string{"paths", "/pets", "get", "responses", "200", "links", "next"},
			wantType:   v3.LinksLabel,
			wantOK:     true,
		},
		{
			name:       "operation callback",
			sourcePath: []string{"paths", "/pets", "post", "callbacks", "created"},
			wantType:   v3.CallbacksLabel,
			wantOK:     true,
		},
		{
			name:       "callback path item",
			sourcePath: []string{"paths", "/pets", "post", "callbacks", "created", "{$request.body#/url}"},
			wantType:   v3.PathItemsLabel,
			wantOK:     true,
		},
		{
			name:       "path item",
			sourcePath: []string{"paths", "/pets"},
			wantType:   v3.PathItemsLabel,
			wantOK:     true,
		},
		{
			name:       "path item component",
			sourcePath: []string{"components", "pathItems", "Pet"},
			wantType:   v3.PathItemsLabel,
			wantOK:     true,
		},
		{
			name:       "webhook path item",
			sourcePath: []string{"webhooks", "petCreated"},
			wantType:   v3.PathItemsLabel,
			wantOK:     true,
		},
		{
			name:       "schema property",
			sourcePath: []string{"components", "schemas", "Pet", "properties", "owner"},
			wantType:   v3.SchemasLabel,
			wantOK:     true,
		},
		{
			name:       "components schema bucket",
			sourcePath: []string{"components", "schemas"},
			wantType:   v3.SchemasLabel,
			wantOK:     true,
		},
		{
			name:       "media type component",
			sourcePath: []string{"components", "mediaTypes", "json"},
			wantType:   v3.MediaTypesLabel,
			wantOK:     true,
		},
		{
			name:       "security scheme component",
			sourcePath: []string{"components", "securitySchemes", "bearerAuth"},
			wantType:   v3.SecuritySchemesLabel,
			wantOK:     true,
		},
		{
			name:       "unknown path",
			sourcePath: []string{"x-private", "thing"},
			wantOK:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotOK := inferComponentTypeFromSourcePath(tt.sourcePath)
			assert.Equal(t, tt.wantOK, gotOK)
			assert.Equal(t, tt.wantType, gotType)
		})
	}
}

func TestIsSingularExampleSourceSegment(t *testing.T) {
	sourcePath := []string{"components", "schemas", "Pet", "example"}

	assert.False(t, isSingularExampleSourceSegment(sourcePath, -1))
	assert.False(t, isSingularExampleSourceSegment(sourcePath, len(sourcePath)))
	assert.False(t, isSingularExampleSourceSegment(sourcePath, 2))
	assert.False(t, isSingularExampleSourceSegment([]string{"components", "schemas", "Pet", "properties", "example"}, 4))
	assert.True(t, isSingularExampleSourceSegment([]string{"example"}, 0))
	assert.True(t, isSingularExampleSourceSegment(sourcePath, 3))
}

func TestDecodeSingleSegmentPointer(t *testing.T) {
	assert.Equal(t, "plain", decodeSingleSegmentPointer("plain"))
	assert.Equal(t, "one/two~three", decodeSingleSegmentPointer("one~1two~0three"))
}

func TestCanComposeContextualReference(t *testing.T) {
	tests := []struct {
		name          string
		componentType string
		source        string
		bareFile      bool
		want          bool
	}{
		{
			name:          "pointer response can be sparse",
			componentType: v3.ResponsesLabel,
			source:        "description: Authentication failed",
			want:          true,
		},
		{
			name:          "bare file response can be description only",
			componentType: v3.ResponsesLabel,
			source:        "description: Authentication failed",
			bareFile:      true,
			want:          true,
		},
		{
			name:          "bare file detected schema must match requested type",
			componentType: v3.ResponsesLabel,
			source:        "type: object",
			bareFile:      true,
			want:          false,
		},
		{
			name:          "bare file schema rejects wrapper map",
			componentType: v3.SchemasLabel,
			source:        "NonRequired:\n  type: object\n",
			bareFile:      true,
			want:          false,
		},
		{
			name:          "bare file schema rejects OpenAPI document",
			componentType: v3.SchemasLabel,
			source:        "openapi: 3.1.0\ninfo:\n  title: External\n  version: 1.0.0\npaths: {}\n",
			bareFile:      true,
			want:          false,
		},
		{
			name:          "bare file schema accepts description annotation",
			componentType: v3.SchemasLabel,
			source:        "description: Sparse schema",
			bareFile:      true,
			want:          true,
		},
		{
			name:          "bare file example accepts summary only",
			componentType: v3.ExamplesLabel,
			source:        "summary: Small example",
			bareFile:      true,
			want:          true,
		},
		{
			name:          "bare file header accepts description only",
			componentType: v3.HeadersLabel,
			source:        "description: Header context",
			bareFile:      true,
			want:          true,
		},
		{
			name:          "bare file media type accepts empty map",
			componentType: v3.MediaTypesLabel,
			source:        "{}",
			bareFile:      true,
			want:          true,
		},
		{
			name:          "bare file media type accepts schema key",
			componentType: v3.MediaTypesLabel,
			source:        "schema:\n  type: string\n",
			bareFile:      true,
			want:          true,
		},
		{
			name:          "bare file empty response is not enough",
			componentType: v3.ResponsesLabel,
			source:        "{}",
			bareFile:      true,
			want:          false,
		},
		{
			name:          "unknown component type is not composed",
			componentType: "unknown",
			source:        "description: Unknown component",
			bareFile:      true,
			want:          false,
		},
		{
			name:          "bare file security scheme accepts HTTP scheme",
			componentType: v3.SecuritySchemesLabel,
			source:        "type: http\nscheme: bearer\n",
			bareFile:      true,
			want:          true,
		},
		{
			name:          "bare file security scheme rejects schema type",
			componentType: v3.SecuritySchemesLabel,
			source:        "type: string\n",
			bareFile:      true,
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var node yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(tt.source), &node))

			got := canComposeContextualReference(tt.componentType, &node, tt.bareFile)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCanComposeContextualReference_NilNode(t *testing.T) {
	assert.False(t, canComposeContextualReference(v3.ResponsesLabel, nil, true))
}

func TestIsSecuritySchemeNode(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   bool
	}{
		{name: "api key", source: "type: apiKey\nname: X-API-Key\nin: header\n", want: true},
		{name: "http", source: "type: http\nscheme: bearer\n", want: true},
		{name: "oauth flows", source: "type: oauth2\nflows: {}\n", want: true},
		{name: "oauth metadata", source: "type: oauth2\noauth2MetadataUrl: https://example.com/oauth\n", want: true},
		{name: "openid connect", source: "type: openIdConnect\nopenIdConnectUrl: https://example.com/openid\n", want: true},
		{name: "mutual TLS", source: "type: mutualTLS\n", want: true},
		{name: "incomplete api key", source: "type: apiKey\nname: X-API-Key\n", want: false},
		{name: "incomplete HTTP", source: "type: http\n", want: false},
		{name: "schema", source: "type: string\n", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var document yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(tt.source), &document))
			assert.Equal(t, tt.want, isSecuritySchemeNode(unwrapDocumentNode(&document)))
		})
	}
}
