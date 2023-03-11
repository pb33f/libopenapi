// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"crypto/sha256"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
	"sort"
	"strings"
)

// ParameterDefinitions is a low-level representation of a Swagger / OpenAPI 2 Parameters Definitions object.
//
// ParameterDefinitions holds parameters to be reused across operations. Parameter definitions can be
// referenced to the ones defined here. It does not define global operation parameters
//  - https://swagger.io/specification/v2/#parametersDefinitionsObject
type ParameterDefinitions struct {
	Definitions map[low.KeyReference[string]]low.ValueReference[*Parameter]
}

// ResponsesDefinitions is a low-level representation of a Swagger / OpenAPI 2 Responses Definitions object.
//
// ResponsesDefinitions is an object to hold responses to be reused across operations. Response definitions can be
// referenced to the ones defined here. It does not define global operation responses
//  - https://swagger.io/specification/v2/#responsesDefinitionsObject
type ResponsesDefinitions struct {
	Definitions map[low.KeyReference[string]]low.ValueReference[*Response]
}

// SecurityDefinitions is a low-level representation of a Swagger / OpenAPI 2 Security Definitions object.
//
// A declaration of the security schemes available to be used in the specification. This does not enforce the security
// schemes on the operations and only serves to provide the relevant details for each scheme
//  - https://swagger.io/specification/v2/#securityDefinitionsObject
type SecurityDefinitions struct {
	Definitions map[low.KeyReference[string]]low.ValueReference[*SecurityScheme]
}

// Definitions is a low-level representation of a Swagger / OpenAPI 2 Definitions object
//
// An object to hold data types that can be consumed and produced by operations. These data types can be primitives,
// arrays or models.
//  - https://swagger.io/specification/v2/#definitionsObject
type Definitions struct {
	Schemas map[low.KeyReference[string]]low.ValueReference[*base.SchemaProxy]
}

// FindSchema will attempt to locate a base.SchemaProxy instance using a name.
func (d *Definitions) FindSchema(schema string) *low.ValueReference[*base.SchemaProxy] {
	return low.FindItemInMap[*base.SchemaProxy](schema, d.Schemas)
}

// FindParameter will attempt to locate a Parameter instance using a name.
func (pd *ParameterDefinitions) FindParameter(parameter string) *low.ValueReference[*Parameter] {
	return low.FindItemInMap[*Parameter](parameter, pd.Definitions)
}

// FindResponse will attempt to locate a Response instance using a name.
func (r *ResponsesDefinitions) FindResponse(response string) *low.ValueReference[*Response] {
	return low.FindItemInMap[*Response](response, r.Definitions)
}

// FindSecurityDefinition will attempt to locate a SecurityScheme using a name.
func (s *SecurityDefinitions) FindSecurityDefinition(securityDef string) *low.ValueReference[*SecurityScheme] {
	return low.FindItemInMap[*SecurityScheme](securityDef, s.Definitions)
}

// Build will extract all definitions into SchemaProxy instances.
func (d *Definitions) Build(root *yaml.Node, idx *index.SpecIndex) error {
	errorChan := make(chan error)
	resultChan := make(chan definitionResult[*base.SchemaProxy])
	var defLabel *yaml.Node
	totalDefinitions := 0
	for i := range root.Content {
		if i%2 == 0 {
			defLabel = root.Content[i]
			continue
		}
		totalDefinitions++
		var buildFunc = func(label *yaml.Node, value *yaml.Node, idx *index.SpecIndex,
			r chan definitionResult[*base.SchemaProxy], e chan error) {

			obj, err, _, rv := low.ExtractObjectRaw[*base.SchemaProxy](value, idx)
			if err != nil {
				e <- err
			}
			r <- definitionResult[*base.SchemaProxy]{k: label, v: low.ValueReference[*base.SchemaProxy]{
				Value: obj, ValueNode: value, Reference: rv,
			}}
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
			completedDefs++
			results[low.KeyReference[string]{
				Value:   sch.k.Value,
				KeyNode: sch.k,
			}] = sch.v
		}
	}
	d.Schemas = results
	return nil
}

