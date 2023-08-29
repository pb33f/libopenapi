// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package renderer

import (
	"encoding/json"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"strings"
	"testing"
)

type fakeMockable struct {
	Schema   *highbase.SchemaProxy
	Example  any
	Examples map[string]*highbase.Example
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
	_ = lowProxy.Build(&root, root.Content[0], nil)
	lowRef := low.NodeReference[*lowbase.SchemaProxy]{
		Value: &lowProxy,
	}
	highSchema := highbase.NewSchemaProxy(&lowRef)
	examples := make(map[string]*highbase.Example)

	for k, v := range values {
		examples[k] = &highbase.Example{
			Value: v,
		}
	}
	return &fakeMockable{
		Schema:   highSchema,
		Example:  example,
		Examples: examples,
	}
}

func TestNewMockGenerator(t *testing.T) {
	mg := NewMockGenerator(MockJSON)
	assert.NotNil(t, mg)
}

func TestNewMockGeneratorWithDictionary(t *testing.T) {
	mg := NewMockGeneratorWithDictionary("", MockJSON)
	assert.NotNil(t, mg)
}

func TestMockGenerator_GenerateJSONMock_BadObject(t *testing.T) {
	type NotMockable struct {
		pizza string
	}

	mg := NewMockGenerator(MockJSON)
	mock, err := mg.GenerateMock(&NotMockable{}, "")
	assert.Error(t, err)
	assert.Nil(t, mock)
}

func TestMockGenerator_GenerateJSONMock_EmptyObject(t *testing.T) {

	mg := NewMockGenerator(MockJSON)
	mock, err := mg.GenerateMock(&fakeMockable{}, "")
	assert.NoError(t, err)
	assert.Nil(t, mock)
}

func TestMockGenerator_GenerateJSONMock_SuppliedExample_JSON(t *testing.T) {

	fakeExample := map[string]any{
		"fish-and-chips": "cod-and-chips-twice",
	}
	fake := createFakeMock(simpleFakeMockSchema, nil, fakeExample)
	mg := NewMockGenerator(MockJSON)
	mock, err := mg.GenerateMock(fake, "")
	assert.NoError(t, err)
	assert.Equal(t, "{\"fish-and-chips\":\"cod-and-chips-twice\"}", string(mock))
}

func TestMockGenerator_GenerateJSONMock_SuppliedExample_YAML(t *testing.T) {

	fakeExample := map[string]any{
		"fish-and-chips": "cod-and-chips-twice",
	}
	fake := createFakeMock(simpleFakeMockSchema, nil, fakeExample)
	mg := NewMockGenerator(MockYAML)
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
	mg := NewMockGenerator(MockJSON)
	mock, err := mg.GenerateMock(fake, "JimmyJammyJimJams") // does not exist
	assert.NoError(t, err)
	assert.Equal(t, "{\"fish-and-chips\":\"cod-and-chips-twice\"}", string(mock))
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
	mg := NewMockGenerator(MockJSON)
	mock, err := mg.GenerateMock(fake, "exampleTwo")
	assert.NoError(t, err)
	assert.Equal(t, "{\"rice-and-peas\":\"brown-or-white-rice\"}", string(mock))
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
	mg := NewMockGenerator(MockYAML)
	mock, err := mg.GenerateMock(fake, "exampleTwo")
	assert.NoError(t, err)
	assert.Equal(t, "rice-and-peas: brown-or-white-rice", strings.TrimSpace(string(mock)))
}

func TestMockGenerator_GenerateJSONMock_NoExamples_JSON(t *testing.T) {

	fake := createFakeMock(simpleFakeMockSchema, nil, nil)
	mg := NewMockGenerator(MockJSON)
	mock, err := mg.GenerateMock(fake, "")
	assert.NoError(t, err)
	assert.Equal(t, "magic-herbs", string(mock))
}

func TestMockGenerator_GenerateJSONMock_NoExamples_YAML(t *testing.T) {

	fake := createFakeMock(simpleFakeMockSchema, nil, nil)
	mg := NewMockGenerator(MockYAML)
	mock, err := mg.GenerateMock(fake, "")
	assert.NoError(t, err)
	assert.Equal(t, "magic-herbs", string(mock))
}

func TestMockGenerator_GenerateJSONMock_Object_NoExamples_JSON(t *testing.T) {

	fake := createFakeMock(objectFakeMockSchema, nil, nil)
	mg := NewMockGenerator(MockJSON)
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
	mg := NewMockGenerator(MockYAML)
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
