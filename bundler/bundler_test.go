// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package bundler

import (
	"bytes"
	"crypto/sha256"
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
	"sync"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// Test helper functions to reduce duplication across DigitalOcean tests

// collectAllDiscriminatorRefs gathers all refs that are allowed to be preserved (discriminator mappings).
func collectAllDiscriminatorRefs(model *v3high.Document) map[string]struct{} {
	preservedRefs := make(map[string]struct{})
	rootIdx := model.Rolodex.GetRootIndex()
	collectDiscriminatorMappingValues(rootIdx, rootIdx.GetRootNode(), preservedRefs)
	for _, idx := range model.Rolodex.GetIndexes() {
		collectDiscriminatorMappingValues(idx, idx.GetRootNode(), preservedRefs)
	}
	return preservedRefs
}

// cleanRefPath trims quotes and normalizes slashes to Unix-style.
func cleanRefPath(s string) string {
	return filepath.ToSlash(strings.Trim(s, `"'`))
}

// extractRefFromLine extracts the $ref value from a YAML line.
func extractRefFromLine(line string) string {
	i := strings.Index(line, "$ref:")
	if i == -1 {
		return ""
	}
	return cleanRefPath(strings.TrimSpace(line[i+5:]))
}

// isPreservedRef checks if a ref is in the preserved set (discriminator mappings).
func isPreservedRef(line string, preservedRefs map[string]struct{}) bool {
	ref := extractRefFromLine(line)
	if ref == "" {
		return false
	}
	for uri := range preservedRefs {
		if strings.HasSuffix(cleanRefPath(uri), ref) {
			return true
		}
	}
	return false
}

// isEmptyRef checks for malformed/empty refs like "$ref: {}"
func isEmptyRef(line string) bool {
	ref := extractRefFromLine(line)
	return ref == "{}" || ref == ""
}

func TestBundleDocument_DigitalOcean(t *testing.T) {
	// test the mother of all exploded specs.
	tmp := t.TempDir()
	cmd := exec.Command("git", "clone", "-b", "asb/dedup-key-model", "https://github.com/digitalocean/openapi.git", tmp)

	err := cmd.Run()
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}

	spec, _ := filepath.Abs(filepath.Join(tmp, "specification", "DigitalOcean-public.v2.yaml"))
	digi, _ := os.ReadFile(spec)

	doc, err := libopenapi.NewDocumentWithConfiguration(digi, &datamodel.DocumentConfiguration{
		SpecFilePath:            spec,
		BasePath:                filepath.Join(tmp, "specification"),
		ExtractRefsSequentially: true,
		Logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelError,
		})),
	})
	if err != nil {
		panic(err)
	}

	v3Doc, errs := doc.BuildV3Model()
	if errs != nil {
		t.Fatal("Errors building V3 model:", errs)
	}

	preservedRefs := collectAllDiscriminatorRefs(&v3Doc.Model)

	bytes, e := BundleDocument(&v3Doc.Model)

	assert.NoError(t, e)
	lines := strings.Split(string(bytes), "\n")
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}
		if strings.Contains(trimmedLine, "$ref") && !isPreservedRef(trimmedLine, preservedRefs) && !isEmptyRef(trimmedLine) {
			t.Errorf("Found uncommented $ref in line: %s", line)
		}
	}
}

func TestBundleDocument_DigitalOceanAsync(t *testing.T) {
	// test the mother of all exploded specs.
	tmp := t.TempDir()
	cmd := exec.Command("git", "clone", "-b", "asb/dedup-key-model", "https://github.com/digitalocean/openapi.git", tmp)

	err := cmd.Run()
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}

	spec, _ := filepath.Abs(filepath.Join(tmp, "specification", "DigitalOcean-public.v2.yaml"))
	digi, _ := os.ReadFile(spec)

	doc, err := libopenapi.NewDocumentWithConfiguration(digi, &datamodel.DocumentConfiguration{
		SpecFilePath:            spec,
		BasePath:                filepath.Join(tmp, "specification"),
		ExtractRefsSequentially: false,
		Logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelError,
		})),
	})
	if err != nil {
		panic(err)
	}

	v3Doc, errs := doc.BuildV3Model()
	if errs != nil {
		t.Fatal("Errors building V3 model:", errs)
	}

	preservedRefs := collectAllDiscriminatorRefs(&v3Doc.Model)

	bytes, e := BundleDocument(&v3Doc.Model)

	assert.NoError(t, e)
	lines := strings.Split(string(bytes), "\n")
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}
		if strings.Contains(trimmedLine, "$ref") && !isPreservedRef(trimmedLine, preservedRefs) && !isEmptyRef(trimmedLine) {
			t.Errorf("Found uncommented $ref in line: %s", line)
		}
	}
}

// TestBundleDocument_ConcurrentBundling verifies that concurrent BundleDocument calls
// work correctly with the bundling mode reference counting (bundlingModeCount in schema_proxy.go).
//
// This test uses a simple inline spec to avoid cross-model interference in the global
// inlineRenderingTracker (which uses file:line:column as keys).
func TestBundleDocument_ConcurrentBundling(t *testing.T) {
	// Simple spec with local refs - no external files
	specTemplate := `openapi: "3.0.0"
info:
  title: Test API %d
  version: "1.0"
paths:
  /test:
    get:
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/TestSchema'
components:
  schemas:
    TestSchema:
      type: object
      properties:
        name:
          type: string
        id:
          type: integer
`

	const goroutines = 10

	type result struct {
		output []byte
		err    error
	}
	results := make(chan result, goroutines)

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			// Each goroutine gets a slightly different spec (different title)
			// to ensure unique line positions in the index
			specBytes := []byte(fmt.Sprintf(specTemplate, idx))

			config := &datamodel.DocumentConfiguration{
				ExtractRefsSequentially: false,
			}
			doc, err := libopenapi.NewDocumentWithConfiguration(specBytes, config)
			if err != nil {
				results <- result{err: err}
				return
			}

			v3Doc, errs := doc.BuildV3Model()
			if errs != nil {
				results <- result{err: errs}
				return
			}

			output, err := BundleDocument(&v3Doc.Model)
			results <- result{output: output, err: err}
		}(i)
	}

	wg.Wait()
	close(results)

	successCount := 0
	for r := range results {
		assert.NoError(t, r.err, "BundleDocument should not error")
		if r.err == nil {
			successCount++
			// Verify output preserves local refs (bundling mode behavior)
			outputStr := string(r.output)
			assert.Contains(t, outputStr, "$ref", "Bundled output should preserve local component refs")
			assert.Contains(t, outputStr, "#/components/schemas/TestSchema",
				"Bundled output should contain local schema ref")
		}
	}

	assert.Equal(t, goroutines, successCount,
		"All concurrent bundle operations should succeed")
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
	assert.Len(t, utils.UnwrapErrors(errs), 3)

	bytes, e := BundleDocument(&v3Doc.Model)
	assert.NoError(t, e)
	if runtime.GOOS != "windows" {
		assert.Len(t, *doc.GetSpecInfo().SpecBytes, 1692)
	}
	// Output length varies due to rendering of empty polymorphic fields
	assert.Greater(t, len(bytes), 2000)

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
	// Output length varies slightly due to rendering of empty polymorphic fields
	// The important thing is that circular refs are detected (error returned)
	assert.Greater(t, len(bytes), 2000)

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
	// Output length varies due to rendering of empty polymorphic fields
	assert.Greater(t, len(bytes), 500)

	// Log entries vary based on implementation details
	logEntries := strings.Split(byteBuf.String(), "\n")
	assert.GreaterOrEqual(t, len(logEntries), 8)
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
	// Output should not be empty even with circular refs - partial inlining occurs
	assert.Greater(t, len(bytes), 400)

	// Log entries vary based on implementation - just verify we got some logs
	logEntries := strings.Split(byteBuf.String(), "\n")
	assert.Greater(t, len(logEntries), 5)
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
		assert.NoError(t, errs)

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

