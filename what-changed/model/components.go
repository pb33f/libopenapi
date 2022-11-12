// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	v2 "github.com/pb33f/libopenapi/datamodel/low/v2"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"reflect"
)

type ComponentsChanges struct {
	PropertyChanges
	SchemaChanges         map[string]*SchemaChanges
	ResponsesChanges      map[string]*ResponseChanges
	ParameterChanges      map[string]*ParameterChanges
	ExamplesChanges       map[string]*ExamplesChanges
	RequestBodyChanges    map[string]*RequestBodyChanges
	HeaderChanges         map[string]*HeaderChanges
	SecuritySchemeChanges map[string]*SecuritySchemeChanges
	LinkChanges           map[string]*LinkChanges
	CallbackChanges       map[string]*CallbackChanges
	ExtensionChanges      *ExtensionChanges
}

func CompareComponents(l, r any) *ComponentsChanges {

	var changes []*Change

	cc := new(ComponentsChanges)

	// Swagger Parameters
	if reflect.TypeOf(&v2.ParameterDefinitions{}) == reflect.TypeOf(l) &&
		reflect.TypeOf(&v2.ParameterDefinitions{}) == reflect.TypeOf(r) {
		lDef := l.(*v2.ParameterDefinitions)
		rDef := r.(*v2.ParameterDefinitions)
		cc.ParameterChanges = CheckMapForChangesUntyped(lDef.Definitions, rDef.Definitions, &changes,
			v3.ParametersLabel, CompareParameters)
	}

	// Swagger Responses
	if reflect.TypeOf(&v2.ResponsesDefinitions{}) == reflect.TypeOf(l) &&
		reflect.TypeOf(&v2.ResponsesDefinitions{}) == reflect.TypeOf(r) {
		lDef := l.(*v2.ResponsesDefinitions)
		rDef := r.(*v2.ResponsesDefinitions)
		cc.ResponsesChanges = CheckMapForChangesUntyped(lDef.Definitions, rDef.Definitions, &changes,
			v3.ResponsesLabel, CompareResponse)
	}

	// Swagger Schemas
	if reflect.TypeOf(&v2.Definitions{}) == reflect.TypeOf(l) &&
		reflect.TypeOf(&v2.Definitions{}) == reflect.TypeOf(r) {
		lDef := l.(*v2.Definitions)
		rDef := r.(*v2.Definitions)
		cc.SchemaChanges = CheckMapForChanges(lDef.Schemas, rDef.Schemas, &changes,
			v2.DefinitionsLabel, CompareSchemas)
	}

	// Swagger Security Definitions
	if reflect.TypeOf(&v2.SecurityDefinitions{}) == reflect.TypeOf(l) &&
		reflect.TypeOf(&v2.SecurityDefinitions{}) == reflect.TypeOf(r) {
		lDef := l.(*v2.SecurityDefinitions)
		rDef := r.(*v2.SecurityDefinitions)
		cc.SecuritySchemeChanges = CheckMapForChangesUntyped(lDef.Definitions, rDef.Definitions, &changes,
			v3.SecurityDefinitionLabel, CompareSecuritySchemes)
	}

	cc.Changes = changes
	if cc.TotalChanges() <= 0 {
		return nil
	}
	return cc
}

func (c *ComponentsChanges) TotalChanges() int {
	v := c.PropertyChanges.TotalChanges()
	for k := range c.SchemaChanges {
		v += c.SchemaChanges[k].TotalChanges()
	}
	for k := range c.ResponsesChanges {
		v += c.ResponsesChanges[k].TotalChanges()
	}
	for k := range c.ParameterChanges {
		v += c.ParameterChanges[k].TotalChanges()
	}
	for k := range c.ExamplesChanges {
		v += c.ExamplesChanges[k].TotalChanges()
	}
	for k := range c.RequestBodyChanges {
		v += c.RequestBodyChanges[k].TotalChanges()
	}
	for k := range c.HeaderChanges {
		v += c.HeaderChanges[k].TotalChanges()
	}
	for k := range c.SecuritySchemeChanges {
		v += c.SecuritySchemeChanges[k].TotalChanges()
	}
	for k := range c.LinkChanges {
		v += c.LinkChanges[k].TotalChanges()
	}
	for k := range c.CallbackChanges {
		v += c.CallbackChanges[k].TotalChanges()
	}
	if c.ExtensionChanges != nil {
		v += c.ExtensionChanges.TotalChanges()
	}
	return v
}

func (c *ComponentsChanges) TotalBreakingChanges() int {
	v := c.PropertyChanges.TotalBreakingChanges()
	for k := range c.SchemaChanges {
		v += c.SchemaChanges[k].TotalBreakingChanges()
	}
	for k := range c.ResponsesChanges {
		v += c.ResponsesChanges[k].TotalBreakingChanges()
	}
	for k := range c.ParameterChanges {
		v += c.ParameterChanges[k].TotalBreakingChanges()
	}
	for k := range c.ExamplesChanges {
		v += c.ExamplesChanges[k].TotalBreakingChanges()
	}
	for k := range c.RequestBodyChanges {
		v += c.RequestBodyChanges[k].TotalBreakingChanges()
	}
	for k := range c.HeaderChanges {
		v += c.HeaderChanges[k].TotalBreakingChanges()
	}
	for k := range c.SecuritySchemeChanges {
		v += c.SecuritySchemeChanges[k].TotalBreakingChanges()
	}
	for k := range c.LinkChanges {
		v += c.LinkChanges[k].TotalBreakingChanges()
	}
	for k := range c.CallbackChanges {
		v += c.CallbackChanges[k].TotalBreakingChanges()
	}
	if c.ExtensionChanges != nil {
		v += c.ExtensionChanges.TotalBreakingChanges()
	}
	return v
}
