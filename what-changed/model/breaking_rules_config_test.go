// Copyright 2022-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateBreakingRulesConfigYAML_ValidConfig(t *testing.T) {
	yaml := `
schema:
  description:
    added: false
    modified: true
discriminator:
  propertyName:
    modified: false
xml:
  name:
    added: true
`
	result := ValidateBreakingRulesConfigYAML([]byte(yaml))
	assert.Nil(t, result, "Valid config should return nil")
}

func TestValidateBreakingRulesConfigYAML_NestedDiscriminator(t *testing.T) {
	// Nest 'discriminator' under 'response' where it's NOT a valid property
	// (discriminator IS valid under schema, so we use response instead)
	// Validator will report TWO errors:
	// 1. 'discriminator' is misplaced (not a valid property of response)
	// 2. 'propertyName' is a property of discriminator, not response.discriminator
	yaml := `
response:
  discriminator:
    propertyName:
      modified: false
`
	result := ValidateBreakingRulesConfigYAML([]byte(yaml))
	require.NotNil(t, result, "Should detect nested discriminator")
	assert.True(t, result.HasErrors())
	assert.Len(t, result.Errors, 2)

	// First error: misplaced component
	err := result.Errors[0]
	assert.Equal(t, "discriminator", err.FoundKey)
	assert.Equal(t, "response.discriminator", err.Path)
	assert.Equal(t, "discriminator", err.SuggestedPath)
	assert.Contains(t, err.Message, "found 'discriminator' nested under 'response'")
	assert.Greater(t, err.Line, 0)

	// Second error: misplaced property - FoundKey is the misplaced block, not the leaf
	err2 := result.Errors[1]
	assert.Equal(t, "discriminator", err2.FoundKey)
	assert.Contains(t, err2.Message, "'discriminator' is incorrectly nested under 'response'")
	assert.Contains(t, err2.Message, "move your 'discriminator:' block to the top level")
}

func TestValidateBreakingRulesConfigYAML_NestedXML(t *testing.T) {
	// Nest 'xml' under 'response' where it's NOT a valid property
	// Validator will report TWO errors: misplaced component + misplaced property
	yaml := `
response:
  xml:
    name:
      added: true
`
	result := ValidateBreakingRulesConfigYAML([]byte(yaml))
	require.NotNil(t, result)
	assert.Len(t, result.Errors, 2)
	assert.Equal(t, "xml", result.Errors[0].FoundKey)
	assert.Equal(t, "response.xml", result.Errors[0].Path)
	assert.Equal(t, "xml", result.Errors[1].FoundKey) // FoundKey is the misplaced block
	assert.Contains(t, result.Errors[1].Message, "'xml' is incorrectly nested under 'response'")
	assert.Contains(t, result.Errors[1].Message, "move your 'xml:' block to the top level")
}

func TestValidateBreakingRulesConfigYAML_NestedContact(t *testing.T) {
	// Nest 'contact' under 'schema' where it's NOT a valid property
	// Validator will report TWO errors: misplaced component + misplaced property
	yaml := `
schema:
  contact:
    name:
      modified: false
`
	result := ValidateBreakingRulesConfigYAML([]byte(yaml))
	require.NotNil(t, result)
	assert.Len(t, result.Errors, 2)
	assert.Equal(t, "contact", result.Errors[0].FoundKey)
	assert.Equal(t, "schema.contact", result.Errors[0].Path)
	assert.Equal(t, "contact", result.Errors[1].FoundKey) // FoundKey is the misplaced block
	assert.Contains(t, result.Errors[1].Message, "'contact' is incorrectly nested under 'schema'")
	assert.Contains(t, result.Errors[1].Message, "move your 'contact:' block to the top level")
}

func TestValidateBreakingRulesConfigYAML_MultipleErrors(t *testing.T) {
	// Use components that are NOT valid properties of their parents
	// Each misplacement generates 2 errors: component misplaced + property misplaced
	yaml := `
response:
  discriminator:
    propertyName:
      modified: false
  xml:
    name:
      added: true
schema:
  contact:
    name:
      modified: false
  license:
    name:
      modified: true
`
	result := ValidateBreakingRulesConfigYAML([]byte(yaml))
	require.NotNil(t, result)
	assert.Len(t, result.Errors, 8) // 4 components + 4 properties

	// Verify all 4 misplaced components are detected
	// FoundKey is now the misplaced block name (e.g., "discriminator"), not the leaf property
	foundKeys := make(map[string]bool)
	for _, err := range result.Errors {
		foundKeys[err.FoundKey] = true
	}
	assert.True(t, foundKeys["discriminator"])
	assert.True(t, foundKeys["xml"])
	assert.True(t, foundKeys["contact"])
	assert.True(t, foundKeys["license"])
}

