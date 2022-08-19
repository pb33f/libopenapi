// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	low "github.com/pb33f/libopenapi/datamodel/low/3.0"
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
	Items                []*Schema
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
		enum = append(req, schema.Enum.Value[i].Value)
	}
	s.Enum = enum

	totalItems := len(schema.AllOf.Value) + len(schema.OneOf.Value) + len(schema.AnyOf.Value) + len(schema.Not.Value) +
		len(schema.Items.Value)

	completedChan := make(chan bool)

	buildOutSchema := func(schemas []lowmodel.NodeReference[*low.Schema], items *[]*Schema, doneChan chan bool) {
		for v := range schemas {
			*items = append(*items, NewSchema(schemas[v].Value))
		}
		doneChan <- true
	}

	allOf := make([]*Schema, len(schema.AllOf.Value))
	oneOf := make([]*Schema, len(schema.OneOf.Value))
	anyOf := make([]*Schema, len(schema.AnyOf.Value))
	not := make([]*Schema, len(schema.Not.Value))
	items := make([]*Schema, len(schema.Items.Value))

	go buildOutSchema(schema.AllOf.Value, &allOf, completedChan)
	go buildOutSchema(schema.AnyOf.Value, &anyOf, completedChan)
	go buildOutSchema(schema.OneOf.Value, &oneOf, completedChan)
	go buildOutSchema(schema.Not.Value, &not, completedChan)
	go buildOutSchema(schema.Items.Value, &items, completedChan)

	complete := 0
	for complete < totalItems {
		select {
		case <-completedChan:
			complete++
		}
	}

	return s

}

func (s *Schema) GoLow() *low.Schema {
	return s.low
}
