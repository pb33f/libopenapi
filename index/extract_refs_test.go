// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"context"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestSpecIndex_ExtractRefs_CheckDescriptionNotMap(t *testing.T) {
	yml := `openapi: 3.1.0
info:
  description: This is a description
paths:
  /herbs/and/spice:
    get:
      description: This is a also a description
      responses:
        200:
          content:
            application/json:
              schema:
                type: array
                properties:
                  description:
                   type: string
   `
	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)
	c := CreateOpenAPIIndexConfig()
	idx := NewSpecIndexWithConfig(&rootNode, c)
	assert.Len(t, idx.allDescriptions, 2)
	assert.Equal(t, 2, idx.descriptionCount)
}

func TestSpecIndex_ExtractRefs_CheckSummarySummary(t *testing.T) {
	yml := `things:
  summary:
    summary:
      - summary`
	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)
	c := CreateOpenAPIIndexConfig()
	idx := NewSpecIndexWithConfig(&rootNode, c)
	assert.Len(t, idx.allSummaries, 3)
	assert.Equal(t, 3, idx.summaryCount)
}

// https://github.com/pb33f/libopenapi/issues/457
func TestSpecIndex_ExtractRefs_SkipSummaryInSchemaProperties(t *testing.T) {
	// Test case for issue #457
	// When a schema has a property named "summary", it should NOT be extracted as a summary description
	yml := `openapi: 3.1.1
info:
  title: Test API
  version: 1.0.0
  summary: This is an API summary
paths:
  /tasks:
    get:
      summary: Get all tasks
      description: Returns all tasks
      responses:
        200:
          description: Successful response
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Task'
components:
  schemas:
    Task:
      type: object
      description: A task object
      properties:
        id:
          type: string
          description: Task ID
        summary:
          type: boolean
          description: Whether this is a summary task
        name:
          type: string
          description: Task name
    Project:
      type: object
      properties:
        summary:
          type: boolean
          description: Project summary flag
        description:
          type: string
          description: Project description text`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)
	c := CreateOpenAPIIndexConfig()
	idx := NewSpecIndexWithConfig(&rootNode, c)

	// Should only capture summaries from info and operations, NOT from schema properties
	assert.Equal(t, 2, idx.summaryCount, "Should only have 2 summaries (info.summary and path operation summary)")

	// Verify that the captured summaries are the correct ones
	summaryContents := []string{}
	for _, summary := range idx.allSummaries {
		summaryContents = append(summaryContents, summary.Content)
	}
	assert.Contains(t, summaryContents, "This is an API summary", "Should contain info.summary")
	assert.Contains(t, summaryContents, "Get all tasks", "Should contain operation summary")

	// Should not contain the boolean property names as summaries
	for _, summary := range idx.allSummaries {
		assert.NotEqual(t, "boolean", summary.Content, "Should not extract schema property type as summary")
	}

	// Check descriptions - should have proper descriptions but not property "description" fields
	descriptionCount := idx.descriptionCount
	assert.Greater(t, descriptionCount, 0, "Should have some descriptions")

	// Verify descriptions are from the right places (API descriptions, not property names)
	descriptionContents := []string{}
	for _, desc := range idx.allDescriptions {
		descriptionContents = append(descriptionContents, desc.Content)
	}
	assert.Contains(t, descriptionContents, "Returns all tasks", "Should contain operation description")
	assert.Contains(t, descriptionContents, "A task object", "Should contain schema description")
}

// https://github.com/pb33f/libopenapi/issues/457
func TestSpecIndex_ExtractRefs_SkipDescriptionInSchemaProperties(t *testing.T) {
	// Test that description properties in schemas are not extracted as API descriptions
	yml := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
  description: Main API description
paths:
  /items:
    get:
      description: Get items operation description
      responses:
        200:
          description: Success response description
          content:
            application/json:
              schema:
                type: object
                properties:
                  description:
                    type: string
                    description: The item's description field
                  title:
                    type: string
                    description: The item's title`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)
	c := CreateOpenAPIIndexConfig()
	idx := NewSpecIndexWithConfig(&rootNode, c)

	// Count descriptions - should not include the "description" property name
	expectedDescriptions := []string{
		"Main API description",
		"Get items operation description",
		"Success response description",
		"The item's description field",
		"The item's title",
	}

	assert.Equal(t, len(expectedDescriptions), idx.descriptionCount,
		"Should only count actual descriptions, not property names")

	// Verify the content
	actualContents := []string{}
	for _, desc := range idx.allDescriptions {
		actualContents = append(actualContents, desc.Content)
	}

	for _, expected := range expectedDescriptions {
		assert.Contains(t, actualContents, expected,
			"Should contain description: %s", expected)
	}
}

