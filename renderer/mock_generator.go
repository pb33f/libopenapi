// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package renderer

import (
	"encoding/json"
	"fmt"
	"reflect"

	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
	"gopkg.in/yaml.v3"
)

const Example = "Example"
const Examples = "Examples"
const Schema = "Schema"

type MockType int

const (
	JSON MockType = iota
	YAML
)

// MockGenerator is used to generate mocks for high-level mockable structs or *base.Schema pointers.
// The mock generator will attempt to generate a mock from a struct using the following fields:
//   - Example: any type, this is the default example to use if no examples are present.
//   - Examples: orderedmap.Map[string, *base.Example], this is a map of examples keyed by name.
//   - Schema: *base.SchemaProxy, this is the schema to use if no examples are present.
//
// The mock generator will attempt to generate a mock from a *base.Schema pointer.
// Use NewMockGenerator or NewMockGeneratorWithDictionary to create a new mock generator.
type MockGenerator struct {
	renderer *SchemaRenderer
	mockType MockType
	pretty   bool
}

// NewMockGeneratorWithDictionary creates a new mock generator using a custom dictionary. This is useful if you want to
// use a custom dictionary to generate mocks. The location of a text file with one word per line is expected.
func NewMockGeneratorWithDictionary(dictionaryLocation string, mockType MockType) *MockGenerator {
	renderer := CreateRendererUsingDictionary(dictionaryLocation)
	return &MockGenerator{renderer: renderer, mockType: mockType}
}

// NewMockGenerator creates a new mock generator using the default dictionary. The default is located at /usr/share/dict/words
// on most systems. Windows users will need to use NewMockGeneratorWithDictionary to specify a custom dictionary.
func NewMockGenerator(mockType MockType) *MockGenerator {
	renderer := CreateRendererUsingDefaultDictionary()
	return &MockGenerator{renderer: renderer, mockType: mockType}
}

// SetPretty sets the pretty flag on the mock generator. If true, the mock will be rendered with indentation and newlines.
// If false, the mock will be rendered as a single line which is good for API responses. False is the default.
// This option only effects JSON mocks, there is no concept of pretty printing YAML.
func (mg *MockGenerator) SetPretty() {
	mg.pretty = true
}

// GenerateMock generates a mock for a given high-level mockable struct. The mockable struct must contain the following fields:
// Example: any type, this is the default example to use if no examples are present.
// Examples: orderedmap.Map[string, *base.Example], this is a map of examples keyed by name.
// Schema: *base.SchemaProxy, this is the schema to use if no examples are present.
// The name parameter is optional, if provided, the mock generator will attempt to find an example with the given name.
// If no name is provided, the first example will be used.
func (mg *MockGenerator) GenerateMock(mock any, name string) ([]byte, error) {
	v := reflect.ValueOf(mock).Elem()
	num := v.NumField()
	fieldCount := 0
	for i := 0; i < num; i++ {
		fieldName := v.Type().Field(i).Name
		switch fieldName {
		case Example:
			fieldCount++
		case Examples:
			fieldCount++
		}
	}
	mockReady := false
	// check if all fields are present, if so, we can generate a mock
	if fieldCount == 2 {
		mockReady = true
	}
	if !mockReady {
		return nil, fmt.Errorf("mockable struct only contains %d of the required "+
			"fields (%s, %s)", fieldCount, Example, Examples)
	}

	// if the value has an example, try and render it out as is.
	exampleValue := v.FieldByName(Example).Interface()
	if exampleValue != nil {
		// try and serialize the example value
		return mg.renderMock(exampleValue), nil
	}

	// if there is no example, but there are multi-examples.
	examples := v.FieldByName(Examples)
	examplesValue := examples.Interface()
	if examplesValue != nil && !examples.IsNil() {

		// cast examples to orderedmap.Map[string, any]
		examplesMap := examplesValue.(orderedmap.Map[string, *highbase.Example])

		// if the name is not empty, try and find the example by name
		for pair := orderedmap.First(examplesMap); pair != nil; pair = pair.Next() {
			k, exp := pair.Key(), pair.Value()
			if k == name {
				return mg.renderMock(exp.Value), nil
			}
		}

		// if the name is empty, just return the first example
		for pair := orderedmap.First(examplesMap); pair != nil; pair = pair.Next() {
			exp := pair.Value()
			return mg.renderMock(exp.Value), nil
		}
	}

	// no examples? no problem, we can try and generate a mock from the schema.
	// check if this is a SchemaProxy, if not, then see if it has a Schema, if not, then we can't generate a mock.
	var schemaValue *highbase.Schema
	switch reflect.TypeOf(mock) {
	case reflect.TypeOf(&highbase.Schema{}):
		schemaValue = mock.(*highbase.Schema)
	default:
		if sv, ok := v.FieldByName(Schema).Interface().(*highbase.Schema); ok {
			if sv != nil {
				schemaValue = sv
			}
		}
		if sv, ok := v.FieldByName(Schema).Interface().(*highbase.SchemaProxy); ok {
			if sv != nil {
				schemaValue = sv.Schema()
			}
		}
	}

	if schemaValue != nil {
		renderMap := mg.renderer.RenderSchema(schemaValue)
		if renderMap != nil {
			return mg.renderMock(renderMap), nil
		}
	}
	return nil, nil
}

func (mg *MockGenerator) renderMock(v any) []byte {
	switch {
	case mg.mockType == YAML:
		return mg.renderMockYAML(v)
	default:
		return mg.renderMockJSON(v)
	}
}

func (mg *MockGenerator) renderMockJSON(v any) []byte {
	var data []byte
	// determine the type, render properly.
	switch reflect.ValueOf(v).Kind() {
	case reflect.Map, reflect.Slice, reflect.Array, reflect.Struct, reflect.Ptr:
		if mg.pretty {
			data, _ = json.MarshalIndent(v, "", "  ")
		} else {
			data, _ = json.Marshal(v)
		}
	default:
		data = []byte(fmt.Sprint(v))
	}
	return data
}

func (mg *MockGenerator) renderMockYAML(v any) []byte {
	var data []byte
	// determine the type, render properly.
	switch reflect.ValueOf(v).Kind() {
	case reflect.Map, reflect.Slice, reflect.Array, reflect.Struct, reflect.Ptr:
		data, _ = yaml.Marshal(v)
	default:
		data = []byte(fmt.Sprint(v))
	}
	return data
}
