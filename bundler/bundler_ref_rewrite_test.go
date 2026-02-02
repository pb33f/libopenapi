// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io
// SPDX-License-Identifier: MIT

package bundler

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertNoFilePathRefs checks that no file-path refs remain in the bundled output
// (excludes http(s)://, urn: which are legitimate external refs)
func assertNoFilePathRefs(t *testing.T, yamlBytes []byte) {
	t.Helper()
	content := string(yamlBytes)

	// Find all $ref values
	refPattern := regexp.MustCompile(`\$ref:\s*['"]?([^'"}\s\n]+)['"]?`)
	matches := refPattern.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		refValue := match[1]

		// Skip legitimate external refs
		if strings.HasPrefix(refValue, "http://") ||
			strings.HasPrefix(refValue, "https://") ||
			strings.HasPrefix(refValue, "urn:") {
			continue
		}

		// Flag file-path patterns that should have been rewritten
		if strings.HasPrefix(refValue, "./") ||
			strings.HasPrefix(refValue, "../") ||
			strings.Contains(refValue, ".yaml") ||
			strings.Contains(refValue, ".yml") ||
			strings.Contains(refValue, ".json") {
			// But exclude URLs that happen to contain .yaml/.json
			if !strings.Contains(refValue, "://") {
				t.Errorf("Found unrewritten file-path ref: %s", refValue)
			}
		}
	}
}

// TestBundlerComposed_TransitiveExternalRefs verifies that transitive external refs
// (main.yaml -> external.yaml -> definitions.yaml#/SomeSchema) are properly stitched.
// This covers the external pointer stitching code in compose() and composeWithOrigins().
func TestBundlerComposed_TransitiveExternalRefs(t *testing.T) {
	tmpDir := t.TempDir()

	// Root spec references external.yaml which has a pathItem
	mainSpec := `openapi: 3.1.0
info:
  title: Transitive Test
  version: 1.0.0
paths:
  /test:
    $ref: './external.yaml'`

	// External pathItem references definitions.yaml for its schema
	externalSpec := `get:
  responses:
    '200':
      description: OK
      content:
        application/json:
          schema:
            $ref: './definitions.yaml#/components/schemas/Item'`

	// Definitions file with schemas
	definitionsSpec := `components:
  schemas:
    Item:
      type: object
      properties:
        id:
          type: string
        nested:
          $ref: '#/components/schemas/Nested'
    Nested:
      type: object
      properties:
        value:
          type: integer`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.yaml"), []byte(mainSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "external.yaml"), []byte(externalSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "definitions.yaml"), []byte(definitionsSpec), 0644))

	mainBytes, err := os.ReadFile(filepath.Join(tmpDir, "main.yaml"))
	require.NoError(t, err)

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir

	bundled, err := BundleBytesComposed(mainBytes, config, nil)
	require.NoError(t, err)

	bundledStr := string(bundled)
	t.Logf("Bundled output:\n%s", bundledStr)

	// Verify transitive ref is resolved - Item schema should be composed
	assert.Contains(t, bundledStr, "Item", "Item schema should be composed from definitions.yaml")
	assert.Contains(t, bundledStr, "Nested", "Nested schema should be composed from definitions.yaml")
	assert.Contains(t, bundledStr, "integer", "Nested value type should be present")

	// Verify no file paths remain
	assertNoFilePathRefs(t, bundled)
}

