// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package bundler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	yaml "github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

// https://github.com/pb33f/libopenapi/issues/644
func TestBundleDocumentComposed_SharedSchemaFromAllOfAndProperty(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0o755))

	files := map[string]string{
		"openapi.yaml": `openapi: 3.1.0
info:
  title: repro
  version: 1.0.0
paths:
  /a:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "./schemas/Wrapper.yaml"
  /b:
    get:
      responses:
        "400":
          description: bad
          content:
            application/json:
              schema:
                $ref: "./schemas/Extended.yaml"
`,
		"schemas/Wrapper.yaml": `type: object
properties:
  error:
    $ref: "./Base.yaml"
`,
		"schemas/Extended.yaml": `allOf:
  - $ref: "./Base.yaml"
  - type: object
    properties:
      errors:
        type: array
        items:
          type: string
`,
		"schemas/Base.yaml": `type: object
properties:
  title:
    type: string
`,
	}
	for name, contents := range files {
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, name), []byte(contents), 0o644))
	}

	for _, sequential := range []bool{true, false} {
		t.Run(map[bool]string{true: "sequential", false: "concurrent"}[sequential], func(t *testing.T) {
			spec, err := os.ReadFile(filepath.Join(tmpDir, "openapi.yaml"))
			require.NoError(t, err)

			config := datamodel.NewDocumentConfiguration()
			config.BasePath = tmpDir
			config.ExtractRefsSequentially = sequential

			doc, err := libopenapi.NewDocumentWithConfiguration(spec, config)
			require.NoError(t, err)
			model, err := doc.BuildV3Model()
			require.NoError(t, err)

			out, err := BundleDocumentComposed(&model.Model, nil)
			require.NoError(t, err)

			var bundled struct {
				Components struct {
					Schemas map[string]any `yaml:"schemas"`
				} `yaml:"components"`
			}
			require.NoError(t, yaml.Unmarshal(out, &bundled))

			schemas := bundled.Components.Schemas
			assert.Len(t, schemas, 3, string(out))
			assert.Contains(t, schemas, "Base")
			assert.Contains(t, schemas, "Wrapper")
			assert.Contains(t, schemas, "Extended")
			assert.NotContains(t, schemas, "Base__schemas")

			output := string(out)
			assert.Contains(t, output, `$ref: "#/components/schemas/Base"`)
			assert.NotContains(t, output, "Base__")
		})
	}
}

// Operation parameters are a sequence, so their refs carry the same index-free
// SourcePath shape as allOf entries and must share a key with component refs.
func TestBundleBytesComposed_SharedParameterFromOperationAndComponent(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "params"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "params", "Limit.yaml"), []byte(`name: limit
in: query
schema:
  type: integer
`), 0o644))

	root := `openapi: 3.1.0
info:
  title: repro
  version: 1.0.0
paths:
  /a:
    get:
      parameters:
        - $ref: "./params/Limit.yaml"
      responses:
        "200":
          description: ok
components:
  parameters:
    Limit:
      $ref: "./params/Limit.yaml"
`
	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir

	out, err := BundleBytesComposed([]byte(root), config, nil)
	require.NoError(t, err)

	output := string(out)
	assert.Equal(t, 1, strings.Count(output, "name: limit"), output)
	assert.NotContains(t, output, "Limit__params__params")
}
