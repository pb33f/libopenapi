// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package renderer

import (
	cryptoRand "crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/lucasjones/reggen"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
)

const (
	rootType         = "rootType"
	stringType       = "string"
	numberType       = "number"
	integerType      = "integer"
	bigIntType       = "bigint"
	decimalType      = "decimal"
	booleanType      = "boolean"
	objectType       = "object"
	arrayType        = "array"
	int32Type        = "int32"
	floatType        = "float"
	doubleType       = "double"
	byteType         = "byte"
	binaryType       = "binary"
	passwordType     = "password"
	dateType         = "date"
	dateTimeType     = "date-time"
	timeType         = "time"
	emailType        = "email"
	hostnameType     = "hostname"
	ipv4Type         = "ipv4"
	ipv6Type         = "ipv6"
	uriType          = "uri"
	uriReferenceType = "uri-reference"
	uuidType         = "uuid"
	allOfType        = "allOf"
	anyOfType        = "anyOf"
	oneOfType        = "oneOf"
	itemsType        = "items"

	mockDepthExceededPlaceholder = "too deep to continue rendering..."

	// DefaultMaxPatternRepeatBudget is the default regex repeat budget used when generating string mocks from patterns.
	DefaultMaxPatternRepeatBudget = 32

	// DefaultMaxGeneratedStringBytes is the default byte ceiling for each generated string mock value.
	DefaultMaxGeneratedStringBytes = 4096

	// DefaultMaxMockDepth is the default maximum recursive schema depth for generated mocks.
	DefaultMaxMockDepth = 100

	// DefaultMaxMockNodes is the default maximum number of schema nodes visited for a generated mock.
	DefaultMaxMockNodes = 10000

	// DefaultMaxMockProperties is the default maximum number of object properties rendered for a generated mock.
	DefaultMaxMockProperties = 5000

	// DefaultMaxMockRefExpansions is the default maximum number of reference expansions for a generated mock.
	DefaultMaxMockRefExpansions = 2000

	// DefaultMaxMockBytes is the default approximate generated mock byte budget before serialization.
	DefaultMaxMockBytes = 1024 * 1024

	// letterBytes is used to generate random words when no dictionary is configured.
	letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

// ErrMockGenerationBudgetExceeded is wrapped by errors caused by configured mock generation budgets.
var ErrMockGenerationBudgetExceeded = errors.New("mock generation budget exceeded")

// UnresolvedRefHandler is called when a $ref property cannot be resolved during rendering.
type UnresolvedRefHandler func(propertyName string, proxy *base.SchemaProxy, err error)

// SchemaRenderer generates mock values from schemas, examples and schema constraints.
//
// When a dictionary is configured, it is used as the source for generated words.
type SchemaRenderer struct {
	words           []string
	disableRequired bool
	rand            *rand.Rand
	onUnresolvedRef UnresolvedRefHandler
	mockOptions     MockGenerationOptions
}

// MockGenerationBudgetError describes which mock generation budget was exceeded.
type MockGenerationBudgetError struct {
	// Budget is the name of the budget that was exceeded.
	Budget string
	// Limit is the configured budget value.
	Limit int
	// Actual is the observed value that exceeded the limit.
	Actual int
}

func (e *MockGenerationBudgetError) Error() string {
	if e == nil {
		return ErrMockGenerationBudgetExceeded.Error()
	}
	return fmt.Sprintf("%s: %s budget exceeded: %d > %d",
		ErrMockGenerationBudgetExceeded, e.Budget, e.Actual, e.Limit)
}

// Unwrap returns ErrMockGenerationBudgetExceeded for errors.Is checks.
func (e *MockGenerationBudgetError) Unwrap() error {
	return ErrMockGenerationBudgetExceeded
}

// MockGenerationOptions controls how much work the renderer may spend generating mock values.
//
// Zero or negative values use the package defaults. OpenAPI schema constraints such as maxLength are still used as
// validity hints, but they are not treated as permission to perform unbounded generation work.
type MockGenerationOptions struct {
	// MaxPatternRepeatBudget limits the repeat budget passed to regex-based string generation.
	MaxPatternRepeatBudget int
	// MaxGeneratedStringBytes limits the final size of generated string values.
	MaxGeneratedStringBytes int
	// MaxMockDepth limits recursive schema depth while building mock structures.
	MaxMockDepth int
	// MaxMockNodes limits the number of schema nodes visited while building a mock.
	MaxMockNodes int
	// MaxMockProperties limits the number of object properties rendered while building a mock.
	MaxMockProperties int
	// MaxMockRefExpansions limits the number of $ref schema expansions while building a mock.
	MaxMockRefExpansions int
	// MaxMockBytes limits approximate mock structure size before serialization.
	MaxMockBytes int
}

// SetUnresolvedRefHandler sets a callback that is invoked when a $ref cannot be resolved during rendering.
func (wr *SchemaRenderer) SetUnresolvedRefHandler(handler UnresolvedRefHandler) {
	wr.onUnresolvedRef = handler
}

// SetMockGenerationOptions sets work and output budgets for generated mock values.
//
// Zero or negative option values are replaced with the package defaults.
func (wr *SchemaRenderer) SetMockGenerationOptions(options MockGenerationOptions) {
	wr.mockOptions = normalizeMockGenerationOptions(options)
}

// CreateRendererUsingDictionary creates a SchemaRenderer using a custom dictionary file.
//
// The location of a text file with one word per line is expected.
func CreateRendererUsingDictionary(dictionaryLocation string) *SchemaRenderer {
	words := ReadDictionary(dictionaryLocation)
	return &SchemaRenderer{
		words:       words,
		rand:        rand.New(rand.NewSource(time.Now().UnixNano())),
		mockOptions: normalizeMockGenerationOptions(MockGenerationOptions{}),
	}
}

// CreateRendererUsingDefaultDictionary creates a SchemaRenderer using the default dictionary file.
//
// The default dictionary is located at /usr/share/dict/words on most systems.
// Windows users need to use CreateRendererUsingDictionary to specify a custom dictionary.
func CreateRendererUsingDefaultDictionary() *SchemaRenderer {
	wr := new(SchemaRenderer)
	wr.words = ReadDictionary("/usr/share/dict/words")
	wr.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	wr.mockOptions = normalizeMockGenerationOptions(MockGenerationOptions{})
	return wr
}

// RenderSchema renders a schema into a value that can be serialized as JSON or YAML.
//
// RenderSchema preserves its historical best-effort behavior. Use RenderSchemaWithError to enforce and inspect mock
// generation work budget failures.
func (wr *SchemaRenderer) RenderSchema(schema *base.Schema) any {
	return wr.renderSchemaBestEffort(schema)
}

// RenderSchemaWithError renders a schema into a value that can be serialized as JSON or YAML.
//
// If mock generation exceeds a configured work budget, the returned error wraps ErrMockGenerationBudgetExceeded.
func (wr *SchemaRenderer) RenderSchemaWithError(schema *base.Schema) (any, error) {
	return wr.renderSchema(schema)
}

func (wr *SchemaRenderer) renderSchema(schema *base.Schema) (any, error) {
	return wr.renderSchemaWithBudgets(schema, true)
}

func (wr *SchemaRenderer) renderSchemaBestEffort(schema *base.Schema) any {
	rendered, _ := wr.renderSchemaWithBudgets(schema, false)
	return rendered
}

func (wr *SchemaRenderer) renderSchemaWithBudgets(schema *base.Schema, enforceBudgets bool) (any, error) {
	structure := make(map[string]any)
	ctx := newMockRenderContext(wr)
	ctx.enforceBudgets = enforceBudgets
	if !ctx.diveIntoSchema(schema, rootType, structure, 0) {
		if ctx.err != nil {
			return nil, ctx.err
		}
		return nil, nil
	}
	return structure[rootType], nil
}

// DisableRequiredCheck disables required-property filtering when rendering a schema.
//
// When disabled, all properties are rendered, not just required properties.
// https://github.com/pb33f/libopenapi/issues/200
func (wr *SchemaRenderer) DisableRequiredCheck() {
	wr.disableRequired = true
}

// SetSeed sets a specific seed for the random number generator used by this renderer.
// This is useful for generating deterministic mocks for testing purposes.
func (wr *SchemaRenderer) SetSeed(seed int64) {
	wr.rand = rand.New(rand.NewSource(seed))
}

// DiveIntoSchema renders a schema into structure at key.
//
// Examples are preferred. If no examples are available, the renderer generates a value from the schema type, format
// and pattern.
func (wr *SchemaRenderer) DiveIntoSchema(schema *base.Schema, key string, structure map[string]any, visited map[string]bool, depth int) bool {
	if schema == nil {
		return false
	}
	if schema.Example != nil {
		var example any
		_ = schema.Example.Decode(&example)

		structure[key] = example
		return true
	}

	// Prevent unbounded recursion on deeply nested schemas.
	if depth > 100 {
		structure[key] = mockDepthExceededPlaceholder
		return true
	}

	// render out a string.
	if slices.Contains(schema.Type, stringType) {
		options := wr.effectiveMockGenerationOptions()
		structure[key] = wr.renderMockStringValue(schema, key, options.MaxGeneratedStringBytes)
		return true
	}

	// handle numbers
	if slices.Contains(schema.Type, numberType) ||
		slices.Contains(schema.Type, integerType) ||
		slices.Contains(schema.Type, bigIntType) ||
		slices.Contains(schema.Type, decimalType) {

		structure[key] = wr.renderMockNumberValue(schema)
		return true
	}

	// handle booleans
	if slices.Contains(schema.Type, booleanType) {
		structure[key] = true
	}

	// handle objects
	if slices.Contains(schema.Type, objectType) || (schema.Properties != nil && schema.Properties.Len() > 0) ||
		schema.AllOf != nil || (schema.DependentSchemas != nil && schema.DependentSchemas.Len() > 0) || schema.OneOf != nil || schema.AnyOf != nil {

		if schema.ParentProxy.IsReference() {
			if visited[schema.ParentProxy.GetReference()] {
				return false
			}
			visited[schema.ParentProxy.GetReference()] = true
		}

		properties := schema.Properties
		propertyMap := make(map[string]any)
		var compositionValue any
		hasCompositionValue := false

		if properties != nil {
			// check if this schema has required properties, if so, then only render required props, if not
			// render everything in the schema.
			checkProps := orderedmap.New[string, *base.SchemaProxy]()
			if wr.disableRequired || len(schema.Required) == 0 {
				for name, value := range properties.FromOldest() {
					checkProps.Set(name, value)
				}
			}

			for _, requiredProp := range schema.Required {
				checkProps.Set(requiredProp, properties.GetOrZero(requiredProp))
			}

			for propName, propValue := range checkProps.FromOldest() {
				// propValue is nil when a required property is listed but absent from the
				// properties map. Emit {} to preserve existing behavior.
				if propValue == nil {
					propertyMap[propName] = make(map[string]any)
					continue
				}
				propertySchema := propValue.Schema()
				required := slices.Contains(schema.Required, propName)
				if propertySchema != nil {
					success := wr.DiveIntoSchema(propertySchema, propName, propertyMap, copyMap(visited), depth+1)
					if !success {
						if required {
							return false
						}
						delete(propertyMap, propName)
						continue
					}
				} else if propValue.IsReference() {
					// Emit null for unresolved $ref properties and notify the callback.
					propertyMap[propName] = nil
					if wr.onUnresolvedRef != nil {
						wr.onUnresolvedRef(propName, propValue, propValue.GetBuildError())
					}
				} else {
					// Emit {} for non-reference properties with no schema to preserve existing behavior.
					propertyMap[propName] = make(map[string]any)
				}
			}
		}

		// handle allOf
		allOf := schema.AllOf
		if allOf != nil {
			allOfMap := make(map[string]any)
			for _, allOfSchema := range allOf {
				allOfCompiled := allOfSchema.Schema()
				if allOfCompiled == nil {
					if wr.onUnresolvedRef != nil {
						wr.onUnresolvedRef(allOfType, allOfSchema, allOfSchema.GetBuildError())
					}
					continue
				}
				success := wr.DiveIntoSchema(allOfCompiled, allOfType, allOfMap, copyMap(visited), depth+1)
				if !success {
					return false
				}

				if value, ok := allOfMap[allOfType]; ok {
					if m, ok := value.(map[string]any); ok {
						for k, v := range m {
							propertyMap[k] = v
						}
					} else {
						compositionValue = value
						hasCompositionValue = true
					}
				}
			}
		}

		// handle dependentSchemas
		dependentSchemas := schema.DependentSchemas
		if dependentSchemas != nil {
			dependentSchemasMap := make(map[string]any)
			for k, dependentSchema := range dependentSchemas.FromOldest() {
				// only map if the property exists
				if propertyMap[k] != nil {
					dependentSchemaCompiled := dependentSchema.Schema()
					if dependentSchemaCompiled == nil {
						if wr.onUnresolvedRef != nil {
							wr.onUnresolvedRef(k, dependentSchema, dependentSchema.GetBuildError())
						}
						continue
					}
					success := wr.DiveIntoSchema(dependentSchemaCompiled, k, dependentSchemasMap, copyMap(visited), depth+1)
					if !success {
						return false
					}
					for i, v := range dependentSchemasMap[k].(map[string]any) {
						propertyMap[k].(map[string]any)[i] = v
					}
				}
			}
		}

		// handle oneOf
		oneOf := schema.OneOf
		oneOfSuccess := true
		for _, oneOfSchema := range oneOf {
			oneOfSuccess = false
			oneOfMap := make(map[string]any)
			oneOfCompiled := oneOfSchema.Schema()
			if oneOfCompiled == nil {
				if wr.onUnresolvedRef != nil {
					wr.onUnresolvedRef(oneOfType, oneOfSchema, oneOfSchema.GetBuildError())
				}
				continue
			}
			success := wr.DiveIntoSchema(oneOfCompiled, oneOfType, oneOfMap, copyMap(visited), depth+1)
			if !success {
				continue
			}
			if value, ok := oneOfMap[oneOfType]; ok {
				if m, ok := value.(map[string]any); ok {
					for k, v := range m {
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

		// handle anyOf
		anyOfSuccess := true
		anyOf := schema.AnyOf
		for _, anyOfSchema := range anyOf {
			anyOfSuccess = false
			anyOfMap := make(map[string]any)
			anyOfCompiled := anyOfSchema.Schema()
			if anyOfCompiled == nil {
				if wr.onUnresolvedRef != nil {
					wr.onUnresolvedRef(anyOfType, anyOfSchema, anyOfSchema.GetBuildError())
				}
				continue
			}
			success := wr.DiveIntoSchema(anyOfCompiled, anyOfType, anyOfMap, copyMap(visited), depth+1)
			if !success {
				continue
			}
			if value, ok := anyOfMap[anyOfType]; ok {
				if m, ok := value.(map[string]any); ok {
					for k, v := range m {
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
			return true
		}

		structure[key] = propertyMap
		return true
	}

	if slices.Contains(schema.Type, arrayType) {

		// an array needs an items schema
		itemsSchema := schema.Items
		if itemsSchema != nil {
			// otherwise the items value is a schema, so we need to dive into it
			if itemsSchema.IsA() {

				// check if the schema contains a minItems value and render up to that number.
				var minItems int64 = 1
				if schema.MinItems != nil {
					minItems = *schema.MinItems
				}

				renderedItems := []any{}
				// build up the array
				for i := int64(0); i < minItems; i++ {
					itemMap := make(map[string]any)
					itemsSchemaCompiled := itemsSchema.A.Schema()

					if itemsSchemaCompiled == nil {
						if wr.onUnresolvedRef != nil {
							wr.onUnresolvedRef(itemsType, itemsSchema.A, itemsSchema.A.GetBuildError())
						}
						renderedItems = append(renderedItems, nil)
						break
					}

					success := wr.DiveIntoSchema(itemsSchemaCompiled, itemsType, itemMap, copyMap(visited), depth+1)
					if !success {
						// to do handle minItems correctly
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
				return true
			}
		}
	}

	return true
}

func normalizeMockGenerationOptions(options MockGenerationOptions) MockGenerationOptions {
	if options.MaxPatternRepeatBudget <= 0 {
		options.MaxPatternRepeatBudget = DefaultMaxPatternRepeatBudget
	}
	if options.MaxGeneratedStringBytes <= 0 {
		options.MaxGeneratedStringBytes = DefaultMaxGeneratedStringBytes
	}
	if options.MaxMockDepth <= 0 {
		options.MaxMockDepth = DefaultMaxMockDepth
	}
	if options.MaxMockNodes <= 0 {
		options.MaxMockNodes = DefaultMaxMockNodes
	}
	if options.MaxMockProperties <= 0 {
		options.MaxMockProperties = DefaultMaxMockProperties
	}
	if options.MaxMockRefExpansions <= 0 {
		options.MaxMockRefExpansions = DefaultMaxMockRefExpansions
	}
	if options.MaxMockBytes <= 0 {
		options.MaxMockBytes = DefaultMaxMockBytes
	}
	return options
}

func (wr *SchemaRenderer) effectiveMockGenerationOptions() MockGenerationOptions {
	if wr == nil {
		return normalizeMockGenerationOptions(MockGenerationOptions{})
	}
	return normalizeMockGenerationOptions(wr.mockOptions)
}

func (wr *SchemaRenderer) generatePatternString(pattern string, schemaMaxLength int64, hasSchemaMaxLength bool) (string, error) {
	options := wr.effectiveMockGenerationOptions()
	repeatBudget := options.MaxPatternRepeatBudget
	if hasSchemaMaxLength && schemaMaxLength > 0 && schemaMaxLength < int64(repeatBudget) {
		repeatBudget = int(schemaMaxLength)
	}
	str, err := reggen.Generate(pattern, repeatBudget)
	if err != nil {
		return "", err
	}
	return truncateStringBytes(str, options.MaxGeneratedStringBytes), nil
}

func boundedGeneratedStringRange(minLength, maxLength int64, maxBytes int) (int64, int64) {
	if maxBytes <= 0 {
		return minLength, maxLength
	}
	capLength := int64(maxBytes)
	if minLength > capLength {
		minLength = capLength
	}
	if maxLength <= 0 || maxLength > capLength {
		maxLength = capLength
	}
	if maxLength < minLength {
		maxLength = minLength
	}
	return minLength, maxLength
}

func truncateStringBytes(value string, maxBytes int) string {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}
	cut := maxBytes
	for cut > 0 && !utf8.RuneStart(value[cut]) {
		cut--
	}
	return value[:cut]
}

func readFile(file io.ReadCloser) []string {
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return []string{}
	}
	return strings.Split(string(bytes), "\n")
}

func copyMap(m map[string]bool) map[string]bool {
	res := make(map[string]bool)
	for key, value := range m {
		res[key] = value
	}
	return res
}

// ReadDictionary reads a dictionary file and returns one entry per line.
func ReadDictionary(dictionaryLocation string) []string {
	file, err := os.Open(dictionaryLocation)
	if err != nil {
		return []string{}
	}
	return readFile(file)
}

// RandomWord returns a random word between the min and max lengths.
//
// If no dictionary is configured, RandomWord returns a generated alphabetic string. Set min and max to 0 to return the
// selected dictionary word without length filtering. The depth parameter prevents unbounded retries.
func (wr *SchemaRenderer) RandomWord(min, max int64, depth int) string {
	if depth > 100 {
		return fmt.Sprintf("no-word-found-%d-%d", min, max)
	}

	if len(wr.words) == 0 {
		if min == 0 {
			min = 7
		}
		b := make([]byte, min)
		for i := range b {
			b[i] = letterBytes[wr.rand.Intn(len(letterBytes))]
		}
		return string(b)
	}

	word := wr.words[wr.rand.Int()%len(wr.words)]
	if min == 0 && max == 0 {
		return word
	}
	if len(word) < int(min) || len(word) > int(max) {
		return wr.RandomWord(min, max, depth+1)
	}
	return word
}

// RandomInt returns a random integer between min and max.
func (wr *SchemaRenderer) RandomInt(min, max int64) int64 {
	if max <= min {
		return min
	}
	return wr.rand.Int63n(max-min) + min
}

// RandomFloat64 returns a random float64 between 0 and 1.
func (wr *SchemaRenderer) RandomFloat64() float64 {
	return wr.rand.Float64()
}

// PseudoUUID returns a UUID-shaped random value for mock data.
func (wr *SchemaRenderer) PseudoUUID() string {
	b := make([]byte, 16)
	_, _ = cryptoRand.Read(b)
	return strings.ToLower(fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]))
}
