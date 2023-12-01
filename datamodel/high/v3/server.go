// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"gopkg.in/yaml.v3"
)

// Server represents a high-level OpenAPI 3+ Server object, that is backed by a low level one.
//   - https://spec.openapis.org/oas/v3.1.0#server-object
type Server struct {
	URL         string                                   `json:"url,omitempty" yaml:"url,omitempty"`
	Description string                                   `json:"description,omitempty" yaml:"description,omitempty"`
	Variables   *orderedmap.Map[string, *ServerVariable] `json:"variables,omitempty" yaml:"variables,omitempty"`
	Extensions  *orderedmap.Map[string, *yaml.Node]      `json:"-" yaml:"-"`
	low         *low.Server
}

// NewServer will create a new high-level Server instance from a low-level one.
func NewServer(server *low.Server) *Server {
	s := new(Server)
	s.low = server
	s.Description = server.Description.Value
	s.URL = server.URL.Value
	vars := orderedmap.New[string, *ServerVariable]()
	for pair := orderedmap.First(server.Variables.Value); pair != nil; pair = pair.Next() {
		vars.Set(pair.Key().Value, NewServerVariable(pair.Value().Value))
	}
	s.Variables = vars
	s.Extensions = high.ExtractExtensions(server.Extensions)
	return s
}

// GoLow returns the low-level Server instance that was used to create the high-level one
func (s *Server) GoLow() *low.Server {
	return s.low
}

// GoLowUntyped will return the low-level Server instance that was used to create the high-level one, with no type
func (s *Server) GoLowUntyped() any {
	return s.low
}

// Render will return a YAML representation of the Server object as a byte slice.
func (s *Server) Render() ([]byte, error) {
	return yaml.Marshal(s)
}

// MarshalYAML will create a ready to render YAML representation of the Server object.
func (s *Server) MarshalYAML() (interface{}, error) {
	nb := high.NewNodeBuilder(s, s.low)
	return nb.Render(), nil
}
