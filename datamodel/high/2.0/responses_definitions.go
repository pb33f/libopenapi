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
	var buildResp = func(name string, resp *low.Response, rChan chan<- asyncResult[*Response]) {
		rChan <- asyncResult[*Response]{
			key:    name,
			result: NewResponse(resp),
		}
	}
	resChan := make(chan asyncResult[*Response])
	for k := range responsesDefinitions.Definitions {
		go buildResp(k.Value, responsesDefinitions.Definitions[k].Value, resChan)
	}
	totalResponses := len(responsesDefinitions.Definitions)
	completedResponses := 0
	for completedResponses < totalResponses {
		select {
		case r := <-resChan:
			completedResponses++
			responses[r.key] = r.result
		}
	}
	rd.Definitions = responses
	return rd
}
