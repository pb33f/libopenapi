// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import low "github.com/pb33f/libopenapi/datamodel/low/3.0"

type Server struct {
	URL         string
	Description string
	Variables   map[string]*ServerVariable
	low         *low.Server
}

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
	return s
}

func (s *Server) GoLow() *low.Server {
	return s.low
}
