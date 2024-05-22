// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package renderer

import (
	"context"
	"encoding/json"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"strings"
	"testing"
)

type fakeMockable struct {
	Schema   *base.SchemaProxy
	Example  any
	Examples *orderedmap.Map[string, *base.Example]
}

type fakeMockableButWithASchemaNotAProxy struct {
	Schema   *base.Schema
	Example  any
	Examples *orderedmap.Map[string, *base.Example]
}

var simpleFakeMockSchema = `type: string
enum: [magic-herbs]`

var objectFakeMockSchema = `type: object
properties:
  coffee:
    type: string
    minLength: 6
  herbs:
    type: number
    minimum: 350
    maximum: 400`

func createFakeMock(mock string, values map[string]any, example any) *fakeMockable {
	var root yaml.Node
	_ = yaml.Unmarshal([]byte(mock), &root)
	var lowProxy lowbase.SchemaProxy
	_ = lowProxy.Build(context.Background(), &root, root.Content[0], nil)
	lowRef := low.NodeReference[*lowbase.SchemaProxy]{
		Value: &lowProxy,
	}
	highSchema := base.NewSchemaProxy(&lowRef)
	examples := orderedmap.New[string, *base.Example]()

	for k, v := range values {
		examples.Set(k, &base.Example{
			Value: utils.CreateYamlNode(v),
		})
	}
	return &fakeMockable{
		Schema:   highSchema,
		Example:  example,
		Examples: examples,
	}
}

func createFakeMockWithoutProxy(mock string, values map[string]any, example any) *fakeMockableButWithASchemaNotAProxy {
	var root yaml.Node
	_ = yaml.Unmarshal([]byte(mock), &root)
	var lowProxy lowbase.SchemaProxy
	_ = lowProxy.Build(context.Background(), &root, root.Content[0], nil)
	lowRef := low.NodeReference[*lowbase.SchemaProxy]{
		Value: &lowProxy,
	}
	highSchema := base.NewSchemaProxy(&lowRef)
	examples := orderedmap.New[string, *base.Example]()

	for k, v := range values {
		examples.Set(k, &base.Example{
			Value: utils.CreateYamlNode(v),
		})
	}
	return &fakeMockableButWithASchemaNotAProxy{
		Schema:   highSchema.Schema(),
		Example:  example,
		Examples: examples,
	}
}

func TestNewMockGenerator(t *testing.T) {
	mg := NewMockGenerator(JSON)
	assert.NotNil(t, mg)
}

func TestNewMockGeneratorWithDictionary(t *testing.T) {
	mg := NewMockGeneratorWithDictionary("", JSON)
	assert.NotNil(t, mg)
}

func TestMockGenerator_GenerateJSONMock_NoObject(t *testing.T) {
	mg := NewMockGenerator(JSON)

	var isNil any

	mock, err := mg.GenerateMock(isNil, "")
	assert.NoError(t, err)
	assert.Nil(t, mock)
}

func TestMockGenerator_GenerateJSONMock_BadObject(t *testing.T) {
	type NotMockable struct {
	}

	mg := NewMockGenerator(JSON)
	mock, err := mg.GenerateMock(&NotMockable{}, "")
	assert.Error(t, err)
	assert.Nil(t, mock)
}

func TestMockGenerator_GenerateJSONMock_EmptyObject(t *testing.T) {
	mg := NewMockGenerator(JSON)
	mock, err := mg.GenerateMock(&fakeMockable{}, "")
	assert.NoError(t, err)
	assert.Nil(t, mock)
}

func TestMockGenerator_GenerateJSONMock_SuppliedExample_JSON(t *testing.T) {
	fakeExample := map[string]any{
		"fish-and-chips": "cod-and-chips-twice",
	}
	fake := createFakeMock(simpleFakeMockSchema, nil, fakeExample)
	mg := NewMockGenerator(JSON)
	mock, err := mg.GenerateMock(fake, "")
	assert.NoError(t, err)
	assert.Equal(t, "{\"fish-and-chips\":\"cod-and-chips-twice\"}", string(mock))
}

