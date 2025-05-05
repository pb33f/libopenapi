// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package renderer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestRenderSchema(t *testing.T) {
	testObject := `type: [object]
properties:
  name:
    type: string
    enum: [pb33f]`

	compiled := getSchema([]byte(testObject))
	wr := createSchemaRenderer()
	schema := wr.RenderSchema(compiled)
	rendered, _ := json.Marshal(schema)
	assert.Equal(t, `{"name":"pb33f"}`, string(rendered))
}

func createSchemaRenderer() *SchemaRenderer {
	osDict := "/usr/share/dict/words"
	if _, err := os.Stat(osDict); err != nil {
		osDict = ""
	}
	if osDict != "" {
		return CreateRendererUsingDefaultDictionary()
	}

	// return empty renderer, will generate random strings
	return &SchemaRenderer{}
}

func getSchema(schema []byte) *highbase.Schema {
	var compNode yaml.Node
	e := yaml.Unmarshal(schema, &compNode)
	if e != nil {
		panic(e)
	}
	sp := new(lowbase.SchemaProxy)
	_ = sp.Build(context.Background(), nil, compNode.Content[0], nil)
	lp := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     sp,
		ValueNode: compNode.Content[0],
	}
	schemaProxy := highbase.NewSchemaProxy(&lp)
	return schemaProxy.Schema()
}

func createVisitedMap() map[string]bool {
  return make(map[string]bool)
}

func TestRenderExample_StringWithExample(t *testing.T) {
	testObject := `type: string
example: dog`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
  wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.Equal(t, journeyMap["pb33f"], "dog")
}

func TestRenderExample_StringWithNoExample(t *testing.T) {
	testObject := `type: string`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.GreaterOrEqual(t, len(journeyMap["pb33f"].(string)), 3)
	assert.LessOrEqual(t, len(journeyMap["pb33f"].(string)), 10)
}

func TestRenderExample_StringWithNoExample_Format_Datetime(t *testing.T) {
	testObject := `type: string
format: date-time`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	now := time.Now()
	assert.NotNil(t, journeyMap["pb33f"])
	assert.Equal(t, journeyMap["pb33f"].(string)[:4], fmt.Sprintf("%d", now.Year()))
}

func TestRenderExample_StringWithNoExample_Pattern_Email(t *testing.T) {
	testObject := `type: string
pattern: "^[a-z]{5,10}@[a-z]{5,10}\\.(com|net|org)$"` // an email address

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])

	value := journeyMap["pb33f"].(string)
	emailRegex, _ := regexp.MatchString("^[a-z]{5,10}@[a-z]{5,10}\\.(com|net|org)$", value)

	assert.True(t, emailRegex)
}

func TestRenderExample_StringWithNoExample_Pattern_PhoneNumber(t *testing.T) {
	testObject := `type: string
pattern: "^\\([0-9]{3}\\)-[0-9]{3}-[0-9]{4}$"` // a phone number

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])

	value := journeyMap["pb33f"].(string)
	phoneRegex, _ := regexp.MatchString("^\\([0-9]{3}\\)-[0-9]{3}-[0-9]{4}$", value)
	assert.True(t, phoneRegex)
}

func TestRenderExample_StringWithNoExample_Format_Date(t *testing.T) {
	testObject := `type: string
format: date`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	now := time.Now().Format("2006-01-02")

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Equal(t, journeyMap["pb33f"].(string), now)
}

func TestRenderExample_StringWithNoExample_Format_Time(t *testing.T) {
	testObject := `type: string
format: time`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	now := time.Now().Format("15:04:05")

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Equal(t, now, journeyMap["pb33f"].(string))
}

func TestRenderExample_StringWithNoExample_Email(t *testing.T) {
	testObject := `type: string
format: email`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.True(t, strings.Contains(journeyMap["pb33f"].(string), "@"))
}

func TestRenderExample_StringWithNoExample_Hostname(t *testing.T) {
	testObject := `type: string
format: hostname`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.True(t, strings.Contains(journeyMap["pb33f"].(string), ".com"))
}

func TestRenderExample_StringWithNoExample_ivp4(t *testing.T) {
	testObject := `type: string
format: ipv4`

	compiled := getSchema([]byte(testObject))

	visited := createVisitedMap()
	journeyMap := make(map[string]any)
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	segs := strings.Split(journeyMap["pb33f"].(string), ".")
	assert.Len(t, segs, 4)
}

func TestRenderExample_StringWithNoExample_ivp6(t *testing.T) {
	testObject := `type: string
format: ipv6`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	segs := strings.Split(journeyMap["pb33f"].(string), ":")
	assert.Len(t, segs, 8)
}

func TestRenderExample_StringWithNoExample_URI(t *testing.T) {
	testObject := `type: string
format: uri`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	uri, e := url.Parse(journeyMap["pb33f"].(string))
	assert.Nil(t, e)
	assert.Equal(t, uri.Scheme, "https")
	assert.True(t, strings.Contains(uri.Host, ".com"))
}

func TestRenderExample_StringWithNoExample_UUID(t *testing.T) {
	testObject := `type: string
format: uuid`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	segs := strings.Split(journeyMap["pb33f"].(string), "-")
	assert.Len(t, segs, 5)
}

func TestRenderExample_StringWithNoExample_Byte(t *testing.T) {
	testObject := `type: string
format: byte`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.NotEmpty(t, journeyMap["pb33f"].(string))
}

func TestRenderExample_StringWithNoExample_Password(t *testing.T) {
	testObject := `type: string
format: password`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.NotEmpty(t, journeyMap["pb33f"].(string))
}

func TestRenderExample_StringWithNoExample_URIReference(t *testing.T) {
	testObject := `type: string
format: uri-reference`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	segs := strings.Split(journeyMap["pb33f"].(string), "/")
	assert.Len(t, segs, 3)
}

func TestRenderExample_StringWithNoExample_Binary(t *testing.T) {
	testObject := `type: string
format: binary`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	encoded := journeyMap["pb33f"].(string)
	decodedString, err := base64.StdEncoding.DecodeString(encoded)
	assert.NoError(t, err)
	assert.NotNil(t, journeyMap["pb33f"])
	assert.NotEmpty(t, decodedString)
}