func TestValidateBreakingRulesConfigYAML_DeeplyNested(t *testing.T) {
	// Nest 'info' deeply under 'schema.properties.foo' - info is NOT a valid property anywhere
	yaml := `
schema:
  properties:
    foo:
      info:
        title:
          modified: false
`
	result := ValidateBreakingRulesConfigYAML([]byte(yaml))
	require.NotNil(t, result)
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, "info", result.Errors[0].FoundKey)
	assert.Equal(t, "schema.properties.foo.info", result.Errors[0].Path)
}

func TestValidateBreakingRulesConfigYAML_ValidPropertiesNotFlagged(t *testing.T) {
	// These are VALID configurations because 'discriminator', 'xml', 'example', 'externalDocs'
	// ARE valid properties of SchemaRules, and 'contact', 'license' ARE valid properties of InfoRules
	yaml := `
schema:
  discriminator:
    modified: false
  xml:
    added: true
  example:
    removed: false
info:
  contact:
    modified: false
  license:
    added: true
`
	result := ValidateBreakingRulesConfigYAML([]byte(yaml))
	assert.Nil(t, result, "Valid nested properties should not be flagged")
}

func TestValidateBreakingRulesConfigYAML_InvalidYAML(t *testing.T) {
	yaml := `
schema:
  - invalid: yaml
    structure: [
`
	result := ValidateBreakingRulesConfigYAML([]byte(yaml))
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Errors[0].Message, "invalid YAML")
}

func TestValidateBreakingRulesConfigYAML_EmptyConfig(t *testing.T) {
	yaml := ``
	result := ValidateBreakingRulesConfigYAML([]byte(yaml))
	assert.Nil(t, result, "Empty config should be valid")
}

func TestValidateBreakingRulesConfigYAML_OnlyValidTopLevel(t *testing.T) {
	yaml := `
openapi:
  modified: false
info:
  title:
    modified: false
pathItem:
  get:
    removed: true
`
	result := ValidateBreakingRulesConfigYAML([]byte(yaml))
	assert.Nil(t, result, "All top-level components should be valid")
}

func TestValidateBreakingRulesConfigYAML_UnknownKeysIgnored(t *testing.T) {
	yaml := `
schema:
  unknownProperty:
    something: true
  description:
    modified: true
`
	result := ValidateBreakingRulesConfigYAML([]byte(yaml))
	// Unknown properties are not flagged - only known top-level components are flagged when nested
	assert.Nil(t, result, "Unknown keys should not trigger errors")
}

func TestConfigValidationError_Error(t *testing.T) {
	err := &ConfigValidationError{
		Message: "test error message",
		Path:    "schema.discriminator",
		Line:    5,
		Column:  3,
	}
	assert.Equal(t, "test error message (line 5, column 3)", err.Error())
}

func TestConfigValidationError_ErrorNoLineInfo(t *testing.T) {
	err := &ConfigValidationError{
		Message: "test error message",
		Path:    "schema.discriminator",
		Line:    0,
		Column:  0,
	}
	assert.Equal(t, "test error message", err.Error())
}

func TestConfigValidationResult_Error(t *testing.T) {
	result := &ConfigValidationResult{
		Errors: []*ConfigValidationError{
			{Message: "error 1", Line: 2, Column: 3},
			{Message: "error 2", Line: 5, Column: 7},
		},
	}
	expected := "error 1 (line 2, column 3)\nerror 2 (line 5, column 7)"
	assert.Equal(t, expected, result.Error())
}

func TestConfigValidationResult_ErrorEmpty(t *testing.T) {
	result := &ConfigValidationResult{}
	assert.Equal(t, "", result.Error())
}

func TestConfigValidationResult_HasErrors(t *testing.T) {
	result := &ConfigValidationResult{}
	assert.False(t, result.HasErrors())

	result.Errors = []*ConfigValidationError{{Message: "test"}}
	assert.True(t, result.HasErrors())
}

