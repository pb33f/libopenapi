// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package bundler

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestBundlerComposed(t *testing.T) {
	specBytes, err := os.ReadFile("test/specs/main.yaml")

	doc, err := libopenapi.NewDocumentWithConfiguration(specBytes, &datamodel.DocumentConfiguration{
		BasePath:                "test/specs",
		ExtractRefsSequentially: true,
		Logger:                  slog.Default(),
	})
	if err != nil {
		panic(err)
	}

	v3Doc, errs := doc.BuildV3Model()
	if len(errs) > 0 {
		panic(errs)
	}

	var bytes []byte

	bytes, err = BundleDocumentComposed(&v3Doc.Model, &BundleCompositionConfig{Delimiter: "__"})
	if err != nil {
		panic(err)
	}

	// windows needs a different byte count
	if runtime.GOOS != "windows" {
		assert.Len(t, bytes, 9099)
	}

	preBundled, bErr := os.ReadFile("test/specs/bundled.yaml")
	assert.NoError(t, bErr)

	if runtime.GOOS != "windows" {
		assert.Equal(t, len(preBundled), len(bytes)) // windows reads the file with different line endings and changes the byte count.
	}

	// write the bundled spec to a file for inspection
	// uncomment this to rebuild the bundled spec file, if the example spec changes.
	// err = os.WriteFile("test/specs/bundled.yaml", bytes, 0644)

	v3Doc.Model.Components = nil
	err = processReference(&v3Doc.Model, &processRef{}, &handleIndexConfig{compositionConfig: &BundleCompositionConfig{}})
	assert.Error(t, err)
}

func TestCheckFileIteration(t *testing.T) {
	name := calculateCollisionName("bundled", "/test/specs/bundled.yaml", "__", 1)
	assert.Equal(t, "bundled__specs", name)

	name = calculateCollisionName("bundled__specs", "/test/specs/bundled.yaml", "__", 2)
	assert.Equal(t, "bundled__specs__test", name)

	name = calculateCollisionName("bundled-||-specs", "/test/specs/bundled.yaml", "-||-", 2)
	assert.Equal(t, "bundled-||-specs-||-test", name)

	reg := regexp.MustCompile("^bundled__[0-9A-Za-z]{1,4}$")

	name = calculateCollisionName("bundled", "/test/specs/bundled.yaml", "__", 8)
	assert.True(t, reg.MatchString(name))
}

func TestBundleDocumentComposed(t *testing.T) {
	_, err := BundleDocumentComposed(nil, nil)
	assert.Error(t, err)
	assert.Equal(t, "model or rolodex is nil", err.Error())

	_, err = BundleDocumentComposed(nil, &BundleCompositionConfig{Delimiter: ""})
	assert.Error(t, err)
	assert.Equal(t, "model or rolodex is nil", err.Error())

	_, err = BundleDocumentComposed(nil, &BundleCompositionConfig{Delimiter: "#"})
	assert.Error(t, err)
	assert.Equal(t, "composition delimiter cannot contain '#' or '/' characters", err.Error())

	_, err = BundleDocumentComposed(nil, &BundleCompositionConfig{Delimiter: "well hello there"})
	assert.Error(t, err)
	assert.Equal(t, "composition delimiter cannot contain spaces", err.Error())
}

func TestCheckReferenceAndBubbleUp(t *testing.T) {
	err := checkReferenceAndBubbleUp[any]("test", "__",
		&processRef{ref: &index.Reference{Node: &yaml.Node{}}},
		nil, nil,
		func(node *yaml.Node, idx *index.SpecIndex) (any, error) {
			return nil, errors.New("test error")
		})
	assert.Error(t, err)
}

func TestRenameReference(t *testing.T) {
	// test the rename reference function
	assert.Equal(t, "#/_oh_#/_yeah", renameRef(nil, "#/_oh_#/_yeah", nil))
}

func TestBuildSchema(t *testing.T) {
	_, err := buildSchema(nil, nil)
	assert.Error(t, err)
}