// https://github.com/pb33f/libopenapi/issues/457
func TestSpecIndex_ExtractRefs_Issue457_SummaryPropertyConfusion(t *testing.T) {
	// Direct test for GitHub issue #457
	// Schema properties named "summary" should not be confused with API summary fields
	yml := `openapi: 3.1.1
info:
  title: Issue 457 Test
  version: 1.0.0
paths:
  /items:
    get:
      summary: List items
      responses:
        200:
          description: Success
          content:
            application/json:
              examples:
                taskExample:
                  value:
                    id: task-1
                    summary: true
                    name: Important task
                projectExample:
                  value:
                    id: project-1
                    summary: false
                    description: Project description
              schema:
                type: object
                properties:
                  items:
                    type: array
                    items:
                      oneOf:
                        - $ref: '#/components/schemas/Task'
                        - $ref: '#/components/schemas/Project'
components:
  schemas:
    Task:
      type: object
      required:
        - id
        - summary
      properties:
        id:
          type: string
        summary:
          type: boolean
          description: Is this a summary task
        name:
          type: string
    Project:
      type: object
      required:
        - id
        - summary
      properties:
        id:
          type: string
        summary:
          type: boolean
          description: Is this a summary project
        description:
          type: string
          description: The project description`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)
	c := CreateOpenAPIIndexConfig()
	idx := NewSpecIndexWithConfig(&rootNode, c)

	// The key assertion: should only have 1 summary (from the operation)
	// NOT from the schema properties named "summary"
	assert.Equal(t, 1, idx.summaryCount, "Should only extract operation summary, not schema property names")

	if idx.summaryCount > 0 {
		assert.Equal(t, "List items", idx.allSummaries[0].Content, "The only summary should be 'List items'")
	}

	// Check that descriptions are properly counted
	// Should have: "Success", "Is this a summary task", "Is this a summary project", "The project description"
	assert.Equal(t, 4, idx.descriptionCount, "Should have 4 descriptions total")
}

