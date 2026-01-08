// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package bundler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBundleDocument_ExternalParameterRef tests that external $ref in components.parameters
// are correctly resolved during bundling (Issue #501)
func TestBundleDocument_ExternalParameterRef(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create the main spec with external parameter ref
	mainSpec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
components:
  parameters:
    FilterParam:
      $ref: "./params.yaml#/FilterParam"
paths:
  /test:
    get:
      parameters:
        - $ref: "#/components/parameters/FilterParam"
      responses:
        "200":
          description: OK
`
	// Create the external params file
	paramsFile := `FilterParam:
  name: filter
  in: query
  description: Filter query parameter
  required: false
  schema:
    type: string
`

	// Write files
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "openapi.yaml"), []byte(mainSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "params.yaml"), []byte(paramsFile), 0644))

	// Parse the spec
	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir
	config.AllowFileReferences = true

	specBytes, err := os.ReadFile(filepath.Join(tmpDir, "openapi.yaml"))
	require.NoError(t, err)

	doc, err := libopenapi.NewDocumentWithConfiguration(specBytes, config)
	require.NoError(t, err)

	v3doc, errs := doc.BuildV3Model()
	require.Empty(t, errs)
	require.NotNil(t, v3doc)

	// Bundle the document
	bundledBytes, err := BundleDocument(&v3doc.Model)
	require.NoError(t, err)
	require.NotNil(t, bundledBytes)

	bundledStr := string(bundledBytes)

	// The bundled output should contain the resolved parameter content
	assert.Contains(t, bundledStr, "name: filter", "bundled output should contain resolved parameter name")
	assert.Contains(t, bundledStr, "in: query", "bundled output should contain resolved parameter location")
	assert.Contains(t, bundledStr, "description: Filter query parameter", "bundled output should contain resolved description")

	// The bundled output should NOT contain empty/malformed fields for the parameter
	// Check that FilterParam section contains actual content
	lines := strings.Split(bundledStr, "\n")
	foundFilterParam := false
	for i, line := range lines {
		if strings.Contains(line, "FilterParam:") {
			foundFilterParam = true
			// The next line should NOT be another key at the same indentation level
			// (which would indicate empty content)
			if i+1 < len(lines) {
				nextLine := lines[i+1]
				// Should contain "name:" with proper indentation (content exists)
				assert.Contains(t, nextLine, "name:", "FilterParam should have content, not be empty")
			}
			break
		}
	}
	assert.True(t, foundFilterParam, "bundled output should contain FilterParam section")
}

// TestBundleDocument_ExternalResponseRef tests external $ref in components.responses
func TestBundleDocument_ExternalResponseRef(t *testing.T) {
	tmpDir := t.TempDir()

	mainSpec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
components:
  responses:
    NotFound:
      $ref: "./responses.yaml#/NotFound"
paths:
  /test:
    get:
      responses:
        "404":
          $ref: "#/components/responses/NotFound"
`
	responsesFile := `NotFound:
  description: Resource not found
  content:
    application/json:
      schema:
        type: object
        properties:
          error:
            type: string
`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "openapi.yaml"), []byte(mainSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "responses.yaml"), []byte(responsesFile), 0644))

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir
	config.AllowFileReferences = true

	specBytes, err := os.ReadFile(filepath.Join(tmpDir, "openapi.yaml"))
	require.NoError(t, err)

	doc, err := libopenapi.NewDocumentWithConfiguration(specBytes, config)
	require.NoError(t, err)

	v3doc, errs := doc.BuildV3Model()
	require.Empty(t, errs)

	bundledBytes, err := BundleDocument(&v3doc.Model)
	require.NoError(t, err)

	bundledStr := string(bundledBytes)

	// Verify resolved content is present
	assert.Contains(t, bundledStr, "description: Resource not found")
	assert.Contains(t, bundledStr, "application/json")
}