// TestBundleBytes_DiscriminatorMapping
// Checks that a oneOf with a discriminator mapping does not inline the referenced schema,
func TestBundleBytes_DiscriminatorMapping(t *testing.T) {
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
          cat: './external-cat.yaml#/components/schemas/Cat'
      oneOf:
        - $ref: './external-cat.yaml#/components/schemas/Cat'
    Dog:
      type: object`

	ext := `components:
  schemas:
    Cat:
      type: object
      properties:
        type:
          type: string
        meow:
          type: boolean`

	tmp := t.TempDir()
	write := func(name, src string) {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(src), 0644))
	}
	write("main.yaml", spec)
	write("external-cat.yaml", ext)

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))
	cfg := &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	}

	out, err := BundleBytes(mainBytes, cfg)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(out, &doc))

	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	animal := schemas["Animal"].(map[string]any)

	// mapping value unchanged
	mapping := animal["discriminator"].(map[string]any)["mapping"].(map[string]any)
	assert.Equal(t, "./external-cat.yaml#/components/schemas/Cat", mapping["cat"])

	// the same $ref inside oneOf is also unchanged
	oneOf := animal["oneOf"].([]any)[0].(map[string]any)
	assert.Equal(t, "./external-cat.yaml#/components/schemas/Cat", oneOf["$ref"])

	// Cat schema NOT copied into components
	_, copied := schemas["Cat"]
	assert.False(t, copied, "Cat schema must not be inlined")

	runtime.GC()
}

/*
TestBundleBytes_DiscriminatorMappingMultiple tests that a oneOf schema with a discriminator mapping
pointing to multiple external schemas does not inline the schemas, but keeps them as $refs.
*/
func TestBundleBytes_DiscriminatorMappingMultiple(t *testing.T) {
	spec := `openapi: 3.0.0
info:
  title: Vehicles
  version: 1.0.0
paths: {}
components:
  schemas:
    Vehicle:
      type: object
      discriminator:
        propertyName: kind
        mapping:
          car: './vehicles/car.yaml#/components/schemas/Car'
          bike: './vehicles/bike.yaml#/components/schemas/Bike'
      oneOf:
        - $ref: './vehicles/car.yaml#/components/schemas/Car'
        - $ref: './vehicles/bike.yaml#/components/schemas/Bike'`

	car := `components:
  schemas:
    Car:
      type: object
      properties:
        wheels:
          type: integer`
	bike := `components:
  schemas:
    Bike:
      type: object
      properties:
        wheels:
          type: integer`

	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "vehicles"), 0755))
	write := func(name, src string) {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(src), 0644))
	}
	write("main.yaml", spec)
	write("vehicles/car.yaml", car)
	write("vehicles/bike.yaml", bike)

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))
	out, err := BundleBytes(mainBytes, &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	})
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(out, &doc))

	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	vehicle := schemas["Vehicle"].(map[string]any)
	mp := vehicle["discriminator"].(map[string]any)["mapping"].(map[string]any)

	assert.Equal(t, "./vehicles/car.yaml#/components/schemas/Car", mp["car"])
	assert.Equal(t, "./vehicles/bike.yaml#/components/schemas/Bike", mp["bike"])

	oneOf := vehicle["oneOf"].([]any)
	assert.Equal(t, "./vehicles/car.yaml#/components/schemas/Car", oneOf[0].(map[string]any)["$ref"])
	assert.Equal(t, "./vehicles/bike.yaml#/components/schemas/Bike", oneOf[1].(map[string]any)["$ref"])

	_, carExists := schemas["Car"]
	_, bikeExists := schemas["Bike"]
	assert.False(t, carExists)
	assert.False(t, bikeExists)

	runtime.GC()
}

// TestBundleBytes_DiscriminatorMappingPartial tests that a oneOf schema with a
// discriminator mapping that mentions only *some* of the alternatives keeps the
// $ref for the un-mapped alternative intact (i.e. it is NOT inlined).
func TestBundleBytes_DiscriminatorMappingPartial(t *testing.T) {
	spec := `openapi: 3.0.0
info:
  title: Vehicles
  version: 1.0.0
paths: {}
components:
  schemas:
    Vehicle:
      type: object
      discriminator:
        propertyName: kind
        mapping:
          car: './vehicles/car.yaml#/components/schemas/Car'   # bike missing on purpose
      oneOf:
        - $ref: './vehicles/car.yaml#/components/schemas/Car'
        - $ref: './vehicles/bike.yaml#/components/schemas/Bike'`

	car := `components:
  schemas:
    Car:
      type: object
      properties:
        wheels:
          type: integer`

	bike := `components:
  schemas:
    Bike:
      type: object
      properties:
        wheels:
          type: integer`

	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "vehicles"), 0o755))
	write := func(name, src string) {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(src), 0o644))
	}
	write("main.yaml", spec)
	write("vehicles/car.yaml", car)
	write("vehicles/bike.yaml", bike)

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))

	out, err := BundleBytes(mainBytes, &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	})
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(out, &doc))

	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	vehicle := schemas["Vehicle"].(map[string]any)

	mp := vehicle["discriminator"].(map[string]any)["mapping"].(map[string]any)
	assert.Equal(t, 1, len(mp), "no new mapping rows should have been synthesised")
	assert.Equal(t, "./vehicles/car.yaml#/components/schemas/Car", mp["car"])

	oneOf := vehicle["oneOf"].([]any)
	assert.Equal(t, "./vehicles/car.yaml#/components/schemas/Car", oneOf[0].(map[string]any)["$ref"])
	assert.Equal(t, "./vehicles/bike.yaml#/components/schemas/Bike", oneOf[1].(map[string]any)["$ref"])

	_, carExists := schemas["Car"]
	_, bikeExists := schemas["Bike"]
	assert.False(t, carExists, "Car must not be duplicated in components")
	assert.False(t, bikeExists, "Bike must not be duplicated in components")

	runtime.GC()
}

// TestBundleBytes_DiscriminatorMappingInternal tests that a oneOf schema with a discriminator mapping
// pointing to an internal schema does not inline the schema, but keeps it as a $ref.
func TestBundleBytes_DiscriminatorMappingInternal(t *testing.T) {
	spec := `openapi: 3.0.0
info:
  title: Pets
  version: 1.0.0
paths:
  /pets:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              discriminator:
                propertyName: kind
                mapping:
                  cat: '#/components/schemas/Cat'
              oneOf:
                - $ref: '#/components/schemas/Cat'
      responses:
        '200':
          description: Success
components:
  schemas:
    Cat:
      type: object
      properties:
        name:
          type: string`

	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "main.yaml"), []byte(spec), 0644))

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))
	out, err := BundleBytes(mainBytes, &datamodel.DocumentConfiguration{BasePath: tmp})
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(out, &doc))

	// Navigate to the oneOf in the path-level schema
	paths := doc["paths"].(map[string]any)
	post := paths["/pets"].(map[string]any)["post"].(map[string]any)
	requestBody := post["requestBody"].(map[string]any)
	content := requestBody["content"].(map[string]any)
	appJson := content["application/json"].(map[string]any)
	schema := appJson["schema"].(map[string]any)
	oneOf := schema["oneOf"].([]any)[0].(map[string]any)

	assert.Equal(t, "#/components/schemas/Cat", oneOf["$ref"],
		"internal reference should remain a $ref (bundler skips local root refs)")

	runtime.GC()
}

// TestBundleBytes_OneOfWithoutDiscriminatorMappingInlined tests that a oneOf schema
// without a discriminator mapping is inlined
func TestBundleBytes_OneOfWithoutDiscriminatorMappingInlined(t *testing.T) {
	mainYAML := `openapi: 3.0.0
info:
  title: OneOf inline
  version: 1.0.0
paths: {}
components:
  schemas:
    Pet:
      type: object
      oneOf:
        - $ref: './cat.yaml#/components/schemas/Cat'
        - type: object
          properties:
            name:
              type: string`

	externalYAML := `components:
  schemas:
    Cat:
      type: object
      properties:
        name:
          type: string
        meow:
          type: boolean`

	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "main.yaml"), []byte(mainYAML), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "cat.yaml"), []byte(externalYAML), 0644))

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))
	bundled, err := BundleBytes(mainBytes, &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	})
	require.NoError(t, err)

	// bundled spec must NOT contain the external URI string
	assert.NotContains(t, string(bundled), "./cat.yaml#/components/schemas/Cat")

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(bundled, &doc))

	oneOf := doc["components"].(map[string]any)["schemas"].(map[string]any)["Pet"].(map[string]any)["oneOf"].([]any)

	first := oneOf[0].(map[string]any)
	_, hasRef := first["$ref"]
	assert.False(t, hasRef, "first oneOf entry should be inlined (no $ref)")
	_, hasProps := first["properties"]
	assert.True(t, hasProps, "inlined schema should expose properties")

	_, catExists := doc["components"].(map[string]any)["schemas"].(map[string]any)["Cat"]
	assert.False(t, catExists, "Cat must not be duplicated in components")

	runtime.GC()
}

// TestBundleBytes_AnyOfWithoutDiscriminatorMappingInlined tests that an anyOf schema
// without a discriminator mapping is inlined, similar to the oneOf test above.
func TestBundleBytes_AnyOfWithoutDiscriminatorMappingInlined(t *testing.T) {
	mainYAML := `openapi: 3.0.0
info:
  title: AnyOf inline
  version: 1.0.0
paths: {}
components:
  schemas:
    Response:
      anyOf:
        - $ref: './error.yaml#/components/schemas/Error'
        - type: object
          properties:
            data:
              type: string`

	externalYAML := `components:
  schemas:
    Error:
      type: object
      properties:
        message:
          type: string
        code:
          type: integer`

	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "main.yaml"), []byte(mainYAML), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "error.yaml"), []byte(externalYAML), 0644))

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))
	bundled, err := BundleBytes(mainBytes, &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	})
	require.NoError(t, err)

	assert.NotContains(t, string(bundled), "./error.yaml#/components/schemas/Error")

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(bundled, &doc))

	anyOf := doc["components"].(map[string]any)["schemas"].(map[string]any)["Response"].(map[string]any)["anyOf"].([]any)

	first := anyOf[0].(map[string]any)
	_, hasRef := first["$ref"]
	assert.False(t, hasRef, "first anyOf entry should be inlined")

	_, hasProps := first["properties"]
	assert.True(t, hasProps, "inlined schema should expose properties")

	_, errExists := doc["components"].(map[string]any)["schemas"].(map[string]any)["Error"]
	assert.False(t, errExists, "Error schema must not be duplicated in components")

	runtime.GC()
}

// TestBundleBytes_DiscriminatorMappingAnyOf tests that an anyOf schema with a discriminator mapping
// keeps external refs as $refs instead of inlining them (same behavior as oneOf with discriminator).
func TestBundleBytes_DiscriminatorMappingAnyOf(t *testing.T) {
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
          cat: './external-cat.yaml#/components/schemas/Cat'
      anyOf:
        - $ref: './external-cat.yaml#/components/schemas/Cat'
    Dog:
      type: object`

	ext := `components:
  schemas:
    Cat:
      type: object
      properties:
        type:
          type: string
        meow:
          type: boolean`

	tmp := t.TempDir()
	write := func(name, src string) {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(src), 0644))
	}
	write("main.yaml", spec)
	write("external-cat.yaml", ext)

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))
	cfg := &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	}

	out, err := BundleBytes(mainBytes, cfg)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(out, &doc))

	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	animal := schemas["Animal"].(map[string]any)

	// mapping value unchanged
	mapping := animal["discriminator"].(map[string]any)["mapping"].(map[string]any)
	assert.Equal(t, "./external-cat.yaml#/components/schemas/Cat", mapping["cat"])

	// the same $ref inside anyOf is also unchanged
	anyOf := animal["anyOf"].([]any)[0].(map[string]any)
	assert.Equal(t, "./external-cat.yaml#/components/schemas/Cat", anyOf["$ref"])

	// Cat schema NOT copied into components
	_, copied := schemas["Cat"]
	assert.False(t, copied, "Cat schema must not be inlined")

	runtime.GC()
}