// https://github.com/pb33f/libopenapi/issues/457
func TestSpecIndex_ExtractRefs_SkipSummaryInPatternProperties(t *testing.T) {
	// Test that summary/description in patternProperties are also skipped
	yml := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /items:
    get:
      summary: Get items
      responses:
        200:
          description: Success
          content:
            application/json:
              schema:
                type: object
                patternProperties:
                  "^S_":
                    type: string
                  summary:
                    type: boolean
                    description: Pattern property named summary
                  description:
                    type: string
                    description: Pattern property named description`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)
	c := CreateOpenAPIIndexConfig()
	idx := NewSpecIndexWithConfig(&rootNode, c)

	// Should only have 1 summary from the operation
	assert.Equal(t, 1, idx.summaryCount, "Should only have operation summary, not patternProperties property names")
	assert.Equal(t, "Get items", idx.allSummaries[0].Content)

	// Should have 3 descriptions: "Success", plus the two pattern property descriptions
	assert.Equal(t, 3, idx.descriptionCount, "Should have 3 descriptions")
}

func TestSpecIndex_ExtractRefs_CheckPropertiesForInlineSchema(t *testing.T) {
	yml := `openapi: 3.1.0
servers:
  - url: http://localhost:8080
paths:
  /test:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  test:
                    type: array
                    items:
                      type: object
                    prefixItems:
                      - $ref: '#/components/schemas/Test'
                    additionalProperties: false
                    unevaluatedProperties: false
components:
  schemas:
    Test:
      type: object
      additionalProperties:
        type: string
      contains:
        type: string
      not:
        type: number
      unevaluatedProperties:
        type: boolean
      patternProperties:
        ^S_:
          type: string
        ^I_:
          type: integer
      prefixItems:
        - type: string
    AllOf:
      allOf:
        - type: object
          properties:
            test:
              type: string
        - type: object
          properties:
            test2:
              type: string
    AnyOf:
      anyOf:
        - type: object
          properties:
            test:
              type: string
        - type: object
          properties:
            test2:
              type: string
    OneOf:
      oneOf:
        - type: string
        - type: number
        - type: boolean
   `
	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)
	c := CreateOpenAPIIndexConfig()
	idx := NewSpecIndexWithConfig(&rootNode, c)
	assert.Len(t, idx.allInlineSchemaDefinitions, 21)
	assert.Len(t, idx.allInlineSchemaObjectDefinitions, 7)
}

// https://github.com/pb33f/libopenapi/issues/112
func TestSpecIndex_ExtractRefs_CheckReferencesWithBracketsInName(t *testing.T) {
	yml := `openapi: 3.0.0
components:
  schemas:
    Cake[Burger]:
      type: string
      description: A cakey burger
    Happy:
      type: object
      properties:
        mingo:
          $ref: '#/components/schemas/Cake[Burger]'
   `
	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)
	c := CreateOpenAPIIndexConfig()
	idx := NewSpecIndexWithConfig(&rootNode, c)
	assert.Len(t, idx.allMappedRefs, 1)
	assert.Equal(t, "Cake[Burger]", idx.allMappedRefs["#/components/schemas/Cake[Burger]"].Name)
}

// https://github.com/daveshanley/vacuum/issues/339
func TestSpecIndex_ExtractRefs_CheckEnumNotPropertyCalledEnum(t *testing.T) {
	yml := `openapi: 3.0.0
components:
  schemas:
    SimpleFieldSchema:
      description: Schema of a field as described in  JSON Schema draft 2019-09
      type: object
      required:
        - type
        - description
      properties:
        type:
          type: string
          enum:
            - string
            - number
        description:
          type: string
          description: A description of the property
        enum:
          type: array
          description: A array of describing the possible values
          items:
            type: string
          example:
            - yo
            - hello
    Schema2:
      type: object
      properties:
        enumRef:
          $ref: '#/components/schemas/enum'
        enum:
          type: string
          enum: [big, small]
          nullable: true
    enum:
      type: [string, null]
      enum: [big, small]
   `
	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)
	c := CreateOpenAPIIndexConfig()
	idx := NewSpecIndexWithConfig(&rootNode, c)
	assert.Len(t, idx.allEnums, 3)
}

func TestSpecIndex_ExtractRefs_CheckRefsUnderExtensionsAreNotIncluded(t *testing.T) {
	yml := `openapi: 3.1.0
components:
  schemas:
    Pasta:
      x-hello:
       thing:
         $ref: '404'
   `
	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)
	c := CreateOpenAPIIndexConfig()
	c.ExcludeExtensionRefs = true
	idx := NewSpecIndexWithConfig(&rootNode, c)
	assert.Len(t, idx.allMappedRefs, 0)
	assert.Len(t, idx.allRefs, 0)
	assert.Len(t, idx.refErrors, 0)
}

func TestSpecIndex_ExtractRefs_IsExtensionRef_MarkedCorrectly(t *testing.T) {
	yml := `openapi: 3.1.0
info:
  title: Test
  version: "1.0"
  x-custom:
    $ref: './external.yaml'
paths:
  /test:
    get:
      responses:
        "200":
          $ref: '#/components/responses/OK'
components:
  responses:
    OK:
      description: OK`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	c := CreateClosedAPIIndexConfig()
	c.AvoidCircularReferenceCheck = true

	idx := NewSpecIndexWithConfig(&rootNode, c)

	refs := idx.GetRawReferencesSequenced()

	// Find the extension ref and normal ref
	var extensionRef, normalRef *Reference
	for _, ref := range refs {
		if strings.Contains(ref.FullDefinition, "external.yaml") {
			extensionRef = ref
		}
		if strings.Contains(ref.Definition, "#/components/responses/OK") {
			normalRef = ref
		}
	}

	assert.NotNil(t, extensionRef, "Extension ref should be found")
	assert.True(t, extensionRef.IsExtensionRef, "Extension ref should be marked as IsExtensionRef")

	assert.NotNil(t, normalRef, "Normal ref should be found")
	assert.False(t, normalRef.IsExtensionRef, "Normal ref should NOT be marked as IsExtensionRef")
}

func TestSpecIndex_GetExtensionRefsSequenced(t *testing.T) {
	yml := `openapi: 3.1.0
info:
  title: Test
  version: "1.0"
  x-custom:
    $ref: './ext1.yaml'
  x-another:
    nested:
      $ref: './ext2.yaml'
paths:
  /test:
    get:
      responses:
        "200":
          $ref: '#/components/responses/OK'
components:
  responses:
    OK:
      description: OK`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	c := CreateClosedAPIIndexConfig()
	c.AvoidCircularReferenceCheck = true
	idx := NewSpecIndexWithConfig(&rootNode, c)

	extensionRefs := idx.GetExtensionRefsSequenced()

	assert.Len(t, extensionRefs, 2, "Should find 2 extension refs")
	for _, ref := range extensionRefs {
		assert.True(t, ref.IsExtensionRef, "All returned refs should be extension refs")
	}

	// Verify the total refs include both extension and non-extension
	allRefs := idx.GetRawReferencesSequenced()
	assert.Greater(t, len(allRefs), len(extensionRefs), "Should have more total refs than extension refs")
}

func TestSpecIndex_ExtractRefs_SiblingPropertiesDetection(t *testing.T) {
	yml := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Base:
      type: object
      properties:
        id:
          type: string
    WithSiblings:
      title: "Custom Title"
      description: "Custom Description"
      $ref: "#/components/schemas/Base"
    OnlyRef:
      $ref: "#/components/schemas/Base"`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)
	c := CreateOpenAPIIndexConfig()
	idx := NewSpecIndexWithConfig(&rootNode, c)

	// check that at least one ref with siblings is detected
	assert.GreaterOrEqual(t, len(idx.refsWithSiblings), 1)

	// check that we have the expected refs
	assert.Contains(t, idx.refsWithSiblings, "#/components/schemas/Base")

	// verify the sibling ref properties
	siblingRef := idx.refsWithSiblings["#/components/schemas/Base"]
	assert.True(t, siblingRef.HasSiblingProperties)
	assert.NotEmpty(t, siblingRef.SiblingProperties)

	// should have title and description from WithSiblings
	assert.Contains(t, siblingRef.SiblingProperties, "title")
	assert.Contains(t, siblingRef.SiblingProperties, "description")
	assert.Equal(t, "Custom Title", siblingRef.SiblingProperties["title"].Value)
	assert.Equal(t, "Custom Description", siblingRef.SiblingProperties["description"].Value)
}

