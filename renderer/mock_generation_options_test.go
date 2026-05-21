// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package renderer

import (
	"encoding/json"
	"errors"
	"math/rand"
	"strconv"
	"testing"

	"github.com/pb33f/libopenapi"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestNormalizeMockGenerationOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input    MockGenerationOptions
		expected MockGenerationOptions
	}{
		"defaults": {
			input: MockGenerationOptions{},
			expected: MockGenerationOptions{
				MaxPatternRepeatBudget:  DefaultMaxPatternRepeatBudget,
				MaxGeneratedStringBytes: DefaultMaxGeneratedStringBytes,
				MaxMockDepth:            DefaultMaxMockDepth,
				MaxMockNodes:            DefaultMaxMockNodes,
				MaxMockProperties:       DefaultMaxMockProperties,
				MaxMockRefExpansions:    DefaultMaxMockRefExpansions,
				MaxMockBytes:            DefaultMaxMockBytes,
			},
		},
		"custom": {
			input: MockGenerationOptions{
				MaxPatternRepeatBudget:  7,
				MaxGeneratedStringBytes: 128,
				MaxMockDepth:            8,
				MaxMockNodes:            9,
				MaxMockProperties:       10,
				MaxMockRefExpansions:    11,
				MaxMockBytes:            12,
			},
			expected: MockGenerationOptions{
				MaxPatternRepeatBudget:  7,
				MaxGeneratedStringBytes: 128,
				MaxMockDepth:            8,
				MaxMockNodes:            9,
				MaxMockProperties:       10,
				MaxMockRefExpansions:    11,
				MaxMockBytes:            12,
			},
		},
		"partial defaults": {
			input: MockGenerationOptions{
				MaxPatternRepeatBudget:  -1,
				MaxGeneratedStringBytes: 256,
				MaxMockDepth:            -1,
				MaxMockProperties:       10,
			},
			expected: MockGenerationOptions{
				MaxPatternRepeatBudget:  DefaultMaxPatternRepeatBudget,
				MaxGeneratedStringBytes: 256,
				MaxMockDepth:            DefaultMaxMockDepth,
				MaxMockNodes:            DefaultMaxMockNodes,
				MaxMockProperties:       10,
				MaxMockRefExpansions:    DefaultMaxMockRefExpansions,
				MaxMockBytes:            DefaultMaxMockBytes,
			},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, normalizeMockGenerationOptions(tc.input))
		})
	}
}

func TestSchemaRenderer_EffectiveMockGenerationOptions_DefaultsNilRenderer(t *testing.T) {
	t.Parallel()

	var renderer *SchemaRenderer

	assert.Equal(t, normalizeMockGenerationOptions(MockGenerationOptions{}), renderer.effectiveMockGenerationOptions())
}

func TestBoundedGeneratedStringRange(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		min      int64
		max      int64
		maxBytes int
		wantMin  int64
		wantMax  int64
	}{
		"disabled": {
			min:     3,
			max:     10,
			wantMin: 3,
			wantMax: 10,
		},
		"caps max": {
			min:      3,
			max:      100,
			maxBytes: 12,
			wantMin:  3,
			wantMax:  12,
		},
		"caps min and max": {
			min:      100,
			max:      200,
			maxBytes: 12,
			wantMin:  12,
			wantMax:  12,
		},
		"raises max to capped min": {
			min:      10,
			max:      4,
			maxBytes: 8,
			wantMin:  8,
			wantMax:  8,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gotMin, gotMax := boundedGeneratedStringRange(tc.min, tc.max, tc.maxBytes)
			assert.Equal(t, tc.wantMin, gotMin)
			assert.Equal(t, tc.wantMax, gotMax)
		})
	}
}

func TestTruncateStringBytes(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "abcdef", truncateStringBytes("abcdef", 0))
	assert.Equal(t, "abc", truncateStringBytes("abcdef", 3))
	assert.Equal(t, "abc", truncateStringBytes("abc", 3))
	assert.Empty(t, truncateStringBytes("éclair", 1))
	assert.Equal(t, "é", truncateStringBytes("éclair", 2))
}

func TestSchemaRenderer_SetMockGenerationOptions_CapsGeneratedStringBytes(t *testing.T) {
	t.Parallel()

	renderer := emptyDictionarySchemaRenderer(1)
	renderer.SetMockGenerationOptions(MockGenerationOptions{MaxGeneratedStringBytes: 12})

	value := renderStringSchema(t, renderer, `type: string
minLength: 100
maxLength: 100`)

	assert.Len(t, value, 12)
}