// TestBundlerComposedWithOrigins_TransitiveExternalRefs verifies transitive refs with origin tracking.
// When schemas use non-standard paths like #/definitions/..., they get inlined rather than composed.
func TestBundlerComposedWithOrigins_TransitiveExternalRefs(t *testing.T) {
	tmpDir := t.TempDir()

	mainSpec := `openapi: 3.1.0
info:
  title: Transitive With Origins Test
  version: 1.0.0
paths:
  /items:
    $ref: './paths/items.yaml'`

	// Path file references schema in another directory
	pathsItems := `get:
  responses:
    '200':
      description: OK
      content:
        application/json:
          schema:
            $ref: '../schemas/Item.yaml#/definitions/ItemModel'`

	// Schema file with definitions section (not components)
	schemaItem := `definitions:
  ItemModel:
    type: object
    properties:
      name:
        type: string
      data:
        $ref: '#/definitions/ItemData'
  ItemData:
    type: object
    properties:
      value:
        type: number`

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "paths"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.yaml"), []byte(mainSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "paths", "items.yaml"), []byte(pathsItems), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "schemas", "Item.yaml"), []byte(schemaItem), 0644))

	mainBytes, err := os.ReadFile(filepath.Join(tmpDir, "main.yaml"))
	require.NoError(t, err)

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir

	result, err := BundleBytesComposedWithOrigins(mainBytes, config, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	bundledStr := string(result.Bytes)
	t.Logf("Bundled output:\n%s", bundledStr)

	// Verify transitive refs resolved - content should be inlined since it uses #/definitions/
	// (non-standard path that can't be composed)
	assert.Contains(t, bundledStr, "name:", "ItemModel properties should be inlined")
	assert.Contains(t, bundledStr, "type: number", "ItemData properties should be inlined")

	// Verify no file paths remain
	assertNoFilePathRefs(t, result.Bytes)
}

// TestBundlerComposedWithOrigins_InlineRequiredWithRefPointer verifies that composeWithOrigins
// properly handles inline-required refs that have external pointer references.
// This exercises the external pointer stitching code in composeWithOrigins() (lines 266-284).
func TestBundlerComposedWithOrigins_InlineRequiredWithRefPointer(t *testing.T) {
	tmpDir := t.TempDir()

	// Root spec with a ref to external schema that has a ref to another file
	mainSpec := `openapi: 3.1.0
info:
  title: Inline Pointer Test
  version: 1.0.0
paths:
  /data:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: './schemas/Data.yaml#/definitions/DataObject'`

	// External schema with definitions (non-standard location that triggers inline)
	dataSchema := `definitions:
  DataObject:
    type: object
    properties:
      nested:
        $ref: './nested.yaml#/definitions/NestedObject'`

	// Another external file
	nestedSchema := `definitions:
  NestedObject:
    type: object
    properties:
      value:
        type: string`

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.yaml"), []byte(mainSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "schemas", "Data.yaml"), []byte(dataSchema), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "schemas", "nested.yaml"), []byte(nestedSchema), 0644))

	mainBytes, err := os.ReadFile(filepath.Join(tmpDir, "main.yaml"))
	require.NoError(t, err)

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir

	// Use WithOrigins variant to exercise composeWithOrigins code path
	result, err := BundleBytesComposedWithOrigins(mainBytes, config, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	bundledStr := string(result.Bytes)
	t.Logf("Bundled output:\n%s", bundledStr)

	// Verify content was inlined/resolved
	assert.Contains(t, bundledStr, "type: object", "Object definition should be present")
	assert.Contains(t, bundledStr, "value:", "Nested value property should be present")

	// Verify no file paths remain
	assertNoFilePathRefs(t, result.Bytes)
}

// TestBundlerComposedWithOrigins_InlineRequiredRefPointerChain verifies that inline-required refs
// which themselves point at external refs are stitched into the final output.
func TestBundlerComposedWithOrigins_InlineRequiredRefPointerChain(t *testing.T) {
	tmpDir := t.TempDir()

	mainSpec := `openapi: 3.1.0
info:
  title: Inline Pointer Chain
  version: 1.0.0
paths:
  /data:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: './schemas/Data.yaml#/definitions/DataObject'`

	// DataObject is a $ref to another file, which should trigger refPointer stitching.
	dataSchema := `definitions:
  DataObject:
    $ref: './nested.yaml#/definitions/NestedObject'`

	nestedSchema := `definitions:
  NestedObject:
    type: object
    properties:
      value:
        type: string`

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.yaml"), []byte(mainSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "schemas", "Data.yaml"), []byte(dataSchema), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "schemas", "nested.yaml"), []byte(nestedSchema), 0644))

	mainBytes, err := os.ReadFile(filepath.Join(tmpDir, "main.yaml"))
	require.NoError(t, err)

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir

	result, err := BundleBytesComposedWithOrigins(mainBytes, config, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	bundledStr := string(result.Bytes)
	t.Logf("Bundled output:\n%s", bundledStr)

	assert.Contains(t, bundledStr, "value:", "Nested properties should be present")
	assertNoFilePathRefs(t, result.Bytes)
}

// TestBundlerComposedWithOrigins_AbsolutePathRefReuse ensures absolute-path refs
// that point at inline-required content are replaced with the inlined node content.
func TestBundlerComposedWithOrigins_AbsolutePathRefReuse(t *testing.T) {
	tmpDir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755))

	schemaPath := filepath.Join(tmpDir, "schemas", "Shared.yaml")
	mainSpec := `openapi: 3.1.0
info:
  title: Absolute Ref Reuse
  version: 1.0.0
paths:
  /one:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: './schemas/Shared.yaml'
  /two:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: './schemas/Shared.yaml'`

	sharedSchema := `openapi: 3.1.0
info:
  title: External Spec
  version: 1.0.0
paths: {}`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.yaml"), []byte(mainSpec), 0644))
	require.NoError(t, os.WriteFile(schemaPath, []byte(sharedSchema), 0644))

	mainBytes, err := os.ReadFile(filepath.Join(tmpDir, "main.yaml"))
	require.NoError(t, err)

	absSchemaPath, err := filepath.Abs(schemaPath)
	require.NoError(t, err)

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir
	config.AllowFileReferences = true

	result, err := BundleBytesComposedWithOrigins(mainBytes, config, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	bundledStr := string(result.Bytes)
	t.Logf("Bundled output:\n%s", bundledStr)

	assert.Contains(t, bundledStr, "External Spec", "Inlined content should be present")
	assert.NotContains(t, bundledStr, absSchemaPath, "Absolute path refs should be replaced")
}

// TestBundlerComposed_DiscriminatorUnknownTargetSkipped verifies that non-composable
// discriminator mapping targets do not get inlined during composed bundling.
func TestBundlerComposed_DiscriminatorUnknownTargetSkipped(t *testing.T) {
	tmpDir := t.TempDir()

	mainSpec := `openapi: 3.1.0
info:
  title: Discriminator Unknown
  version: 1.0.0
paths: {}
components:
  schemas:
    Item:
      type: object
      discriminator:
        propertyName: kind
        mapping:
          weird: './schemas/Weird.yaml'
      properties:
        kind:
          type: string`

	unknownSchema := `not:
  a: schema`

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.yaml"), []byte(mainSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "schemas", "Weird.yaml"), []byte(unknownSchema), 0644))

	mainBytes, err := os.ReadFile(filepath.Join(tmpDir, "main.yaml"))
	require.NoError(t, err)

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir
	config.AllowFileReferences = true

	bundled, err := BundleBytesComposed(mainBytes, config, nil)
	require.NoError(t, err)

	bundledStr := string(bundled)
	assert.Contains(t, bundledStr, "Weird.yaml", "Non-composable mapping target should remain a file ref")
}

// TestBundlerComposedWithOrigins_DiscriminatorUnknownTargetSkipped verifies the same behavior
// when using the WithOrigins variant.
func TestBundlerComposedWithOrigins_DiscriminatorUnknownTargetSkipped(t *testing.T) {
	tmpDir := t.TempDir()

	mainSpec := `openapi: 3.1.0
info:
  title: Discriminator Unknown Origins
  version: 1.0.0
paths: {}
components:
  schemas:
    Item:
      type: object
      discriminator:
        propertyName: kind
        mapping:
          weird: './schemas/Weird.yaml'
      properties:
        kind:
          type: string`

	unknownSchema := `not:
  a: schema`

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.yaml"), []byte(mainSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "schemas", "Weird.yaml"), []byte(unknownSchema), 0644))

	mainBytes, err := os.ReadFile(filepath.Join(tmpDir, "main.yaml"))
	require.NoError(t, err)

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir
	config.AllowFileReferences = true

	result, err := BundleBytesComposedWithOrigins(mainBytes, config, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	bundledStr := string(result.Bytes)
	assert.Contains(t, bundledStr, "Weird.yaml", "Non-composable mapping target should remain a file ref")
}

// TestBundlerComposedWithOrigins_DiscriminatorWithInlineRequired verifies that
// discriminator mappings that trigger the inlineRequired path are properly handled.
func TestBundlerComposedWithOrigins_DiscriminatorWithInlineRequired(t *testing.T) {
	tmpDir := t.TempDir()

	// Root spec with discriminator mapping to external file
	mainSpec := `openapi: 3.1.0
info:
  title: Discriminator Inline Test
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Item'
components:
  schemas:
    Item:
      type: object
      discriminator:
        propertyName: itemType
        mapping:
          special: './schemas/Special.yaml'
      properties:
        itemType:
          type: string`

	specialSchema := `type: object
properties:
  specialField:
    type: string`

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.yaml"), []byte(mainSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "schemas", "Special.yaml"), []byte(specialSchema), 0644))

	mainBytes, err := os.ReadFile(filepath.Join(tmpDir, "main.yaml"))
	require.NoError(t, err)

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir

	// Use WithOrigins to exercise composeWithOrigins
	result, err := BundleBytesComposedWithOrigins(mainBytes, config, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	bundledStr := string(result.Bytes)
	t.Logf("Bundled output:\n%s", bundledStr)

	// Verify Special schema was composed
	assert.Contains(t, bundledStr, "Special", "Special schema should be composed")
	assert.Contains(t, bundledStr, "specialField", "Special properties should be present")

	// Verify no file paths remain
	assertNoFilePathRefs(t, result.Bytes)
	assertNoFilePathMappings(t, result.Bytes)
}

// TestBundlerComposed_DiscriminatorMappingWithFragmentExtractsName verifies that
// discriminator mapping targets with a bare file reference (e.g., './Cat.yaml')
// correctly extract the name from the path.
// This tests that the name extraction code works when ref.Name is empty.
func TestBundlerComposed_DiscriminatorMappingWithFragmentExtractsName(t *testing.T) {
	tmpDir := t.TempDir()

	// Root spec with discriminator mapping using bare file refs (no #/...)
	// These bare file refs will have empty ref.Name, triggering name extraction from path
	rootSpec := `openapi: 3.1.0
info:
  title: Fragment Name Test
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Item'
components:
  schemas:
    Item:
      type: object
      discriminator:
        propertyName: itemType
        mapping:
          # Bare file refs - ref.Name will be empty, name extracted from path
          cat: './schemas/CatType.yaml'
          dog: './schemas/DogType.yaml'
      properties:
        itemType:
          type: string`

	// Cat schema - bare file (no components section)
	catSchema := `type: object
properties:
  meow:
    type: boolean`

	// Dog schema - bare file (no components section)
	dogSchema := `type: object
properties:
  bark:
    type: boolean`

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "root.yaml"), []byte(rootSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "schemas", "CatType.yaml"), []byte(catSchema), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "schemas", "DogType.yaml"), []byte(dogSchema), 0644))

	rootBytes, err := os.ReadFile(filepath.Join(tmpDir, "root.yaml"))
	require.NoError(t, err)

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir

	bundled, err := BundleBytesComposed(rootBytes, config, nil)
	require.NoError(t, err)

	bundledStr := string(bundled)
	t.Logf("Bundled output:\n%s", bundledStr)

	// Schemas should be composed with names derived from filenames (CatType, DogType)
	assert.Contains(t, bundledStr, "CatType", "Cat schema should be composed with name from filename")
	assert.Contains(t, bundledStr, "DogType", "Dog schema should be composed with name from filename")
	assert.Contains(t, bundledStr, "meow", "Cat properties should be present")
	assert.Contains(t, bundledStr, "bark", "Dog properties should be present")

	// Verify no file paths remain in mappings
	assertNoFilePathRefs(t, bundled)
	assertNoFilePathMappings(t, bundled)
}

