// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package bundler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func bundleComposedFiles(t *testing.T, files map[string]string, config *datamodel.DocumentConfiguration) (string, map[string]any) {
	t.Helper()
	tmp := t.TempDir()
	for name, content := range files {
		path := filepath.Join(tmp, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	if config == nil {
		config = &datamodel.DocumentConfiguration{}
	}
	config.BasePath = tmp
	config.AllowFileReferences = true

	bundled, err := BundleBytesComposed([]byte(files["main.yaml"]), config, nil)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal(bundled, &parsed))
	return string(bundled), parsed
}

func bundleInlineFiles(t *testing.T, files map[string]string, config *datamodel.DocumentConfiguration) map[string]any {
	t.Helper()
	tmp := t.TempDir()
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(content), 0o644))
	}
	config.BasePath = tmp
	config.AllowFileReferences = true

	bundled, err := BundleBytes([]byte(files["main.yaml"]), config)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal(bundled, &parsed))
	return parsed
}

func composedSchemas(t *testing.T, parsed map[string]any) map[string]any {
	t.Helper()
	components, ok := parsed["components"].(map[string]any)
	require.True(t, ok, "bundle has no components")
	schemas, ok := components["schemas"].(map[string]any)
	require.True(t, ok, "bundle has no component schemas")
	return schemas
}

const instanceValuePetSchema = `type: object
properties:
  name:
    type: string
`

func TestBundleBytesComposed_FragmentNamedDefault(t *testing.T) {
	bundled, parsed := bundleComposedFiles(t, map[string]string{
		"main.yaml": `openapi: 3.1.0
info:
  title: Catalog
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: 'catalog.yaml#/default'
`,
		"catalog.yaml": `default:
  type: object
  properties:
    pet:
      $ref: 'pet.yaml'
`,
		"pet.yaml": instanceValuePetSchema,
	}, nil)

	assert.NotContains(t, bundled, "pet.yaml", "a schema named default is not instance data")
	schemas := composedSchemas(t, parsed)
	assert.Contains(t, schemas, "default")
	assert.Contains(t, schemas, "pet")
}

func TestBundleBytesComposed_CatalogEntryNamedLikeSchemaKeyword(t *testing.T) {
	t.Run("parameter catalog with an entry named type", func(t *testing.T) {
		bundled, parsed := bundleComposedFiles(t, map[string]string{
			"main.yaml": `openapi: 3.1.0
info:
  title: Parameter catalog
  version: 1.0.0
paths:
  /pets:
    get:
      parameters:
        - $ref: 'params.yaml#/default'
        - $ref: 'params.yaml#/type'
      responses:
        "200":
          description: OK
`,
			"params.yaml": `default:
  name: default
  in: query
  schema:
    $ref: 'pet.yaml'
type:
  name: type
  in: query
  schema:
    type: string
`,
			"pet.yaml": instanceValuePetSchema,
		}, nil)

		assert.NotContains(t, bundled, "pet.yaml", "a parameter catalog is not a schema")
		assert.Contains(t, composedSchemas(t, parsed), "pet")
		parameters := parsed["components"].(map[string]any)["parameters"].(map[string]any)
		assert.Equal(t, "#/components/schemas/pet", parameters["default"].(map[string]any)["schema"].(map[string]any)["$ref"])
	})

	for name, catalog := range map[string]map[string]string{
		"schema catalog with an entry named format": {"catalog.yaml": `default:
  type: object
  properties:
    pet:
      $ref: 'pet.yaml'
format:
  type: string
`},
		"json schema catalog with an entry named required": {"catalog.json": `{"default": {"type": "object", "properties": {"pet": {"$ref": "pet.yaml"}}},
 "required": {"type": "string"}}`},
	} {
		t.Run(name, func(t *testing.T) {
			files := map[string]string{"pet.yaml": instanceValuePetSchema}
			for file, content := range catalog {
				files[file] = content
				files["main.yaml"] = `openapi: 3.1.0
info:
  title: Schema catalog
  version: 1.0.0
paths: {}
components:
  schemas:
    Owner:
      $ref: '` + file + `#/default'
`
			}
			bundled, parsed := bundleComposedFiles(t, files, nil)

			assert.NotContains(t, bundled, "pet.yaml", "a schema catalog is not a schema")
			schemas := composedSchemas(t, parsed)
			require.Contains(t, schemas, "pet")
			require.Contains(t, schemas, "default")
			assert.Equal(t, "#/components/schemas/pet", schemas["default"].(map[string]any)["properties"].(map[string]any)["pet"].(map[string]any)["$ref"])
		})
	}
}