func TestMockGenerator_GenerateJSONMock_SuppliedExample_YAML(t *testing.T) {
	fakeExample := map[string]any{
		"fish-and-chips": "cod-and-chips-twice",
	}
	fake := createFakeMock(simpleFakeMockSchema, nil, fakeExample)
	mg := NewMockGenerator(YAML)
	mock, err := mg.GenerateMock(fake, "")
	assert.NoError(t, err)
	assert.Equal(t, "fish-and-chips: cod-and-chips-twice", strings.TrimSpace(string(mock)))
}

func TestMockGenerator_GenerateJSONMock_MultiExamples_NoName_JSON(t *testing.T) {
	fakeExample := map[string]any{
		"exampleOne": map[string]any{
			"fish-and-chips": "cod-and-chips-twice",
		},
		"exampleTwo": map[string]any{
			"rice-and-peas": "brown-or-white-rice",
		},
	}
	fake := createFakeMock(simpleFakeMockSchema, fakeExample, nil)
	mg := NewMockGenerator(JSON)
	mock, err := mg.GenerateMock(fake, "JimmyJammyJimJams") // does not exist
	assert.NoError(t, err)
	assert.NotEmpty(t, string(mock))
}

func TestMockGenerator_GenerateJSONMock_MultiExamples_JSON(t *testing.T) {
	fakeExample := map[string]any{
		"exampleOne": map[string]any{
			"fish-and-chips": "cod-and-chips-twice",
		},
		"exampleTwo": map[string]any{
			"rice-and-peas": "brown-or-white-rice",
		},
	}
	fake := createFakeMock(simpleFakeMockSchema, fakeExample, nil)
	mg := NewMockGenerator(JSON)
	mock, err := mg.GenerateMock(fake, "exampleTwo")
	assert.NoError(t, err)
	assert.Equal(t, "{\"rice-and-peas\":\"brown-or-white-rice\"}", string(mock))
}

func TestMockGenerator_GenerateJSONMock_MultiExamples_PrettyJSON(t *testing.T) {
	fakeExample := map[string]any{
		"exampleOne": map[string]any{
			"fish-and-chips": "cod-and-chips-twice",
		},
		"exampleTwo": map[string]any{
			"rice-and-peas": "brown-or-white-rice",
			"peas":          "buttery",
		},
	}
	fake := createFakeMock(simpleFakeMockSchema, fakeExample, nil)
	mg := NewMockGenerator(JSON)
	mg.SetPretty()
	mock, err := mg.GenerateMock(fake, "exampleTwo")
	assert.NoError(t, err)
	assert.Equal(t, "{\n  \"peas\": \"buttery\",\n  \"rice-and-peas\": \"brown-or-white-rice\"\n}", string(mock))
}

func TestMockGenerator_GenerateJSONMock_MultiExamples_YAML(t *testing.T) {
	fakeExample := map[string]any{
		"exampleOne": map[string]any{
			"fish-and-chips": "cod-and-chips-twice",
		},
		"exampleTwo": map[string]any{
			"rice-and-peas": "brown-or-white-rice",
		},
	}
	fake := createFakeMock(simpleFakeMockSchema, fakeExample, nil)
	mg := NewMockGenerator(YAML)
	mock, err := mg.GenerateMock(fake, "exampleTwo")
	assert.NoError(t, err)
	assert.Equal(t, "rice-and-peas: brown-or-white-rice", strings.TrimSpace(string(mock)))
}

func TestMockGenerator_GenerateJSONMock_NoExamples_JSON(t *testing.T) {
	fake := createFakeMock(simpleFakeMockSchema, nil, nil)
	mg := NewMockGenerator(JSON)
	mock, err := mg.GenerateMock(fake, "")
	assert.NoError(t, err)
	assert.Equal(t, "magic-herbs", string(mock))
}