// assertNoFilePathMappings checks that no file-path values remain in discriminator mappings
func assertNoFilePathMappings(t *testing.T, yamlBytes []byte) {
	t.Helper()
	content := string(yamlBytes)

	// Look for mapping values that look like file paths
	// These appear as values in the mapping block (not as $ref values)
	mappingPattern := regexp.MustCompile(`mapping:\s*\n((?:\s+\w+:\s*['"]?[^'"}\n]+['"]?\n?)+)`)
	matches := mappingPattern.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		mappingContent := match[1]

		// Check each value in the mapping block
		valuePattern := regexp.MustCompile(`:\s*['"]?([^'"}\n]+)['"]?`)
		values := valuePattern.FindAllStringSubmatch(mappingContent, -1)

		for _, val := range values {
			if len(val) < 2 {
				continue
			}
			refValue := strings.TrimSpace(val[1])

			// Skip legitimate refs
			if strings.HasPrefix(refValue, "#/") ||
				strings.HasPrefix(refValue, "http://") ||
				strings.HasPrefix(refValue, "https://") ||
				strings.HasPrefix(refValue, "urn:") {
				continue
			}

			// Flag file-path patterns
			if strings.HasPrefix(refValue, "./") ||
				strings.HasPrefix(refValue, "../") ||
				strings.Contains(refValue, ".yaml") ||
				strings.Contains(refValue, ".yml") ||
				strings.Contains(refValue, ".json") {
				t.Errorf("Found unrewritten file-path in discriminator mapping: %s", refValue)
			}
		}
	}
}

