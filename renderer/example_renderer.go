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

var randomWord *WordRenderer

func init() {
    rand.New(rand.NewSource(time.Now().UnixNano()))
    randomWord = CreateRendererUsingDefaultDictionary()
}

// RenderSchema takes a schema and renders it into a map[string]any, ready to be converted to JSON or YAML.
func RenderSchema(schema *base.Schema) map[string]any {
    // dive into the schema and render it
    structure := make(map[string]any)
    DiveIntoSchema(schema, "root", structure, 0)
    return structure["root"].(map[string]any)
}

// DiveIntoSchema will dive into a schema and inject values from examples into a map. If there are no examples in
// the schema, then the renderer will attempt to generate a value based on the schema type, format and pattern.
func DiveIntoSchema(schema *base.Schema, key string, structure map[string]any, depth int) {

    if schema.Example != nil {
        structure[key] = schema.Example
        return
    }

    if depth > 100 {
        structure[key] = "to deep to continue rendering..."
        return
    }

    // TODO: handle required, minItems, maxItems, uniqueItems, minProperties, maxProperties, patternProperties, additionalProperties
	
    var minLength int64 = 3
    var maxLength int64 = 10
    var minimum int64 = 1
    var maximum int64 = 100
    if schema.MinLength != nil {
        minLength = *schema.MinLength
    }
    if schema.MaxLength != nil {
        maxLength = *schema.MaxLength
    }
    if schema.Minimum != nil {
        minimum = int64(*schema.Minimum)
    }
    if schema.Maximum != nil {
        maximum = int64(*schema.Maximum)
    }

    if slices.Contains(schema.Type, "string") {

        if schema.Enum != nil && len(schema.Enum) > 0 {
            structure[key] = schema.Enum[rand.Int()%len(schema.Enum)]
        } else {

            switch schema.Format {
            case "date-time":
                structure[key] = time.Now().Format(time.RFC3339)
            case "date":
                structure[key] = time.Now().Format("2006-01-02")
            case "time":
                structure[key] = time.Now().Format("15:04:05")
            case "email":
                structure[key] = fmt.Sprintf("%s@%s.com",
                    randomWord.RandomWord(minLength, maxLength, 0),
                    randomWord.RandomWord(minLength, maxLength, 0))
            case "hostname":
                structure[key] = fmt.Sprintf("%s.com", randomWord.RandomWord(minLength, maxLength, 0))
            case "ipv4":
                structure[key] = fmt.Sprintf("%d.%d.%d.%d", rand.Int()%255, rand.Int()%255, rand.Int()%255, rand.Int()%255)
            case "ipv6":
                structure[key] = fmt.Sprintf("%04x:%04x:%04x:%04x:%04x:%04x:%04x:%04x",
                    rand.Intn(65535), rand.Intn(65535), rand.Intn(65535), rand.Intn(65535),
                    rand.Intn(65535), rand.Intn(65535), rand.Intn(65535), rand.Intn(65535),
                )
            case "uri":
                structure[key] = fmt.Sprintf("https://%s-%s-%s.com/%s",
                    randomWord.RandomWord(minLength, maxLength, 0),
                    randomWord.RandomWord(minLength, maxLength, 0),
                    randomWord.RandomWord(minLength, maxLength, 0),
                    randomWord.RandomWord(minLength, maxLength, 0))
            case "uri-reference":
                structure[key] = fmt.Sprintf("/%s/%s",
                    randomWord.RandomWord(minLength, maxLength, 0),
                    randomWord.RandomWord(minLength, maxLength, 0))
            case "uuid":
                structure[key] = randomWord.PseudoUUID()
            case "byte":
                structure[key] = fmt.Sprintf("%x", randomWord.RandomWord(minLength, maxLength, 0))
            case "password":
                structure[key] = fmt.Sprintf("%s", randomWord.RandomWord(minLength, maxLength, 0))
            case "binary":
                structure[key] = fmt.Sprintf("%s",
                    base64.StdEncoding.EncodeToString([]byte(randomWord.RandomWord(minLength, maxLength, 0))))
            default:
                // if there is a pattern supplied, then try and generate a string from it.
                if schema.Pattern != "" {
                    str, err := reggen.Generate(schema.Pattern, int(maxLength))
                    if err == nil {
                        structure[key] = str
                    }
                } else {
                    structure[key] = randomWord.RandomWord(minLength, maxLength, 0)
                }
            }
        }
        return
    }

    if slices.Contains(schema.Type, "number") || slices.Contains(schema.Type, "integer") {

        if schema.Enum != nil && len(schema.Enum) > 0 {
            structure[key] = schema.Enum[rand.Int()%len(schema.Enum)]
        } else {
            switch schema.Format {
            case "float":
                structure[key] = rand.Float32()
            case "double":
                structure[key] = rand.Float64()
            case "int32":
                structure[key] = int(randomWord.RandomInt(minimum, maximum))
            default:
                structure[key] = randomWord.RandomInt(minimum, maximum)
            }
        }
        return
    }

    if slices.Contains(schema.Type, "boolean") {
        structure[key] = true
    }

    if slices.Contains(schema.Type, "object") {
        properties := schema.Properties
        propertyMap := make(map[string]any)
        if properties != nil {
            for propName, propValue := range properties {
                // render property
                propertySchema := propValue.Schema()
                DiveIntoSchema(propertySchema, propName, propertyMap, depth+1)
            }
        }

        // handle allOf
        allOf := schema.AllOf
        if allOf != nil {
            allOfMap := make(map[string]any)
            for _, allOfSchema := range allOf {
                allOfCompiled := allOfSchema.Schema()
                DiveIntoSchema(allOfCompiled, "allOf", allOfMap, depth+1)
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
                    DiveIntoSchema(dependentSchemaCompiled, k, dependentSchemasMap, depth+1)
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
                DiveIntoSchema(oneOfCompiled, "oneOf", oneOfMap, depth+1)
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
                DiveIntoSchema(anyOfCompiled, "anyOf", anyOfMap, depth+1)
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
                itemMap := make(map[string]any)
                itemsSchemaCompiled := itemsSchema.A.Schema()
                DiveIntoSchema(itemsSchemaCompiled, "items", itemMap, depth+1)
                structure[key] = []interface{}{itemMap["items"]}
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

// WordRenderer is a renderer that will generate random words, numbers and values based on a dictionary file.
type WordRenderer struct {
    count int
    words []string
}

// CreateRendererUsingDefaultDictionary will create a new WordRenderer using the default dictionary file.
func CreateRendererUsingDefaultDictionary() *WordRenderer {
    wr := new(WordRenderer)
    wr.words = ReadDictionary("/usr/share/dict/words")
    return wr
}

// RandomWord will return a random word from the dictionary file between the min and max values. The depth is used
// to prevent a stack overflow, the maximum depth is 100 (anything more than this is probably a bug).
// set the values to 0 to return the first word returned, essentially ignore the min and max values.
func (wr *WordRenderer) RandomWord(min, max int64, depth int) string {
    if depth > 100 {
        return "no-word"
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
func (wr *WordRenderer) RandomInt(min, max int64) int64 {
    return rand.Int63n(max-min) + min
}

// RandomFloat64 will return a random float64 between 0 and 1.
func (wr *WordRenderer) RandomFloat64() float64 {
    return rand.Float64()
}

// PseudoUUID will return a random UUID, it's not a real UUID, but it's good enough for mock /example data.
func (wr *WordRenderer) PseudoUUID() string {
    b := make([]byte, 16)
    _, _ = cryptoRand.Read(b)
    return strings.ToLower(fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]))
}
