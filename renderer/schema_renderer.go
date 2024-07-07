// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package renderer

import (
	cryptoRand "crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/lucasjones/reggen"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
	"golang.org/x/exp/slices"
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
)

// used to generate random words if there is no dictionary applied.
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func init() {
	// create a new random seed
	rand.New(rand.NewSource(time.Now().UnixNano()))
}

// SchemaRenderer is a renderer that will generate random words, numbers and values based on a dictionary file.
// The dictionary is just a slice of strings that is used to generate random words.
type SchemaRenderer struct {
	words           []string
	disableRequired bool
}

// CreateRendererUsingDictionary will create a new SchemaRenderer using a custom dictionary file.
// The location of a text file with one word per line is expected.
func CreateRendererUsingDictionary(dictionaryLocation string) *SchemaRenderer {
	// try and read in the dictionary file
	words := ReadDictionary(dictionaryLocation)
	return &SchemaRenderer{words: words}
}

// CreateRendererUsingDefaultDictionary will create a new SchemaRenderer using the default dictionary file.
// The default dictionary is located at /usr/share/dict/words on most systems.
// Windows users will need to use CreateRendererUsingDictionary to specify a custom dictionary.
func CreateRendererUsingDefaultDictionary() *SchemaRenderer {
	wr := new(SchemaRenderer)
	wr.words = ReadDictionary("/usr/share/dict/words")
	return wr
}

// RenderSchema takes a schema and renders it into an interface, ready to be converted to JSON or YAML.
func (wr *SchemaRenderer) RenderSchema(schema *base.Schema) any {
	// dive into the schema and render it
	structure := make(map[string]any)
	wr.DiveIntoSchema(schema, rootType, structure, 0)
	return structure[rootType]
}

// DisableRequiredCheck will disable the required check when rendering a schema. This means that all properties
// will be rendered, not just the required ones.
// https://github.com/pb33f/libopenapi/issues/200
func (wr *SchemaRenderer) DisableRequiredCheck() {
	wr.disableRequired = true
}