func TestBundlerComposed_StrangeRefs(t *testing.T) {
	specBytes, err := os.ReadFile("../test_specs/first.yaml")

	doc, err := libopenapi.NewDocumentWithConfiguration(specBytes, &datamodel.DocumentConfiguration{
		BasePath:                "../test_specs/",
		ExtractRefsSequentially: true,
		Logger:                  slog.Default(),
	})
	if err != nil {
		panic(err)
	}

	v3Doc, errs := doc.BuildV3Model()
	if len(errs) > 0 {
		panic(errs)
	}

	var bytes []byte

	bytes, err = BundleDocumentComposed(&v3Doc.Model, &BundleCompositionConfig{Delimiter: "__"})
	if err != nil {
		panic(err)
	}

	// windows needs a different byte count
	if runtime.GOOS != "windows" {
		assert.Len(t, bytes, 3397)
	}
}

// TestDiscriminatorMappingComposed tests discriminator mapping with composed bundling.
func TestDiscriminatorMappingComposed(t *testing.T) {
	// Create a spec with external reference in discriminator mapping
	spec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Animal:
      type: object
      discriminator:
        propertyName: type
        mapping:
          dog: '#/components/schemas/Dog'
          cat: './external-cat.yaml#/Cat'
    Dog:
      type: object
      properties:
        type:
          type: string
        bark:
          type: boolean
    # Add a proper $ref to the external file so it gets processed by composed bundling
    ExternalCat:
      $ref: './external-cat.yaml#/Cat'
`

	// Create external file
	externalSpec := `Cat:
  type: object
  properties:
    type:
      type: string
    meow:
      type: boolean`

	// Create temporary files
	tempDir := t.TempDir()
	mainFile := filepath.Join(tempDir, "main.yaml")
	externalFile := filepath.Join(tempDir, "external-cat.yaml")

	err := os.WriteFile(mainFile, []byte(spec), 0644)
	require.NoError(t, err)

	err = os.WriteFile(externalFile, []byte(externalSpec), 0644)
	require.NoError(t, err)

	// Load the document
	mainBytes, err := os.ReadFile(mainFile)
	require.NoError(t, err)

	config := &datamodel.DocumentConfiguration{
		BasePath: tempDir,
	}

	// Bundle the document using composed bundling
	bundled, err := BundleBytesComposed(mainBytes, config, &BundleCompositionConfig{
		Delimiter: "__",
	})
	require.NoError(t, err)

	// Parse the bundled result
	var bundledSpec map[string]interface{}
	err = yaml.Unmarshal(bundled, &bundledSpec)
	require.NoError(t, err)

	// Check the discriminator mapping in the bundled result
	components, ok := bundledSpec["components"].(map[string]interface{})
	require.True(t, ok, "components should exist")

	schemas, ok := components["schemas"].(map[string]interface{})
	require.True(t, ok, "schemas should exist")

	animal, ok := schemas["Animal"].(map[string]interface{})
	require.True(t, ok, "Animal schema should exist")

	discriminator, ok := animal["discriminator"].(map[string]interface{})
	require.True(t, ok, "discriminator should exist")

	mapping, ok := discriminator["mapping"].(map[string]interface{})
	require.True(t, ok, "mapping should exist")

	// after composed bundling, the external reference should point to the components
	catMapping, ok := mapping["cat"].(string)
	require.True(t, ok, "cat mapping should exist")

	// The external schema is placed as ExternalCat so the discriminator mapping should point to the correct location
	assert.Equal(t, "#/components/schemas/ExternalCat", catMapping, "cat mapping should point to the correct component location")

	t.Logf("Discriminator cat mapping: '%s'", catMapping)
	t.Logf("Expected: '#/components/schemas/ExternalCat'")
	t.Logf("Current behavior: '%s'", catMapping)

	// Also verify that the ExternalCat schema was actually bundled
	_, externalCatExists := schemas["ExternalCat"]
	assert.True(t, externalCatExists, "ExternalCat schema should be bundled into the main document")

	// Force garbage collection to close any open file handles (Windows fix)
	runtime.GC()
}

// TestDiscriminatorMappingNonExistentComposed tests that discriminator mappings pointing to non-existent files are left unchanged during composed bundling
func TestDiscriminatorMappingNonExistentComposed(t *testing.T) {
	spec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Animal:
      type: object
      discriminator:
        propertyName: type
        mapping:
          dog: '#/components/schemas/Dog'
          cat: './external-cat.yaml#/Cat'
          nonexistent: './does-not-exist.yaml#/Missing'
          invalid: './also-missing.yaml#/Schema'
    Dog:
      type: object
      properties:
        type:
          type: string
        bark:
          type: boolean
    # Add a proper $ref to the external file so it gets processed by composed bundling
    ExternalCat:
      $ref: './external-cat.yaml#/Cat'
`

	// Create external file (but not the non-existent ones)
	externalSpec := `Cat:
  type: object
  properties:
    type:
      type: string
    meow:
      type: boolean`

	// Create temporary files
	tempDir := t.TempDir()
	mainFile := filepath.Join(tempDir, "main.yaml")
	externalFile := filepath.Join(tempDir, "external-cat.yaml")

	err := os.WriteFile(mainFile, []byte(spec), 0644)
	require.NoError(t, err)

	err = os.WriteFile(externalFile, []byte(externalSpec), 0644)
	require.NoError(t, err)

	// Load the document
	mainBytes, err := os.ReadFile(mainFile)
	require.NoError(t, err)

	config := &datamodel.DocumentConfiguration{
		BasePath: tempDir,
	}

	// Bundle the document using composed bundling
	bundled, err := BundleBytesComposed(mainBytes, config, &BundleCompositionConfig{
		Delimiter: "__",
	})
	require.NoError(t, err)

	// Parse the bundled result
	var bundledSpec map[string]interface{}
	err = yaml.Unmarshal(bundled, &bundledSpec)
	require.NoError(t, err)

	// Check the discriminator mapping in the bundled result
	components, ok := bundledSpec["components"].(map[string]interface{})
	require.True(t, ok, "components should exist")

	schemas, ok := components["schemas"].(map[string]interface{})
	require.True(t, ok, "schemas should exist")

	animal, ok := schemas["Animal"].(map[string]interface{})
	require.True(t, ok, "Animal schema should exist")

	discriminator, ok := animal["discriminator"].(map[string]interface{})
	require.True(t, ok, "discriminator should exist")

	mapping, ok := discriminator["mapping"].(map[string]interface{})
	require.True(t, ok, "mapping should exist")

	// Valid mappings should be updated
	dogMapping, ok := mapping["dog"].(string)
	require.True(t, ok, "dog mapping should exist")
	assert.Equal(t, "#/components/schemas/Dog", dogMapping, "dog mapping should point to component")

	catMapping, ok := mapping["cat"].(string)
	require.True(t, ok, "cat mapping should exist")
	assert.Equal(t, "#/components/schemas/ExternalCat", catMapping, "cat mapping should point to component")

	// Non-existent mappings should be left unchanged
	nonExistentMapping, ok := mapping["nonexistent"].(string)
	require.True(t, ok, "nonexistent mapping should still exist")
	assert.Equal(t, "./does-not-exist.yaml#/Missing", nonExistentMapping, "nonexistent mapping should be unchanged")

	invalidMapping, ok := mapping["invalid"].(string)
	require.True(t, ok, "invalid mapping should still exist")
	assert.Equal(t, "./also-missing.yaml#/Schema", invalidMapping, "invalid mapping should be unchanged")

	// Verify that the valid external schema was bundled
	_, externalCatExists := schemas["ExternalCat"]
	assert.True(t, externalCatExists, "ExternalCat schema should be bundled into the main document")

	// Force garbage collection to close any open file handles (Windows fix)
	runtime.GC()
}
