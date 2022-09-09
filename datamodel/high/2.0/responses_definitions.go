// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import low "github.com/pb33f/libopenapi/datamodel/low/2.0"

type ResponsesDefinitions struct {
	Definitions map[string]*Response
	low         *low.ResponsesDefinitions
}

func NewResponsesDefinitions(responsesDefinitions *low.ResponsesDefinitions) *ResponsesDefinitions {
	rd := new(ResponsesDefinitions)
	rd.low = responsesDefinitions
	responses := make(map[string]*Response)
	for k := range responsesDefinitions.Definitions {
		responses[k.Value] = NewResponse(responsesDefinitions.Definitions[k].Value)
	}
	rd.Definitions = responses
	return rd
}