func TestBundleBytesComposed_AdditionalOperationNamedDefault(t *testing.T) {
	bundled, parsed := bundleComposedFiles(t, map[string]string{
		"main.yaml": `openapi: 3.2.0
info:
  title: Additional operations
  version: 1.0.0
paths:
  /pets:
    additionalOperations:
      default:
        responses:
          "200":
            description: OK
            content:
              application/json:
                schema:
                  $ref: 'pet.yaml'
`,
		"pet.yaml": instanceValuePetSchema,
	}, nil)

	assert.NotContains(t, bundled, "pet.yaml", "an operation named default is not instance data")
	assert.Contains(t, composedSchemas(t, parsed), "pet")
}

func TestBundleBytesComposed_SchemaInstanceValuesNotRewritten(t *testing.T) {
	bundled, parsed := bundleComposedFiles(t, map[string]string{
		"main.yaml": `openapi: 3.1.0
info:
  title: Instance values
  version: 1.0.0
paths: {}
components:
  schemas:
    Owner:
      type: object
      properties:
        pet:
          $ref: 'pet.yaml'
      default:
        $ref: 'pet.yaml'
      examples:
        - $ref: 'pet.yaml'
      enum:
        - $ref: 'pet.yaml'
      const:
        $ref: 'pet.yaml'
`,
		"pet.yaml": instanceValuePetSchema,
	}, nil)

	schemas := composedSchemas(t, parsed)
	require.Contains(t, schemas, "pet")
	owner := schemas["Owner"].(map[string]any)
	assert.Equal(t, "#/components/schemas/pet", owner["properties"].(map[string]any)["pet"].(map[string]any)["$ref"])

	literal := map[string]any{"$ref": "pet.yaml"}
	assert.Equal(t, literal, owner["default"], "default is data")
	assert.Equal(t, []any{literal}, owner["examples"], "examples is data")
	assert.Equal(t, []any{literal}, owner["enum"], "enum is data")
	assert.Equal(t, literal, owner["const"], "const is data")
	assert.Contains(t, bundled, "#/components/schemas/pet")
}

func TestBundleBytesComposed_PropertyNamedLikeExtension(t *testing.T) {
	for _, exclude := range []bool{false, true} {
		t.Run(map[bool]string{false: "include", true: "exclude"}[exclude], func(t *testing.T) {
			bundled, parsed := bundleComposedFiles(t, map[string]string{
				"main.yaml": `openapi: 3.1.0
info:
  title: Extension-like property
  version: 1.0.0
paths: {}
components:
  schemas:
    Owner:
      type: object
      properties:
        x-name:
          $ref: 'pet.yaml'
`,
				"pet.yaml": instanceValuePetSchema,
			}, &datamodel.DocumentConfiguration{ExcludeExtensionRefs: exclude})

			assert.NotContains(t, bundled, "pet.yaml", "a property named x-name is a schema")
			schemas := composedSchemas(t, parsed)
			require.Contains(t, schemas, "pet")
			owner := schemas["Owner"].(map[string]any)
			assert.Equal(t, "#/components/schemas/pet", owner["properties"].(map[string]any)["x-name"].(map[string]any)["$ref"])
		})
	}
}

func TestBundleBytesComposed_ExampleObjectNamedBrackets(t *testing.T) {
	bundled, parsed := bundleComposedFiles(t, map[string]string{
		"main.yaml": `openapi: 3.1.0
info:
  title: Example named brackets
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                type: object
              examples:
                "[]":
                  $ref: 'sample.yaml'
`,
		"sample.yaml": `summary: A sample
value:
  name: Fido
`,
	}, nil)

	assert.NotContains(t, bundled, "sample.yaml", "an Example Object named [] is a reference")
	components := parsed["components"].(map[string]any)
	assert.Contains(t, components["examples"].(map[string]any), "sample")
}