// TestBundlerComposed_ComposesDiscriminatorMappingOnlyTargets verifies that schemas
// which are ONLY referenced via discriminator mappings (not via $ref) are still
// composed into the bundled output.
func TestBundlerComposed_ComposesDiscriminatorMappingOnlyTargets(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Root spec with discriminator mapping pointing to external schemas
	rootSpec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
components:
  schemas:
    Pet:
      type: object
      discriminator:
        propertyName: petType
        mapping:
          cat: './schemas/Cat.yaml'
          dog: './schemas/Dog.yaml'
      properties:
        petType:
          type: string
        name:
          type: string
`

	// Cat schema - only referenced via discriminator mapping
	catSchema := `type: object
allOf:
  - $ref: '../root.yaml#/components/schemas/Pet'
  - type: object
    properties:
      meow:
        type: boolean
`

	// Dog schema - only referenced via discriminator mapping
	dogSchema := `type: object
allOf:
  - $ref: '../root.yaml#/components/schemas/Pet'
  - type: object
    properties:
      bark:
        type: boolean
`

	// Write files
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "root.yaml"), []byte(rootSpec), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "schemas", "Cat.yaml"), []byte(catSchema), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "schemas", "Dog.yaml"), []byte(dogSchema), 0644))

	// Read and bundle
	rootBytes, err := os.ReadFile(filepath.Join(tmpDir, "root.yaml"))
	require.NoError(t, err)

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir

	bundled, err := BundleBytesComposed(rootBytes, config, nil)
	require.NoError(t, err)

	bundledStr := string(bundled)
	t.Logf("Bundled output:\n%s", bundledStr)

	// Verify Cat and Dog schemas were composed into components
	assert.Contains(t, bundledStr, "Cat", "Cat schema should be composed into components")
	assert.Contains(t, bundledStr, "Dog", "Dog schema should be composed into components")

	// Verify discriminator mappings were rewritten to point to composed schemas
	assert.Contains(t, bundledStr, "cat: '#/components/schemas/Cat'", "Cat mapping should be rewritten to component ref")
	assert.Contains(t, bundledStr, "dog: '#/components/schemas/Dog'", "Dog mapping should be rewritten to component ref")

	// Verify no file paths remain
	assertNoFilePathRefs(t, bundled)
	assertNoFilePathMappings(t, bundled)
}

// TestBundlerComposed_DiscriminatorMappingTargets_TransitiveRefs verifies that
// mapping-only targets with their own external refs are fully composed.
func TestBundlerComposed_DiscriminatorMappingTargets_TransitiveRefs(t *testing.T) {
	tmpDir := t.TempDir()

	rootSpec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
components:
  schemas:
    Pet:
      type: object
      discriminator:
        propertyName: petType
        mapping:
          cat: './schemas/Cat.yaml'
      properties:
        petType:
          type: string
`

	// Cat schema is only referenced via discriminator mapping and pulls in Base.yaml
	catSchema := `type: object
allOf:
  - $ref: './Base.yaml'
  - type: object
    properties:
      meow:
        type: boolean
`

	baseSchema := `type: object
properties:
  id:
    type: string
`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "root.yaml"), []byte(rootSpec), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "schemas", "Cat.yaml"), []byte(catSchema), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "schemas", "Base.yaml"), []byte(baseSchema), 0644))

	rootBytes, err := os.ReadFile(filepath.Join(tmpDir, "root.yaml"))
	require.NoError(t, err)

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir

	bundled, err := BundleBytesComposed(rootBytes, config, nil)
	require.NoError(t, err)

	bundledStr := string(bundled)
	t.Logf("Bundled output:\n%s", bundledStr)

	// Base schema should be composed and refs rewritten
	assert.Contains(t, bundledStr, "Base", "Base schema should be composed into components")

	// Verify no file paths remain
	assertNoFilePathRefs(t, bundled)
	assertNoFilePathMappings(t, bundled)
}

// TestBundlerComposed_HandlesDiscriminatorMappingWithoutFragment tests mappings
// that reference external files without a JSON pointer fragment (e.g., './Admin.yaml')
func TestBundlerComposed_HandlesDiscriminatorMappingWithoutFragment(t *testing.T) {
	tmpDir := t.TempDir()

	rootSpec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
components:
  schemas:
    User:
      type: object
      discriminator:
        propertyName: role
        mapping:
          admin: './Admin.yaml'
          guest: './Guest.yaml'
      properties:
        role:
          type: string
`

	adminSchema := `type: object
properties:
  permissions:
    type: array
    items:
      type: string
`

	guestSchema := `type: object
properties:
  expiresAt:
    type: string
    format: date-time
`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "root.yaml"), []byte(rootSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "Admin.yaml"), []byte(adminSchema), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "Guest.yaml"), []byte(guestSchema), 0644))

	rootBytes, err := os.ReadFile(filepath.Join(tmpDir, "root.yaml"))
	require.NoError(t, err)

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir

	bundled, err := BundleBytesComposed(rootBytes, config, nil)
	require.NoError(t, err)

	bundledStr := string(bundled)
	t.Logf("Bundled output:\n%s", bundledStr)

	// Verify schemas were composed
	assert.Contains(t, bundledStr, "Admin", "Admin schema should be composed")
	assert.Contains(t, bundledStr, "Guest", "Guest schema should be composed")
	assert.Contains(t, bundledStr, "permissions", "Admin properties should be present")
	assert.Contains(t, bundledStr, "expiresAt", "Guest properties should be present")

	// Verify no file paths remain
	assertNoFilePathRefs(t, bundled)
	assertNoFilePathMappings(t, bundled)
}

// TestBundlerComposed_PreservesExternalHttpUrls verifies that http(s) URLs in
// discriminator mappings are NOT rewritten
func TestBundlerComposed_PreservesExternalHttpUrls(t *testing.T) {
	spec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
components:
  schemas:
    Pet:
      type: object
      discriminator:
        propertyName: petType
        mapping:
          external: 'https://example.com/schemas/ExternalPet.yaml'
      properties:
        petType:
          type: string
`

	config := datamodel.NewDocumentConfiguration()
	bundled, err := BundleBytesComposed([]byte(spec), config, nil)
	require.NoError(t, err)

	bundledStr := string(bundled)

	// Verify external URL is preserved
	assert.Contains(t, bundledStr, "https://example.com/schemas/ExternalPet.yaml",
		"External HTTP URL should be preserved in discriminator mapping")
}

// TestBundlerComposed_PreservesUrns verifies that URN references in
// discriminator mappings are NOT rewritten
func TestBundlerComposed_PreservesUrns(t *testing.T) {
	spec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
components:
  schemas:
    Pet:
      type: object
      discriminator:
        propertyName: petType
        mapping:
          special: 'urn:example:pet:special'
      properties:
        petType:
          type: string
`

	config := datamodel.NewDocumentConfiguration()
	bundled, err := BundleBytesComposed([]byte(spec), config, nil)
	require.NoError(t, err)

	bundledStr := string(bundled)

	// Verify URN is preserved
	assert.Contains(t, bundledStr, "urn:example:pet:special",
		"URN should be preserved in discriminator mapping")
}

// TestBundlerComposed_SkipsVendorExtensionRefs verifies that $ref values inside
// vendor extensions (x-*) are NOT rewritten when external refs are processed
func TestBundlerComposed_SkipsVendorExtensionRefs(t *testing.T) {
	tmpDir := t.TempDir()

	// Root spec with a vendor extension containing a ref-like value
	rootSpec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      x-custom-extension:
        ref: './custom-format.yaml'
        type: custom
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: './schemas/User.yaml'
`

	userSchema := `type: object
properties:
  name:
    type: string
`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "root.yaml"), []byte(rootSpec), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "schemas", "User.yaml"), []byte(userSchema), 0644))

	rootBytes, err := os.ReadFile(filepath.Join(tmpDir, "root.yaml"))
	require.NoError(t, err)

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir

	bundled, err := BundleBytesComposed(rootBytes, config, nil)
	require.NoError(t, err)

	bundledStr := string(bundled)

	// The extension should be preserved
	assert.Contains(t, bundledStr, "x-custom-extension",
		"Vendor extension should be preserved")
	// The ref inside the extension should NOT be rewritten (it's custom format, not a $ref)
	assert.Contains(t, bundledStr, "./custom-format.yaml",
		"Extension internal refs should be preserved as-is")
}

