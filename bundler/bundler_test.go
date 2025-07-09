// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package bundler

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestBundleDocument_DigitalOcean(t *testing.T) {
	// test the mother of all exploded specs.
	tmp, _ := os.MkdirTemp("", "openapi")
	cmd := exec.Command("git", "clone", "https://github.com/digitalocean/openapi", tmp)
	defer os.RemoveAll(filepath.Join(tmp, "openapi"))

	err := cmd.Run()
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}

	spec, _ := filepath.Abs(filepath.Join(tmp+"/specification", "DigitalOcean-public.v2.yaml"))
	digi, _ := os.ReadFile(spec)

	doc, err := libopenapi.NewDocumentWithConfiguration([]byte(digi), &datamodel.DocumentConfiguration{
		SpecFilePath:            spec,
		BasePath:                tmp + "/specification",
		ExtractRefsSequentially: true,
		Logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelWarn,
		})),
	})
	if err != nil {
		panic(err)
	}

	v3Doc, errs := doc.BuildV3Model()
	if len(errs) > 0 {
		panic(errs)
	}

	bytes, e := BundleDocument(&v3Doc.Model)

	assert.NoError(t, e)
	assert.False(t, strings.Contains("$ref", string(bytes)), "should not contain $ref")
}

func TestBundleDocument_Circular(t *testing.T) {
	digi, _ := os.ReadFile("../test_specs/circular-tests.yaml")

	var logs []byte
	byteBuf := bytes.NewBuffer(logs)

	config := &datamodel.DocumentConfiguration{
		ExtractRefsSequentially: true,
		Logger: slog.New(slog.NewJSONHandler(byteBuf, &slog.HandlerOptions{
			Level: slog.LevelWarn,
		})),
	}
	doc, err := libopenapi.NewDocumentWithConfiguration(digi, config)
	if err != nil {
		panic(err)
	}

	v3Doc, errs := doc.BuildV3Model()

	// three circular ref issues.
	assert.Len(t, errs, 3)

	bytes, e := BundleDocument(&v3Doc.Model)
	assert.NoError(t, e)
	if runtime.GOOS != "windows" {
		assert.Len(t, *doc.GetSpecInfo().SpecBytes, 1563)
	} else {
		assert.Len(t, *doc.GetSpecInfo().SpecBytes, 1637)
	}
	assert.Len(t, bytes, 2016)

	logEntries := strings.Split(byteBuf.String(), "\n")
	if len(logEntries) == 1 && logEntries[0] == "" {
		logEntries = []string{}
	}

	assert.Len(t, logEntries, 0)
}

func TestBundleDocument_MinimalRemoteRefsBundledLocally(t *testing.T) {
	specBytes, err := os.ReadFile("../test_specs/minimal_remote_refs/openapi.yaml")
	require.NoError(t, err)

	require.NoError(t, err)

	config := &datamodel.DocumentConfiguration{
		AllowFileReferences:   true,
		AllowRemoteReferences: false,
		BundleInlineRefs:      false,
		BasePath:              "../test_specs/minimal_remote_refs",
		BaseURL:               nil,
	}
	require.NoError(t, err)

	bytes, e := BundleBytes(specBytes, config)
	assert.NoError(t, e)
	assert.Contains(t, string(bytes), "Name of the account", "should contain all reference targets")
}