// DiveIntoSchema will dive into a schema and inject values from examples into a map. If there are no examples in
// the schema, then the renderer will attempt to generate a value based on the schema type, format and pattern.
func (wr *SchemaRenderer) DiveIntoSchema(schema *base.Schema, key string, structure map[string]any, depth int) {
	// got an example? use it, we're done here.
	if schema.Example != nil {
		var example any
		_ = schema.Example.Decode(&example)

		structure[key] = example
		return
	}

	// emergency break to prevent stack overflow from ever occurring
	if depth > 100 {
		structure[key] = "to deep to continue rendering..."
		return
	}

	// render out a string.
	if slices.Contains(schema.Type, stringType) {
		// check for an enum, if there is one, then pick a random value from it.
		if schema.Enum != nil && len(schema.Enum) > 0 {
			enum := schema.Enum[rand.Int()%len(schema.Enum)]

			var example any
			_ = enum.Decode(&example)

			structure[key] = example
		} else {

			// generate a random value based on the schema format, pattern and length values.
			var minLength int64 = 3
			var maxLength int64 = 10

			if schema.MinLength != nil {
				minLength = *schema.MinLength
			}
			if schema.MaxLength != nil {
				maxLength = *schema.MaxLength
			}

			// if there are examples, use them.
			if schema.Examples != nil && len(schema.Examples) > 0 {
				var renderedExample any

				// multi examples and the type is an array? then render all examples.
				if len(schema.Examples) > 1 && key == itemsType {
					renderedExamples := make([]any, len(schema.Examples))
					for i, exmp := range schema.Examples {
						if exmp != nil {
							var ex any
							_ = exmp.Decode(&ex)
							renderedExamples[i] = fmt.Sprint(ex)
						}
					}
					structure[key] = renderedExamples
					return
				} else {
					// render the first example
					exmp := schema.Examples[0]
					if exmp != nil {
						var ex any
						_ = exmp.Decode(&ex)
						renderedExample = fmt.Sprint(ex)
					}
					structure[key] = renderedExample
					return
				}
			}

			switch schema.Format {
			case dateTimeType:
				structure[key] = time.Now().Format(time.RFC3339)
			case dateType:
				structure[key] = time.Now().Format("2006-01-02")
			case timeType:
				structure[key] = time.Now().Format("15:04:05")
			case emailType:
				structure[key] = fmt.Sprintf("%s@%s.com",
					wr.RandomWord(minLength, maxLength, 0),
					wr.RandomWord(minLength, maxLength, 0))
			case hostnameType:
				structure[key] = fmt.Sprintf("%s.com", wr.RandomWord(minLength, maxLength, 0))
			case ipv4Type:
				structure[key] = fmt.Sprintf("%d.%d.%d.%d",
					rand.Int()%255, rand.Int()%255, rand.Int()%255, rand.Int()%255)
			case ipv6Type:
				structure[key] = fmt.Sprintf("%04x:%04x:%04x:%04x:%04x:%04x:%04x:%04x",
					rand.Intn(65535), rand.Intn(65535), rand.Intn(65535), rand.Intn(65535),
					rand.Intn(65535), rand.Intn(65535), rand.Intn(65535), rand.Intn(65535),
				)
			case uriType:
				structure[key] = fmt.Sprintf("https://%s-%s-%s.com/%s",
					wr.RandomWord(minLength, maxLength, 0),
					wr.RandomWord(minLength, maxLength, 0),
					wr.RandomWord(minLength, maxLength, 0),
					wr.RandomWord(minLength, maxLength, 0))
			case uriReferenceType:
				structure[key] = fmt.Sprintf("/%s/%s",
					wr.RandomWord(minLength, maxLength, 0),
					wr.RandomWord(minLength, maxLength, 0))
			case uuidType:
				structure[key] = wr.PseudoUUID()
			case byteType:
				structure[key] = wr.RandomWord(minLength, maxLength, 0)
			case passwordType:
				structure[key] = wr.RandomWord(minLength, maxLength, 0)
			case binaryType:
				structure[key] = base64.StdEncoding.EncodeToString([]byte(wr.RandomWord(minLength, maxLength, 0)))
			case bigIntType:
				structure[key] = fmt.Sprint(wr.RandomInt(minLength, maxLength))
			case decimalType:
				structure[key] = fmt.Sprint(wr.RandomFloat64())
			default:
				// if there is a pattern supplied, then try and generate a string from it.
				if schema.Pattern != "" {
					str, err := reggen.Generate(schema.Pattern, int(maxLength))
					if err == nil {
						structure[key] = str
					}
				} else {
					// last resort, generate a random value
					structure[key] = wr.RandomWord(minLength, maxLength, 0)
				}
			}
		}
		return
	}

	// handle numbers
	if slices.Contains(schema.Type, numberType) ||
		slices.Contains(schema.Type, integerType) ||
		slices.Contains(schema.Type, bigIntType) ||
		slices.Contains(schema.Type, decimalType) {

		if schema.Enum != nil && len(schema.Enum) > 0 {
			enum := schema.Enum[rand.Int()%len(schema.Enum)]

			var example any
			_ = enum.Decode(&example)

			structure[key] = example
		} else {

			var minimum int64 = 1
			var maximum int64 = 100

			if schema.Minimum != nil {
				minimum = int64(*schema.Minimum)
			}
			if schema.Maximum != nil {
				maximum = int64(*schema.Maximum)
			}

			if schema.Examples != nil {
				if len(schema.Examples) > 0 {
					var renderedExample any
					exmp := schema.Examples[0]
					if exmp != nil {
						var ex any
						_ = exmp.Decode(&ex)
						renderedExample = ex
					}
					structure[key] = renderedExample
					return
				}
			}

			switch schema.Format {
			case floatType:
				structure[key] = rand.Float32()
			case doubleType:
				structure[key] = rand.Float64()
			case int32Type:
				structure[key] = int(wr.RandomInt(minimum, maximum))
			case bigIntType:
				structure[key] = wr.RandomInt(minimum, maximum)
			case decimalType:
				structure[key] = wr.RandomFloat64()
			default:
				structure[key] = wr.RandomInt(minimum, maximum)
			}
		}
		return
	}

	// handle booleans
	if slices.Contains(schema.Type, booleanType) {
		structure[key] = true
	}

	// handle objects
	if slices.Contains(schema.Type, objectType) {
		properties := schema.Properties
		propertyMap := make(map[string]any)

		if properties != nil {
			// check if this schema has required properties, if so, then only render required props, if not
			// render everything in the schema.
			checkProps := orderedmap.New[string, *base.SchemaProxy]()
			if !wr.disableRequired && len(schema.Required) > 0 {
				for _, requiredProp := range schema.Required {
					checkProps.Set(requiredProp, properties.GetOrZero(requiredProp))
				}
			} else {
				checkProps = properties
			}
			for pair := orderedmap.First(checkProps); pair != nil; pair = pair.Next() {
				// render property
				propName, propValue := pair.Key(), pair.Value()
				propertySchema := propValue.Schema()
				wr.DiveIntoSchema(propertySchema, propName, propertyMap, depth+1)
			}
		}

		// handle allOf
		allOf := schema.AllOf
		if allOf != nil {
			allOfMap := make(map[string]any)
			for _, allOfSchema := range allOf {
				allOfCompiled := allOfSchema.Schema()
				wr.DiveIntoSchema(allOfCompiled, allOfType, allOfMap, depth+1)
				if m, ok := allOfMap[allOfType].(map[string]any); ok {
					for k, v := range m {
						propertyMap[k] = v
					}
				}
				if m, ok := allOfMap[allOfType].(string); ok {
					propertyMap[allOfType] = m
				}
			}
		}

		// handle dependentSchemas
		dependentSchemas := schema.DependentSchemas
		if dependentSchemas != nil {
			dependentSchemasMap := make(map[string]any)
			for pair := orderedmap.First(dependentSchemas); pair != nil; pair = pair.Next() {
				// only map if the property exists
				k, dependentSchema := pair.Key(), pair.Value()
				if propertyMap[k] != nil {
					dependentSchemaCompiled := dependentSchema.Schema()
					wr.DiveIntoSchema(dependentSchemaCompiled, k, dependentSchemasMap, depth+1)
					for i, v := range dependentSchemasMap[k].(map[string]any) {
						propertyMap[k].(map[string]any)[i] = v
					}
				}
			}
		}

		// handle oneOf
		oneOf := schema.OneOf
		if len(oneOf) > 0 {
			oneOfMap := make(map[string]any)
			oneOfCompiled := oneOf[0].Schema()
			wr.DiveIntoSchema(oneOfCompiled, oneOfType, oneOfMap, depth+1)
			if m, ok := oneOfMap[oneOfType].(map[string]any); ok {
				for k, v := range m {
					propertyMap[k] = v
				}
			}
			if m, ok := oneOfMap[oneOfType].(string); ok {
				propertyMap[oneOfType] = m
			}
		}

		// handle anyOf
		anyOf := schema.AnyOf
		if len(anyOf) > 0 {
			anyOfMap := make(map[string]any)
			anyOfCompiled := anyOf[0].Schema()
			wr.DiveIntoSchema(anyOfCompiled, anyOfType, anyOfMap, depth+1)
			if m, ok := anyOfMap[anyOfType].(map[string]any); ok {
				for k, v := range m {
					propertyMap[k] = v
				}
			}
			if m, ok := anyOfMap[anyOfType].(string); ok {
				propertyMap[anyOfType] = m
			}
		}
		structure[key] = propertyMap
		return
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

				var renderedItems []any
				// build up the array
				for i := int64(0); i < minItems; i++ {
					itemMap := make(map[string]any)
					itemsSchemaCompiled := itemsSchema.A.Schema()
					wr.DiveIntoSchema(itemsSchemaCompiled, itemsType, itemMap, depth+1)
					if multipleItems, ok := itemMap[itemsType].([]any); ok {
						renderedItems = multipleItems
					} else {
						renderedItems = append(renderedItems, itemMap[itemsType])
					}
				}
				structure[key] = renderedItems
				return
			}
		}
	}
}

