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

func (s *Server) GoLow() *low.Server {
	return s.low
}