func TestBundleBytesComposed_ExamplePayloadsNotRewritten(t *testing.T) {
	_, parsed := bundleComposedFiles(t, map[string]string{
		"main.yaml": `openapi: 3.1.0
info:
  title: Example payloads
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: 'pet.yaml'
              example:
                owner:
                  $ref: 'pet.yaml'
            application/vnd.pets+json:
              schema:
                $ref: 'pet.yaml'
              examples:
                sample:
                  value:
                    $ref: 'pet.yaml'
`,
		"pet.yaml": instanceValuePetSchema,
	}, nil)

	require.Contains(t, composedSchemas(t, parsed), "pet")
	content := parsed["paths"].(map[string]any)["/pets"].(map[string]any)["get"].(map[string]any)["responses"].(map[string]any)["200"].(map[string]any)["content"].(map[string]any)
	media := content["application/json"].(map[string]any)
	assert.Equal(t, "#/components/schemas/pet", media["schema"].(map[string]any)["$ref"])

	literal := map[string]any{"$ref": "pet.yaml"}
	assert.Equal(t, map[string]any{"owner": literal}, media["example"], "example payload is data")

	// example and examples are mutually exclusive on one Media Type Object, so examples lives on its own.
	vendor := content["application/vnd.pets+json"].(map[string]any)
	assert.Equal(t, "#/components/schemas/pet", vendor["schema"].(map[string]any)["$ref"])
	assert.Equal(t, literal, vendor["examples"].(map[string]any)["sample"].(map[string]any)["value"], "example value is data")
}

func TestBundleBytesComposed_PathItemFragmentParameterExamplesNotRewritten(t *testing.T) {
	_, parsed := bundleComposedFiles(t, map[string]string{
		"main.yaml": `openapi: 3.1.0
info:
  title: Path item fragment
  version: 1.0.0
paths:
  /pets:
    $ref: 'pets.yaml'
`,
		"pets.yaml": `parameters:
  - name: owner
    in: query
    schema:
      $ref: 'pet.yaml'
    example:
      owner:
        $ref: 'pet.yaml'
  - name: filter
    in: query
    schema:
      type: object
    examples:
      sample:
        value:
          $ref: 'pet.yaml'
get:
  responses:
    "200":
      description: OK
`,
		"pet.yaml": instanceValuePetSchema,
	}, nil)

	require.Contains(t, composedSchemas(t, parsed), "pet")
	pathItems := parsed["components"].(map[string]any)["pathItems"].(map[string]any)
	parameters := pathItems["pets"].(map[string]any)["parameters"].([]any)
	require.Len(t, parameters, 2)
	owner := parameters[0].(map[string]any)
	filter := parameters[1].(map[string]any)
	assert.Equal(t, "#/components/schemas/pet", owner["schema"].(map[string]any)["$ref"])

	literal := map[string]any{"$ref": "pet.yaml"}
	assert.Equal(t, map[string]any{"owner": literal}, owner["example"], "example payload is data")
	assert.Equal(t, literal, filter["examples"].(map[string]any)["sample"].(map[string]any)["value"], "example value is data")
}

// A property named default inside an extension's schema, and below a property named like an
// extension, is a property name, not a schema default.
const extensionSchemaPropertyNamedDefault = `openapi: 3.1.0
info:
  title: Extension schema
  version: 1.0.0
paths: {}
components:
  schemas:
    Owner:
      type: object
      x-model:
        type: object
        properties:
          items:
            type: array
          default:
            $ref: 'pet.yaml'
`

const extensionLikePropertyWithPropertyNamedDefault = `openapi: 3.1.0
info:
  title: Extension-like property
  version: 1.0.0
paths: {}
components:
  schemas:
    Owner:
      type: object
      properties:
        x-meta:
          type: object
          properties:
            default:
              $ref: 'pet.yaml'
`