func TestBundleDocument_MinimalRemoteRefsBundledRemotely(t *testing.T) {
	baseURL, err := url.Parse("https://raw.githubusercontent.com/felixjung/libopenapi/authed-remote/test_specs/minimal_remote_refs")

	refBytes, err := os.ReadFile("../test_specs/minimal_remote_refs/schemas/components.openapi.yaml")
	require.NoError(t, err)

	wantURL := fmt.Sprintf("%s/%s", baseURL.String(), "schemas/components.openapi.yaml")

	newRemoteHandlerFunc := func() utils.RemoteURLHandler {
		handler := func(w http.ResponseWriter, r *http.Request) {
			if r.URL.String() != wantURL {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			w.Write(refBytes)
		}

		return func(url string) (*http.Response, error) {
			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()
			handler(w, req)

			return w.Result(), nil
		}
	}

	specBytes, err := os.ReadFile("../test_specs/minimal_remote_refs/openapi.yaml")
	require.NoError(t, err)

	require.NoError(t, err)

	config := &datamodel.DocumentConfiguration{
		BaseURL:               baseURL,
		AllowFileReferences:   false,
		AllowRemoteReferences: true,
		BundleInlineRefs:      false,
		RemoteURLHandler:      newRemoteHandlerFunc(),
	}
	require.NoError(t, err)

	bytes, e := BundleBytes(specBytes, config)
	assert.NoError(t, e)
	assert.Contains(t, string(bytes), "Name of the account", "should contain all reference targets")
}

func TestBundleBytes(t *testing.T) {
	digi, _ := os.ReadFile("../test_specs/circular-tests.yaml")

	var logs []byte
	byteBuf := bytes.NewBuffer(logs)

	config := &datamodel.DocumentConfiguration{
		ExtractRefsSequentially: true,
		Logger: slog.New(slog.NewJSONHandler(byteBuf, &slog.HandlerOptions{
			Level: slog.LevelWarn,
		})),
	}

	bytes, e := BundleBytes(digi, config)
	assert.Error(t, e)
	assert.Len(t, bytes, 2016)

	logEntries := strings.Split(byteBuf.String(), "\n")
	if len(logEntries) == 1 && logEntries[0] == "" {
		logEntries = []string{}
	}

	assert.Len(t, logEntries, 0)
}

func TestBundleBytes_Invalid(t *testing.T) {
	digi := []byte(`openapi: 3.1.0
components:
  schemas:
    toto:
      $ref: bork`)

	var logs []byte
	byteBuf := bytes.NewBuffer(logs)

	config := &datamodel.DocumentConfiguration{
		ExtractRefsSequentially: true,
		Logger: slog.New(slog.NewJSONHandler(byteBuf, &slog.HandlerOptions{
			Level: slog.LevelWarn,
		})),
	}

	_, e := BundleBytes(digi, config)
	require.Error(t, e)
	unwrap := utils.UnwrapErrors(e)
	require.Len(t, unwrap, 2)
	assert.ErrorIs(t, unwrap[0], ErrInvalidModel)
	unwrapNext := utils.UnwrapErrors(unwrap[1])
	require.Len(t, unwrapNext, 2)
	assert.Equal(t, "component `bork` does not exist in the specification", unwrapNext[0].Error())
	assert.Equal(t, "cannot resolve reference `bork`, it's missing: $.bork [5:7]", unwrapNext[1].Error())

	logEntries := strings.Split(byteBuf.String(), "\n")
	if len(logEntries) == 1 && logEntries[0] == "" {
		logEntries = []string{}
	}

	assert.Len(t, logEntries, 0)
}

func TestBundleBytes_CircularArray(t *testing.T) {
	digi := []byte(`openapi: 3.1.0
info:
  title: FailureCases
  version: 0.1.0
servers:
  - url: http://localhost:35123
    description: The default server.
paths:
  /test:
    get:
      responses:
        '200':
          description: OK
components:
  schemas:
    Obj:
      type: object
      properties:
        children:
          type: array
          items:
            $ref: '#/components/schemas/Obj'
      required:
        - children`)

	var logs []byte
	byteBuf := bytes.NewBuffer(logs)

	config := &datamodel.DocumentConfiguration{
		ExtractRefsSequentially:       true,
		IgnoreArrayCircularReferences: true,
		Logger: slog.New(slog.NewJSONHandler(byteBuf, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})),
	}

	bytes, e := BundleBytes(digi, config)
	assert.NoError(t, e)
	assert.Len(t, bytes, 537)

	logEntries := strings.Split(byteBuf.String(), "\n")
	assert.Len(t, logEntries, 10)
}

func TestBundleBytes_CircularFile(t *testing.T) {
	digi := []byte(`openapi: 3.1.0
info:
  title: FailureCases
  version: 0.1.0
servers:
  - url: http://localhost:35123
    description: The default server.
paths:
  /test:
    get:
      responses:
        '200':
          description: OK
components:
  schemas:
    Obj:
      type: object
      properties:
        children:
          $ref: '../test_specs/circular-tests.yaml#/components/schemas/One'`)

	var logs []byte
	byteBuf := bytes.NewBuffer(logs)

	config := &datamodel.DocumentConfiguration{
		BasePath:                ".",
		ExtractRefsSequentially: true,
		Logger: slog.New(slog.NewJSONHandler(byteBuf, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})),
	}

	bytes, e := BundleBytes(digi, config)
	assert.Error(t, e)
	assert.Len(t, bytes, 458)

	logEntries := strings.Split(byteBuf.String(), "\n")
	assert.Len(t, logEntries, 13)
}