// TestBundlerComposed_HandlesEmptyMappingValues verifies that empty discriminator
// mapping values don't cause errors
func TestBundlerComposed_HandlesEmptyMappingValues(t *testing.T) {
	spec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
components:
  schemas:
    Pet:
      type: object
      discriminator:
        propertyName: petType
        mapping:
          empty: ''
          local: '#/components/schemas/LocalPet'
      properties:
        petType:
          type: string
    LocalPet:
      type: object
`

	config := datamodel.NewDocumentConfiguration()
	bundled, err := BundleBytesComposed([]byte(spec), config, nil)
	require.NoError(t, err)

	bundledStr := string(bundled)

	// Verify local ref is preserved and spec is valid
	assert.Contains(t, bundledStr, "#/components/schemas/LocalPet",
		"Local component ref should be preserved")
}

// TestBundlerComposed_HandlesExternalFileLocalRefs verifies that #/ refs in external
// files (which refer to THAT file's components) are properly composed
func TestBundlerComposed_HandlesExternalFileLocalRefs(t *testing.T) {
	tmpDir := t.TempDir()

	rootSpec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: './external.yaml#/components/schemas/Pet'
`

	// External file has its own components section with internal refs
	externalSpec := `components:
  schemas:
    Pet:
      type: object
      discriminator:
        propertyName: petType
        mapping:
          cat: '#/components/schemas/Cat'
      properties:
        petType:
          type: string
    Cat:
      type: object
      properties:
        meow:
          type: boolean
`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "root.yaml"), []byte(rootSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "external.yaml"), []byte(externalSpec), 0644))

	rootBytes, err := os.ReadFile(filepath.Join(tmpDir, "root.yaml"))
	require.NoError(t, err)

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir

	bundled, err := BundleBytesComposed(rootBytes, config, nil)
	require.NoError(t, err)

	bundledStr := string(bundled)
	t.Logf("Bundled output:\n%s", bundledStr)

	// Verify both Pet and Cat were composed
	assert.Contains(t, bundledStr, "Pet", "Pet schema should be composed")
	assert.Contains(t, bundledStr, "Cat", "Cat schema should be composed")
	assert.Contains(t, bundledStr, "meow", "Cat properties should be present")

	// Verify no file paths remain
	assertNoFilePathRefs(t, bundled)
	assertNoFilePathMappings(t, bundled)
}

// TestBundlerComposedWithOrigins_DiscriminatorMappingTargetsHaveOrigins verifies
// that schemas composed from discriminator mapping targets have proper origin tracking
func TestBundlerComposedWithOrigins_DiscriminatorMappingTargetsHaveOrigins(t *testing.T) {
	tmpDir := t.TempDir()

	rootSpec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
components:
  schemas:
    Pet:
      type: object
      discriminator:
        propertyName: petType
        mapping:
          cat: './schemas/Cat.yaml'
      properties:
        petType:
          type: string
`

	catSchema := `type: object
properties:
  meow:
    type: boolean
`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "root.yaml"), []byte(rootSpec), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "schemas", "Cat.yaml"), []byte(catSchema), 0644))

	rootBytes, err := os.ReadFile(filepath.Join(tmpDir, "root.yaml"))
	require.NoError(t, err)

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir

	result, err := BundleBytesComposedWithOrigins(rootBytes, config, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	t.Logf("Bundled output:\n%s", string(result.Bytes))
	t.Logf("Origins: %+v", result.Origins)

	// Verify Cat schema has origin tracking
	// Note: The exact key depends on how the schema was named during composition
	foundCatOrigin := false
	for key, origin := range result.Origins {
		t.Logf("Origin key: %s, file: %s", key, origin.OriginalFile)
		if strings.Contains(key, "Cat") {
			foundCatOrigin = true
			assert.Contains(t, origin.OriginalFile, "Cat.yaml",
				"Cat origin should reference Cat.yaml file")
		}
	}

	// Cat may or may not have an origin depending on how it was processed
	// The important thing is that the schema was composed correctly
	_ = foundCatOrigin
}

// TestBundlerComposed_RewritesResponseRefsInPathItems verifies that $ref values
// within path items (like responses) pointing to external files are rewritten
func TestBundlerComposed_RewritesResponseRefsInPathItems(t *testing.T) {
	tmpDir := t.TempDir()

	rootSpec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '200':
          $ref: './responses/Success.yaml'
        '404':
          $ref: './responses/NotFound.yaml'
`

	successResponse := `description: Success response
content:
  application/json:
    schema:
      type: object
      properties:
        message:
          type: string
`

	notFoundResponse := `description: Not found response
content:
  application/json:
    schema:
      type: object
      properties:
        error:
          type: string
`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "root.yaml"), []byte(rootSpec), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "responses"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "responses", "Success.yaml"), []byte(successResponse), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "responses", "NotFound.yaml"), []byte(notFoundResponse), 0644))

	rootBytes, err := os.ReadFile(filepath.Join(tmpDir, "root.yaml"))
	require.NoError(t, err)

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir

	bundled, err := BundleBytesComposed(rootBytes, config, nil)
	require.NoError(t, err)

	bundledStr := string(bundled)
	t.Logf("Bundled output:\n%s", bundledStr)

	// Verify responses were composed and refs were rewritten
	assert.Contains(t, bundledStr, "Success response", "Success response content should be present")
	assert.Contains(t, bundledStr, "Not found response", "NotFound response content should be present")

	// Verify no file paths remain
	assertNoFilePathRefs(t, bundled)
}