func TestSchemaRenderer_SetMockGenerationOptions_BoundsAWSARNPattern(t *testing.T) {
	t.Parallel()

	renderer := emptyDictionarySchemaRenderer(1)
	renderer.SetMockGenerationOptions(MockGenerationOptions{
		MaxPatternRepeatBudget:  2,
		MaxGeneratedStringBytes: 128,
	})

	value := renderStringSchema(t, renderer, `type: string
pattern: 'arn:(aws|aws-us-gov|aws-cn|aws-iso|aws-iso-b):iam::[0-9]{12}:(role|role/service-role)/[\w+=,.@/-]{1,1000}'
maxLength: 1024`)

	assert.NotEmpty(t, value)
	assert.LessOrEqual(t, len(value), 128)
	assert.Contains(t, value, "arn:")
}

func TestSchemaRenderer_SetMockGenerationOptions_UsesSchemaMaxLengthWhenSmallerThanRepeatBudget(t *testing.T) {
	t.Parallel()

	renderer := emptyDictionarySchemaRenderer(1)
	renderer.SetMockGenerationOptions(MockGenerationOptions{
		MaxPatternRepeatBudget:  32,
		MaxGeneratedStringBytes: 128,
	})

	value := renderStringSchema(t, renderer, `type: string
pattern: '[a-z]{0,100}'
maxLength: 2`)

	assert.LessOrEqual(t, len(value), 2)
}

func TestSchemaRenderer_SetMockGenerationOptions_InvalidPatternFallsBackToBoundedWord(t *testing.T) {
	t.Parallel()

	renderer := emptyDictionarySchemaRenderer(1)
	renderer.SetMockGenerationOptions(MockGenerationOptions{MaxGeneratedStringBytes: 8})

	value := renderStringSchema(t, renderer, `type: string
pattern: '['
minLength: 20
maxLength: 20`)

	assert.Len(t, value, 8)
}

func TestMockGenerator_SetMockGenerationOptions(t *testing.T) {
	t.Parallel()

	fake := createFakeMock(`type: object
required: [arn]
properties:
  arn:
    type: string
    pattern: 'arn:(aws|aws-us-gov|aws-cn|aws-iso|aws-iso-b):iam::[0-9]{12}:(role|role/service-role)/[\w+=,.@/-]{1,1000}'
    maxLength: 1024`, nil, nil)

	mg := NewMockGenerator(JSON)
	mg.SetMockGenerationOptions(MockGenerationOptions{
		MaxPatternRepeatBudget:  1,
		MaxGeneratedStringBytes: 64,
	})

	mock, err := mg.GenerateMock(fake, "")
	require.NoError(t, err)

	var payload map[string]string
	require.NoError(t, json.Unmarshal(mock, &payload))
	assert.NotEmpty(t, payload["arn"])
	assert.LessOrEqual(t, len(payload["arn"]), 64)
}

func TestMockGenerator_GenerationBudgetExceeded(t *testing.T) {
	t.Parallel()

	fake := createFakeMock(`type: object
properties:
  alpha:
    type: string
  beta:
    type: string`, nil, nil)

	mg := NewMockGenerator(JSON)
	mg.SetMockGenerationOptions(MockGenerationOptions{MaxMockProperties: 1})

	mock, err := mg.GenerateMock(fake, "")

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrMockGenerationBudgetExceeded))
	assert.Nil(t, mock)
}

func TestSchemaRenderer_RenderSchemaWithErrorBudgetExceeded(t *testing.T) {
	t.Parallel()

	renderer := emptyDictionarySchemaRenderer(1)
	schema := schemaWithStringProperties(DefaultMaxMockProperties + 1)

	rendered, err := renderer.RenderSchemaWithError(schema)

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrMockGenerationBudgetExceeded))
	assert.Nil(t, rendered)
}

func TestSchemaRenderer_RenderSchemaBestEffortPastBudget(t *testing.T) {
	t.Parallel()

	renderer := emptyDictionarySchemaRenderer(1)
	schema := schemaWithStringProperties(DefaultMaxMockProperties + 1)

	rendered := renderer.RenderSchema(schema)

	require.NotNil(t, rendered)
	payload, ok := rendered.(map[string]any)
	require.True(t, ok)
	assert.Len(t, payload, DefaultMaxMockProperties+1)
}