func TestBundleBytes_PropertyNamedDefaultUnderExtensionNames(t *testing.T) {
	inlined := map[string]any{
		"type":       "object",
		"properties": map[string]any{"name": map[string]any{"type": "string"}},
	}
	extensionDefault := func(parsed map[string]any) any {
		owner := parsed["components"].(map[string]any)["schemas"].(map[string]any)["Owner"].(map[string]any)
		return owner["x-model"].(map[string]any)["properties"].(map[string]any)["default"]
	}

	t.Run("extension schema is inlined when extension refs are included", func(t *testing.T) {
		parsed := bundleInlineFiles(t, map[string]string{
			"main.yaml": extensionSchemaPropertyNamedDefault,
			"pet.yaml":  instanceValuePetSchema,
		}, &datamodel.DocumentConfiguration{})
		assert.Equal(t, inlined, extensionDefault(parsed))
	})

	t.Run("extension schema is left alone when extension refs are excluded", func(t *testing.T) {
		parsed := bundleInlineFiles(t, map[string]string{
			"main.yaml": extensionSchemaPropertyNamedDefault,
			"pet.yaml":  instanceValuePetSchema,
		}, &datamodel.DocumentConfiguration{ExcludeExtensionRefs: true})
		assert.Equal(t, map[string]any{"$ref": "pet.yaml"}, extensionDefault(parsed))
	})

	for _, exclude := range []bool{false, true} {
		t.Run(map[bool]string{false: "property named like an extension, include", true: "property named like an extension, exclude"}[exclude], func(t *testing.T) {
			parsed := bundleInlineFiles(t, map[string]string{
				"main.yaml": extensionLikePropertyWithPropertyNamedDefault,
				"pet.yaml":  instanceValuePetSchema,
			}, &datamodel.DocumentConfiguration{ExcludeExtensionRefs: exclude})
			owner := parsed["components"].(map[string]any)["schemas"].(map[string]any)["Owner"].(map[string]any)
			meta := owner["properties"].(map[string]any)["x-meta"].(map[string]any)
			assert.Equal(t, inlined, meta["properties"].(map[string]any)["default"])
		})
	}
}

func TestBundleBytesComposed_PropertyNamedDefaultUnderExtensionNames(t *testing.T) {
	for _, exclude := range []bool{false, true} {
		suffix := map[bool]string{false: ", include", true: ", exclude"}[exclude]

		t.Run("extension schema keeps its extension ref"+suffix, func(t *testing.T) {
			_, parsed := bundleComposedFiles(t, map[string]string{
				"main.yaml": extensionSchemaPropertyNamedDefault,
				"pet.yaml":  instanceValuePetSchema,
			}, &datamodel.DocumentConfiguration{ExcludeExtensionRefs: exclude})
			owner := parsed["components"].(map[string]any)["schemas"].(map[string]any)["Owner"].(map[string]any)
			model := owner["x-model"].(map[string]any)
			assert.Equal(t, map[string]any{"$ref": "pet.yaml"}, model["properties"].(map[string]any)["default"])
		})

		t.Run("property named like an extension is lifted"+suffix, func(t *testing.T) {
			bundled, parsed := bundleComposedFiles(t, map[string]string{
				"main.yaml": extensionLikePropertyWithPropertyNamedDefault,
				"pet.yaml":  instanceValuePetSchema,
			}, &datamodel.DocumentConfiguration{ExcludeExtensionRefs: exclude})
			assert.NotContains(t, bundled, "pet.yaml")
			schemas := composedSchemas(t, parsed)
			require.Contains(t, schemas, "pet")
			meta := schemas["Owner"].(map[string]any)["properties"].(map[string]any)["x-meta"].(map[string]any)
			assert.Equal(t, "#/components/schemas/pet", meta["properties"].(map[string]any)["default"].(map[string]any)["$ref"])
		})
	}
}

// An OpenAPI 3.2 itemSchema holds a Schema Object even when no keyword says so.
const itemSchemaWithDefault = `openapi: 3.2.0
info:
  title: Item schema default
  version: 1.0.0
paths:
  /stream:
    get:
      responses:
        "200":
          description: ok
          content:
            application/jsonl:
              itemSchema:
                description: no keyword says this is a schema
                default:
                  $ref: 'pet.yaml'
`

func itemSchemaOf(t *testing.T, parsed map[string]any) map[string]any {
	t.Helper()
	content := parsed["paths"].(map[string]any)["/stream"].(map[string]any)["get"].(map[string]any)["responses"].(map[string]any)["200"].(map[string]any)["content"].(map[string]any)
	return content["application/jsonl"].(map[string]any)["itemSchema"].(map[string]any)
}