// TestBundleDocument_ExternalHeaderRef tests external $ref in components.headers
func TestBundleDocument_ExternalHeaderRef(t *testing.T) {
	tmpDir := t.TempDir()

	mainSpec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
components:
  headers:
    RateLimitHeader:
      $ref: "./headers.yaml#/RateLimitHeader"
paths:
  /test:
    get:
      responses:
        "200":
          description: OK
          headers:
            X-Rate-Limit:
              $ref: "#/components/headers/RateLimitHeader"
`
	headersFile := `RateLimitHeader:
  description: Rate limit header
  schema:
    type: integer
`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "openapi.yaml"), []byte(mainSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "headers.yaml"), []byte(headersFile), 0644))

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir
	config.AllowFileReferences = true

	specBytes, err := os.ReadFile(filepath.Join(tmpDir, "openapi.yaml"))
	require.NoError(t, err)

	doc, err := libopenapi.NewDocumentWithConfiguration(specBytes, config)
	require.NoError(t, err)

	v3doc, errs := doc.BuildV3Model()
	require.Empty(t, errs)

	bundledBytes, err := BundleDocument(&v3doc.Model)
	require.NoError(t, err)

	bundledStr := string(bundledBytes)

	assert.Contains(t, bundledStr, "description: Rate limit header")
	assert.Contains(t, bundledStr, "type: integer")
}

// TestBundleDocument_ExternalRequestBodyRef tests external $ref in components.requestBodies
func TestBundleDocument_ExternalRequestBodyRef(t *testing.T) {
	tmpDir := t.TempDir()

	mainSpec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
components:
  requestBodies:
    UserInput:
      $ref: "./request_bodies.yaml#/UserInput"
paths:
  /users:
    post:
      requestBody:
        $ref: "#/components/requestBodies/UserInput"
      responses:
        "201":
          description: Created
`
	requestBodiesFile := `UserInput:
  description: User input data
  required: true
  content:
    application/json:
      schema:
        type: object
        properties:
          name:
            type: string
`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "openapi.yaml"), []byte(mainSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "request_bodies.yaml"), []byte(requestBodiesFile), 0644))

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir
	config.AllowFileReferences = true

	specBytes, err := os.ReadFile(filepath.Join(tmpDir, "openapi.yaml"))
	require.NoError(t, err)

	doc, err := libopenapi.NewDocumentWithConfiguration(specBytes, config)
	require.NoError(t, err)

	v3doc, errs := doc.BuildV3Model()
	require.Empty(t, errs)

	bundledBytes, err := BundleDocument(&v3doc.Model)
	require.NoError(t, err)

	bundledStr := string(bundledBytes)

	assert.Contains(t, bundledStr, "description: User input data")
	assert.Contains(t, bundledStr, "required: true")
}

// TestBundleDocument_ExternalLinkRef tests external $ref in components.links
func TestBundleDocument_ExternalLinkRef(t *testing.T) {
	tmpDir := t.TempDir()

	mainSpec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
components:
  links:
    GetUserById:
      $ref: "./links.yaml#/GetUserById"
paths:
  /users/{id}:
    get:
      operationId: getUser
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          description: OK
          links:
            GetUserById:
              $ref: "#/components/links/GetUserById"
`
	linksFile := `GetUserById:
  operationId: getUser
  description: Get user by ID
  parameters:
    userId: $response.body#/id
`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "openapi.yaml"), []byte(mainSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "links.yaml"), []byte(linksFile), 0644))

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir
	config.AllowFileReferences = true

	specBytes, err := os.ReadFile(filepath.Join(tmpDir, "openapi.yaml"))
	require.NoError(t, err)

	doc, err := libopenapi.NewDocumentWithConfiguration(specBytes, config)
	require.NoError(t, err)

	v3doc, errs := doc.BuildV3Model()
	require.Empty(t, errs)

	bundledBytes, err := BundleDocument(&v3doc.Model)
	require.NoError(t, err)

	bundledStr := string(bundledBytes)

	assert.Contains(t, bundledStr, "operationId: getUser")
	assert.Contains(t, bundledStr, "description: Get user by ID")
}

// TestBundleDocument_ExternalSecuritySchemeRef tests external $ref in components.securitySchemes
func TestBundleDocument_ExternalSecuritySchemeRef(t *testing.T) {
	tmpDir := t.TempDir()

	mainSpec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
components:
  securitySchemes:
    BearerAuth:
      $ref: "./security.yaml#/BearerAuth"
security:
  - BearerAuth: []
paths:
  /test:
    get:
      responses:
        "200":
          description: OK
`
	securityFile := `BearerAuth:
  type: http
  scheme: bearer
  bearerFormat: JWT
  description: JWT Bearer authentication
`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "openapi.yaml"), []byte(mainSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "security.yaml"), []byte(securityFile), 0644))

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir
	config.AllowFileReferences = true

	specBytes, err := os.ReadFile(filepath.Join(tmpDir, "openapi.yaml"))
	require.NoError(t, err)

	doc, err := libopenapi.NewDocumentWithConfiguration(specBytes, config)
	require.NoError(t, err)

	v3doc, errs := doc.BuildV3Model()
	require.Empty(t, errs)

	bundledBytes, err := BundleDocument(&v3doc.Model)
	require.NoError(t, err)

	bundledStr := string(bundledBytes)

	assert.Contains(t, bundledStr, "type: http")
	assert.Contains(t, bundledStr, "scheme: bearer")
	assert.Contains(t, bundledStr, "bearerFormat: JWT")
}

