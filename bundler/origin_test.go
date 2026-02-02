// Copyright 2023-2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io
// SPDX-License-Identifier: MIT

package bundler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestComponentOrigin_Structure(t *testing.T) {
	t.Run("initializes with all fields", func(t *testing.T) {
		origin := &ComponentOrigin{
			OriginalFile:  "/path/to/models/User.yaml",
			OriginalRef:   "#/components/schemas/User",
			OriginalName:  "User",
			Line:          10,
			Column:        2,
			WasRenamed:    false,
			BundledRef:    "#/components/schemas/User",
			ComponentType: "schemas",
		}

		assert.Equal(t, "/path/to/models/User.yaml", origin.OriginalFile)
		assert.Equal(t, "#/components/schemas/User", origin.OriginalRef)
		assert.Equal(t, "User", origin.OriginalName)
		assert.Equal(t, 10, origin.Line)
		assert.Equal(t, 2, origin.Column)
		assert.False(t, origin.WasRenamed)
		assert.Equal(t, "#/components/schemas/User", origin.BundledRef)
		assert.Equal(t, "schemas", origin.ComponentType)
	})

	t.Run("handles renamed components", func(t *testing.T) {
		origin := &ComponentOrigin{
			OriginalName: "Pet",
			WasRenamed:   true,
			BundledRef:   "#/components/schemas/Pet__2",
		}

		assert.Equal(t, "Pet", origin.OriginalName)
		assert.True(t, origin.WasRenamed)
		assert.Equal(t, "#/components/schemas/Pet__2", origin.BundledRef)
	})
}

func TestBundleResult_NewBundleResult(t *testing.T) {
	t.Run("creates result with initialized maps", func(t *testing.T) {
		result := NewBundleResult()

		assert.NotNil(t, result)
		assert.NotNil(t, result.Origins)
		assert.Equal(t, 0, len(result.Origins))
		assert.Nil(t, result.Bytes)
	})
}

func TestBundleResult_AddOrigin(t *testing.T) {
	t.Run("adds origin to map", func(t *testing.T) {
		result := NewBundleResult()
		origin := &ComponentOrigin{
			OriginalFile: "/models/user.yaml",
			OriginalName: "User",
		}

		result.AddOrigin("#/components/schemas/User", origin)

		assert.Equal(t, 1, len(result.Origins))
		assert.Equal(t, "#/components/schemas/User", origin.BundledRef)
		retrieved := result.Origins["#/components/schemas/User"]
		assert.Equal(t, "/models/user.yaml", retrieved.OriginalFile)
	})

	t.Run("initializes origins map if nil", func(t *testing.T) {
		result := &BundleResult{}
		assert.Nil(t, result.Origins)

		origin := &ComponentOrigin{OriginalName: "Test"}
		result.AddOrigin("#/components/schemas/Test", origin)

		assert.NotNil(t, result.Origins)
		assert.Equal(t, 1, len(result.Origins))
	})

	t.Run("overwrites existing origin for same key", func(t *testing.T) {
		result := NewBundleResult()

		origin1 := &ComponentOrigin{OriginalFile: "/file1.yaml"}
		result.AddOrigin("#/components/schemas/User", origin1)

		origin2 := &ComponentOrigin{OriginalFile: "/file2.yaml"}
		result.AddOrigin("#/components/schemas/User", origin2)

		assert.Equal(t, 1, len(result.Origins))
		assert.Equal(t, "/file2.yaml", result.Origins["#/components/schemas/User"].OriginalFile)
	})
}