func TestBundleBytes_ItemSchemaDefaultNotInlined(t *testing.T) {
	parsed := bundleInlineFiles(t, map[string]string{
		"main.yaml": itemSchemaWithDefault,
		"pet.yaml":  instanceValuePetSchema,
	}, &datamodel.DocumentConfiguration{})

	assert.Equal(t, map[string]any{"$ref": "pet.yaml"}, itemSchemaOf(t, parsed)["default"], "default is data")

	// Data that looks like a reference to a missing file is not a lookup failure.
	parsed = bundleInlineFiles(t, map[string]string{
		"main.yaml": strings.Replace(itemSchemaWithDefault, "pet.yaml", "missing.yaml", 1),
	}, &datamodel.DocumentConfiguration{})
	assert.Equal(t, map[string]any{"$ref": "missing.yaml"}, itemSchemaOf(t, parsed)["default"], "default is data")
}

func TestBundleBytesComposed_ItemSchemaDefaultNotRewritten(t *testing.T) {
	bundled, parsed := bundleComposedFiles(t, map[string]string{
		"main.yaml": itemSchemaWithDefault,
		"pet.yaml":  instanceValuePetSchema,
	}, nil)

	assert.Equal(t, map[string]any{"$ref": "pet.yaml"}, itemSchemaOf(t, parsed)["default"], "default is data")
	assert.NotContains(t, bundled, "#/components/schemas/pet", "instance data is not lifted")
}

func TestBundleBytesComposed_MergedFragmentRootDefaultNotRewritten(t *testing.T) {
	_, parsed := bundleComposedFiles(t, map[string]string{
		"main.yaml": `openapi: 3.1.0
info:
  title: Merged fragment
  version: 1.0.0
paths: {}
components:
  schemas:
    Thing:
      $ref: 'thing.yaml'
    Pet:
      $ref: 'pet.yaml'
`,
		"thing.yaml": `<<: {type: object, description: merged}
default:
  $ref: 'pet.yaml'
`,
		"pet.yaml": instanceValuePetSchema,
	}, nil)

	schemas := composedSchemas(t, parsed)
	require.Contains(t, schemas, "pet")
	assert.Equal(t, "#/components/schemas/pet", schemas["Pet"].(map[string]any)["$ref"])
	assert.Equal(t, map[string]any{"$ref": "pet.yaml"}, schemas["Thing"].(map[string]any)["default"], "default is data")
}

func TestBundleBytesComposed_PropertyNamedExtensionInSubdirFileKeepsData(t *testing.T) {
	files := map[string]string{
		"main.yaml": `openapi: 3.1.0
info:
  title: Property named x-meta
  version: 1.0.0
paths: {}
components:
  schemas:
    Owner:
      $ref: 'schemas/owner.yaml'
`,
		"schemas/owner.yaml": `type: object
x-ext:
  $ref: 'pet.yaml'
properties:
  meta:
    type: object
    default: {$ref: 'pet.yaml'}
  x-meta:
    type: object
    default: {$ref: 'pet.yaml'}
    examples:
      - {$ref: 'pet.yaml#/properties'}
`,
		"schemas/pet.yaml": instanceValuePetSchema,
	}
	for _, exclude := range []bool{false, true} {
		t.Run(fmt.Sprintf("exclude_%v", exclude), func(t *testing.T) {
			_, parsed := bundleComposedFiles(t, files, &datamodel.DocumentConfiguration{ExcludeExtensionRefs: exclude})
			owner := composedSchemas(t, parsed)["owner"].(map[string]any)
			assert.Equal(t, map[string]any{"$ref": "schemas/pet.yaml"}, owner["x-ext"], "an extension's ref is rebased to the root")
			properties := owner["properties"].(map[string]any)
			meta := properties["meta"].(map[string]any)
			xMeta := properties["x-meta"].(map[string]any)
			assert.Equal(t, map[string]any{"$ref": "pet.yaml"}, meta["default"])
			assert.Equal(t, meta["default"], xMeta["default"], "a property named x-meta is a schema, and its default is data")
			assert.Equal(t, []any{map[string]any{"$ref": "pet.yaml#/properties"}}, xMeta["examples"])
		})
	}
}
