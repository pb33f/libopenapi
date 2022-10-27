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
	ExclusiveMaximumBool *bool

	// In version 3.1, ExclusiveMaximum is an integer.
	ExclusiveMaximum *int64

	// In versions 2 and 3.0, this ExclusiveMinimum can only be a boolean.
	ExclusiveMinimum *int64

	// In version 3.1, ExclusiveMinimum is an integer.
	ExclusiveMinimumBool *bool

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

	// Compatible with all versions
	Not                  []*SchemaProxy
	Items                []*SchemaProxy
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
	ReadOnly             *bool
	WriteOnly            *bool
	XML                  *XML
	ExternalDocs         *ExternalDoc
	Example              any
	Deprecated           *bool
	Extensions           map[string]any
	low                  *base.Schema
}

// NewSchema will create a new high-level schema from a low-level one.
func NewSchema(schema *base.Schema) *Schema {
	s := new(Schema)
	s.low = schema
	s.Title = schema.Title.Value
	s.MultipleOf = &schema.MultipleOf.Value
	s.Maximum = &schema.Maximum.Value
	s.Minimum = &schema.Minimum.Value
	// if we're dealing with a 3.0 spec using a bool
	if !schema.ExclusiveMaximum.IsEmpty() && schema.ExclusiveMaximum.Value.IsA() {
		s.ExclusiveMaximumBool = &schema.ExclusiveMaximum.Value.A
	}
	// if we're dealing with a 3.1 spec using an int
	if !schema.ExclusiveMaximum.IsEmpty() && schema.ExclusiveMaximum.Value.IsB() {
		s.ExclusiveMaximum = &schema.ExclusiveMaximum.Value.B
	}
	// if we're dealing with a 3.0 spec using a bool
	if !schema.ExclusiveMinimum.IsEmpty() && schema.ExclusiveMinimum.Value.IsA() {
		s.ExclusiveMinimumBool = &schema.ExclusiveMinimum.Value.A
	}
	// if we're dealing with a 3.1 spec, using an int
	if !schema.ExclusiveMinimum.IsEmpty() && schema.ExclusiveMinimum.Value.IsB() {
		s.ExclusiveMinimum = &schema.ExclusiveMinimum.Value.B
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
	s.AdditionalProperties = schema.AdditionalProperties.Value
	s.Description = schema.Description.Value
	s.Default = schema.Default.Value
	if !schema.Nullable.IsEmpty() {
		s.Nullable = &schema.Nullable.Value
	}
	if !schema.ReadOnly.IsEmpty() {
		s.ReadOnly = &schema.ReadOnly.Value
	}
	if !schema.WriteOnly.IsEmpty() {
		s.WriteOnly = &schema.WriteOnly.Value
	}
	if !schema.Deprecated.IsEmpty() {
		s.Deprecated = &schema.Deprecated.Value
	}
	s.Example = schema.Example.Value
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

	// schema async
	buildOutSchema := func(schemas []lowmodel.ValueReference[*base.SchemaProxy], items *[]*SchemaProxy,
		doneChan chan bool, e chan error) {
		bChan := make(chan *SchemaProxy)

		// for every item, build schema async
		buildSchemaChild := func(sch lowmodel.ValueReference[*base.SchemaProxy], bChan chan *SchemaProxy) {
			p := &SchemaProxy{schema: &lowmodel.NodeReference[*base.SchemaProxy]{
				ValueNode: sch.ValueNode,
				Value:     sch.Value,
			}}
			bChan <- p
		}
		totalSchemas := len(schemas)
		for v := range schemas {
			go buildSchemaChild(schemas[v], bChan)
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
	plock := sync.RWMutex{}
	var buildProps = func(k lowmodel.KeyReference[string], v lowmodel.ValueReference[*base.SchemaProxy], c chan bool,
		props map[string]*SchemaProxy) {
		defer plock.Unlock()
		plock.Lock()
		props[k.Value] = &SchemaProxy{schema: &lowmodel.NodeReference[*base.SchemaProxy]{
			Value:     v.Value,
			KeyNode:   k.KeyNode,
			ValueNode: v.ValueNode,
		},
		}
		s.Properties = props
		c <- true
	}

	props := make(map[string]*SchemaProxy)
	for k, v := range schema.Properties.Value {
		go buildProps(k, v, propsChan, props)
	}

	var allOf []*SchemaProxy
	var oneOf []*SchemaProxy
	var anyOf []*SchemaProxy
	var not []*SchemaProxy
	var items []*SchemaProxy

	if !schema.AllOf.IsEmpty() {
		go buildOutSchema(schema.AllOf.Value, &allOf, polyCompletedChan, errChan)
	}
	if !schema.AnyOf.IsEmpty() {
		go buildOutSchema(schema.AnyOf.Value, &anyOf, polyCompletedChan, errChan)
	}
	if !schema.OneOf.IsEmpty() {
		go buildOutSchema(schema.OneOf.Value, &oneOf, polyCompletedChan, errChan)
	}
	if !schema.Not.IsEmpty() {
		go buildOutSchema(schema.Not.Value, &not, polyCompletedChan, errChan)
	}
	if !schema.Items.IsEmpty() {
		go buildOutSchema(schema.Items.Value, &items, polyCompletedChan, errChan)
	}

	completeChildren := 0
	completedProps := 0
	totalProps := len(schema.Properties.Value)
	totalChildren := len(schema.AllOf.Value) + len(schema.OneOf.Value) + len(schema.AnyOf.Value) + len(schema.Items.Value) + len(schema.Not.Value)
	if totalProps+totalChildren > 0 {
	allDone:
		for true {
			select {
			case <-polyCompletedChan:
				completeChildren++
				if totalProps == completedProps && totalChildren == completeChildren {
					break allDone
				}
			case <-propsChan:
				completedProps++
				if totalProps == completedProps && totalChildren == completeChildren {
					break allDone
				}
			}
		}
	}
	s.OneOf = oneOf
	s.AnyOf = anyOf
	s.AllOf = allOf
	s.Items = items
	s.Not = not

	return s
}

// GoLow will return the low-level instance of Schema that was used to create the high level one.
func (s *Schema) GoLow() *base.Schema {
	return s.low
}