func TestBundleBytes_Bad(t *testing.T) {
	bytes, e := BundleBytes(nil, nil)
	assert.Error(t, e)
	assert.Nil(t, bytes)
}

func TestBundleBytes_RootDocumentRefs(t *testing.T) {
	spec, err := os.ReadFile("../test_specs/ref-followed.yaml")
	assert.NoError(t, err)

	{ // Making sure indentation is identical
		doc, err := libopenapi.NewDocument(spec)
		assert.NoError(t, err)

		v3Doc, errs := doc.BuildV3Model()
		assert.NoError(t, errors.Join(errs...))

		spec, err = v3Doc.Model.Render()
		assert.NoError(t, err)
	}

	config := &datamodel.DocumentConfiguration{
		BasePath:                ".",
		ExtractRefsSequentially: true,
	}

	bundledSpec, err := BundleBytes(spec, config)
	assert.NoError(t, err)

	assert.Equal(t, string(spec), string(bundledSpec))
}

func TestBundleDocument_BundleBytesComposed_NestedFiles(t *testing.T) {
	specBytes, _ := os.ReadFile("../test_specs/nested_files/openapi.yaml")

	config := &datamodel.DocumentConfiguration{
		AllowFileReferences:     true,
		BasePath:                "../test_specs/nested_files",
		ExtractRefsSequentially: true,
	}

	bundledBytes, e := BundleBytesComposed(specBytes, config, nil)

	assert.NoError(t, e)

	if runtime.GOOS != "windows" {

		preBundled, _ := os.ReadFile("../test_specs/nested_files/openapi-bundled.yaml")

		len1 := len(preBundled)
		len2 := len(bundledBytes)
		assert.Equal(t, len1, len2)

		// hash the two files and ensure they match
		hash1 := low.HashToString(sha256.Sum256(preBundled))
		hash2 := low.HashToString(sha256.Sum256(bundledBytes))
		assert.Equal(t, hash1, hash2)
	}
}

func TestBundleDocument_BundleBytesComposed_ErrorDoc(t *testing.T) {
	specBytes := []byte(`borked`)

	_, e := BundleBytesComposed(specBytes, nil, nil)

	assert.Error(t, e)
}

func TestBundleDocument_BundleBytesComposed_ErrorModel(t *testing.T) {
	specBytes := []byte(`openapi: 3.1.0
paths:
  /cake:
    $ref: '#/components/schemas/Cake'`)

	_, e := BundleBytesComposed(specBytes, nil, nil)

	assert.Error(t, e)
}

func TestDiscriminatorMappings_PropertyNameOnly(t *testing.T) {
	// Test discriminator with propertyName only (no mapping)
	specBytes := []byte(`openapi: "3.1.0"
info:
  title: PropertyName Only Test
  version: "1.0.0"
paths:
  /vehicles:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/Vehicle"
      responses:
        '200':
          description: Success
components:
  schemas:
    Vehicle:
      oneOf:
        - $ref: "#/components/schemas/Car"
        - $ref: "#/components/schemas/Truck"
      discriminator:
        propertyName: vehicleType
    Car:
      type: object
      properties:
        vehicleType:
          type: string
          enum: [Car]
    Truck:
      type: object
      properties:
        vehicleType:
          type: string
          enum: [Truck]
        capacity:
          type: integer`)

	config := &datamodel.DocumentConfiguration{
		BasePath:                "test/specs",
		ExtractRefsSequentially: true,
	}

	// Test both bundling modes
	bundledBytes, err := BundleBytes(specBytes, config)
	assert.NoError(t, err)
	assert.Contains(t, string(bundledBytes), "discriminator:")
	assert.Contains(t, string(bundledBytes), "propertyName:")

	composedBytes, err := BundleBytesComposed(specBytes, config, &BundleCompositionConfig{Delimiter: "__"})
	assert.NoError(t, err)
	assert.Contains(t, string(composedBytes), "discriminator:")
	assert.Contains(t, string(composedBytes), "propertyName:")
}

