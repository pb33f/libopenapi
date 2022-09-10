// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	low "github.com/pb33f/libopenapi/datamodel/low/2.0"
)

type Swagger struct {
	Swagger             string
	Info                *base.Info
	Host                string
	BasePath            string
	Schemes             []string
	Consumes            []string
	Produces            []string
	Paths               *Paths
	Definitions         *Definitions
	Parameters          *ParameterDefinitions
	Responses           *ResponsesDefinitions
	SecurityDefinitions *SecurityDefinitions
	Security            []*SecurityRequirement
	Tags                []*base.Tag
	ExternalDocs        *base.ExternalDoc
	Extensions          map[string]any
	low                 *low.Swagger
}

func NewSwaggerDocument(document *low.Swagger) *Swagger {
	d := new(Swagger)
	d.low = document
	d.Extensions = high.ExtractExtensions(document.Extensions)
	if !document.Info.IsEmpty() {
		d.Info = base.NewInfo(document.Info.Value)
	}
	if !document.Swagger.IsEmpty() {
		d.Swagger = document.Swagger.Value
	}
	if !document.Host.IsEmpty() {
		d.Host = document.Host.Value
	}
	if !document.BasePath.IsEmpty() {
		d.BasePath = document.BasePath.Value
	}

	if !document.Schemes.IsEmpty() {
		var schemes []string
		for s := range document.Schemes.Value {
			schemes = append(schemes, document.Schemes.Value[s].Value)
		}
		d.Schemes = schemes
	}
	if !document.Consumes.IsEmpty() {
		var consumes []string
		for c := range document.Consumes.Value {
			consumes = append(consumes, document.Consumes.Value[c].Value)
		}
		d.Consumes = consumes
	}
	if !document.Produces.IsEmpty() {
		var produces []string
		for p := range document.Produces.Value {
			produces = append(produces, document.Produces.Value[p].Value)
		}
		d.Produces = produces
	}
	if !document.Paths.IsEmpty() {
		d.Paths = NewPaths(document.Paths.Value)
	}
	if !document.Definitions.IsEmpty() {
		d.Definitions = NewDefinitions(document.Definitions.Value)
	}
	if !document.Parameters.IsEmpty() {
		d.Parameters = NewParametersDefinitions(document.Parameters.Value)
	}

	if !document.Responses.IsEmpty() {
		d.Responses = NewResponsesDefinitions(document.Responses.Value)
	}
	if !document.SecurityDefinitions.IsEmpty() {
		d.SecurityDefinitions = NewSecurityDefinitions(document.SecurityDefinitions.Value)
	}
	if !document.Security.IsEmpty() {
		var security []*SecurityRequirement
		for s := range document.Security.Value {
			security = append(security, NewSecurityRequirement(document.Security.Value[s].Value))
		}
		d.Security = security
	}
	if !document.Tags.IsEmpty() {
		var tags []*base.Tag
		for t := range document.Tags.Value {
			tags = append(tags, base.NewTag(document.Tags.Value[t].Value))
		}
		d.Tags = tags
	}
	if !document.ExternalDocs.IsEmpty() {
		d.ExternalDocs = base.NewExternalDoc(document.ExternalDocs.Value)
	}
	return d
}

func (s *Swagger) GoLow() *low.Swagger {
	return s.low
}

type asyncResult[T any] struct {
	key    string
	result T
}