func TestSpecIndex_ExtractRefs_SiblingPropertiesVariousTypes(t *testing.T) {
	yml := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Base:
      type: object
    WithMultipleSiblings:
      title: "String Value"
      nullable: true
      example: {"key": "value"}
      enum: ["one", "two", "three"]
      $ref: "#/components/schemas/Base"`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)
	c := CreateOpenAPIIndexConfig()
	idx := NewSpecIndexWithConfig(&rootNode, c)

	// should detect refs with siblings
	assert.GreaterOrEqual(t, len(idx.refsWithSiblings), 1)

	// check the multisibling ref
	if ref, exists := idx.refsWithSiblings["#/components/schemas/Base"]; exists {
		assert.True(t, ref.HasSiblingProperties)
		assert.Equal(t, 4, len(ref.SiblingProperties)) // title, nullable, example, enum
		assert.Contains(t, ref.SiblingProperties, "title")
		assert.Contains(t, ref.SiblingProperties, "nullable")
		assert.Contains(t, ref.SiblingProperties, "example")
		assert.Contains(t, ref.SiblingProperties, "enum")
	}
}

func TestSpecIndex_ExtractRefs_BackwardsCompatibility(t *testing.T) {
	// test that existing behavior is unchanged when TransformSiblingRefs is false
	yml := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Base:
      type: object
    WithSiblings:
      title: "Custom Title"
      $ref: "#/components/schemas/Base"`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)
	c := CreateOpenAPIIndexConfig()
	c.TransformSiblingRefs = false // explicitly disable
	idx := NewSpecIndexWithConfig(&rootNode, c)

	// should still detect siblings for backwards compatibility with existing tooling
	assert.Len(t, idx.refsWithSiblings, 1)

	// check that sibling properties are still captured even when transformation is disabled
	for _, ref := range idx.refsWithSiblings {
		assert.True(t, ref.HasSiblingProperties)
		assert.Contains(t, ref.SiblingProperties, "title")
		assert.Equal(t, "Custom Title", ref.SiblingProperties["title"].Value)
	}
}

func TestSpecIndex_ExtractRefs_LowCPUConcurrencyFloor(t *testing.T) {
	// Set GOMAXPROCS to 2 (less than 4) to trigger the concurrency floor
	oldMaxProcs := runtime.GOMAXPROCS(2)
	defer runtime.GOMAXPROCS(oldMaxProcs)

	// Test spec with multiple references to trigger async processing
	yml := `openapi: 3.1.0
info:
  title: Test
  version: "1.0"
components:
  schemas:
    A:
      type: object
    B:
      $ref: '#/components/schemas/A'
    C:
      $ref: '#/components/schemas/A'
    D:
      $ref: '#/components/schemas/A'
    E:
      $ref: '#/components/schemas/A'`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	c := CreateOpenAPIIndexConfig()
	c.ExtractRefsSequentially = false // Ensure async mode

	idx := NewSpecIndexWithConfig(&rootNode, c)

	// Verify the index was built successfully with refs resolved
	assert.NotNil(t, idx)
	assert.Greater(t, len(idx.GetAllReferences()), 0)
}