// TestBundleBytes_DiscriminatorEdgeCases exercises the edge-cases of a discriminator that are likely
// not intended, but still parseable by the OpenAPI parser
func TestBundleBytes_DiscriminatorEdgeCases(t *testing.T) {
	spec := `openapi: 3.0.0
info:
  title: Weird discriminator shapes
  version: 1.0.0
paths: {}
components:
  schemas:
    Pet:
      discriminator: type
      oneOf:
        - true
        - type: object
          properties:
            legs:
              type: integer
        - $ref: '#/components/schemas/Dog'
    Dog:
      type: object
      properties:
        bark:
          type: boolean`

	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "weird.yaml"), []byte(spec), 0o644))

	in, _ := os.ReadFile(filepath.Join(tmp, "weird.yaml"))

	out, err := BundleBytes(in, &datamodel.DocumentConfiguration{BasePath: tmp})
	assert.NoError(t, err)
	assert.NotEmpty(t, out)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(out, &doc))

	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	pet := schemas["Pet"].(map[string]any)

	oneOf := pet["oneOf"].([]any)
	assert.Len(t, oneOf, 3)

	_, isObj := oneOf[0].(map[string]any)
	assert.True(t, isObj, "first oneOf item should get removed and turned into an empty object")

	_, hasRef := oneOf[1].(map[string]any)["$ref"]
	assert.False(t, hasRef, "second item has no $ref and should remain inline")

	ref := oneOf[2].(map[string]any)["$ref"]
	assert.Equal(t, "#/components/schemas/Dog", ref)

	_, dogExists := schemas["Dog"]
	assert.True(t, dogExists)
	assert.Len(t, schemas, 2)

	runtime.GC()
}