func TestMockGenerator_RenderContextUsesCompletedRefCache(t *testing.T) {
	t.Parallel()

	schema := componentSchemaForMockTest(t, "Root", `openapi: 3.1.0
info:
  title: cache
  version: 1.0.0
paths: {}
components:
  schemas:
    Root:
      type: object
      properties:
        leafA:
          $ref: '#/components/schemas/Leaf'
        leafB:
          $ref: '#/components/schemas/Leaf'
    Leaf:
      type: object
      properties:
        name:
          type: string`)

	mg := NewMockGenerator(JSON)
	mg.SetSeed(1)
	mg.SetMockGenerationOptions(MockGenerationOptions{MaxMockRefExpansions: 1})

	mock, err := mg.GenerateMock(schema, "")

	require.NoError(t, err)
	var payload map[string]map[string]string
	require.NoError(t, json.Unmarshal(mock, &payload))
	assert.NotEmpty(t, payload["leafA"]["name"])
	assert.NotEmpty(t, payload["leafB"]["name"])
}

func TestMockGenerator_RenderContextCacheSeparatesArrayItemsFromScalar(t *testing.T) {
	t.Parallel()

	schema := componentSchemaForMockTest(t, "Root", `openapi: 3.1.0
info:
  title: cache-shape
  version: 1.0.0
paths: {}
components:
  schemas:
    Root:
      type: object
      properties:
        list:
          type: array
          items:
            $ref: '#/components/schemas/Code'
        scalar:
          $ref: '#/components/schemas/Code'
    Code:
      type: string
      examples:
        - one
        - two`)

	mg := NewMockGenerator(JSON)
	mg.SetSeed(1)

	mock, err := mg.GenerateMock(schema, "")

	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(mock, &payload))
	assert.Equal(t, []any{"one", "two"}, payload["list"])
	assert.Equal(t, "one", payload["scalar"])
}

func TestMockGenerator_RenderContextCopiesMutableCachedValues(t *testing.T) {
	t.Parallel()

	schema := componentSchemaForMockTest(t, "Root", `openapi: 3.1.0
info:
  title: cache-copy
  version: 1.0.0
paths: {}
components:
  schemas:
    Root:
      type: object
      properties:
        a:
          $ref: '#/components/schemas/Foo'
        b:
          $ref: '#/components/schemas/Foo'
      dependentSchemas:
        a:
          type: object
          properties:
            dependent:
              type: string
              enum: [only-a]
    Foo:
      type: object
      properties:
        base:
          type: string
          enum: [base]`)

	mg := NewMockGenerator(JSON)
	mg.SetMockGenerationOptions(MockGenerationOptions{MaxMockRefExpansions: 1})

	mock, err := mg.GenerateMock(schema, "")

	require.NoError(t, err)
	var payload map[string]map[string]string
	require.NoError(t, json.Unmarshal(mock, &payload))
	assert.Equal(t, "base", payload["a"]["base"])
	assert.Equal(t, "only-a", payload["a"]["dependent"])
	assert.Equal(t, "base", payload["b"]["base"])
	assert.NotContains(t, payload["b"], "dependent")
}

func TestMockGenerator_RenderContextStopsActiveReferenceCycle(t *testing.T) {
	t.Parallel()

	schema := componentSchemaForMockTest(t, "Node", `openapi: 3.1.0
info:
  title: cycle
  version: 1.0.0
paths: {}
components:
  schemas:
    Node:
      type: object
      properties:
        child:
          $ref: '#/components/schemas/Node'
        label:
          type: string`)

	mg := NewMockGenerator(JSON)
	mg.SetSeed(1)

	mock, err := mg.GenerateMock(schema, "")

	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(mock, &payload))
	assert.NotContains(t, payload, "child")
	assert.NotEmpty(t, payload["label"])
}

func TestMockGenerationBudgetErrorUnwrap(t *testing.T) {
	t.Parallel()

	err := &MockGenerationBudgetError{Budget: "nodes", Limit: 1, Actual: 2}

	assert.ErrorIs(t, err, ErrMockGenerationBudgetExceeded)
	assert.Equal(t, "mock generation budget exceeded: nodes budget exceeded: 2 > 1", err.Error())
}

func TestSchemaRenderer_RenderSchemaWithErrorEnforcesRaisedDepthBudget(t *testing.T) {
	t.Parallel()

	limit := DefaultMaxMockDepth + 50
	renderer := emptyDictionarySchemaRenderer(1)
	renderer.SetMockGenerationOptions(MockGenerationOptions{MaxMockDepth: limit})

	rendered, err := renderer.RenderSchemaWithError(nestedObjectSchema(limit + 1))

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrMockGenerationBudgetExceeded))
	assert.Nil(t, rendered)
}

func TestMockRenderContext_NilRendererAndSchemaKeyFallbacks(t *testing.T) {
	t.Parallel()

	ctx := newMockRenderContext(nil)
	require.NotNil(t, ctx.renderer)

	refSchema := &highbase.Schema{ParentProxy: highbase.CreateSchemaProxyRef("#/components/schemas/Thing")}
	refKey, ok := ctx.schemaKey(refSchema)
	require.True(t, ok)
	assert.Equal(t, "#/components/schemas/Thing", refKey.ref)

	inlineSchema := &highbase.Schema{}
	inlineKey, ok := ctx.schemaKey(inlineSchema)
	require.True(t, ok)
	assert.Same(t, inlineSchema, inlineKey.schema)
}