func TestSpecIndex_isExternalReference_Nil(t *testing.T) {
	assert.False(t, isExternalReference(nil))
}

func TestUnderOpenAPIExamplePath(t *testing.T) {
	tests := []struct {
		name string
		path []string
		want bool
	}{
		{"empty", nil, false},
		{"no_example_segments", []string{"paths", "get", "responses", "200", "content", "application/json", "schema"}, false},
		{"under_example", []string{"paths", "get", "responses", "200", "content", "application/json", "schema", "example"}, true},
		{"under_examples", []string{"content", "application/json", "schema", "examples", "sample", "value"}, true},
		{"example_not_whole_segment", []string{"paths", "exampled"}, false},
		{"example_as_property_name", []string{"components", "schemas", "Foo", "properties", "example"}, false},
		{"examples_as_property_name", []string{"components", "schemas", "Foo", "properties", "examples"}, false},
		{"nested_under_property_example", []string{"components", "schemas", "Foo", "properties", "example", "properties", "id"}, false},
		{"patternProperties_example", []string{"components", "schemas", "Foo", "patternProperties", "example"}, false},
		{"real_example_after_property_example", []string{"components", "schemas", "Foo", "properties", "example", "example"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, underOpenAPIExamplePath(tt.path))
		})
	}
}

func TestUnderOpenAPIExamplePayloadPath(t *testing.T) {
	tests := []struct {
		name string
		path []string
		want bool
	}{
		{"empty", nil, false},
		{"example_root", []string{"paths", "get", "responses", "200", "content", "application/json", "schema", "example"}, false},
		{"nested_under_example_payload", []string{"components", "schemas", "Foo", "example", "nested"}, true},
		{"examples_collection", []string{"components", "examples"}, false},
		{"example_object_entry", []string{"components", "examples", "ReusableExample"}, false},
		{"examples_value_payload", []string{"content", "application/json", "examples", "sample", "value"}, true},
		{"examples_value_nested_payload", []string{"content", "application/json", "examples", "sample", "value", "nested"}, true},
		{"examples_data_value_payload", []string{"components", "examples", "sample", "dataValue"}, true},
		{"property_named_example", []string{"components", "schemas", "Foo", "properties", "example"}, false},
		{"property_named_examples_value", []string{"components", "schemas", "Foo", "properties", "examples", "value"}, false},
		{"real_example_after_property_example", []string{"components", "schemas", "Foo", "properties", "example", "example"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, underOpenAPIExamplePayloadPath(tt.path))
		})
	}
}

func TestIsOpenAPIExampleKeywordSegment(t *testing.T) {
	path := []string{"components", "examples", "ReusableExample"}

	tests := []struct {
		name string
		idx  int
		want bool
	}{
		{"negative index", -1, false},
		{"index too large", len(path), false},
		{"examples keyword", 1, true},
		{"non keyword segment", 2, false},
		{"property named example", 2, false},
	}

	propertyPath := []string{"components", "schemas", "Foo", "properties", "example"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetPath := path
			if tt.name == "property named example" {
				targetPath = propertyPath
			}
			assert.Equal(t, tt.want, isOpenAPIExampleKeywordSegment(targetPath, tt.idx))
		})
	}
}

func TestExtractRefs_InlineSchemaHelpers(t *testing.T) {
	idx := NewTestSpecIndex().Load().(*SpecIndex)
	idx.specAbsolutePath = "test.yaml"

	var inlineNode yaml.Node
	_ = yaml.Unmarshal([]byte(`additionalProperties: true`), &inlineNode)
	idx.collectInlineSchemaDefinition(nil, inlineNode.Content[0], []string{"components", "schemas", "Pet"}, 0)
	assert.Empty(t, idx.allInlineSchemaDefinitions)

	var refNode yaml.Node
	_ = yaml.Unmarshal([]byte(`schema:
  $ref: '#/components/schemas/Pet'`), &refNode)
	idx.collectInlineSchemaDefinition(nil, refNode.Content[0], []string{"paths", "/pets"}, 0)
	assert.Len(t, idx.allRefSchemaDefinitions, 1)

	before := len(idx.allInlineSchemaDefinitions)
	idx.collectInlineSchemaDefinition(nil, refNode.Content[0], []string{"paths"}, 1)
	assert.Len(t, idx.allInlineSchemaDefinitions, before)
}