func TestRenderExample_NumberWithExample(t *testing.T) {
	testObject := `type: number
example: 3.14`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Equal(t, 3.14, journeyMap["pb33f"])
}

func TestRenderExample_MinLength(t *testing.T) {
	testObject := `type: string
minLength: 10`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.GreaterOrEqual(t, len(journeyMap["pb33f"].(string)), 10)
}

func TestRenderExample_MaxLength(t *testing.T) {
	testObject := `type: string
maxLength: 10`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.LessOrEqual(t, len(journeyMap["pb33f"].(string)), 10)
}

func TestRenderExample_MaxLength_MinLength(t *testing.T) {
	testObject := `type: string
maxLength: 8
minLength: 3`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.LessOrEqual(t, len(journeyMap["pb33f"].(string)), 8)
	assert.GreaterOrEqual(t, len(journeyMap["pb33f"].(string)), 3)
}

func TestRenderExample_NumberNoExample_Default(t *testing.T) {
	testObject := `type: number`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Greater(t, journeyMap["pb33f"], int64(0))
	assert.Less(t, journeyMap["pb33f"], int64(100))
}

func TestRenderExample_Number_Minimum(t *testing.T) {
	testObject := `type: number
minimum: 60`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.GreaterOrEqual(t, journeyMap["pb33f"], int64(60))
	assert.Less(t, journeyMap["pb33f"], int64(100))
}

func TestRenderExample_Number_Maximum(t *testing.T) {
	testObject := `type: number
maximum: 4`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.GreaterOrEqual(t, journeyMap["pb33f"], int64(1))
	assert.LessOrEqual(t, journeyMap["pb33f"], int64(4))
}

func TestRenderExample_NumberNoExample_Float(t *testing.T) {
	testObject := `type: number
format: float`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Greater(t, journeyMap["pb33f"], float32(0))
}

func TestRenderExample_NumberNoExample_Float64(t *testing.T) {
	testObject := `type: number
format: double`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Greater(t, journeyMap["pb33f"], float64(0))
}

func TestRenderExample_NumberNoExample_Int32(t *testing.T) {
	testObject := `type: number
format: int32`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Greater(t, journeyMap["pb33f"], 0)
}

func TestRenderExample_NumberNoExample_Int64(t *testing.T) {
	testObject := `type: number
format: int64`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Greater(t, journeyMap["pb33f"], int64(0))
}

func TestRenderExample_Boolean_WithExample(t *testing.T) {
	testObject := `type: boolean
example: true`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.True(t, journeyMap["pb33f"].(bool))
}

func TestRenderExample_Boolean_WithoutExample(t *testing.T) {
	testObject := `type: boolean`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.True(t, journeyMap["pb33f"].(bool))
}

func TestRenderExample_Array_String(t *testing.T) {
	testObject := `type: array
items:
  type: string`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Len(t, journeyMap["pb33f"], 1)
	assert.Greater(t, len(journeyMap["pb33f"].([]interface{})[0].(string)), 0)
}

func TestRenderExample_Array_Number(t *testing.T) {
	testObject := `type: array
items:
  type: number`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Len(t, journeyMap["pb33f"], 1)
	assert.Greater(t, journeyMap["pb33f"].([]interface{})[0].(int64), int64(0))
}

func TestRenderExample_Array_MinItems(t *testing.T) {
	testObject := `type: array
minItems: 3
items:
  type: string`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Len(t, journeyMap["pb33f"], 3)
}

// TODO: object array!

func TestRenderExample_Object_StringProps(t *testing.T) {
	testObject := `type: object
properties:
  fishCake:
    type: string
  fishPie:
    type: string`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Len(t, journeyMap["pb33f"], 2)
	assert.Greater(t, len(journeyMap["pb33f"].(map[string]interface{})["fishPie"].(string)), 0)
}

func TestRenderExample_Object_String_Object(t *testing.T) {
	testObject := `type: object
properties:
  fishCake:
    type: object
    properties:
      bones:
        type: string
  fishPie:
    type: object
    properties:
      cream:
        type: number
        format: double`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Len(t, journeyMap["pb33f"], 2)
	assert.Greater(t, journeyMap["pb33f"].(map[string]interface{})["fishPie"].(map[string]interface{})["cream"].(float64), float64(0))
	assert.Greater(t, len(journeyMap["pb33f"].(map[string]interface{})["fishCake"].(map[string]interface{})["bones"].(string)), 0)
}

func TestRenderExample_Object_AllOf(t *testing.T) {
	testObject := `type: object
allOf:
  - type: object
    properties:
      bones:
        type: string
  - type: object
    properties:
      cream:
        type: number
        format: double`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Len(t, journeyMap["pb33f"], 2)
	assert.Greater(t, journeyMap["pb33f"].(map[string]interface{})["cream"].(float64), float64(0))
	assert.Greater(t, len(journeyMap["pb33f"].(map[string]interface{})["bones"].(string)), 0)
}

func TestRenderExample_Object_DependentSchemas(t *testing.T) {
	testObject := `type: object
properties:
  fishCake:
    type: object
    properties:
      bones:
        type: boolean
dependentSchemas:
  fishCake:
    type: object
    properties:
      cream:
        type: number
        format: double`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Len(t, journeyMap["pb33f"], 1)
	assert.Greater(t, journeyMap["pb33f"].(map[string]interface{})["fishCake"].(map[string]interface{})["cream"].(float64), float64(0))
	assert.True(t, journeyMap["pb33f"].(map[string]interface{})["fishCake"].(map[string]interface{})["bones"].(bool))
}


func TestRenderExample_String_AllOf(t *testing.T) {
	testObject := `type: object
allOf:
  - type: string`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Len(t, journeyMap["pb33f"], 1)
	assert.Greater(t, len(journeyMap["pb33f"].(map[string]interface{})["allOf"].(string)), 0)
}

