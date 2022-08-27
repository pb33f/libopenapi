// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	low "github.com/pb33f/libopenapi/datamodel/low/3.0"
	"sync"
)

type Schema struct {
	Title                string
	MultipleOf           int
	Maximum              int
	ExclusiveMaximum     int
	Minimum              int
	ExclusiveMinimum     int
	MaxLength            int
	MinLength            int
	Pattern              string
	Format               string
	MaxItems             int
	MinItems             int
	UniqueItems          int
	MaxProperties        int
	MinProperties        int
	Required             []string
	Enum                 []string
	Type                 string
	AllOf                []*SchemaProxy
	OneOf                []*SchemaProxy
	AnyOf                []*SchemaProxy
	Not                  []*SchemaProxy
	Items                []*SchemaProxy
	Properties           map[string]*SchemaProxy
	AdditionalProperties any
	Description          string
	Default              any
	Nullable             bool
	Discriminator        *Discriminator
	ReadOnly             bool
	WriteOnly            bool
	XML                  *XML
	ExternalDocs         *ExternalDoc
	Example              any
	Deprecated           bool
	Extensions           map[string]any
	low                  *low.Schema
}

func NewSchema(schema *low.Schema) *Schema {
	s := new(Schema)
	s.low = schema
	s.Title = schema.Title.Value
	s.MultipleOf = schema.MultipleOf.Value
	s.Maximum = schema.Maximum.Value
	s.ExclusiveMaximum = schema.ExclusiveMaximum.Value
	s.Minimum = schema.Minimum.Value
	s.ExclusiveMinimum = schema.ExclusiveMinimum.Value
	s.MaxLength = schema.MaxLength.Value
	s.MinLength = schema.MinLength.Value
	s.Pattern = schema.Pattern.Value
	s.Format = schema.Format.Value
	s.MaxItems = schema.MaxItems.Value
	s.MinItems = schema.MinItems.Value
	s.MaxProperties = schema.MaxProperties.Value
	s.MinProperties = schema.MinProperties.Value
	s.Type = schema.Type.Value
	s.AdditionalProperties = schema.AdditionalProperties.Value
	s.Description = schema.Description.Value
	s.Default = schema.Default.Value
	s.Nullable = schema.Nullable.Value
	s.ReadOnly = schema.ReadOnly.Value
	s.WriteOnly = schema.WriteOnly.Value
	s.Example = schema.Example.Value
	s.Deprecated = schema.Deprecated.Value
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

	var enum []string
	for i := range schema.Enum.Value {
		enum = append(enum, schema.Enum.Value[i].Value)
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
	buildOutSchema := func(schemas []lowmodel.ValueReference[*low.SchemaProxy], items *[]*SchemaProxy,
		doneChan chan bool, e chan error) {
		bChan := make(chan *SchemaProxy)

		// for every item, build schema async
		buildSchemaChild := func(sch lowmodel.ValueReference[*low.SchemaProxy], bChan chan *SchemaProxy) {
			p := &SchemaProxy{schema: &lowmodel.NodeReference[*low.SchemaProxy]{
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
	var buildProps = func(k lowmodel.KeyReference[string], v lowmodel.ValueReference[*low.SchemaProxy], c chan bool,
		props map[string]*SchemaProxy) {
		defer plock.Unlock()
		plock.Lock()
		props[k.Value] = &SchemaProxy{schema: &lowmodel.NodeReference[*low.SchemaProxy]{
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

func (s *Schema) GoLow() *low.Schema {
	return s.low
}
