// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package renderer

import (
	"encoding/json"
	"fmt"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"gopkg.in/yaml.v3"
	"reflect"
)

const Example = "Example"
const Examples = "Examples"
const Schema = "Schema"

type MockType int

const (
	MockJSON MockType = iota
	MockYAML
)

type MockGenerator struct {
	renderer *SchemaRenderer
	mockType MockType
}

func NewMockGeneratorWithDictionary(dictionaryLocation string, mockType MockType) *MockGenerator {
	renderer := CreateRendererUsingDictionary(dictionaryLocation)
	return &MockGenerator{renderer: renderer, mockType: mockType}
}

func NewMockGenerator(mockType MockType) *MockGenerator {
	renderer := CreateRendererUsingDefaultDictionary()
	return &MockGenerator{renderer: renderer, mockType: mockType}
}

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

		// cast examples to map[string]interface{}
		examplesMap := examplesValue.(map[string]*highbase.Example)

		// if the name is not empty, try and find the example by name
		for k, exp := range examplesMap {
			if k == name {
				return mg.renderMock(exp.Value), nil
			}
		}

		// if the name is empty, just return the first example
		for _, exp := range examplesMap {
			return mg.renderMock(exp.Value), nil
		}
	}

	// no examples? no problem, we can try and generate a mock from the schema.
	// check if this is a SchemaProxy, if not, then see if it has a Schema, if not, then we can't generate a mock.
	var schemaValue *highbase.Schema
	switch reflect.TypeOf(mock) {
	case reflect.TypeOf(&highbase.SchemaProxy{}):
		schemaValue = mock.(*highbase.SchemaProxy).Schema()
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
	case mg.mockType == MockYAML:
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
		data, _ = json.Marshal(v)
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