func TestRenderExample_Object_OneOf(t *testing.T) {
	testObject := `type: object
oneOf:
  - type: object
    properties:
      bones:
        type: string
  - type: object
    properties:
      cream:
        type: number
        format: double`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Len(t, journeyMap["pb33f"], 1)
	assert.Greater(t, len(journeyMap["pb33f"].(map[string]interface{})["bones"].(string)), 0)
}

func TestRenderExample_String_OneOf(t *testing.T) {
	testObject := `type: object
oneOf:
  - type: string`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Len(t, journeyMap["pb33f"], 1)
	assert.Greater(t, len(journeyMap["pb33f"].(map[string]interface{})["oneOf"].(string)), 0)
}

func TestRenderExample_Object_AnyOf(t *testing.T) {
	testObject := `type: object
anyOf:
  - type: object
    properties:
      bones:
        type: string
  - type: object
    properties:
      cream:
        type: number
        format: double`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Len(t, journeyMap["pb33f"], 1)
	assert.Greater(t, len(journeyMap["pb33f"].(map[string]interface{})["bones"].(string)), 0)
}

func TestRenderExample_String_AnyOf(t *testing.T) {
	testObject := `type: object
anyOf:
  - type: string`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Len(t, journeyMap["pb33f"], 1)
	assert.Greater(t, len(journeyMap["pb33f"].(map[string]interface{})["anyOf"].(string)), 0)
}

func TestRenderExample_TestGiftshopProduct_UsingExamples(t *testing.T) {
	testObject := `type: [object]
properties:
  id:
    type: [string]
    maxLength: 50
    format: uuid
    example: d1404c5c-69bd-4cd2-a4cf-b47c79a30112
  shortCode:
    type: [ string ]
    maxLength: 50
    format: string
    example: "pb0001"
  name:
    type: [string]
    description: "The name of the product"
    maxLength: 50
    format: string
    example: "pb33f t-shirt"
  description:
    type: [string]
    maxLength: 500
    format: string
    example: "A t-shirt with the pb33f logo on the front"
  price:
    type: [number]
    format: float
    maxLength: 5
    example: 19.99
  category:
    type: [string]
    maxLength: 50
    format: string
    example: "shirts"
  image:
    type: [string]
    maxLength: 100
    format: string
    example: "https://pb33f.io/images/t-shirt.png"`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Equal(t, journeyMap["pb33f"].(map[string]interface{})["id"].(string), "d1404c5c-69bd-4cd2-a4cf-b47c79a30112")
	assert.Equal(t, journeyMap["pb33f"].(map[string]interface{})["shortCode"].(string), "pb0001")
	assert.Equal(t, journeyMap["pb33f"].(map[string]interface{})["name"].(string), "pb33f t-shirt")
	assert.Equal(t, journeyMap["pb33f"].(map[string]interface{})["description"].(string), "A t-shirt with the pb33f logo on the front")
	assert.Equal(t, journeyMap["pb33f"].(map[string]interface{})["price"].(float64), 19.99)
	assert.Equal(t, journeyMap["pb33f"].(map[string]interface{})["category"].(string), "shirts")
	assert.Equal(t, journeyMap["pb33f"].(map[string]interface{})["image"].(string), "https://pb33f.io/images/t-shirt.png")
}

func TestRenderExample_TestGiftshopProduct_UsingTopLevelExample(t *testing.T) {
	testObject := `type: [object]
properties:
  id:
    type: [string]
    maxLength: 50
    format: uuid
    example: d1404c5c-69bd-4cd2-a4cf-b47c79a30112
  shortCode:
    type: [ string ]
    maxLength: 50
    format: string
    example: "pb0001"
  name:
    type: [string]
    description: "The name of the product"
    maxLength: 50
    format: string
    example: "pb33f t-shirt"
  description:
    type: [string]
    maxLength: 500
    format: string
    example: "A t-shirt with the pb33f logo on the front"
  price:
    type: [number]
    format: float
    maxLength: 5
    example: 19.99
  category:
    type: [string]
    maxLength: 50
    format: string
    example: "shirts"
  image:
    type: [string]
    maxLength: 100
    format: string
    example: "https://pb33f.io/images/t-shirt.png"
example:
  id: not-a-uuid
  shortCode: not-a-shortcode
  name: not-a-name
  description: not-a-description
  price: not-a-price
  category: not-a-category
  image: not-an-image`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Equal(t, journeyMap["pb33f"].(map[string]interface{})["id"].(string), "not-a-uuid")
	assert.Equal(t, journeyMap["pb33f"].(map[string]interface{})["shortCode"].(string), "not-a-shortcode")
	assert.Equal(t, journeyMap["pb33f"].(map[string]interface{})["name"].(string), "not-a-name")
	assert.Equal(t, journeyMap["pb33f"].(map[string]interface{})["description"].(string), "not-a-description")

	// examples can completely override the schema, so this is a string now
	assert.Equal(t, journeyMap["pb33f"].(map[string]interface{})["price"].(string), "not-a-price")

	assert.Equal(t, journeyMap["pb33f"].(map[string]interface{})["category"].(string), "not-a-category")
	assert.Equal(t, journeyMap["pb33f"].(map[string]interface{})["image"].(string), "not-an-image")
}

func TestRenderExample_TestGiftshopProduct_UsingNoExamples(t *testing.T) {
	testObject := `type: [object]
properties:
  id:
    type: [string]
    maxLength: 50
    format: uuid
  shortCode:
    type: [ string ]
    maxLength: 50
    format: string
  name:
    type: [string]
    description: "The name of the product"
    maxLength: 50
    format: string
  description:
    type: [string]
    maxLength: 500
    format: string
  price:
    type: [number]
    format: float
    maxLength: 5
  category:
    type: [string]
    maxLength: 50
    format: string
  image:
    type: [string]
    maxLength: 100
    format: string`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.NotEmpty(t, journeyMap["pb33f"].(map[string]interface{})["id"].(string))
	assert.NotEmpty(t, journeyMap["pb33f"].(map[string]interface{})["shortCode"].(string))
	assert.NotEmpty(t, journeyMap["pb33f"].(map[string]interface{})["name"].(string))
	assert.NotEmpty(t, journeyMap["pb33f"].(map[string]interface{})["description"])
	assert.NotEmpty(t, journeyMap["pb33f"].(map[string]interface{})["price"].(float32))
	assert.NotEmpty(t, journeyMap["pb33f"].(map[string]interface{})["category"].(string))
	assert.NotEmpty(t, journeyMap["pb33f"].(map[string]interface{})["image"].(string))
}

