// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/high"
	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"sync"
)

// DynamicValue is used to hold multiple possible values for a schema property. There are two values, a left
// value (A) and a right value (B). The left value (A) is a 3.0 schema property value, the right value (B) is a 3.1
// schema value.
//
// OpenAPI 3.1 treats a Schema as a real JSON schema, which means some properties become incompatible, or others
// now support more than one primitive type or structure.
// The N value is a bit to make it each to know which value (A or B) is used, this prevents having to
// if/else on the value to determine which one is set.
type DynamicValue[A any, B any] struct {
	N int // 0 == A, 1 == B
	A A
	B B
}

// IsA will return true if the 'A' or left value is set. (OpenAPI 3)
func (s *DynamicValue[A, B]) IsA() bool {
	return s.N == 0
}

// IsB will return true if the 'B' or right value is set (OpenAPI 3.1)
func (s *DynamicValue[A, B]) IsB() bool {
	return s.N == 1
}

// Schema represents a JSON Schema that support Swagger, OpenAPI 3 and OpenAPI 3.1
//
// Until 3.1 OpenAPI had a strange relationship with JSON Schema. It's been a super-set/sub-set
// mix, which has been confusing. So, instead of building a bunch of different models, we have compressed
// all variations into a single model that makes it easy to support multiple spec types.
//
//  - v2 schema: https://swagger.io/specification/v2/#schemaObject
//  - v3 schema: https://swagger.io/specification/#schema-object
//  - v3.1 schema: https://spec.openapis.org/oas/v3.1.0#schema-object
type Schema struct {

	// 3.1 only, used to define a dialect for this schema, label is '$schema'.
	SchemaTypeRef string

	// In versions 2 and 3.0, this ExclusiveMaximum can only be a boolean.
	// In version 3.1, ExclusiveMaximum is an integer.
	ExclusiveMaximum *DynamicValue[bool, int64]

	// In versions 2 and 3.0, this ExclusiveMinimum can only be a boolean.
	// In version 3.1, ExclusiveMinimum is an integer.
	ExclusiveMinimum *DynamicValue[bool, int64]

	// In versions 2 and 3.0, this Type is a single value, so array will only ever have one value
	// in version 3.1, Type can be multiple values
	Type []string

	// Schemas are resolved on demand using a SchemaProxy
	AllOf []*SchemaProxy

	// Polymorphic Schemas are only available in version 3+
	OneOf         []*SchemaProxy
	AnyOf         []*SchemaProxy
	Discriminator *Discriminator

	// in 3.1 examples can be an array (which is recommended)
	Examples []any

	// in 3.1 prefixItems provides tuple validation support.
	PrefixItems []*SchemaProxy

	// 3.1 Specific properties
	Contains              *SchemaProxy
	MinContains           *int64
	MaxContains           *int64
	If                    *SchemaProxy
	Else                  *SchemaProxy
	Then                  *SchemaProxy
	DependentSchemas      map[string]*SchemaProxy
	PatternProperties     map[string]*SchemaProxy
	PropertyNames         *SchemaProxy
	UnevaluatedItems      *SchemaProxy
	UnevaluatedProperties *SchemaProxy

	// in 3.1 Items can be a Schema or a boolean
	Items *DynamicValue[*SchemaProxy, bool]

	// Compatible with all versions
	Not                  *SchemaProxy
	Properties           map[string]*SchemaProxy
	Title                string
	MultipleOf           *int64
	Maximum              *int64
	Minimum              *int64
	MaxLength            *int64
	MinLength            *int64
	Pattern              string
	Format               string
	MaxItems             *int64
	MinItems             *int64
	UniqueItems          *int64
	MaxProperties        *int64
	MinProperties        *int64
	Required             []string
	Enum                 []any
	AdditionalProperties any
	Description          string
	Default              any
	Nullable             *bool
	ReadOnly             bool // https://github.com/pb33f/libopenapi/issues/30
	WriteOnly            bool // https://github.com/pb33f/libopenapi/issues/30
	XML                  *XML
	ExternalDocs         *ExternalDoc
	Example              any
	Deprecated           *bool
	Extensions           map[string]any
	low                  *base.Schema

	// Parent Proxy refers back to the low level SchemaProxy that is proxying this schema.
	ParentProxy *SchemaProxy
}

