// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package renderer

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"
)

func getSchema(schema []byte) *highbase.Schema {
	var compNode yaml.Node
	e := yaml.Unmarshal(schema, &compNode)
	if e != nil {
		panic(e)
	}
	sp := new(lowbase.SchemaProxy)
	_ = sp.Build(nil, compNode.Content[0], nil)
	lp := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     sp,
		ValueNode: compNode.Content[0],
	}
	schemaProxy := highbase.NewSchemaProxy(&lp)
	return schemaProxy.Schema()
}

func TestRenderExample_StringWithExample(t *testing.T) {

	testObject := `type: string
example: dog`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	assert.Equal(t, journeyMap["pb33f"], "dog")

}

func TestRenderExample_StringWithNoExample(t *testing.T) {

	testObject := `type: string`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.GreaterOrEqual(t, len(journeyMap["pb33f"].(string)), 3)
	assert.LessOrEqual(t, len(journeyMap["pb33f"].(string)), 10)

}

func TestRenderExample_StringWithNoExample_Format_Datetime(t *testing.T) {

	testObject := `type: string
format: date-time`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	now := time.Now()
	assert.NotNil(t, journeyMap["pb33f"])
	assert.Equal(t, journeyMap["pb33f"].(string)[:4], fmt.Sprintf("%d", now.Year()))
}

func TestRenderExample_StringWithNoExample_Pattern_Email(t *testing.T) {

	testObject := `type: string
pattern: "^[a-z]{5,10}@[a-z]{5,10}\\.(com|net|org)$"` // an email address

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

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
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

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
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	now := time.Now().Format("2006-01-02")

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Equal(t, journeyMap["pb33f"].(string), now)
}

func TestRenderExample_StringWithNoExample_Format_Time(t *testing.T) {
	testObject := `type: string
format: time`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	now := time.Now().Format("15:04:05")

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Equal(t, now, journeyMap["pb33f"].(string))
}

func TestRenderExample_StringWithNoExample_Email(t *testing.T) {
	testObject := `type: string
format: email`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.True(t, strings.Contains(journeyMap["pb33f"].(string), "@"))
}

func TestRenderExample_StringWithNoExample_Hostname(t *testing.T) {
	testObject := `type: string
format: hostname`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.True(t, strings.Contains(journeyMap["pb33f"].(string), ".com"))
}

func TestRenderExample_StringWithNoExample_ivp4(t *testing.T) {
	testObject := `type: string
format: ipv4`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	segs := strings.Split(journeyMap["pb33f"].(string), ".")
	assert.Len(t, segs, 4)
}

func TestRenderExample_StringWithNoExample_ivp6(t *testing.T) {
	testObject := `type: string
format: ipv6`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	segs := strings.Split(journeyMap["pb33f"].(string), ":")
	assert.Len(t, segs, 8)
}

func TestRenderExample_StringWithNoExample_URI(t *testing.T) {
	testObject := `type: string
format: uri`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

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
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	segs := strings.Split(journeyMap["pb33f"].(string), "-")
	assert.Len(t, segs, 5)
}

func TestRenderExample_StringWithNoExample_Byte(t *testing.T) {
	testObject := `type: string
format: byte`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.NotEmpty(t, journeyMap["pb33f"].(string))
}

func TestRenderExample_StringWithNoExample_Password(t *testing.T) {
	testObject := `type: string
format: password`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.NotEmpty(t, journeyMap["pb33f"].(string))
}

func TestRenderExample_StringWithNoExample_URIReference(t *testing.T) {
	testObject := `type: string
format: uri-reference`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	segs := strings.Split(journeyMap["pb33f"].(string), "/")
	assert.Len(t, segs, 3)
}

func TestRenderExample_StringWithNoExample_Binary(t *testing.T) {
	testObject := `type: string
format: binary`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

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
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Equal(t, 3.14, journeyMap["pb33f"])
}

func TestRenderExample_MinLength(t *testing.T) {
	testObject := `type: string
minLength: 10`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.GreaterOrEqual(t, 10, len(journeyMap["pb33f"].(string)))
}

func TestRenderExample_MaxLength(t *testing.T) {
	testObject := `type: string
maxLength: 10`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.LessOrEqual(t, 10, len(journeyMap["pb33f"].(string)))
}

func TestRenderExample_MaxLength_MinLength(t *testing.T) {
	testObject := `type: string
maxLength: 8
minLength: 3`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.LessOrEqual(t, len(journeyMap["pb33f"].(string)), 8)
	assert.GreaterOrEqual(t, len(journeyMap["pb33f"].(string)), 3)

}

func TestRenderExample_NumberNoExample_Default(t *testing.T) {
	testObject := `type: number`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Greater(t, journeyMap["pb33f"], int64(0))
	assert.Less(t, journeyMap["pb33f"], int64(100))
}