func TestRenderExample_Test_MultiPolymorphic(t *testing.T) {
	testObject := `type: [object]
properties:
  burger:
    type: [object]
    properties:
      name:
        type: string
        pattern: "^(Big|Heavy|Junior) (Grilled|Broiled|Fried) (Burger|Cheese Burger|Hotdog)$"
    allOf:
      - type: [object]
        properties:
          patty:
            type: string
            enum: [beef, pork, vegetables]
          weight:
            type: number
            format: integer
            enum: [8, 12, 16, 18]
    oneOf:
      - type: [object]
        properties:
          frozen:
            type: boolean
      - type: [object]
        properties:
          seasoned:
            type: boolean`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	burger := journeyMap["pb33f"].(map[string]interface{})["burger"].(map[string]interface{})
	assert.NotNil(t, burger)
	assert.NotEmpty(t, burger["name"].(string))
	assert.NotZero(t, burger["weight"].(int))
	assert.NotEmpty(t, burger["patty"].(string))
	assert.True(t, burger["frozen"].(bool))
}

func TestRenderExample_Test_RequiredRendered(t *testing.T) {
	testObject := `type: [object]
required:
  - drink
properties:
  burger:
    type: string
  fries:
    type: string
  drink:
    type: string`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	drink := journeyMap["pb33f"].(map[string]interface{})["drink"].(string)
	assert.NotNil(t, drink)
	assert.Nil(t, journeyMap["pb33f"].(map[string]interface{})["burger"])
	assert.Nil(t, journeyMap["pb33f"].(map[string]interface{})["fries"])
}

func TestRenderExample_Test_RequiredWithMissingProp(t *testing.T) {
	testObject := `type: [object]
required:
  - missing
  - drink
properties:
  burger:
    type: string
  drink:
    type: string`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	drink := journeyMap["pb33f"].(map[string]interface{})["drink"].(string)
	assert.NotNil(t, drink)

	_, exists := journeyMap["pb33f"].(map[string]interface{})["missing"]
	assert.True(t, exists)

	assert.Nil(t, journeyMap["pb33f"].(map[string]interface{})["burger"])
}


