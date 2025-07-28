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
	cmd := exec.Command("git", "clone", "-b", "asb/dedup-key-model", "https://github.com/digitalocean/openapi.git", tmp)
	defer os.RemoveAll(filepath.Join(tmp, "openapi"))

	err := cmd.Run()
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}

	spec, _ := filepath.Abs(filepath.Join(tmp+"/specification", "DigitalOcean-public.v2.yaml"))
	digi, _ := os.ReadFile(spec)

	doc, err := libopenapi.NewDocumentWithConfiguration(digi, &datamodel.DocumentConfiguration{
		SpecFilePath:            spec,
		BasePath:                tmp + "/specification",
		ExtractRefsSequentially: true,
		Logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})),
	})
	if err != nil {
		panic(err)
	}

	v3Doc, errs := doc.BuildV3Model()
	if len(errs) > 0 {
		t.Fatal("Errors building V3 model:", errs)
	}

	// collect refs that are allowed to be preserved.
	preservedRefs := map[string]struct{}{}
	rootIdx := v3Doc.Model.Rolodex.GetRootIndex()
	collectDiscriminatorMappingValues(rootIdx, rootIdx.GetRootNode(), preservedRefs)
	for _, idx := range v3Doc.Model.Rolodex.GetIndexes() {
		collectDiscriminatorMappingValues(idx, idx.GetRootNode(), preservedRefs)
	}

	clean := func(s string) string {
		// trim quotes and make slashes Unix-style
		return filepath.ToSlash(strings.Trim(s, `"'`))
	}

	extractRef := func(line string) string {
		i := strings.Index(line, "$ref:")
		if i == -1 {
			return ""
		}
		return clean(strings.TrimSpace(line[i+5:]))
	}

	isPreserved := func(line string) bool {
		ref := extractRef(line)
		if ref == "" {
			return false
		}
		for uri := range preservedRefs {
			if strings.HasSuffix(clean(uri), ref) {
				return true
			}
		}
		return false
	}

	bytes, e := BundleDocument(&v3Doc.Model)

	assert.NoError(t, e)
	lines := strings.Split(string(bytes), "\n")
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}
		if strings.Contains(trimmedLine, "$ref") && !isPreserved(trimmedLine) {
			t.Errorf("Found uncommented $ref in line: %s", line)
		}
	}
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
		assert.Len(t, *doc.GetSpecInfo().SpecBytes, 1692)
	}
	assert.Len(t, bytes, 2068)

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
	assert.Len(t, bytes, 2068)

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
	assert.Len(t, logEntries, 17)
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