func TestMockGenerator_GenerateJSONMock_NoExamples_YAML(t *testing.T) {
	fake := createFakeMock(simpleFakeMockSchema, nil, nil)
	mg := NewMockGenerator(YAML)
	mock, err := mg.GenerateMock(fake, "")
	assert.NoError(t, err)
	assert.Equal(t, "magic-herbs", strings.TrimSpace(string(mock)))
}

func TestMockGenerator_GenerateJSONMock_Object_NoExamples_JSON(t *testing.T) {
	fake := createFakeMock(objectFakeMockSchema, nil, nil)
	mg := NewMockGenerator(JSON)
	mock, err := mg.GenerateMock(fake, "")
	assert.NoError(t, err)

	// re-serialize back into a map and check the values
	var m map[string]any
	err = json.Unmarshal(mock, &m)
	assert.NoError(t, err)

	assert.Len(t, m, 2)
	assert.GreaterOrEqual(t, len(m["coffee"].(string)), 6)
	assert.GreaterOrEqual(t, m["herbs"].(float64), float64(350))
	assert.LessOrEqual(t, m["herbs"].(float64), float64(400))
}

func TestMockGenerator_GenerateJSONMock_Object_NoExamples_YAML(t *testing.T) {
	fake := createFakeMock(objectFakeMockSchema, nil, nil)
	mg := NewMockGenerator(YAML)
	mock, err := mg.GenerateMock(fake, "")
	assert.NoError(t, err)

	// re-serialize back into a map and check the values
	var m map[string]any
	err = yaml.Unmarshal(mock, &m)
	assert.NoError(t, err)

	assert.Len(t, m, 2)
	assert.GreaterOrEqual(t, len(m["coffee"].(string)), 6)
	assert.GreaterOrEqual(t, m["herbs"].(int), 350)
	assert.LessOrEqual(t, m["herbs"].(int), 400)
}

// should result in the exact same output as the above test
func TestMockGenerator_GenerateJSONMock_Object_RawSchema(t *testing.T) {
	fake := createFakeMockWithoutProxy(objectFakeMockSchema, nil, nil)

	mg := NewMockGenerator(YAML)
	mock, err := mg.GenerateMock(fake, "")
	assert.NoError(t, err)

	// re-serialize back into a map and check the values
	var m map[string]any
	err = yaml.Unmarshal(mock, &m)
	assert.NoError(t, err)

	assert.Len(t, m, 2)
	assert.GreaterOrEqual(t, len(m["coffee"].(string)), 6)
	assert.GreaterOrEqual(t, m["herbs"].(int), 350)
	assert.LessOrEqual(t, m["herbs"].(int), 400)
}

func TestMockGenerator_GenerateMock_YamlNode(t *testing.T) {
	mg := NewMockGenerator(YAML)

	type mockable struct {
		Example  *yaml.Node
		Examples *orderedmap.Map[string, *base.Example]
	}

	mock, err := mg.GenerateMock(&mockable{
		Example: utils.CreateStringNode("hello"),
	}, "")
	assert.NoError(t, err)
	assert.Equal(t, "hello", strings.TrimSpace(string(mock)))
}

func TestMockGenerator_GenerateMock_YamlNode_Nil(t *testing.T) {
	mg := NewMockGenerator(YAML)

	var example *yaml.Node

	type mockable struct {
		Example  any
		Examples *orderedmap.Map[string, *base.Example]
	}

	examples := orderedmap.New[string, *base.Example]()
	examples.Set("exampleOne", &base.Example{
		Value: utils.CreateStringNode("hello"),
	})

	mock, err := mg.GenerateMock(&mockable{
		Example:  example,
		Examples: examples,
	}, "")
	assert.NoError(t, err)
	assert.Equal(t, "hello", strings.TrimSpace(string(mock)))
}

