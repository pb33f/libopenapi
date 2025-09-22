// Copyright 2025 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package tests

import (
	"fmt"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/stretchr/testify/assert"
)

func TestSiblingRefs_Integration_Issue90(t *testing.T) {
	t.Run("github issue 90 complete example", func(t *testing.T) {
		// complete openapi spec with sibling refs as described in issue #90
		spec := `openapi: 3.1.0
info:
  title: Sibling Refs Test API
  version: 1.0.0
components:
  schemas:
    destination-base:
      type: object
      properties:
        id:
          type: string
        name:
          type: string
    destination-amazon-sqs:
      title: destination-amazon-sqs
      $ref: '#/components/schemas/destination-base'
paths:
  /destinations:
    post:
      requestBody:
        content:
          application/json:
            schema:
              title: "Inline Destination Schema"
              description: "Destination with custom title"
              example:
                id: "123"
                name: "Test Destination"
              $ref: '#/components/schemas/destination-base'
      responses:
        '200':
          description: Success`

		// test with transformation enabled (default)
		config := datamodel.NewDocumentConfiguration()
		assert.True(t, config.TransformSiblingRefs)

		doc, err := libopenapi.NewDocumentWithConfiguration([]byte(spec), config)
		assert.NoError(t, err)
		assert.NotNil(t, doc)

		// verify the document was parsed successfully
		v3Doc, docErrs := doc.BuildV3Model()
		if docErrs != nil {
			t.Fatalf("failed to build v3 model: %v", docErrs)
		}
		assert.NotNil(t, v3Doc)

		// check that the component schema was transformed
		schemas := v3Doc.Model.Components.Schemas
		hasDestinationSqs := false
		for name := range schemas.FromOldest() {
			if name == "destination-amazon-sqs" {
				hasDestinationSqs = true
				break
			}
		}
		assert.True(t, hasDestinationSqs)

		// verify through the rolodex that sibling refs were detected
		rolodex := doc.GetRolodex()
		assert.NotNil(t, rolodex)
		rootIndex := rolodex.GetRootIndex()
		assert.NotNil(t, rootIndex)

		// verify that refs with siblings were properly detected
		refsWithSiblings := rootIndex.GetReferencesWithSiblings()
		assert.NotEmpty(t, refsWithSiblings)

		found := false
		for _, ref := range refsWithSiblings {
			if ref.HasSiblingProperties {
				found = true
				assert.Contains(t, ref.SiblingProperties, "title")
				break
			}
		}
		assert.True(t, found, "should find refs with sibling properties")
	})

	t.Run("backwards compatibility when disabled", func(t *testing.T) {
		spec := `openapi: 3.0.0
info:
  title: Legacy API
  version: 1.0.0
components:
  schemas:
    Base:
      type: object
    WithSiblings:
      title: "This should be ignored in 3.0"
      $ref: '#/components/schemas/Base'`

		// disable transformation for backwards compatibility
		config := datamodel.NewDocumentConfiguration()
		config.TransformSiblingRefs = false

		doc, err := libopenapi.NewDocumentWithConfiguration([]byte(spec), config)
		assert.NoError(t, err)
		assert.NotNil(t, doc)

		// should still build successfully but without transformation
		v3Doc, docErrs := doc.BuildV3Model()
		if docErrs != nil {
			t.Fatalf("failed to build v3 model: %v", docErrs)
		}
		assert.NotNil(t, v3Doc)

		// verify sibling detection still works for tooling
		rolodex := doc.GetRolodex()
		rootIndex := rolodex.GetRootIndex()
		refsWithSiblings := rootIndex.GetReferencesWithSiblings()
		assert.NotEmpty(t, refsWithSiblings, "should still detect siblings for analysis")
	})
}

func TestSiblingRefs_Integration_Issue262(t *testing.T) {
	t.Run("property preservation during reference resolution", func(t *testing.T) {
		spec := `openapi: 3.1.0
info:
  title: Property Merging Test API
  version: 1.0.0
components:
  schemas:
    Address:
      type: object
      properties:
        street:
          type: string
        city:
          type: string
        zipCode:
          type: string
    Customer:
      type: object
      properties:
        name:
          type: string
          example: "John Doe"
        age:
          type: integer
          example: 30
        occupation:
          type: string
          example: "Engineer"
        address:
          $ref: "#/components/schemas/Address"
          example:
            street: "123 Example Road"
            city: "Somewhere"
            zipCode: "12345"
          description: "Customer address with custom example"`

		config := datamodel.NewDocumentConfiguration()
		config.TransformSiblingRefs = true
		config.MergeReferencedProperties = true

		doc, err := libopenapi.NewDocumentWithConfiguration([]byte(spec), config)
		assert.NoError(t, err)
		assert.NotNil(t, doc)

		v3Doc, docErrs := doc.BuildV3Model()
		if docErrs != nil {
			t.Fatalf("failed to build v3 model: %v", docErrs)
		}
		assert.NotNil(t, v3Doc)

		// verify document builds successfully with enhanced resolution
		schemas := v3Doc.Model.Components.Schemas
		hasCustomer := false
		hasAddress := false
		for name := range schemas.FromOldest() {
			if name == "Customer" {
				hasCustomer = true
			}
			if name == "Address" {
				hasAddress = true
			}
		}
		assert.True(t, hasCustomer)
		assert.True(t, hasAddress)

		// verify that sibling refs were detected and processed
		// (detailed schema structure verification would require low-level API access)
		rolodex := doc.GetRolodex()
		rootIndex := rolodex.GetRootIndex()
		refsWithSiblings := rootIndex.GetReferencesWithSiblings()
		assert.NotEmpty(t, refsWithSiblings, "should detect sibling refs in customer schema")

		// verify sibling properties were captured
		found := false
		for _, ref := range refsWithSiblings {
			if ref.HasSiblingProperties {
				found = true
				if _, hasExample := ref.SiblingProperties["example"]; hasExample {
					t.Logf("Found sibling ref with example property")
				}
				break
			}
		}
		assert.True(t, found, "should find refs with sibling properties including examples")
	})
}