func TestMockGenerator_RenderContextCoversScalarAndArrayBranches(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"date time":      "type: string\nformat: date-time",
		"date":           "type: string\nformat: date",
		"time":           "type: string\nformat: time",
		"email":          "type: string\nformat: email",
		"hostname":       "type: string\nformat: hostname",
		"ipv4":           "type: string\nformat: ipv4",
		"ipv6":           "type: string\nformat: ipv6",
		"uri":            "type: string\nformat: uri",
		"uri reference":  "type: string\nformat: uri-reference",
		"uuid":           "type: string\nformat: uuid",
		"byte":           "type: string\nformat: byte",
		"password":       "type: string\nformat: password",
		"binary":         "type: string\nformat: binary",
		"bigint string":  "type: string\nformat: bigint",
		"decimal string": "type: string\nformat: decimal",
		"pattern":        "type: string\npattern: '[a-z]{3}'",
		"enum":           "type: string\nenum: [one, two]",
		"array":          "type: array\nitems:\n  type: string",
		"float":          "type: number\nformat: float",
		"double":         "type: number\nformat: double",
		"int32":          "type: integer\nformat: int32",
		"bigint number":  "type: bigint",
		"decimal number": "type: decimal",
		"number enum":    "type: number\nenum: [1, 2]",
	}

	for name, schemaYAML := range tests {
		name := name
		schemaYAML := schemaYAML
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mg := NewMockGenerator(JSON)
			mg.SetSeed(1)
			mock, err := mg.GenerateMock(&highbase.Schema{
				Type:    getSchema([]byte(schemaYAML)).Type,
				Format:  getSchema([]byte(schemaYAML)).Format,
				Pattern: getSchema([]byte(schemaYAML)).Pattern,
				Enum:    getSchema([]byte(schemaYAML)).Enum,
				Items:   getSchema([]byte(schemaYAML)).Items,
			}, "")

			require.NoError(t, err)
			assert.NotEmpty(t, mock)
		})
	}
}

func TestMockRenderContext_GuardBranchesAndBudgets(t *testing.T) {
	t.Parallel()

	structure := make(map[string]any)
	var nilContext *mockRenderContext
	assert.False(t, nilContext.diveIntoSchema(&highbase.Schema{}, "root", structure, 0))

	ctx := newMockRenderContext(emptyDictionarySchemaRenderer(1))
	ctx.err = errors.New("already stopped")
	assert.False(t, ctx.diveIntoSchema(&highbase.Schema{}, "root", structure, 0))
	assert.False(t, ctx.checkBudget("nodes", 1, 2))

	ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
	assert.False(t, ctx.diveIntoSchema(nil, "root", structure, 0))

	ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
	ctx.nodes = 1
	ctx.options.MaxMockNodes = 1
	assert.False(t, ctx.diveIntoSchema(&highbase.Schema{}, "root", structure, 0))
	assert.ErrorIs(t, ctx.err, ErrMockGenerationBudgetExceeded)

	ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
	ctx.options.MaxMockDepth = 0
	assert.False(t, ctx.diveIntoSchema(&highbase.Schema{}, "root", structure, 1))
	assert.ErrorIs(t, ctx.err, ErrMockGenerationBudgetExceeded)

	ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
	ctx.options.MaxMockDepth = DefaultMaxMockDepth
	assert.False(t, ctx.diveIntoSchema(&highbase.Schema{}, "root", structure, DefaultMaxMockDepth+1))
	assert.ErrorIs(t, ctx.err, ErrMockGenerationBudgetExceeded)

	ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
	ctx.enforceBudgets = false
	ctx.options.MaxMockDepth = DefaultMaxMockDepth
	require.True(t, ctx.diveIntoSchema(&highbase.Schema{}, "root", structure, DefaultMaxMockDepth+1))
	assert.Equal(t, mockDepthExceededPlaceholder, structure["root"])

	ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
	ctx.options.MaxMockBytes = 1
	assert.False(t, ctx.diveIntoSchema(&highbase.Schema{Type: []string{booleanType}}, "root", structure, 0))
	assert.ErrorIs(t, ctx.err, ErrMockGenerationBudgetExceeded)

	ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
	_, _, hasKey, entered, ok := ctx.enterSchema(nil, "root", structure)
	assert.False(t, hasKey)
	assert.True(t, entered)
	assert.True(t, ok)

	refSchema := &highbase.Schema{ParentProxy: highbase.CreateSchemaProxyRef("#/components/schemas/Thing")}
	ctx.refs = 1
	ctx.options.MaxMockRefExpansions = 1
	_, _, hasKey, entered, ok = ctx.enterSchema(refSchema, "root", structure)
	assert.True(t, hasKey)
	assert.False(t, entered)
	assert.False(t, ok)

	key := mockSchemaKey{ref: "#/components/schemas/Thing"}
	ctx.active[key] = 2
	ctx.leaveSchema(mockSchemaKey{}, mockSchemaCacheKey{}, false, nil)
	ctx.leaveSchema(key, mockSchemaCacheKey{schema: key, role: mockCacheRole("root")}, true, "cached")
	assert.Equal(t, 1, ctx.active[key])

	_, ok = ctx.schemaKey(nil)
	assert.False(t, ok)
	assert.Equal(t, 4, estimatedMockValueBytes(nil))
	assert.NotZero(t, estimatedMockValueBytes(struct{ Name string }{Name: "thing"}))
}