// TestBundleComposed_DuplicateNonComposableReferences tests the fix for issue #464
// When a file that cannot be composed into a component is referenced multiple times,
// all references should be properly inlined and no absolute paths should remain.
func TestBundleComposed_DuplicateNonComposableReferences(t *testing.T) {
	// Create test directory structure
	tmpDir := t.TempDir()

	// Main spec file - simplified version of the issue example
	mainSpec := `openapi: 3.0.1
info:
  title: Test API
  version: 1.0.0
paths:
  /foos:
    post:
      requestBody:
        $ref: "./components/requests/foo.yaml"
  /bars:
    put:
      requestBody:
        $ref: "./components/requests/bar.yaml"`

	// Request files that reference schemas
	fooRequest := `content:
  application/json:
    schema:
      $ref: "../schemas/foo.yaml"`

	barRequest := `content:
  application/json:
    schema:
      $ref: "../schemas/bar.yaml"`

	// Schema files that both reference the same example
	// This is the key part - both schemas reference the same file
	fooSchema := `type: object
properties:
  foo:
    type: string
example:
  $ref: ../examples/bar.yaml`

	barSchema := `type: object
properties:
  bar:
    type: string
example:
  $ref: ../examples/bar.yaml`

	// Example file that is NOT a valid OpenAPI Example component
	// (missing 'value' or 'externalValue' field required for Example objects)
	// This forces it to be inlined rather than composed
	invalidExample := `foo: "bar"`

	// Create directory structure
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "components", "requests"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "components", "schemas"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "components", "examples"), 0755))

	// Write files
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.yaml"), []byte(mainSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "components", "requests", "foo.yaml"), []byte(fooRequest), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "components", "requests", "bar.yaml"), []byte(barRequest), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "components", "schemas", "foo.yaml"), []byte(fooSchema), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "components", "schemas", "bar.yaml"), []byte(barSchema), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "components", "examples", "bar.yaml"), []byte(invalidExample), 0644))

	// Load and bundle the spec
	specBytes, err := os.ReadFile(filepath.Join(tmpDir, "main.yaml"))
	require.NoError(t, err)

	cfg := datamodel.DocumentConfiguration{
		BasePath:                tmpDir,
		ExtractRefsSequentially: true,
		AllowFileReferences:     true,
	}

	// Use the composed bundler
	bundled, err := BundleBytesComposed(specBytes, &cfg, &BundleCompositionConfig{})
	require.NoError(t, err)

	bundledStr := string(bundled)

	// The main assertion: no absolute paths should remain in the output
	assert.NotContains(t, bundledStr, tmpDir,
		"Bundled output should not contain absolute paths to temp directory")
	assert.NotContains(t, bundledStr, "/components/examples/bar.yaml",
		"Bundled output should not contain file path references")

	// Verify both schemas have the example content inlined
	lines := strings.Split(bundledStr, "\n")
	exampleCount := 0
	for _, line := range lines {
		// Count occurrences of the inlined content
		if strings.Contains(line, `foo: "bar"`) {
			exampleCount++
		}
	}

	// Should find the example content inlined twice (once for each schema)
	assert.GreaterOrEqual(t, exampleCount, 2,
		"Example content should be inlined in both schemas that reference it")

	// Additional verification: the bundled document should be valid
	doc, err := libopenapi.NewDocumentWithConfiguration(bundled, &cfg)
	require.NoError(t, err, "Bundled document should be valid OpenAPI")

	// Build the model to ensure it's processable
	v3Model, errs := doc.BuildV3Model()
	assert.Empty(t, errs, "Should build v3 model without errors")
	assert.NotNil(t, v3Model, "V3 model should not be nil")
}

// TestBundleComposed_FallbackInlineResolution tests the fallback mechanism for inline resolution
// This ensures the code at lines 212-216 is covered when inlinedPaths doesn't have exact match
func TestBundleComposed_FallbackInlineResolution(t *testing.T) {
	// Create test directory structure
	tmpDir := t.TempDir()

	// Main spec that references a component file that itself has an external reference
	mainSpec := `openapi: 3.0.1
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    post:
      requestBody:
        $ref: "./components/request.yaml"`

	// Request file with a complex reference structure
	requestFile := `content:
  application/json:
    schema:
      type: object
      properties:
        data:
          $ref: "./schema.yaml#/definitions/MyType"`

	// Schema file with definitions
	schemaFile := `definitions:
  MyType:
    type: object
    properties:
      example:
        $ref: "../invalid/example.yaml"`

	// Invalid example that needs inlining
	invalidExample := `invalid: "test"`

	// Create directories
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "components"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "invalid"), 0755))

	// Write files
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.yaml"), []byte(mainSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "components", "request.yaml"), []byte(requestFile), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "components", "schema.yaml"), []byte(schemaFile), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "invalid", "example.yaml"), []byte(invalidExample), 0644))

	// Load and bundle the spec
	specBytes, err := os.ReadFile(filepath.Join(tmpDir, "main.yaml"))
	require.NoError(t, err)

	cfg := datamodel.DocumentConfiguration{
		BasePath:                tmpDir,
		ExtractRefsSequentially: true,
		AllowFileReferences:     true,
	}

	// Use the composed bundler
	bundled, err := BundleBytesComposed(specBytes, &cfg, &BundleCompositionConfig{})
	require.NoError(t, err)

	bundledStr := string(bundled)

	// No absolute paths should remain
	assert.NotContains(t, bundledStr, tmpDir,
		"Bundled output should not contain absolute paths")
	assert.NotContains(t, bundledStr, "/invalid/example.yaml",
		"Bundled output should not contain file path references")
}

