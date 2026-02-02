package bundler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStrictValidation_RefWithSiblings_ShouldError(t *testing.T) {
	spec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/TestSchema'
                description: This is invalid - $ref cannot have siblings
components:
  schemas:
    TestSchema:
      type: object
      properties:
        name:
          type: string`

	config := &BundleCompositionConfig{
		StrictValidation: true,
	}

	docConfig := &datamodel.DocumentConfiguration{
		AllowFileReferences: false,
	}

	_, err := BundleBytesComposed([]byte(spec), docConfig, config)

	require.Error(t, err, "Strict validation must fail on invalid $ref siblings for 3.0 specs")
	assert.Contains(
		t,
		err.Error(),
		"invalid OpenAPI 3.0 specification: $ref cannot have sibling properties",
	)
	assert.Contains(t, err.Error(), "siblings [description]")
	assert.Contains(t, err.Error(), "line 14")
	assert.Contains(t, err.Error(), "column 17")
}

func TestStrictValidation_RefWithSiblings_WithOrigins_ShouldError(t *testing.T) {
	spec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/TestSchema'
                description: This is invalid - $ref cannot have siblings
components:
  schemas:
    TestSchema:
      type: object
      properties:
        name:
          type: string`

	config := &BundleCompositionConfig{
		StrictValidation: true,
	}

	docConfig := &datamodel.DocumentConfiguration{
		AllowFileReferences: false,
	}

	result, err := BundleBytesComposedWithOrigins([]byte(spec), docConfig, config)

	require.Error(t, err, "Strict validation must fail on invalid $ref siblings for 3.0 specs")
	assert.Nil(t, result)
	assert.Contains(
		t,
		err.Error(),
		"invalid OpenAPI 3.0 specification: $ref cannot have sibling properties",
	)
}

func TestStrictValidation_DiscriminatorMappingTarget_ShouldError(t *testing.T) {
	tmpDir := t.TempDir()

	rootSpec := `openapi: 3.0.0
info:
  title: Root
  version: 1.0.0
paths: {}
components:
  schemas:
    Animal:
      type: object
      discriminator:
        propertyName: kind
        mapping:
          bad: './bad.yaml#/components/schemas/Bad'`

	badSpec := `openapi: 3.0.0
info:
  title: Bad
  version: 1.0.0
paths: {}
components:
  schemas:
    Good:
      type: object
    Bad:
      $ref: '#/components/schemas/Good'
      description: invalid sibling`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "root.yaml"), []byte(rootSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "bad.yaml"), []byte(badSpec), 0644))

	docConfig := &datamodel.DocumentConfiguration{
		BasePath:            tmpDir,
		AllowFileReferences: true,
		SpecFilePath:        "root.yaml",
	}

	config := &BundleCompositionConfig{
		StrictValidation: true,
	}

	_, err := BundleBytesComposed([]byte(rootSpec), docConfig, config)

	require.Error(t, err)
	assert.Contains(
		t,
		err.Error(),
		"invalid OpenAPI 3.0 specification: $ref cannot have sibling properties",
	)
}

func TestStrictValidation_DiscriminatorMappingTarget_WithOrigins_ShouldError(t *testing.T) {
	tmpDir := t.TempDir()

	rootSpec := `openapi: 3.0.0
info:
  title: Root
  version: 1.0.0
paths: {}
components:
  schemas:
    Animal:
      type: object
      discriminator:
        propertyName: kind
        mapping:
          bad: './bad.yaml#/components/schemas/Bad'`

	badSpec := `openapi: 3.0.0
info:
  title: Bad
  version: 1.0.0
paths: {}
components:
  schemas:
    Good:
      type: object
    Bad:
      $ref: '#/components/schemas/Good'
      description: invalid sibling`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "root.yaml"), []byte(rootSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "bad.yaml"), []byte(badSpec), 0644))

	docConfig := &datamodel.DocumentConfiguration{
		BasePath:            tmpDir,
		AllowFileReferences: true,
		SpecFilePath:        "root.yaml",
	}

	config := &BundleCompositionConfig{
		StrictValidation: true,
	}

	result, err := BundleBytesComposedWithOrigins([]byte(rootSpec), docConfig, config)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(
		t,
		err.Error(),
		"invalid OpenAPI 3.0 specification: $ref cannot have sibling properties",
	)
}

func TestStrictValidation_RefWithoutSiblings_ShouldSucceed(t *testing.T) {
	spec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
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
          type: string`

	config := &BundleCompositionConfig{
		StrictValidation: true,
	}

	docConfig := &datamodel.DocumentConfiguration{
		AllowFileReferences: false,
	}

	result, err := BundleBytesComposed([]byte(spec), docConfig, config)

	require.NoError(t, err, "Valid $ref without siblings should succeed")
	assert.NotNil(t, result)
	assert.True(t, len(result) > 0)
}

func TestStrictValidation_Disabled_ShouldNotError(t *testing.T) {
	spec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/TestSchema'
                description: This would be invalid with strict validation
components:
  schemas:
    TestSchema:
      type: object
      properties:
        name:
          type: string`

	config := &BundleCompositionConfig{
		StrictValidation: false, // Disabled - should not error
	}

	docConfig := &datamodel.DocumentConfiguration{
		AllowFileReferences: false,
	}

	result, err := BundleBytesComposed([]byte(spec), docConfig, config)

	require.NoError(t, err, "Disabled strict validation should allow invalid siblings")
	assert.NotNil(t, result)
}

func TestStrictValidation_openapi_3_1_ShouldNotError(t *testing.T) {
	spec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/TestSchema'
                description: This is valid with strict validation in 3.1 spec
components:
  schemas:
    TestSchema:
      type: object
      properties:
        name:
          type: string`

	config := &BundleCompositionConfig{
		StrictValidation: true,
	}

	docConfig := &datamodel.DocumentConfiguration{
		AllowFileReferences: false,
	}

	result, err := BundleBytesComposed([]byte(spec), docConfig, config)

	require.NoError(t, err, "Strict validation in OpenAPI 3.1 spec should allow invalid siblings")
	assert.NotNil(t, result)
}

func TestStrictValidation_RecursiveIndexError(t *testing.T) {
	spec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          $ref: 'external.yaml#/components/responses/TestResponse'
components:
  schemas:
    TestSchema:
      type: object`

	external := `openapi: 3.0.0
components:
  responses:
    TestResponse:
      description: OK
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ExternalSchema'
            description: Invalid sibling property
  schemas:
    ExternalSchema:
      type: object
      properties:
        name:
          type: string`

	tmp := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(tmp, "main.yaml"), []byte(spec), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "external.yaml"), []byte(external), 0o644))

	mainBytes, err := os.ReadFile(filepath.Join(tmp, "main.yaml"))
	require.NoError(t, err)

	config := &BundleCompositionConfig{
		StrictValidation: true,
	}

	docConfig := &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	}

	_, err = BundleBytesComposed(mainBytes, docConfig, config)
	require.Error(t, err)

	assert.Contains(
		t,
		err.Error(),
		"invalid OpenAPI 3.0 specification: $ref cannot have sibling properties",
	)
}

func TestBundleCompositionConfig_DefaultValues(t *testing.T) {
	config := &BundleCompositionConfig{}
	assert.False(t, config.StrictValidation)
	assert.Empty(t, config.Delimiter)
}