func TestMockRenderContext_RenderStringExamplesAndFallbacks(t *testing.T) {
	t.Parallel()

	ctx := newMockRenderContext(emptyDictionarySchemaRenderer(1))
	structure := make(map[string]any)
	schema := &highbase.Schema{
		Type: []string{stringType},
		Examples: []*yaml.Node{
			utils.CreateYamlNode("first"),
			nil,
		},
	}
	require.True(t, ctx.renderString(schema, itemsType, structure))
	assert.Equal(t, []any{"first", nil}, structure[itemsType])

	ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
	structure = make(map[string]any)
	schema = &highbase.Schema{
		Type:     []string{stringType},
		Examples: []*yaml.Node{utils.CreateYamlNode("single")},
	}
	require.True(t, ctx.renderString(schema, "name", structure))
	assert.Equal(t, "single", structure["name"])

	ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
	structure = make(map[string]any)
	schema = &highbase.Schema{
		Type:     []string{stringType},
		Examples: []*yaml.Node{nil},
	}
	require.True(t, ctx.renderString(schema, "name", structure))
	assert.Nil(t, structure["name"])

	ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
	structure = make(map[string]any)
	schema = &highbase.Schema{
		Type:      []string{stringType},
		Pattern:   "[",
		MinLength: int64Ptr(5),
		MaxLength: int64Ptr(5),
	}
	require.True(t, ctx.renderString(schema, "name", structure))
	assert.Len(t, structure["name"], 5)
}

func TestMockRenderContext_RenderNumberExamplesAndFormats(t *testing.T) {
	t.Parallel()

	ctx := newMockRenderContext(emptyDictionarySchemaRenderer(1))
	structure := make(map[string]any)
	schema := &highbase.Schema{
		Type:     []string{numberType},
		Examples: []*yaml.Node{utils.CreateYamlNode(42)},
	}
	require.True(t, ctx.renderNumber(schema, "count", structure))
	assert.Equal(t, 42, structure["count"])

	ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
	structure = make(map[string]any)
	schema = &highbase.Schema{
		Type:     []string{numberType},
		Examples: []*yaml.Node{nil},
	}
	require.True(t, ctx.renderNumber(schema, "count", structure))
	assert.Nil(t, structure["count"])

	for _, format := range []string{bigIntType, decimalType} {
		format := format
		t.Run(format, func(t *testing.T) {
			t.Parallel()

			ctx := newMockRenderContext(emptyDictionarySchemaRenderer(1))
			structure := make(map[string]any)
			schema := &highbase.Schema{
				Type:    []string{numberType},
				Format:  format,
				Minimum: float64Ptr(1),
				Maximum: float64Ptr(2),
			}
			require.True(t, ctx.renderNumber(schema, "count", structure))
			assert.NotNil(t, structure["count"])
		})
	}
}

func TestMockGenerationBudgetErrorNil(t *testing.T) {
	t.Parallel()

	var err *MockGenerationBudgetError
	assert.Equal(t, ErrMockGenerationBudgetExceeded.Error(), err.Error())
}

