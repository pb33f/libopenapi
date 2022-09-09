// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	low "github.com/pb33f/libopenapi/datamodel/low/2.0"
)

type Parameter struct {
	Name             string
	In               string
	Type             string
	Format           string
	Description      string
	Required         bool
	AllowEmptyValue  bool
	Schema           *base.SchemaProxy
	Items            *Items
	CollectionFormat string
	Default          any
	Maximum          int
	ExclusiveMaximum bool
	Minimum          int
	ExclusiveMinimum bool
	MaxLength        int
	MinLength        int
	Pattern          string
	MaxItems         int
	MinItems         int
	UniqueItems      bool
	Enum             []string
	MultipleOf       int
	Extensions       map[string]any
	low              *low.Parameter
}

func NewParameter(parameter *low.Parameter) *Parameter {
	p := new(Parameter)
	p.low = parameter
	p.Extensions = high.ExtractExtensions(parameter.Extensions)
	if !parameter.Name.IsEmpty() {
		p.Name = parameter.Name.Value
	}
	if !parameter.In.IsEmpty() {
		p.In = parameter.In.Value
	}
	if !parameter.Type.IsEmpty() {
		p.Type = parameter.Type.Value
	}
	if !parameter.Format.IsEmpty() {
		p.Format = parameter.Type.Value
	}
	if !parameter.Description.IsEmpty() {
		p.Description = parameter.Description.Value
	}
	if !parameter.Required.IsEmpty() {
		p.Required = parameter.Required.Value
	}
	if !parameter.AllowEmptyValue.IsEmpty() {
		p.AllowEmptyValue = parameter.AllowEmptyValue.Value
	}
	if !parameter.Schema.IsEmpty() {
		p.Schema = base.NewSchemaProxy(&parameter.Schema)
	}
	if !parameter.Items.IsEmpty() {
		p.Items = NewItems(parameter.Items.Value)
	}
	if !parameter.CollectionFormat.IsEmpty() {
		p.CollectionFormat = parameter.CollectionFormat.Value
	}
	if !parameter.Default.IsEmpty() {
		p.Default = parameter.Default.Value
	}
	if !parameter.Maximum.IsEmpty() {
		p.Maximum = parameter.Maximum.Value
	}
	if !parameter.ExclusiveMaximum.IsEmpty() {
		p.ExclusiveMaximum = parameter.ExclusiveMaximum.Value
	}
	if !parameter.Minimum.IsEmpty() {
		p.Minimum = parameter.Minimum.Value
	}
	if !parameter.ExclusiveMinimum.Value {
		p.ExclusiveMinimum = parameter.ExclusiveMinimum.Value
	}
	if !parameter.MaxLength.IsEmpty() {
		p.MaxLength = parameter.MaxLength.Value
	}
	if !parameter.MinLength.IsEmpty() {
		p.MinLength = parameter.MinLength.Value
	}
	if !parameter.Pattern.IsEmpty() {
		p.Pattern = parameter.Pattern.Value
	}
	if !parameter.MinItems.IsEmpty() {
		p.MinItems = parameter.MinItems.Value
	}
	if !parameter.MaxItems.IsEmpty() {
		p.MaxItems = parameter.MaxItems.Value
	}
	if !parameter.UniqueItems.IsEmpty() {
		p.UniqueItems = parameter.UniqueItems.IsEmpty()
	}
	if !parameter.Enum.IsEmpty() {
		p.Enum = parameter.Enum.Value
	}
	if !parameter.MultipleOf.IsEmpty() {
		p.MultipleOf = parameter.MultipleOf.Value
	}

	return p
}