// TestBundleDocument_ExternalExampleRef tests external $ref in components.examples
func TestBundleDocument_ExternalExampleRef(t *testing.T) {
	tmpDir := t.TempDir()

	mainSpec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
components:
  examples:
    UserExample:
      $ref: "./examples.yaml#/UserExample"
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
          content:
            application/json:
              examples:
                user:
                  $ref: "#/components/examples/UserExample"
`
	examplesFile := `UserExample:
  summary: Example user
  description: An example user object
  value:
    id: 123
    name: John Doe
`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "openapi.yaml"), []byte(mainSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "examples.yaml"), []byte(examplesFile), 0644))

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir
	config.AllowFileReferences = true

	specBytes, err := os.ReadFile(filepath.Join(tmpDir, "openapi.yaml"))
	require.NoError(t, err)

	doc, err := libopenapi.NewDocumentWithConfiguration(specBytes, config)
	require.NoError(t, err)

	v3doc, errs := doc.BuildV3Model()
	require.Empty(t, errs)

	bundledBytes, err := BundleDocument(&v3doc.Model)
	require.NoError(t, err)

	bundledStr := string(bundledBytes)

	assert.Contains(t, bundledStr, "summary: Example user")
	assert.Contains(t, bundledStr, "description: An example user object")
}

// TestBundleDocument_ExternalCallbackRef tests external $ref in components.callbacks
func TestBundleDocument_ExternalCallbackRef(t *testing.T) {
	tmpDir := t.TempDir()

	mainSpec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
components:
  callbacks:
    WebhookCallback:
      $ref: "./callbacks.yaml#/WebhookCallback"
paths:
  /subscribe:
    post:
      callbacks:
        onEvent:
          $ref: "#/components/callbacks/WebhookCallback"
      responses:
        "200":
          description: OK
`
	callbacksFile := `WebhookCallback:
  "{$request.body#/callbackUrl}":
    post:
      summary: Webhook event
      requestBody:
        content:
          application/json:
            schema:
              type: object
      responses:
        "200":
          description: OK
`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "openapi.yaml"), []byte(mainSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "callbacks.yaml"), []byte(callbacksFile), 0644))

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir
	config.AllowFileReferences = true

	specBytes, err := os.ReadFile(filepath.Join(tmpDir, "openapi.yaml"))
	require.NoError(t, err)

	doc, err := libopenapi.NewDocumentWithConfiguration(specBytes, config)
	require.NoError(t, err)

	v3doc, errs := doc.BuildV3Model()
	require.Empty(t, errs)

	bundledBytes, err := BundleDocument(&v3doc.Model)
	require.NoError(t, err)

	bundledStr := string(bundledBytes)

	assert.Contains(t, bundledStr, "summary: Webhook event")
	assert.Contains(t, bundledStr, "{$request.body#/callbackUrl}")
}

// TestBundleDocument_ExternalPathItemRef tests external $ref in components.pathItems
func TestBundleDocument_ExternalPathItemRef(t *testing.T) {
	tmpDir := t.TempDir()

	mainSpec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
components:
  pathItems:
    CommonPath:
      $ref: "./path_items.yaml#/CommonPath"
paths:
  /common:
    $ref: "#/components/pathItems/CommonPath"
`
	pathItemsFile := `CommonPath:
  get:
    summary: Common GET operation
    responses:
      "200":
        description: OK
  post:
    summary: Common POST operation
    responses:
      "201":
        description: Created
`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "openapi.yaml"), []byte(mainSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "path_items.yaml"), []byte(pathItemsFile), 0644))

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir
	config.AllowFileReferences = true

	specBytes, err := os.ReadFile(filepath.Join(tmpDir, "openapi.yaml"))
	require.NoError(t, err)

	doc, err := libopenapi.NewDocumentWithConfiguration(specBytes, config)
	require.NoError(t, err)

	v3doc, errs := doc.BuildV3Model()
	require.Empty(t, errs)

	bundledBytes, err := BundleDocument(&v3doc.Model)
	require.NoError(t, err)

	bundledStr := string(bundledBytes)

	assert.Contains(t, bundledStr, "summary: Common GET operation")
	assert.Contains(t, bundledStr, "summary: Common POST operation")
}
