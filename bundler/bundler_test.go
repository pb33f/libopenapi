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

// TestDiscriminatorMapping tests whether discriminator mappings
// are correctly bundled when they reference external files.
func TestDiscriminatorMapping(t *testing.T) {
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

	// after bundling, the external reference should be updated or removed
	catMapping, ok := mapping["cat"].(string)
	require.True(t, ok, "cat mapping should exist")

	// the mapping should point to the component location where the external schema was placed, not be empty or point to an invalid external file
	assert.Equal(t, "#/components/schemas/ExternalCat", catMapping, "cat mapping should point to the component location after inline bundling")

	// verify that the ExternalCat schema was actually bundled
	_, externalCatExists := schemas["ExternalCat"]
	assert.True(t, externalCatExists, "ExternalCat schema should be bundled via $ref")

	// Force garbage collection to close any open file handles (Windows fix)
	runtime.GC()
}

// TestDiscriminatorMappingNonExistent tests that discriminator mappings pointing to non-existent files are left unchanged
func TestDiscriminatorMappingNonExistent(t *testing.T) {
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
    # Add a proper $ref to the external file so it gets bundled
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
	assert.True(t, externalCatExists, "ExternalCat schema should be bundled")

	// Force garbage collection to close any open file handles (Windows fix)
	runtime.GC()
}