// Hash will return a consistent SHA256 Hash of the Definitions object
func (d *Definitions) Hash() [32]byte {
	var f []string
	keys := make([]string, len(d.Schemas))
	z := 0
	for k := range d.Schemas {
		keys[z] = k.Value
		z++
	}
	sort.Strings(keys)
	for k := range keys {
		f = append(f, low.GenerateHashString(d.FindSchema(keys[k]).Value))
	}
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}

// Build will extract all ParameterDefinitions into Parameter instances.
func (pd *ParameterDefinitions) Build(root *yaml.Node, idx *index.SpecIndex) error {
	errorChan := make(chan error)
	resultChan := make(chan definitionResult[*Parameter])
	var defLabel *yaml.Node
	totalDefinitions := 0
	for i := range root.Content {
		if i%2 == 0 {
			defLabel = root.Content[i]
			continue
		}
		totalDefinitions++
		var buildFunc = func(label *yaml.Node, value *yaml.Node, idx *index.SpecIndex,
			r chan definitionResult[*Parameter], e chan error) {

			obj, err, _, rv := low.ExtractObjectRaw[*Parameter](value, idx)
			if err != nil {
				e <- err
			}
			r <- definitionResult[*Parameter]{k: label, v: low.ValueReference[*Parameter]{Value: obj,
				ValueNode: value, Reference: rv}}
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
			completedDefs++
			results[low.KeyReference[string]{
				Value:   sch.k.Value,
				KeyNode: sch.k,
			}] = sch.v
		}
	}
	pd.Definitions = results
	return nil
}

// re-usable struct for holding results as k/v pairs.
type definitionResult[T any] struct {
	k *yaml.Node
	v low.ValueReference[T]
}

// Build will extract all ResponsesDefinitions into Response instances.
func (r *ResponsesDefinitions) Build(root *yaml.Node, idx *index.SpecIndex) error {
	errorChan := make(chan error)
	resultChan := make(chan definitionResult[*Response])
	var defLabel *yaml.Node
	totalDefinitions := 0
	for i := range root.Content {
		if i%2 == 0 {
			defLabel = root.Content[i]
			continue
		}
		totalDefinitions++
		var buildFunc = func(label *yaml.Node, value *yaml.Node, idx *index.SpecIndex,
			r chan definitionResult[*Response], e chan error) {

			obj, err, _, rv := low.ExtractObjectRaw[*Response](value, idx)
			if err != nil {
				e <- err
			}
			r <- definitionResult[*Response]{k: label, v: low.ValueReference[*Response]{Value: obj,
				ValueNode: value, Reference: rv}}
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
			completedDefs++
			results[low.KeyReference[string]{
				Value:   sch.k.Value,
				KeyNode: sch.k,
			}] = sch.v
		}
	}
	r.Definitions = results
	return nil
}

// Build will extract all SecurityDefinitions into SecurityScheme instances.
func (s *SecurityDefinitions) Build(root *yaml.Node, idx *index.SpecIndex) error {
	errorChan := make(chan error)
	resultChan := make(chan definitionResult[*SecurityScheme])
	var defLabel *yaml.Node
	totalDefinitions := 0
	for i := range root.Content {
		if i%2 == 0 {
			defLabel = root.Content[i]
			continue
		}
		totalDefinitions++
		var buildFunc = func(label *yaml.Node, value *yaml.Node, idx *index.SpecIndex,
			r chan definitionResult[*SecurityScheme], e chan error) {

			obj, err, _, rv := low.ExtractObjectRaw[*SecurityScheme](value, idx)
			if err != nil {
				e <- err
			}
			r <- definitionResult[*SecurityScheme]{k: label, v: low.ValueReference[*SecurityScheme]{
				Value: obj, ValueNode: value, Reference: rv,
			}}
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
			completedDefs++
			results[low.KeyReference[string]{
				Value:   sch.k.Value,
				KeyNode: sch.k,
			}] = sch.v
		}
	}
	s.Definitions = results
	return nil
}