// NewSchema will create a new high-level schema from a low-level one.
func NewSchema(schema *base.Schema) *Schema {
	s := new(Schema)
	s.low = schema
	s.Title = schema.Title.Value
	if !schema.MultipleOf.IsEmpty() {
		s.MultipleOf = &schema.MultipleOf.Value
	}
	if !schema.Maximum.IsEmpty() {
		s.Maximum = &schema.Maximum.Value
	}
	if !schema.Minimum.IsEmpty() {
		s.Minimum = &schema.Minimum.Value
	}
	// if we're dealing with a 3.0 spec using a bool
	if !schema.ExclusiveMaximum.IsEmpty() && schema.ExclusiveMaximum.Value.IsA() {
		s.ExclusiveMaximum = &DynamicValue[bool, int64]{
			A: schema.ExclusiveMaximum.Value.A,
		}
	}
	// if we're dealing with a 3.1 spec using an int
	if !schema.ExclusiveMaximum.IsEmpty() && schema.ExclusiveMaximum.Value.IsB() {
		s.ExclusiveMaximum = &DynamicValue[bool, int64]{
			B: schema.ExclusiveMaximum.Value.B,
		}
	}
	// if we're dealing with a 3.0 spec using a bool
	if !schema.ExclusiveMinimum.IsEmpty() && schema.ExclusiveMinimum.Value.IsA() {
		s.ExclusiveMinimum = &DynamicValue[bool, int64]{
			A: schema.ExclusiveMinimum.Value.A,
		}
	}
	// if we're dealing with a 3.1 spec, using an int
	if !schema.ExclusiveMinimum.IsEmpty() && schema.ExclusiveMinimum.Value.IsB() {
		s.ExclusiveMinimum = &DynamicValue[bool, int64]{
			B: schema.ExclusiveMinimum.Value.B,
		}
	}
	if !schema.MaxLength.IsEmpty() {
		s.MaxLength = &schema.MaxLength.Value
	}
	if !schema.MinLength.IsEmpty() {
		s.MinLength = &schema.MinLength.Value
	}
	if !schema.MaxItems.IsEmpty() {
		s.MaxItems = &schema.MaxItems.Value
	}
	if !schema.MinItems.IsEmpty() {
		s.MinItems = &schema.MinItems.Value
	}
	if !schema.MaxProperties.IsEmpty() {
		s.MaxProperties = &schema.MaxProperties.Value
	}
	if !schema.MinProperties.IsEmpty() {
		s.MinProperties = &schema.MinProperties.Value
	}

	if !schema.MaxContains.IsEmpty() {
		s.MaxContains = &schema.MaxContains.Value
	}
	if !schema.MinContains.IsEmpty() {
		s.MinContains = &schema.MinContains.Value
	}
	if !schema.Contains.IsEmpty() {
		s.Contains = &SchemaProxy{schema: &lowmodel.NodeReference[*base.SchemaProxy]{
			ValueNode: schema.Contains.ValueNode,
			Value:     schema.Contains.Value,
		}}
	}

	if !schema.If.IsEmpty() {
		s.If = &SchemaProxy{schema: &lowmodel.NodeReference[*base.SchemaProxy]{
			ValueNode: schema.If.ValueNode,
			Value:     schema.If.Value,
		}}
	}
	if !schema.Else.IsEmpty() {
		s.Else = &SchemaProxy{schema: &lowmodel.NodeReference[*base.SchemaProxy]{
			ValueNode: schema.Else.ValueNode,
			Value:     schema.Else.Value,
		}}
	}
	if !schema.Then.IsEmpty() {
		s.Then = &SchemaProxy{schema: &lowmodel.NodeReference[*base.SchemaProxy]{
			ValueNode: schema.Then.ValueNode,
			Value:     schema.Then.Value,
		}}
	}
	if !schema.PropertyNames.IsEmpty() {
		s.PropertyNames = &SchemaProxy{schema: &lowmodel.NodeReference[*base.SchemaProxy]{
			ValueNode: schema.PropertyNames.ValueNode,
			Value:     schema.PropertyNames.Value,
		}}
	}
	if !schema.UnevaluatedItems.IsEmpty() {
		s.UnevaluatedItems = &SchemaProxy{schema: &lowmodel.NodeReference[*base.SchemaProxy]{
			ValueNode: schema.UnevaluatedItems.ValueNode,
			Value:     schema.UnevaluatedItems.Value,
		}}
	}
	if !schema.UnevaluatedProperties.IsEmpty() {
		s.UnevaluatedProperties = &SchemaProxy{schema: &lowmodel.NodeReference[*base.SchemaProxy]{
			ValueNode: schema.UnevaluatedProperties.ValueNode,
			Value:     schema.UnevaluatedProperties.Value,
		}}
	}

	s.Pattern = schema.Pattern.Value
	s.Format = schema.Format.Value

	// 3.0 spec is a single value
	if !schema.Type.IsEmpty() && schema.Type.Value.IsA() {
		s.Type = []string{schema.Type.Value.A}
	}
	// 3.1 spec may have multiple values
	if !schema.Type.IsEmpty() && schema.Type.Value.IsB() {
		for i := range schema.Type.Value.B {
			s.Type = append(s.Type, schema.Type.Value.B[i].Value)
		}
	}
	if schema.AdditionalProperties.Value != nil {
		if addPropSchema, ok := schema.AdditionalProperties.Value.(*base.SchemaProxy); ok {
			s.AdditionalProperties = NewSchemaProxy(&lowmodel.NodeReference[*base.SchemaProxy]{
				KeyNode:   schema.AdditionalProperties.KeyNode,
				ValueNode: schema.AdditionalProperties.ValueNode,
				Value:     addPropSchema,
			})
		} else {
			s.AdditionalProperties = schema.AdditionalProperties.Value
		}
	}
	s.Description = schema.Description.Value
	s.Default = schema.Default.Value
	if !schema.Nullable.IsEmpty() {
		s.Nullable = &schema.Nullable.Value
	}
	if !schema.ReadOnly.IsEmpty() {
		s.ReadOnly = schema.ReadOnly.Value
	}
	if !schema.WriteOnly.IsEmpty() {
		s.WriteOnly = schema.WriteOnly.Value
	}
	if !schema.Deprecated.IsEmpty() {
		s.Deprecated = &schema.Deprecated.Value
	}
	s.Example = schema.Example.Value
	if len(schema.Examples.Value) > 0 {
		examples := make([]any, len(schema.Examples.Value))
		for i := 0; i < len(schema.Examples.Value); i++ {
			examples[i] = schema.Examples.Value[i].Value
		}
		s.Examples = examples
	}
	s.Extensions = high.ExtractExtensions(schema.Extensions)
	if !schema.Discriminator.IsEmpty() {
		s.Discriminator = NewDiscriminator(schema.Discriminator.Value)
	}
	if !schema.XML.IsEmpty() {
		s.XML = NewXML(schema.XML.Value)
	}
	if !schema.ExternalDocs.IsEmpty() {
		s.ExternalDocs = NewExternalDoc(schema.ExternalDocs.Value)
	}
	var req []string
	for i := range schema.Required.Value {
		req = append(req, schema.Required.Value[i].Value)
	}
	s.Required = req

	var enum []any
	for i := range schema.Enum.Value {
		enum = append(enum, fmt.Sprint(schema.Enum.Value[i].Value))
	}
	s.Enum = enum

	// async work.
	// any polymorphic properties need to be handled in their own threads
	// any properties each need to be processed in their own thread.
	// we go as fast as we can.
	polyCompletedChan := make(chan bool)
	propsChan := make(chan bool)
	errChan := make(chan error)

	// for every item, build schema async
	buildSchema := func(sch lowmodel.ValueReference[*base.SchemaProxy], bChan chan *SchemaProxy) {
		p := &SchemaProxy{schema: &lowmodel.NodeReference[*base.SchemaProxy]{
			ValueNode: sch.ValueNode,
			Value:     sch.Value,
		}}
		bChan <- p
	}

	// schema async
	buildOutSchemas := func(schemas []lowmodel.ValueReference[*base.SchemaProxy], items *[]*SchemaProxy,
		doneChan chan bool, e chan error) {
		bChan := make(chan *SchemaProxy)
		totalSchemas := len(schemas)
		for v := range schemas {
			go buildSchema(schemas[v], bChan)
		}
		j := 0
		for j < totalSchemas {
			select {
			case t := <-bChan:
				j++
				*items = append(*items, t)
			}
		}
		doneChan <- true
	}

	// props async
	var plock sync.Mutex
	var buildProps = func(k lowmodel.KeyReference[string], v lowmodel.ValueReference[*base.SchemaProxy], c chan bool,
		props map[string]*SchemaProxy, sw int) {
		plock.Lock()
		props[k.Value] = &SchemaProxy{
			schema: &lowmodel.NodeReference[*base.SchemaProxy]{
				Value:     v.Value,
				KeyNode:   k.KeyNode,
				ValueNode: v.ValueNode,
			},
		}
		plock.Unlock()

		switch sw {
		case 0:
			s.Properties = props
		case 1:
			s.DependentSchemas = props
		case 2:
			s.PatternProperties = props
		}
		c <- true
	}

	props := make(map[string]*SchemaProxy)
	for k, v := range schema.Properties.Value {
		go buildProps(k, v, propsChan, props, 0)
	}

	dependents := make(map[string]*SchemaProxy)
	for k, v := range schema.DependentSchemas.Value {
		go buildProps(k, v, propsChan, dependents, 1)
	}
	patternProps := make(map[string]*SchemaProxy)
	for k, v := range schema.PatternProperties.Value {
		go buildProps(k, v, propsChan, patternProps, 2)
	}

	var allOf []*SchemaProxy
	var oneOf []*SchemaProxy
	var anyOf []*SchemaProxy
	var not *SchemaProxy
	var items *DynamicValue[*SchemaProxy, bool]
	var prefixItems []*SchemaProxy

	children := 0
	if !schema.AllOf.IsEmpty() {
		children++
		go buildOutSchemas(schema.AllOf.Value, &allOf, polyCompletedChan, errChan)
	}
	if !schema.AnyOf.IsEmpty() {
		children++
		go buildOutSchemas(schema.AnyOf.Value, &anyOf, polyCompletedChan, errChan)
	}
	if !schema.OneOf.IsEmpty() {
		children++
		go buildOutSchemas(schema.OneOf.Value, &oneOf, polyCompletedChan, errChan)
	}
	if !schema.Not.IsEmpty() {
		not = NewSchemaProxy(&schema.Not)
	}
	if !schema.Items.IsEmpty() {
		if schema.Items.Value.IsA() {
			items = &DynamicValue[*SchemaProxy, bool]{A: &SchemaProxy{schema: &lowmodel.NodeReference[*base.SchemaProxy]{
				ValueNode: schema.Items.ValueNode,
				Value:     schema.Items.Value.A,
				KeyNode:   schema.Items.KeyNode,
			}}}
		} else {
			items = &DynamicValue[*SchemaProxy, bool]{B: schema.Items.Value.B}
		}
	}
	if !schema.PrefixItems.IsEmpty() {
		children++
		go buildOutSchemas(schema.PrefixItems.Value, &prefixItems, polyCompletedChan, errChan)
	}

	completeChildren := 0
	completedProps := 0
	totalProps := len(schema.Properties.Value) + len(schema.DependentSchemas.Value) + len(schema.PatternProperties.Value)
	if totalProps+children > 0 {
	allDone:
		for true {
			select {
			case <-polyCompletedChan:
				completeChildren++
				if totalProps == completedProps && children == completeChildren {
					break allDone
				}
			case <-propsChan:
				completedProps++
				if totalProps == completedProps && children == completeChildren {
					break allDone
				}
			}
		}
	}
	s.OneOf = oneOf
	s.AnyOf = anyOf
	s.AllOf = allOf
	s.Items = items
	s.PrefixItems = prefixItems
	s.Not = not
	return s
}

// GoLow will return the low-level instance of Schema that was used to create the high level one.
func (s *Schema) GoLow() *base.Schema {
	return s.low
}
