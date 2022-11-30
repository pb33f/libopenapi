// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/v3"
)

// Server represents a high-level OpenAPI 3+ Server object, that is backed by a low level one.
//  - https://spec.openapis.org/oas/v3.1.0#server-object
type Server struct {
	URL         string
	Description string
	Variables   map[string]*ServerVariable
	Extensions  map[string]any
	low         *low.Server
}

// NewServer will create a new high-level Server instance from a low-level one.
func NewServer(server *low.Server) *Server {
	s := new(Server)
	s.low = server
	s.Description = server.Description.Value
	s.URL = server.URL.Value
	vars := make(map[string]*ServerVariable)
	for k, val := range server.Variables.Value {
		vars[k.Value] = NewServerVariable(val.Value)
	}
	s.Variables = vars
	s.Extensions = high.ExtractExtensions(server.Extensions)
	return s
}

// GoLow returns the low-level Server instance that was used to create the high-level one
func (s *Server) GoLow() *low.Server {
	return s.low
}
