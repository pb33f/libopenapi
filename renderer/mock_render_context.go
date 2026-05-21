// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package renderer

import (
	"fmt"
	"slices"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

const mockSchemaCacheRoleDefault = "schema"

type mockSchemaKey struct {
	node   *yaml.Node
	ref    string
	schema *base.Schema
}

type mockSchemaCacheKey struct {
	schema mockSchemaKey
	role   string
}

type mockRenderContext struct {
	renderer       *SchemaRenderer
	options        MockGenerationOptions
	active         map[mockSchemaKey]int
	completed      map[mockSchemaCacheKey]any
	enforceBudgets bool
	nodes          int
	props          int
	refs           int
	bytes          int
	err            error
}

func newMockRenderContext(renderer *SchemaRenderer) *mockRenderContext {
	if renderer == nil {
		renderer = CreateRendererUsingDefaultDictionary()
	}
	return &mockRenderContext{
		renderer:       renderer,
		options:        renderer.effectiveMockGenerationOptions(),
		active:         make(map[mockSchemaKey]int),
		completed:      make(map[mockSchemaCacheKey]any),
		enforceBudgets: true,
	}
}

func (ctx *mockRenderContext) diveIntoSchema(schema *base.Schema, key string, structure map[string]any, depth int) bool {
	if ctx == nil || ctx.err != nil {
		return false
	}
	if schema == nil {
		return false
	}
	if schema.Example != nil {
		var example any
		_ = schema.Example.Decode(&example)
		structure[key] = example
		return ctx.noteValue(example)
	}
	if !ctx.noteNode() {
		return false
	}

	schemaKey, cacheKey, hasSchemaKey, entered, ok := ctx.enterSchema(schema, key, structure)
	if !entered {
		return ok && ctx.err == nil
	}
	defer func() {
		ctx.leaveSchema(schemaKey, cacheKey, hasSchemaKey, structure[key])
	}()

	if depth > ctx.options.MaxMockDepth {
		if ctx.enforceBudgets {
			ctx.err = &MockGenerationBudgetError{Budget: "depth", Limit: ctx.options.MaxMockDepth, Actual: depth}
			return false
		}
		structure[key] = mockDepthExceededPlaceholder
		return ctx.noteValue(structure[key])
	}

	if slices.Contains(schema.Type, stringType) {
		return ctx.renderString(schema, key, structure)
	}

	if slices.Contains(schema.Type, numberType) ||
		slices.Contains(schema.Type, integerType) ||
		slices.Contains(schema.Type, bigIntType) ||
		slices.Contains(schema.Type, decimalType) {
		return ctx.renderNumber(schema, key, structure)
	}

	if slices.Contains(schema.Type, booleanType) {
		structure[key] = true
		if !ctx.noteValue(true) {
			return false
		}
	}

	if ctx.isObjectSchema(schema) {
		return ctx.renderObject(schema, key, structure, depth)
	}

	if slices.Contains(schema.Type, arrayType) {
		return ctx.renderArray(schema, key, structure, depth)
	}

	return true
}

func (ctx *mockRenderContext) renderString(schema *base.Schema, key string, structure map[string]any) bool {
	structure[key] = ctx.renderer.renderMockStringValue(schema, key, ctx.options.MaxGeneratedStringBytes)
	return ctx.noteValue(structure[key])
}

func (ctx *mockRenderContext) renderNumber(schema *base.Schema, key string, structure map[string]any) bool {
	structure[key] = ctx.renderer.renderMockNumberValue(schema)
	return ctx.noteValue(structure[key])
}

func (ctx *mockRenderContext) renderObject(schema *base.Schema, key string, structure map[string]any, depth int) bool {
	propertyMap := make(map[string]any)
	var compositionValue any
	hasCompositionValue := false

	if schema.Properties != nil {
		checkProps := orderedmap.New[string, *base.SchemaProxy]()
		if ctx.renderer.disableRequired || len(schema.Required) == 0 {
			for name, value := range schema.Properties.FromOldest() {
				checkProps.Set(name, value)
			}
		}
		for _, requiredProp := range schema.Required {
			checkProps.Set(requiredProp, schema.Properties.GetOrZero(requiredProp))
		}

		for propName, propValue := range checkProps.FromOldest() {
			if !ctx.noteProperty(propName) {
				return false
			}
			if propValue == nil {
				propertyMap[propName] = make(map[string]any)
				continue
			}
			propertySchema := propValue.Schema()
			required := slices.Contains(schema.Required, propName)
			if propertySchema != nil {
				success := ctx.diveIntoSchema(propertySchema, propName, propertyMap, depth+1)
				if !success {
					if required {
						return false
					}
					delete(propertyMap, propName)
					continue
				}
			} else if propValue.IsReference() {
				propertyMap[propName] = nil
				if ctx.renderer.onUnresolvedRef != nil {
					ctx.renderer.onUnresolvedRef(propName, propValue, propValue.GetBuildError())
				}
			} else {
				propertyMap[propName] = make(map[string]any)
			}
		}
	}

	if schema.AllOf != nil {
		allOfMap := make(map[string]any)
		for _, allOfSchema := range schema.AllOf {
			allOfCompiled := allOfSchema.Schema()
			if allOfCompiled == nil {
				if ctx.renderer.onUnresolvedRef != nil {
					ctx.renderer.onUnresolvedRef(allOfType, allOfSchema, allOfSchema.GetBuildError())
				}
				continue
			}
			if !ctx.diveIntoSchema(allOfCompiled, allOfType, allOfMap, depth+1) {
				return false
			}
			if value, ok := allOfMap[allOfType]; ok {
				if m, ok := value.(map[string]any); ok {
					for k, v := range m {
						if !ctx.noteProperty(k) {
							return false
						}
						propertyMap[k] = v
					}
				} else {
					compositionValue = value
					hasCompositionValue = true
				}
			}
		}
	}

	if schema.DependentSchemas != nil {
		dependentSchemasMap := make(map[string]any)
		for k, dependentSchema := range schema.DependentSchemas.FromOldest() {
			if propertyMap[k] == nil {
				continue
			}
			dependentSchemaCompiled := dependentSchema.Schema()
			if dependentSchemaCompiled == nil {
				if ctx.renderer.onUnresolvedRef != nil {
					ctx.renderer.onUnresolvedRef(k, dependentSchema, dependentSchema.GetBuildError())
				}
				continue
			}
			if !ctx.diveIntoSchema(dependentSchemaCompiled, k, dependentSchemasMap, depth+1) {
				return false
			}
			for i, v := range dependentSchemasMap[k].(map[string]any) {
				propertyMap[k].(map[string]any)[i] = v
			}
		}
	}

	oneOfSuccess := true
	for _, oneOfSchema := range schema.OneOf {
		oneOfSuccess = false
		oneOfMap := make(map[string]any)
		oneOfCompiled := oneOfSchema.Schema()
		if oneOfCompiled == nil {
			if ctx.renderer.onUnresolvedRef != nil {
				ctx.renderer.onUnresolvedRef(oneOfType, oneOfSchema, oneOfSchema.GetBuildError())
			}
			continue
		}
		if !ctx.diveIntoSchema(oneOfCompiled, oneOfType, oneOfMap, depth+1) {
			continue
		}
		if value, ok := oneOfMap[oneOfType]; ok {
			if m, ok := value.(map[string]any); ok {
				for k, v := range m {
					if !ctx.noteProperty(k) {
						return false
					}
					propertyMap[k] = v
				}
			} else {
				compositionValue = value
				hasCompositionValue = true
			}
		}
		oneOfSuccess = true
		break
	}
	if !oneOfSuccess {
		return false
	}

	anyOfSuccess := true
	for _, anyOfSchema := range schema.AnyOf {
		anyOfSuccess = false
		anyOfMap := make(map[string]any)
		anyOfCompiled := anyOfSchema.Schema()
		if anyOfCompiled == nil {
			if ctx.renderer.onUnresolvedRef != nil {
				ctx.renderer.onUnresolvedRef(anyOfType, anyOfSchema, anyOfSchema.GetBuildError())
			}
			continue
		}
		if !ctx.diveIntoSchema(anyOfCompiled, anyOfType, anyOfMap, depth+1) {
			continue
		}
		if value, ok := anyOfMap[anyOfType]; ok {
			if m, ok := value.(map[string]any); ok {
				for k, v := range m {
					if !ctx.noteProperty(k) {
						return false
					}
					propertyMap[k] = v
				}
			} else {
				compositionValue = value
				hasCompositionValue = true
			}
		}
		anyOfSuccess = true
		break
	}
	if !anyOfSuccess {
		return false
	}

	if len(propertyMap) == 0 && hasCompositionValue {
		structure[key] = compositionValue
		return ctx.noteValue(compositionValue)
	}
	structure[key] = propertyMap
	return ctx.noteValue(propertyMap)
}

func (ctx *mockRenderContext) renderArray(schema *base.Schema, key string, structure map[string]any, depth int) bool {
	itemsSchema := schema.Items
	if itemsSchema == nil || !itemsSchema.IsA() {
		return true
	}

	var minItems int64 = 1
	if schema.MinItems != nil {
		minItems = *schema.MinItems
	}

	renderedItems := []any{}
	for i := int64(0); i < minItems; i++ {
		itemMap := make(map[string]any)
		itemsSchemaCompiled := itemsSchema.A.Schema()
		if itemsSchemaCompiled == nil {
			if ctx.renderer.onUnresolvedRef != nil {
				ctx.renderer.onUnresolvedRef(itemsType, itemsSchema.A, itemsSchema.A.GetBuildError())
			}
			renderedItems = append(renderedItems, nil)
			break
		}
		if !ctx.diveIntoSchema(itemsSchemaCompiled, itemsType, itemMap, depth+1) {
			renderedItems = []any{}
			break
		}
		if multipleItems, ok := itemMap[itemsType].([]any); ok {
			renderedItems = multipleItems
		} else {
			renderedItems = append(renderedItems, itemMap[itemsType])
		}
	}
	structure[key] = renderedItems
	return ctx.noteValue(renderedItems)
}

func (ctx *mockRenderContext) isObjectSchema(schema *base.Schema) bool {
	return slices.Contains(schema.Type, objectType) ||
		(schema.Properties != nil && schema.Properties.Len() > 0) ||
		schema.AllOf != nil ||
		(schema.DependentSchemas != nil && schema.DependentSchemas.Len() > 0) ||
		schema.OneOf != nil ||
		schema.AnyOf != nil
}

func (ctx *mockRenderContext) enterSchema(schema *base.Schema, key string, structure map[string]any) (mockSchemaKey, mockSchemaCacheKey, bool, bool, bool) {
	schemaKey, ok := ctx.schemaKey(schema)
	if !ok {
		return mockSchemaKey{}, mockSchemaCacheKey{}, false, true, true
	}
	cacheKey := mockSchemaCacheKey{schema: schemaKey, role: mockCacheRole(key)}
	if cached, found := ctx.completed[cacheKey]; found {
		copied := copyMockValue(cached)
		structure[key] = copied
		_ = ctx.noteValue(copied)
		return schemaKey, cacheKey, true, false, true
	}
	if ctx.active[schemaKey] > 0 {
		return schemaKey, cacheKey, true, false, false
	}
	if schema.ParentProxy != nil && schema.ParentProxy.IsReference() && !ctx.noteRefExpansion() {
		return schemaKey, cacheKey, true, false, false
	}
	ctx.active[schemaKey]++
	return schemaKey, cacheKey, true, true, true
}

func (ctx *mockRenderContext) leaveSchema(schemaKey mockSchemaKey, cacheKey mockSchemaCacheKey, hasSchemaKey bool, value any) {
	if !hasSchemaKey {
		return
	}
	if count := ctx.active[schemaKey]; count <= 1 {
		delete(ctx.active, schemaKey)
	} else {
		ctx.active[schemaKey] = count - 1
	}
	if ctx.err == nil && value != nil {
		ctx.completed[cacheKey] = copyMockValue(value)
	}
}

func mockCacheRole(key string) string {
	switch key {
	case itemsType, allOfType, oneOfType, anyOfType:
		return key
	default:
		return mockSchemaCacheRoleDefault
	}
}

func (ctx *mockRenderContext) schemaKey(schema *base.Schema) (mockSchemaKey, bool) {
	if schema == nil {
		return mockSchemaKey{}, false
	}
	if low := schema.GoLow(); low != nil && low.RootNode != nil {
		return mockSchemaKey{node: low.RootNode}, true
	}
	if schema.ParentProxy != nil && schema.ParentProxy.IsReference() {
		if ref := schema.ParentProxy.GetReference(); ref != "" {
			return mockSchemaKey{ref: ref}, true
		}
	}
	return mockSchemaKey{schema: schema}, true
}

func (ctx *mockRenderContext) noteNode() bool {
	ctx.nodes++
	return ctx.checkBudget("nodes", ctx.options.MaxMockNodes, ctx.nodes)
}

func (ctx *mockRenderContext) noteProperty(name string) bool {
	ctx.props++
	ctx.bytes += len(name) + 4
	return ctx.checkBudget("properties", ctx.options.MaxMockProperties, ctx.props) &&
		ctx.checkBudget("bytes", ctx.options.MaxMockBytes, ctx.bytes)
}

func (ctx *mockRenderContext) noteRefExpansion() bool {
	ctx.refs++
	return ctx.checkBudget("ref expansions", ctx.options.MaxMockRefExpansions, ctx.refs)
}

func (ctx *mockRenderContext) noteValue(value any) bool {
	ctx.bytes += estimatedMockValueBytes(value)
	return ctx.checkBudget("bytes", ctx.options.MaxMockBytes, ctx.bytes)
}

func (ctx *mockRenderContext) checkBudget(name string, limit int, actual int) bool {
	if ctx.err != nil {
		return false
	}
	if !ctx.enforceBudgets {
		return true
	}
	if limit <= 0 || actual <= limit {
		return true
	}
	ctx.err = &MockGenerationBudgetError{Budget: name, Limit: limit, Actual: actual}
	return false
}

func copyMockValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		copied := make(map[string]any, len(v))
		for key, child := range v {
			copied[key] = copyMockValue(child)
		}
		return copied
	case []any:
		copied := make([]any, len(v))
		for i, child := range v {
			copied[i] = copyMockValue(child)
		}
		return copied
	default:
		return value
	}
}

func estimatedMockValueBytes(value any) int {
	switch v := value.(type) {
	case nil:
		return 4
	case string:
		return len(v) + 2
	case []any:
		return 2 + len(v)*2
	case map[string]any:
		return 2 + len(v)*4
	case bool:
		return 5
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return 20
	case float32, float64:
		return 24
	default:
		return len(fmt.Sprint(v))
	}
}
