// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/v2"
	"github.com/pb33f/libopenapi/datamodel/low/v3"
	"reflect"
)

type ResponseChanges struct {
	PropertyChanges
	ExtensionChanges *ExtensionChanges         `json:"extensions,omitempty" yaml:"extensions,omitempty"`
	HeadersChanges   map[string]*HeaderChanges `json:"headers,omitempty" yaml:"headers,omitempty"`

	// v2
	SchemaChanges   *SchemaChanges   `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	ExamplesChanges *ExamplesChanges `json:"examples,omitempty" yaml:"examples,omitempty"`

	// v3
	ContentChanges map[string]*MediaTypeChanges `json:"content,omitempty" yaml:"content,omitempty"`
	LinkChanges    map[string]*LinkChanges      `json:"links,omitempty" yaml:"links,omitempty"`
	ServerChanges  *ServerChanges               `json:"server,omitempty" yaml:"server,omitempty"`
}

func (r *ResponseChanges) TotalChanges() int {
	c := r.PropertyChanges.TotalChanges()
	if r.ExtensionChanges != nil {
		c += r.ExtensionChanges.TotalChanges()
	}
	if r.SchemaChanges != nil {
		c += r.SchemaChanges.TotalChanges()
	}
	if r.ExamplesChanges != nil {
		c += r.ExamplesChanges.TotalChanges()
	}
	for k := range r.HeadersChanges {
		c += r.HeadersChanges[k].TotalChanges()
	}
	for k := range r.ContentChanges {
		c += r.ContentChanges[k].TotalChanges()
	}
	for k := range r.LinkChanges {
		c += r.LinkChanges[k].TotalChanges()
	}
	return c
}

func (r *ResponseChanges) TotalBreakingChanges() int {
	c := r.PropertyChanges.TotalBreakingChanges()
	if r.SchemaChanges != nil {
		c += r.SchemaChanges.TotalBreakingChanges()
	}
	for k := range r.HeadersChanges {
		c += r.HeadersChanges[k].TotalBreakingChanges()
	}
	for k := range r.ContentChanges {
		c += r.ContentChanges[k].TotalBreakingChanges()
	}
	for k := range r.LinkChanges {
		c += r.LinkChanges[k].TotalBreakingChanges()
	}
	return c
}

func CompareResponseV2(l, r *v2.Response) *ResponseChanges {
	return CompareResponse(l, r)
}

func CompareResponseV3(l, r *v3.Response) *ResponseChanges {
	return CompareResponse(l, r)
}

func CompareResponse(l, r any) *ResponseChanges {

	var changes []*Change
	var props []*PropertyCheck

	rc := new(ResponseChanges)

	if reflect.TypeOf(&v2.Response{}) == reflect.TypeOf(l) && reflect.TypeOf(&v2.Response{}) == reflect.TypeOf(r) {

		lResponse := l.(*v2.Response)
		rResponse := r.(*v2.Response)

		// perform hash check to avoid further processing
		if low.AreEqual(lResponse, rResponse) {
			return nil
		}

		// description
		addPropertyCheck(&props, lResponse.Description.ValueNode, rResponse.Description.ValueNode,
			lResponse.Description.Value, rResponse.Description.Value, &changes, v3.DescriptionLabel, false)

		if !lResponse.Schema.IsEmpty() && !rResponse.Schema.IsEmpty() {
			rc.SchemaChanges = CompareSchemas(lResponse.Schema.Value, rResponse.Schema.Value)
		}
		if !lResponse.Schema.IsEmpty() && rResponse.Schema.IsEmpty() {
			CreateChange(&changes, ObjectRemoved, v3.SchemaLabel,
				lResponse.Schema.ValueNode, nil, true,
				lResponse.Schema.Value, nil)
		}
		if lResponse.Schema.IsEmpty() && !rResponse.Schema.IsEmpty() {
			CreateChange(&changes, ObjectAdded, v3.SchemaLabel,
				nil, rResponse.Schema.ValueNode, true,
				nil, rResponse.Schema.Value)
		}

		rc.HeadersChanges =
			CheckMapForChanges(lResponse.Headers.Value, rResponse.Headers.Value,
				&changes, v3.HeadersLabel, CompareHeadersV2)

		if !lResponse.Examples.IsEmpty() && !rResponse.Examples.IsEmpty() {
			rc.ExamplesChanges = CompareExamplesV2(lResponse.Examples.Value, rResponse.Examples.Value)
		}
		if !lResponse.Examples.IsEmpty() && rResponse.Examples.IsEmpty() {
			CreateChange(&changes, PropertyRemoved, v3.ExamplesLabel,
				lResponse.Schema.ValueNode, nil, false,
				lResponse.Schema.Value, nil)
		}
		if lResponse.Examples.IsEmpty() && !rResponse.Examples.IsEmpty() {
			CreateChange(&changes, ObjectAdded, v3.ExamplesLabel,
				nil, rResponse.Schema.ValueNode, false,
				nil, lResponse.Schema.Value)
		}

		rc.ExtensionChanges = CompareExtensions(lResponse.Extensions, rResponse.Extensions)
	}

	if reflect.TypeOf(&v3.Response{}) == reflect.TypeOf(l) && reflect.TypeOf(&v3.Response{}) == reflect.TypeOf(r) {

		lResponse := l.(*v3.Response)
		rResponse := r.(*v3.Response)

		// perform hash check to avoid further processing
		if low.AreEqual(lResponse, rResponse) {
			return nil
		}

		// description
		addPropertyCheck(&props, lResponse.Description.ValueNode, rResponse.Description.ValueNode,
			lResponse.Description.Value, lResponse.Description.Value, &changes, v3.DescriptionLabel, false)

		rc.HeadersChanges =
			CheckMapForChanges(lResponse.Headers.Value, rResponse.Headers.Value,
				&changes, v3.HeadersLabel, CompareHeadersV3)

		rc.ContentChanges =
			CheckMapForChanges(lResponse.Content.Value, rResponse.Content.Value,
				&changes, v3.ContentLabel, CompareMediaTypes)

		rc.LinkChanges =
			CheckMapForChanges(lResponse.Links.Value, rResponse.Links.Value,
				&changes, v3.LinksLabel, CompareLinks)

		rc.ExtensionChanges = CompareExtensions(lResponse.Extensions, rResponse.Extensions)
	}

	CheckProperties(props)
	rc.Changes = changes
	return rc

}