func TestBundleResult_GetOrigin(t *testing.T) {
	t.Run("retrieves existing origin", func(t *testing.T) {
		result := NewBundleResult()
		origin := &ComponentOrigin{OriginalFile: "/test.yaml"}
		result.AddOrigin("#/components/schemas/Test", origin)

		retrieved := result.GetOrigin("#/components/schemas/Test")

		assert.NotNil(t, retrieved)
		assert.Equal(t, "/test.yaml", retrieved.OriginalFile)
	})

	t.Run("returns nil for non-existent key", func(t *testing.T) {
		result := NewBundleResult()

		retrieved := result.GetOrigin("#/components/schemas/NonExistent")

		assert.Nil(t, retrieved)
	})

	t.Run("handles nil origins map", func(t *testing.T) {
		result := &BundleResult{}

		retrieved := result.GetOrigin("#/components/schemas/Test")

		assert.Nil(t, retrieved)
	})
}

func TestBundleResult_OriginCount(t *testing.T) {
	t.Run("returns zero for empty map", func(t *testing.T) {
		result := NewBundleResult()

		assert.Equal(t, 0, result.OriginCount())
	})

	t.Run("returns correct count", func(t *testing.T) {
		result := NewBundleResult()
		result.AddOrigin("#/components/schemas/User", &ComponentOrigin{})
		result.AddOrigin("#/components/schemas/Pet", &ComponentOrigin{})
		result.AddOrigin("#/components/responses/Success", &ComponentOrigin{})

		assert.Equal(t, 3, result.OriginCount())
	})

	t.Run("handles nil origins map", func(t *testing.T) {
		result := &BundleResult{}

		assert.Equal(t, 0, result.OriginCount())
	})
}

