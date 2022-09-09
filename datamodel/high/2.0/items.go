// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import low "github.com/pb33f/libopenapi/datamodel/low/2.0"

type Items struct {
	Type             string
	Format           string
	CollectionFormat string
	Items            *Items
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
	low              *low.Items
}

func NewItems(items *low.Items) *Items {
	i := new(Items)
	i.low = items
	if !items.Type.IsEmpty() {
		i.Type = items.Type.Value
	}
	if !items.Format.IsEmpty() {
		i.Format = items.Type.Value
	}
	if !items.Items.IsEmpty() {
		i.Items = NewItems(items.Items.Value)
	}
	if !items.CollectionFormat.IsEmpty() {
		i.CollectionFormat = items.CollectionFormat.Value
	}
	if !items.Default.IsEmpty() {
		i.Default = items.Default.Value
	}
	if !items.Maximum.IsEmpty() {
		i.Maximum = items.Maximum.Value
	}
	if !items.ExclusiveMaximum.IsEmpty() {
		i.ExclusiveMaximum = items.ExclusiveMaximum.Value
	}
	if !items.Minimum.IsEmpty() {
		i.Minimum = items.Minimum.Value
	}
	if !items.ExclusiveMinimum.Value {
		i.ExclusiveMinimum = items.ExclusiveMinimum.Value
	}
	if !items.MaxLength.IsEmpty() {
		i.MaxLength = items.MaxLength.Value
	}
	if !items.MinLength.IsEmpty() {
		i.MinLength = items.MinLength.Value
	}
	if !items.Pattern.IsEmpty() {
		i.Pattern = items.Pattern.Value
	}
	if !items.MinItems.IsEmpty() {
		i.MinItems = items.MinItems.Value
	}
	if !items.MaxItems.IsEmpty() {
		i.MaxItems = items.MaxItems.Value
	}
	if !items.UniqueItems.IsEmpty() {
		i.UniqueItems = items.UniqueItems.IsEmpty()
	}
	if !items.Enum.IsEmpty() {
		i.Enum = items.Enum.Value
	}
	if !items.MultipleOf.IsEmpty() {
		i.MultipleOf = items.MultipleOf.Value
	}
	return i
}

func (i *Items) GoLow() *low.Items {
	return i.low
}