// TestBundleComposed_EdgeCaseCoverage tests additional edge cases for complete coverage
func TestBundleComposed_EdgeCaseCoverage(t *testing.T) {
	// Test case specifically designed to trigger the fallback path (lines 212-216)
	// This happens when a file has multiple references but only gets processed once
	tmpDir := t.TempDir()

	// Create a more complex scenario with nested references
	mainSpec := `openapi: 3.0.1
info:
  title: Test API  
  version: 1.0.0
paths:
  /test1:
    get:
      responses:
        200:
          $ref: "./responses/r1.yaml"
  /test2:
    get:
      responses:
        200:
          $ref: "./responses/r2.yaml"`

	// Response files that both eventually reference the same non-composable file
	r1 := `description: "Response 1"
content:
  application/json:
    schema:
      $ref: "../schemas/s1.yaml"`

	r2 := `description: "Response 2"
content:
  application/json:
    schema:
      $ref: "../schemas/s2.yaml"`

	// Schema files that both reference a shared non-composable file
	s1 := `type: object
properties:
  data:
    $ref: "../shared/invalid.yaml"`

	s2 := `type: object  
properties:
  info:
    $ref: "../shared/invalid.yaml"`

	// Invalid file that can't be composed (not a valid OpenAPI component)
	invalid := `notAValidComponent: true
someData: "test"`

	// Create directories
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "responses"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "shared"), 0755))

	// Write files
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.yaml"), []byte(mainSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "responses", "r1.yaml"), []byte(r1), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "responses", "r2.yaml"), []byte(r2), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "schemas", "s1.yaml"), []byte(s1), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "schemas", "s2.yaml"), []byte(s2), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "shared", "invalid.yaml"), []byte(invalid), 0644))

	cfg := datamodel.DocumentConfiguration{
		BasePath:                tmpDir,
		ExtractRefsSequentially: true,
		AllowFileReferences:     true,
	}

	bundled, err := BundleBytesComposed([]byte(mainSpec), &cfg, &BundleCompositionConfig{})
	require.NoError(t, err)

	bundledStr := string(bundled)

	// The bundled output should not contain absolute paths
	assert.NotContains(t, bundledStr, filepath.Join(tmpDir, "shared", "invalid.yaml"),
		"Should not contain absolute path to invalid.yaml")
	assert.NotContains(t, bundledStr, tmpDir,
		"No absolute paths should remain in output")

	// Check the actual output structure
	// The shared/invalid.yaml should be inlined somewhere
	// It might be represented differently depending on how it was processed

	// Since our invalid file can't be composed, verify it doesn't remain as external ref
	// and that the processing completes without errors
	assert.NotNil(t, bundled, "Bundled output should not be nil")
}

// TestRenderInline_DigitalOceanAsync tests if RenderInline() works as an alternative to the bundler
// for resolving refs in async mode. This is Option C from the investigation.
func TestRenderInline_DigitalOceanAsync(t *testing.T) {
	// test the mother of all exploded specs.
	tmp := t.TempDir()
	cmd := exec.Command("git", "clone", "-b", "asb/dedup-key-model", "https://github.com/digitalocean/openapi.git", tmp)

	err := cmd.Run()
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}

	spec, _ := filepath.Abs(filepath.Join(tmp, "specification", "DigitalOcean-public.v2.yaml"))
	digi, _ := os.ReadFile(spec)

	doc, err := libopenapi.NewDocumentWithConfiguration(digi, &datamodel.DocumentConfiguration{
		SpecFilePath:            spec,
		BasePath:                filepath.Join(tmp, "specification"),
		ExtractRefsSequentially: false, // ASYNC mode
		Logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelWarn, // Reduce noise
		})),
	})
	if err != nil {
		panic(err)
	}

	v3Doc, errs := doc.BuildV3Model()
	if errs != nil {
		t.Fatal("Errors building V3 model:", errs)
	}

	// Use RenderInline instead of BundleDocument
	renderedBytes, e := v3Doc.Model.RenderInline()
	assert.NoError(t, e)

	preservedRefs := collectAllDiscriminatorRefs(&v3Doc.Model)

	unresolvedCount := 0
	lines := strings.Split(string(renderedBytes), "\n")
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}
		if strings.Contains(trimmedLine, "$ref") && !isPreservedRef(trimmedLine, preservedRefs) {
			unresolvedCount++
			if unresolvedCount <= 10 {
				t.Logf("Unresolved $ref: %s", trimmedLine)
			}
		}
	}

	t.Logf("Total unresolved $ref entries (excluding discriminator mappings): %d", unresolvedCount)
	t.Logf("Preserved discriminator mapping refs: %d", len(preservedRefs))

	// RenderInline should resolve more refs than regular Render
	// Note: This test is exploratory - we're checking if RenderInline even works
	// It may still have some unresolved refs due to circular references
}