func TestDiscriminatorMappings_ExternalFileReferences(t *testing.T) {
	// Test discriminator with mapping to external files
	specBytes := []byte(`openapi: "3.1.0"
info:
  title: External File References Test
  version: "1.0.0"
paths:
  /vehicles:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/Vehicle"
      responses:
        '200':
          description: Success
components:
  schemas:
    Vehicle:
      oneOf:
        - $ref: "vehicle.yaml"
      discriminator:
        propertyName: vehicleType
        mapping:
          vehicle: "vehicle.yaml"`)

	config := &datamodel.DocumentConfiguration{
		BasePath:                "test/specs",
		ExtractRefsSequentially: true,
	}

	// Test inline bundling
	bundledBytes, err := BundleBytes(specBytes, config)
	assert.NoError(t, err)
	bundledSpec := string(bundledBytes)
	assert.Contains(t, bundledSpec, "discriminator:")
	assert.Contains(t, bundledSpec, "wheels:") // from vehicle.yaml

	// Test composition bundling
	composedBytes, err := BundleBytesComposed(specBytes, config, &BundleCompositionConfig{Delimiter: "__"})
	assert.NoError(t, err)
	composedSpec := string(composedBytes)
	assert.Contains(t, composedSpec, "discriminator:")
	assert.Contains(t, composedSpec, "components:")
	assert.Contains(t, composedSpec, "schemas:")
}

func TestDiscriminatorMappings_MixedLocalExternal(t *testing.T) {
	// Test discriminator with mixed local and external mappings
	specBytes := []byte(`openapi: "3.1.0"
info:
  title: Mixed References Test
  version: "1.0.0"
paths:
  /vehicles:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/Vehicle"
      responses:
        '200':
          description: Success
components:
  schemas:
    Vehicle:
      oneOf:
        - $ref: "#/components/schemas/Car"
        - $ref: "vehicle.yaml"
      discriminator:
        propertyName: vehicleType
        mapping:
          car: "#/components/schemas/Car"
          vehicle: "vehicle.yaml"
    Car:
      type: object
      properties:
        vehicleType:
          type: string
          enum: [car]
        doors:
          type: integer`)

	config := &datamodel.DocumentConfiguration{
		BasePath:                "test/specs",
		ExtractRefsSequentially: true,
	}

	// Test inline bundling
	bundledBytes, err := BundleBytes(specBytes, config)
	assert.NoError(t, err)
	bundledSpec := string(bundledBytes)
	assert.Contains(t, bundledSpec, "discriminator:")
	assert.Contains(t, bundledSpec, "doors:")  // from local Car schema
	assert.Contains(t, bundledSpec, "wheels:") // from external vehicle.yaml

	// Test composition bundling
	composedBytes, err := BundleBytesComposed(specBytes, config, &BundleCompositionConfig{Delimiter: "__"})
	assert.NoError(t, err)
	composedSpec := string(composedBytes)
	assert.Contains(t, composedSpec, "discriminator:")
	assert.Contains(t, composedSpec, "components:")
	assert.Contains(t, composedSpec, "schemas:")
}