func TestRenderExample_Test_RequiredCheckDisabled(t *testing.T) {
	testObject := `type: [object]
required:
  - drink
  - missing
properties:
  burger:
    type: string
  fries:
    type: string
  drink:
    type: string`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DisableRequiredCheck()
	wr.DiveIntoSchema(compiled, "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	drink := journeyMap["pb33f"].(map[string]interface{})["drink"].(string)
	assert.NotNil(t, drink)

	_, exists := journeyMap["pb33f"].(map[string]interface{})["missing"]
	assert.True(t, exists)

	assert.NotNil(t, journeyMap["pb33f"].(map[string]interface{})["burger"])
	assert.NotNil(t, journeyMap["pb33f"].(map[string]interface{})["fries"])
}

func TestRenderSchema_WithExample(t *testing.T) {
	testObject := `type: [object]
properties:
  name:
    type: string
    example: pb33f`

	compiled := getSchema([]byte(testObject))
	schema := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", schema, visited, 0)
	rendered, _ := json.Marshal(schema["pb33f"])
	assert.Equal(t, `{"name":"pb33f"}`, string(rendered))
}

func TestRenderSchema_WithEnum(t *testing.T) {
	testObject := `type: [object]
properties:
  name:
    type: string
    enum: [pb33f]`

	compiled := getSchema([]byte(testObject))
	schema := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", schema, visited, 0)
	rendered, _ := json.Marshal(schema["pb33f"])
	assert.Equal(t, `{"name":"pb33f"}`, string(rendered))
}

func TestRenderSchema_WithEnum_Float(t *testing.T) {
	testObject := `type: [object]
properties:
  count:
    type: number
    enum: [9934.223]`

	compiled := getSchema([]byte(testObject))
	schema := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", schema, visited, 0)
	rendered, _ := json.Marshal(schema["pb33f"])
	assert.Equal(t, `{"count":9934.223}`, string(rendered))
}

func TestRenderSchema_WithEnum_Integer(t *testing.T) {
	testObject := `type: [object]
properties:
  count:
    type: number
    enum: [9934]`

	compiled := getSchema([]byte(testObject))
	schema := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", schema, visited, 0)
	rendered, _ := json.Marshal(schema["pb33f"])
	assert.Equal(t, `{"count":9934}`, string(rendered))
}

func TestRenderSchema_Items_WithExample(t *testing.T) {
	testObject := `type: object
properties:
  args:
    type: object
    properties:
      arrParam:
        type: string
        example: test,test2
      arrParamExploded:
        type: array
        items:
          type: string
          example: "1"`

	compiled := getSchema([]byte(testObject))
	schema := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", schema, visited, 0)
	rendered, _ := json.Marshal(schema["pb33f"])
	assert.Equal(t, `{"args":{"arrParam":"test,test2","arrParamExploded":["1"]}}`, string(rendered))
}

func TestRenderSchema_Items_WithExamples(t *testing.T) {
	testObject := `type: object
properties:
  args:
    type: object
    properties:
      arrParam:
        type: string
        example: test,test2
      arrParamExploded:
        type: array
        items:
          type: string
          examples:
            - 1
            - 2`

	compiled := getSchema([]byte(testObject))
	schema := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", schema, visited, 0)
	rendered, _ := json.Marshal(schema["pb33f"])
	assert.Equal(t, `{"args":{"arrParam":"test,test2","arrParamExploded":["1","2"]}}`, string(rendered))
}

// https://github.com/pb33f/wiretap/issues/93
func TestRenderSchema_NonStandard_Format(t *testing.T) {
	testObject := `type: object
properties:
  bigint:
    type: integer
    format: bigint
    example: 8821239038968084
  bigintStr:
    type: string
    format: bigint
    example: "9223372036854775808"
  decimal:
    type: number
    format: decimal
    example: 3.141592653589793
  decimalStr:
    type: string
    format: decimal
    example: "3.14159265358979344719667586"`

	compiled := getSchema([]byte(testObject))
	schema := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", schema, visited, 0)
	rendered, _ := json.Marshal(schema["pb33f"])
	assert.Equal(t, `{"bigint":8821239038968084,"bigintStr":"9223372036854775808","decimal":3.141592653589793,"decimalStr":"3.14159265358979344719667586"}`, string(rendered))
}

func TestRenderSchema_NonStandard_Format_MultiExample(t *testing.T) {
	testObject := `type: object
properties:
  bigint:
    type: integer
    format: bigint
    examples: 
      - 8821239038968084
  bigintStr:
    type: string
    format: bigint
    examples: 
      - "9223372036854775808"
  decimal:
    type: number
    format: decimal
    examples: 
      - 3.141592653589793
  decimalStr:
    type: string
    format: decimal
    examples: 
      - "3.14159265358979344719667586"`

	compiled := getSchema([]byte(testObject))
	schema := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", schema, visited, 0)
	rendered, _ := json.Marshal(schema["pb33f"])
	assert.Equal(t, `{"bigint":8821239038968084,"bigintStr":"9223372036854775808","decimal":3.141592653589793,"decimalStr":"3.14159265358979344719667586"}`, string(rendered))
}

func TestRenderSchema_NonStandard_Format_NoExamples(t *testing.T) {
	testObject := `type: object
properties:
  bigint:
    type: integer
    format: bigint
  bigintStr:
    type: string
    format: bigint
  decimal:
    type: number
    format: decimal
  decimalStr:
    type: string
    format: decimal
`

	compiled := getSchema([]byte(testObject))
	schema := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(compiled, "pb33f", schema, visited, 0)
	assert.NotEmpty(t, schema["pb33f"].(map[string]interface{})["bigint"])
	assert.NotEmpty(t, schema["pb33f"].(map[string]interface{})["bigintStr"])
	assert.NotEmpty(t, schema["pb33f"].(map[string]interface{})["decimal"])
	assert.NotEmpty(t, schema["pb33f"].(map[string]interface{})["decimalStr"])
}

func TestRenderSchema_Ref(t *testing.T) {
	yml := `
schemas:
  restaurant:
    type: object
    properties:
      address:
        type: string
        example: Baker Street
      owner:
        $ref: "#/schemas/person"
  person:
    type: object
    properties:
      name:
        type: string
        example: John Doe
`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var components v3.Components
	err := low.BuildModel(idxNode.Content[0], &components)
	assert.NoError(t, err)

	err = components.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

	lowRestaurant := components.FindSchema("restaurant")
	lowNode := low.NodeReference[*lowbase.SchemaProxy]{
		ValueNode: lowRestaurant.ValueNode,
		Reference: lowRestaurant.Reference,
		Value: lowRestaurant.Value,
	}
	schemaProxy := highbase.NewSchemaProxy(&lowNode)
	schema := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(schemaProxy.Schema(), "pb33f", schema, visited, 0)
	rendered, _ := json.Marshal(schema["pb33f"])
	assert.Equal(t, `{"address":"Baker Street","owner":{"name":"John Doe"}}`, string(rendered))
}

func TestRenderSchema_Ref_NoExample(t *testing.T) {
	yml := `
schemas:
  restaurant:
    type: object
    properties:
      address:
        type: string
      owner:
        $ref: "#/schemas/person"
  person:
    type: object
    properties:
      name:
        type: string
`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var components v3.Components
	err := low.BuildModel(idxNode.Content[0], &components)
	assert.NoError(t, err)

	err = components.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

	lowRestaurant := components.FindSchema("restaurant")
	lowNode := low.NodeReference[*lowbase.SchemaProxy]{
		ValueNode: lowRestaurant.ValueNode,
		Reference: lowRestaurant.Reference,
		Value: lowRestaurant.Value,
	}
	schemaProxy := highbase.NewSchemaProxy(&lowNode)
	schema := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(schemaProxy.Schema(), "pb33f", schema, visited, 0)
	assert.NotEmpty(t, schema["pb33f"].(map[string]interface{})["address"])
	assert.NotEmpty(t, schema["pb33f"].(map[string]interface{})["owner"])
	assert.NotEmpty(t, schema["pb33f"].(map[string]interface{})["owner"].(map[string]interface{})["name"])
}


func TestRenderSchema_Ref_CircularArray(t *testing.T) {
	yml := `
schemas:
  human:
    type: object
    properties:
      name:
        type: string
        example: John Doe
      pets:  
        type: array
        items: 
          $ref: "#/schemas/animal"
  animal:
    type: object
    properties:
      name:
        type: string
        example: Bob the cat
      offspring:
        type: array
        items:
          $ref: "#/schemas/animal"
    required:
      - name
      - offspring
`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var components v3.Components
	err := low.BuildModel(idxNode.Content[0], &components)
	assert.NoError(t, err)

	err = components.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

  lowRestaurant := components.FindSchema("human")
	lowNode := low.NodeReference[*lowbase.SchemaProxy]{
		ValueNode: lowRestaurant.ValueNode,
		Reference: lowRestaurant.Reference,
		Value: lowRestaurant.Value,
	}
	schemaProxy := highbase.NewSchemaProxy(&lowNode)
	schema := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(schemaProxy.Schema(), "pb33f", schema, visited, 0)
	rendered, _ := json.Marshal(schema["pb33f"])
	assert.Equal(t, `{"name":"John Doe","pets":[{"name":"Bob the cat","offspring":[]}]}`, string(rendered))
}

func TestRenderSchema_Ref_AllOfCircularArray(t *testing.T) {
	yml := `
schemas:
  human:
    type: object
    properties:
      name:
        type: string
        example: John Doe
      pets:  
        type: array
        items: 
          allOf:
            - $ref: "#/schemas/animal"
  animal:
    type: object
    properties:
      name:
        type: string
        example: Bob the cat
      offspring:
        type: array
        items:
          $ref: "#/schemas/animal"
    required:
      - name
      - offspring
`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var components v3.Components
	err := low.BuildModel(idxNode.Content[0], &components)
	assert.NoError(t, err)

	err = components.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

  lowRestaurant := components.FindSchema("human")
	lowNode := low.NodeReference[*lowbase.SchemaProxy]{
		ValueNode: lowRestaurant.ValueNode,
		Reference: lowRestaurant.Reference,
		Value: lowRestaurant.Value,
	}
	schemaProxy := highbase.NewSchemaProxy(&lowNode)
	schema := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(schemaProxy.Schema(), "pb33f", schema, visited, 0)
	rendered, _ := json.Marshal(schema["pb33f"])
	assert.Equal(t, `{"name":"John Doe","pets":[{"name":"Bob the cat","offspring":[]}]}`, string(rendered))
}

func TestRenderSchema_Ref_AllOfCircularArray2(t *testing.T) {
	yml := `
schemas:
  human:
    type: object
    properties:
      name:
        type: string
        example: John Doe
      friends:  
        type: array
        items: 
          allOf:
            - $ref: "#/schemas/human"
`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var components v3.Components
	err := low.BuildModel(idxNode.Content[0], &components)
	assert.NoError(t, err)

	err = components.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

  lowRestaurant := components.FindSchema("human")
	lowNode := low.NodeReference[*lowbase.SchemaProxy]{
		ValueNode: lowRestaurant.ValueNode,
		Reference: lowRestaurant.Reference,
		Value: lowRestaurant.Value,
	}
	schemaProxy := highbase.NewSchemaProxy(&lowNode)
	schema := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(schemaProxy.Schema(), "pb33f", schema, visited, 0)
	rendered, _ := json.Marshal(schema["pb33f"])
	assert.Equal(t, `{"friends":[{"friends":[],"name":"John Doe"}],"name":"John Doe"}`, string(rendered))
}

func TestRenderSchema_Ref_AnyOfCircularArray(t *testing.T) {
	yml := `
schemas:
  human:
    type: object
    properties:
      name:
        type: string
        example: John Doe
      pets:  
        type: array
        items: 
          anyOf:
            - $ref: "#/schemas/animal"
  animal:
    type: object
    properties:
      name:
        type: string
        example: Bob the cat
      offspring:
        type: array
        items:
          $ref: "#/schemas/animal"
    required:
      - name
      - offspring
`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var components v3.Components
	err := low.BuildModel(idxNode.Content[0], &components)
	assert.NoError(t, err)

	err = components.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

  lowRestaurant := components.FindSchema("human")
	lowNode := low.NodeReference[*lowbase.SchemaProxy]{
		ValueNode: lowRestaurant.ValueNode,
		Reference: lowRestaurant.Reference,
		Value: lowRestaurant.Value,
	}
	schemaProxy := highbase.NewSchemaProxy(&lowNode)
	schema := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(schemaProxy.Schema(), "pb33f", schema, visited, 0)
	rendered, _ := json.Marshal(schema["pb33f"])
	assert.Equal(t, `{"name":"John Doe","pets":[{"name":"Bob the cat","offspring":[]}]}`, string(rendered))
}

func TestRenderSchema_Ref_AnyOfCircularArray2(t *testing.T) {
	yml := `
schemas:
  human:
    type: object
    properties:
      name:
        type: string
        example: John Doe
      friends:  
        type: array
        items: 
          anyOf:
            - $ref: "#/schemas/human"
`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var components v3.Components
	err := low.BuildModel(idxNode.Content[0], &components)
	assert.NoError(t, err)

	err = components.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

  lowRestaurant := components.FindSchema("human")
	lowNode := low.NodeReference[*lowbase.SchemaProxy]{
		ValueNode: lowRestaurant.ValueNode,
		Reference: lowRestaurant.Reference,
		Value: lowRestaurant.Value,
	}
	schemaProxy := highbase.NewSchemaProxy(&lowNode)
	schema := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(schemaProxy.Schema(), "pb33f", schema, visited, 0)
	rendered, _ := json.Marshal(schema["pb33f"])
	assert.Equal(t, `{"friends":[{"friends":[],"name":"John Doe"}],"name":"John Doe"}`, string(rendered))
}


func TestRenderSchema_Ref_AnyOfCircularArraySkip(t *testing.T) {
	yml := `
schemas:
  country:
    type: object
    properties:
      president:
        $ref: "#/schemas/human"
  human:
    type: object
    properties:
      name:
        type: string
        example: John Doe
      pet:
        $ref: "#/schemas/pet"
  pet:
    type: object
    properties:
      name:
        type: string
        example: Hilbert the fish
      bestFriend:
        anyOf:
          - $ref: "#/schemas/pet"
          - $ref: "#/schemas/toy"
  toy:
    type: object
    properties:
      model:
        type: string
        example: ball
      age: 
        type: number
        example: 1
`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var components v3.Components
	err := low.BuildModel(idxNode.Content[0], &components)
	assert.NoError(t, err)

	err = components.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

  lowRestaurant := components.FindSchema("human")
	lowNode := low.NodeReference[*lowbase.SchemaProxy]{
		ValueNode: lowRestaurant.ValueNode,
		Reference: lowRestaurant.Reference,
		Value: lowRestaurant.Value,
	}
	schemaProxy := highbase.NewSchemaProxy(&lowNode)
	schema := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(schemaProxy.Schema(), "pb33f", schema, visited, 0)
	rendered, _ := json.Marshal(schema["pb33f"])
	assert.Equal(t, `{"name":"John Doe","pet":{"bestFriend":{"age":1,"model":"ball"},"name":"Hilbert the fish"}}`, string(rendered))
}

func TestRenderSchema_Ref_OneOfCircularArray(t *testing.T) {
	yml := `
schemas:
  human:
    type: object
    properties:
      name:
        type: string
        example: John Doe
      pets:  
        type: array
        items: 
          oneOf:
            - $ref: "#/schemas/animal"
  animal:
    type: object
    properties:
      name:
        type: string
        example: Bob the cat
      offspring:
        type: array
        items:
          $ref: "#/schemas/animal"
    required:
      - name
      - offspring
`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var components v3.Components
	err := low.BuildModel(idxNode.Content[0], &components)
	assert.NoError(t, err)

	err = components.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

  lowRestaurant := components.FindSchema("human")
	lowNode := low.NodeReference[*lowbase.SchemaProxy]{
		ValueNode: lowRestaurant.ValueNode,
		Reference: lowRestaurant.Reference,
		Value: lowRestaurant.Value,
	}
	schemaProxy := highbase.NewSchemaProxy(&lowNode)
	schema := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(schemaProxy.Schema(), "pb33f", schema, visited, 0)
	rendered, _ := json.Marshal(schema["pb33f"])
	assert.Equal(t, `{"name":"John Doe","pets":[{"name":"Bob the cat","offspring":[]}]}`, string(rendered))
}

func TestRenderSchema_Ref_OneOfCircularArraySkip(t *testing.T) {
	yml := `
schemas:
  human:
    type: object
    properties:
      name:
        type: string
        example: John Doe
      friend:
        oneOf:
          - $ref: "#/schemas/human"
`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var components v3.Components
	err := low.BuildModel(idxNode.Content[0], &components)
	assert.NoError(t, err)

	err = components.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

  lowRestaurant := components.FindSchema("human")
	lowNode := low.NodeReference[*lowbase.SchemaProxy]{
		ValueNode: lowRestaurant.ValueNode,
		Reference: lowRestaurant.Reference,
		Value: lowRestaurant.Value,
	}
	schemaProxy := highbase.NewSchemaProxy(&lowNode)
	schema := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(schemaProxy.Schema(), "pb33f", schema, visited, 0)
  rendered, _ := json.Marshal(schema["pb33f"])
  assert.Equal(t, `{"friend":{"name":"John Doe"},"name":"John Doe"}`, string(rendered))
}

func TestRenderSchema_Ref_OneOfCircularArrayFail(t *testing.T) {
	yml := `
schemas:
  human:
    type: object
    properties:
      name:
        type: string
        example: John Doe
      friend:
        oneOf:
          - $ref: "#/schemas/human"
    required:
      - name
      - friend
`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var components v3.Components
	err := low.BuildModel(idxNode.Content[0], &components)
	assert.NoError(t, err)

	err = components.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

  lowRestaurant := components.FindSchema("human")
	lowNode := low.NodeReference[*lowbase.SchemaProxy]{
		ValueNode: lowRestaurant.ValueNode,
		Reference: lowRestaurant.Reference,
		Value: lowRestaurant.Value,
	}
	schemaProxy := highbase.NewSchemaProxy(&lowNode)
	schema := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	success := wr.DiveIntoSchema(schemaProxy.Schema(), "pb33f", schema, visited, 0)
	assert.False(t, success)
}


func TestRenderSchema_Ref_SkipOptional(t *testing.T) {
	yml := `
schemas:
  pet:
    type: object
    properties:
      name:
        type: string
        example: Maria the frog
      color:
        type: string
        example: green
      bestFriend:
        $ref: "#/schemas/pet"
    required:
      - name
`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var components v3.Components
	err := low.BuildModel(idxNode.Content[0], &components)
	assert.NoError(t, err)

	err = components.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

  lowRestaurant := components.FindSchema("pet")
	lowNode := low.NodeReference[*lowbase.SchemaProxy]{
		ValueNode: lowRestaurant.ValueNode,
		Reference: lowRestaurant.Reference,
		Value: lowRestaurant.Value,
	}
	schemaProxy := highbase.NewSchemaProxy(&lowNode)
	schema := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(schemaProxy.Schema(), "pb33f", schema, visited, 0)
	rendered, _ := json.Marshal(schema["pb33f"])
	assert.Equal(t, `{"name":"Maria the frog"}`, string(rendered))
}

func TestRenderSchema_Ref_SkipCircularProp(t *testing.T) {
	yml := `
schemas:
  pet:
    type: object
    properties:
      name:
        type: string
        example: Maria the frog
      color:
        type: string
        example: green
      bestFriend:
        $ref: "#/schemas/pet"
`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var components v3.Components
	err := low.BuildModel(idxNode.Content[0], &components)
	assert.NoError(t, err)

	err = components.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

  lowRestaurant := components.FindSchema("pet")
	lowNode := low.NodeReference[*lowbase.SchemaProxy]{
		ValueNode: lowRestaurant.ValueNode,
		Reference: lowRestaurant.Reference,
		Value: lowRestaurant.Value,
	}
	schemaProxy := highbase.NewSchemaProxy(&lowNode)
	schema := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
  wr.DiveIntoSchema(schemaProxy.Schema(), "pb33f", schema, visited, 0)
	rendered, _ := json.Marshal(schema["pb33f"])
	assert.Equal(t, `{"bestFriend":{"color":"green","name":"Maria the frog"},"color":"green","name":"Maria the frog"}`, string(rendered))
}

func TestRenderSchema_Ref_FailRenderOfCircularRef(t *testing.T) {
	yml := `
schemas:
  pet:
    type: object
    properties:
      name:
        type: string
        example: Maria the frog
      color:
        type: string
        example: green
      bestFriend:
        $ref: "#/schemas/pet"
    required:
      - name
      - bestFriend
`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var components v3.Components
	err := low.BuildModel(idxNode.Content[0], &components)
	assert.NoError(t, err)

	err = components.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

  lowRestaurant := components.FindSchema("pet")
	lowNode := low.NodeReference[*lowbase.SchemaProxy]{
		ValueNode: lowRestaurant.ValueNode,
		Reference: lowRestaurant.Reference,
		Value: lowRestaurant.Value,
	}
	schemaProxy := highbase.NewSchemaProxy(&lowNode)
	schema := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
  success := wr.DiveIntoSchema(schemaProxy.Schema(), "pb33f", schema, visited, 0)
  assert.False(t, success)

}

func TestRenderExample_Ref_DependentSchemasFail(t *testing.T) {
	yml := `
schemas:
  human:
    type: object
    properties:
      name:
        type: string
        example: "Lena"
      pet:
        type: object
        properties: 
          name:
            type: string
            example: Luis the kangaroo
    dependentSchemas:
      pet:
        properties:
          friend:
            $ref: "#/schemas/toy"
        required:
          - friend
  toy:
    type: object
    properties:
      enemy:
        $ref: "#/schemas/toy"
    required:
      - enemy
`
	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var components v3.Components
	err := low.BuildModel(idxNode.Content[0], &components)
	assert.NoError(t, err)

	err = components.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

  lowRestaurant := components.FindSchema("human")
	lowNode := low.NodeReference[*lowbase.SchemaProxy]{
		ValueNode: lowRestaurant.ValueNode,
		Reference: lowRestaurant.Reference,
		Value: lowRestaurant.Value,
	}
	schemaProxy := highbase.NewSchemaProxy(&lowNode)
	schema := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
  success := wr.DiveIntoSchema(schemaProxy.Schema(), "pb33f", schema, visited, 0)
  assert.False(t, success)
}

func TestCreateRendererUsingDefaultDictionary(t *testing.T) {
	assert.NotNil(t, CreateRendererUsingDefaultDictionary())
}

func TestReadDictionary(t *testing.T) {
	if _, err := os.Stat("/usr/share/dict/wddords"); !os.IsNotExist(err) {
		words := ReadDictionary("/usr/share/dict/words")
		assert.Greater(t, len(words), 500)
	}
}

func TestCreateFakeDictionary(t *testing.T) {
	// create a temp file and create a simple temp dictionary
	tmpFile, _ := os.CreateTemp("", "pb33f")
	tmpFile.Write([]byte("one\nfive\nthree"))
	words := ReadDictionary(tmpFile.Name())
	renderer := CreateRendererUsingDictionary(tmpFile.Name())
	assert.Len(t, words, 3)
	assert.Equal(t, "five", renderer.RandomWord(4, 4, 0))
	assert.Equal(t, "one", renderer.RandomWord(2, 3, 0))
	assert.Equal(t, "three", renderer.RandomWord(5, 5, 0))
	assert.NotEmpty(t, "three", renderer.RandomWord(0, 0, 0))
}

func TestReadDictionary_BadReadFile(t *testing.T) {
	words := ReadDictionary("/do/not/exist")
	assert.LessOrEqual(t, len(words), 0)
}

type errReader int

func (errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("pb33f error")
}

func TestReadDictionary_BadReader(t *testing.T) {
	words := readFile(errReader(0))
	assert.LessOrEqual(t, len(words), 0)
}

func TestWordRenderer_RandomWord_TooBig(t *testing.T) {
	wr := CreateRendererUsingDefaultDictionary()
	word := wr.RandomWord(int64(1000), int64(1001), 0)
	if _, err := os.Stat("/usr/share/dict/words"); !os.IsNotExist(err) {
		assert.Equal(t, "no-word-found-1000-1001", word)
	} else {
		word = wr.RandomWord(int64(1000), int64(1001), 101)
		assert.Equal(t, "no-word-found-1000-1001", word)
	}
}

func TestWordRenderer_RandomFloat64(t *testing.T) {
	wr := CreateRendererUsingDefaultDictionary()
	word := wr.RandomFloat64()
	assert.GreaterOrEqual(t, word, float64(0))
	assert.LessOrEqual(t, word, float64(100))
}

func TestWordRenderer_RandomWordMinMaxZero(t *testing.T) {
	wr := CreateRendererUsingDefaultDictionary()
	word := wr.RandomWord(int64(0), int64(0), 0)
	assert.NotEmpty(t, word)
}

func TestRenderSchema_NestedDeep(t *testing.T) {
	deepNest := createNestedStructure()
	journeyMap := make(map[string]any)
	visited := createVisitedMap()
	wr := createSchemaRenderer()
	wr.DiveIntoSchema(deepNest.Schema(), "pb33f", journeyMap, visited, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	journeyLevel := 0
	var dive func(mapNode map[string]any, level int)
	// count the levels to validate the recursion hard limit.
	dive = func(mapNode map[string]any, level int) {
		child := mapNode["child"]
		if level >= 100 {
			assert.Equal(t, "to deep to continue rendering...", child)
			journeyLevel = level
			return
		}
		if child != nil {
			dive(child.(map[string]any), level+1)
		}
	}
	dive(journeyMap["pb33f"].(map[string]any), journeyLevel)
	assert.Equal(t, 100, journeyLevel)
}

func TestCreateRendererUsingDictionary(t *testing.T) {
	assert.NotEmpty(t, CreateRendererUsingDictionary("nowhere").RandomWord(0, 0, 0))
	assert.NotEmpty(t, CreateRendererUsingDictionary("/usr/share/dict/words").RandomWord(0, 0, 0))
}

func createNestedStructure() *highbase.SchemaProxy {
	schema := `type: [object]
properties:
  name:
    type: string
    example: pb33f`

	var compNode yaml.Node
	e := yaml.Unmarshal([]byte(schema), &compNode)
	if e != nil {
		panic(e)
	}

	buildSchema := func() *highbase.SchemaProxy {
		sp := new(lowbase.SchemaProxy)
		_ = sp.Build(context.Background(), nil, compNode.Content[0], nil)
		lp := low.NodeReference[*lowbase.SchemaProxy]{
			Value:     sp,
			ValueNode: compNode.Content[0],
		}
		return highbase.NewSchemaProxy(&lp)
	}

	var loopMe func(parent *highbase.SchemaProxy, level int)

	loopMe = func(parent *highbase.SchemaProxy, level int) {
		schemaProxy := buildSchema()
		if parent != nil {
			parent.Schema().Properties.Set("child", schemaProxy)
		}
		if level < 110 {
			loopMe(schemaProxy, level+1)
		}
	}
	root := buildSchema()
	loopMe(root, 0)
	return root
}