func TestBuildValidComponentSet(t *testing.T) {
	components := buildValidComponentSet()

	// Verify key components are present
	assert.True(t, components["schema"])
	assert.True(t, components["discriminator"])
	assert.True(t, components["xml"])
	assert.True(t, components["info"])
	assert.True(t, components["contact"])
	assert.True(t, components["license"])
	assert.True(t, components["pathItem"])
	assert.True(t, components["operation"])
	assert.True(t, components["parameter"])
	assert.True(t, components["response"])
	assert.True(t, components["mediaType"])
	assert.True(t, components["tag"])
	assert.True(t, components["securityScheme"])

	// Verify internal fields are not included
	assert.False(t, components["ruleCache"])
	assert.False(t, components["cacheOnce"])
}

func TestValidateBreakingRulesConfigYAML_LineNumbers(t *testing.T) {
	// Use 'info' which is NOT a valid property of schema (unlike 'discriminator' which is)
	// Validator reports 2 errors: misplaced component + misplaced property
	yaml := `schema:
  info:
    title:
      modified: false`

	result := ValidateBreakingRulesConfigYAML([]byte(yaml))
	require.NotNil(t, result)
	require.Len(t, result.Errors, 2)

	// info should be on line 2
	assert.Equal(t, 2, result.Errors[0].Line)
	assert.Equal(t, 3, result.Errors[0].Column)
	assert.Equal(t, "info", result.Errors[0].FoundKey)

	// Second error is for "title" being a property of "info" - FoundKey is "info" (the misplaced block)
	assert.Equal(t, 3, result.Errors[1].Line)
	assert.Equal(t, "info", result.Errors[1].FoundKey)
}

func TestValidateBreakingRulesConfigYAML_AllTopLevelComponents(t *testing.T) {
	// Test top-level components that are NOT valid properties of schema
	// Components like 'discriminator', 'xml', 'example', 'externalDocs' ARE valid properties
	// of schema, so they should NOT be flagged when nested under schema.
	componentsNotInSchema := []string{
		"contact", "license", "info",
		"pathItem", "operation", "parameter", "response", "mediaType",
		"header", "encoding", "requestBody", "responses", "tag",
		"securityScheme", "securityRequirement",
		"oauthFlows", "oauthFlow", "callback", "link",
		"server", "serverVariable",
	}

	for _, comp := range componentsNotInSchema {
		yaml := `schema:
  ` + comp + `:
    test: true`

		result := ValidateBreakingRulesConfigYAML([]byte(yaml))
		require.NotNil(t, result, "Should detect nested %s", comp)
		assert.Len(t, result.Errors, 1, "Should have exactly one error for %s", comp)
		assert.Equal(t, comp, result.Errors[0].FoundKey, "FoundKey should be %s", comp)
	}
}

func TestValidateBreakingRulesConfigYAML_ValidNestedProperties(t *testing.T) {
	// Test that properties which ARE valid for a component are NOT flagged
	// 'discriminator', 'xml', 'example', 'externalDocs' are valid properties of schema
	yaml := `schema:
  discriminator:
    modified: false
  xml:
    added: true
  example:
    removed: false
  externalDocs:
    modified: true`

	result := ValidateBreakingRulesConfigYAML([]byte(yaml))
	assert.Nil(t, result, "Valid nested properties should not be flagged")
}

func TestValidateBreakingRulesConfigYAML_BreakingRuleFieldsAtWrongLevel(t *testing.T) {
	// Test that added/modified/removed directly under a component are detected
	// Wrong: paths.added: true
	// Correct: paths.path.added: true
	yaml := `paths:
  added: true
  removed: false`

	result := ValidateBreakingRulesConfigYAML([]byte(yaml))
	require.NotNil(t, result)
	assert.Len(t, result.Errors, 2)

	// Check that both 'added' and 'removed' are flagged
	foundAdded := false
	foundRemoved := false
	for _, err := range result.Errors {
		if err.FoundKey == "added" {
			foundAdded = true
			assert.Contains(t, err.Message, "directly under 'paths'")
			assert.Contains(t, err.Message, "paths.path.added")
		}
		if err.FoundKey == "removed" {
			foundRemoved = true
			assert.Contains(t, err.Message, "directly under 'paths'")
		}
	}
	assert.True(t, foundAdded, "Should detect 'added' at wrong level")
	assert.True(t, foundRemoved, "Should detect 'removed' at wrong level")
}