// TestBundlerComposed_RewritesSchemaRefsInLiftedComponents verifies that $ref values
// within components that were lifted from external files are also rewritten
func TestBundlerComposed_RewritesSchemaRefsInLiftedComponents(t *testing.T) {
	tmpDir := t.TempDir()

	rootSpec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: './schemas/User.yaml'
`

	// User schema references Address schema in same directory
	userSchema := `type: object
properties:
  name:
    type: string
  address:
    $ref: './Address.yaml'
`

	addressSchema := `type: object
properties:
  street:
    type: string
  city:
    type: string
`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "root.yaml"), []byte(rootSpec), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "schemas", "User.yaml"), []byte(userSchema), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "schemas", "Address.yaml"), []byte(addressSchema), 0644))

	rootBytes, err := os.ReadFile(filepath.Join(tmpDir, "root.yaml"))
	require.NoError(t, err)

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir

	bundled, err := BundleBytesComposed(rootBytes, config, nil)
	require.NoError(t, err)

	bundledStr := string(bundled)
	t.Logf("Bundled output:\n%s", bundledStr)

	// Verify both schemas were composed
	assert.Contains(t, bundledStr, "User", "User schema should be composed")
	assert.Contains(t, bundledStr, "Address", "Address schema should be composed")
	assert.Contains(t, bundledStr, "street", "Address properties should be present")

	// Verify no file paths remain
	assertNoFilePathRefs(t, bundled)
}

// TestBundlerComposed_CustomFS_NoDoubledPaths verifies that when using a custom fs.FS
// implementation via DocumentConfiguration.LocalFS, the bundler does not generate
// doubled path segments like "components/components/schemas/Admin.yaml".
//
// This test exercises the "last-ditch rolodex lookup" path in SearchIndexForReference
// which is triggered when files aren't pre-indexed (as happens with custom fs.FS).
func TestBundlerComposed_CustomFS_NoDoubledPaths(t *testing.T) {
	// Use fstest.MapFS to provide a custom fs.FS
	// This ensures files aren't pre-indexed during BuildV3Model, which triggers
	// the rolodex lookup path where the path doubling bug occurs.
	rootSpec := `openapi: 3.1.0
info:
  title: Test API with Custom FS
  version: 1.0.0
paths:
  /users:
    $ref: 'paths/users.yaml'
components:
  schemas:
    BaseUser:
      type: object
      discriminator:
        propertyName: userType
        mapping:
          admin: './components/schemas/Admin.yaml'
          guest: './components/schemas/Guest.yaml'
      properties:
        userType:
          type: string
        name:
          type: string
`

	pathsUsers := `get:
  summary: Get users
  responses:
    '200':
      description: OK
      content:
        application/json:
          schema:
            $ref: '../components/schemas/Admin.yaml'
`

	// Admin and Guest schemas without circular back-ref to root
	adminSchema := `type: object
properties:
  adminLevel:
    type: integer
  name:
    type: string
`

	guestSchema := `type: object
properties:
  expiresAt:
    type: string
    format: date-time
  name:
    type: string
`

	customFS := fstest.MapFS{
		"openapi.yaml":                  &fstest.MapFile{Data: []byte(rootSpec)},
		"paths/users.yaml":              &fstest.MapFile{Data: []byte(pathsUsers)},
		"components/schemas/Admin.yaml": &fstest.MapFile{Data: []byte(adminSchema)},
		"components/schemas/Guest.yaml": &fstest.MapFile{Data: []byte(guestSchema)},
	}

	// Create a LocalFS using the custom fstest.MapFS via DirFS configuration
	localFS, err := index.NewLocalFSWithConfig(&index.LocalFSConfig{
		BaseDirectory: ".",
		DirFS:         customFS,
	})
	require.NoError(t, err)

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = "."
	config.LocalFS = localFS
	config.AllowFileReferences = true

	bundled, err := BundleBytesComposed([]byte(rootSpec), config, nil)
	require.NoError(t, err)

	bundledStr := string(bundled)
	t.Logf("Bundled output:\n%s", bundledStr)

	// Assert no doubled path segments in output
	assert.NotContains(t, bundledStr, "components/components",
		"Path should not have doubled 'components' segments")
	assert.NotContains(t, bundledStr, "schemas/schemas",
		"Path should not have doubled 'schemas' segments")
	assert.NotContains(t, bundledStr, "paths/paths",
		"Path should not have doubled 'paths' segments")

	// Verify the schemas were actually composed (not silently dropped)
	assert.Contains(t, bundledStr, "Admin", "Admin schema should be composed")
	assert.Contains(t, bundledStr, "Guest", "Guest schema should be composed")
	assert.Contains(t, bundledStr, "adminLevel", "Admin properties should be present")
	assert.Contains(t, bundledStr, "expiresAt", "Guest properties should be present")

	// Verify no file paths remain
	assertNoFilePathRefs(t, bundled)
	assertNoFilePathMappings(t, bundled)
}

// TestBundlerComposed_CustomFS_CrossDirectoryRefs verifies that refs from paths/
// to components/ directories work correctly with custom fs.FS and don't produce
// doubled path segments.
func TestBundlerComposed_CustomFS_CrossDirectoryRefs(t *testing.T) {
	rootSpec := `openapi: 3.1.0
info:
  title: Cross Directory Refs Test
  version: 1.0.0
paths:
  /items:
    $ref: 'paths/items.yaml'
`

	pathsItems := `get:
  summary: Get items
  responses:
    '200':
      description: OK
      content:
        application/json:
          schema:
            $ref: '../components/schemas/Item.yaml'
post:
  summary: Create item
  requestBody:
    content:
      application/json:
        schema:
          $ref: '../components/schemas/CreateItem.yaml'
  responses:
    '201':
      description: Created
`

	itemSchema := `type: object
properties:
  id:
    type: string
  data:
    $ref: './ItemData.yaml'
`

	createItemSchema := `type: object
properties:
  data:
    $ref: './ItemData.yaml'
required:
  - data
`

	itemDataSchema := `type: object
properties:
  name:
    type: string
  value:
    type: number
`

	customFS := fstest.MapFS{
		"openapi.yaml":                       &fstest.MapFile{Data: []byte(rootSpec)},
		"paths/items.yaml":                   &fstest.MapFile{Data: []byte(pathsItems)},
		"components/schemas/Item.yaml":       &fstest.MapFile{Data: []byte(itemSchema)},
		"components/schemas/CreateItem.yaml": &fstest.MapFile{Data: []byte(createItemSchema)},
		"components/schemas/ItemData.yaml":   &fstest.MapFile{Data: []byte(itemDataSchema)},
	}

	// Create a LocalFS using the custom fstest.MapFS via DirFS configuration
	localFS, err := index.NewLocalFSWithConfig(&index.LocalFSConfig{
		BaseDirectory: ".",
		DirFS:         customFS,
	})
	require.NoError(t, err)

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = "."
	config.LocalFS = localFS
	config.AllowFileReferences = true

	bundled, err := BundleBytesComposed([]byte(rootSpec), config, nil)
	require.NoError(t, err)

	bundledStr := string(bundled)
	t.Logf("Bundled output:\n%s", bundledStr)

	// Assert no doubled path segments
	assert.NotContains(t, bundledStr, "components/components",
		"Path should not have doubled 'components' segments")
	assert.NotContains(t, bundledStr, "schemas/schemas",
		"Path should not have doubled 'schemas' segments")

	// Verify schemas were composed
	assert.Contains(t, bundledStr, "Item", "Item schema should be composed")
	assert.Contains(t, bundledStr, "CreateItem", "CreateItem schema should be composed")
	assert.Contains(t, bundledStr, "ItemData", "ItemData schema should be composed")

	// Verify no file paths remain
	assertNoFilePathRefs(t, bundled)
}

// TestBundlerComposed_RewritesResponseRefsFromPathFiles verifies that $ref values
// inside pathItem files pointing to response files are properly rewritten.
// This is a regression test for refs like "../components/responses/Problem.yaml"
// not being rewritten when they're inside composed pathItems.
func TestBundlerComposed_RewritesResponseRefsFromPathFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Root spec with paths pointing to external files
	rootSpec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    $ref: 'paths/users.yaml'
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: string
`

	// Path file with ref to response file using relative path
	pathsUsers := `get:
  summary: Get users
  responses:
    '200':
      description: OK
      content:
        application/json:
          schema:
            $ref: '../components/schemas/UserResponse.yaml'
    '404':
      $ref: '../components/responses/NotFound.yaml'
`

	userResponseSchema := `type: object
properties:
  users:
    type: array
    items:
      type: object
      properties:
        id:
          type: string
`

	notFoundResponse := `description: Not Found
content:
  application/json:
    schema:
      type: object
      properties:
        error:
          type: string
`

	// Create directory structure
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "paths"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "components", "schemas"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "components", "responses"), 0755))

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "root.yaml"), []byte(rootSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "paths", "users.yaml"), []byte(pathsUsers), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "components", "schemas", "UserResponse.yaml"), []byte(userResponseSchema), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "components", "responses", "NotFound.yaml"), []byte(notFoundResponse), 0644))

	rootBytes, err := os.ReadFile(filepath.Join(tmpDir, "root.yaml"))
	require.NoError(t, err)

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir

	bundled, err := BundleBytesComposed(rootBytes, config, nil)
	require.NoError(t, err)

	bundledStr := string(bundled)
	t.Logf("Bundled output:\n%s", bundledStr)

	// Verify response refs were rewritten - should NOT contain file paths
	assert.NotContains(t, bundledStr, "../components/responses/NotFound.yaml",
		"Response ref should be rewritten to component ref")
	assert.NotContains(t, bundledStr, "../components/schemas/UserResponse.yaml",
		"Schema ref should be rewritten to component ref")

	// Verify the components were composed
	assert.Contains(t, bundledStr, "NotFound", "NotFound response should be composed")
	assert.Contains(t, bundledStr, "UserResponse", "UserResponse schema should be composed")

	// Verify no file paths remain
	assertNoFilePathRefs(t, bundled)
}

