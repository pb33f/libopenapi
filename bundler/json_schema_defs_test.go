// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io
// SPDX-License-Identifier: MIT

package bundler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestBundleBytesComposed_JSONSchemaDefsLiftedAndRewritten(t *testing.T) {
	tmp := t.TempDir()

	spec := `openapi: "3.2.0"
info:
  title: JSON Schema $defs bundle
  version: 0.0.1
paths:
  /widgets/{id}:
    get:
      responses:
        "200":
          description: A widget.
          content:
            application/json:
              schema:
                $ref: "./widget.json"
`
	widget := `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://example.com/schemas/widget.json",
  "title": "Widget",
  "type": "object",
  "additionalProperties": false,
  "required": ["id", "manufacturer"],
  "properties": {
    "id": { "type": "string", "title": "ID" },
    "manufacturer": { "$ref": "#/$defs/company~1profile", "title": "Manufacturer" },
    "distributor":  { "$ref": "#/$defs/company~1profile", "title": "Distributor" },
    "subsidiaries": {
      "type": "array",
      "items": { "$ref": "#/$defs/company~1profile" },
      "title": "Subsidiaries"
    },
    "reseller": { "$ref": "#/$defs/company_profile", "title": "Reseller" }
  },
  "$defs": {
    "company/profile": {
      "type": "object",
      "additionalProperties": false,
      "required": ["name"],
      "properties": {
        "name":    { "type": "string", "title": "Name" },
        "country": { "type": "string", "title": "Country" }
      }
    },
    "company_profile": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "name": { "type": "string", "title": "Name" }
      }
    }
  }
}
`

	require.NoError(t, os.WriteFile(filepath.Join(tmp, "spec.yaml"), []byte(spec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "widget.json"), []byte(widget), 0644))

	specBytes, err := os.ReadFile(filepath.Join(tmp, "spec.yaml"))
	require.NoError(t, err)

	bundled, err := BundleBytesComposed(specBytes, &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		SpecFilePath:        filepath.Join(tmp, "spec.yaml"),
		AllowFileReferences: true,
	}, nil)
	require.NoError(t, err)

	bundledText := string(bundled)
	assert.NotContains(t, bundledText, "#/$defs/company~1profile")
	assert.NotContains(t, bundledText, "#/$defs/company_profile")
	assert.Equal(t, 1, strings.Count(bundledText, "#/components/schemas/widget__company_profile__1"))

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(bundled, &doc))

	components, ok := doc["components"].(map[string]any)
	require.True(t, ok)
	schemas, ok := components["schemas"].(map[string]any)
	require.True(t, ok)
	require.Contains(t, schemas, "widget")
	require.Contains(t, schemas, "widget__company_profile")
	require.Contains(t, schemas, "widget__company_profile__1")
	for name := range schemas {
		assert.Regexp(t, `^[a-zA-Z0-9._-]+$`, name)
	}

	widgetSchema, ok := schemas["widget"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, widgetSchema, "$defs", "$defs should remain valid JSON Schema data on the schema model")
	properties, ok := widgetSchema["properties"].(map[string]any)
	require.True(t, ok)
	manufacturer, ok := properties["manufacturer"].(map[string]any)
	require.True(t, ok)
	reseller, ok := properties["reseller"].(map[string]any)
	require.True(t, ok)

	manufacturerRef, ok := manufacturer["$ref"].(string)
	require.True(t, ok)
	resellerRef, ok := reseller["$ref"].(string)
	require.True(t, ok)
	assert.NotEqual(t, manufacturerRef, resellerRef)
	for _, ref := range []string{manufacturerRef, resellerRef} {
		componentName := strings.TrimPrefix(ref, "#/components/schemas/")
		require.Contains(t, schemas, componentName)
		assert.Regexp(t, `^[a-zA-Z0-9._-]+$`, componentName)
	}
}