func readFile(file io.Reader) []string {
	bytes, err := io.ReadAll(file)
	if err != nil {
		return []string{}
	}
	return strings.Split(string(bytes), "\n")
}

// ReadDictionary will read a dictionary file and return a slice of strings.
func ReadDictionary(dictionaryLocation string) []string {
	file, err := os.Open(dictionaryLocation)
	if err != nil {
		return []string{}
	}
	return readFile(file)
}

// RandomWord will return a random word from the dictionary file between the min and max values. The depth is used
// to prevent a stack overflow, the maximum depth is 100 (anything more than this is probably a bug).
// set the values to 0 to return the first word returned, essentially ignore the min and max values.
func (wr *SchemaRenderer) RandomWord(min, max int64, depth int) string {
	// break out if we've gone too deep
	if depth > 100 {
		return fmt.Sprintf("no-word-found-%d-%d", min, max)
	}

	// no dictionary? then just return a random string.
	if len(wr.words) == 0 {
		if min == 0 {
			min = 7 // seems like a good default
		}
		b := make([]byte, min)
		for i := range b {
			b[i] = letterBytes[rand.Intn(len(letterBytes))]
		}
		return string(b)
	}

	word := wr.words[rand.Int()%len(wr.words)]
	if min == 0 && max == 0 {
		return word
	}
	if len(word) < int(min) || len(word) > int(max) {
		return wr.RandomWord(min, max, depth+1)
	}
	return word
}

// RandomInt will return a random int between the min and max values.
func (wr *SchemaRenderer) RandomInt(min, max int64) int64 {
	return rand.Int63n(max-min) + min
}

// RandomFloat64 will return a random float64 between 0 and 1.
func (wr *SchemaRenderer) RandomFloat64() float64 {
	return rand.Float64()
}

// PseudoUUID will return a random UUID, it's not a real UUID, but it's good enough for mock /example data.
func (wr *SchemaRenderer) PseudoUUID() string {
	b := make([]byte, 16)
	_, _ = cryptoRand.Read(b)
	return strings.ToLower(fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]))
}
