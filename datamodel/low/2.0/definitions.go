// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
)

const (
	DefinitionsLabel = "definitions"
)

type ParameterDefinitions struct {
	Definitions map[low.KeyReference[string]]low.ValueReference[*Parameter]
}

type ResponsesDefinitions struct {
	Definitions map[low.KeyReference[string]]low.ValueReference[*Response]
}

type SecurityDefinitions struct {
	Definitions map[low.KeyReference[string]]low.ValueReference[*SecurityScheme]
}

type Definitions struct {
	Schemas map[low.KeyReference[string]]low.ValueReference[*base.SchemaProxy]
}

func (d *Definitions) FindSchema(schema string) *low.ValueReference[*base.SchemaProxy] {
	return low.FindItemInMap[*base.SchemaProxy](schema, d.Schemas)
}

func (pd *ParameterDefinitions) FindParameter(schema string) *low.ValueReference[*Parameter] {
	return low.FindItemInMap[*Parameter](schema, pd.Definitions)
}

func (r *ResponsesDefinitions) FindResponse(schema string) *low.ValueReference[*Response] {
	return low.FindItemInMap[*Response](schema, r.Definitions)
}

func (s *SecurityDefinitions) FindSecurityScheme(schema string) *low.ValueReference[*SecurityScheme] {
	return low.FindItemInMap[*SecurityScheme](schema, s.Definitions)
}

func (d *Definitions) Build(root *yaml.Node, idx *index.SpecIndex) error {
	errorChan := make(chan error)
	resultChan := make(chan definitionResult)
	var defLabel *yaml.Node
	totalDefinitions := 0
	for i := range root.Content {
		if i%2 == 0 {
			defLabel = root.Content[i]
			continue
		}
		totalDefinitions++
		var buildFunc = func(label *yaml.Node, value *yaml.Node, idx *index.SpecIndex, r chan definitionResult, e chan error) {
			obj, err := low.ExtractObjectRaw[*base.SchemaProxy](value, idx)
			if err != nil {
				e <- err
			}
			r <- definitionResult{k: label, v: low.ValueReference[any]{Value: obj, ValueNode: value}}
		}
		go buildFunc(defLabel, root.Content[i], idx, resultChan, errorChan)
	}

	completedDefs := 0
	results := make(map[low.KeyReference[string]]low.ValueReference[*base.SchemaProxy])
	for completedDefs < totalDefinitions {
		select {
		case err := <-errorChan:
			return err
		case sch := <-resultChan:
			results[low.KeyReference[string]{
				Value:   sch.k.Value,
				KeyNode: sch.k,
			}] = sch.v.(low.ValueReference[*base.SchemaProxy])
		}
	}
	d.Schemas = results
	return nil
}

func (pd *ParameterDefinitions) Build(root *yaml.Node, idx *index.SpecIndex) error {
	errorChan := make(chan error)
	resultChan := make(chan definitionResult)
	var defLabel *yaml.Node
	totalDefinitions := 0
	for i := range root.Content {
		if i%2 == 0 {
			defLabel = root.Content[i]
			continue
		}
		totalDefinitions++
		var buildFunc = func(label *yaml.Node, value *yaml.Node, idx *index.SpecIndex, r chan definitionResult, e chan error) {
			obj, err := low.ExtractObjectRaw[*Parameter](value, idx)
			if err != nil {
				e <- err
			}
			r <- definitionResult{k: label, v: low.ValueReference[any]{Value: obj, ValueNode: value}}
		}
		go buildFunc(defLabel, root.Content[i], idx, resultChan, errorChan)
	}

	completedDefs := 0
	results := make(map[low.KeyReference[string]]low.ValueReference[*Parameter])
	for completedDefs < totalDefinitions {
		select {
		case err := <-errorChan:
			return err
		case sch := <-resultChan:
			results[low.KeyReference[string]{
				Value:   sch.k.Value,
				KeyNode: sch.k,
			}] = sch.v.(low.ValueReference[*Parameter])
		}
	}
	pd.Definitions = results
	return nil
}

type definitionResult struct {
	k *yaml.Node
	v any
}

func (r *ResponsesDefinitions) Build(root *yaml.Node, idx *index.SpecIndex) error {
	errorChan := make(chan error)
	resultChan := make(chan definitionResult)
	var defLabel *yaml.Node
	totalDefinitions := 0
	for i := range root.Content {
		if i%2 == 0 {
			defLabel = root.Content[i]
			continue
		}
		totalDefinitions++
		var buildFunc = func(label *yaml.Node, value *yaml.Node, idx *index.SpecIndex, r chan definitionResult, e chan error) {
			obj, err := low.ExtractObjectRaw[*Response](value, idx)
			if err != nil {
				e <- err
			}
			r <- definitionResult{k: label, v: low.ValueReference[any]{Value: obj, ValueNode: value}}
		}
		go buildFunc(defLabel, root.Content[i], idx, resultChan, errorChan)
	}

	completedDefs := 0
	results := make(map[low.KeyReference[string]]low.ValueReference[*Response])
	for completedDefs < totalDefinitions {
		select {
		case err := <-errorChan:
			return err
		case sch := <-resultChan:
			results[low.KeyReference[string]{
				Value:   sch.k.Value,
				KeyNode: sch.k,
			}] = sch.v.(low.ValueReference[*Response])
		}
	}
	r.Definitions = results
	return nil
}

func (s *SecurityDefinitions) Build(root *yaml.Node, idx *index.SpecIndex) error {
	errorChan := make(chan error)
	resultChan := make(chan definitionResult)
	var defLabel *yaml.Node
	totalDefinitions := 0
	for i := range root.Content {
		if i%2 == 0 {
			defLabel = root.Content[i]
			continue
		}
		totalDefinitions++
		var buildFunc = func(label *yaml.Node, value *yaml.Node, idx *index.SpecIndex, r chan definitionResult, e chan error) {
			obj, err := low.ExtractObjectRaw[*SecurityScheme](value, idx)
			if err != nil {
				e <- err
			}
			r <- definitionResult{k: label, v: low.ValueReference[any]{Value: obj, ValueNode: value}}
		}
		go buildFunc(defLabel, root.Content[i], idx, resultChan, errorChan)
	}

	completedDefs := 0
	results := make(map[low.KeyReference[string]]low.ValueReference[*SecurityScheme])
	for completedDefs < totalDefinitions {
		select {
		case err := <-errorChan:
			return err
		case sch := <-resultChan:
			results[low.KeyReference[string]{
				Value:   sch.k.Value,
				KeyNode: sch.k,
			}] = sch.v.(low.ValueReference[*SecurityScheme])
		}
	}
	s.Definitions = results
	return nil
}