func TestValidateBreakingRulesConfigYAML_BreakingRuleFieldsAtCorrectLevel(t *testing.T) {
	// Test that added/modified/removed at the correct depth are NOT flagged
	yaml := `paths:
  path:
    added: true
    removed: false
schema:
  type:
    modified: true`

	result := ValidateBreakingRulesConfigYAML([]byte(yaml))
	assert.Nil(t, result, "Should not flag breaking rule fields at correct depth")
}

func TestValidateBreakingRulesConfigYAML_MultipleComponentsWithWrongLevel(t *testing.T) {
	// Test multiple components with breaking rules at wrong level
	yaml := `paths:
  added: true
schema:
  modified: false
pathItem:
  removed: true`

	result := ValidateBreakingRulesConfigYAML([]byte(yaml))
	require.NotNil(t, result)
	assert.Len(t, result.Errors, 3)
}

func TestValidateBreakingRulesConfigYAML_SimpleComponentsAllowDirectRules(t *testing.T) {
	// Test that "simple" components (directly BreakingChangeRule) allow added/modified/removed
	// at depth 1. These components don't have nested properties.
	yaml := `openapi:
  modified: false
jsonSchemaDialect:
  added: true
schemas:
  removed: true
servers:
  modified: false
tags:
  added: false
security:
  removed: false`

	result := ValidateBreakingRulesConfigYAML([]byte(yaml))
	assert.Nil(t, result, "Simple rule components should allow added/modified/removed at depth 1")
}

func TestIsValidPropertyForComponent_UnknownComponent(t *testing.T) {
	// Test that isValidPropertyForComponent returns false for unknown components
	// This covers line 328 in breaking_rules_config.go

	// Test with a completely unknown component name
	result := isValidPropertyForComponent("nonExistentComponent", "someProperty")
	assert.False(t, result, "Unknown component should return false")

	// Test with an empty component name
	result = isValidPropertyForComponent("", "someProperty")
	assert.False(t, result, "Empty component should return false")

	// Test with a known component but invalid property (should also be false)
	result = isValidPropertyForComponent("schema", "nonExistentProperty")
	assert.False(t, result, "Unknown property for known component should return false")

	// Verify a valid component/property combination works
	result = isValidPropertyForComponent("schema", "description")
	assert.True(t, result, "Valid component/property should return true")
}

func TestValidateBreakingRulesConfigYAML_NonScalarKeyNode(t *testing.T) {
	// Test that non-scalar key nodes are skipped (line 362-363)
	// YAML allows complex keys like arrays or maps as keys, though it's rare.
	// This config uses a complex key which should be skipped without error.

	// YAML with a complex key (array as key) - this is valid YAML but unusual
	// The validator should skip this key and not crash
	yamlContent := `
schema:
  description:
    added: true
? [complex, key]
: value
`
	result := ValidateBreakingRulesConfigYAML([]byte(yamlContent))
	// Should not panic and should either be nil or have errors only for valid concerns
	// The complex key should be silently skipped
	if result != nil {
		// If there are errors, they should not be about the complex key
		for _, err := range result.Errors {
			assert.NotContains(t, err.Message, "complex", "Complex keys should be skipped, not flagged")
		}
	}
}

func TestComponentPropertiesMap_Coverage(t *testing.T) {
	// Test that componentProperties map is properly built
	// This indirectly tests buildComponentPropertiesMap including line 307
	// which handles non-struct field types

	// Verify the map exists and has expected components
	assert.NotNil(t, componentProperties, "componentProperties should be initialized")

	// Check some expected components exist
	assert.Contains(t, componentProperties, "schema", "Should have schema component")
	assert.Contains(t, componentProperties, "pathItem", "Should have pathItem component")
	assert.Contains(t, componentProperties, "discriminator", "Should have discriminator component")

	// Check that schema has expected properties
	schemaProps := componentProperties["schema"]
	assert.NotNil(t, schemaProps)
	assert.True(t, schemaProps["description"], "schema should have description property")
	assert.True(t, schemaProps["type"], "schema should have type property")

	// Check discriminator properties
	discProps := componentProperties["discriminator"]
	assert.NotNil(t, discProps)
	assert.True(t, discProps["propertyName"], "discriminator should have propertyName property")
}
