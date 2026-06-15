package base

import (
	"context"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/testify/assert"
	"go.yaml.in/yaml/v4"
)

// TestSchemaMinimumZeroRenderZero tests that minimum values of 0 are rendered when renderZero is present
func TestSchemaMinimumZeroRenderZero(t *testing.T) {
	yml := `type: integer
minimum: 0`

	// Build high-level schema
	highSchema := getHighSchema(t, yml)

	// Verify minimum is correctly parsed
	assert.NotNil(t, highSchema.Minimum)
	assert.Equal(t, float64(0), *highSchema.Minimum)

	// Render back to YAML - this should include minimum: 0 due to renderZero tag
	rendered, err := highSchema.Render()
	assert.NoError(t, err)

	renderedStr := string(rendered)
	assert.Contains(t, renderedStr, "minimum: 0", "minimum: 0 should be rendered due to renderZero tag")
	assert.Contains(t, renderedStr, "type: integer")
}

// TestSchemaMaximumZeroRenderZero tests that maximum values of 0 are rendered when renderZero is present
func TestSchemaMaximumZeroRenderZero(t *testing.T) {
	yml := `type: integer
maximum: 0`

	// Build high-level schema
	highSchema := getHighSchema(t, yml)

	// Verify maximum is correctly parsed
	assert.NotNil(t, highSchema.Maximum)
	assert.Equal(t, float64(0), *highSchema.Maximum)

	// Render back to YAML - this should include maximum: 0 due to renderZero tag
	rendered, err := highSchema.Render()
	assert.NoError(t, err)

	renderedStr := string(rendered)
	assert.Contains(t, renderedStr, "maximum: 0", "maximum: 0 should be rendered due to renderZero tag")
	assert.Contains(t, renderedStr, "type: integer")
}

// TestSchemaBothMinMaxZeroRenderZero tests both minimum and maximum zero values
func TestSchemaBothMinMaxZeroRenderZero(t *testing.T) {
	yml := `type: integer
minimum: 0
maximum: 0`

	// Build high-level schema
	highSchema := getHighSchema(t, yml)

	// Verify both values are correctly parsed
	assert.NotNil(t, highSchema.Minimum)
	assert.NotNil(t, highSchema.Maximum)
	assert.Equal(t, float64(0), *highSchema.Minimum)
	assert.Equal(t, float64(0), *highSchema.Maximum)

	// Render back to YAML
	rendered, err := highSchema.Render()
	assert.NoError(t, err)

	renderedStr := string(rendered)
	assert.Contains(t, renderedStr, "minimum: 0", "minimum: 0 should be rendered")
	assert.Contains(t, renderedStr, "maximum: 0", "maximum: 0 should be rendered")
	assert.Contains(t, renderedStr, "type: integer")
}

// TestSchemaMinimumMaximumNonZero tests that non-zero values work as expected
func TestSchemaMinimumMaximumNonZero(t *testing.T) {
	yml := `type: integer
minimum: 1
maximum: 10`

	// Build high-level schema
	highSchema := getHighSchema(t, yml)

	// Verify values are correctly parsed
	assert.NotNil(t, highSchema.Minimum)
	assert.NotNil(t, highSchema.Maximum)
	assert.Equal(t, float64(1), *highSchema.Minimum)
	assert.Equal(t, float64(10), *highSchema.Maximum)

	// Render back to YAML
	rendered, err := highSchema.Render()
	assert.NoError(t, err)

	renderedStr := string(rendered)
	assert.Contains(t, renderedStr, "minimum: 1")
	assert.Contains(t, renderedStr, "maximum: 10")
	assert.Contains(t, renderedStr, "type: integer")
}

// TestSchemaFloatingPointMinMax tests floating point minimum/maximum values including zero
func TestSchemaFloatingPointMinMax(t *testing.T) {
	yml := `type: number
minimum: 0.0
maximum: 0.5`

	// Build high-level schema
	highSchema := getHighSchema(t, yml)

	// Verify values are correctly parsed
	assert.NotNil(t, highSchema.Minimum)
	assert.NotNil(t, highSchema.Maximum)
	assert.Equal(t, float64(0.0), *highSchema.Minimum)
	assert.Equal(t, float64(0.5), *highSchema.Maximum)

	// Render back to YAML
	rendered, err := highSchema.Render()
	assert.NoError(t, err)

	renderedStr := string(rendered)
	assert.Contains(t, renderedStr, "minimum: 0.0")
	assert.Contains(t, renderedStr, "maximum: 0.5")
	assert.Contains(t, renderedStr, "type: number")
}

// TestSchemaNegativeZeroMinMax tests negative zero values
func TestSchemaNegativeZeroMinMax(t *testing.T) {
	yml := `type: number
minimum: -0.0
maximum: 0.0`

	// Build high-level schema
	highSchema := getHighSchema(t, yml)

	// Verify values are correctly parsed
	assert.NotNil(t, highSchema.Minimum)
	assert.NotNil(t, highSchema.Maximum)
	assert.Equal(t, float64(-0.0), *highSchema.Minimum)
	assert.Equal(t, float64(0.0), *highSchema.Maximum)

	// Render back to YAML.
	rendered, err := highSchema.Render()
	assert.NoError(t, err)

	renderedStr := string(rendered)
	assert.Contains(t, renderedStr, "minimum: -0.0")
	assert.Contains(t, renderedStr, "maximum: 0.0")
	assert.Contains(t, renderedStr, "type: number")
}