func TestDiscriminatorMappings_ExternalWithFragment(t *testing.T) {
	// Test discriminator with external file references that include fragments
	// First create a vehicle file with proper structure
	vehicleWithFragment := []byte(`
Vehicle:
  type: object
  properties:
    vehicleType:
      type: string
    wheels:
      type: integer
  required:
    - vehicleType`)

	// Write to temp file
	tempFile, err := os.CreateTemp("test/specs", "vehicle-fragment-*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	_, err = tempFile.Write(vehicleWithFragment)
	assert.NoError(t, err)
	tempFile.Close()

	fileName := filepath.Base(tempFile.Name())

	specBytes := []byte(`openapi: "3.1.0"
info:
  title: External Fragment Test
  version: "1.0.0"
paths:
  /vehicles:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/Vehicle"
      responses:
        '200':
          description: Success
components:
  schemas:
    Vehicle:
      oneOf:
        - $ref: "` + fileName + `#/Vehicle"
      discriminator:
        propertyName: vehicleType
        mapping:
          vehicle: "` + fileName + `#/Vehicle"`)

	config := &datamodel.DocumentConfiguration{
		BasePath:                "test/specs",
		ExtractRefsSequentially: true,
	}

	// Test inline bundling
	bundledBytes, err := BundleBytes(specBytes, config)
	assert.NoError(t, err)
	bundledSpec := string(bundledBytes)
	assert.Contains(t, bundledSpec, "discriminator:")

	// Test composition bundling
	composedBytes, err := BundleBytesComposed(specBytes, config, &BundleCompositionConfig{Delimiter: "__"})
	assert.NoError(t, err)
	composedSpec := string(composedBytes)
	assert.Contains(t, composedSpec, "discriminator:")
	assert.Contains(t, composedSpec, "components:")
}

func TestDiscriminatorMappings_EdgeCases(t *testing.T) {
	// Test edge cases like empty mappings, non-existent references, etc.
	specBytes := []byte(`openapi: "3.1.0"
info:
  title: Edge Cases Test
  version: "1.0.0"
paths:
  /vehicles:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/Vehicle"
      responses:
        '200':
          description: Success
components:
  schemas:
    Vehicle:
      oneOf:
        - $ref: "#/components/schemas/Car"
      discriminator:
        propertyName: vehicleType
        mapping:
          car: "#/components/schemas/Car"
          nonexistent: "#/components/schemas/NonExistent"
          empty: ""
    Car:
      type: object
      properties:
        vehicleType:
          type: string
          enum: [car]
        doors:
          type: integer`)

	config := &datamodel.DocumentConfiguration{
		BasePath:                "test/specs",
		ExtractRefsSequentially: true,
	}

	// Should not error even with problematic mappings
	bundledBytes, err := BundleBytes(specBytes, config)
	assert.NoError(t, err)
	bundledSpec := string(bundledBytes)
	assert.Contains(t, bundledSpec, "discriminator:")
	assert.Contains(t, bundledSpec, "doors:")

	composedBytes, err := BundleBytesComposed(specBytes, config, &BundleCompositionConfig{Delimiter: "__"})
	assert.NoError(t, err)
	composedSpec := string(composedBytes)
	assert.Contains(t, composedSpec, "discriminator:")
}

func TestDiscoverDiscriminatorMappings(t *testing.T) {
	specBytes := []byte(`openapi: "3.1.0"
info:
  title: Discovery Test
  version: "1.0.0"
components:
  schemas:
    Vehicle:
      oneOf:
        - $ref: "#/components/schemas/Car"
        - $ref: "#/components/schemas/Truck"
      discriminator:
        propertyName: vehicleType
        mapping:
          car: "#/components/schemas/Car"
          truck: "#/components/schemas/Truck"
          external: "vehicle.yaml"
    Car:
      type: object
    Truck:
      type: object`)

	doc, err := libopenapi.NewDocumentWithConfiguration(specBytes, &datamodel.DocumentConfiguration{
		BasePath: "test/specs",
	})
	assert.NoError(t, err)

	_, errs := doc.BuildV3Model()
	assert.NoError(t, errors.Join(errs...))
}

func TestDiscriminatorMappings_BugFix_Integration(t *testing.T) {
	// This test verifies the original discriminator mapping bug is fixed
	specBytes := []byte(`openapi: "3.1.0"
info:
  title: Bug Fix Test
  version: "1.0.0"
paths:
  /vehicles:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/Vehicle"
      responses:
        '200':
          description: Success
components:
  schemas:
    Vehicle:
      oneOf:
        - $ref: "vehicle.yaml"
      discriminator:
        propertyName: vehicleType
        mapping:
          vehicle: "vehicle.yaml"`)

	config := &datamodel.DocumentConfiguration{
		BasePath:                "test/specs",
		ExtractRefsSequentially: true,
	}

	// Test composition bundling - mappings should be updated to point to components
	bundledBytes, err := BundleBytesComposed(specBytes, config, &BundleCompositionConfig{Delimiter: "__"})
	assert.NoError(t, err)

	bundledSpec := string(bundledBytes)

	// Verify external schemas were composed into components
	assert.Contains(t, bundledSpec, "components:")
	assert.Contains(t, bundledSpec, "schemas:")
	assert.Contains(t, bundledSpec, "wheels:") // from vehicle.yaml

	// Verify discriminator mappings were updated to point to components (bug fix)
	assert.Contains(t, bundledSpec, "#/components/schemas/")

	// The original bug would leave mappings pointing to "vehicle.yaml"
	// which would be invalid after bundling. Our fix updates them to proper component refs.

	// Test inline bundling - mappings should be cleared since schemas are inlined
	inlineBundledBytes, err := BundleBytes(specBytes, config)
	assert.NoError(t, err)

	inlineBundledSpec := string(inlineBundledBytes)

	// Verify schemas were inlined
	assert.Contains(t, inlineBundledSpec, "wheels:") // from vehicle.yaml inlined

	// Verify discriminator mapping structure is preserved
	assert.Contains(t, inlineBundledSpec, "discriminator:")
	assert.Contains(t, inlineBundledSpec, "mapping:")
}

// TestDiscriminatorMappingBugDemo demonstrates the actual bug where discriminator mappings
// point to invalid locations after bundling without our fix
func TestDiscriminatorMappingBugDemo(t *testing.T) {
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
    # Let's also add a proper $ref to the external file to make sure it gets bundled
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
		BasePath:                tempDir,
		AllowFileReferences:     true,
		AllowRemoteReferences:   false,
		ExtractRefsSequentially: true,
	}

	// Bundle the document
	bundled, err := BundleBytes(mainBytes, config)
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

	// This is the key test - after bundling, the external reference should be updated or removed
	// Without our fix, this would still point to './external-cat.yaml#/Cat' which doesn't exist in the bundled spec
	catMapping, ok := mapping["cat"].(string)
	require.True(t, ok, "cat mapping should exist")

	// With our fix for inline bundling, the mapping should point to the component location
	// where the external schema was placed, not be empty or point to an invalid external file
	assert.Equal(t, "#/components/schemas/ExternalCat", catMapping, "cat mapping should point to the component location after inline bundling")

	// Also verify that the ExternalCat schema was actually bundled
	_, externalCatExists := schemas["ExternalCat"]
	assert.True(t, externalCatExists, "ExternalCat schema should be bundled via $ref")
}

