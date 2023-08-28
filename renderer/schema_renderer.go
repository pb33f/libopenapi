// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package renderer

import (
	cryptoRand "crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/lucasjones/reggen"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"golang.org/x/exp/slices"
	"io"
	"math/rand"
	"os"
	"strings"
	"time"
)

// used to generate random words if there is no dictionary applied.
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func init() {
	// create a new random seed
	rand.New(rand.NewSource(time.Now().UnixNano()))
}

// RenderSchema takes a schema and renders it into a map[string]any, ready to be converted to JSON or YAML.
func (wr *SchemaRenderer) RenderSchema(schema *base.Schema) map[string]any {
	// dive into the schema and render it
	structure := make(map[string]any)
	wr.DiveIntoSchema(schema, "root", structure, 0)
	return structure["root"].(map[string]any)
}

// DiveIntoSchema will dive into a schema and inject values from examples into a map. If there are no examples in
// the schema, then the renderer will attempt to generate a value based on the schema type, format and pattern.
func (wr *SchemaRenderer) DiveIntoSchema(schema *base.Schema, key string, structure map[string]any, depth int) {

	// got an example? use it, we're done here.
	if schema.Example != nil {
		structure[key] = schema.Example
		return
	}

	// emergency break to prevent stack overflow from ever occurring
	if depth > 100 {
		structure[key] = "to deep to continue rendering..."
		return
	}

	// render out a string.
	if slices.Contains(schema.Type, "string") {
		// check for an enum, if there is one, then pick a random value from it.
		if schema.Enum != nil && len(schema.Enum) > 0 {
			structure[key] = schema.Enum[rand.Int()%len(schema.Enum)]
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

			switch schema.Format {
			case "date-time":
				structure[key] = time.Now().Format(time.RFC3339)
			case "date":
				structure[key] = time.Now().Format("2006-01-02")
			case "time":
				structure[key] = time.Now().Format("15:04:05")
			case "email":
				structure[key] = fmt.Sprintf("%s@%s.com",
					wr.RandomWord(minLength, maxLength, 0),
					wr.RandomWord(minLength, maxLength, 0))
			case "hostname":
				structure[key] = fmt.Sprintf("%s.com", wr.RandomWord(minLength, maxLength, 0))
			case "ipv4":
				structure[key] = fmt.Sprintf("%d.%d.%d.%d",
					rand.Int()%255, rand.Int()%255, rand.Int()%255, rand.Int()%255)
			case "ipv6":
				structure[key] = fmt.Sprintf("%04x:%04x:%04x:%04x:%04x:%04x:%04x:%04x",
					rand.Intn(65535), rand.Intn(65535), rand.Intn(65535), rand.Intn(65535),
					rand.Intn(65535), rand.Intn(65535), rand.Intn(65535), rand.Intn(65535),
				)
			case "uri":
				structure[key] = fmt.Sprintf("https://%s-%s-%s.com/%s",
					wr.RandomWord(minLength, maxLength, 0),
					wr.RandomWord(minLength, maxLength, 0),
					wr.RandomWord(minLength, maxLength, 0),
					wr.RandomWord(minLength, maxLength, 0))
			case "uri-reference":
				structure[key] = fmt.Sprintf("/%s/%s",
					wr.RandomWord(minLength, maxLength, 0),
					wr.RandomWord(minLength, maxLength, 0))
			case "uuid":
				structure[key] = wr.PseudoUUID()
			case "byte":
				structure[key] = fmt.Sprintf("%x", wr.RandomWord(minLength, maxLength, 0))
			case "password":
				structure[key] = fmt.Sprintf("%s", wr.RandomWord(minLength, maxLength, 0))
			case "binary":
				structure[key] = fmt.Sprintf("%s",
					base64.StdEncoding.EncodeToString([]byte(wr.RandomWord(minLength, maxLength, 0))))
			default:
				// if there is a pattern supplied, then try and generate a string from it.
				if schema.Pattern != "" {
					str, err := reggen.Generate(schema.Pattern, int(maxLength))
					if err == nil {
						structure[key] = str
					}
				} else {
					structure[key] = wr.RandomWord(minLength, maxLength, 0)
				}
			}
		}
		return
	}

	// handle numbers
	if slices.Contains(schema.Type, "number") || slices.Contains(schema.Type, "integer") {

		if schema.Enum != nil && len(schema.Enum) > 0 {
			structure[key] = schema.Enum[rand.Int()%len(schema.Enum)]
		} else {

			var minimum int64 = 1
			var maximum int64 = 100

			if schema.Minimum != nil {
				minimum = int64(*schema.Minimum)
			}
			if schema.Maximum != nil {
				maximum = int64(*schema.Maximum)
			}

			switch schema.Format {
			case "float":
				structure[key] = rand.Float32()
			case "double":
				structure[key] = rand.Float64()
			case "int32":
				structure[key] = int(wr.RandomInt(minimum, maximum))
			default:
				structure[key] = wr.RandomInt(minimum, maximum)
			}
		}
		return
	}

	// handle booleans
	if slices.Contains(schema.Type, "boolean") {
		structure[key] = true
	}

	// handle objects
	if slices.Contains(schema.Type, "object") {
		properties := schema.Properties
		propertyMap := make(map[string]any)

		if properties != nil {
			// check if this schema has required properties, if so, then only render required props, if not
			// render everything in the schema.
			checkProps := make(map[string]*base.SchemaProxy)
			if len(schema.Required) > 0 {
				for _, requiredProp := range schema.Required {
					checkProps[requiredProp] = properties[requiredProp]
				}
			} else {
				checkProps = properties
			}
			for propName, propValue := range checkProps {
				// render property
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
				wr.DiveIntoSchema(allOfCompiled, "allOf", allOfMap, depth+1)
				for k, v := range allOfMap["allOf"].(map[string]any) {
					propertyMap[k] = v
				}
			}
		}

		// handle dependentSchemas
		dependentSchemas := schema.DependentSchemas
		if dependentSchemas != nil {
			dependentSchemasMap := make(map[string]any)
			for k, dependentSchema := range dependentSchemas {
				// only map if the property exists
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
		if oneOf != nil {
			oneOfMap := make(map[string]any)
			for _, oneOfSchema := range oneOf {
				oneOfCompiled := oneOfSchema.Schema()
				wr.DiveIntoSchema(oneOfCompiled, "oneOf", oneOfMap, depth+1)
				for k, v := range oneOfMap["oneOf"].(map[string]any) {
					propertyMap[k] = v
				}
				break // one run once for the first result.
			}
		}

		// handle anyOf
		anyOf := schema.AnyOf
		if anyOf != nil {
			anyOfMap := make(map[string]any)
			for _, anyOfSchema := range anyOf {
				anyOfCompiled := anyOfSchema.Schema()
				wr.DiveIntoSchema(anyOfCompiled, "anyOf", anyOfMap, depth+1)
				for k, v := range anyOfMap["anyOf"].(map[string]any) {
					propertyMap[k] = v
				}
				break // one run once for the first result only, same as oneOf
			}
		}
		structure[key] = propertyMap
		return
	}

	if slices.Contains(schema.Type, "array") {

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
					wr.DiveIntoSchema(itemsSchemaCompiled, "items", itemMap, depth+1)
					renderedItems = append(renderedItems, itemMap["items"])
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

// SchemaRenderer is a renderer that will generate random words, numbers and values based on a dictionary file.
type SchemaRenderer struct {
	words []string
}

func CreateRendererUsingDictionary(dictionaryLocation string) *SchemaRenderer {
	// try and read in the dictionary file
	words := ReadDictionary(dictionaryLocation)
	return &SchemaRenderer{words: words}
}

// CreateRendererUsingDefaultDictionary will create a new SchemaRenderer using the default dictionary file.
func CreateRendererUsingDefaultDictionary() *SchemaRenderer {
	wr := new(SchemaRenderer)
	wr.words = ReadDictionary("/usr/share/dict/words")
	return wr
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
