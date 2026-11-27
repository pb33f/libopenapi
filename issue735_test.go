// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package libopenapi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

// https://github.com/pb33f/libopenapi/issues/735
func TestIssue735_RelativeRefOutsideBasePath(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"api/openapi.yaml": `openapi: 3.0.3
info: {title: root, version: '1'}
paths:
  /events:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                $ref: '../other/events.yaml#/components/schemas/Event'
`,
		"other/events.yaml": `components:
  schemas:
    Event:
      type: object
      properties:
        pet:
          $ref: './openapi.yaml#/components/schemas/Pet'
`,
		"other/openapi.yaml": `openapi: 3.0.3
info: {title: other, version: '1'}
paths: {}
components:
  schemas:
    Pet:
      type: object
      properties:
        name: {type: string}
`,
	}
	for name, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	}

	specPath := filepath.Join(dir, "api", "openapi.yaml")
	spec, err := os.ReadFile(specPath)
	require.NoError(t, err)

	doc, err := NewDocumentWithConfiguration(spec, &datamodel.DocumentConfiguration{
		SpecFilePath: specPath,
		BasePath:     filepath.Join(dir, "api"),
	})
	require.NoError(t, err)

	model, err := doc.BuildV3Model()
	require.NoError(t, err)

	media := model.Model.Paths.PathItems.GetOrZero("/events").Get.Responses.Codes.
		GetOrZero("200").Content.GetOrZero("application/json")
	pet := media.Schema.Schema().Properties.GetOrZero("pet").Schema()
	require.NotNil(t, pet)
	assert.NotNil(t, pet.Properties.GetOrZero("name"))
}
