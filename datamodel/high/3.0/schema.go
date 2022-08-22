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
	AllOf                []*Schema
	OneOf                []*Schema
	AnyOf                []*Schema
	Not                  []*Schema
	Items                *Schema
	Properties           map[string]*Schema
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

	// schema async
	buildOutSchema := func(schemas []lowmodel.NodeReference[*low.Schema], items *[]*Schema, doneChan chan bool) {
		bChan := make(chan *Schema)

		// for every item, build schema async
		buildSchemaChild := func(sch lowmodel.NodeReference[*low.Schema], bChan chan *Schema) {
			if ss := getSeenSchema(sch.GenerateMapKey()); ss != nil {
				bChan <- ss
				return
			}
			ns := NewSchema(sch.Value)
			addSeenSchema(sch.GenerateMapKey(), ns)
			bChan <- ns
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
	var buildProps = func(k lowmodel.KeyReference[string], v lowmodel.ValueReference[*low.Schema], c chan bool,
		props map[string]*Schema) {
		if ss := getSeenSchema(v.GenerateMapKey()); ss != nil {
			defer plock.Unlock()
			plock.Lock()
			props[k.Value] = ss

		} else {
			defer plock.Unlock()
			plock.Lock()
			props[k.Value] = NewSchema(v.Value)
			addSeenSchema(k.GenerateMapKey(), props[k.Value])
		}
		s.Properties = props
		c <- true
	}

	props := make(map[string]*Schema)
	for k, v := range schema.Properties.Value {
		go buildProps(k, v, propsChan, props)
	}

	var allOf []*Schema
	var oneOf []*Schema
	var anyOf []*Schema
	var not []*Schema
	var items []*Schema

	if !schema.AllOf.IsEmpty() {
		go buildOutSchema(schema.AllOf.Value, &allOf, polyCompletedChan)
	}
	if !schema.AnyOf.IsEmpty() {
		go buildOutSchema(schema.AnyOf.Value, &anyOf, polyCompletedChan)
	}
	if !schema.OneOf.IsEmpty() {
		go buildOutSchema(schema.OneOf.Value, &oneOf, polyCompletedChan)
	}
	if !schema.Not.IsEmpty() {
		go buildOutSchema(schema.Not.Value, &not, polyCompletedChan)
	}
	if !schema.Items.IsEmpty() {
		// items is only a single prop, however the method uses an array, so pack it up in one.
		var itms []lowmodel.NodeReference[*low.Schema]
		itms = append(itms, lowmodel.NodeReference[*low.Schema]{
			Value:     schema.Items.Value,
			KeyNode:   schema.Items.KeyNode,
			ValueNode: schema.Items.ValueNode,
		})
		go buildOutSchema(itms, &items, polyCompletedChan)
	}

	completePoly := 0
	completedProps := 0
	totalProps := len(schema.Properties.Value)
	totalPoly := len(schema.AllOf.Value) + len(schema.OneOf.Value) + len(schema.AnyOf.Value) + len(schema.Not.Value)
	if !schema.Items.IsEmpty() {
		totalPoly++ // only a single item can be present.
	}
	if totalProps+totalPoly > 0 {
	allDone:
		for true {
			select {
			case <-polyCompletedChan:
				completePoly++
				if totalProps == completedProps && totalPoly == completePoly {
					break allDone
				}
			case <-propsChan:
				completedProps++
				if totalProps == completedProps && totalPoly == completePoly {
					break allDone
				}
			}
		}
	}
	s.OneOf = oneOf
	s.AnyOf = anyOf
	s.AllOf = allOf
	s.Not = not
	if len(items) > 0 {
		s.Items = items[0] // there will only ever be one.
	}

	return s
}

func (s *Schema) GoLow() *low.Schema {
	return s.low
}