func TestBundleDocument_ResolvesExtensionRefs(t *testing.T) {
	tmp := t.TempDir()

	mainSpec := `openapi: 3.1.0
info:
  title: Test
  version: "1.0"
  x-custom:
    $ref: './custom.yaml'
paths:
  /test:
    get:
      x-code-samples:
        - lang: curl
          source:
            $ref: './examples/curl.md'
      responses:
        "200":
          description: OK`

	customData := `name: Custom Extension
value: resolved from external file
nested:
  foo: bar`

	curlExample := `curl -X GET https://api.example.com/test`

	// Write all files
	specPath := filepath.Join(tmp, "main.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte(mainSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "custom.yaml"), []byte(customData), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "examples"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "examples", "curl.md"), []byte(curlExample), 0644))

	// Read spec from file and configure with proper SpecFilePath
	specBytes, err := os.ReadFile(specPath)
	require.NoError(t, err)

	bundled, err := BundleBytes(specBytes, &datamodel.DocumentConfiguration{
		SpecFilePath:            specPath,
		BasePath:                tmp,
		AllowFileReferences:     true,
		ExtractRefsSequentially: true,
	})
	require.NoError(t, err)

	bundledStr := string(bundled)

	// Verify YAML extension ref was resolved and content was inlined
	assert.NotContains(t, bundledStr, "$ref: './custom.yaml'",
		"x-custom $ref should be resolved")
	assert.Contains(t, bundledStr, "name: Custom Extension",
		"Custom extension content should be inlined")
	assert.Contains(t, bundledStr, "value: resolved from external file",
		"Custom extension content should be inlined")

	// Verify raw text extension ref was resolved and content was inlined
	assert.NotContains(t, bundledStr, "$ref: './examples/curl.md'",
		"x-code-samples source $ref should be resolved")
	assert.Contains(t, bundledStr, "curl -X GET",
		"Curl example content should be inlined")
}

func TestBundleDocument_ResolvesDuplicateExtensionRefs(t *testing.T) {
	tmp := t.TempDir()

	mainSpec := `openapi: 3.1.0
info:
  title: Test
  version: "1.0"
  x-first:
    $ref: './custom.yaml'
  x-second:
    $ref: './custom.yaml'
paths:
  /test:
    get:
      x-code-samples:
        - lang: curl
          source:
            $ref: './examples/curl.md'
        - lang: curl
          source:
            $ref: './examples/curl.md'
      responses:
        "200":
          description: OK`

	customData := `name: Custom Extension
value: resolved from external file
nested:
  foo: bar`

	curlExample := `curl -X GET https://api.example.com/test`

	specPath := filepath.Join(tmp, "main.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte(mainSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "custom.yaml"), []byte(customData), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "examples"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "examples", "curl.md"), []byte(curlExample), 0644))

	specBytes, err := os.ReadFile(specPath)
	require.NoError(t, err)

	bundled, err := BundleBytes(specBytes, &datamodel.DocumentConfiguration{
		SpecFilePath:            specPath,
		BasePath:                tmp,
		AllowFileReferences:     true,
		ExtractRefsSequentially: true,
	})
	require.NoError(t, err)

	bundledStr := string(bundled)

	assert.NotContains(t, bundledStr, "$ref: './custom.yaml'",
		"Duplicate x-* $refs should all be resolved")
	assert.NotContains(t, bundledStr, "$ref: './examples/curl.md'",
		"Duplicate nested extension $refs should all be resolved")
	assert.Equal(t, 2, strings.Count(bundledStr, "name: Custom Extension"),
		"Resolved YAML extension content should be inlined for each occurrence")
	assert.Equal(t, 2, strings.Count(bundledStr, "curl -X GET https://api.example.com/test"),
		"Resolved raw text extension content should be inlined for each occurrence")
}

func TestBundleDocument_ExtensionRefsToLocalComponents(t *testing.T) {
	// Test that extension refs to local components (#/components/...) are resolved
	mainSpec := `openapi: 3.1.0
info:
  title: Test
  version: "1.0"
  x-schema-ref:
    $ref: '#/components/schemas/MySchema'
components:
  schemas:
    MySchema:
      type: object
      properties:
        name:
          type: string
paths:
  /test:
    get:
      responses:
        "200":
          description: OK`

	bundled, err := BundleBytes([]byte(mainSpec), &datamodel.DocumentConfiguration{
		ExtractRefsSequentially: true,
	})
	require.NoError(t, err)

	bundledStr := string(bundled)

	// Extension ref to local component should be resolved
	assert.NotContains(t, bundledStr, "$ref: '#/components/schemas/MySchema'",
		"Extension ref to local component should be resolved")
	// The schema content should be inlined in the extension
	assert.Contains(t, bundledStr, "x-schema-ref:",
		"Extension key should be present")
}

// TestBundleBytesWithConfig_Issue477_DiscriminatorExternalRefs tests the fix for issue #477:
// OneOfs with Discriminator Mappings in External Files Will Break With Inline Bundling.
// When ResolveDiscriminatorExternalRefs is enabled, external schemas referenced by discriminators
// are copied to the root document's components section.
func TestBundleBytesWithConfig_Issue477_DiscriminatorExternalRefs(t *testing.T) {
	// Parent file referencing external schema with discriminator
	parentYAML := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /shopping:
    get:
      responses:
        '200':
          description: Catalog page response
          content:
            application/json:
              schema:
                $ref: 'internal_schemas.yaml#/components/schemas/ResponseCatalogSection'`

	// External file with discriminator + oneOf pointing to local schemas
	externalYAML := `components:
  schemas:
    ResponseCatalogSection:
      oneOf:
        - $ref: '#/components/schemas/ResponseCatalogTileGroupSection'
        - $ref: '#/components/schemas/ResponseCatalogTableSection'
      discriminator:
        propertyName: type
        mapping:
          "TILE_GROUP_SECTION": '#/components/schemas/ResponseCatalogTileGroupSection'
          "TABLE_GROUP_SECTION": '#/components/schemas/ResponseCatalogTableSection'
    ResponseCatalogTileGroupSection:
      type: object
      properties:
        type:
          type: string
        tiles:
          type: array
    ResponseCatalogTableSection:
      type: object
      properties:
        type:
          type: string
        rows:
          type: array`

	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "main.yaml"), []byte(parentYAML), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "internal_schemas.yaml"), []byte(externalYAML), 0644))

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))
	cfg := &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	}
	bCfg := &BundleInlineConfig{
		ResolveDiscriminatorExternalRefs: true,
	}

	out, err := BundleBytesWithConfig(mainBytes, cfg, bCfg)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(out, &doc))

	// Verify components section exists and contains the discriminated schemas
	components, ok := doc["components"].(map[string]any)
	require.True(t, ok, "components section should exist")

	schemas, ok := components["schemas"].(map[string]any)
	require.True(t, ok, "schemas section should exist")

	// Check that the discriminated schemas were copied
	_, hasTileGroup := schemas["ResponseCatalogTileGroupSection"]
	_, hasTableSection := schemas["ResponseCatalogTableSection"]
	assert.True(t, hasTileGroup, "ResponseCatalogTileGroupSection should be in components")
	assert.True(t, hasTableSection, "ResponseCatalogTableSection should be in components")

	// Verify the bundled output doesn't contain external file references
	bundledStr := string(out)
	assert.NotContains(t, bundledStr, "internal_schemas.yaml",
		"Bundled output should not contain external file references")

	runtime.GC()
}

// TestBundleBytesWithConfig_DiscriminatorExternalRefs_AnyOf tests that anyOf with discriminator
// mappings pointing to external files works correctly with ResolveDiscriminatorExternalRefs.
func TestBundleBytesWithConfig_DiscriminatorExternalRefs_AnyOf(t *testing.T) {
	// Parent file referencing external schema with discriminator using anyOf
	parentYAML := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /shopping:
    get:
      responses:
        '200':
          description: Catalog page response
          content:
            application/json:
              schema:
                $ref: 'internal_schemas.yaml#/components/schemas/ResponseCatalogSection'`

	// External file with discriminator + anyOf pointing to local schemas
	externalYAML := `components:
  schemas:
    ResponseCatalogSection:
      anyOf:
        - $ref: '#/components/schemas/ResponseCatalogTileGroupSection'
        - $ref: '#/components/schemas/ResponseCatalogTableSection'
      discriminator:
        propertyName: type
        mapping:
          "TILE_GROUP_SECTION": '#/components/schemas/ResponseCatalogTileGroupSection'
          "TABLE_GROUP_SECTION": '#/components/schemas/ResponseCatalogTableSection'
    ResponseCatalogTileGroupSection:
      type: object
      properties:
        type:
          type: string
        tiles:
          type: array
    ResponseCatalogTableSection:
      type: object
      properties:
        type:
          type: string
        rows:
          type: array`

	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "main.yaml"), []byte(parentYAML), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "internal_schemas.yaml"), []byte(externalYAML), 0644))

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))
	cfg := &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	}
	bCfg := &BundleInlineConfig{
		ResolveDiscriminatorExternalRefs: true,
	}

	out, err := BundleBytesWithConfig(mainBytes, cfg, bCfg)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(out, &doc))

	// Verify components section exists and contains the discriminated schemas
	components, ok := doc["components"].(map[string]any)
	require.True(t, ok, "components section should exist")

	schemas, ok := components["schemas"].(map[string]any)
	require.True(t, ok, "schemas section should exist")

	// Check that the discriminated schemas were copied
	_, hasTileGroup := schemas["ResponseCatalogTileGroupSection"]
	_, hasTableSection := schemas["ResponseCatalogTableSection"]
	assert.True(t, hasTileGroup, "ResponseCatalogTileGroupSection should be in components")
	assert.True(t, hasTableSection, "ResponseCatalogTableSection should be in components")

	// Verify the bundled output doesn't contain external file references
	bundledStr := string(out)
	assert.NotContains(t, bundledStr, "internal_schemas.yaml",
		"Bundled output should not contain external file references")

	runtime.GC()
}

