// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/datamodel/low/v2"
	"github.com/pb33f/libopenapi/datamodel/low/v3"
	"reflect"
)

type DocumentChanges struct {
	PropertyChanges
	InfoChanges                *InfoChanges
	PathsChanges               *PathsChanges
	TagChanges                 *TagChanges
	ExternalDocChanges         *ExternalDocChanges
	WebhookChanges             map[string]*PathItemChanges
	ServerChanges              []*ServerChanges
	SecurityRequirementChanges []*SecurityRequirementChanges
	ComponentsChanges          *ComponentsChanges
	ExtensionChanges           *ExtensionChanges
}

func (d *DocumentChanges) TotalChanges() int {
	c := d.PropertyChanges.TotalChanges()
	if d.InfoChanges != nil {
		c += d.InfoChanges.TotalChanges()
	}
	if d.PathsChanges != nil {
		c += d.PathsChanges.TotalChanges()
	}
	if d.TagChanges != nil {
		c += d.TagChanges.TotalChanges()
	}
	if d.ExternalDocChanges != nil {
		c += d.ExternalDocChanges.TotalChanges()
	}
	for k := range d.WebhookChanges {
		c += d.WebhookChanges[k].TotalChanges()
	}
	for k := range d.ServerChanges {
		c += d.ServerChanges[k].TotalChanges()
	}
	for k := range d.SecurityRequirementChanges {
		c += d.SecurityRequirementChanges[k].TotalChanges()
	}
	if d.ComponentsChanges != nil {
		c += d.ComponentsChanges.TotalChanges()
	}
	if d.ExtensionChanges != nil {
		c += d.ExtensionChanges.TotalChanges()
	}
	return c
}

func (d *DocumentChanges) TotalBreakingChanges() int {
	c := d.PropertyChanges.TotalBreakingChanges()
	if d.InfoChanges != nil {
		c += d.InfoChanges.TotalBreakingChanges()
	}
	if d.PathsChanges != nil {
		c += d.PathsChanges.TotalBreakingChanges()
	}
	if d.TagChanges != nil {
		c += d.TagChanges.TotalBreakingChanges()
	}
	if d.ExternalDocChanges != nil {
		c += d.ExternalDocChanges.TotalBreakingChanges()
	}
	for k := range d.WebhookChanges {
		c += d.WebhookChanges[k].TotalBreakingChanges()
	}
	for k := range d.ServerChanges {
		c += d.ServerChanges[k].TotalBreakingChanges()
	}
	for k := range d.SecurityRequirementChanges {
		c += d.SecurityRequirementChanges[k].TotalBreakingChanges()
	}
	if d.ComponentsChanges != nil {
		c += d.ComponentsChanges.TotalBreakingChanges()
	}
	return c
}