func TestMockRenderContext_RenderObjectBranches(t *testing.T) {
	t.Parallel()

	t.Run("property nil and unresolved refs", func(t *testing.T) {
		t.Parallel()

		props := orderedmap.New[string, *highbase.SchemaProxy]()
		props.Set("empty", nil)
		props.Set("missing", highbase.CreateSchemaProxyRef("#/components/schemas/Missing"))
		props.Set("nilSchema", highbase.NewSchemaProxy(nil))

		ctx := newMockRenderContext(emptyDictionarySchemaRenderer(1))
		var callbackName string
		ctx.renderer.SetUnresolvedRefHandler(func(name string, _ *highbase.SchemaProxy, _ error) {
			callbackName = name
		})
		structure := make(map[string]any)
		require.True(t, ctx.renderObject(&highbase.Schema{
			Type:       []string{objectType},
			Properties: props,
		}, "root", structure, 0))

		root := structure["root"].(map[string]any)
		assert.Empty(t, root["empty"])
		assert.Nil(t, root["missing"])
		assert.Empty(t, root["nilSchema"])
		assert.Equal(t, "missing", callbackName)
	})

	t.Run("required property failure aborts object", func(t *testing.T) {
		t.Parallel()

		props := orderedmap.New[string, *highbase.SchemaProxy]()
		props.Set("required", highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{booleanType}}))
		ctx := newMockRenderContext(emptyDictionarySchemaRenderer(1))
		ctx.options.MaxMockBytes = 13

		assert.False(t, ctx.renderObject(&highbase.Schema{
			Type:       []string{objectType},
			Required:   []string{"required"},
			Properties: props,
		}, "root", make(map[string]any), 0))
	})

	t.Run("allOf branches", func(t *testing.T) {
		t.Parallel()

		ctx := newMockRenderContext(emptyDictionarySchemaRenderer(1))
		var callbackName string
		ctx.renderer.SetUnresolvedRefHandler(func(name string, _ *highbase.SchemaProxy, _ error) {
			callbackName = name
		})
		require.True(t, ctx.renderObject(&highbase.Schema{
			Type:  []string{objectType},
			AllOf: []*highbase.SchemaProxy{highbase.CreateSchemaProxyRef("#/missing")},
		}, "root", make(map[string]any), 0))
		assert.Equal(t, allOfType, callbackName)

		ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
		ctx.options.MaxMockBytes = 1
		assert.False(t, ctx.renderObject(&highbase.Schema{
			Type:  []string{objectType},
			AllOf: []*highbase.SchemaProxy{highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{booleanType}})},
		}, "root", make(map[string]any), 0))

		props := orderedmap.New[string, *highbase.SchemaProxy]()
		props.Set("name", highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{stringType}}))
		ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
		structure := make(map[string]any)
		require.True(t, ctx.renderObject(&highbase.Schema{
			Type:  []string{objectType},
			AllOf: []*highbase.SchemaProxy{highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{objectType}, Properties: props})},
		}, "root", structure, 0))
		assert.NotEmpty(t, structure["root"].(map[string]any)["name"])

		ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
		structure = make(map[string]any)
		require.True(t, ctx.renderObject(&highbase.Schema{
			Type:  []string{objectType},
			AllOf: []*highbase.SchemaProxy{highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{stringType}, Enum: []*yaml.Node{utils.CreateYamlNode("scalar")}})},
		}, "root", structure, 0))
		assert.Equal(t, "scalar", structure["root"])

		ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
		ctx.options.MaxMockProperties = 1
		assert.False(t, ctx.renderObject(&highbase.Schema{
			Type:  []string{objectType},
			AllOf: []*highbase.SchemaProxy{highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{objectType}, Properties: props})},
		}, "root", make(map[string]any), 0))
	})

	t.Run("dependent schemas", func(t *testing.T) {
		t.Parallel()

		dependentSchemas := orderedmap.New[string, *highbase.SchemaProxy]()
		dependentSchemas.Set("missingProp", highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{objectType}}))

		ctx := newMockRenderContext(emptyDictionarySchemaRenderer(1))
		require.True(t, ctx.renderObject(&highbase.Schema{
			Type:             []string{objectType},
			DependentSchemas: dependentSchemas,
		}, "root", make(map[string]any), 0))

		props := orderedmap.New[string, *highbase.SchemaProxy]()
		props.Set("foo", highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{objectType}}))
		dependentSchemas = orderedmap.New[string, *highbase.SchemaProxy]()
		dependentSchemas.Set("foo", highbase.CreateSchemaProxyRef("#/missing"))

		ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
		var callbackName string
		ctx.renderer.SetUnresolvedRefHandler(func(name string, _ *highbase.SchemaProxy, _ error) {
			callbackName = name
		})
		require.True(t, ctx.renderObject(&highbase.Schema{
			Type:             []string{objectType},
			Properties:       props,
			DependentSchemas: dependentSchemas,
		}, "root", make(map[string]any), 0))
		assert.Equal(t, "foo", callbackName)

		dependentProps := orderedmap.New[string, *highbase.SchemaProxy]()
		dependentProps.Set("bar", highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{stringType}}))
		dependentSchemas = orderedmap.New[string, *highbase.SchemaProxy]()
		dependentSchemas.Set("foo", highbase.CreateSchemaProxy(&highbase.Schema{
			Type:       []string{objectType},
			Properties: dependentProps,
		}))

		ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
		structure := make(map[string]any)
		require.True(t, ctx.renderObject(&highbase.Schema{
			Type:             []string{objectType},
			Properties:       props,
			DependentSchemas: dependentSchemas,
		}, "root", structure, 0))
		assert.NotEmpty(t, structure["root"].(map[string]any)["foo"].(map[string]any)["bar"])

		ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
		ctx.options.MaxMockNodes = 1
		dependentSchemas = orderedmap.New[string, *highbase.SchemaProxy]()
		dependentSchemas.Set("foo", highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{objectType}}))
		assert.False(t, ctx.renderObject(&highbase.Schema{
			Type:             []string{objectType},
			Properties:       props,
			DependentSchemas: dependentSchemas,
		}, "root", make(map[string]any), 0))
	})

	t.Run("oneOf and anyOf branches", func(t *testing.T) {
		t.Parallel()

		ctx := newMockRenderContext(emptyDictionarySchemaRenderer(1))
		var callbackName string
		ctx.renderer.SetUnresolvedRefHandler(func(name string, _ *highbase.SchemaProxy, _ error) {
			callbackName = name
		})
		assert.False(t, ctx.renderObject(&highbase.Schema{
			Type:  []string{objectType},
			OneOf: []*highbase.SchemaProxy{highbase.CreateSchemaProxyRef("#/missing")},
		}, "root", make(map[string]any), 0))
		assert.Equal(t, oneOfType, callbackName)

		ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
		ctx.options.MaxMockBytes = 4
		assert.False(t, ctx.renderObject(&highbase.Schema{
			Type:  []string{objectType},
			OneOf: []*highbase.SchemaProxy{highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{booleanType}})},
		}, "root", make(map[string]any), 0))

		props := orderedmap.New[string, *highbase.SchemaProxy]()
		props.Set("choice", highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{stringType}}))
		ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
		structure := make(map[string]any)
		require.True(t, ctx.renderObject(&highbase.Schema{
			Type:  []string{objectType},
			OneOf: []*highbase.SchemaProxy{highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{objectType}, Properties: props})},
		}, "root", structure, 0))
		assert.NotEmpty(t, structure["root"].(map[string]any)["choice"])

		ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
		structure = make(map[string]any)
		require.True(t, ctx.renderObject(&highbase.Schema{
			Type:  []string{objectType},
			OneOf: []*highbase.SchemaProxy{highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{stringType}, Enum: []*yaml.Node{utils.CreateYamlNode("scalar")}})},
		}, "root", structure, 0))
		assert.Equal(t, "scalar", structure["root"])

		ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
		ctx.options.MaxMockProperties = 1
		assert.False(t, ctx.renderObject(&highbase.Schema{
			Type:  []string{objectType},
			OneOf: []*highbase.SchemaProxy{highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{objectType}, Properties: props})},
		}, "root", make(map[string]any), 0))

		ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
		callbackName = ""
		ctx.renderer.SetUnresolvedRefHandler(func(name string, _ *highbase.SchemaProxy, _ error) {
			callbackName = name
		})
		assert.False(t, ctx.renderObject(&highbase.Schema{
			Type:  []string{objectType},
			AnyOf: []*highbase.SchemaProxy{highbase.CreateSchemaProxyRef("#/missing")},
		}, "root", make(map[string]any), 0))
		assert.Equal(t, anyOfType, callbackName)

		ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
		ctx.options.MaxMockBytes = 4
		assert.False(t, ctx.renderObject(&highbase.Schema{
			Type:  []string{objectType},
			AnyOf: []*highbase.SchemaProxy{highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{booleanType}})},
		}, "root", make(map[string]any), 0))

		ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
		structure = make(map[string]any)
		require.True(t, ctx.renderObject(&highbase.Schema{
			Type:  []string{objectType},
			AnyOf: []*highbase.SchemaProxy{highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{objectType}, Properties: props})},
		}, "root", structure, 0))
		assert.NotEmpty(t, structure["root"].(map[string]any)["choice"])

		ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
		structure = make(map[string]any)
		require.True(t, ctx.renderObject(&highbase.Schema{
			Type:  []string{objectType},
			AnyOf: []*highbase.SchemaProxy{highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{stringType}, Enum: []*yaml.Node{utils.CreateYamlNode("scalar")}})},
		}, "root", structure, 0))
		assert.Equal(t, "scalar", structure["root"])

		ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
		ctx.options.MaxMockProperties = 1
		assert.False(t, ctx.renderObject(&highbase.Schema{
			Type:  []string{objectType},
			AnyOf: []*highbase.SchemaProxy{highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{objectType}, Properties: props})},
		}, "root", make(map[string]any), 0))
	})
}