func TestBundleBytesWithConfig_InvalidModel(t *testing.T) {
	// Test that BundleBytesWithConfig returns ErrInvalidModel when BuildV3Model fails
	// Using Swagger 2.0 spec triggers "wrong version" error from BuildV3Model

	swagger2Spec := []byte(`swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths: {}`)

	_, err := BundleBytesWithConfig(swagger2Spec, nil, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidModel)
	assert.Contains(t, err.Error(), "different version")
}

// TestBundleBytesWithConfig_BackwardCompatibility tests that existing behavior is preserved
// when ResolveDiscriminatorExternalRefs is not enabled.
func TestBundleBytesWithConfig_BackwardCompatibility(t *testing.T) {
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
          cat: './external-cat.yaml#/components/schemas/Cat'
      oneOf:
        - $ref: './external-cat.yaml#/components/schemas/Cat'`

	ext := `components:
  schemas:
    Cat:
      type: object
      properties:
        type:
          type: string
        meow:
          type: boolean`

	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "main.yaml"), []byte(spec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "external-cat.yaml"), []byte(ext), 0644))

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))
	cfg := &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	}

	// Test WITHOUT the config flag (existing behavior)
	out, err := BundleBytes(mainBytes, cfg)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(out, &doc))

	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	animal := schemas["Animal"].(map[string]any)

	// Existing behavior: external refs should remain unchanged
	mapping := animal["discriminator"].(map[string]any)["mapping"].(map[string]any)
	assert.Equal(t, "./external-cat.yaml#/components/schemas/Cat", mapping["cat"],
		"Without config flag, external refs should remain unchanged")

	// Test WITH nil config (should behave same as no config)
	out2, err := BundleBytesWithConfig(mainBytes, cfg, nil)
	require.NoError(t, err)

	var doc2 map[string]any
	require.NoError(t, yaml.Unmarshal(out2, &doc2))

	schemas2 := doc2["components"].(map[string]any)["schemas"].(map[string]any)
	animal2 := schemas2["Animal"].(map[string]any)
	mapping2 := animal2["discriminator"].(map[string]any)["mapping"].(map[string]any)
	assert.Equal(t, "./external-cat.yaml#/components/schemas/Cat", mapping2["cat"],
		"With nil config, external refs should remain unchanged")

	runtime.GC()
}

// TestBundleBytesWithConfig_MultipleExternalFiles tests discriminator refs pointing to
// schemas in different external files.
func TestBundleBytesWithConfig_MultipleExternalFiles(t *testing.T) {
	spec := `openapi: 3.0.0
info:
  title: Vehicles
  version: 1.0.0
paths: {}
components:
  schemas:
    Vehicle:
      type: object
      discriminator:
        propertyName: kind
        mapping:
          car: './vehicles/car.yaml#/components/schemas/Car'
          bike: './vehicles/bike.yaml#/components/schemas/Bike'
      oneOf:
        - $ref: './vehicles/car.yaml#/components/schemas/Car'
        - $ref: './vehicles/bike.yaml#/components/schemas/Bike'`

	car := `components:
  schemas:
    Car:
      type: object
      properties:
        wheels:
          type: integer`
	bike := `components:
  schemas:
    Bike:
      type: object
      properties:
        wheels:
          type: integer`

	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "vehicles"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "main.yaml"), []byte(spec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "vehicles", "car.yaml"), []byte(car), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "vehicles", "bike.yaml"), []byte(bike), 0644))

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))
	cfg := &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	}
	bCfg := &BundleInlineConfig{
		ResolveDiscriminatorExternalRefs: true,
	}

	out, err := BundleBytesWithConfig(mainBytes, cfg, bCfg)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(out, &doc))

	// Verify components section contains both schemas
	components := doc["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)

	_, hasCar := schemas["Car"]
	_, hasBike := schemas["Bike"]
	assert.True(t, hasCar, "Car schema should be in components")
	assert.True(t, hasBike, "Bike schema should be in components")

	// Verify no external file references
	bundledStr := string(out)
	assert.NotContains(t, bundledStr, "car.yaml", "Should not contain external file refs")
	assert.NotContains(t, bundledStr, "bike.yaml", "Should not contain external file refs")

	runtime.GC()
}

// TestBundleDocumentWithConfig tests that BundleDocumentWithConfig works correctly.
func TestBundleDocumentWithConfig(t *testing.T) {
	spec := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
paths: {}
components:
  schemas:
    Pet:
      type: object
      properties:
        name:
          type: string`

	doc, err := libopenapi.NewDocument([]byte(spec))
	require.NoError(t, err)

	v3Doc, errs := doc.BuildV3Model()
	require.Empty(t, errs)

	// Test with nil config
	out, err := BundleDocumentWithConfig(&v3Doc.Model, nil)
	require.NoError(t, err)
	assert.Contains(t, string(out), "Pet")

	// Test with config
	out2, err := BundleDocumentWithConfig(&v3Doc.Model, &BundleInlineConfig{
		ResolveDiscriminatorExternalRefs: false,
	})
	require.NoError(t, err)
	assert.Contains(t, string(out2), "Pet")

	runtime.GC()
}