func TestExtractRefs_MapAndArraySchemaHelpers(t *testing.T) {
	idx := NewTestSpecIndex().Load().(*SpecIndex)
	idx.specAbsolutePath = "test.yaml"

	var propsNode yaml.Node
	_ = yaml.Unmarshal([]byte(`properties:
  foo:
    $ref: '#/components/schemas/Pet'
  bar:
    type: object`), &propsNode)
	idx.collectMapSchemaDefinitions(nil, propsNode.Content[0], []string{"components", "schemas", "Thing"}, 0)
	assert.Len(t, idx.allRefSchemaDefinitions, 1)
	assert.Len(t, idx.allInlineSchemaDefinitions, 1)
	assert.Len(t, idx.allInlineSchemaObjectDefinitions, 1)

	inlineBefore := len(idx.allInlineSchemaDefinitions)
	idx.collectMapSchemaDefinitions(nil, propsNode.Content[0], []string{"examples"}, 0)
	assert.Len(t, idx.allInlineSchemaDefinitions, inlineBefore)
	idx.collectMapSchemaDefinitions(nil, propsNode.Content[0], []string{"x-test"}, 0)
	assert.Len(t, idx.allInlineSchemaDefinitions, inlineBefore)

	var arrayNode yaml.Node
	_ = yaml.Unmarshal([]byte(`oneOf:
  - $ref: '#/components/schemas/Pet'
  - type: string`), &arrayNode)
	idx.collectArraySchemaDefinitions(nil, arrayNode.Content[0], nil, 0)
	assert.Len(t, idx.allRefSchemaDefinitions, 2)
	assert.Len(t, idx.allInlineSchemaDefinitions, 2)

	idx.collectArraySchemaDefinitions(nil, arrayNode.Content[0], nil, 1)
	assert.Len(t, idx.allRefSchemaDefinitions, 2)
	idx.collectMapSchemaDefinitions(nil, propsNode.Content[0], nil, 1)
	idx.collectArraySchemaDefinitions(nil, arrayNode.Content[0], nil, 2)
}

func TestInlineSchemaIsObjectOrArray(t *testing.T) {
	assert.False(t, inlineSchemaIsObjectOrArray(nil))

	var noType yaml.Node
	_ = yaml.Unmarshal([]byte(`description: nope`), &noType)
	assert.False(t, inlineSchemaIsObjectOrArray(noType.Content[0]))

	var objectNode yaml.Node
	_ = yaml.Unmarshal([]byte(`type: object`), &objectNode)
	assert.True(t, inlineSchemaIsObjectOrArray(objectNode.Content[0]))

	var arrayNode yaml.Node
	_ = yaml.Unmarshal([]byte(`type: array`), &arrayNode)
	assert.True(t, inlineSchemaIsObjectOrArray(arrayNode.Content[0]))
}

func TestRegisterSchemaIDAt_HelperBranches(t *testing.T) {
	idx := NewTestSpecIndex().Load().(*SpecIndex)
	idx.specAbsolutePath = "test.yaml"

	var invalidNode yaml.Node
	_ = yaml.Unmarshal([]byte(`$id: '#/bad'`), &invalidNode)
	idx.registerSchemaIDAt(invalidNode.Content[0], 0, []string{"components", "schemas", "Pet"}, "test.yaml")
	assert.NotEmpty(t, idx.refErrors)

	var nonStringNode yaml.Node
	_ = yaml.Unmarshal([]byte(`$id:
  type: string`), &nonStringNode)
	errorsBefore := len(idx.refErrors)
	idx.registerSchemaIDAt(nonStringNode.Content[0], 0, nil, "test.yaml")
	assert.Len(t, idx.refErrors, errorsBefore)

	var fallbackNode yaml.Node
	_ = yaml.Unmarshal([]byte(`$id: schema.json`), &fallbackNode)
	idx.registerSchemaIDAt(fallbackNode.Content[0], 0, []string{"components", "schemas", "Pet"}, "://bad-base")
	entry := idx.schemaIdRegistry["schema.json"]
	assert.NotNil(t, entry)
	assert.Equal(t, "schema.json", entry.ResolvedUri)
	assert.Equal(t, "://bad-base", entry.ParentId)
}