func TestSchemaRenderer_RenderSchemaNilSchema(t *testing.T) {
	t.Parallel()

	rendered, err := emptyDictionarySchemaRenderer(1).renderSchema(nil)
	require.NoError(t, err)
	assert.Nil(t, rendered)
}

func TestMockRenderContext_RenderArrayBranches(t *testing.T) {
	t.Parallel()

	ctx := newMockRenderContext(emptyDictionarySchemaRenderer(1))
	structure := make(map[string]any)
	require.True(t, ctx.renderArray(&highbase.Schema{Type: []string{arrayType}}, "root", structure, 0))
	assert.NotContains(t, structure, "root")

	ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
	structure = make(map[string]any)
	schema := &highbase.Schema{
		Type:     []string{arrayType},
		MinItems: int64Ptr(2),
		Items: &highbase.DynamicValue[*highbase.SchemaProxy, bool]{
			A: highbase.CreateSchemaProxy(&highbase.Schema{
				Type:     []string{stringType},
				Examples: []*yaml.Node{utils.CreateYamlNode("a"), utils.CreateYamlNode("b")},
			}),
		},
	}
	require.True(t, ctx.renderArray(schema, "root", structure, 0))
	assert.Equal(t, []any{"a", "b"}, structure["root"])

	ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
	ctx.renderer.SetUnresolvedRefHandler(func(string, *highbase.SchemaProxy, error) {})
	structure = make(map[string]any)
	schema = &highbase.Schema{
		Type: []string{arrayType},
		Items: &highbase.DynamicValue[*highbase.SchemaProxy, bool]{
			A: highbase.CreateSchemaProxyRef("#/missing"),
		},
	}
	require.True(t, ctx.renderArray(schema, "root", structure, 0))
	assert.Equal(t, []any{nil}, structure["root"])

	ctx = newMockRenderContext(emptyDictionarySchemaRenderer(1))
	ctx.options.MaxMockBytes = 1
	structure = make(map[string]any)
	schema = &highbase.Schema{
		Type: []string{arrayType},
		Items: &highbase.DynamicValue[*highbase.SchemaProxy, bool]{
			A: highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{booleanType}}),
		},
	}
	assert.False(t, ctx.renderArray(schema, "root", structure, 0))
	assert.Equal(t, []any{}, structure["root"])
}