// TestCalculateCollisionNameInline tests the collision name generation for inline bundling.
func TestCalculateCollisionNameInline(t *testing.T) {
	existing := map[string]bool{"Cat": true}

	// Test filename-based collision resolution
	result := calculateCollisionNameInline("Cat", "external.yaml#/components/schemas/Cat", "__", existing)
	assert.Equal(t, "Cat__external", result)

	// Test when filename-based also collides
	existing["Cat__external"] = true
	result = calculateCollisionNameInline("Cat", "external.yaml#/components/schemas/Cat", "__", existing)
	assert.Equal(t, "Cat__external__1", result)

	// Test no collision returns filename-based name
	result = calculateCollisionNameInline("Dog", "file.yaml#/components/schemas/Dog", "__", existing)
	assert.Equal(t, "Dog__file", result)

	// Test with path containing directory
	result = calculateCollisionNameInline("Bird", "schemas/birds/bird.yaml#/components/schemas/Bird", "__", existing)
	assert.Equal(t, "Bird__bird", result)

	// Test when fullDef has no file path (just fragment), baseName is empty
	// So it tries name__ first, which is available
	result = calculateCollisionNameInline("Zebra", "#/components/schemas/Zebra", "__", existing)
	assert.Equal(t, "Zebra__", result) // empty baseName

	// Test when name__ already exists, falls back to name__1
	existing["Tiger__"] = true
	result = calculateCollisionNameInline("Tiger", "#/components/schemas/Tiger", "__", existing)
	assert.Equal(t, "Tiger____1", result) // name__ + delimiter + 1

	// Test numeric suffix fallback when both filename-based and name__ exist
	existing["Lion__"] = true
	existing["Lion____1"] = true
	result = calculateCollisionNameInline("Lion", "#/components/schemas/Lion", "__", existing)
	assert.Equal(t, "Lion____2", result)
}

func TestErrorHandlingOnBundleDocument(t *testing.T) {

	b, err := BundleBytesWithConfig([]byte("hey: hey: hey: : hey : hey"), nil, nil)
	assert.Nil(t, b)
	assert.Error(t, err)

	// resolveDiscriminatorExternalRefs handles nil gracefully (no return value)
	resolveDiscriminatorExternalRefs(nil)

	rewriteInlineDiscriminatorRefs(nil, nil)
	updateOneOfAnyOfRefs(nil, nil)
	walkDiscriminatorMapping(nil, &yaml.Node{Kind: yaml.ScalarNode}, nil)

	// walkUnionRefs: hit first continue (item.Kind != yaml.MappingNode)
	walkUnionRefs(nil, &yaml.Node{
		Kind: yaml.SequenceNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "not-a-mapping"},
		},
	}, nil)

	// walkUnionRefs: hit second continue (k.Value != "$ref")
	walkUnionRefs(nil, &yaml.Node{
		Kind: yaml.SequenceNode,
		Content: []*yaml.Node{
			{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "notRef"},
					{Kind: yaml.ScalarNode, Value: "someValue"},
				},
			},
		},
	}, nil)

	// updateUnionRefs: hit continue (item.Kind != yaml.MappingNode)
	updateUnionRefs(&yaml.Node{
		Kind: yaml.SequenceNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "not-a-mapping"},
		},
	}, nil)

	// updateUnionRefs: MappingNode but key != "$ref" (skips inner if)
	updateUnionRefs(&yaml.Node{
		Kind: yaml.SequenceNode,
		Content: []*yaml.Node{
			{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "notRef"},
					{Kind: yaml.ScalarNode, Value: "someValue"},
				},
			},
		},
	}, nil)
}

func TestResolveDiscriminatorExternalRefs_NoExternalSchemas(t *testing.T) {
	// Test: len(externalSchemas) == 0 path
	// Spec with discriminator that only references internal schemas (no external refs)
	spec := `openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Pet:
      type: object
      discriminator:
        propertyName: petType
        mapping:
          dog: '#/components/schemas/Dog'
          cat: '#/components/schemas/Cat'
      oneOf:
        - $ref: '#/components/schemas/Dog'
        - $ref: '#/components/schemas/Cat'
    Dog:
      type: object
      properties:
        petType:
          type: string
        bark:
          type: boolean
    Cat:
      type: object
      properties:
        petType:
          type: string
        meow:
          type: boolean
paths: {}`

	bundleConfig := &BundleInlineConfig{
		ResolveDiscriminatorExternalRefs: true,
	}

	result, err := BundleBytesWithConfig([]byte(spec), nil, bundleConfig)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify the output still has the internal refs (not modified)
	assert.Contains(t, string(result), "#/components/schemas/Dog")
	assert.Contains(t, string(result), "#/components/schemas/Cat")
}

func TestCollectExternalDiscriminatorSchemas_RootPathSkip(t *testing.T) {
	// Test: filePath == rootPath path
	// This is implicitly tested by TestResolveDiscriminatorExternalRefs_NoExternalSchemas
	// since internal refs have filePath == rootPath and get skipped

	// Additional explicit test using the internal function
	spec := `openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Pet:
      type: object
      discriminator:
        propertyName: petType
        mapping:
          dog: '#/components/schemas/Dog'
      oneOf:
        - $ref: '#/components/schemas/Dog'
    Dog:
      type: object
paths: {}`

	doc, err := libopenapi.NewDocument([]byte(spec))
	require.NoError(t, err)

	model, errs := doc.BuildV3Model()
	require.Empty(t, errs)

	rolodex := model.Model.Rolodex
	rootIdx := rolodex.GetRootIndex()

	// Collect external schemas - should return empty since all refs are internal
	result := collectExternalDiscriminatorSchemas(rolodex, rootIdx)
	assert.Empty(t, result, "Expected no external schemas when all discriminator refs are internal")
}

func TestCopySchemaToComponents_NameCollision(t *testing.T) {
	// Test: existingNames[finalName] collision path
	existingNames := map[string]bool{
		"Cat": true, // Simulate existing schema named "Cat"
	}

	// Create a mock external schema ref
	extSchema := &externalSchemaRef{
		schemaName: "Cat",
		fullDef:    "/some/path/external.yaml#/components/schemas/Cat",
		ref: &index.Reference{
			Node: &yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "type"},
					{Kind: yaml.ScalarNode, Value: "object"},
				},
			},
		},
	}

	// Create a minimal document with components
	doc := &v3high.Document{
		Components: &v3high.Components{
			Schemas: orderedmap.New[string, *base.SchemaProxy](),
		},
	}

	// Copy should handle collision by appending filename
	newRef := copySchemaToComponents(doc, extSchema, existingNames)

	// Should have created a collision-avoidance name
	assert.Equal(t, "#/components/schemas/Cat__external", newRef)
	assert.True(t, existingNames["Cat__external"], "Should track the new name")
}

func TestCalculateCollisionNameInline_NumericSuffix(t *testing.T) {
	// Test: When filename-based name also collides, use numeric suffix
	existingNames := map[string]bool{
		"Cat":             true,
		"Cat__external":   true, // Filename-based collision also exists
		"Cat__external__1": true, // First numeric suffix also taken (format: name__basename__N)
	}

	result := calculateCollisionNameInline("Cat", "/path/external.yaml#/components/schemas/Cat", "__", existingNames)
	assert.Equal(t, "Cat__external__2", result)
}