// TestDiscriminatorMappingBugDemoComposed demonstrates the bug for composed bundling
func TestDiscriminatorMappingBugDemoComposed(t *testing.T) {
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

	// This is the key test - after composed bundling, the external reference should point to the components
	// Without our fix, this would still point to './external-cat.yaml#/Cat' which doesn't exist in the bundled spec
	catMapping, ok := mapping["cat"].(string)
	require.True(t, ok, "cat mapping should exist")

	// With our fix for composed bundling, the mapping should point to the components section
	// The external schema is placed as ExternalCat so the discriminator mapping should point to the correct location
	assert.Equal(t, "#/components/schemas/ExternalCat", catMapping, "cat mapping should point to the correct component location")

	t.Logf("Discriminator cat mapping: '%s'", catMapping)
	t.Logf("Expected: '#/components/schemas/ExternalCat'")
	t.Logf("Current behavior: '%s'", catMapping)

	// Also verify that the ExternalCat schema was actually bundled
	_, externalCatExists := schemas["ExternalCat"]
	assert.True(t, externalCatExists, "ExternalCat schema should be bundled into the main document")

	// Log the actual structure to understand what's happening
	t.Logf("Components schemas keys: %v", func() []string {
		keys := make([]string, 0)
		for k := range schemas {
			keys = append(keys, k)
		}
		return keys
	}())
}

// TestDiscriminatorMappingInlineReality tests what actually happens to discriminators with pure inlining
func TestDiscriminatorMappingInlineReality(t *testing.T) {
	// Create a spec that actually uses the discriminator mapping in a reference
	spec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: A list of pets
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Animal'
components:
  schemas:
    Animal:
      type: object
      discriminator:
        propertyName: type
        mapping:
          dog: '#/components/schemas/Dog'
          cat: './external-cat.yaml#/Cat'
      oneOf:
        - $ref: '#/components/schemas/Dog'
        - $ref: './external-cat.yaml#/Cat'
    Dog:
      type: object
      properties:
        type:
          type: string
        bark:
          type: boolean
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
		BasePath:                tempDir,
		AllowFileReferences:     true,
		AllowRemoteReferences:   false,
		ExtractRefsSequentially: true,
	}

	// Bundle the document (inline bundling)
	bundled, err := BundleBytes(mainBytes, config)
	require.NoError(t, err)

	t.Logf("Bundled result:\n%s", string(bundled))

	// Parse the bundled result
	var bundledSpec map[string]interface{}
	err = yaml.Unmarshal(bundled, &bundledSpec)
	require.NoError(t, err)

	// Check what happened to the discriminator
	paths := bundledSpec["paths"].(map[string]interface{})
	getPets := paths["/pets"].(map[string]interface{})
	getMethod := getPets["get"].(map[string]interface{})
	responses := getMethod["responses"].(map[string]interface{})
	response200 := responses["200"].(map[string]interface{})
	content := response200["content"].(map[string]interface{})
	appJson := content["application/json"].(map[string]interface{})
	schema := appJson["schema"].(map[string]interface{})
	items := schema["items"].(map[string]interface{})

	t.Logf("Items schema: %v", items)

	// Check if discriminator still exists and is valid
	if discriminator, exists := items["discriminator"]; exists {
		t.Logf("Discriminator still exists: %v", discriminator)

		if disc, ok := discriminator.(map[string]interface{}); ok {
			if mapping, exists := disc["mapping"]; exists {
				t.Logf("Mapping still exists: %v", mapping)
			}
		}
	}
}