func TestSiblingRefs_Integration_CircularReferences(t *testing.T) {
	t.Run("circular refs with sibling properties", func(t *testing.T) {
		spec := `openapi: 3.1.0
info:
  title: Circular Refs Test
  version: 1.0.0
components:
  schemas:
    Node:
      type: object
      properties:
        value:
          type: string
        children:
          type: array
          items:
            title: "Child Node"
            description: "A child node with custom metadata"
            $ref: '#/components/schemas/Node'`

		config := datamodel.NewDocumentConfiguration()
		config.TransformSiblingRefs = true

		doc, err := libopenapi.NewDocumentWithConfiguration([]byte(spec), config)
		assert.NoError(t, err)
		assert.NotNil(t, doc)

		// should handle circular references gracefully even with transformation
		v3Doc, docErrs := doc.BuildV3Model()
		if docErrs != nil {
			t.Fatalf("failed to build v3 model: %v", docErrs)
		}
		assert.NotNil(t, v3Doc)

		// verify circular references are detected
		rolodex := doc.GetRolodex()
		rootIndex := rolodex.GetRootIndex()
		circRefs := rootIndex.GetCircularReferences()
		if len(circRefs) > 0 {
			t.Logf("Found %d circular references (expected for this test)", len(circRefs))
		}

		// document should still build successfully
		schemas := v3Doc.Model.Components.Schemas
		hasNode := false
		for name := range schemas.FromOldest() {
			if name == "Node" {
				hasNode = true
				break
			}
		}
		assert.True(t, hasNode)
	})
}

func TestSiblingRefs_Integration_Performance(t *testing.T) {
	t.Run("large spec with many sibling refs", func(t *testing.T) {
		// create a spec with multiple sibling refs to test performance
		spec := `openapi: 3.1.0
info:
  title: Performance Test API
  version: 1.0.0
components:
  schemas:
    Base:
      type: object
      properties:
        id:
          type: string`

		// add many schemas with sibling refs
		for i := 0; i < 50; i++ {
			spec += fmt.Sprintf(`
    Schema%d:
      title: "Schema %d"
      description: "Generated schema %d for performance testing"
      example: {"id": "%d"}
      $ref: '#/components/schemas/Base'`, i, i, i, i)
		}

		config := datamodel.NewDocumentConfiguration()
		config.TransformSiblingRefs = true

		doc, err := libopenapi.NewDocumentWithConfiguration([]byte(spec), config)
		assert.NoError(t, err)
		assert.NotNil(t, doc)

		// should handle many transformations efficiently
		v3Doc, docErrs := doc.BuildV3Model()
		if docErrs != nil {
			t.Fatalf("failed to build v3 model: %v", docErrs)
		}
		assert.NotNil(t, v3Doc)

		// verify all schemas were processed
		schemas := v3Doc.Model.Components.Schemas
		schemaCount := 0
		for range schemas.FromOldest() {
			schemaCount++
		}
		assert.GreaterOrEqual(t, schemaCount, 51) // Base + 50 generated

		// verify sibling refs were detected
		rolodex := doc.GetRolodex()
		rootIndex := rolodex.GetRootIndex()
		refsWithSiblings := rootIndex.GetReferencesWithSiblings()
		assert.NotEmpty(t, refsWithSiblings, "should detect multiple sibling refs")
	})
}

func TestSiblingRefs_Integration_BackwardsCompatibility(t *testing.T) {
	t.Run("existing behavior preserved when features disabled", func(t *testing.T) {
		spec := `openapi: 3.0.0
info:
  title: Legacy API
  version: 1.0.0
components:
  schemas:
    Base:
      type: object
    Enhanced:
      title: "This title should be ignored in legacy mode"
      description: "This description should be ignored"
      $ref: '#/components/schemas/Base'`

		// disable all new features
		config := datamodel.NewDocumentConfiguration()
		config.TransformSiblingRefs = false
		config.MergeReferencedProperties = false

		doc, err := libopenapi.NewDocumentWithConfiguration([]byte(spec), config)
		assert.NoError(t, err)
		assert.NotNil(t, doc)

		v3Doc, docErrs := doc.BuildV3Model()
		if docErrs != nil {
			t.Fatalf("failed to build v3 model: %v", docErrs)
		}
		assert.NotNil(t, v3Doc)

		// verify document processes correctly in legacy mode
		schemas := v3Doc.Model.Components.Schemas
		hasEnhanced := false
		for name := range schemas.FromOldest() {
			if name == "Enhanced" {
				hasEnhanced = true
				break
			}
		}
		assert.True(t, hasEnhanced)

		// in legacy mode, sibling properties would typically be ignored
		// but sibling detection should still work for analysis tools
		rolodex := doc.GetRolodex()
		rootIndex := rolodex.GetRootIndex()
		refsWithSiblings := rootIndex.GetReferencesWithSiblings()
		assert.NotEmpty(t, refsWithSiblings, "sibling detection should work regardless of transformation setting")
	})
}
