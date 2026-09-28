// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package sdk

import (
	"github.com/pb33f/go-yaml"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	highv3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
)

// Contract is a borrowed effective view over one OpenAPI document. It retains
// the document's schema proxies so emitters can reuse libopenapi's model
// generators without introducing a second schema representation. Callers must
// keep the source document alive for the Contract's lifetime and should release
// both together. Schema traversal may warm caches on the source document.
type Contract struct {
	Servers         []string
	Operations      []*Operation
	SecuritySchemes []*SecurityScheme
	Schemas         *orderedmap.Map[string, *highbase.SchemaProxy]
}

// Workflow is a finite Arazzo composition supported by a language emitter.
// Inputs borrows the workflow's JSON Schema node from the source Arazzo document.
// The initial compiler deliberately accepts one typed operation step; unsupported
// control flow is reported instead of being omitted.
type Workflow struct {
	ID                string
	Summary           string
	Description       string
	Operation         *Operation
	Inputs            *yaml.Node
	ParameterBindings []*ParameterBinding
	PayloadBindings   []*PayloadBinding
	SuccessStatuses   []string
}

// ParameterBinding maps one OpenAPI operation parameter to one Arazzo workflow input.
type ParameterBinding struct {
	Name  string
	In    string
	Input string
}

// PayloadBinding maps one JSON request property to one Arazzo workflow input.
type PayloadBinding struct {
	Property string
	Input    string
}

// Operation is one effective OpenAPI operation after path-level inheritance.
type Operation struct {
	ID          string
	Method      string
	Path        string
	Resource    string
	Name        string
	Summary     string
	Description string
	Servers     []string
	Parameters  []*Parameter
	RequestBody *RequestBody
	Responses   []*Response
	Security    []*SecurityRequirement
}

// Parameter is an effective operation parameter.
type Parameter struct {
	Name          string
	In            string
	Description   string
	Required      bool
	Style         string
	Explode       bool
	AllowReserved bool
	Schema        *highbase.SchemaProxy
}

// RequestBody is an operation request body.
type RequestBody struct {
	Required    bool
	Description string
	Content     *orderedmap.Map[string, *highv3.MediaType]
}

// Response is one declared operation response.
type Response struct {
	Status      string
	Description string
	Content     *orderedmap.Map[string, *highv3.MediaType]
}

// SecurityRequirement is one OR alternative. All schemes inside an alternative
// are required together.
type SecurityRequirement struct {
	Schemes []*RequiredSecurityScheme
}

// RequiredSecurityScheme is one named scheme and its required OAuth/OpenID scopes.
type RequiredSecurityScheme struct {
	Name   string
	Scopes []string
}

// SecurityScheme describes how a named OpenAPI security scheme reaches a request.
type SecurityScheme struct {
	Name             string
	Type             string
	In               string
	ParameterName    string
	Scheme           string
	OpenIDConnectURL string
}

// OperationName overrides the public resource and method name for an operation.
type OperationName struct {
	Resource string
	Method   string
}

// PrepareOptions selects operations and controls their public names. An empty
// Operations slice selects every operation with an operationId.
type PrepareOptions struct {
	Operations []string
	Names      map[string]OperationName
}

// Diagnostic reports a notable model-generation decision.
type Diagnostic struct {
	Code    string
	Path    string
	Message string
}

// File is one deterministic generated source file.
type File struct {
	Path    string
	Content []byte
}

// Result contains generated files and non-fatal model-generation diagnostics.
type Result struct {
	Files       []File
	Diagnostics []Diagnostic
}