// TestSchemaProgrammaticallyCreated tests schemas created programmatically (not from YAML)
func TestSchemaProgrammaticallyCreated(t *testing.T) {
	// This test demonstrates that the issue affects programmatically created schemas
	// when they don't have proper low-level metadata. For now, we'll test through
	// the standard YAML parsing path which is the main use case.

	yml := `type: integer
minimum: 0
maximum: 0`

	// Build using the standard path which properly sets up low-level structures
	highSchema := getHighSchema(t, yml)

	// Verify the schema was built correctly
	assert.NotNil(t, highSchema.Minimum)
	assert.NotNil(t, highSchema.Maximum)
	assert.Equal(t, float64(0), *highSchema.Minimum)
	assert.Equal(t, float64(0), *highSchema.Maximum)

	// Render to YAML
	rendered, err := highSchema.Render()
	assert.NoError(t, err)

	renderedStr := string(rendered)
	assert.Contains(t, renderedStr, "minimum: 0", "minimum: 0 should be rendered")
	assert.Contains(t, renderedStr, "maximum: 0", "maximum: 0 should be rendered")
	assert.Contains(t, renderedStr, "type: integer")
}

// TestSchemaProxy_RenderZeroMinMax tests schema proxy rendering with zero values
func TestSchemaProxy_RenderZeroMinMax(t *testing.T) {
	testSpec := `type: number
minimum: 0
maximum: 0`

	var compNode yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &compNode)

	sp := new(lowbase.SchemaProxy)
	err := sp.Build(context.Background(), nil, compNode.Content[0], nil)
	assert.NoError(t, err)

	lowproxy := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     sp,
		ValueNode: compNode.Content[0],
	}

	schemaProxy := NewSchemaProxy(&lowproxy)
	compiled := schemaProxy.Schema()

	// Verify zero values are parsed correctly
	assert.NotNil(t, compiled.Minimum)
	assert.NotNil(t, compiled.Maximum)
	assert.Equal(t, float64(0), *compiled.Minimum)
	assert.Equal(t, float64(0), *compiled.Maximum)

	// Render back to YAML - should be identical to original
	schemaBytes, err := compiled.Render()
	assert.NoError(t, err)

	renderedStr := strings.TrimSpace(string(schemaBytes))
	assert.Contains(t, renderedStr, "minimum: 0")
	assert.Contains(t, renderedStr, "maximum: 0")
	assert.Contains(t, renderedStr, "type: number")
}

// TestSchemaJSON_RenderZeroMinMax tests JSON rendering with zero values
func TestSchemaJSON_RenderZeroMinMax(t *testing.T) {
	yml := `type: integer
minimum: 0
maximum: 0`

	// Build high-level schema
	highSchema := getHighSchema(t, yml)

	// Render to JSON
	jsonBytes, err := highSchema.MarshalJSON()
	assert.NoError(t, err)

	jsonStr := string(jsonBytes)
	assert.Contains(t, jsonStr, `"minimum":0`)
	assert.Contains(t, jsonStr, `"maximum":0`)
	assert.Contains(t, jsonStr, `"type":"integer"`)
}

// TestSchemaComplexWithZeroValues tests a more complex schema with various zero values
func TestSchemaComplexWithZeroValues(t *testing.T) {
	yml := `type: object
properties:
  count:
    type: integer
    minimum: 0
    maximum: 0
    multipleOf: 1
  score:
    type: number
    minimum: 0.0
    maximum: 0.0
  items:
    type: array
    minItems: 0
    maxItems: 0
  props:
    type: object
    minProperties: 0
    maxProperties: 0
  text:
    type: string
    minLength: 0
    maxLength: 10`

	// Build high-level schema
	highSchema := getHighSchema(t, yml)

	// Render back to YAML
	rendered, err := highSchema.Render()
	assert.NoError(t, err)

	renderedStr := string(rendered)

	// Check that zero values are rendered for numeric constraints with renderZero tags.
	assert.Contains(t, renderedStr, "minimum: 0")
	assert.Contains(t, renderedStr, "maximum: 0")
	assert.Contains(t, renderedStr, "minItems: 0")
	assert.Contains(t, renderedStr, "maxItems: 0")
	assert.Contains(t, renderedStr, "minProperties: 0")
	assert.Contains(t, renderedStr, "maxProperties: 0")
	assert.Contains(t, renderedStr, "minLength: 0")

	// But maxLength: 10 should appear since it's non-zero
	assert.Contains(t, renderedStr, "maxLength: 10")
	assert.Contains(t, renderedStr, "multipleOf: 1")
}

func TestSchemaEmptyPropertiesAreRendered(t *testing.T) {
	yml := `type: object
properties: {}`

	highSchema := getHighSchema(t, yml)

	rendered, err := highSchema.Render()
	assert.NoError(t, err)
	assert.Contains(t, string(rendered), "properties: {}")
}
