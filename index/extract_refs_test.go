// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
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