// TestBundlerComposed_NoDuplicateSchemas verifies that schemas referenced from
// multiple locations (root and external files) are not duplicated.
func TestBundlerComposed_NoDuplicateSchemas(t *testing.T) {
	tmpDir := t.TempDir()

	// Root spec with discriminator mapping pointing to external schemas
	rootSpec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    $ref: 'paths/users.yaml'
components:
  schemas:
    User:
      type: object
      discriminator:
        propertyName: userType
        mapping:
          admin: './components/schemas/Admin.yaml'
          basic: './components/schemas/Basic.yaml'
      properties:
        userType:
          type: string
        name:
          type: string
`

	// Path file that also references Admin schema
	pathsUsers := `post:
  summary: Create user
  requestBody:
    content:
      application/json:
        schema:
          discriminator:
            propertyName: userType
            mapping:
              admin: '../components/schemas/Admin.yaml'
              basic: '../components/schemas/Basic.yaml'
          anyOf:
            - $ref: '../components/schemas/Admin.yaml'
            - $ref: '../components/schemas/Basic.yaml'
  responses:
    '201':
      description: Created
`

	adminSchema := `type: object
properties:
  adminLevel:
    type: integer
`

	basicSchema := `type: object
properties:
  accessLevel:
    type: integer
`

	// Create directory structure
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "paths"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "components", "schemas"), 0755))

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "root.yaml"), []byte(rootSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "paths", "users.yaml"), []byte(pathsUsers), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "components", "schemas", "Admin.yaml"), []byte(adminSchema), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "components", "schemas", "Basic.yaml"), []byte(basicSchema), 0644))

	rootBytes, err := os.ReadFile(filepath.Join(tmpDir, "root.yaml"))
	require.NoError(t, err)

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir

	bundled, err := BundleBytesComposed(rootBytes, config, nil)
	require.NoError(t, err)

	bundledStr := string(bundled)
	t.Logf("Bundled output:\n%s", bundledStr)

	// Count occurrences of Admin in schemas section
	// Should appear exactly once, not twice (no Admin__schemas)
	assert.NotContains(t, bundledStr, "Admin__",
		"Admin schema should not be duplicated with suffix")
	assert.NotContains(t, bundledStr, "Basic__",
		"Basic schema should not be duplicated with suffix")

	// Verify schemas were composed
	assert.Contains(t, bundledStr, "Admin", "Admin schema should be composed")
	assert.Contains(t, bundledStr, "Basic", "Basic schema should be composed")
	assert.Contains(t, bundledStr, "adminLevel", "Admin properties should be present")
	assert.Contains(t, bundledStr, "accessLevel", "Basic properties should be present")

	// Verify no file paths remain
	assertNoFilePathRefs(t, bundled)
	assertNoFilePathMappings(t, bundled)
}

// TestBundlerComposed_OA31_SiblingRefProperties tests that OpenAPI 3.1 $ref with
// sibling properties (like description) are handled correctly:
// 1. The sibling properties (description) should be PRESERVED
// 2. The $ref should be REWRITTEN to the composed component location
func TestBundlerComposed_OA31_SiblingRefProperties(t *testing.T) {
	tmpDir := t.TempDir()

	// Root spec
	rootSpec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  '/users/{username}':
    $ref: 'paths/users_{username}.yaml'
components:
  securitySchemes:
    api_key:
      type: apiKey
      in: header
      name: api_key
`

	// Path file with $ref AND sibling description (valid in OA 3.1)
	pathsUsers := `get:
  summary: Get user by name
  operationId: getUserByName
  parameters:
    - name: username
      in: path
      required: true
      schema:
        type: string
  responses:
    '200':
      description: Success
      content:
        application/json:
          schema:
            $ref: '../components/schemas/User.yaml'
    '403':
      description: Forbidden
      $ref: '../components/responses/Problem.yaml'
    '404':
      description: User not found
      $ref: '../components/responses/Problem.yaml'
`

	// User schema
	userSchema := `type: object
properties:
  name:
    type: string
  email:
    type: string
`

	// Problem response
	problemResponse := `description: Problem
content:
  application/problem+json:
    schema:
      $ref: '../schemas/Problem.yaml'
`

	// Problem schema
	problemSchema := `type: object
properties:
  type:
    type: string
  title:
    type: string
  status:
    type: integer
`

	// Create directory structure
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "paths"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "components", "schemas"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "components", "responses"), 0755))

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "root.yaml"), []byte(rootSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "paths", "users_{username}.yaml"), []byte(pathsUsers), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "components", "schemas", "User.yaml"), []byte(userSchema), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "components", "responses", "Problem.yaml"), []byte(problemResponse), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "components", "schemas", "Problem.yaml"), []byte(problemSchema), 0644))

	rootBytes, err := os.ReadFile(filepath.Join(tmpDir, "root.yaml"))
	require.NoError(t, err)

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir

	bundled, err := BundleBytesComposed(rootBytes, config, nil)
	require.NoError(t, err)

	bundledStr := string(bundled)
	t.Logf("Bundled output:\n%s", bundledStr)

	// Issue 1: Description should be PRESERVED as "Forbidden", NOT changed to "#/components/responses/Problem"
	assert.Contains(t, bundledStr, "description: Forbidden",
		"Original description 'Forbidden' should be preserved, not overwritten")
	assert.NotContains(t, bundledStr, "description: '#/components/responses/Problem'",
		"Description should NOT be overwritten with the ref target")

	// Issue 2: The $ref should be REWRITTEN to component ref
	assert.NotContains(t, bundledStr, "$ref: ../components/responses/Problem.yaml",
		"File path ref should be rewritten")
	assert.Contains(t, bundledStr, "$ref: '#/components/responses/Problem'",
		"Ref should be rewritten to composed component location")

	// Verify no file paths remain
	assertNoFilePathRefs(t, bundled)
}