func TestRenderExample_Number_Minimum(t *testing.T) {
	testObject := `type: number
minimum: 60`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.GreaterOrEqual(t, journeyMap["pb33f"], int64(60))
	assert.Less(t, journeyMap["pb33f"], int64(100))
}

func TestRenderExample_Number_Maximum(t *testing.T) {
	testObject := `type: number
maximum: 4`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.GreaterOrEqual(t, journeyMap["pb33f"], int64(1))
	assert.LessOrEqual(t, journeyMap["pb33f"], int64(4))
}

func TestRenderExample_NumberNoExample_Float(t *testing.T) {
	testObject := `type: number
format: float`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Greater(t, journeyMap["pb33f"], float32(0))
}

func TestRenderExample_NumberNoExample_Float64(t *testing.T) {
	testObject := `type: number
format: double`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Greater(t, journeyMap["pb33f"], float64(0))
}

func TestRenderExample_NumberNoExample_Int32(t *testing.T) {
	testObject := `type: number
format: int32`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Greater(t, journeyMap["pb33f"], 0)
}

func TestRenderExample_NumberNoExample_Int64(t *testing.T) {
	testObject := `type: number
format: int64`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Greater(t, journeyMap["pb33f"], int64(0))
}

func TestRenderExample_Boolean_WithExample(t *testing.T) {
	testObject := `type: boolean
example: true`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.True(t, journeyMap["pb33f"].(bool))
}

func TestRenderExample_Boolean_WithoutExample(t *testing.T) {
	testObject := `type: boolean`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.True(t, journeyMap["pb33f"].(bool))
}

func TestRenderExample_Array_String(t *testing.T) {
	testObject := `type: array
items:
  type: string`

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

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
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Len(t, journeyMap["pb33f"], 1)
	assert.Greater(t, journeyMap["pb33f"].([]interface{})[0].(int64), int64(0))
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
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

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
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

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
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

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
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Len(t, journeyMap["pb33f"], 1)
	assert.Greater(t, journeyMap["pb33f"].(map[string]interface{})["fishCake"].(map[string]interface{})["cream"].(float64), float64(0))
	assert.True(t, journeyMap["pb33f"].(map[string]interface{})["fishCake"].(map[string]interface{})["bones"].(bool))
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
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Len(t, journeyMap["pb33f"], 1)
	assert.Greater(t, len(journeyMap["pb33f"].(map[string]interface{})["bones"].(string)), 0)
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
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	assert.Len(t, journeyMap["pb33f"], 1)
	assert.Greater(t, len(journeyMap["pb33f"].(map[string]interface{})["bones"].(string)), 0)
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
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

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
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

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
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

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
            type: boolean
                            `

	compiled := getSchema([]byte(testObject))

	journeyMap := make(map[string]any)
	DiveIntoSchema(compiled, "pb33f", journeyMap, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	burger := journeyMap["pb33f"].(map[string]interface{})["burger"].(map[string]interface{})
	assert.NotNil(t, burger)
	assert.NotEmpty(t, burger["name"].(string))
	assert.NotZero(t, burger["weight"].(int64))
	assert.NotEmpty(t, burger["patty"].(string))
	assert.True(t, burger["frozen"].(bool))
}

func TestRenderSchema_WithExample(t *testing.T) {
	testObject := `type: [object]
properties:
  name:
    type: string
    example: pb33f`

	compiled := getSchema([]byte(testObject))

	schema := RenderSchema(compiled)
	rendered, _ := json.Marshal(schema)
	assert.Equal(t, `{"name":"pb33f"}`, string(rendered))
}

func TestRenderSchema_WithEnum(t *testing.T) {
	testObject := `type: [object]
properties:
  name:
    type: string
    enum: [pb33f]`

	compiled := getSchema([]byte(testObject))

	schema := RenderSchema(compiled)
	rendered, _ := json.Marshal(schema)
	assert.Equal(t, `{"name":"pb33f"}`, string(rendered))
}

func TestCreateRendererUsingDefaultDictionary(t *testing.T) {
	assert.NotNil(t, CreateRendererUsingDefaultDictionary())
}

func TestReadDictionary(t *testing.T) {
	words := ReadDictionary("/usr/share/dict/words")
	assert.Greater(t, len(words), 500)
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
	assert.Equal(t, "no-word", word)
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
	DiveIntoSchema(deepNest.Schema(), "pb33f", journeyMap, 0)

	assert.NotNil(t, journeyMap["pb33f"])
	var journeyLevel = 0
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
		_ = sp.Build(nil, compNode.Content[0], nil)
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
			parent.Schema().Properties["child"] = schemaProxy
		}
		if level < 110 {
			loopMe(schemaProxy, level+1)
		}
	}
	root := buildSchema()
	loopMe(root, 0)
	return root
}