func CompareDocuments(l, r any) *DocumentChanges {

	var changes []*Change
	var props []*PropertyCheck

	dc := new(DocumentChanges)

	if reflect.TypeOf(&v2.Swagger{}) == reflect.TypeOf(l) && reflect.TypeOf(&v2.Swagger{}) == reflect.TypeOf(r) {
		lDoc := l.(*v2.Swagger)
		rDoc := r.(*v2.Swagger)

		// version
		addPropertyCheck(&props, lDoc.Swagger.ValueNode, rDoc.Swagger.ValueNode,
			lDoc.Swagger.Value, rDoc.Swagger.Value, &changes, v3.SwaggerLabel, true)

		// host
		addPropertyCheck(&props, lDoc.Host.ValueNode, rDoc.Host.ValueNode,
			lDoc.Host.Value, rDoc.Host.Value, &changes, v3.HostLabel, true)

		// base path
		addPropertyCheck(&props, lDoc.BasePath.ValueNode, rDoc.BasePath.ValueNode,
			lDoc.BasePath.Value, rDoc.BasePath.Value, &changes, v3.BasePathLabel, true)

		// schemes
		if len(lDoc.Schemes.Value) > 0 || len(lDoc.Schemes.Value) > 0 {
			ExtractStringValueSliceChanges(lDoc.Schemes.Value, rDoc.Schemes.Value,
				&changes, v3.SchemesLabel, true)
		}
		// consumes
		if len(lDoc.Consumes.Value) > 0 || len(lDoc.Consumes.Value) > 0 {
			ExtractStringValueSliceChanges(lDoc.Consumes.Value, rDoc.Consumes.Value,
				&changes, v3.ConsumesLabel, true)
		}
		// produces
		if len(lDoc.Produces.Value) > 0 || len(lDoc.Produces.Value) > 0 {
			ExtractStringValueSliceChanges(lDoc.Produces.Value, rDoc.Produces.Value,
				&changes, v3.ProducesLabel, true)
		}

		// tags
		dc.TagChanges = CompareTags(lDoc.Tags.Value, rDoc.Tags.Value)

		// paths
		if !lDoc.Paths.IsEmpty() || !rDoc.Paths.IsEmpty() {
			dc.PathsChanges = ComparePaths(lDoc.Paths.Value, rDoc.Paths.Value)
		}

		// external docs
		compareDocumentExternalDocs(lDoc, rDoc, dc, &changes)

		// info
		compareDocumentInfo(&lDoc.Info, &rDoc.Info, dc, &changes)

		// security
		if !lDoc.Security.IsEmpty() || !rDoc.Security.IsEmpty() {
			checkSecurity(lDoc.Security, rDoc.Security, &changes, dc)
		}

		// components / definitions
		// swagger (damn you) decided to put all this stuff at the document root, rather than cleanly
		// placing it under a parent, like they did with OpenAPI. This means picking through each definition
		// creating a new set of changes and then morphing them into a single changes object.
		cc := new(ComponentsChanges)
		if n := CompareComponents(lDoc.Definitions.Value, rDoc.Definitions.Value); n != nil {
			cc.SchemaChanges = n.SchemaChanges
		}
		if n := CompareComponents(lDoc.SecurityDefinitions.Value, rDoc.SecurityDefinitions.Value); n != nil {
			cc.SecuritySchemeChanges = n.SecuritySchemeChanges
		}
		if n := CompareComponents(lDoc.Parameters.Value, rDoc.Parameters.Value); n != nil {
			cc.ParameterChanges = n.ParameterChanges
		}
		if n := CompareComponents(lDoc.Responses.Value, rDoc.Responses.Value); n != nil {
			cc.ResponsesChanges = n.ResponsesChanges
		}
		dc.ExtensionChanges = CompareExtensions(lDoc.Extensions, rDoc.Extensions)
		if cc.TotalChanges() > 0 {
			dc.ComponentsChanges = cc
		}
	}

	CheckProperties(props)
	dc.Changes = changes
	if dc.TotalChanges() <= 0 {
		return nil
	}
	return dc
}

func compareDocumentExternalDocs(l, r low.HasExternalDocs, dc *DocumentChanges, changes *[]*Change) {
	// external docs
	if !l.GetExternalDocs().IsEmpty() && !r.GetExternalDocs().IsEmpty() {
		lExtDoc := l.GetExternalDocs().Value.(*base.ExternalDoc)
		rExtDoc := r.GetExternalDocs().Value.(*base.ExternalDoc)
		if !low.AreEqual(lExtDoc, rExtDoc) {
			dc.ExternalDocChanges = CompareExternalDocs(lExtDoc, rExtDoc)
		}
	}
	if l.GetExternalDocs().IsEmpty() && !r.GetExternalDocs().IsEmpty() {
		CreateChange(changes, PropertyAdded, v3.ExternalDocsLabel,
			nil, r.GetExternalDocs().ValueNode, false, nil,
			r.GetExternalDocs().Value)
	}
	if !l.GetExternalDocs().IsEmpty() && r.GetExternalDocs().IsEmpty() {
		CreateChange(changes, PropertyRemoved, v3.ExternalDocsLabel,
			l.GetExternalDocs().ValueNode, nil, false, l.GetExternalDocs().Value,
			nil)
	}
}

func compareDocumentInfo(l, r *low.NodeReference[*base.Info], dc *DocumentChanges, changes *[]*Change) {
	// info
	if !l.IsEmpty() && !r.IsEmpty() {
		lInfo := l.Value
		rInfo := r.Value
		if !low.AreEqual(lInfo, rInfo) {
			dc.InfoChanges = CompareInfo(lInfo, rInfo)
		}
	}
	if l.IsEmpty() && !r.IsEmpty() {
		CreateChange(changes, PropertyAdded, v3.InfoLabel,
			nil, r.ValueNode, false, nil,
			r.Value)
	}
	if !l.IsEmpty() && r.IsEmpty() {
		CreateChange(changes, PropertyRemoved, v3.InfoLabel,
			l.ValueNode, nil, false, l.Value,
			nil)
	}
}