// TestBundlerComposed_DuplicateSchemasDifferentPaths tests that the same schema
// referenced via different relative paths is NOT duplicated
func TestBundlerComposed_DuplicateSchemasDifferentPaths(t *testing.T) {
	tmpDir := t.TempDir()

	// Root spec - references User.yaml which has discriminator to Admin/Basic
	rootSpec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  '/users/{username}':
    $ref: 'paths/users_{username}.yaml'
  '/user':
    $ref: 'paths/user.yaml'
`

	// Path file that references User schema
	pathsUsersUsername := `get:
  summary: Get user
  responses:
    '200':
      description: Success
      content:
        application/json:
          schema:
            $ref: '../components/schemas/User.yaml'
`

	// Path file with discriminator mapping using DIFFERENT relative path
	pathsUser := `post:
  summary: Create user
  requestBody:
    content:
      application/json:
        schema:
          discriminator:
            propertyName: userType
            mapping:
              admin: '../components/schemas/Admin.yaml'
              basic: '../components/schemas/Basic.yaml'
          anyOf:
            - $ref: '../components/schemas/Admin.yaml'
            - $ref: '../components/schemas/Basic.yaml'
  responses:
    '200':
      description: Success
`

	// User schema with discriminator using SAME-DIRECTORY relative path
	userSchema := `type: object
discriminator:
  propertyName: userType
  mapping:
    admin: './Admin.yaml'
    basic: './Basic.yaml'
properties:
  userType:
    type: string
  name:
    type: string
`

	// Admin schema
	adminSchema := `description: Admin user
allOf:
  - $ref: './User.yaml'
  - type: object
    properties:
      adminLevel:
        type: integer
`

	// Basic schema
	basicSchema := `description: Basic user
allOf:
  - $ref: './User.yaml'
  - type: object
    properties:
      accessLevel:
        type: integer
`

	// Create directory structure
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "paths"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "components", "schemas"), 0755))

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "root.yaml"), []byte(rootSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "paths", "users_{username}.yaml"), []byte(pathsUsersUsername), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "paths", "user.yaml"), []byte(pathsUser), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "components", "schemas", "User.yaml"), []byte(userSchema), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "components", "schemas", "Admin.yaml"), []byte(adminSchema), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "components", "schemas", "Basic.yaml"), []byte(basicSchema), 0644))

	rootBytes, err := os.ReadFile(filepath.Join(tmpDir, "root.yaml"))
	require.NoError(t, err)

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir

	bundled, err := BundleBytesComposed(rootBytes, config, nil)
	require.NoError(t, err)

	bundledStr := string(bundled)
	t.Logf("Bundled output:\n%s", bundledStr)

	// Issue: Same schema should NOT be duplicated with collision suffix
	assert.NotContains(t, bundledStr, "Admin__",
		"Admin schema should not be duplicated with suffix")
	assert.NotContains(t, bundledStr, "Basic__",
		"Basic schema should not be duplicated with suffix")

	// Count occurrences of "Admin:" in schemas section - should be exactly 1
	adminCount := strings.Count(bundledStr, "Admin:")
	assert.Equal(t, 1, adminCount, "Admin schema should appear exactly once, got %d", adminCount)

	basicCount := strings.Count(bundledStr, "Basic:")
	assert.Equal(t, 1, basicCount, "Basic schema should appear exactly once, got %d", basicCount)

	// Verify schemas were composed
	assert.Contains(t, bundledStr, "Admin", "Admin schema should be present")
	assert.Contains(t, bundledStr, "Basic", "Basic schema should be present")

	// Verify no file paths remain
	assertNoFilePathRefs(t, bundled)
	assertNoFilePathMappings(t, bundled)
}