func TestExtractRefs_MetadataHelpers(t *testing.T) {
	idx := NewTestSpecIndex().Load().(*SpecIndex)
	idx.specAbsolutePath = "test.yaml"

	var emptyNode yaml.Node
	_ = yaml.Unmarshal([]byte(`$ref: '#/components/schemas/Pet'`), &emptyNode)
	action := idx.extractNodeMetadata(emptyNode.Content[0], nil, nil, 0)
	assert.False(t, action.appendSegment)
	assert.False(t, action.stop)

	seqNode := &yaml.Node{
		Kind: yaml.SequenceNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "description"},
		},
	}
	action = idx.extractNodeMetadata(seqNode, nil, nil, 0)
	assert.True(t, action.stop)
	assert.False(t, action.appendSegment)

	var descNode yaml.Node
	_ = yaml.Unmarshal([]byte(`description: hello`), &descNode)
	action = idx.extractNodeMetadata(descNode.Content[0], nil, []string{"properties"}, 0)
	assert.True(t, action.stop)
	assert.True(t, action.appendSegment)

	var summaryNode yaml.Node
	_ = yaml.Unmarshal([]byte(`summary: hello`), &summaryNode)
	action = idx.extractNodeMetadata(summaryNode.Content[0], nil, []string{"examples"}, 0)
	assert.True(t, action.stop)
	assert.False(t, action.appendSegment)

	var securityScalar yaml.Node
	_ = yaml.Unmarshal([]byte(`security: nope`), &securityScalar)
	action = idx.extractNodeMetadata(securityScalar.Content[0], nil, nil, 0)
	assert.True(t, action.appendSegment)
	assert.Empty(t, idx.securityRequirementRefs)

	var securityNode yaml.Node
	_ = yaml.Unmarshal([]byte(`security:
  - apiKey:
      - read
      - write
    oauth:
      - admin`), &securityNode)
	action = idx.extractNodeMetadata(securityNode.Content[0], nil, []string{"paths", "/pets"}, 0)
	assert.True(t, action.appendSegment)
	assert.Len(t, idx.securityRequirementRefs["apiKey"]["read"], 1)
	assert.Len(t, idx.securityRequirementRefs["apiKey"]["write"], 1)
	assert.Len(t, idx.securityRequirementRefs["oauth"]["admin"], 1)

	var securityAppendNode yaml.Node
	_ = yaml.Unmarshal([]byte(`security:
  - apiKey:
      - read`), &securityAppendNode)
	idx.collectSecurityRequirementMetadata(securityAppendNode.Content[0], 0, "$.paths./pets.security")
	assert.Len(t, idx.securityRequirementRefs["apiKey"]["read"], 2)

	var securitySkipNode yaml.Node
	_ = yaml.Unmarshal([]byte(`security:
  - skip-me
  - apiKey: read
  - apiKey:
      - admin`), &securitySkipNode)
	idx.collectSecurityRequirementMetadata(securitySkipNode.Content[0], 0, "$.paths./pets.security")
	assert.Len(t, idx.securityRequirementRefs["apiKey"]["admin"], 1)

	var enumPropertyNode yaml.Node
	_ = yaml.Unmarshal([]byte(`type: string
enum:
  - one`), &enumPropertyNode)
	action = idx.extractNodeMetadata(enumPropertyNode.Content[0], nil, []string{"properties"}, 2)
	assert.True(t, action.stop)
	assert.True(t, action.appendSegment)

	var enumNoType yaml.Node
	_ = yaml.Unmarshal([]byte(`enum:
  - one`), &enumNoType)
	idx.collectEnumMetadata(enumNoType.Content[0], nil, 0, "$.enum")
	assert.Empty(t, idx.allEnums)
	idx.collectEnumMetadata(enumNoType.Content[0], nil, 1, "$.enum")

	var enumWithType yaml.Node
	_ = yaml.Unmarshal([]byte(`type: string
enum:
  - one`), &enumWithType)
	idx.collectEnumMetadata(enumWithType.Content[0], nil, 2, "$.enum")
	assert.Len(t, idx.allEnums, 1)

	var objectProps yaml.Node
	_ = yaml.Unmarshal([]byte(`type:
  - string
  - object
properties:
  name:
    type: string`), &objectProps)
	action = idx.extractNodeMetadata(objectProps.Content[0], nil, []string{"components", "schemas", "Pet"}, 2)
	assert.True(t, action.appendSegment)
	assert.Len(t, idx.allObjectsWithProperties, 1)

	metadataFallbackNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "summary"},
		},
	}
	assert.Equal(t, "summary", metadataValueNode(metadataFallbackNode, 0).Value)
}