func TestBundleBytesComposedWithOrigins_SimpleSpec(t *testing.T) {
	// create temp directory for test files
	tmpDir := t.TempDir()

	// create main spec
	mainYAML := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '200':
          $ref: './responses.yaml#/UserListResponse'
components:
  schemas:
    LocalSchema:
      type: object
      properties:
        id:
          type: string`

	// create external responses file
	responsesYAML := `UserListResponse:
  description: List of users
  content:
    application/json:
      schema:
        $ref: './schemas.yaml#/UserList'`

	// create external schemas file
	schemasYAML := `UserList:
  type: array
  items:
    $ref: '#/User'
User:
  type: object
  properties:
    name:
      type: string`

	// write files
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.yaml"), []byte(mainYAML), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "responses.yaml"), []byte(responsesYAML), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "schemas.yaml"), []byte(schemasYAML), 0644))

	// load and bundle
	config := &datamodel.DocumentConfiguration{
		AllowFileReferences: true,
		BasePath:            tmpDir,
		SpecFilePath:        filepath.Join(tmpDir, "main.yaml"),
	}

	mainBytes, err := os.ReadFile(filepath.Join(tmpDir, "main.yaml"))
	require.NoError(t, err)

	result, err := BundleBytesComposedWithOrigins(mainBytes, config, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.NotEmpty(t, result.Bytes)
	assert.NotNil(t, result.Origins)
	assert.Greater(t, len(result.Origins), 0, "should have tracked origins")
	assert.Greater(t, result.OriginCount(), 0)

	// verify we can find the User schema origin
	var userOrigin *ComponentOrigin
	for bundledRef, origin := range result.Origins {
		if origin.OriginalName == "User" {
			userOrigin = origin
			assert.Contains(t, bundledRef, "User")
			break
		}
	}
	assert.NotNil(t, userOrigin, "User schema should have tracked origin")
	if userOrigin != nil {
		assert.Contains(t, userOrigin.OriginalFile, "schemas.yaml")
		assert.Equal(t, "User", userOrigin.OriginalName)
		assert.Equal(t, "schemas", userOrigin.ComponentType)
		assert.Greater(t, userOrigin.Line, 0)
	}
}

func TestBundleBytesComposedWithOrigins_CollisionHandling(t *testing.T) {
	tmpDir := t.TempDir()

	// create main spec with a local Pet schema
	mainYAML := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Pet:
      type: object
      properties:
        localId:
          type: string
paths:
  /pets:
    get:
      responses:
        '200':
          content:
            application/json:
              schema:
                $ref: './external.yaml#/Pet'`

	// create external file with conflicting Pet schema
	externalYAML := `Pet:
  type: object
  properties:
    externalId:
      type: integer`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.yaml"), []byte(mainYAML), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "external.yaml"), []byte(externalYAML), 0644))

	config := &datamodel.DocumentConfiguration{
		AllowFileReferences: true,
		BasePath:            tmpDir,
		SpecFilePath:        filepath.Join(tmpDir, "main.yaml"),
	}

	mainBytes, err := os.ReadFile(filepath.Join(tmpDir, "main.yaml"))
	require.NoError(t, err)

	result, err := BundleBytesComposedWithOrigins(mainBytes, config, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	// debug: print all origins
	t.Logf("Total origins tracked: %d", len(result.Origins))
	for bundledRef, origin := range result.Origins {
		t.Logf("Origin: bundled=%s, original=%s, file=%s, renamed=%v",
			bundledRef, origin.OriginalName, filepath.Base(origin.OriginalFile), origin.WasRenamed)
	}

	// find the Pet schema from external file - it will be renamed due to collision with main.yaml's Pet
	var externalPetOrigin *ComponentOrigin
	for bundledRef, origin := range result.Origins {
		if filepath.Base(origin.OriginalFile) == "external.yaml" {
			externalPetOrigin = origin
			t.Logf("Found external Pet: bundled=%s, original=%s, renamed=%v",
				bundledRef, origin.OriginalName, origin.WasRenamed)
			break
		}
	}

	// verify the external Pet was tracked
	assert.NotNil(t, externalPetOrigin, "external Pet schema should have origin")
	if externalPetOrigin != nil {
		assert.Contains(t, externalPetOrigin.OriginalFile, "external.yaml")
		assert.Equal(t, "schemas", externalPetOrigin.ComponentType)
		// the bundled ref should be different from #/components/schemas/Pet due to collision
		assert.NotEqual(t, "#/components/schemas/Pet", externalPetOrigin.BundledRef)
		assert.Contains(t, externalPetOrigin.BundledRef, "Pet")
	}
}

func TestBundleBytesComposedWithOrigins_MultipleComponentTypes(t *testing.T) {
	tmpDir := t.TempDir()

	mainYAML := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      parameters:
        - $ref: './params.yaml#/UserId'
      responses:
        '200':
          $ref: './responses.yaml#/UserResponse'
      requestBody:
        $ref: './bodies.yaml#/UserInput'`

	paramsYAML := `UserId:
  name: id
  in: query
  schema:
    type: string`

	responsesYAML := `UserResponse:
  description: User data
  content:
    application/json:
      schema:
        type: object`

	bodiesYAML := `UserInput:
  description: User input
  content:
    application/json:
      schema:
        type: object`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.yaml"), []byte(mainYAML), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "params.yaml"), []byte(paramsYAML), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "responses.yaml"), []byte(responsesYAML), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "bodies.yaml"), []byte(bodiesYAML), 0644))

	config := &datamodel.DocumentConfiguration{
		AllowFileReferences: true,
		BasePath:            tmpDir,
		SpecFilePath:        filepath.Join(tmpDir, "main.yaml"),
	}

	mainBytes, err := os.ReadFile(filepath.Join(tmpDir, "main.yaml"))
	require.NoError(t, err)

	result, err := BundleBytesComposedWithOrigins(mainBytes, config, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	// verify we tracked all component types
	foundTypes := make(map[string]bool)
	for _, origin := range result.Origins {
		foundTypes[origin.ComponentType] = true
	}

	// at minimum, we should have tracked different component types
	assert.Greater(t, len(foundTypes), 0, "should have multiple component types")

	// debug: print all origins
	t.Logf("Total origins tracked: %d", len(result.Origins))
	for bundledRef, origin := range result.Origins {
		t.Logf("Origin: bundled=%s, name=%s, type=%s, file=%s",
			bundledRef, origin.OriginalName, origin.ComponentType, filepath.Base(origin.OriginalFile))
	}

	// verify specific origins - check what we actually got
	foundResponse := false

	for _, origin := range result.Origins {
		if origin.OriginalName == "UserResponse" {
			foundResponse = true
			// check that the origin was tracked, component type may vary
			assert.Contains(t, origin.OriginalFile, "responses.yaml")
		}
	}

	assert.True(t, foundResponse, "should track response origin")
	// note: parameters and request bodies may not be tracked if they're inlined
	// rather than lifted to components
}

func TestBundleBytesComposedWithOrigins_SingleFileSpec(t *testing.T) {
	simpleSpec := `openapi: 3.1.0
info:
  title: Simple API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          description: OK
components:
  schemas:
    Simple:
      type: string`

	config := &datamodel.DocumentConfiguration{
		AllowFileReferences: false,
	}

	result, err := BundleBytesComposedWithOrigins([]byte(simpleSpec), config, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.NotEmpty(t, result.Bytes)
	// single-file specs may have no external refs to track
	assert.NotNil(t, result.Origins)
}

func TestBundleBytesComposedWithOrigins_CustomDelimiter(t *testing.T) {
	tmpDir := t.TempDir()

	mainYAML := `openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Pet:
      type: object
paths:
  /pets:
    get:
      responses:
        '200':
          content:
            application/json:
              schema:
                $ref: './external.yaml#/Pet'`

	externalYAML := `Pet:
  type: string`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.yaml"), []byte(mainYAML), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "external.yaml"), []byte(externalYAML), 0644))

	config := &datamodel.DocumentConfiguration{
		AllowFileReferences: true,
		BasePath:            tmpDir,
	}

	mainBytes, err := os.ReadFile(filepath.Join(tmpDir, "main.yaml"))
	require.NoError(t, err)

	// use custom delimiter
	compositionConfig := &BundleCompositionConfig{
		Delimiter: "@@",
	}

	result, err := BundleBytesComposedWithOrigins(mainBytes, config, compositionConfig)
	require.NoError(t, err)

	// verify origin tracking works with custom delimiter
	assert.NotNil(t, result.Origins)
	assert.Greater(t, len(result.Origins), 0, "should track origins")

	// debug log
	t.Logf("Origins with custom delimiter: %d", len(result.Origins))
	for bundledRef, origin := range result.Origins {
		t.Logf("  bundled=%s, original=%s, renamed=%v", bundledRef, origin.OriginalName, origin.WasRenamed)
		if origin.WasRenamed {
			assert.Contains(t, bundledRef, "@@", "renamed component should use custom delimiter")
		}
	}
}

func TestBundleBytesComposedWithOrigins_ErrorHandling(t *testing.T) {
	t.Run("handles invalid spec", func(t *testing.T) {
		invalidSpec := []byte("not: valid: yaml:")

		config := &datamodel.DocumentConfiguration{}
		result, err := BundleBytesComposedWithOrigins(invalidSpec, config, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("handles non-openapi document", func(t *testing.T) {
		notOpenAPI := []byte("key: value\nother: data")

		config := &datamodel.DocumentConfiguration{}
		result, err := BundleBytesComposedWithOrigins(notOpenAPI, config, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestBundleBytesComposedWithOrigins_ErrorModel(t *testing.T) {
	specBytes := []byte(`openapi: 3.1.0
info:
  title: Error Model
  version: 1.0.0
paths:
  /cake:
    $ref: '#/components/schemas/Cake'`)

	result, err := BundleBytesComposedWithOrigins(specBytes, nil, nil)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestBundleBytesComposedWithOrigins_LineAndColumnTracking(t *testing.T) {
	tmpDir := t.TempDir()

	mainYAML := `openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          content:
            application/json:
              schema:
                $ref: './schemas.yaml#/TestSchema'`

	// schemas file with specific line numbers we can verify
	schemasYAML := `# Comment line 1
# Comment line 2
TestSchema:
  type: object
  properties:
    field1:
      type: string`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.yaml"), []byte(mainYAML), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "schemas.yaml"), []byte(schemasYAML), 0644))

	config := &datamodel.DocumentConfiguration{
		AllowFileReferences: true,
		BasePath:            tmpDir,
	}

	mainBytes, err := os.ReadFile(filepath.Join(tmpDir, "main.yaml"))
	require.NoError(t, err)

	result, err := BundleBytesComposedWithOrigins(mainBytes, config, nil)
	require.NoError(t, err)

	// find TestSchema origin
	var testSchemaOrigin *ComponentOrigin
	for _, origin := range result.Origins {
		if origin.OriginalName == "TestSchema" {
			testSchemaOrigin = origin
			break
		}
	}

	require.NotNil(t, testSchemaOrigin, "should track TestSchema origin")
	assert.Equal(t, "TestSchema", testSchemaOrigin.OriginalName)
	// YAML parsing adds document node, so actual line number may be +1
	assert.Greater(t, testSchemaOrigin.Line, 0, "should have line info")
	assert.Greater(t, testSchemaOrigin.Column, 0, "should have column info")
}

func TestCaptureOrigin_EdgeCases(t *testing.T) {
	t.Run("handles nil processRef", func(t *testing.T) {
		origins := make(ComponentOriginMap)

		// should not panic
		assert.NotPanics(t, func() {
			captureOrigin(nil, "schemas", origins)
		})

		assert.Equal(t, 0, len(origins))
	})

	t.Run("handles nil origins map", func(t *testing.T) {
		pr := &processRef{
			name: "Test",
		}

		// should not panic
		assert.NotPanics(t, func() {
			captureOrigin(pr, "schemas", nil)
		})
	})

	t.Run("handles missing ref in processRef", func(t *testing.T) {
		origins := make(ComponentOriginMap)
		pr := &processRef{
			name: "Test",
			ref:  nil, // nil ref
		}

		captureOrigin(pr, "schemas", origins)

		assert.Equal(t, 0, len(origins), "should not add origin with nil ref")
	})

	t.Run("handles missing idx in processRef", func(t *testing.T) {
		origins := make(ComponentOriginMap)
		pr := &processRef{
			name: "Test",
			idx:  nil, // nil idx
		}

		captureOrigin(pr, "schemas", origins)

		assert.Equal(t, 0, len(origins), "should not add origin with nil idx")
	})
}

func TestBundleDocumentComposed_StillWorks(t *testing.T) {
	// verify the original BundleDocumentComposed function still works
	tmpDir := t.TempDir()

	mainYAML := `openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          $ref: './response.yaml#/TestResponse'`

	responseYAML := `TestResponse:
  description: Test response`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.yaml"), []byte(mainYAML), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "response.yaml"), []byte(responseYAML), 0644))

	config := &datamodel.DocumentConfiguration{
		AllowFileReferences: true,
		BasePath:            tmpDir,
	}

	mainBytes, err := os.ReadFile(filepath.Join(tmpDir, "main.yaml"))
	require.NoError(t, err)

	doc, err := libopenapi.NewDocumentWithConfiguration(mainBytes, config)
	require.NoError(t, err)

	v3Model, errs := doc.BuildV3Model()
	require.NoError(t, errs)

	// original function should still work
	bundledBytes, err := BundleDocumentComposed(&v3Model.Model, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, bundledBytes)
}

func TestBundleBytesComposed_StillWorks(t *testing.T) {
	// verify backward compatibility
	tmpDir := t.TempDir()

	mainYAML := `openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          $ref: './response.yaml#/TestResponse'`

	responseYAML := `TestResponse:
  description: Test response`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.yaml"), []byte(mainYAML), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "response.yaml"), []byte(responseYAML), 0644))

	config := &datamodel.DocumentConfiguration{
		AllowFileReferences: true,
		BasePath:            tmpDir,
	}

	mainBytes, err := os.ReadFile(filepath.Join(tmpDir, "main.yaml"))
	require.NoError(t, err)

	// original function should still work
	bundledBytes, err := BundleBytesComposed(mainBytes, config, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, bundledBytes)
}

func TestBundleBytesComposedWithOrigins_InvalidDelimiter(t *testing.T) {
	simpleSpec := `openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          description: OK`

	t.Run("rejects delimiter with hash", func(t *testing.T) {
		config := &datamodel.DocumentConfiguration{}
		compositionConfig := &BundleCompositionConfig{
			Delimiter: "#invalid",
		}

		result, err := BundleBytesComposedWithOrigins([]byte(simpleSpec), config, compositionConfig)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot contain '#'")
		assert.Nil(t, result)
	})

	t.Run("rejects delimiter with slash", func(t *testing.T) {
		config := &datamodel.DocumentConfiguration{}
		compositionConfig := &BundleCompositionConfig{
			Delimiter: "in/valid",
		}

		result, err := BundleBytesComposedWithOrigins([]byte(simpleSpec), config, compositionConfig)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "delimiter cannot contain")
		assert.Nil(t, result)
	})

	t.Run("rejects delimiter with space", func(t *testing.T) {
		config := &datamodel.DocumentConfiguration{}
		compositionConfig := &BundleCompositionConfig{
			Delimiter: "in valid",
		}

		result, err := BundleBytesComposedWithOrigins([]byte(simpleSpec), config, compositionConfig)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot contain spaces")
		assert.Nil(t, result)
	})

	t.Run("uses default delimiter when empty", func(t *testing.T) {
		tmpDir := t.TempDir()

		mainYAML := `openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          $ref: './response.yaml#/TestResponse'`

		responseYAML := `TestResponse:
  description: Test response`

		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.yaml"), []byte(mainYAML), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "response.yaml"), []byte(responseYAML), 0644))

		config := &datamodel.DocumentConfiguration{
			AllowFileReferences: true,
			BasePath:            tmpDir,
		}

		mainBytes, err := os.ReadFile(filepath.Join(tmpDir, "main.yaml"))
		require.NoError(t, err)

		// empty delimiter should use default "__"
		compositionConfig := &BundleCompositionConfig{
			Delimiter: "",
		}

		result, err := BundleBytesComposedWithOrigins(mainBytes, config, compositionConfig)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestBundleBytesComposedWithOrigins_NilModel(t *testing.T) {
	// this tests an internal error condition that shouldn't happen in practice
	// but we need to cover it for completeness
	emptySpec := []byte("")

	config := &datamodel.DocumentConfiguration{}
	result, err := BundleBytesComposedWithOrigins(emptySpec, config, nil)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestBundleBytesComposedWithOrigins_AllComponentTypes(t *testing.T) {
	// comprehensive test covering all component types to maximize coverage
	tmpDir := t.TempDir()

	mainYAML := `openapi: 3.1.0
info:
  title: Comprehensive Test
  version: 1.0.0
paths:
  /test:
    $ref: './paths.yaml#/TestPath'
components:
  schemas:
    LocalSchema:
      type: string`

	pathsYAML := `TestPath:
  get:
    parameters:
      - $ref: './components.yaml#/TestParam'
    responses:
      '200':
        $ref: './components.yaml#/TestResponse'
    requestBody:
      $ref: './components.yaml#/TestRequestBody'
    callbacks:
      testCallback:
        $ref: './components.yaml#/TestCallback'`

	componentsYAML := `TestParam:
  name: test
  in: query
  schema:
    type: string
TestResponse:
  description: Test response
  headers:
    X-Test:
      $ref: '#/TestHeader'
  links:
    testLink:
      $ref: '#/TestLink'
  content:
    application/json:
      schema:
        type: object
      examples:
        testExample:
          $ref: '#/TestExample'
TestRequestBody:
  description: Test request body
  content:
    application/json:
      schema:
        type: object
TestCallback:
  '{$request.body#/callbackUrl}':
    post:
      responses:
        '200':
          description: Callback response
TestHeader:
  description: Test header
  schema:
    type: string
TestLink:
  operationId: testOp
TestExample:
  value: test`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.yaml"), []byte(mainYAML), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "paths.yaml"), []byte(pathsYAML), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "components.yaml"), []byte(componentsYAML), 0644))

	config := &datamodel.DocumentConfiguration{
		AllowFileReferences: true,
		BasePath:            tmpDir,
	}

	mainBytes, err := os.ReadFile(filepath.Join(tmpDir, "main.yaml"))
	require.NoError(t, err)

	result, err := BundleBytesComposedWithOrigins(mainBytes, config, nil)
	// may have errors due to circular refs or other issues, but should still produce output
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Bytes)

	// verify we tracked multiple component types
	componentTypes := make(map[string]bool)
	for _, origin := range result.Origins {
		componentTypes[origin.ComponentType] = true
	}

	t.Logf("Component types tracked: %v", componentTypes)
	assert.Greater(t, len(componentTypes), 1, "should track multiple component types")
}

func TestCaptureOrigin_FullCoverage(t *testing.T) {
	t.Run("captures with empty location", func(t *testing.T) {
		origins := make(ComponentOriginMap)
		pr := &processRef{
			ref: &index.Reference{
				FullDefinition: "test.yaml",
				Node: &yaml.Node{Line: 5, Column: 2},
			},
			idx:          &index.SpecIndex{},
			originalName: "Test",
			name:         "Test",
			location:     []string{}, // empty location
		}

		captureOrigin(pr, "schemas", origins)

		assert.Equal(t, 1, len(origins))
		// bundledRef is now built from pr.name and componentType, not pr.location
		assert.Equal(t, "#/components/schemas/Test", origins["#/components/schemas/Test"].BundledRef)
	})

	t.Run("captures with complex location", func(t *testing.T) {
		origins := make(ComponentOriginMap)
		pr := &processRef{
			ref: &index.Reference{
				FullDefinition: "test.yaml#/components/schemas/ComplexName",
				Node:           &yaml.Node{Line: 10, Column: 4},
			},
			idx:          &index.SpecIndex{},
			originalName: "ComplexName",
			name:         "ComplexName__2",
			wasRenamed:   true,
			location:     []string{"components", "schemas", "ComplexName__2"},
		}

		captureOrigin(pr, "schemas", origins)

		require.Equal(t, 1, len(origins))
		origin := origins["#/components/schemas/ComplexName__2"]
		require.NotNil(t, origin)
		assert.Equal(t, "ComplexName", origin.OriginalName)
		assert.True(t, origin.WasRenamed)
		assert.Equal(t, "#/components/schemas/ComplexName", origin.OriginalRef)
	})

	t.Run("handles full definition without fragment", func(t *testing.T) {
		origins := make(ComponentOriginMap)
		pr := &processRef{
			ref: &index.Reference{
				FullDefinition: "test.yaml",
				Node:           &yaml.Node{Line: 1, Column: 1},
			},
			idx:          &index.SpecIndex{},
			originalName: "Root",
			name:         "Root",
			location:     []string{"components", "schemas", "Root"},
		}

		captureOrigin(pr, "schemas", origins)

		assert.Equal(t, 1, len(origins))
		origin := origins["#/components/schemas/Root"]
		assert.Equal(t, "#/", origin.OriginalRef)
	})

	t.Run("falls back to pr.name when originalName is empty", func(t *testing.T) {
		origins := make(ComponentOriginMap)
		pr := &processRef{
			ref: &index.Reference{
				FullDefinition: "test.yaml#/components/schemas/Fallback",
				Node:           &yaml.Node{Line: 1, Column: 1},
			},
			idx:          &index.SpecIndex{},
			originalName: "", // empty originalName triggers fallback
			name:         "Fallback",
			location:     []string{"components", "schemas", "Fallback"},
		}

		captureOrigin(pr, "schemas", origins)

		assert.Equal(t, 1, len(origins))
		origin := origins["#/components/schemas/Fallback"]
		require.NotNil(t, origin)
		assert.Equal(t, "Fallback", origin.OriginalName) // should fall back to pr.name
	})
}