func emptyDictionarySchemaRenderer(seed int64) *SchemaRenderer {
	return &SchemaRenderer{
		rand: rand.New(rand.NewSource(seed)),
	}
}

func renderStringSchema(t *testing.T, renderer *SchemaRenderer, schemaYAML string) string {
	t.Helper()

	compiled := getSchema([]byte(schemaYAML))
	journeyMap := make(map[string]any)
	visited := createVisitedMap()

	require.True(t, renderer.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0))
	value, ok := journeyMap["pb33f"].(string)
	require.True(t, ok)
	return value
}

func componentSchemaForMockTest(t *testing.T, name string, spec string) any {
	t.Helper()

	doc, err := libopenapi.NewDocument([]byte(spec))
	require.NoError(t, err)
	model, err := doc.BuildV3Model()
	require.NoError(t, err)
	require.NotNil(t, model.Model.Components)
	schemaProxy := model.Model.Components.Schemas.GetOrZero(name)
	require.NotNil(t, schemaProxy)
	schema := schemaProxy.Schema()
	require.NotNil(t, schema)
	return schema
}

func schemaWithStringProperties(count int) *highbase.Schema {
	properties := orderedmap.New[string, *highbase.SchemaProxy]()
	for i := 0; i < count; i++ {
		properties.Set("prop"+strconv.Itoa(i), highbase.CreateSchemaProxy(&highbase.Schema{
			Type: []string{stringType},
		}))
	}
	return &highbase.Schema{
		Type:       []string{objectType},
		Properties: properties,
	}
}

func nestedObjectSchema(depth int) *highbase.Schema {
	if depth <= 0 {
		return &highbase.Schema{
			Type: []string{stringType},
			Enum: []*yaml.Node{utils.CreateYamlNode("leaf")},
		}
	}
	properties := orderedmap.New[string, *highbase.SchemaProxy]()
	properties.Set("child", highbase.CreateSchemaProxy(nestedObjectSchema(depth-1)))
	return &highbase.Schema{
		Type:       []string{objectType},
		Required:   []string{"child"},
		Properties: properties,
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}

func float64Ptr(v float64) *float64 {
	return &v
}