func TestExtractRefs_WalkHelpers(t *testing.T) {
	idx := NewTestSpecIndex().Load().(*SpecIndex)
	state := idx.initializeExtractRefsState(context.Background(), nil, nil, 0, false, "")

	node := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			nil,
		},
	}
	var found []*Reference
	assert.False(t, idx.handleExtractRefsKey(node, nil, &state, 0, &found))
	assert.Empty(t, found)

	assert.False(t, shouldSkipMapSchemaCollection(nil))
	assert.True(t, shouldSkipMapSchemaCollection([]string{"example"}))
	assert.False(t, shouldSkipMapSchemaCollection([]string{"properties", "example"}))
	assert.False(t, shouldSkipMapSchemaCollection([]string{"patternProperties", "example"}))
	assert.True(t, shouldSkipMapSchemaCollection([]string{"x-test"}))

	var noAppendNode yaml.Node
	_ = yaml.Unmarshal([]byte("summary: hello\nvalue: world"), &noAppendNode)
	state.seenPath = []string{"components", "examples", "sample"}
	state.lastAppended = false
	idx.unwindExtractRefsPath(noAppendNode.Content[0], &state, 1)
	assert.Equal(t, []string{"components", "examples", "sample"}, state.seenPath)

	var appendNode yaml.Node
	_ = yaml.Unmarshal([]byte("value: hello\nnext: world"), &appendNode)
	state.lastAppended = true
	idx.unwindExtractRefsPath(appendNode.Content[0], &state, 1)
	assert.Equal(t, []string{"components", "examples"}, state.seenPath)
	assert.False(t, state.lastAppended)
}

func TestExtractReferenceAt_IgnoresRefsInsideExamplePayloads(t *testing.T) {
	idx := NewTestSpecIndex().Load().(*SpecIndex)
	idx.specAbsolutePath = "test.yaml"

	var refNode yaml.Node
	_ = yaml.Unmarshal([]byte(`$ref: '#/components/schemas/Pet'`), &refNode)

	ref := idx.extractReferenceAt(refNode.Content[0], nil, 0, []string{"components", "examples", "sample", "value"}, nil, false, "")
	assert.Nil(t, ref)
	assert.Empty(t, idx.GetAllReferences())
	assert.Empty(t, idx.GetAllSequencedReferences())
}

func TestSpecIndex_ExtractRefs_ExampleObjectRefsIndexedButPayloadRefsIgnored(t *testing.T) {
	spec := `openapi: 3.2.0
info:
  title: Example refs
  version: 1.0.0
paths:
  /widgets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              examples:
                responseRef:
                  $ref: '#/components/examples/ReusableExample'
                inlinePayload:
                  summary: payload example
                  value:
                    nested:
                      $ref: '#/components/schemas/ShouldNotIndex'
components:
  examples:
    ReusableExample:
      $ref: '#/components/examples/LeafExample'
    LeafExample:
      summary: reusable
      value:
        ok: true
    DataValueExample:
      dataValue:
        nested:
          $ref: '#/components/schemas/ShouldNotIndexData'
  schemas:
    ShouldNotIndex:
      type: object
    ShouldNotIndexData:
      type: object
`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(spec), &rootNode)

	idx := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())

	rawRefs := make(map[string]bool)
	for _, ref := range idx.GetAllReferences() {
		rawRefs[ref.RawRef] = true
	}

	assert.True(t, rawRefs["#/components/examples/ReusableExample"])
	assert.True(t, rawRefs["#/components/examples/LeafExample"])
	assert.False(t, rawRefs["#/components/schemas/ShouldNotIndex"])
	assert.False(t, rawRefs["#/components/schemas/ShouldNotIndexData"])
}

func TestSpecIndex_ExtractRefs_SchemaExampleRefIndexedButNestedPayloadRefsIgnored(t *testing.T) {
	spec := `openapi: 3.2.0
info:
  title: Schema example refs
  version: 1.0.0
components:
  schemas:
    UsesExampleRef:
      type: object
      example:
        $ref: '#/components/examples/ReusableExample'
    InlineExamplePayload:
      type: object
      example:
        nested:
          $ref: '#/components/schemas/ShouldNotIndex'
    ShouldNotIndex:
      type: object
  examples:
    ReusableExample:
      value:
        ok: true
`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(spec), &rootNode)

	idx := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())

	rawRefs := make(map[string]bool)
	for _, ref := range idx.GetAllReferences() {
		rawRefs[ref.RawRef] = true
	}

	assert.True(t, rawRefs["#/components/examples/ReusableExample"])
	assert.False(t, rawRefs["#/components/schemas/ShouldNotIndex"])
}
