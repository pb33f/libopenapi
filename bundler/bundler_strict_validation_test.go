package bundler

import (
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

	require.Error(t, err, "Strict validation must fail on invalid $ref siblings")
	assert.Contains(
		t,
		err.Error(),
		"invalid OpenAPI specification: $ref cannot have sibling properties",
	)
	assert.Contains(t, err.Error(), "siblings [description]")
	assert.Contains(t, err.Error(), "line 14")
	assert.Contains(t, err.Error(), "column 17")
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

func TestBundleCompositionConfig_DefaultValues(t *testing.T) {
	config := &BundleCompositionConfig{}
	assert.False(t, config.StrictValidation)
	assert.Empty(t, config.Delimiter)
}
