// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low/v3"
)

type MediaTypeChanges struct {
	PropertyChanges
	SchemaChanges    *SchemaChanges
	ExtensionChanges *ExtensionChanges
	ExampleChanges   map[string]*ExampleChanges
	EncodingChanges  map[string]*EncodingChanges
}

func (m *MediaTypeChanges) TotalChanges() int {
	c := m.PropertyChanges.TotalChanges()
	for k := range m.ExampleChanges {
		c += m.ExampleChanges[k].TotalChanges()
	}
	if m.SchemaChanges != nil {
		c += m.SchemaChanges.TotalChanges()
	}
	if len(m.EncodingChanges) > 0 {
		for i := range m.EncodingChanges {
			c += m.EncodingChanges[i].TotalChanges()
		}
	}
	if m.ExtensionChanges != nil {
		c += m.ExtensionChanges.TotalChanges()
	}
	return c
}

func (m *MediaTypeChanges) TotalBreakingChanges() int {
	c := m.PropertyChanges.TotalBreakingChanges()
	for k := range m.ExampleChanges {
		c += m.ExampleChanges[k].TotalBreakingChanges()
	}
	if m.SchemaChanges != nil {
		c += m.SchemaChanges.TotalBreakingChanges()
	}
	if len(m.EncodingChanges) > 0 {
		for i := range m.EncodingChanges {
			c += m.EncodingChanges[i].TotalBreakingChanges()
		}
	}
	return c
}

func CompareMediaTypes(l, r *v3.MediaType) *MediaTypeChanges {

	var props []*PropertyCheck
	var changes []*Change

	mc := new(MediaTypeChanges)

	// Example
	addPropertyCheck(&props, l.Example.ValueNode, r.Example.ValueNode,
		l.Example.Value, r.Example.Value, &changes, v3.ExampleLabel, false)

	CheckProperties(props)

	// schema
	if !l.Schema.IsEmpty() && !r.Schema.IsEmpty() {
		mc.SchemaChanges = CompareSchemas(l.Schema.Value, r.Schema.Value)
	}
	if !l.Schema.IsEmpty() && r.Schema.IsEmpty() {
		CreateChange(&changes, ObjectRemoved, v3.SchemaLabel, l.Schema.ValueNode,
			nil, true, l.Schema.Value, nil)
	}
	if l.Schema.IsEmpty() && !r.Schema.IsEmpty() {
		CreateChange(&changes, ObjectAdded, v3.SchemaLabel, nil,
			r.Schema.ValueNode, true, nil, r.Schema.Value)
	}

	// examples
	mc.ExampleChanges = CheckMapForChanges(l.Examples.Value, r.Examples.Value,
		&changes, v3.ExamplesLabel, CompareExamples)

	// encoding
	mc.EncodingChanges = CheckMapForChanges(l.Encoding.Value, r.Encoding.Value,
		&changes, v3.EncodingLabel, CompareEncoding)

	mc.ExtensionChanges = CompareExtensions(l.Extensions, r.Extensions)
	mc.Changes = changes

	if mc.TotalChanges() <= 0 {
		return nil
	}
	return mc
}