func TestMockGenerator_GenerateJSONMock_Object_SchemaExamples(t *testing.T) {

	yml := `type: object
examples:
  - name: happy days
    description: a terrible show from a time that never existed.
  - name: robocop
    description: perhaps the best cyberpunk movie ever made.
properties:
  name:
    type: string
    example: nameExample
  description:
    type: string
    example: descriptionExample`

	fake := createFakeMock(yml, nil, nil)
	mg := NewMockGenerator(YAML)
	mock, err := mg.GenerateMock(fake, "")
	assert.NoError(t, err)

	// re-serialize back into a map and check the values
	var m map[string]any
	err = yaml.Unmarshal(mock, &m)
	assert.NoError(t, err)

	assert.Len(t, m, 2)
	assert.Equal(t, "happy days", m["name"].(string))
	assert.Equal(t, "a terrible show from a time that never existed.", m["description"].(string))
}

func TestMockGenerator_GenerateJSONMock_Object_SchemaExamples_Preferred(t *testing.T) {

	yml := `type: object
examples:
  - name: happy days
    description: a terrible show from a time that never existed.
  - name: robocop
    description: perhaps the best cyberpunk movie ever made.
properties:
  name:
    type: string
    example: nameExample
  description:
    type: string
    example: descriptionExample`

	fake := createFakeMock(yml, nil, nil)
	mg := NewMockGenerator(YAML)
	mock, err := mg.GenerateMock(fake, "1")
	assert.NoError(t, err)

	// re-serialize back into a map and check the values
	var m map[string]any
	err = yaml.Unmarshal(mock, &m)
	assert.NoError(t, err)

	assert.Len(t, m, 2)
	assert.Equal(t, "robocop", m["name"].(string))
	assert.Equal(t, "perhaps the best cyberpunk movie ever made.", m["description"].(string))
}

func TestMockGenerator_GenerateJSONMock_Object_SchemaExample(t *testing.T) {

	yml := `type: object
example:
  name: robocop
  description: perhaps the best cyberpunk movie ever made.
properties:
  name:
    type: string
    example: nameExample
  description:
    type: string
    example: descriptionExample`

	fake := createFakeMock(yml, nil, nil)
	mg := NewMockGenerator(YAML)
	mock, err := mg.GenerateMock(fake, "")
	assert.NoError(t, err)

	// re-serialize back into a map and check the values
	var m map[string]any
	err = yaml.Unmarshal(mock, &m)
	assert.NoError(t, err)

	assert.Len(t, m, 2)
	assert.Equal(t, "robocop", m["name"].(string))
	assert.Equal(t, "perhaps the best cyberpunk movie ever made.", m["description"].(string))
}

func TestMockGenerator_GeneratePropertyExamples(t *testing.T) {
	fake := createFakeMock(`type: object
required:
  - id
  - name
properties:
  id:
    type: integer
    example: 123
  name:
    type: string
    example: "John Doe"
  active:
    type: boolean
    example: true
  balance:
    type: number
    format: float
    example: 99.99
  tags:
    type: array
    items:
      type: string
    example: ["tag1", "tag2", "tag3"]
`, nil, nil)

	for name, tc := range map[string]struct {
		mockGen      func() *MockGenerator
		expectedMock string
	}{
		"OnlyRequired": {
			mockGen: func() *MockGenerator {
				mg := NewMockGenerator(JSON)
				return mg
			},
			expectedMock: `{"id":123,"name":"John Doe"}`,
		},
		"All": {
			mockGen: func() *MockGenerator {
				mg := NewMockGenerator(JSON)

				// Test schema rendering for property examples, regardless of
				// whether the property is marked as required or not.
				mg.DisableRequiredCheck()

				return mg
			},
			expectedMock: `{"active":true,"balance":99.99,"id":123,"name":"John Doe","tags":["tag1","tag2","tag3"]}`,
		},
	} {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mock, err := tc.mockGen().GenerateMock(fake, "")
			require.NoError(t, err)

			assert.Equal(t, tc.expectedMock, string(mock))
		})
	}
}
